package websocket

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	// Registered clients by tenant ID
	clients map[uuid.UUID]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to a tenant
	broadcast chan *BroadcastMessage

	// Mutex for clients map
	mu sync.RWMutex

	// Logger
	logger *slog.Logger

	// Done channel for shutdown
	done chan struct{}
}

// BroadcastMessage holds a message to broadcast to a tenant
type BroadcastMessage struct {
	TenantID uuid.UUID
	Event    *Event
}

// NewHub creates a new WebSocket hub
func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
		logger:     logger,
		done:       make(chan struct{}),
	}
}

// Run starts the hub event loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return
		case <-h.done:
			return
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		case message := <-h.broadcast:
			h.broadcastToTenant(message)
		}
	}
}

// Stop shuts down the hub
func (h *Hub) Stop() {
	close(h.done)
}

// Register adds a client to the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends an event to all clients of a tenant
func (h *Hub) Broadcast(tenantID uuid.UUID, event *Event) {
	select {
	case h.broadcast <- &BroadcastMessage{TenantID: tenantID, Event: event}:
	default:
		h.logger.Warn("broadcast channel full, dropping message",
			"tenant_id", tenantID,
			"event_type", event.Type)
	}
}

// registerClient adds a client to the hub
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.TenantID]; !ok {
		h.clients[client.TenantID] = make(map[*Client]bool)
	}
	h.clients[client.TenantID][client] = true

	h.logger.Debug("client registered",
		"tenant_id", client.TenantID,
		"user_id", client.UserID,
		"total_clients", len(h.clients[client.TenantID]))
}

// unregisterClient removes a client from the hub
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.clients[client.TenantID]; ok {
		if _, ok := clients[client]; ok {
			delete(clients, client)
			close(client.send)

			if len(clients) == 0 {
				delete(h.clients, client.TenantID)
			}

			h.logger.Debug("client unregistered",
				"tenant_id", client.TenantID,
				"user_id", client.UserID)
		}
	}
}

// broadcastToTenant sends a message to all clients of a tenant
func (h *Hub) broadcastToTenant(message *BroadcastMessage) {
	h.mu.RLock()
	clients, ok := h.clients[message.TenantID]
	if !ok {
		h.mu.RUnlock()
		return
	}

	// Copy slice to avoid holding lock during send
	clientList := make([]*Client, 0, len(clients))
	for client := range clients {
		clientList = append(clientList, client)
	}
	h.mu.RUnlock()

	for _, client := range clientList {
		select {
		case client.send <- message.Event:
		default:
			// Client's buffer is full, close connection
			h.unregister <- client
		}
	}
}

// shutdown closes all client connections
func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, clients := range h.clients {
		for client := range clients {
			close(client.send)
		}
	}
	h.clients = make(map[uuid.UUID]map[*Client]bool)
}

// GetConnectionCount returns the number of connected clients for a tenant
func (h *Hub) GetConnectionCount(tenantID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[tenantID]; ok {
		return len(clients)
	}
	return 0
}

// GetTotalConnections returns total number of connected clients
func (h *Hub) GetTotalConnections() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	total := 0
	for _, clients := range h.clients {
		total += len(clients)
	}
	return total
}
