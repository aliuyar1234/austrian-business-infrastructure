package websocket

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/auth"
)

// Handler handles WebSocket HTTP requests
type Handler struct {
	hub            *Hub
	upgrader       websocket.Upgrader
	logger         *slog.Logger
	jwtManager     *auth.JWTManager
	allowedOrigins map[string]bool
	devMode        bool
}

// HandlerConfig configures the WebSocket handler
type HandlerConfig struct {
	AllowedOrigins []string // Empty in production = reject all (secure default)
	JWTManager     *auth.JWTManager
	DevMode        bool // Only allow all origins if explicitly in dev mode
}

// NewHandler creates a new WebSocket handler
func NewHandler(hub *Hub, logger *slog.Logger, config *HandlerConfig) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	h := &Handler{
		hub:            hub,
		logger:         logger,
		allowedOrigins: make(map[string]bool),
	}

	if config != nil {
		h.jwtManager = config.JWTManager
		h.devMode = config.DevMode
		for _, origin := range config.AllowedOrigins {
			h.allowedOrigins[origin] = true
		}
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	return h
}

// checkOrigin validates the request origin against allowed origins
// Secure default: reject all origins if not configured (unless explicitly in dev mode)
func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Same-origin requests don't have Origin header
	}

	// If no origins configured, only allow in dev mode (secure default)
	if len(h.allowedOrigins) == 0 {
		if h.devMode {
			h.logger.Warn("WebSocket accepting all origins - development mode only")
			return true
		}
		h.logger.Error("WebSocket rejecting connection - AllowedOrigins not configured", "origin", origin)
		return false
	}

	// Check against allowlist
	if h.allowedOrigins[origin] {
		return true
	}

	// Check if origin matches any allowed pattern (e.g., *.example.com)
	for allowed := range h.allowedOrigins {
		if strings.HasPrefix(allowed, "*.") {
			suffix := allowed[1:] // ".example.com"
			if strings.HasSuffix(origin, suffix) || origin == "https://"+allowed[2:] || origin == "http://"+allowed[2:] {
				return true
			}
		}
	}

	h.logger.Warn("WebSocket origin rejected", "origin", origin)
	return false
}

// RegisterRoutes registers WebSocket routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/ws", h.HandleWebSocket)
}

// HandleWebSocket upgrades HTTP connection to WebSocket
// Authentication is done via first message after connection (not query params)
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if already authenticated via middleware (e.g., Authorization header)
	tenantID := api.GetTenantID(ctx)
	userID := api.GetUserID(ctx)
	preAuthenticated := tenantID != "" && userID != ""

	// Upgrade to WebSocket first (auth happens after via first message)
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	if preAuthenticated {
		// Already authenticated - proceed directly
		h.completeWebSocketSetup(conn, tenantID, userID)
		return
	}

	// Not pre-authenticated - require first-message auth
	h.handleFirstMessageAuth(conn)
}

// handleFirstMessageAuth waits for authentication message before completing setup
func (h *Handler) handleFirstMessageAuth(conn *websocket.Conn) {
	// Set deadline for auth message (10 seconds)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Read first message (must be auth)
	_, message, err := conn.ReadMessage()
	if err != nil {
		h.logger.Debug("websocket auth failed - no message", "error", err)
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"code":    "auth_timeout",
			"message": "Authentication required within 10 seconds",
		})
		conn.Close()
		return
	}

	// Parse auth message
	var authMsg struct {
		Type  string `json:"type"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(message, &authMsg); err != nil || authMsg.Type != "auth" {
		h.logger.Debug("websocket auth failed - invalid message format")
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"code":    "invalid_auth",
			"message": "First message must be {\"type\":\"auth\",\"token\":\"...\"}",
		})
		conn.Close()
		return
	}

	// Validate token
	tenantID, userID := h.validateToken(authMsg.Token)
	if tenantID == "" {
		h.logger.Debug("websocket auth failed - invalid token")
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"code":    "invalid_token",
			"message": "Invalid or expired token",
		})
		conn.Close()
		return
	}

	// Clear read deadline for normal operation
	conn.SetReadDeadline(time.Time{})

	// Complete setup
	h.completeWebSocketSetup(conn, tenantID, userID)
}

// completeWebSocketSetup finishes WebSocket setup after authentication
func (h *Handler) completeWebSocketSetup(conn *websocket.Conn, tenantID, userID string) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		conn.WriteJSON(map[string]interface{}{
			"type":    "error",
			"code":    "invalid_tenant",
			"message": "Invalid tenant ID",
		})
		conn.Close()
		return
	}

	var userUUID uuid.UUID
	if userID != "" {
		userUUID, _ = uuid.Parse(userID)
	}

	// Create client
	client := NewClient(h.hub, conn, tenantUUID, userUUID, h.logger)

	// Register client
	h.hub.Register(client)

	// Send connected event (confirms auth success)
	clientID := uuid.New().String()
	client.send <- ConnectedEvent(clientID, 5) // 5 second reconnect delay

	// Start read/write pumps
	go client.WritePump()
	go client.ReadPump()
}

// validateToken validates a JWT token and returns tenant and user IDs
// SECURITY: Only accepts access tokens, not refresh tokens
func (h *Handler) validateToken(token string) (tenantID, userID string) {
	if h.jwtManager == nil {
		h.logger.Error("WebSocket JWT manager not configured")
		return "", ""
	}

	// Only accept access tokens for WebSocket authentication
	// Refresh tokens are long-lived and should only be used to obtain new access tokens
	claims, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		h.logger.Debug("WebSocket token validation failed", "error", err)
		return "", ""
	}

	return claims.TenantID, claims.UserID
}
