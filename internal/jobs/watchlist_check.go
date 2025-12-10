package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"austrian-business-infrastructure/internal/fb"
	"austrian-business-infrastructure/internal/job"
	"austrian-business-infrastructure/internal/watchlist"
	"austrian-business-infrastructure/internal/webhook"
	"github.com/google/uuid"
)

// WatchlistCheck is the job type for checking Firmenbuch watchlist entries
const WatchlistCheckJobType = "watchlist_check"

// WatchlistCheckHandler handles Firmenbuch watchlist checking
type WatchlistCheckHandler struct {
	watchlistRepo  *watchlist.Repository
	fbClient       *fb.Client
	webhookService *webhook.Service
	logger         *slog.Logger
	concurrency    int
}

// WatchlistCheckConfig holds configuration for the watchlist check handler
type WatchlistCheckConfig struct {
	Logger      *slog.Logger
	Concurrency int // Max concurrent FB API calls
}

// NewWatchlistCheckHandler creates a new watchlist check handler
func NewWatchlistCheckHandler(
	watchlistRepo *watchlist.Repository,
	fbClient *fb.Client,
	webhookService *webhook.Service,
	cfg *WatchlistCheckConfig,
) *WatchlistCheckHandler {
	logger := slog.Default()
	concurrency := 3 // Conservative default to respect FB API limits

	if cfg != nil {
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.Concurrency > 0 {
			concurrency = cfg.Concurrency
		}
	}

	return &WatchlistCheckHandler{
		watchlistRepo:  watchlistRepo,
		fbClient:       fbClient,
		webhookService: webhookService,
		logger:         logger,
		concurrency:    concurrency,
	}
}

// WatchlistCheckPayload defines the job payload
type WatchlistCheckPayload struct {
	TenantID *uuid.UUID `json:"tenant_id,omitempty"` // Optional: specific tenant
	ItemID   *uuid.UUID `json:"item_id,omitempty"`   // Optional: specific item
}

// WatchlistCheckResult contains the results of a watchlist check
type WatchlistCheckResult struct {
	ItemsChecked  int      `json:"items_checked"`
	ItemsChanged  int      `json:"items_changed"`
	ItemsFailed   int      `json:"items_failed"`
	ChangedItems  []string `json:"changed_items,omitempty"`
	FailedItems   []string `json:"failed_items,omitempty"`
}

// Handle executes the watchlist check job
func (h *WatchlistCheckHandler) Handle(ctx context.Context, j *job.Job) (json.RawMessage, error) {
	h.logger.Info("starting watchlist check job", "job_id", j.ID)

	// Parse payload
	var payload WatchlistCheckPayload
	if len(j.Payload) > 0 {
		if err := json.Unmarshal(j.Payload, &payload); err != nil {
			return nil, fmt.Errorf("parse payload: %w", err)
		}
	}

	// Get items to check
	var items []*watchlist.Item
	var err error

	if payload.ItemID != nil {
		// Check specific item
		item, err := h.watchlistRepo.GetByID(ctx, *payload.ItemID)
		if err != nil {
			return nil, fmt.Errorf("get watchlist item: %w", err)
		}
		items = []*watchlist.Item{item}
	} else if payload.TenantID != nil {
		// Check all items for a specific tenant
		items, err = h.watchlistRepo.List(ctx, *payload.TenantID, true)
		if err != nil {
			return nil, fmt.Errorf("list tenant watchlist items: %w", err)
		}
	} else {
		// Check all enabled items across all tenants
		items, err = h.watchlistRepo.ListAllEnabled(ctx)
		if err != nil {
			return nil, fmt.Errorf("list all enabled watchlist items: %w", err)
		}
	}

	if len(items) == 0 {
		h.logger.Info("no watchlist items to check")
		return json.Marshal(WatchlistCheckResult{})
	}

	h.logger.Info("checking watchlist items", "count", len(items))

	// Process items with limited concurrency
	result := h.processItems(ctx, items)

	h.logger.Info("watchlist check completed",
		"checked", result.ItemsChecked,
		"changed", result.ItemsChanged,
		"failed", result.ItemsFailed)

	return json.Marshal(result)
}

// processItems checks all items with limited concurrency
func (h *WatchlistCheckHandler) processItems(ctx context.Context, items []*watchlist.Item) WatchlistCheckResult {
	var (
		mu       sync.Mutex
		result   WatchlistCheckResult
		wg       sync.WaitGroup
		sem      = make(chan struct{}, h.concurrency)
	)

	for _, item := range items {
		select {
		case <-ctx.Done():
			return result
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(item *watchlist.Item) {
			defer wg.Done()
			defer func() { <-sem }()

			changed, err := h.checkItem(ctx, item)

			mu.Lock()
			defer mu.Unlock()

			result.ItemsChecked++
			if err != nil {
				result.ItemsFailed++
				result.FailedItems = append(result.FailedItems, item.CompanyNumber)
				h.logger.Error("failed to check watchlist item",
					"item_id", item.ID,
					"company_number", item.CompanyNumber,
					"error", err)
			} else if changed {
				result.ItemsChanged++
				result.ChangedItems = append(result.ChangedItems, item.CompanyNumber)
			}
		}(item)
	}

	wg.Wait()
	return result
}

// checkItem checks a single watchlist item for changes
func (h *WatchlistCheckHandler) checkItem(ctx context.Context, item *watchlist.Item) (bool, error) {
	// Fetch current data from Firmenbuch
	extract, err := h.fbClient.Extract(item.CompanyNumber)
	if err != nil {
		return false, fmt.Errorf("fetch FB data: %w", err)
	}

	// Serialize new snapshot
	newSnapshot, err := json.Marshal(extract)
	if err != nil {
		return false, fmt.Errorf("marshal snapshot: %w", err)
	}

	// Compare with previous snapshot
	changed := false
	if item.LastSnapshot != nil {
		changed = !jsonEqual(item.LastSnapshot, newSnapshot)
	} else {
		// First check - not considered a "change"
		changed = false
	}

	// Update the snapshot in database
	if err := h.watchlistRepo.UpdateSnapshot(ctx, item.ID, newSnapshot, changed); err != nil {
		return changed, fmt.Errorf("update snapshot: %w", err)
	}

	// Trigger webhook if changed and notifications enabled
	if changed && item.NotifyOnChange && h.webhookService != nil {
		h.triggerChangeWebhook(ctx, item, extract)
	}

	h.logger.Debug("checked watchlist item",
		"item_id", item.ID,
		"company_number", item.CompanyNumber,
		"changed", changed)

	return changed, nil
}

// triggerChangeWebhook sends a webhook notification for FB changes
func (h *WatchlistCheckHandler) triggerChangeWebhook(ctx context.Context, item *watchlist.Item, extract *fb.FBExtract) {
	eventData := map[string]interface{}{
		"watchlist_item_id": item.ID.String(),
		"company_number":    item.CompanyNumber,
		"company_name":      extract.Firma,
		"status":            string(extract.Status),
		"rechtsform":        string(extract.Rechtsform),
		"sitz":              extract.Sitz,
		"uid":               extract.UID,
		"last_change":       extract.LetzteAenderungString(),
	}

	if err := h.webhookService.TriggerEvent(ctx, item.TenantID, webhook.EventFBChange, eventData); err != nil {
		h.logger.Error("failed to trigger FB change webhook",
			"item_id", item.ID,
			"error", err)
	}
}

// jsonEqual compares two JSON byte slices for equality
func jsonEqual(a, b []byte) bool {
	var va, vb interface{}
	if err := json.Unmarshal(a, &va); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &vb); err != nil {
		return false
	}

	// Re-marshal to normalize
	na, _ := json.Marshal(va)
	nb, _ := json.Marshal(vb)

	return string(na) == string(nb)
}

// Register registers the watchlist check handler with a job registry
func (h *WatchlistCheckHandler) Register(registry *job.Registry) {
	registry.MustRegister(WatchlistCheckJobType, h)
}
