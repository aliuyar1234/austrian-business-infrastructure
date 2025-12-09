package share

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/client"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles share-related HTTP requests
type Handler struct {
	service       *Service
	clientService *client.Service
}

// NewHandler creates a new share handler
func NewHandler(service *Service, clientService *client.Service) *Handler {
	return &Handler{
		service:       service,
		clientService: clientService,
	}
}

// StaffRoutes returns the routes for staff managing shares
func (h *Handler) StaffRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Share)
	r.Get("/", h.ListByDocument)
	r.Delete("/", h.Unshare)

	return r
}

// PortalRoutes returns the routes for portal clients
func (h *Handler) PortalRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListForClient)
	r.Get("/{id}", h.GetDocument)

	return r
}

// Share creates a new document share
func (h *Handler) Share(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userIDStr := api.GetUserID(ctx)
	userID, _ := uuid.Parse(userIDStr)

	var req struct {
		DocumentID  uuid.UUID  `json:"document_id"`
		ClientID    uuid.UUID  `json:"client_id"`
		CanDownload *bool      `json:"can_download,omitempty"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.DocumentID == uuid.Nil || req.ClientID == uuid.Nil {
		http.Error(w, "document_id and client_id are required", http.StatusBadRequest)
		return
	}

	canDownload := true
	if req.CanDownload != nil {
		canDownload = *req.CanDownload
	}

	share, err := h.service.Share(ctx, &ShareRequest{
		DocumentID:  req.DocumentID,
		ClientID:    req.ClientID,
		SharedBy:    userID,
		CanDownload: canDownload,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		if errors.Is(err, ErrShareExists) {
			http.Error(w, "document already shared with this client", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create share", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(share)
}

// Unshare removes a document share
func (h *Handler) Unshare(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	documentIDStr := r.URL.Query().Get("document_id")
	clientIDStr := r.URL.Query().Get("client_id")

	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		http.Error(w, "valid document_id required", http.StatusBadRequest)
		return
	}

	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		http.Error(w, "valid client_id required", http.StatusBadRequest)
		return
	}

	if err := h.service.Unshare(ctx, documentID, clientID); err != nil {
		if errors.Is(err, ErrShareNotFound) {
			http.Error(w, "share not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to remove share", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListByDocument returns all shares for a document
func (h *Handler) ListByDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	documentIDStr := r.URL.Query().Get("document_id")
	documentID, err := uuid.Parse(documentIDStr)
	if err != nil {
		http.Error(w, "valid document_id required", http.StatusBadRequest)
		return
	}

	shares, err := h.service.ListForDocument(ctx, documentID)
	if err != nil {
		http.Error(w, "failed to list shares", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"shares": shares,
	})
}

// ============== Portal Endpoints ==============

// ListForClient returns documents shared with the current client
func (h *Handler) ListForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	shares, total, err := h.service.ListForClient(ctx, claims.ClientID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list documents", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"documents": shares,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}

// GetDocument returns a shared document
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	shareID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid share ID", http.StatusBadRequest)
		return
	}

	share, err := h.service.GetByID(ctx, shareID)
	if err != nil {
		if errors.Is(err, ErrShareNotFound) {
			http.Error(w, "document not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get document", http.StatusInternalServerError)
		return
	}

	// Verify client access
	if share.ClientID != claims.ClientID {
		http.Error(w, "document not found", http.StatusNotFound)
		return
	}

	// Check expiry
	if share.ExpiresAt != nil && share.ExpiresAt.Before(time.Now()) {
		http.Error(w, "document access has expired", http.StatusGone)
		return
	}

	// Record view
	_ = h.service.RecordView(ctx, shareID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(share)
}
