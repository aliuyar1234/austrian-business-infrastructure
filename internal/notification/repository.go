package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Errors
var (
	ErrPreferencesNotFound = errors.New("preferences not found")
)

// NotificationMode constants
const (
	ModeImmediate = "immediate"
	ModeDigest    = "digest"
	ModeOff       = "off"
)

// NotificationPreferences represents user notification settings
type NotificationPreferences struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	TenantID      uuid.UUID
	EmailEnabled  bool
	EmailMode     string   // immediate, digest, off
	DigestTime    string   // HH:MM format for daily digest
	DocumentTypes []string // empty = all types
	AccountIDs    []uuid.UUID // empty = all accounts
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NotificationQueueItem represents a queued notification
type NotificationQueueItem struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	UserID       uuid.UUID
	DocumentID   uuid.UUID
	Type         string
	Status       string
	Attempts     int
	LastError    string
	ScheduledFor time.Time
	SentAt       *time.Time
	CreatedAt    time.Time
}

// Repository handles notification database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new notification repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetPreferences retrieves notification preferences for a user
func (r *Repository) GetPreferences(ctx context.Context, userID, tenantID uuid.UUID) (*NotificationPreferences, error) {
	query := `
		SELECT id, user_id, tenant_id, email_enabled, email_mode, digest_time,
		       document_types, account_ids, created_at, updated_at
		FROM notification_preferences
		WHERE user_id = $1 AND tenant_id = $2
	`

	var prefs NotificationPreferences
	err := r.db.QueryRow(ctx, query, userID, tenantID).Scan(
		&prefs.ID, &prefs.UserID, &prefs.TenantID,
		&prefs.EmailEnabled, &prefs.EmailMode, &prefs.DigestTime,
		&prefs.DocumentTypes, &prefs.AccountIDs,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPreferencesNotFound
		}
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	return &prefs, nil
}

// UpsertPreferences creates or updates notification preferences
func (r *Repository) UpsertPreferences(ctx context.Context, prefs *NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
			id, user_id, tenant_id, email_enabled, email_mode, digest_time,
			document_types, account_ids, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, tenant_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			email_mode = EXCLUDED.email_mode,
			digest_time = EXCLUDED.digest_time,
			document_types = EXCLUDED.document_types,
			account_ids = EXCLUDED.account_ids,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if prefs.ID == uuid.Nil {
		prefs.ID = uuid.New()
	}
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = now
	}
	prefs.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		prefs.ID, prefs.UserID, prefs.TenantID,
		prefs.EmailEnabled, prefs.EmailMode, prefs.DigestTime,
		prefs.DocumentTypes, prefs.AccountIDs,
		prefs.CreatedAt, prefs.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("upsert preferences: %w", err)
	}

	return nil
}

// QueueNotification adds a notification to the queue
func (r *Repository) QueueNotification(ctx context.Context, item *NotificationQueueItem) error {
	query := `
		INSERT INTO notification_queue (
			id, tenant_id, user_id, document_id, type, status,
			attempts, last_error, scheduled_for, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	if item.ScheduledFor.IsZero() {
		item.ScheduledFor = now
	}
	if item.Status == "" {
		item.Status = "pending"
	}

	_, err := r.db.Exec(ctx, query,
		item.ID, item.TenantID, item.UserID, item.DocumentID, item.Type, item.Status,
		item.Attempts, item.LastError, item.ScheduledFor, item.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("queue notification: %w", err)
	}

	return nil
}

// GetPendingNotifications retrieves pending notifications ready to send
func (r *Repository) GetPendingNotifications(ctx context.Context, limit int) ([]*NotificationQueueItem, error) {
	query := `
		SELECT id, tenant_id, user_id, document_id, type, status,
		       attempts, last_error, scheduled_for, sent_at, created_at
		FROM notification_queue
		WHERE status = 'pending' AND scheduled_for <= NOW() AND attempts < 3
		ORDER BY scheduled_for ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending notifications: %w", err)
	}
	defer rows.Close()

	var items []*NotificationQueueItem
	for rows.Next() {
		var item NotificationQueueItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.UserID, &item.DocumentID, &item.Type, &item.Status,
			&item.Attempts, &item.LastError, &item.ScheduledFor, &item.SentAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		items = append(items, &item)
	}

	return items, nil
}

// MarkNotificationSent marks a notification as sent
func (r *Repository) MarkNotificationSent(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE notification_queue
		SET status = 'sent', sent_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("mark notification sent: %w", err)
	}

	return nil
}

// MarkNotificationFailed marks a notification as failed
func (r *Repository) MarkNotificationFailed(ctx context.Context, id uuid.UUID, errorMsg string) error {
	query := `
		UPDATE notification_queue
		SET status = CASE WHEN attempts >= 2 THEN 'failed' ELSE 'pending' END,
		    attempts = attempts + 1,
		    last_error = $2,
		    scheduled_for = NOW() + (INTERVAL '1 second' * POWER(2, attempts))
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id, errorMsg)
	if err != nil {
		return fmt.Errorf("mark notification failed: %w", err)
	}

	return nil
}

// GetDigestItems retrieves notifications for digest email
func (r *Repository) GetDigestItems(ctx context.Context, userID, tenantID uuid.UUID, since time.Time) ([]*NotificationQueueItem, error) {
	query := `
		SELECT id, tenant_id, user_id, document_id, type, status,
		       attempts, last_error, scheduled_for, sent_at, created_at
		FROM notification_queue
		WHERE user_id = $1 AND tenant_id = $2 AND created_at >= $3 AND type = 'digest'
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID, tenantID, since)
	if err != nil {
		return nil, fmt.Errorf("get digest items: %w", err)
	}
	defer rows.Close()

	var items []*NotificationQueueItem
	for rows.Next() {
		var item NotificationQueueItem
		if err := rows.Scan(
			&item.ID, &item.TenantID, &item.UserID, &item.DocumentID, &item.Type, &item.Status,
			&item.Attempts, &item.LastError, &item.ScheduledFor, &item.SentAt, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan digest item: %w", err)
		}
		items = append(items, &item)
	}

	return items, nil
}

// GetUsersWithDigestEnabled returns users who have digest mode enabled
func (r *Repository) GetUsersWithDigestEnabled(ctx context.Context, digestTime string) ([]NotificationPreferences, error) {
	query := `
		SELECT id, user_id, tenant_id, email_enabled, email_mode, digest_time,
		       document_types, account_ids, created_at, updated_at
		FROM notification_preferences
		WHERE email_enabled = true AND email_mode = 'digest' AND digest_time = $1
	`

	rows, err := r.db.Query(ctx, query, digestTime)
	if err != nil {
		return nil, fmt.Errorf("get digest users: %w", err)
	}
	defer rows.Close()

	var prefs []NotificationPreferences
	for rows.Next() {
		var p NotificationPreferences
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.TenantID,
			&p.EmailEnabled, &p.EmailMode, &p.DigestTime,
			&p.DocumentTypes, &p.AccountIDs,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan preferences: %w", err)
		}
		prefs = append(prefs, p)
	}

	return prefs, nil
}
