package message

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMessageNotFound    = errors.New("message not found")
	ErrThreadNotFound     = errors.New("thread not found")
	ErrAttachmentNotFound = errors.New("attachment not found")
)

// Message represents a chat message
type Message struct {
	ID         uuid.UUID    `json:"id"`
	ThreadID   uuid.UUID    `json:"thread_id"`
	SenderType string       `json:"sender_type"` // 'staff' or 'client'
	SenderID   uuid.UUID    `json:"sender_id"`
	Content    string       `json:"content"`
	ReadAt     *time.Time   `json:"read_at,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`

	// Joined fields
	SenderName string       `json:"sender_name,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Thread represents a message thread between staff and client
type Thread struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     uuid.UUID  `json:"tenant_id"`
	ClientID     uuid.UUID  `json:"client_id"`
	Subject      string     `json:"subject"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`

	// Joined fields
	ClientName    string `json:"client_name,omitempty"`
	UnreadCount   int    `json:"unread_count,omitempty"`
	LastMessage   string `json:"last_message,omitempty"`
}

// Attachment represents a file attached to a message
type Attachment struct {
	ID           uuid.UUID `json:"id"`
	MessageID    uuid.UUID `json:"message_id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	ContentType  string    `json:"content_type"`
	StoragePath  string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

// Repository provides message data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new message repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// CreateThread creates a new message thread
func (r *Repository) CreateThread(ctx context.Context, thread *Thread) error {
	if thread.ID == uuid.Nil {
		thread.ID = uuid.New()
	}

	query := `
		INSERT INTO message_threads (id, tenant_id, client_id, subject)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		thread.ID,
		thread.TenantID,
		thread.ClientID,
		thread.Subject,
	).Scan(&thread.CreatedAt)

	return err
}

// GetThreadByID retrieves a thread by ID
func (r *Repository) GetThreadByID(ctx context.Context, id uuid.UUID) (*Thread, error) {
	query := `
		SELECT t.id, t.tenant_id, t.client_id, t.subject, t.last_message_at, t.created_at,
			c.name as client_name
		FROM message_threads t
		JOIN clients c ON t.client_id = c.id
		WHERE t.id = $1
	`

	thread := &Thread{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&thread.ID, &thread.TenantID, &thread.ClientID, &thread.Subject,
		&thread.LastMessageAt, &thread.CreatedAt, &thread.ClientName,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrThreadNotFound
		}
		return nil, err
	}

	return thread, nil
}

// GetOrCreateThread gets an existing thread or creates a new one
func (r *Repository) GetOrCreateThread(ctx context.Context, tenantID, clientID uuid.UUID, subject string) (*Thread, error) {
	// Try to find existing thread
	query := `
		SELECT t.id, t.tenant_id, t.client_id, t.subject, t.last_message_at, t.created_at,
			c.name as client_name
		FROM message_threads t
		JOIN clients c ON t.client_id = c.id
		WHERE t.tenant_id = $1 AND t.client_id = $2 AND t.subject = $3
	`

	thread := &Thread{}
	err := r.pool.QueryRow(ctx, query, tenantID, clientID, subject).Scan(
		&thread.ID, &thread.TenantID, &thread.ClientID, &thread.Subject,
		&thread.LastMessageAt, &thread.CreatedAt, &thread.ClientName,
	)

	if err == nil {
		return thread, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create new thread
	thread = &Thread{
		TenantID: tenantID,
		ClientID: clientID,
		Subject:  subject,
	}

	if err := r.CreateThread(ctx, thread); err != nil {
		return nil, err
	}

	return thread, nil
}

// ListThreadsForClient returns threads for a client
func (r *Repository) ListThreadsForClient(ctx context.Context, clientID uuid.UUID, limit, offset int) ([]*Thread, int, error) {
	countQuery := `SELECT COUNT(*) FROM message_threads WHERE client_id = $1`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, clientID).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT t.id, t.tenant_id, t.client_id, t.subject, t.last_message_at, t.created_at,
			(SELECT COUNT(*) FROM messages m WHERE m.thread_id = t.id AND m.sender_type = 'staff' AND m.read_at IS NULL) as unread_count,
			(SELECT content FROM messages m WHERE m.thread_id = t.id ORDER BY m.created_at DESC LIMIT 1) as last_message
		FROM message_threads t
		WHERE t.client_id = $1
		ORDER BY COALESCE(t.last_message_at, t.created_at) DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, listQuery, clientID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var threads []*Thread
	for rows.Next() {
		thread := &Thread{}
		var lastMessage *string
		err := rows.Scan(
			&thread.ID, &thread.TenantID, &thread.ClientID, &thread.Subject,
			&thread.LastMessageAt, &thread.CreatedAt, &thread.UnreadCount, &lastMessage,
		)
		if err != nil {
			return nil, 0, err
		}
		if lastMessage != nil {
			thread.LastMessage = *lastMessage
		}
		threads = append(threads, thread)
	}

	return threads, total, rows.Err()
}

// ListThreadsForTenant returns threads for a tenant
func (r *Repository) ListThreadsForTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Thread, int, error) {
	countQuery := `SELECT COUNT(*) FROM message_threads WHERE tenant_id = $1`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT t.id, t.tenant_id, t.client_id, t.subject, t.last_message_at, t.created_at,
			c.name as client_name,
			(SELECT COUNT(*) FROM messages m WHERE m.thread_id = t.id AND m.sender_type = 'client' AND m.read_at IS NULL) as unread_count,
			(SELECT content FROM messages m WHERE m.thread_id = t.id ORDER BY m.created_at DESC LIMIT 1) as last_message
		FROM message_threads t
		JOIN clients c ON t.client_id = c.id
		WHERE t.tenant_id = $1
		ORDER BY COALESCE(t.last_message_at, t.created_at) DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, listQuery, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var threads []*Thread
	for rows.Next() {
		thread := &Thread{}
		var lastMessage *string
		err := rows.Scan(
			&thread.ID, &thread.TenantID, &thread.ClientID, &thread.Subject,
			&thread.LastMessageAt, &thread.CreatedAt, &thread.ClientName,
			&thread.UnreadCount, &lastMessage,
		)
		if err != nil {
			return nil, 0, err
		}
		if lastMessage != nil {
			thread.LastMessage = *lastMessage
		}
		threads = append(threads, thread)
	}

	return threads, total, rows.Err()
}

// CreateMessage creates a new message
func (r *Repository) CreateMessage(ctx context.Context, msg *Message) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO messages (id, thread_id, sender_type, sender_id, content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`

	err = tx.QueryRow(ctx, query,
		msg.ID,
		msg.ThreadID,
		msg.SenderType,
		msg.SenderID,
		msg.Content,
	).Scan(&msg.CreatedAt)

	if err != nil {
		return err
	}

	// Update thread's last_message_at
	updateQuery := `UPDATE message_threads SET last_message_at = $1 WHERE id = $2`
	_, err = tx.Exec(ctx, updateQuery, msg.CreatedAt, msg.ThreadID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetMessageByID retrieves a message by ID
func (r *Repository) GetMessageByID(ctx context.Context, id uuid.UUID) (*Message, error) {
	query := `
		SELECT id, thread_id, sender_type, sender_id, content, read_at, created_at
		FROM messages
		WHERE id = $1
	`

	msg := &Message{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&msg.ID, &msg.ThreadID, &msg.SenderType, &msg.SenderID,
		&msg.Content, &msg.ReadAt, &msg.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	return msg, nil
}

// ListMessagesForThread returns messages for a thread
func (r *Repository) ListMessagesForThread(ctx context.Context, threadID uuid.UUID, limit, offset int) ([]*Message, int, error) {
	countQuery := `SELECT COUNT(*) FROM messages WHERE thread_id = $1`

	var total int
	if err := r.pool.QueryRow(ctx, countQuery, threadID).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT m.id, m.thread_id, m.sender_type, m.sender_id, m.content, m.read_at, m.created_at,
			CASE
				WHEN m.sender_type = 'staff' THEN u.name
				WHEN m.sender_type = 'client' THEN c.name
			END as sender_name
		FROM messages m
		LEFT JOIN users u ON m.sender_type = 'staff' AND m.sender_id = u.id
		LEFT JOIN clients c ON m.sender_type = 'client' AND m.sender_id = c.id
		WHERE m.thread_id = $1
		ORDER BY m.created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, listQuery, threadID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var senderName *string
		err := rows.Scan(
			&msg.ID, &msg.ThreadID, &msg.SenderType, &msg.SenderID,
			&msg.Content, &msg.ReadAt, &msg.CreatedAt, &senderName,
		)
		if err != nil {
			return nil, 0, err
		}
		if senderName != nil {
			msg.SenderName = *senderName
		}
		messages = append(messages, msg)
	}

	return messages, total, rows.Err()
}

// MarkAsRead marks messages as read
func (r *Repository) MarkAsRead(ctx context.Context, threadID uuid.UUID, readerType string) error {
	// Mark messages from the opposite sender type as read
	var senderType string
	if readerType == "staff" {
		senderType = "client"
	} else {
		senderType = "staff"
	}

	query := `
		UPDATE messages
		SET read_at = NOW()
		WHERE thread_id = $1 AND sender_type = $2 AND read_at IS NULL
	`

	_, err := r.pool.Exec(ctx, query, threadID, senderType)
	return err
}

// CountUnreadForClient counts unread messages for a client
func (r *Repository) CountUnreadForClient(ctx context.Context, clientID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM messages m
		JOIN message_threads t ON m.thread_id = t.id
		WHERE t.client_id = $1 AND m.sender_type = 'staff' AND m.read_at IS NULL
	`

	var count int
	err := r.pool.QueryRow(ctx, query, clientID).Scan(&count)
	return count, err
}

// CreateAttachment creates a new attachment
func (r *Repository) CreateAttachment(ctx context.Context, att *Attachment) error {
	if att.ID == uuid.Nil {
		att.ID = uuid.New()
	}

	query := `
		INSERT INTO message_attachments (id, message_id, file_name, file_size, content_type, storage_path)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	err := r.pool.QueryRow(ctx, query,
		att.ID,
		att.MessageID,
		att.FileName,
		att.FileSize,
		att.ContentType,
		att.StoragePath,
	).Scan(&att.CreatedAt)

	return err
}

// GetAttachmentByID retrieves an attachment by ID
func (r *Repository) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*Attachment, error) {
	query := `
		SELECT id, message_id, file_name, file_size, content_type, storage_path, created_at
		FROM message_attachments
		WHERE id = $1
	`

	att := &Attachment{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&att.ID, &att.MessageID, &att.FileName, &att.FileSize,
		&att.ContentType, &att.StoragePath, &att.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAttachmentNotFound
		}
		return nil, err
	}

	return att, nil
}

// ListAttachmentsForMessage returns attachments for a message
func (r *Repository) ListAttachmentsForMessage(ctx context.Context, messageID uuid.UUID) ([]*Attachment, error) {
	query := `
		SELECT id, message_id, file_name, file_size, content_type, storage_path, created_at
		FROM message_attachments
		WHERE message_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []*Attachment
	for rows.Next() {
		att := &Attachment{}
		err := rows.Scan(
			&att.ID, &att.MessageID, &att.FileName, &att.FileSize,
			&att.ContentType, &att.StoragePath, &att.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, att)
	}

	return attachments, rows.Err()
}
