package message

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/client"
	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles message-related HTTP requests
type Handler struct {
	service *Service
	hub     *Hub
}

// NewHandler creates a new message handler
func NewHandler(service *Service, hub *Hub) *Handler {
	return &Handler{
		service: service,
		hub:     hub,
	}
}

// StaffRoutes returns the routes for staff managing messages
func (h *Handler) StaffRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/threads", h.ListThreads)
	r.Post("/threads", h.StartThread)
	r.Get("/threads/{id}", h.GetThread)
	r.Get("/threads/{id}/messages", h.ListMessages)
	r.Post("/threads/{id}/messages", h.SendMessage)
	r.Post("/threads/{id}/read", h.MarkAsRead)
	r.Get("/ws", h.WebSocket)

	return r
}

// PortalRoutes returns the routes for portal clients
func (h *Handler) PortalRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/threads", h.ListThreadsForClient)
	r.Post("/threads", h.StartThreadForClient)
	r.Get("/threads/{id}", h.GetThreadForClient)
	r.Get("/threads/{id}/messages", h.ListMessagesForClient)
	r.Post("/threads/{id}/messages", h.SendMessageForClient)
	r.Post("/threads/{id}/read", h.MarkAsReadForClient)
	r.Get("/unread", h.CountUnread)
	r.Get("/ws", h.PortalWebSocket)

	return r
}

// ============== Staff Endpoints ==============

// ListThreads returns message threads for the tenant
func (h *Handler) ListThreads(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
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

	threads, total, err := h.service.ListThreadsForTenant(ctx, tenantID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list threads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"threads": threads,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// StartThread creates a new message thread
func (h *Handler) StartThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userIDStr := api.GetUserID(ctx)
	userID, _ := uuid.Parse(userIDStr)

	var req struct {
		ClientID uuid.UUID `json:"client_id"`
		Subject  string    `json:"subject"`
		Content  string    `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientID == uuid.Nil || req.Subject == "" || req.Content == "" {
		http.Error(w, "client_id, subject, and content are required", http.StatusBadRequest)
		return
	}

	thread, msg, err := h.service.StartThread(ctx, &StartThreadRequest{
		TenantID:   tenantID,
		ClientID:   req.ClientID,
		Subject:    req.Subject,
		Content:    req.Content,
		SenderType: "staff",
		SenderID:   userID,
	})
	if err != nil {
		http.Error(w, "failed to start thread", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"thread":  thread,
		"message": msg,
	})
}

// GetThread returns a thread by ID
func (h *Handler) GetThread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	// Verify tenant access
	if thread.TenantID != tenantID {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(thread)
}

// ListMessages returns messages for a thread
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	// Verify thread access
	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	if thread.TenantID != tenantID {
		http.Error(w, "thread not found", http.StatusNotFound)
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

	messages, total, err := h.service.ListMessages(ctx, threadID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// SendMessage sends a message to a thread
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userIDStr := api.GetUserID(ctx)
	userID, _ := uuid.Parse(userIDStr)

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	// Verify thread access
	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	if thread.TenantID != tenantID {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	msg, err := h.service.SendMessage(ctx, &SendMessageRequest{
		ThreadID:   threadID,
		SenderType: "staff",
		SenderID:   userID,
		Content:    req.Content,
	})
	if err != nil {
		http.Error(w, "failed to send message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

// MarkAsRead marks messages as read
func (h *Handler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	// Verify thread access
	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	if thread.TenantID != tenantID {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	if err := h.service.MarkAsRead(ctx, threadID, "staff"); err != nil {
		http.Error(w, "failed to mark as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// WebSocket handles WebSocket connections for staff
func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userIDStr := api.GetUserID(ctx)
	userID, _ := uuid.Parse(userIDStr)

	h.hub.ServeWS(w, r, userID, tenantID)
}

// ============== Portal Endpoints ==============

// ListThreadsForClient returns threads for the current client
func (h *Handler) ListThreadsForClient(w http.ResponseWriter, r *http.Request) {
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

	threads, total, err := h.service.ListThreadsForClient(ctx, claims.ClientID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list threads", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"threads": threads,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// StartThreadForClient creates a new thread for the client
func (h *Handler) StartThreadForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Subject string `json:"subject"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Subject == "" || req.Content == "" {
		http.Error(w, "subject and content are required", http.StatusBadRequest)
		return
	}

	thread, msg, err := h.service.StartThread(ctx, &StartThreadRequest{
		TenantID:   claims.TenantID,
		ClientID:   claims.ClientID,
		Subject:    req.Subject,
		Content:    req.Content,
		SenderType: "client",
		SenderID:   claims.ClientID,
	})
	if err != nil {
		http.Error(w, "failed to start thread", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"thread":  thread,
		"message": msg,
	})
}

// GetThreadForClient returns a thread by ID for the client
func (h *Handler) GetThreadForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	// Verify client access
	if thread.ClientID != claims.ClientID {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(thread)
}

// ListMessagesForClient returns messages for a thread
func (h *Handler) ListMessagesForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	// Verify thread access
	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	if thread.ClientID != claims.ClientID {
		http.Error(w, "thread not found", http.StatusNotFound)
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

	messages, total, err := h.service.ListMessages(ctx, threadID, limit, offset)
	if err != nil {
		http.Error(w, "failed to list messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": messages,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// SendMessageForClient sends a message from the client
func (h *Handler) SendMessageForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	// Verify thread access
	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	if thread.ClientID != claims.ClientID {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	msg, err := h.service.SendMessage(ctx, &SendMessageRequest{
		ThreadID:   threadID,
		SenderType: "client",
		SenderID:   claims.ClientID,
		Content:    req.Content,
	})
	if err != nil {
		http.Error(w, "failed to send message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(msg)
}

// MarkAsReadForClient marks messages as read by the client
func (h *Handler) MarkAsReadForClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	threadID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid thread ID", http.StatusBadRequest)
		return
	}

	// Verify thread access
	thread, err := h.service.GetThread(ctx, threadID)
	if err != nil {
		if errors.Is(err, ErrThreadNotFound) {
			http.Error(w, "thread not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get thread", http.StatusInternalServerError)
		return
	}

	if thread.ClientID != claims.ClientID {
		http.Error(w, "thread not found", http.StatusNotFound)
		return
	}

	if err := h.service.MarkAsRead(ctx, threadID, "client"); err != nil {
		http.Error(w, "failed to mark as read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CountUnread returns the count of unread messages for the client
func (h *Handler) CountUnread(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	count, err := h.service.CountUnreadForClient(ctx, claims.ClientID)
	if err != nil {
		http.Error(w, "failed to count unread", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"unread_count": count,
	})
}

// PortalWebSocket handles WebSocket connections for portal clients
func (h *Handler) PortalWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, ok := client.ClientFromContext(ctx)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	h.hub.ServePortalWS(w, r, claims.ClientID, claims.TenantID)
}
