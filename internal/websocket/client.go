package websocket

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = 30 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 512

	// Size of the client send buffer
	sendBufferSize = 256
)

// Client represents a WebSocket connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan *Event
	TenantID uuid.UUID
	UserID   uuid.UUID
	logger   *slog.Logger
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, tenantID, userID uuid.UUID, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan *Event, sendBufferSize),
		TenantID: tenantID,
		UserID:   userID,
		logger:   logger,
	}
}

// ReadPump reads messages from the WebSocket connection
// The client is responsible for calling this in a goroutine
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
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
				c.logger.Warn("websocket read error",
					"tenant_id", c.TenantID,
					"user_id", c.UserID,
					"error", err)
			}
			break
		}

		// Handle client messages (e.g., ping, subscribe)
		c.handleMessage(message)
	}
}

// WritePump writes messages to the WebSocket connection
// The client is responsible for calling this in a goroutine
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case event, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.writeEvent(event); err != nil {
				c.logger.Warn("websocket write error",
					"tenant_id", c.TenantID,
					"user_id", c.UserID,
					"error", err)
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

// handleMessage processes incoming client messages
func (c *Client) handleMessage(message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.logger.Debug("invalid client message",
			"tenant_id", c.TenantID,
			"error", err)
		return
	}

	switch msg.Type {
	case "ping":
		// Send pong response
		c.send <- &Event{
			Type:      EventTypePong,
			Timestamp: time.Now(),
		}
	case "subscribe":
		// Handle subscription to specific events
		c.logger.Debug("client subscribed",
			"tenant_id", c.TenantID,
			"events", msg.Events)
	default:
		c.logger.Debug("unknown client message type",
			"tenant_id", c.TenantID,
			"type", msg.Type)
	}
}

// writeEvent writes an event as JSON to the WebSocket
func (c *Client) writeEvent(event *Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type   string   `json:"type"`
	Events []string `json:"events,omitempty"`
}
