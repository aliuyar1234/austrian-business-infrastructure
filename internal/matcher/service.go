package matcher

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/foerderung"
)

// LLMClient interface for LLM integration
type LLMClient interface {
	AnalyzeEligibility(ctx context.Context, profile *ProfileInput, fd *foerderung.Foerderung) (*foerderung.LLMEligibilityResult, error)
}

// Service orchestrates the matching process
type Service struct {
	foerderungRepo *foerderung.Repository
	searchRepo     *SearchRepository
	filter         *Filter
	llmClient      LLMClient
	config         *config.FoerderungConfig
}

// NewService creates a new matcher service
func NewService(
	foerderungRepo *foerderung.Repository,
	searchRepo *SearchRepository,
	llmClient LLMClient,
	cfg *config.FoerderungConfig,
) *Service {
	return &Service{
		foerderungRepo: foerderungRepo,
		searchRepo:     searchRepo,
		filter:         NewFilter(),
		llmClient:      llmClient,
		config:         cfg,
	}
}

// SearchInput contains the input for a new search
type SearchInput struct {
	TenantID  uuid.UUID
	ProfileID uuid.UUID
	Profile   *ProfileInput
	CreatedBy *uuid.UUID
}

// SearchOutput contains the result of a search
type SearchOutput struct {
	SearchID       uuid.UUID                   `json:"search_id"`
	TotalChecked   int                         `json:"total_checked"`
	TotalMatches   int                         `json:"total_matches"`
	Matches        []foerderung.FoerderungsMatch `json:"matches"`
	LLMTokensUsed  int                         `json:"llm_tokens_used"`
	LLMCostCents   int                         `json:"llm_cost_cents"`
	Duration       time.Duration               `json:"duration"`
	LLMFallback    bool                        `json:"llm_fallback"` // True if LLM was skipped
}

// RunSearch executes a complete search (rule filtering + LLM analysis)
func (s *Service) RunSearch(ctx context.Context, input *SearchInput) (*SearchOutput, error) {
	startTime := time.Now()

	// Create search record
	search := &foerderung.FoerderungsSuche{
		TenantID:  input.TenantID,
		ProfileID: input.ProfileID,
		Status:    foerderung.SearchStatusPending,
		CreatedBy: input.CreatedBy,
	}

	now := time.Now()
	search.StartedAt = &now

	if err := s.searchRepo.Create(ctx, search); err != nil {
		return nil, fmt.Errorf("failed to create search: %w", err)
	}

	// Get all active Förderungen
	foerderungen, err := s.foerderungRepo.ListActive(ctx)
	if err != nil {
		s.updateSearchError(ctx, search, err)
		return nil, fmt.Errorf("failed to list foerderungen: %w", err)
	}

	search.TotalFoerderungen = len(foerderungen)

	// Phase 1: Rule-based filtering
	if err := s.searchRepo.UpdateStatus(ctx, search.ID, foerderung.SearchStatusRuleFiltering, 10); err != nil {
		return nil, err
	}

	candidates := s.filter.GetCandidates(input.Profile, foerderungen)

	// Phase 2: LLM analysis (if available)
	var matches []foerderung.FoerderungsMatch
	llmFallback := false
	llmTokensUsed := 0
	llmCostCents := 0

	if s.llmClient != nil && len(candidates) > 0 {
		if err := s.searchRepo.UpdateStatus(ctx, search.ID, foerderung.SearchStatusLLMAnalysis, 50); err != nil {
			return nil, err
		}

		matches, llmTokensUsed, llmCostCents, err = s.runLLMAnalysis(ctx, input.Profile, candidates)
		if err != nil {
			// LLM failed - fall back to rule-only
			if s.config.LLMFallbackEnabled {
				llmFallback = true
				matches = s.convertToRuleOnlyMatches(candidates)
			} else {
				s.updateSearchError(ctx, search, err)
				return nil, fmt.Errorf("LLM analysis failed: %w", err)
			}
		}
	} else {
		// No LLM client - rule-only mode
		llmFallback = true
		matches = s.convertToRuleOnlyMatches(candidates)
	}

	// Sort matches by total score
	sortMatchesByScore(matches)

	// Limit results
	if len(matches) > s.config.MaxResultsPerSearch {
		matches = matches[:s.config.MaxResultsPerSearch]
	}

	// Update search record
	completedAt := time.Now()
	search.Status = foerderung.SearchStatusCompleted
	search.TotalMatches = len(matches)
	search.CompletedAt = &completedAt
	search.LLMTokensUsed = llmTokensUsed
	search.LLMCostCents = llmCostCents

	matchesJSON, _ := json.Marshal(matches)
	search.Matches = matchesJSON

	if err := s.searchRepo.Update(ctx, search); err != nil {
		return nil, fmt.Errorf("failed to update search: %w", err)
	}

	return &SearchOutput{
		SearchID:      search.ID,
		TotalChecked:  len(foerderungen),
		TotalMatches:  len(matches),
		Matches:       matches,
		LLMTokensUsed: llmTokensUsed,
		LLMCostCents:  llmCostCents,
		Duration:      time.Since(startTime),
		LLMFallback:   llmFallback,
	}, nil
}

// runLLMAnalysis runs LLM analysis on candidates in parallel
func (s *Service) runLLMAnalysis(
	ctx context.Context,
	profile *ProfileInput,
	candidates []*MatchCandidate,
) ([]foerderung.FoerderungsMatch, int, int, error) {
	// Limit concurrency
	semaphore := make(chan struct{}, s.config.LLMMaxConcurrent)
	var wg sync.WaitGroup

	type llmResult struct {
		candidate *MatchCandidate
		result    *foerderung.LLMEligibilityResult
		err       error
	}

	results := make(chan llmResult, len(candidates))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, s.config.SearchTimeout)
	defer cancel()

	// Run LLM analysis in parallel
	for _, candidate := range candidates {
		wg.Add(1)
		go func(c *MatchCandidate) {
			defer wg.Done()

			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				results <- llmResult{candidate: c, err: ctx.Err()}
				return
			}

			llmCtx, llmCancel := context.WithTimeout(ctx, s.config.LLMTimeout)
			defer llmCancel()

			result, err := s.llmClient.AnalyzeEligibility(llmCtx, profile, c.Foerderung)
			results <- llmResult{candidate: c, result: result, err: err}
		}(candidate)
	}

	// Wait for all to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	matches := make([]foerderung.FoerderungsMatch, 0, len(candidates))
	totalTokens := 0
	totalCost := 0

	for r := range results {
		match := foerderung.FoerderungsMatch{
			FoerderungID:   r.candidate.Foerderung.ID,
			FoerderungName: r.candidate.Foerderung.Name,
			Provider:       r.candidate.Foerderung.Provider,
			RuleScore:      r.candidate.FilterResult.TotalScore,
		}

		if r.err != nil {
			// LLM failed for this candidate - use rule score only
			match.LLMScore = 0
			match.TotalScore = match.RuleScore * s.config.RuleScoreWeight // Rule weight only
		} else if r.result != nil {
			match.LLMResult = r.result

			// Calculate LLM score from result
			llmScore := 0.0
			if r.result.Eligible {
				switch r.result.Confidence {
				case ConfidenceHigh:
					llmScore = 0.9
				case ConfidenceMedium:
					llmScore = 0.7
				default:
					llmScore = 0.5
				}
			} else {
				llmScore = 0.2
			}
			match.LLMScore = llmScore

			// Calculate total score
			match.TotalScore = (match.RuleScore * s.config.RuleScoreWeight) +
				(match.LLMScore * s.config.LLMScoreWeight)

			// TODO: Track actual token usage from LLM response
			totalTokens += 500 // Estimated
			totalCost += 3     // Estimated 3 cents per call
		}

		matches = append(matches, match)
	}

	return matches, totalTokens, totalCost, nil
}

// convertToRuleOnlyMatches converts candidates to matches without LLM
func (s *Service) convertToRuleOnlyMatches(candidates []*MatchCandidate) []foerderung.FoerderungsMatch {
	matches := make([]foerderung.FoerderungsMatch, 0, len(candidates))

	for _, c := range candidates {
		matches = append(matches, foerderung.FoerderungsMatch{
			FoerderungID:   c.Foerderung.ID,
			FoerderungName: c.Foerderung.Name,
			Provider:       c.Foerderung.Provider,
			RuleScore:      c.FilterResult.TotalScore,
			LLMScore:       0, // No LLM
			TotalScore:     c.FilterResult.TotalScore, // Rule score only
		})
	}

	return matches
}

func (s *Service) updateSearchError(ctx context.Context, search *foerderung.FoerderungsSuche, err error) {
	search.Status = foerderung.SearchStatusFailed
	errMsg := err.Error()
	search.ErrorMessage = &errMsg
	s.searchRepo.Update(ctx, search)
}

func sortMatchesByScore(matches []foerderung.FoerderungsMatch) {
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].TotalScore > matches[i].TotalScore {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
}

// GetSearch retrieves a search by ID
func (s *Service) GetSearch(ctx context.Context, id uuid.UUID) (*foerderung.FoerderungsSuche, error) {
	return s.searchRepo.GetByID(ctx, id)
}
