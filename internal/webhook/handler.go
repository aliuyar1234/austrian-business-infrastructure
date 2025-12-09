package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles webhook HTTP requests
type Handler struct {
	repo    *Repository
	service *Service
}

// NewHandler creates a new webhook handler
func NewHandler(repo *Repository, service *Service) *Handler {
	return &Handler{
		repo:    repo,
		service: service,
	}
}

// RegisterRoutes registers webhook routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/webhooks", h.List)
	mux.HandleFunc("POST /api/v1/webhooks", h.Create)
	mux.HandleFunc("GET /api/v1/webhooks/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/webhooks/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/webhooks/{id}", h.Delete)
	mux.HandleFunc("POST /api/v1/webhooks/{id}/rotate-secret", h.RotateSecret)
	mux.HandleFunc("GET /api/v1/webhooks/{id}/deliveries", h.ListDeliveries)
	mux.HandleFunc("POST /api/v1/webhooks/{id}/test", h.TestWebhook)
}

// WebhookResponse represents a webhook in API responses
type WebhookResponse struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Events         []string          `json:"events"`
	Enabled        bool              `json:"enabled"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	MaxRetries     int               `json:"max_retries"`
	Headers        map[string]string `json:"headers,omitempty"`
	CreatedAt      string            `json:"created_at"`
	UpdatedAt      string            `json:"updated_at"`
}

// List lists webhooks for a tenant
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	webhooks, err := h.repo.List(ctx, tenantUUID, false)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list webhooks", api.ErrCodeInternalError)
		return
	}

	response := make([]*WebhookResponse, len(webhooks))
	for i, wh := range webhooks {
		response[i] = webhookToResponse(wh)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"webhooks": response,
	})
}

// CreateRequest represents the request to create a webhook
type CreateRequest struct {
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Events         []string          `json:"events"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
	MaxRetries     int               `json:"max_retries,omitempty"`
	Headers        map[string]string `json:"headers,omitempty"`
}

// Create creates a new webhook
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	// Validate
	if req.Name == "" {
		api.JSONError(w, http.StatusBadRequest, "name is required", api.ErrCodeValidation)
		return
	}
	if req.URL == "" {
		api.JSONError(w, http.StatusBadRequest, "url is required", api.ErrCodeValidation)
		return
	}
	if len(req.Events) == 0 {
		api.JSONError(w, http.StatusBadRequest, "at least one event is required", api.ErrCodeValidation)
		return
	}

	// Validate events
	validEvents := map[string]bool{
		EventNewDocument:     true,
		EventDeadlineWarning: true,
		EventFBChange:        true,
		EventSyncComplete:    true,
		EventDocumentRead:    true,
	}
	for _, event := range req.Events {
		if !validEvents[event] {
			api.JSONError(w, http.StatusBadRequest, "invalid event: "+event, api.ErrCodeValidation)
			return
		}
	}

	// Generate secret
	secret, err := generateSecret(32)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to generate secret", api.ErrCodeInternalError)
		return
	}

	webhook := &Webhook{
		TenantID:       tenantUUID,
		Name:           req.Name,
		URL:            req.URL,
		Secret:         secret,
		Events:         req.Events,
		Enabled:        true,
		TimeoutSeconds: req.TimeoutSeconds,
		MaxRetries:     req.MaxRetries,
		Headers:        req.Headers,
	}

	if err := h.repo.Create(ctx, webhook); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to create webhook", api.ErrCodeInternalError)
		return
	}

	// Return response with secret (only shown on creation)
	response := webhookToResponse(webhook)
	api.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"webhook": response,
		"secret":  secret, // Only returned on creation
	})
}

// Get retrieves a webhook by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid webhook ID", api.ErrCodeBadRequest)
		return
	}

	webhook, err := h.repo.GetByID(ctx, webhookID)
	if err != nil {
		if err == ErrWebhookNotFound {
			api.JSONError(w, http.StatusNotFound, "webhook not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get webhook", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, webhookToResponse(webhook))
}

// Update updates a webhook
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	webhookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid webhook ID", api.ErrCodeBadRequest)
		return
	}

	var req struct {
		Name           *string           `json:"name,omitempty"`
		URL            *string           `json:"url,omitempty"`
		Events         []string          `json:"events,omitempty"`
		Enabled        *bool             `json:"enabled,omitempty"`
		TimeoutSeconds *int              `json:"timeout_seconds,omitempty"`
		MaxRetries     *int              `json:"max_retries,omitempty"`
		Headers        map[string]string `json:"headers,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	// Get existing webhook
	webhook, err := h.repo.GetByID(ctx, webhookID)
	if err != nil {
		if err == ErrWebhookNotFound {
			api.JSONError(w, http.StatusNotFound, "webhook not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get webhook", api.ErrCodeInternalError)
		return
	}

	// Verify tenant
	if webhook.TenantID != tenantUUID {
		api.JSONError(w, http.StatusNotFound, "webhook not found", api.ErrCodeNotFound)
		return
	}

	// Update fields
	if req.Name != nil {
		webhook.Name = *req.Name
	}
	if req.URL != nil {
		webhook.URL = *req.URL
	}
	if len(req.Events) > 0 {
		webhook.Events = req.Events
	}
	if req.Enabled != nil {
		webhook.Enabled = *req.Enabled
	}
	if req.TimeoutSeconds != nil {
		webhook.TimeoutSeconds = *req.TimeoutSeconds
	}
	if req.MaxRetries != nil {
		webhook.MaxRetries = *req.MaxRetries
	}
	if req.Headers != nil {
		webhook.Headers = req.Headers
	}

	if err := h.repo.Update(ctx, webhook); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to update webhook", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, webhookToResponse(webhook))
}

// Delete deletes a webhook
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	webhookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid webhook ID", api.ErrCodeBadRequest)
		return
	}

	if err := h.repo.Delete(ctx, webhookID, tenantUUID); err != nil {
		if err == ErrWebhookNotFound {
			api.JSONError(w, http.StatusNotFound, "webhook not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to delete webhook", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RotateSecret generates a new secret for a webhook
func (h *Handler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid webhook ID", api.ErrCodeBadRequest)
		return
	}

	// Generate new secret
	newSecret, err := generateSecret(32)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to generate secret", api.ErrCodeInternalError)
		return
	}

	if err := h.repo.UpdateSecret(ctx, webhookID, newSecret); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to update secret", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"secret": newSecret,
	})
}

// ListDeliveries lists deliveries for a webhook
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	webhookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid webhook ID", api.ErrCodeBadRequest)
		return
	}

	limit := 50
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	deliveries, total, err := h.repo.ListDeliveries(ctx, webhookID, limit, offset)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list deliveries", api.ErrCodeInternalError)
		return
	}

	response := make([]map[string]interface{}, len(deliveries))
	for i, d := range deliveries {
		response[i] = map[string]interface{}{
			"id":              d.ID.String(),
			"event_type":      d.EventType,
			"status":          d.Status,
			"response_status": d.ResponseStatus,
			"attempt_count":   d.AttemptCount,
			"last_error":      d.LastError,
			"created_at":      d.CreatedAt.Format(time.RFC3339),
		}
		if d.DeliveredAt != nil {
			response[i]["delivered_at"] = d.DeliveredAt.Format(time.RFC3339)
		}
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"deliveries": response,
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

// TestWebhook sends a test event to a webhook
func (h *Handler) TestWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := api.GetTenantID(ctx)

	if tenantID == "" {
		api.JSONError(w, http.StatusUnauthorized, "unauthorized", api.ErrCodeUnauthorized)
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid tenant ID", api.ErrCodeBadRequest)
		return
	}

	webhookID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid webhook ID", api.ErrCodeBadRequest)
		return
	}

	// Verify webhook exists and belongs to tenant
	webhook, err := h.repo.GetByID(ctx, webhookID)
	if err != nil {
		if err == ErrWebhookNotFound {
			api.JSONError(w, http.StatusNotFound, "webhook not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get webhook", api.ErrCodeInternalError)
		return
	}

	if webhook.TenantID != tenantUUID {
		api.JSONError(w, http.StatusNotFound, "webhook not found", api.ErrCodeNotFound)
		return
	}

	// Trigger test event
	testData := map[string]interface{}{
		"message": "This is a test webhook event",
		"test":    true,
	}

	if err := h.service.TriggerEvent(ctx, tenantUUID, "test", testData); err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to trigger test event", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{
		"status":  "test_sent",
		"message": "Test webhook event has been queued",
	})
}

// Helper functions

func webhookToResponse(wh *Webhook) *WebhookResponse {
	return &WebhookResponse{
		ID:             wh.ID.String(),
		Name:           wh.Name,
		URL:            wh.URL,
		Events:         wh.Events,
		Enabled:        wh.Enabled,
		TimeoutSeconds: wh.TimeoutSeconds,
		MaxRetries:     wh.MaxRetries,
		Headers:        wh.Headers,
		CreatedAt:      wh.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      wh.UpdatedAt.Format(time.RFC3339),
	}
}

func generateSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
