package profil

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// Handler handles profile HTTP requests
type Handler struct {
	service       *Service
	deriveService *DeriveService
}

// NewHandler creates a new profile handler
func NewHandler(service *Service, deriveService *DeriveService) *Handler {
	return &Handler{
		service:       service,
		deriveService: deriveService,
	}
}

// RegisterRoutes registers profile routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/profile", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/derive/{accountId}", h.DeriveFromAccount)
	})
}

// CreateRequest represents the create profile request
type CreateRequest struct {
	AccountID          *string  `json:"account_id,omitempty"`
	Name               string   `json:"name"`
	LegalForm          *string  `json:"legal_form,omitempty"`
	FoundedYear        *int     `json:"founded_year,omitempty"`
	State              *string  `json:"state,omitempty"`
	District           *string  `json:"district,omitempty"`
	EmployeesCount     *int     `json:"employees_count,omitempty"`
	AnnualRevenue      *int     `json:"annual_revenue,omitempty"`
	BalanceTotal       *int     `json:"balance_total,omitempty"`
	Industry           *string  `json:"industry,omitempty"`
	OnaceCodes         []string `json:"onace_codes,omitempty"`
	IsStartup          bool     `json:"is_startup"`
	ProjectDescription *string  `json:"project_description,omitempty"`
	InvestmentAmount   *int     `json:"investment_amount,omitempty"`
	ProjectTopics      []string `json:"project_topics,omitempty"`
}

// UpdateRequest represents the update profile request
type UpdateRequest struct {
	Name               *string  `json:"name,omitempty"`
	LegalForm          *string  `json:"legal_form,omitempty"`
	FoundedYear        *int     `json:"founded_year,omitempty"`
	State              *string  `json:"state,omitempty"`
	District           *string  `json:"district,omitempty"`
	EmployeesCount     *int     `json:"employees_count,omitempty"`
	AnnualRevenue      *int     `json:"annual_revenue,omitempty"`
	BalanceTotal       *int     `json:"balance_total,omitempty"`
	Industry           *string  `json:"industry,omitempty"`
	OnaceCodes         []string `json:"onace_codes,omitempty"`
	IsStartup          *bool    `json:"is_startup,omitempty"`
	ProjectDescription *string  `json:"project_description,omitempty"`
	InvestmentAmount   *int     `json:"investment_amount,omitempty"`
	ProjectTopics      []string `json:"project_topics,omitempty"`
}

// ListResponse represents the list profiles response
type ListResponse struct {
	Profiles []*ProfileResponse `json:"profiles"`
	Total    int                `json:"total"`
	Limit    int                `json:"limit"`
	Offset   int                `json:"offset"`
}

// ProfileResponse represents a profile in API responses
type ProfileResponse struct {
	ID                 string   `json:"id"`
	TenantID           string   `json:"tenant_id"`
	AccountID          *string  `json:"account_id,omitempty"`
	Name               string   `json:"name"`
	LegalForm          *string  `json:"legal_form,omitempty"`
	FoundedYear        *int     `json:"founded_year,omitempty"`
	State              *string  `json:"state,omitempty"`
	District           *string  `json:"district,omitempty"`
	EmployeesCount     *int     `json:"employees_count,omitempty"`
	AnnualRevenue      *int     `json:"annual_revenue,omitempty"`
	BalanceTotal       *int     `json:"balance_total,omitempty"`
	Industry           *string  `json:"industry,omitempty"`
	OnaceCodes         []string `json:"onace_codes,omitempty"`
	IsStartup          bool     `json:"is_startup"`
	ProjectDescription *string  `json:"project_description,omitempty"`
	InvestmentAmount   *int     `json:"investment_amount,omitempty"`
	ProjectTopics      []string `json:"project_topics,omitempty"`
	IsKMU              *bool    `json:"is_kmu,omitempty"`
	CompanyAgeCategory *string  `json:"company_age_category,omitempty"`
	Status             string   `json:"status"`
	DerivedFromAccount bool     `json:"derived_from_account"`
	LastSearchAt       *string  `json:"last_search_at,omitempty"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
}

// Create handles POST /api/v1/profile
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	userID := getUserIDFromContext(r)

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	input := &CreateInput{
		TenantID:           tenantID,
		Name:               req.Name,
		LegalForm:          req.LegalForm,
		FoundedYear:        req.FoundedYear,
		State:              req.State,
		District:           req.District,
		EmployeesCount:     req.EmployeesCount,
		AnnualRevenue:      req.AnnualRevenue,
		BalanceTotal:       req.BalanceTotal,
		Industry:           req.Industry,
		OnaceCodes:         req.OnaceCodes,
		IsStartup:          req.IsStartup,
		ProjectDescription: req.ProjectDescription,
		InvestmentAmount:   req.InvestmentAmount,
		ProjectTopics:      req.ProjectTopics,
		CreatedBy:          userID,
	}

	if req.AccountID != nil {
		accountUUID, err := uuid.Parse(*req.AccountID)
		if err == nil {
			input.AccountID = &accountUUID
		}
	}

	profile, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toProfileResponse(profile))
}

// List handles GET /api/v1/profile
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
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

	profiles, total, err := h.service.ListByTenant(r.Context(), tenantID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list profiles")
		return
	}

	resp := ListResponse{
		Profiles: make([]*ProfileResponse, 0, len(profiles)),
		Total:    total,
		Limit:    limit,
		Offset:   offset,
	}
	for _, p := range profiles {
		resp.Profiles = append(resp.Profiles, toProfileResponse(p))
	}

	writeJSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/profile/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	profile, err := h.service.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Profile not found")
		return
	}

	writeJSON(w, http.StatusOK, toProfileResponse(profile))
}

// Update handles PUT /api/v1/profile/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	input := &UpdateInput{
		Name:               req.Name,
		LegalForm:          req.LegalForm,
		FoundedYear:        req.FoundedYear,
		State:              req.State,
		District:           req.District,
		EmployeesCount:     req.EmployeesCount,
		AnnualRevenue:      req.AnnualRevenue,
		BalanceTotal:       req.BalanceTotal,
		Industry:           req.Industry,
		OnaceCodes:         req.OnaceCodes,
		IsStartup:          req.IsStartup,
		ProjectDescription: req.ProjectDescription,
		InvestmentAmount:   req.InvestmentAmount,
		ProjectTopics:      req.ProjectTopics,
	}

	profile, err := h.service.Update(r.Context(), id, tenantID, input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toProfileResponse(profile))
}

// Delete handles DELETE /api/v1/profile/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, tenantID); err != nil {
		writeError(w, http.StatusNotFound, "Profile not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeriveFromAccount handles POST /api/v1/profile/derive/{accountId}
func (h *Handler) DeriveFromAccount(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	accountID, err := uuid.Parse(chi.URLParam(r, "accountId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid account ID")
		return
	}

	userID := getUserIDFromContext(r)

	if h.deriveService == nil {
		writeError(w, http.StatusNotImplemented, "Profile derivation not configured")
		return
	}

	profile, err := h.deriveService.DeriveFromAccount(r.Context(), tenantID, accountID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to derive profile: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toProfileResponse(profile))
}

// Helper functions

func toProfileResponse(p *foerderung.Unternehmensprofil) *ProfileResponse {
	resp := &ProfileResponse{
		ID:                 p.ID.String(),
		TenantID:           p.TenantID.String(),
		Name:               p.Name,
		LegalForm:          p.LegalForm,
		FoundedYear:        p.FoundedYear,
		State:              p.State,
		District:           p.District,
		EmployeesCount:     p.EmployeesCount,
		AnnualRevenue:      p.AnnualRevenue,
		BalanceTotal:       p.BalanceTotal,
		Industry:           p.Industry,
		OnaceCodes:         p.OnaceCodes,
		IsStartup:          p.IsStartup,
		ProjectDescription: p.ProjectDescription,
		InvestmentAmount:   p.InvestmentAmount,
		ProjectTopics:      p.ProjectTopics,
		IsKMU:              p.IsKMU,
		CompanyAgeCategory: p.CompanyAgeCategory,
		Status:             p.Status,
		DerivedFromAccount: p.DerivedFromAccount,
		CreatedAt:          p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if p.AccountID != nil {
		s := p.AccountID.String()
		resp.AccountID = &s
	}
	if p.LastSearchAt != nil {
		s := p.LastSearchAt.Format("2006-01-02T15:04:05Z")
		resp.LastSearchAt = &s
	}

	return resp
}

// Context helper functions - these should match the auth middleware pattern

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
