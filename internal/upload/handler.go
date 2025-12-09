package upload

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/client"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles upload-related HTTP requests
type Handler struct {
	service       *Service
	clientService *client.Service
	maxUploadSize int64
}

// NewHandler creates a new upload handler
func NewHandler(service *Service, clientService *client.Service, maxUploadSize int64) *Handler {
	return &Handler{
		service:       service,
		clientService: clientService,
		maxUploadSize: maxUploadSize,
	}
}

// StaffRoutes returns the routes for staff managing uploads
func (h *Handler) StaffRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.ListForTenant)
	r.Get("/{id}", h.GetByID)
	r.Post("/{id}/process", h.MarkProcessed)
	r.Delete("/{id}", h.Delete)

	return r
}

// PortalRoutes returns the routes for portal clients
func (h *Handler) PortalRoutes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Upload)
	r.Get("/", h.ListForClient)
	r.Get("/{id}", h.GetByIDForClient)
	r.Delete("/{id}", h.DeleteForClient)

	return r
}

// Upload handles file upload from portal client
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get client from context
	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)

	// Parse multipart form
	err := r.ParseMultipartForm(h.maxUploadSize)
	if err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Get file
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get account ID
	accountIDStr := r.FormValue("account_id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		http.Error(w, "valid account_id required", http.StatusBadRequest)
		return
	}

	// Verify client has access to account
	hasAccess, err := h.clientService.Repository().HasAccountAccess(ctx, claims.ClientID, accountID)
	if err != nil || !hasAccess {
		http.Error(w, "no access to this account", http.StatusForbidden)
		return
	}

	// Get optional fields
	var category *Category
	if cat := r.FormValue("category"); cat != "" {
		if !IsValidCategory(cat) {
			http.Error(w, "invalid category", http.StatusBadRequest)
			return
		}
		c := Category(cat)
		category = &c
	}

	var note *string
	if n := r.FormValue("note"); n != "" {
		note = &n
	}

	// Detect content type
	buffer := make([]byte, 512)
	n, _ := file.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])

	// Reset file position
	file.Seek(0, 0)

	// Create upload
	req := &UploadRequest{
		ClientID:  claims.ClientID,
		AccountID: accountID,
		Filename:  header.Filename,
		FileSize:  header.Size,
		MimeType:  mimeType,
		Category:  category,
		Note:      note,
		Reader:    file,
	}

	upload, err := h.service.Upload(ctx, req)
	if err != nil {
		if errors.Is(err, ErrFileTooLarge) {
			http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
			return
		}
		if errors.Is(err, ErrInvalidFileType) {
			http.Error(w, "file type not allowed", http.StatusBadRequest)
			return
		}
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(upload)
}

// ListForClient returns uploads for the current client
func (h *Handler) ListForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var status *Status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if !IsValidStatus(statusStr) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
		s := Status(statusStr)
		status = &s
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	uploads, total, err := h.service.ListByClient(ctx, claims.ClientID, status, limit, offset)
	if err != nil {
		http.Error(w, "failed to list uploads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uploads": uploads,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetByIDForClient returns a specific upload for the current client
func (h *Handler) GetByIDForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	uploadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid upload ID", http.StatusBadRequest)
		return
	}

	upload, err := h.service.GetByID(ctx, uploadID)
	if err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get upload", http.StatusInternalServerError)
		return
	}

	// Verify ownership
	if upload.ClientID != claims.ClientID {
		http.Error(w, "upload not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(upload)
}

// DeleteForClient deletes an upload owned by the current client
func (h *Handler) DeleteForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	uploadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid upload ID", http.StatusBadRequest)
		return
	}

	upload, err := h.service.GetByID(ctx, uploadID)
	if err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get upload", http.StatusInternalServerError)
		return
	}

	// Verify ownership
	if upload.ClientID != claims.ClientID {
		http.Error(w, "upload not found", http.StatusNotFound)
		return
	}

	// Only allow deletion of new uploads
	if upload.Status != StatusNew {
		http.Error(w, "can only delete unprocessed uploads", http.StatusForbidden)
		return
	}

	if err := h.service.Delete(ctx, uploadID); err != nil {
		http.Error(w, "failed to delete upload", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============== Staff Endpoints ==============

// ListForTenant returns all uploads for the tenant
func (h *Handler) ListForTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var status *Status
	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if !IsValidStatus(statusStr) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}
		s := Status(statusStr)
		status = &s
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	uploads, total, err := h.service.ListByTenant(ctx, tenantID, status, limit, offset)
	if err != nil {
		http.Error(w, "failed to list uploads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"uploads": uploads,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetByID returns a specific upload
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	uploadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid upload ID", http.StatusBadRequest)
		return
	}

	// Get file content for download
	reader, upload, err := h.service.GetFile(ctx, uploadID)
	if err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get upload", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	// Set headers for download
	mimeType := "application/octet-stream"
	if upload.MimeType != nil {
		mimeType = *upload.MimeType
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+upload.Filename+`"`)
	w.Header().Set("Content-Length", strconv.FormatInt(upload.FileSize, 10))

	io.Copy(w, reader)
}

// MarkProcessed marks an upload as processed
func (h *Handler) MarkProcessed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user ID from context (staff user)
	userIDStr := r.Context().Value("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	uploadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid upload ID", http.StatusBadRequest)
		return
	}

	if err := h.service.MarkProcessed(ctx, uploadID, userID); err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to mark processed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}

// Delete deletes an upload (staff only)
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	uploadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid upload ID", http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(ctx, uploadID); err != nil {
		if errors.Is(err, ErrUploadNotFound) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to delete upload", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
