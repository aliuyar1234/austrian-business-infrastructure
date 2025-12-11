package antrag

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/foerderung"
)

// Handler handles application HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new application handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers application routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/antraege", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/stats", h.GetStats)
		r.Get("/{id}", h.Get)
		r.Put("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
		r.Post("/{id}/status", h.UpdateStatus)
		r.Post("/{id}/attachments", h.AddAttachment)
		r.Delete("/{id}/attachments/{name}", h.RemoveAttachment)
	})
}

// CreateRequest represents the create application request
type CreateRequest struct {
	ProfileID         string  `json:"profile_id"`
	FoerderungID      string  `json:"foerderung_id"`
	InternalReference *string `json:"internal_reference,omitempty"`
	RequestedAmount   *int    `json:"requested_amount,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

// UpdateRequest represents the update application request
type UpdateRequest struct {
	InternalReference *string `json:"internal_reference,omitempty"`
	RequestedAmount   *int    `json:"requested_amount,omitempty"`
	ApprovedAmount    *int    `json:"approved_amount,omitempty"`
	DecisionNotes     *string `json:"decision_notes,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

// StatusUpdateRequest represents the status update request
type StatusUpdateRequest struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

// AddAttachmentRequest represents the add attachment request
type AddAttachmentRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

// AntragResponse represents an application in API responses
type AntragResponse struct {
	ID                string                       `json:"id"`
	TenantID          string                       `json:"tenant_id"`
	ProfileID         string                       `json:"profile_id"`
	FoerderungID      string                       `json:"foerderung_id"`
	Status            string                       `json:"status"`
	InternalReference *string                      `json:"internal_reference,omitempty"`
	SubmittedAt       *string                      `json:"submitted_at,omitempty"`
	RequestedAmount   *int                         `json:"requested_amount,omitempty"`
	ApprovedAmount    *int                         `json:"approved_amount,omitempty"`
	DecisionDate      *string                      `json:"decision_date,omitempty"`
	DecisionNotes     *string                      `json:"decision_notes,omitempty"`
	Attachments       []foerderung.Attachment      `json:"attachments,omitempty"`
	Timeline          []foerderung.TimelineEntry   `json:"timeline,omitempty"`
	Notes             *string                      `json:"notes,omitempty"`
	CreatedAt         string                       `json:"created_at"`
	UpdatedAt         string                       `json:"updated_at"`
}

// ListResponse represents the list applications response
type ListResponse struct {
	Antraege []*AntragResponse `json:"antraege"`
	Total    int               `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// Create handles POST /api/v1/antraege
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	profileID, err := uuid.Parse(req.ProfileID)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid profile ID")
		return
	}

	foerderungID, err := uuid.Parse(req.FoerderungID)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid foerderung ID")
		return
	}

	userID := getUserIDFromContext(r)

	input := &CreateInput{
		TenantID:          tenantID,
		ProfileID:         profileID,
		FoerderungID:      foerderungID,
		InternalReference: req.InternalReference,
		RequestedAmount:   req.RequestedAmount,
		Notes:             req.Notes,
		CreatedBy:         userID,
	}

	antrag, err := h.service.Create(r.Context(), input)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusCreated, toAntragResponse(antrag))
}

// List handles GET /api/v1/antraege
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	q := r.URL.Query()
	filter := ListFilter{
		TenantID: tenantID,
		Status:   q.Get("status"),
	}

	if profileIDStr := q.Get("profile_id"); profileIDStr != "" {
		if profileID, err := uuid.Parse(profileIDStr); err == nil {
			filter.ProfileID = &profileID
		}
	}
	if foerderungIDStr := q.Get("foerderung_id"); foerderungIDStr != "" {
		if foerderungID, err := uuid.Parse(foerderungIDStr); err == nil {
			filter.FoerderungID = &foerderungID
		}
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
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	antraege, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, "Failed to list applications")
		return
	}

	resp := ListResponse{
		Antraege: make([]*AntragResponse, 0, len(antraege)),
		Total:    total,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	}
	for _, a := range antraege {
		resp.Antraege = append(resp.Antraege, toAntragResponse(a))
	}

	api.RespondJSON(w, http.StatusOK, resp)
}

// Get handles GET /api/v1/antraege/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	antrag, err := h.service.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		api.RespondError(w, http.StatusNotFound, "Application not found")
		return
	}

	api.RespondJSON(w, http.StatusOK, toAntragResponse(antrag))
}

// Update handles PUT /api/v1/antraege/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	userID := getUserIDFromContext(r)

	input := &UpdateInput{
		InternalReference: req.InternalReference,
		RequestedAmount:   req.RequestedAmount,
		ApprovedAmount:    req.ApprovedAmount,
		DecisionNotes:     req.DecisionNotes,
		Notes:             req.Notes,
	}

	antrag, err := h.service.Update(r.Context(), id, tenantID, input, userID)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, toAntragResponse(antrag))
}

// Delete handles DELETE /api/v1/antraege/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, tenantID); err != nil {
		api.RespondError(w, http.StatusNotFound, "Application not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateStatus handles POST /api/v1/antraege/{id}/status
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var req StatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Status == "" {
		api.RespondError(w, http.StatusBadRequest, "Status is required")
		return
	}

	userID := getUserIDFromContext(r)

	antrag, err := h.service.UpdateStatus(r.Context(), id, tenantID, req.Status, req.Description, userID)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, toAntragResponse(antrag))
}

// GetStats handles GET /api/v1/antraege/stats
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	stats, err := h.service.GetStats(r.Context(), tenantID)
	if err != nil {
		api.RespondError(w, http.StatusInternalServerError, "Failed to get stats")
		return
	}

	api.RespondJSON(w, http.StatusOK, stats)
}

// AddAttachment handles POST /api/v1/antraege/{id}/attachments
func (h *Handler) AddAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	var req AddAttachmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	attachment := foerderung.Attachment{
		Name: req.Name,
		Type: req.Type,
		URL:  req.URL,
	}

	antrag, err := h.service.AddAttachment(r.Context(), id, tenantID, attachment)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, toAntragResponse(antrag))
}

// RemoveAttachment handles DELETE /api/v1/antraege/{id}/attachments/{name}
func (h *Handler) RemoveAttachment(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		api.RespondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, "Invalid application ID")
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		api.RespondError(w, http.StatusBadRequest, "Attachment name is required")
		return
	}

	antrag, err := h.service.RemoveAttachment(r.Context(), id, tenantID, name)
	if err != nil {
		api.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.RespondJSON(w, http.StatusOK, toAntragResponse(antrag))
}

// Helper functions

func toAntragResponse(a *foerderung.FoerderungsAntrag) *AntragResponse {
	resp := &AntragResponse{
		ID:                a.ID.String(),
		TenantID:          a.TenantID.String(),
		ProfileID:         a.ProfileID.String(),
		FoerderungID:      a.FoerderungID.String(),
		Status:            a.Status,
		InternalReference: a.InternalReference,
		RequestedAmount:   a.RequestedAmount,
		ApprovedAmount:    a.ApprovedAmount,
		DecisionNotes:     a.DecisionNotes,
		Attachments:       a.Attachments,
		Timeline:          a.Timeline,
		Notes:             a.Notes,
		CreatedAt:         a.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:         a.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if a.SubmittedAt != nil {
		s := a.SubmittedAt.Format("2006-01-02T15:04:05Z")
		resp.SubmittedAt = &s
	}
	if a.DecisionDate != nil {
		s := a.DecisionDate.Format("2006-01-02T15:04:05Z")
		resp.DecisionDate = &s
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

