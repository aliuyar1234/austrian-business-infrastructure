package matcher

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// Handler handles matcher HTTP requests
type Handler struct {
	service     *Service
	profileRepo ProfileRepository
}

// ProfileRepository interface for profile access
type ProfileRepository interface {
	GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.Unternehmensprofil, error)
	UpdateLastSearchAt(ctx context.Context, id uuid.UUID) error
}

// NewHandler creates a new matcher handler
func NewHandler(service *Service, profileRepo ProfileRepository) *Handler {
	return &Handler{
		service:     service,
		profileRepo: profileRepo,
	}
}

// RegisterRoutes registers matcher routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/foerderungssuche", func(r chi.Router) {
		r.Post("/", h.Search)
		r.Get("/{id}", h.GetSearch)
		r.Get("/{id}/export", h.ExportSearch)
		r.Get("/", h.ListSearches)
	})
}

// SearchRequest represents the search request
type SearchRequest struct {
	ProfileID          string   `json:"profile_id"`
	// Inline profile data (alternative to profile_id)
	CompanyName        string   `json:"company_name,omitempty"`
	State              string   `json:"state,omitempty"`
	EmployeesCount     *int     `json:"employees_count,omitempty"`
	AnnualRevenue      *int     `json:"annual_revenue,omitempty"`
	FoundedYear        *int     `json:"founded_year,omitempty"`
	Industry           string   `json:"industry,omitempty"`
	IsStartup          bool     `json:"is_startup,omitempty"`
	ProjectDescription string   `json:"project_description,omitempty"`
	ProjectTopics      []string `json:"project_topics,omitempty"`
	InvestmentAmount   *int     `json:"investment_amount,omitempty"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	SearchID      string           `json:"search_id"`
	TotalChecked  int              `json:"total_checked"`
	TotalMatches  int              `json:"total_matches"`
	Matches       []MatchResponse  `json:"matches"`
	LLMTokensUsed int              `json:"llm_tokens_used"`
	LLMCostCents  int              `json:"llm_cost_cents"`
	DurationMs    int64            `json:"duration_ms"`
	LLMFallback   bool             `json:"llm_fallback"`
}

// MatchResponse represents a match in the search response
type MatchResponse struct {
	FoerderungID   string        `json:"foerderung_id"`
	FoerderungName string        `json:"foerderung_name"`
	Provider       string        `json:"provider"`
	RuleScore      float64       `json:"rule_score"`
	LLMScore       float64       `json:"llm_score"`
	TotalScore     float64       `json:"total_score"`
	LLMResult      *LLMResponse  `json:"llm_result,omitempty"`
}

// LLMResponse represents the LLM analysis in the response
type LLMResponse struct {
	Eligible        bool     `json:"eligible"`
	Confidence      string   `json:"confidence"`
	MatchedCriteria []string `json:"matched_criteria"`
	ImplicitMatches []string `json:"implicit_matches,omitempty"`
	Concerns        []string `json:"concerns,omitempty"`
	EstimatedAmount *int     `json:"estimated_amount,omitempty"`
	CombinationHint *string  `json:"combination_hint,omitempty"`
	NextSteps       []string `json:"next_steps,omitempty"`
	InsiderTip      *string  `json:"insider_tip,omitempty"`
}

// SearchListResponse represents the list of searches
type SearchListResponse struct {
	Searches []*SearchSummary `json:"searches"`
	Total    int              `json:"total"`
	Limit    int              `json:"limit"`
	Offset   int              `json:"offset"`
}

// SearchSummary represents a search in list view
type SearchSummary struct {
	ID               string  `json:"id"`
	ProfileID        string  `json:"profile_id"`
	TotalFoerderungen int    `json:"total_foerderungen"`
	TotalMatches     int     `json:"total_matches"`
	Status           string  `json:"status"`
	CreatedAt        string  `json:"created_at"`
}

// Search handles POST /api/v1/foerderungssuche
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID := getUserIDFromContext(r)

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var profile *ProfileInput
	var profileID uuid.UUID

	// Use existing profile or inline data
	if req.ProfileID != "" {
		profileID, err = uuid.Parse(req.ProfileID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid profile ID")
			return
		}

		// Get profile from database
		p, err := h.profileRepo.GetByIDAndTenant(r.Context(), profileID, tenantID)
		if err != nil {
			writeError(w, http.StatusNotFound, "Profile not found")
			return
		}

		profile = profileFromUnternehmensprofil(p)

		// Update last search timestamp
		h.profileRepo.UpdateLastSearchAt(r.Context(), profileID)
	} else if req.CompanyName != "" {
		// Use inline profile data
		profileID = uuid.New() // Generate temporary ID
		profile = &ProfileInput{
			CompanyName:        req.CompanyName,
			State:              req.State,
			EmployeesCount:     req.EmployeesCount,
			AnnualRevenue:      req.AnnualRevenue,
			FoundedYear:        req.FoundedYear,
			Industry:           req.Industry,
			IsStartup:          req.IsStartup,
			ProjectDescription: req.ProjectDescription,
			ProjectTopics:      req.ProjectTopics,
			InvestmentAmount:   req.InvestmentAmount,
		}
	} else {
		writeError(w, http.StatusBadRequest, "Either profile_id or company_name is required")
		return
	}

	// Run search
	input := &SearchInput{
		TenantID:  tenantID,
		ProfileID: profileID,
		Profile:   profile,
		CreatedBy: userID,
	}

	output, err := h.service.RunSearch(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Search failed: "+err.Error())
		return
	}

	// Convert to response
	resp := toSearchResponse(output)
	writeJSON(w, http.StatusOK, resp)
}

// GetSearch handles GET /api/v1/foerderungssuche/{id}
func (h *Handler) GetSearch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid search ID")
		return
	}

	search, err := h.service.searchRepo.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Search not found")
		return
	}

	// Parse matches from JSON
	matches, _ := search.GetMatchesSlice()

	resp := SearchResponse{
		SearchID:      search.ID.String(),
		TotalChecked:  search.TotalFoerderungen,
		TotalMatches:  search.TotalMatches,
		LLMTokensUsed: search.LLMTokensUsed,
		LLMCostCents:  search.LLMCostCents,
		Matches:       make([]MatchResponse, 0, len(matches)),
	}

	for _, m := range matches {
		resp.Matches = append(resp.Matches, toMatchResponse(m))
	}

	writeJSON(w, http.StatusOK, resp)
}

// ExportSearch handles GET /api/v1/foerderungssuche/{id}/export
func (h *Handler) ExportSearch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid search ID")
		return
	}

	search, err := h.service.searchRepo.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Search not found")
		return
	}

	// Parse matches from JSON
	matches, err := search.GetMatchesSlice()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to parse search results")
		return
	}

	// Get format from query params (default to markdown)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "markdown"
	}

	switch format {
	case "markdown":
		h.exportMarkdown(w, search, matches)
	case "pdf":
		// PDF export is not yet implemented - return markdown with appropriate message
		writeError(w, http.StatusNotImplemented, "PDF export is not yet implemented. Please use format=markdown")
	default:
		writeError(w, http.StatusBadRequest, "Invalid format. Use 'markdown' or 'pdf'")
	}
}

// exportMarkdown exports search results as markdown
func (h *Handler) exportMarkdown(w http.ResponseWriter, search *foerderung.FoerderungsSuche, matches []foerderung.FoerderungsMatch) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=foerderungssuche-"+search.ID.String()+".md")

	// Build markdown content
	md := "# Foerderungssuche Export\n\n"
	md += "**Suche ID:** " + search.ID.String() + "\n"
	md += "**Durchsucht:** " + strconv.Itoa(search.TotalFoerderungen) + " Foerderungen\n"
	md += "**Treffer:** " + strconv.Itoa(search.TotalMatches) + "\n"
	md += "**Erstellt am:** " + search.CreatedAt.Format("02.01.2006 15:04") + "\n\n"

	if len(matches) == 0 {
		md += "Keine passenden Foerderungen gefunden.\n"
	} else {
		md += "## Passende Foerderungen\n\n"
		for i, m := range matches {
			md += "### " + strconv.Itoa(i+1) + ". " + m.FoerderungName + "\n\n"
			md += "- **Anbieter:** " + m.Provider + "\n"
			md += "- **Gesamtbewertung:** " + strconv.FormatFloat(m.TotalScore*100, 'f', 0, 64) + "%\n"
			md += "- **Regel-Score:** " + strconv.FormatFloat(m.RuleScore*100, 'f', 0, 64) + "%\n"
			if m.LLMScore > 0 {
				md += "- **KI-Score:** " + strconv.FormatFloat(m.LLMScore*100, 'f', 0, 64) + "%\n"
			}

			if m.LLMResult != nil {
				md += "\n**KI-Analyse:**\n"
				if m.LLMResult.Eligible {
					md += "- Foerderberechtigt: Ja (" + m.LLMResult.Confidence + " Konfidenz)\n"
				} else {
					md += "- Foerderberechtigt: Nein\n"
				}

				if len(m.LLMResult.MatchedCriteria) > 0 {
					md += "- Erfuellte Kriterien:\n"
					for _, c := range m.LLMResult.MatchedCriteria {
						md += "  - " + c + "\n"
					}
				}

				if len(m.LLMResult.Concerns) > 0 {
					md += "- Bedenken:\n"
					for _, c := range m.LLMResult.Concerns {
						md += "  - " + c + "\n"
					}
				}

				if m.LLMResult.EstimatedAmount != nil {
					md += "- Geschaetzter Betrag: EUR " + strconv.Itoa(*m.LLMResult.EstimatedAmount) + "\n"
				}

				if len(m.LLMResult.NextSteps) > 0 {
					md += "- Naechste Schritte:\n"
					for _, s := range m.LLMResult.NextSteps {
						md += "  - " + s + "\n"
					}
				}

				if m.LLMResult.InsiderTip != nil {
					md += "- **Tipp:** " + *m.LLMResult.InsiderTip + "\n"
				}
			}
			md += "\n---\n\n"
		}
	}

	w.Write([]byte(md))
}

// ListSearches handles GET /api/v1/foerderungssuche
func (h *Handler) ListSearches(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 {
		limit = 20
	}

	// Check for profile_id filter
	var searches []*foerderung.FoerderungsSuche
	var total int

	if profileIDStr := r.URL.Query().Get("profile_id"); profileIDStr != "" {
		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid profile ID")
			return
		}
		searches, total, err = h.service.searchRepo.ListByProfile(r.Context(), profileID, limit, offset)
	} else {
		searches, total, err = h.service.searchRepo.ListByTenant(r.Context(), tenantID, limit, offset)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list searches")
		return
	}

	resp := SearchListResponse{
		Searches: make([]*SearchSummary, 0, len(searches)),
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}

	for _, s := range searches {
		resp.Searches = append(resp.Searches, &SearchSummary{
			ID:               s.ID.String(),
			ProfileID:        s.ProfileID.String(),
			TotalFoerderungen: s.TotalFoerderungen,
			TotalMatches:     s.TotalMatches,
			Status:           s.Status,
			CreatedAt:        s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

// Helper functions

func profileFromUnternehmensprofil(p *foerderung.Unternehmensprofil) *ProfileInput {
	profile := &ProfileInput{
		CompanyName:        p.Name,
		EmployeesCount:     p.EmployeesCount,
		AnnualRevenue:      p.AnnualRevenue,
		BalanceTotal:       p.BalanceTotal,
		FoundedYear:        p.FoundedYear,
		IsStartup:          p.IsStartup,
		ProjectTopics:      p.ProjectTopics,
		InvestmentAmount:   p.InvestmentAmount,
		OnaceCodes:         p.OnaceCodes,
	}

	if p.State != nil {
		profile.State = *p.State
	}
	if p.LegalForm != nil {
		profile.LegalForm = *p.LegalForm
	}
	if p.Industry != nil {
		profile.Industry = *p.Industry
	}
	if p.ProjectDescription != nil {
		profile.ProjectDescription = *p.ProjectDescription
	}
	if p.IsKMU != nil {
		profile.IsKMU = p.IsKMU
	}

	return profile
}

func toSearchResponse(output *SearchOutput) *SearchResponse {
	resp := &SearchResponse{
		SearchID:      output.SearchID.String(),
		TotalChecked:  output.TotalChecked,
		TotalMatches:  output.TotalMatches,
		LLMTokensUsed: output.LLMTokensUsed,
		LLMCostCents:  output.LLMCostCents,
		DurationMs:    output.Duration.Milliseconds(),
		LLMFallback:   output.LLMFallback,
		Matches:       make([]MatchResponse, 0, len(output.Matches)),
	}

	for _, m := range output.Matches {
		resp.Matches = append(resp.Matches, toMatchResponse(m))
	}

	return resp
}

func toMatchResponse(m foerderung.FoerderungsMatch) MatchResponse {
	resp := MatchResponse{
		FoerderungID:   m.FoerderungID.String(),
		FoerderungName: m.FoerderungName,
		Provider:       m.Provider,
		RuleScore:      m.RuleScore,
		LLMScore:       m.LLMScore,
		TotalScore:     m.TotalScore,
	}

	if m.LLMResult != nil {
		resp.LLMResult = &LLMResponse{
			Eligible:        m.LLMResult.Eligible,
			Confidence:      m.LLMResult.Confidence,
			MatchedCriteria: m.LLMResult.MatchedCriteria,
			ImplicitMatches: m.LLMResult.ImplicitMatches,
			Concerns:        m.LLMResult.Concerns,
			EstimatedAmount: m.LLMResult.EstimatedAmount,
			CombinationHint: m.LLMResult.CombinationHint,
			NextSteps:       m.LLMResult.NextSteps,
			InsiderTip:      m.LLMResult.InsiderTip,
		}
	}

	return resp
}

// Context helper functions

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	userIDKey   contextKey = "user_id"
)

func getTenantIDFromContext(r *http.Request) (uuid.UUID, error) {
	v := r.Context().Value(tenantIDKey)
	if v == nil {
		// Try header fallback for testing
		if h := r.Header.Get("X-Tenant-ID"); h != "" {
			return uuid.Parse(h)
		}
		return uuid.Nil, nil
	}
	if id, ok := v.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, nil
}

func getUserIDFromContext(r *http.Request) *uuid.UUID {
	v := r.Context().Value(userIDKey)
	if v == nil {
		return nil
	}
	if id, ok := v.(uuid.UUID); ok {
		return &id
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
