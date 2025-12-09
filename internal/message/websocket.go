package message

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper origin checking based on tenant config
		return true
	},
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	userType   string    // "staff" or "client"
	userID     uuid.UUID
	tenantID   uuid.UUID
	clientID   uuid.UUID // Only for portal clients
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	clients map[uuid.UUID]*Client

	// Clients by tenant ID (for broadcasting to all staff)
	tenantClients map[uuid.UUID]map[*Client]bool

	// Clients by client ID (for portal clients)
	portalClients map[uuid.UUID]*Client

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for concurrent access
	mu sync.RWMutex
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[uuid.UUID]*Client),
		tenantClients: make(map[uuid.UUID]map[*Client]bool),
		portalClients: make(map[uuid.UUID]*Client),
		register:      make(chan *Client),
		unregister:    make(chan *Client),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.userID] = client

			if client.userType == "staff" {
				if h.tenantClients[client.tenantID] == nil {
					h.tenantClients[client.tenantID] = make(map[*Client]bool)
				}
				h.tenantClients[client.tenantID][client] = true
			} else {
				h.portalClients[client.clientID] = client
			}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.userID]; ok {
				delete(h.clients, client.userID)
				close(client.send)

				if client.userType == "staff" {
					if clients, ok := h.tenantClients[client.tenantID]; ok {
						delete(clients, client)
						if len(clients) == 0 {
							delete(h.tenantClients, client.tenantID)
						}
					}
				} else {
					delete(h.portalClients, client.clientID)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID uuid.UUID, msg *WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		select {
		case client.send <- data:
		default:
			// Client's send buffer is full, skip
		}
	}
}

// SendToClient sends a message to a portal client
func (h *Hub) SendToClient(clientID uuid.UUID, msg *WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	client, ok := h.portalClients[clientID]
	h.mu.RUnlock()

	if ok {
		select {
		case client.send <- data:
		default:
			// Client's send buffer is full, skip
		}
	}
}

// SendToTenant sends a message to all staff in a tenant
func (h *Hub) SendToTenant(tenantID uuid.UUID, msg *WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := h.tenantClients[tenantID]
	h.mu.RUnlock()

	for client := range clients {
		select {
		case client.send <- data:
		default:
			// Client's send buffer is full, skip
		}
	}
}

// ServeWS handles WebSocket requests for staff
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, userID, tenantID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		userType: "staff",
		userID:   userID,
		tenantID: tenantID,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// ServePortalWS handles WebSocket requests for portal clients
func (h *Hub) ServePortalWS(w http.ResponseWriter, r *http.Request, clientID, tenantID uuid.UUID) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:      h,
		conn:     conn,
		send:     make(chan []byte, 256),
		userType: "client",
		userID:   clientID, // Use clientID as userID for portal clients
		tenantID: tenantID,
		clientID: clientID,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., typing indicators, read receipts)
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			continue
		}

		c.handleMessage(&wsMsg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(msg *WSMessage) {
	switch msg.Type {
	case "typing":
		// Broadcast typing indicator
		c.broadcastTyping(msg.Payload)
	case "read":
		// Handle read receipt
		c.handleReadReceipt(msg.Payload)
	}
}

// broadcastTyping broadcasts a typing indicator
func (c *Client) broadcastTyping(payload interface{}) {
	data, ok := payload.(map[string]interface{})
	if !ok {
		return
	}

	threadIDStr, ok := data["thread_id"].(string)
	if !ok {
		return
	}

	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return
	}

	msg := &WSMessage{
		Type: "typing",
		Payload: map[string]interface{}{
			"thread_id": threadID,
			"user_id":   c.userID,
			"user_type": c.userType,
		},
	}

	// Send to the other party
	if c.userType == "staff" {
		c.hub.SendToClient(c.clientID, msg)
	} else {
		c.hub.SendToTenant(c.tenantID, msg)
	}
}

// handleReadReceipt handles a read receipt
func (c *Client) handleReadReceipt(payload interface{}) {
	// Read receipts are handled via the REST API to ensure persistence
	// This is just for real-time notification
	data, ok := payload.(map[string]interface{})
	if !ok {
		return
	}

	threadIDStr, ok := data["thread_id"].(string)
	if !ok {
		return
	}

	threadID, err := uuid.Parse(threadIDStr)
	if err != nil {
		return
	}

	msg := &WSMessage{
		Type: "read",
		Payload: map[string]interface{}{
			"thread_id": threadID,
			"user_id":   c.userID,
			"user_type": c.userType,
		},
	}

	// Notify the other party
	if c.userType == "staff" {
		c.hub.SendToClient(c.clientID, msg)
	} else {
		c.hub.SendToTenant(c.tenantID, msg)
	}
}
