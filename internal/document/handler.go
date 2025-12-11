package document

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles document HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new document handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// getTenantID extracts and parses tenant ID from request context
func getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := api.GetTenantID(r.Context())
	if tenantIDStr == "" {
		return uuid.Nil, ErrDocumentNotFound // Return not found to avoid info leak
	}
	return uuid.Parse(tenantIDStr)
}

// RegisterRoutes registers document routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/documents", h.List)
	mux.HandleFunc("GET /api/v1/documents/{id}", h.Get)
	mux.HandleFunc("GET /api/v1/documents/{id}/content", h.GetContent)
	mux.HandleFunc("GET /api/v1/documents/{id}/download-url", h.GetDownloadURL)
	mux.HandleFunc("PUT /api/v1/documents/{id}/status", h.UpdateStatus)
	mux.HandleFunc("POST /api/v1/documents/{id}/read", h.MarkAsRead)
	mux.HandleFunc("POST /api/v1/documents/{id}/archive", h.Archive)
	mux.HandleFunc("POST /api/v1/documents/archive", h.BulkArchive)
	mux.HandleFunc("DELETE /api/v1/documents/{id}", h.Delete)
	mux.HandleFunc("GET /api/v1/documents/stats", h.GetStats)
	mux.HandleFunc("GET /api/v1/documents/expired", h.GetExpired)
}

// ListResponse represents the response for listing documents
type ListResponse struct {
	Documents []*DocumentResponse `json:"documents"`
	Total     int                 `json:"total"`
	Limit     int                 `json:"limit"`
	Offset    int                 `json:"offset"`
	HasMore   bool                `json:"has_more"`
	Facets    *FacetsResponse     `json:"facets,omitempty"`
}

// FacetsResponse contains filter facets for UI
type FacetsResponse struct {
	ByType   map[string]int `json:"by_type"`
	ByStatus map[string]int `json:"by_status"`
}

// DocumentResponse represents a document in API responses
type DocumentResponse struct {
	ID          uuid.UUID              `json:"id"`
	AccountID   uuid.UUID              `json:"account_id"`
	AccountName string                 `json:"account_name,omitempty"`
	AccountType string                 `json:"account_type,omitempty"`
	ExternalID  string                 `json:"external_id,omitempty"`
	Type        string                 `json:"type"`
	Title       string                 `json:"title"`
	Sender      string                 `json:"sender"`
	ReceivedAt  time.Time              `json:"received_at"`
	FileSize    int                    `json:"file_size"`
	MimeType    string                 `json:"mime_type"`
	Status      string                 `json:"status"`
	Priority    int                    `json:"priority"`
	ArchivedAt  *time.Time             `json:"archived_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// toResponse converts a Document to DocumentResponse
func toResponse(doc *Document) *DocumentResponse {
	return &DocumentResponse{
		ID:          doc.ID,
		AccountID:   doc.AccountID,
		AccountName: doc.AccountName,
		AccountType: doc.AccountType,
		ExternalID:  doc.ExternalID,
		Type:        doc.Type,
		Title:       doc.Title,
		Sender:      doc.Sender,
		ReceivedAt:  doc.ReceivedAt,
		FileSize:    doc.FileSize,
		MimeType:    doc.MimeType,
		Status:      doc.Status,
		Priority:    TypePriority(doc.Type),
		ArchivedAt:  doc.ArchivedAt,
		Metadata:    doc.Metadata,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

// List returns a paginated list of documents
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

	// Parse query parameters
	filter := &DocumentFilter{
		TenantID: tenantUUID,
		Limit:    50,
		Offset:   0,
		SortBy:   "received_at",
		SortDesc: true,
	}

	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offset := r.URL.Query().Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	if accountID := r.URL.Query().Get("account_id"); accountID != "" {
		if id, err := uuid.Parse(accountID); err == nil {
			filter.AccountID = &id
		}
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = status
	}

	if docType := r.URL.Query().Get("type"); docType != "" {
		filter.Type = docType
	}

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = search
	}

	if archived := r.URL.Query().Get("archived"); archived == "true" {
		filter.Archived = true
	}

	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		filter.SortBy = sortBy
	}

	if sortDir := r.URL.Query().Get("sort_dir"); sortDir == "asc" {
		filter.SortDesc = false
	}

	if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			filter.DateFrom = &t
		}
	}

	if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			filter.DateTo = &t
		}
	}

	// Get documents
	documents, total, err := h.service.List(ctx, filter)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to list documents", api.ErrCodeInternalError)
		return
	}

	// Convert to response format
	responses := make([]*DocumentResponse, len(documents))
	for i, doc := range documents {
		responses[i] = toResponse(doc)
	}

	response := &ListResponse{
		Documents: responses,
		Total:     total,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
		HasMore:   filter.Offset+len(documents) < total,
	}

	// Include facets if requested
	if r.URL.Query().Get("include_facets") == "true" {
		stats, err := h.service.GetStats(ctx, tenantUUID)
		if err == nil {
			response.Facets = &FacetsResponse{
				ByType:   stats.ByType,
				ByStatus: stats.ByStatus,
			}
		}
	}

	api.JSONResponse(w, http.StatusOK, response)
}

// Get returns a single document by ID
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	doc, err := h.service.GetByID(ctx, tenantID, id)
	if err != nil {
		if err == ErrDocumentNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get document", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, toResponse(doc))
}

// GetContent returns the document content for download
func (h *Handler) GetContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	// Mark as read when content is accessed
	h.service.MarkAsRead(ctx, tenantID, id)

	// Get content
	content, info, err := h.service.GetContent(ctx, tenantID, id)
	if err != nil {
		if err == ErrDocumentNotFound || err == ErrStorageNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to get document content", api.ErrCodeInternalError)
		return
	}
	defer content.Close()

	// Set headers
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(info.Size, 10))

	// Stream content with buffering for large files
	bufSize := 32 * 1024 // 32KB buffer
	if info.Size > 10*1024*1024 { // >10MB
		bufSize = 256 * 1024 // 256KB buffer for large files
	}
	buf := make([]byte, bufSize)
	io.CopyBuffer(w, content, buf)
}

// DownloadURLResponse represents the response for signed download URL
type DownloadURLResponse struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in"` // seconds
}

// GetDownloadURL returns a time-limited signed URL for document download
func (h *Handler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	// Default expiry 15 minutes
	expiry := 15 * time.Minute
	if expiryStr := r.URL.Query().Get("expiry"); expiryStr != "" {
		if minutes, err := strconv.Atoi(expiryStr); err == nil && minutes > 0 && minutes <= 60 {
			expiry = time.Duration(minutes) * time.Minute
		}
	}

	url, err := h.service.GetSignedURL(ctx, tenantID, id, expiry)
	if err != nil {
		if err == ErrDocumentNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		if err == ErrSignedURLNotSupported {
			api.JSONError(w, http.StatusBadRequest, "signed URLs not supported for this storage backend", api.ErrCodeBadRequest)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to generate download URL", api.ErrCodeInternalError)
		return
	}

	// Mark as read when URL is generated
	h.service.MarkAsRead(ctx, tenantID, id)

	api.JSONResponse(w, http.StatusOK, &DownloadURLResponse{
		URL:       url,
		ExpiresIn: int(expiry.Seconds()),
	})
}

// UpdateStatusRequest represents the request to update document status
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus updates the status of a document
func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	if req.Status != StatusNew && req.Status != StatusRead && req.Status != StatusArchived {
		api.JSONError(w, http.StatusBadRequest, "invalid status", api.ErrCodeValidation)
		return
	}

	if err := h.service.UpdateStatus(ctx, tenantID, id, req.Status); err != nil {
		if err == ErrDocumentNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to update document status", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// MarkAsRead marks a document as read (convenience endpoint for frontend)
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	if err := h.service.MarkAsRead(ctx, tenantID, id); err != nil {
		if err == ErrDocumentNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to mark document as read", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "read"})
}

// Archive archives a single document
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	if err := h.service.Archive(ctx, tenantID, id); err != nil {
		if err == ErrDocumentNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to archive document", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "archived"})
}

// BulkArchiveRequest represents the request to archive multiple documents
type BulkArchiveRequest struct {
	IDs []uuid.UUID `json:"ids"`
}

// BulkArchive archives multiple documents
func (h *Handler) BulkArchive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusForbidden, "access denied", api.ErrCodeForbidden)
		return
	}

	var req BulkArchiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid request body", api.ErrCodeBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		api.JSONError(w, http.StatusBadRequest, "no document IDs provided", api.ErrCodeValidation)
		return
	}

	if len(req.IDs) > 100 {
		api.JSONError(w, http.StatusBadRequest, "maximum 100 documents per request", api.ErrCodeValidation)
		return
	}

	count, err := h.service.BulkArchive(ctx, tenantID, req.IDs)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to archive documents", api.ErrCodeInternalError)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "archived",
		"archived": count,
	})
}

// Delete permanently removes a document
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := getTenantID(r)
	if err != nil {
		api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
		return
	}

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		api.JSONError(w, http.StatusBadRequest, "invalid document ID", api.ErrCodeBadRequest)
		return
	}

	if err := h.service.Delete(ctx, tenantID, id); err != nil {
		if err == ErrDocumentNotFound {
			api.JSONError(w, http.StatusNotFound, "document not found", api.ErrCodeNotFound)
			return
		}
		api.JSONError(w, http.StatusInternalServerError, "failed to delete document", api.ErrCodeInternalError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// StatsResponse represents the response for document statistics
type StatsResponse struct {
	Total           int            `json:"total"`
	New             int            `json:"new"`
	Read            int            `json:"read"`
	ByType          map[string]int `json:"by_type"`
	ByAccount       map[string]int `json:"by_account"`
	UnreadByAccount map[string]int `json:"unread_by_account"`
}

// GetStats returns document statistics
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
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

	stats, err := h.service.GetStats(ctx, tenantUUID)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to get statistics", api.ErrCodeInternalError)
		return
	}

	// Get unread counts per account for badge display
	unreadCounts, err := h.service.GetUnreadCounts(ctx, tenantUUID)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to get unread counts", api.ErrCodeInternalError)
		return
	}

	// Convert account UUIDs to strings for JSON
	byAccount := make(map[string]int)
	for k, v := range stats.ByAccount {
		byAccount[k.String()] = v
	}

	unreadByAccount := make(map[string]int)
	for k, v := range unreadCounts {
		unreadByAccount[k.String()] = v
	}

	response := &StatsResponse{
		Total:           stats.TotalCount,
		New:             stats.NewCount,
		Read:            stats.ReadCount,
		ByType:          stats.ByType,
		ByAccount:       byAccount,
		UnreadByAccount: unreadByAccount,
	}

	api.JSONResponse(w, http.StatusOK, response)
}

// GetExpired returns documents past their retention date with pagination
func (h *Handler) GetExpired(w http.ResponseWriter, r *http.Request) {
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

	// Parse pagination params with defaults
	limit := 50
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	documents, total, err := h.service.GetExpired(ctx, tenantUUID, limit, offset)
	if err != nil {
		api.JSONError(w, http.StatusInternalServerError, "failed to get expired documents", api.ErrCodeInternalError)
		return
	}

	responses := make([]*DocumentResponse, len(documents))
	for i, doc := range documents {
		responses[i] = toResponse(doc)
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"documents": responses,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
		"has_more":  offset+len(documents) < total,
	})
}
