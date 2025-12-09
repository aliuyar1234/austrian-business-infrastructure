package websocket

import (
	"time"

	"github.com/google/uuid"
)

// Event types
const (
	EventTypeNewDocument   = "new_document"
	EventTypeSyncProgress  = "sync_progress"
	EventTypeSyncComplete  = "sync_complete"
	EventTypeSyncFailed    = "sync_failed"
	EventTypeDocumentRead  = "document_read"
	EventTypeNotification  = "notification"
	EventTypePong          = "pong"
	EventTypeConnected     = "connected"
	EventTypeError         = "error"
)

// Event represents a WebSocket event
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// NewEvent creates a new event
func NewEvent(eventType string, data interface{}) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// NewDocumentData holds data for new document events
type NewDocumentData struct {
	DocumentID   uuid.UUID `json:"document_id"`
	AccountID    uuid.UUID `json:"account_id"`
	AccountName  string    `json:"account_name"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Sender       string    `json:"sender"`
	ReceivedAt   time.Time `json:"received_at"`
	Priority     int       `json:"priority"`
}

// SyncProgressData holds data for sync progress events
type SyncProgressData struct {
	JobID            uuid.UUID  `json:"job_id"`
	AccountID        *uuid.UUID `json:"account_id,omitempty"`
	AccountName      string     `json:"account_name,omitempty"`
	Status           string     `json:"status"`
	DocumentsFound   int        `json:"documents_found"`
	DocumentsNew     int        `json:"documents_new"`
	DocumentsSkipped int        `json:"documents_skipped"`
	Progress         float64    `json:"progress,omitempty"` // 0-100
}

// SyncCompleteData holds data for sync complete events
type SyncCompleteData struct {
	JobID            uuid.UUID `json:"job_id"`
	AccountID        *uuid.UUID `json:"account_id,omitempty"`
	DocumentsFound   int        `json:"documents_found"`
	DocumentsNew     int        `json:"documents_new"`
	DocumentsSkipped int        `json:"documents_skipped"`
	Duration         float64    `json:"duration_seconds"`
}

// SyncFailedData holds data for sync failed events
type SyncFailedData struct {
	JobID        uuid.UUID  `json:"job_id"`
	AccountID    *uuid.UUID `json:"account_id,omitempty"`
	ErrorMessage string     `json:"error_message"`
}

// NotificationData holds data for notification events
type NotificationData struct {
	ID      uuid.UUID `json:"id"`
	Type    string    `json:"type"`
	Title   string    `json:"title"`
	Message string    `json:"message"`
}

// ErrorData holds data for error events
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ConnectedData holds data for connected events
type ConnectedData struct {
	ClientID       string `json:"client_id"`
	ReconnectDelay int    `json:"reconnect_delay"` // seconds
}

// Helper functions to create events

// NewDocumentEvent creates a new document event
func NewDocumentEvent(data *NewDocumentData) *Event {
	return NewEvent(EventTypeNewDocument, data)
}

// SyncProgressEvent creates a sync progress event
func SyncProgressEvent(data *SyncProgressData) *Event {
	return NewEvent(EventTypeSyncProgress, data)
}

// SyncCompleteEvent creates a sync complete event
func SyncCompleteEvent(data *SyncCompleteData) *Event {
	return NewEvent(EventTypeSyncComplete, data)
}

// SyncFailedEvent creates a sync failed event
func SyncFailedEvent(data *SyncFailedData) *Event {
	return NewEvent(EventTypeSyncFailed, data)
}

// NotificationEvent creates a notification event
func NotificationEvent(data *NotificationData) *Event {
	return NewEvent(EventTypeNotification, data)
}

// ErrorEvent creates an error event
func ErrorEvent(code, message string) *Event {
	return NewEvent(EventTypeError, &ErrorData{Code: code, Message: message})
}

// ConnectedEvent creates a connected event
func ConnectedEvent(clientID string, reconnectDelay int) *Event {
	return NewEvent(EventTypeConnected, &ConnectedData{
		ClientID:       clientID,
		ReconnectDelay: reconnectDelay,
	})
}
