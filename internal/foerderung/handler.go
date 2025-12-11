package foerderung

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler provides HTTP handlers for Förderung operations
type Handler struct {
	repo               *Repository
	combinationService *CombinationService
}

// NewHandler creates a new Förderung handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:               repo,
		combinationService: NewCombinationService(repo),
	}
}

// CreateRequest is the request body for creating a Förderung
type CreateRequest struct {
	Name        string  `json:"name"`
	ShortName   *string `json:"short_name,omitempty"`
	Description *string `json:"description,omitempty"`
	Provider    string  `json:"provider"`
	Type        string  `json:"type"`

	FundingRateMin *float64 `json:"funding_rate_min,omitempty"`
	FundingRateMax *float64 `json:"funding_rate_max,omitempty"`
	MaxAmount      *int     `json:"max_amount,omitempty"`
	MinAmount      *int     `json:"min_amount,omitempty"`

	TargetSize       *string  `json:"target_size,omitempty"`
	TargetAge        *string  `json:"target_age,omitempty"`
	TargetLegalForms []string `json:"target_legal_forms,omitempty"`
	TargetIndustries []string `json:"target_industries,omitempty"`
	TargetStates     []string `json:"target_states,omitempty"`

	Topics     []string `json:"topics"`
	Categories []string `json:"categories,omitempty"`

	Requirements        *string         `json:"requirements,omitempty"`
	EligibilityCriteria json.RawMessage `json:"eligibility_criteria,omitempty"`

	ApplicationDeadline *string `json:"application_deadline,omitempty"`
	DeadlineType        *string `json:"deadline_type,omitempty"`
	CallStart           *string `json:"call_start,omitempty"`
	CallEnd             *string `json:"call_end,omitempty"`

	URL            *string `json:"url,omitempty"`
	ApplicationURL *string `json:"application_url,omitempty"`
	GuidelineURL   *string `json:"guideline_url,omitempty"`

	Status        *string `json:"status,omitempty"`
	IsHighlighted *bool   `json:"is_highlighted,omitempty"`
}

// ListResponse is the response for listing Förderungen
type ListResponse struct {
	Foerderungen []*Foerderung `json:"foerderungen"`
	Total        int           `json:"total"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}

// Create handles POST /api/v1/foerderungen
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	// TODO: Check admin role

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}

	// Build Förderung
	f := &Foerderung{
		Name:                req.Name,
		ShortName:           req.ShortName,
		Description:         req.Description,
		Provider:            req.Provider,
		Type:                FoerderungType(req.Type),
		FundingRateMin:      req.FundingRateMin,
		FundingRateMax:      req.FundingRateMax,
		MaxAmount:           req.MaxAmount,
		MinAmount:           req.MinAmount,
		TargetLegalForms:    req.TargetLegalForms,
		TargetIndustries:    req.TargetIndustries,
		TargetStates:        req.TargetStates,
		Topics:              req.Topics,
		Categories:          req.Categories,
		Requirements:        req.Requirements,
		EligibilityCriteria: req.EligibilityCriteria,
		URL:                 req.URL,
		ApplicationURL:      req.ApplicationURL,
		GuidelineURL:        req.GuidelineURL,
		Status:              StatusActive,
	}

	if req.TargetSize != nil {
		ts := TargetSize(*req.TargetSize)
		f.TargetSize = &ts
	}
	if req.TargetAge != nil {
		f.TargetAge = req.TargetAge
	}
	if req.DeadlineType != nil {
		dt := DeadlineType(*req.DeadlineType)
		f.DeadlineType = &dt
	}
	if req.Status != nil {
		f.Status = FoerderungStatus(*req.Status)
	}
	if req.IsHighlighted != nil {
		f.IsHighlighted = *req.IsHighlighted
	}

	// Parse dates
	if req.ApplicationDeadline != nil {
		t, err := parseDate(*req.ApplicationDeadline)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid application_deadline format")
			return
		}
		f.ApplicationDeadline = &t
	}
	if req.CallStart != nil {
		t, err := parseDate(*req.CallStart)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid call_start format")
			return
		}
		f.CallStart = &t
	}
	if req.CallEnd != nil {
		t, err := parseDate(*req.CallEnd)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid call_end format")
			return
		}
		f.CallEnd = &t
	}

	if err := h.repo.Create(r.Context(), f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, f)
}

// List handles GET /api/v1/foerderungen
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := ListFilter{
		Provider: q.Get("provider"),
		State:    q.Get("state"),
		Topic:    q.Get("topic"),
		Search:   q.Get("search"),
	}

	if t := q.Get("type"); t != "" {
		filter.Type = FoerderungType(t)
	}
	if s := q.Get("status"); s != "" {
		filter.Status = FoerderungStatus(s)
	}
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			filter.Limit = parsed
		}
	}
	if o := q.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			filter.Offset = parsed
		}
	}

	foerderungen, total, err := h.repo.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Foerderungen: foerderungen,
		Total:        total,
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	})
}

// Get handles GET /api/v1/foerderungen/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid foerderung id")
		return
	}

	f, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "foerderung not found")
		return
	}

	writeJSON(w, http.StatusOK, f)
}

// Update handles PUT /api/v1/foerderungen/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	// TODO: Check admin role

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid foerderung id")
		return
	}

	// Get existing
	f, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "foerderung not found")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if req.Name != "" {
		f.Name = req.Name
	}
	if req.ShortName != nil {
		f.ShortName = req.ShortName
	}
	if req.Description != nil {
		f.Description = req.Description
	}
	if req.Provider != "" {
		f.Provider = req.Provider
	}
	if req.Type != "" {
		f.Type = FoerderungType(req.Type)
	}
	if req.FundingRateMin != nil {
		f.FundingRateMin = req.FundingRateMin
	}
	if req.FundingRateMax != nil {
		f.FundingRateMax = req.FundingRateMax
	}
	if req.MaxAmount != nil {
		f.MaxAmount = req.MaxAmount
	}
	if req.MinAmount != nil {
		f.MinAmount = req.MinAmount
	}
	if req.TargetSize != nil {
		ts := TargetSize(*req.TargetSize)
		f.TargetSize = &ts
	}
	if req.TargetAge != nil {
		f.TargetAge = req.TargetAge
	}
	if req.TargetLegalForms != nil {
		f.TargetLegalForms = req.TargetLegalForms
	}
	if req.TargetIndustries != nil {
		f.TargetIndustries = req.TargetIndustries
	}
	if req.TargetStates != nil {
		f.TargetStates = req.TargetStates
	}
	if req.Topics != nil {
		f.Topics = req.Topics
	}
	if req.Categories != nil {
		f.Categories = req.Categories
	}
	if req.Requirements != nil {
		f.Requirements = req.Requirements
	}
	if req.EligibilityCriteria != nil {
		f.EligibilityCriteria = req.EligibilityCriteria
	}
	if req.DeadlineType != nil {
		dt := DeadlineType(*req.DeadlineType)
		f.DeadlineType = &dt
	}
	if req.URL != nil {
		f.URL = req.URL
	}
	if req.ApplicationURL != nil {
		f.ApplicationURL = req.ApplicationURL
	}
	if req.GuidelineURL != nil {
		f.GuidelineURL = req.GuidelineURL
	}
	if req.Status != nil {
		f.Status = FoerderungStatus(*req.Status)
	}
	if req.IsHighlighted != nil {
		f.IsHighlighted = *req.IsHighlighted
	}

	// Parse dates
	if req.ApplicationDeadline != nil {
		t, err := parseDate(*req.ApplicationDeadline)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid application_deadline format")
			return
		}
		f.ApplicationDeadline = &t
	}
	if req.CallStart != nil {
		t, err := parseDate(*req.CallStart)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid call_start format")
			return
		}
		f.CallStart = &t
	}
	if req.CallEnd != nil {
		t, err := parseDate(*req.CallEnd)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid call_end format")
			return
		}
		f.CallEnd = &t
	}

	if err := h.repo.Update(r.Context(), f); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, f)
}

// Delete handles DELETE /api/v1/foerderungen/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	// TODO: Check admin role

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid foerderung id")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetStats handles GET /api/v1/foerderungen/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Helper functions

func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// GetCombinations handles GET /api/v1/foerderungen/{id}/combinations
func (h *Handler) GetCombinations(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid foerderung id")
		return
	}

	analysis, err := h.combinationService.GetCombinablePrograms(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "foerderung not found")
		return
	}

	writeJSON(w, http.StatusOK, analysis)
}

// ValidateCombinationRequest is the request body for combination validation
type ValidateCombinationRequest struct {
	FoerderungID1 string `json:"foerderung_id_1"`
	FoerderungID2 string `json:"foerderung_id_2"`
}

// ValidateCombination handles POST /api/v1/foerderungen/validate-combination
func (h *Handler) ValidateCombination(w http.ResponseWriter, r *http.Request) {
	var req ValidateCombinationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id1, err := uuid.Parse(req.FoerderungID1)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid foerderung_id_1")
		return
	}

	id2, err := uuid.Parse(req.FoerderungID2)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid foerderung_id_2")
		return
	}

	validation, err := h.combinationService.ValidateCombination(r.Context(), id1, id2)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, validation)
}

// RegisterRoutes registers foerderung routes with chi router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/foerderungen", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/stats", h.GetStats)
		r.Post("/validate-combination", h.ValidateCombination)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Get("/{id}/combinations", h.GetCombinations)
	})
}
