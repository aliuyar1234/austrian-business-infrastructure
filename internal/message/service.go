package message

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SendMessageRequest contains data for sending a message
type SendMessageRequest struct {
	ThreadID   uuid.UUID `json:"thread_id"`
	SenderType string    `json:"sender_type"`
	SenderID   uuid.UUID `json:"sender_id"`
	Content    string    `json:"content"`
}

// StartThreadRequest contains data for starting a new thread
type StartThreadRequest struct {
	TenantID uuid.UUID `json:"tenant_id"`
	ClientID uuid.UUID `json:"client_id"`
	Subject  string    `json:"subject"`
	Content  string    `json:"content"`
	SenderType string  `json:"sender_type"`
	SenderID uuid.UUID `json:"sender_id"`
}

// Service provides messaging business logic
type Service struct {
	repo *Repository
	pool *pgxpool.Pool
	hub  *Hub
}

// NewService creates a new message service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo: NewRepository(pool),
		pool: pool,
	}
}

// SetHub sets the WebSocket hub for real-time delivery
func (s *Service) SetHub(hub *Hub) {
	s.hub = hub
}

// Repository returns the underlying repository
func (s *Service) Repository() *Repository {
	return s.repo
}

// StartThread creates a new thread with an initial message
func (s *Service) StartThread(ctx context.Context, req *StartThreadRequest) (*Thread, *Message, error) {
	thread, err := s.repo.GetOrCreateThread(ctx, req.TenantID, req.ClientID, req.Subject)
	if err != nil {
		return nil, nil, err
	}

	msg := &Message{
		ThreadID:   thread.ID,
		SenderType: req.SenderType,
		SenderID:   req.SenderID,
		Content:    req.Content,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, nil, err
	}

	// Broadcast via WebSocket if hub is available
	if s.hub != nil {
		s.broadcastMessage(thread, msg)
	}

	return thread, msg, nil
}

// SendMessage sends a message to a thread
func (s *Service) SendMessage(ctx context.Context, req *SendMessageRequest) (*Message, error) {
	msg := &Message{
		ThreadID:   req.ThreadID,
		SenderType: req.SenderType,
		SenderID:   req.SenderID,
		Content:    req.Content,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Broadcast via WebSocket if hub is available
	if s.hub != nil {
		thread, _ := s.repo.GetThreadByID(ctx, req.ThreadID)
		if thread != nil {
			s.broadcastMessage(thread, msg)
		}
	}

	return msg, nil
}

// GetThread retrieves a thread by ID
func (s *Service) GetThread(ctx context.Context, id uuid.UUID) (*Thread, error) {
	return s.repo.GetThreadByID(ctx, id)
}

// ListThreadsForClient returns threads for a client
func (s *Service) ListThreadsForClient(ctx context.Context, clientID uuid.UUID, limit, offset int) ([]*Thread, int, error) {
	return s.repo.ListThreadsForClient(ctx, clientID, limit, offset)
}

// ListThreadsForTenant returns threads for a tenant
func (s *Service) ListThreadsForTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Thread, int, error) {
	return s.repo.ListThreadsForTenant(ctx, tenantID, limit, offset)
}

// ListMessages returns messages for a thread
func (s *Service) ListMessages(ctx context.Context, threadID uuid.UUID, limit, offset int) ([]*Message, int, error) {
	return s.repo.ListMessagesForThread(ctx, threadID, limit, offset)
}

// MarkAsRead marks messages in a thread as read
func (s *Service) MarkAsRead(ctx context.Context, threadID uuid.UUID, readerType string) error {
	return s.repo.MarkAsRead(ctx, threadID, readerType)
}

// CountUnreadForClient counts unread messages for a client
func (s *Service) CountUnreadForClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	return s.repo.CountUnreadForClient(ctx, clientID)
}

// AddAttachment adds an attachment to a message
func (s *Service) AddAttachment(ctx context.Context, att *Attachment) error {
	return s.repo.CreateAttachment(ctx, att)
}

// GetAttachment retrieves an attachment by ID
func (s *Service) GetAttachment(ctx context.Context, id uuid.UUID) (*Attachment, error) {
	return s.repo.GetAttachmentByID(ctx, id)
}

// ListAttachments returns attachments for a message
func (s *Service) ListAttachments(ctx context.Context, messageID uuid.UUID) ([]*Attachment, error) {
	return s.repo.ListAttachmentsForMessage(ctx, messageID)
}

// broadcastMessage sends a message to WebSocket clients
func (s *Service) broadcastMessage(thread *Thread, msg *Message) {
	if s.hub == nil {
		return
	}

	// Send to client
	s.hub.SendToClient(thread.ClientID, &WSMessage{
		Type: "new_message",
		Payload: map[string]interface{}{
			"thread_id":   thread.ID,
			"message":     msg,
		},
	})

	// Send to tenant staff
	s.hub.SendToTenant(thread.TenantID, &WSMessage{
		Type: "new_message",
		Payload: map[string]interface{}{
			"thread_id":   thread.ID,
			"client_id":   thread.ClientID,
			"message":     msg,
		},
	})
}
