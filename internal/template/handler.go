package template

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// Handler provides HTTP handlers for signature template operations
type Handler struct {
	repo *Repository
}

// NewHandler creates a new template handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// CreateTemplateRequest is the request body for creating a template
type CreateTemplateRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Signers     []SignerTemplate   `json:"signers"`
	Fields      []FieldTemplate    `json:"fields,omitempty"`
	Settings    *TemplateSettings  `json:"settings,omitempty"`
}

// UpdateTemplateRequest is the request body for updating a template
type UpdateTemplateRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Signers     []SignerTemplate   `json:"signers"`
	Fields      []FieldTemplate    `json:"fields,omitempty"`
	Settings    *TemplateSettings  `json:"settings,omitempty"`
	IsActive    *bool              `json:"is_active,omitempty"`
}

// TemplateResponse is the response for a template
type TemplateResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Signers     []SignerTemplate   `json:"signers"`
	Fields      []FieldTemplate    `json:"fields,omitempty"`
	Settings    *TemplateSettings  `json:"settings,omitempty"`
	IsActive    bool               `json:"is_active"`
	UsageCount  int                `json:"usage_count"`
	CreatedBy   string             `json:"created_by"`
	CreatedAt   string             `json:"created_at"`
	UpdatedAt   string             `json:"updated_at"`
}

// TemplateListResponse is the response for listing templates
type TemplateListResponse struct {
	Templates []*TemplateResponse `json:"templates"`
	Total     int                 `json:"total"`
	Limit     int                 `json:"limit"`
	Offset    int                 `json:"offset"`
}

// Create handles POST /api/v1/signature-templates
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Signers) == 0 {
		writeError(w, http.StatusBadRequest, "at least one signer is required")
		return
	}

	// Create template
	template := &SignatureTemplate{
		TenantID:    tenantID,
		Name:        req.Name,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := template.SetSignerTemplates(req.Signers); err != nil {
		writeError(w, http.StatusBadRequest, "invalid signers")
		return
	}

	if len(req.Fields) > 0 {
		if err := template.SetFieldTemplates(req.Fields); err != nil {
			writeError(w, http.StatusBadRequest, "invalid fields")
			return
		}
	}

	if req.Settings != nil {
		if err := template.SetSettings(req.Settings); err != nil {
			writeError(w, http.StatusBadRequest, "invalid settings")
			return
		}
	} else {
		// Set defaults
		if err := template.SetSettings(&TemplateSettings{
			ExpiryDays:      14,
			NotifyRequester: true,
			AutoRemind:      true,
			RemindDays:      7,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to set default settings")
			return
		}
	}

	if err := h.repo.Create(r.Context(), template); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toTemplateResponse(template))
}

// List handles GET /api/v1/signature-templates
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query parameters
	limit := 50
	offset := 0
	activeOnly := true

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	if a := r.URL.Query().Get("active_only"); a == "false" {
		activeOnly = false
	}

	templates, total, err := h.repo.ListByTenant(r.Context(), tenantID, activeOnly, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]*TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = toTemplateResponse(t)
	}

	writeJSON(w, http.StatusOK, TemplateListResponse{
		Templates: responses,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	})
}

// Get handles GET /api/v1/signature-templates/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}

	template, err := h.repo.GetByIDAndTenant(r.Context(), templateID, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	writeJSON(w, http.StatusOK, toTemplateResponse(template))
}

// Update handles PUT /api/v1/signature-templates/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}

	var req UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get existing template
	template, err := h.repo.GetByIDAndTenant(r.Context(), templateID, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	// Update fields
	if req.Name != "" {
		template.Name = req.Name
	}
	template.Description = req.Description

	if len(req.Signers) > 0 {
		if err := template.SetSignerTemplates(req.Signers); err != nil {
			writeError(w, http.StatusBadRequest, "invalid signers")
			return
		}
	}

	if len(req.Fields) > 0 {
		if err := template.SetFieldTemplates(req.Fields); err != nil {
			writeError(w, http.StatusBadRequest, "invalid fields")
			return
		}
	}

	if req.Settings != nil {
		if err := template.SetSettings(req.Settings); err != nil {
			writeError(w, http.StatusBadRequest, "invalid settings")
			return
		}
	}

	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if err := h.repo.Update(r.Context(), template); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toTemplateResponse(template))
}

// Delete handles DELETE /api/v1/signature-templates/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	templateID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template id")
		return
	}

	// Verify ownership
	_, err = h.repo.GetByIDAndTenant(r.Context(), templateID, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "template not found")
		return
	}

	if err := h.repo.Delete(r.Context(), templateID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Search handles GET /api/v1/signature-templates/search
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "search query is required")
		return
	}

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	templates, err := h.repo.Search(r.Context(), tenantID, query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]*TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = toTemplateResponse(t)
	}

	writeJSON(w, http.StatusOK, responses)
}

// GetMostUsed handles GET /api/v1/signature-templates/popular
func (h *Handler) GetMostUsed(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	templates, err := h.repo.GetMostUsed(r.Context(), tenantID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]*TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = toTemplateResponse(t)
	}

	writeJSON(w, http.StatusOK, responses)
}

func toTemplateResponse(t *SignatureTemplate) *TemplateResponse {
	signers, _ := t.GetSignerTemplates()
	fields, _ := t.GetFieldTemplates()
	settings, _ := t.GetSettings()

	return &TemplateResponse{
		ID:          t.ID.String(),
		Name:        t.Name,
		Description: t.Description,
		Signers:     signers,
		Fields:      fields,
		Settings:    settings,
		IsActive:    t.IsActive,
		UsageCount:  t.UsageCount,
		CreatedBy:   t.CreatedBy.String(),
		CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   t.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Helper functions

func getContextIDs(r *http.Request) (tenantID uuid.UUID, userID uuid.UUID, ok bool) {
	tenantIDValue := r.Context().Value("tenant_id")
	userIDValue := r.Context().Value("user_id")

	if tenantIDValue == nil || userIDValue == nil {
		return uuid.Nil, uuid.Nil, false
	}

	tenantID, ok1 := tenantIDValue.(uuid.UUID)
	userID, ok2 := userIDValue.(uuid.UUID)

	return tenantID, userID, ok1 && ok2
}

func getPathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
