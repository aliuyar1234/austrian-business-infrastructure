package matcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/foerderung"
)

// Cache provides caching for search results and LLM responses
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// InMemoryCache provides simple in-memory caching with TTL
type InMemoryCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	stop    chan struct{}
	done    chan struct{}
}

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache() *InMemoryCache {
	c := &InMemoryCache{
		entries: make(map[string]cacheEntry),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	// Start cleanup goroutine
	go c.cleanup()
	return c
}

// Close stops the cleanup goroutine and releases resources
func (c *InMemoryCache) Close() error {
	close(c.stop)
	<-c.done // Wait for cleanup goroutine to finish
	return nil
}

func (c *InMemoryCache) Get(ctx context.Context, key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.value, true
}

func (c *InMemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, key)
	return nil
}

func (c *InMemoryCache) cleanup() {
	defer close(c.done)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.entries {
				if now.After(entry.expiresAt) {
					delete(c.entries, key)
				}
			}
			c.mu.Unlock()
		}
	}
}

// CachingLLMClient wraps an LLM client with caching
type CachingLLMClient struct {
	client  LLMClient
	cache   Cache
	ttl     time.Duration
	enabled bool
}

// NewCachingLLMClient creates a new caching LLM client
func NewCachingLLMClient(client LLMClient, cache Cache, cfg *config.FoerderungConfig) *CachingLLMClient {
	return &CachingLLMClient{
		client:  client,
		cache:   cache,
		ttl:     time.Duration(cfg.SearchCacheTTLHours) * time.Hour,
		enabled: cfg.SearchCacheTTLHours > 0,
	}
}

// AnalyzeEligibility implements LLMClient with caching
func (c *CachingLLMClient) AnalyzeEligibility(
	ctx context.Context,
	profile *ProfileInput,
	fd *foerderung.Foerderung,
) (*foerderung.LLMEligibilityResult, error) {
	if !c.enabled {
		return c.client.AnalyzeEligibility(ctx, profile, fd)
	}

	// Generate cache key from profile and foerderung
	cacheKey := c.generateCacheKey(profile, fd)

	// Try to get from cache
	if cached, ok := c.cache.Get(ctx, cacheKey); ok {
		var result foerderung.LLMEligibilityResult
		if err := json.Unmarshal(cached, &result); err == nil {
			return &result, nil
		}
	}

	// Call actual LLM
	result, err := c.client.AnalyzeEligibility(ctx, profile, fd)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if result != nil {
		if data, err := json.Marshal(result); err == nil {
			c.cache.Set(ctx, cacheKey, data, c.ttl)
		}
	}

	return result, nil
}

// generateCacheKey creates a deterministic cache key from profile and foerderung
func (c *CachingLLMClient) generateCacheKey(profile *ProfileInput, fd *foerderung.Foerderung) string {
	// Create a hash of relevant profile and foerderung fields
	data := struct {
		FoerderungID string
		FoerderungV  time.Time // Use updated_at for cache invalidation
		CompanyName  string
		State        string
		Industry     string
		Employees    *int
		Revenue      *int
		Topics       []string
		IsStartup    bool
		FoundedYear  *int
	}{
		FoerderungID: fd.ID.String(),
		FoerderungV:  fd.UpdatedAt,
		CompanyName:  profile.CompanyName,
		State:        profile.State,
		Industry:     profile.Industry,
		Employees:    profile.EmployeesCount,
		Revenue:      profile.AnnualRevenue,
		Topics:       profile.ProjectTopics,
		IsStartup:    profile.IsStartup,
		FoundedYear:  profile.FoundedYear,
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("llm:%s", hex.EncodeToString(hash[:16]))
}

// SearchResultCache caches entire search results
type SearchResultCache struct {
	cache Cache
	ttl   time.Duration
}

// NewSearchResultCache creates a new search result cache
func NewSearchResultCache(cache Cache, cfg *config.FoerderungConfig) *SearchResultCache {
	return &SearchResultCache{
		cache: cache,
		ttl:   time.Duration(cfg.SearchCacheTTLHours) * time.Hour,
	}
}

// CachedSearchResult represents a cached search result
type CachedSearchResult struct {
	Matches       []foerderung.FoerderungsMatch `json:"matches"`
	TotalChecked  int                           `json:"total_checked"`
	LLMTokensUsed int                           `json:"llm_tokens_used"`
	LLMCostCents  int                           `json:"llm_cost_cents"`
	CachedAt      time.Time                     `json:"cached_at"`
}

// Get retrieves a cached search result
func (c *SearchResultCache) Get(ctx context.Context, profileID string) (*CachedSearchResult, bool) {
	key := fmt.Sprintf("search:%s", profileID)

	data, ok := c.cache.Get(ctx, key)
	if !ok {
		return nil, false
	}

	var result CachedSearchResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, false
	}

	return &result, true
}

// Set stores a search result in cache
func (c *SearchResultCache) Set(ctx context.Context, profileID string, result *CachedSearchResult) error {
	key := fmt.Sprintf("search:%s", profileID)
	result.CachedAt = time.Now()

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return c.cache.Set(ctx, key, data, c.ttl)
}

// Invalidate removes a cached search result
func (c *SearchResultCache) Invalidate(ctx context.Context, profileID string) error {
	key := fmt.Sprintf("search:%s", profileID)
	return c.cache.Delete(ctx, key)
}

// CacheStats provides cache statistics
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Size   int64 `json:"size"`
}

// StatsCache wraps a cache with statistics tracking
type StatsCache struct {
	cache  Cache
	mu     sync.Mutex
	hits   int64
	misses int64
}

// NewStatsCache creates a new statistics-tracking cache
func NewStatsCache(cache Cache) *StatsCache {
	return &StatsCache{cache: cache}
}

func (c *StatsCache) Get(ctx context.Context, key string) ([]byte, bool) {
	data, ok := c.cache.Get(ctx, key)
	c.mu.Lock()
	if ok {
		c.hits++
	} else {
		c.misses++
	}
	c.mu.Unlock()
	return data, ok
}

func (c *StatsCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.cache.Set(ctx, key, value, ttl)
}

func (c *StatsCache) Delete(ctx context.Context, key string) error {
	return c.cache.Delete(ctx, key)
}

func (c *StatsCache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		Hits:   c.hits,
		Misses: c.misses,
	}
}
