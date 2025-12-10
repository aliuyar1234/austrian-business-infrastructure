package websocket

import (
	"github.com/google/uuid"
	"austrian-business-infrastructure/internal/document"
)

// Broadcaster provides methods to broadcast events to connected clients
type Broadcaster struct {
	hub *Hub
}

// NewBroadcaster creates a new broadcaster
func NewBroadcaster(hub *Hub) *Broadcaster {
	return &Broadcaster{hub: hub}
}

// BroadcastNewDocument broadcasts a new document event
func (b *Broadcaster) BroadcastNewDocument(tenantID uuid.UUID, doc *document.Document) {
	if b.hub == nil {
		return
	}

	event := NewDocumentEvent(&NewDocumentData{
		DocumentID:  doc.ID,
		AccountID:   doc.AccountID,
		AccountName: doc.AccountName,
		Type:        doc.Type,
		Title:       doc.Title,
		Sender:      doc.Sender,
		ReceivedAt:  doc.ReceivedAt,
		Priority:    document.TypePriority(doc.Type),
	})

	b.hub.Broadcast(tenantID, event)
}

// BroadcastSyncProgress broadcasts sync progress
func (b *Broadcaster) BroadcastSyncProgress(tenantID, jobID uuid.UUID, accountID *uuid.UUID, accountName string, found, new, skipped int, progress float64) {
	if b.hub == nil {
		return
	}

	event := SyncProgressEvent(&SyncProgressData{
		JobID:            jobID,
		AccountID:        accountID,
		AccountName:      accountName,
		Status:           "running",
		DocumentsFound:   found,
		DocumentsNew:     new,
		DocumentsSkipped: skipped,
		Progress:         progress,
	})

	b.hub.Broadcast(tenantID, event)
}

// BroadcastSyncComplete broadcasts sync completion
func (b *Broadcaster) BroadcastSyncComplete(tenantID, jobID uuid.UUID, accountID *uuid.UUID, found, new, skipped int, durationSecs float64) {
	if b.hub == nil {
		return
	}

	event := SyncCompleteEvent(&SyncCompleteData{
		JobID:            jobID,
		AccountID:        accountID,
		DocumentsFound:   found,
		DocumentsNew:     new,
		DocumentsSkipped: skipped,
		Duration:         durationSecs,
	})

	b.hub.Broadcast(tenantID, event)
}

// BroadcastSyncFailed broadcasts sync failure
func (b *Broadcaster) BroadcastSyncFailed(tenantID, jobID uuid.UUID, accountID *uuid.UUID, errorMessage string) {
	if b.hub == nil {
		return
	}

	event := SyncFailedEvent(&SyncFailedData{
		JobID:        jobID,
		AccountID:    accountID,
		ErrorMessage: errorMessage,
	})

	b.hub.Broadcast(tenantID, event)
}

// BroadcastNotification broadcasts a notification
func (b *Broadcaster) BroadcastNotification(tenantID uuid.UUID, notificationID uuid.UUID, notificationType, title, message string) {
	if b.hub == nil {
		return
	}

	event := NotificationEvent(&NotificationData{
		ID:      notificationID,
		Type:    notificationType,
		Title:   title,
		Message: message,
	})

	b.hub.Broadcast(tenantID, event)
}
