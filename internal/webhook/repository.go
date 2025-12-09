package webhook

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrWebhookNotFound  = errors.New("webhook not found")
	ErrDeliveryNotFound = errors.New("delivery not found")
)

// Webhook represents a webhook configuration
type Webhook struct {
	ID             uuid.UUID         `json:"id"`
	TenantID       uuid.UUID         `json:"tenant_id"`
	Name           string            `json:"name"`
	URL            string            `json:"url"`
	Secret         string            `json:"-"` // Never expose secret
	Events         []string          `json:"events"`
	Enabled        bool              `json:"enabled"`
	TimeoutSeconds int               `json:"timeout_seconds"`
	MaxRetries     int               `json:"max_retries"`
	Headers        map[string]string `json:"headers,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// Delivery represents a webhook delivery attempt
type Delivery struct {
	ID              uuid.UUID   `json:"id"`
	WebhookID       uuid.UUID   `json:"webhook_id"`
	TenantID        uuid.UUID   `json:"tenant_id"`
	EventType       string      `json:"event_type"`
	Payload         interface{} `json:"payload"`
	Status          string      `json:"status"` // pending, success, failed
	ResponseStatus  *int        `json:"response_status,omitempty"`
	ResponseBody    string      `json:"response_body,omitempty"`
	ResponseHeaders interface{} `json:"response_headers,omitempty"`
	AttemptCount    int         `json:"attempt_count"`
	LastError       string      `json:"last_error,omitempty"`
	NextRetryAt     *time.Time  `json:"next_retry_at,omitempty"`
	DeliveredAt     *time.Time  `json:"delivered_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
}

// Repository handles webhook database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new webhook repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new webhook
func (r *Repository) Create(ctx context.Context, wh *Webhook) error {
	now := time.Now()
	if wh.ID == uuid.Nil {
		wh.ID = uuid.New()
	}
	wh.CreatedAt = now
	wh.UpdatedAt = now

	if wh.TimeoutSeconds == 0 {
		wh.TimeoutSeconds = 30
	}
	if wh.MaxRetries == 0 {
		wh.MaxRetries = 3
	}

	query := `
		INSERT INTO webhooks (
			id, tenant_id, name, url, secret, events, enabled,
			timeout_seconds, max_retries, headers, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.Exec(ctx, query,
		wh.ID, wh.TenantID, wh.Name, wh.URL, wh.Secret, wh.Events, wh.Enabled,
		wh.TimeoutSeconds, wh.MaxRetries, wh.Headers, wh.CreatedAt, wh.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create webhook: %w", err)
	}

	return nil
}

// GetByID retrieves a webhook by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, events, enabled,
		       timeout_seconds, max_retries, headers, created_at, updated_at
		FROM webhooks WHERE id = $1
	`

	wh := &Webhook{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events, &wh.Enabled,
		&wh.TimeoutSeconds, &wh.MaxRetries, &wh.Headers, &wh.CreatedAt, &wh.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWebhookNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get webhook: %w", err)
	}

	return wh, nil
}

// List retrieves webhooks for a tenant
func (r *Repository) List(ctx context.Context, tenantID uuid.UUID, enabledOnly bool) ([]*Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, events, enabled,
		       timeout_seconds, max_retries, headers, created_at, updated_at
		FROM webhooks WHERE tenant_id = $1
	`

	if enabledOnly {
		query += " AND enabled = TRUE"
	}

	query += " ORDER BY name"

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list webhooks: %w", err)
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		wh := &Webhook{}
		err := rows.Scan(
			&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events, &wh.Enabled,
			&wh.TimeoutSeconds, &wh.MaxRetries, &wh.Headers, &wh.CreatedAt, &wh.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		webhooks = append(webhooks, wh)
	}

	return webhooks, rows.Err()
}

// ListByEvent retrieves enabled webhooks subscribed to a specific event
func (r *Repository) ListByEvent(ctx context.Context, tenantID uuid.UUID, eventType string) ([]*Webhook, error) {
	query := `
		SELECT id, tenant_id, name, url, secret, events, enabled,
		       timeout_seconds, max_retries, headers, created_at, updated_at
		FROM webhooks
		WHERE tenant_id = $1 AND enabled = TRUE AND $2 = ANY(events)
		ORDER BY name
	`

	rows, err := r.db.Query(ctx, query, tenantID, eventType)
	if err != nil {
		return nil, fmt.Errorf("list webhooks by event: %w", err)
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		wh := &Webhook{}
		err := rows.Scan(
			&wh.ID, &wh.TenantID, &wh.Name, &wh.URL, &wh.Secret, &wh.Events, &wh.Enabled,
			&wh.TimeoutSeconds, &wh.MaxRetries, &wh.Headers, &wh.CreatedAt, &wh.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan webhook: %w", err)
		}
		webhooks = append(webhooks, wh)
	}

	return webhooks, rows.Err()
}

// Update updates a webhook
func (r *Repository) Update(ctx context.Context, wh *Webhook) error {
	wh.UpdatedAt = time.Now()

	query := `
		UPDATE webhooks SET
			name = $1, url = $2, events = $3, enabled = $4,
			timeout_seconds = $5, max_retries = $6, headers = $7, updated_at = $8
		WHERE id = $9 AND tenant_id = $10
	`

	result, err := r.db.Exec(ctx, query,
		wh.Name, wh.URL, wh.Events, wh.Enabled,
		wh.TimeoutSeconds, wh.MaxRetries, wh.Headers, wh.UpdatedAt,
		wh.ID, wh.TenantID,
	)
	if err != nil {
		return fmt.Errorf("update webhook: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWebhookNotFound
	}

	return nil
}

// UpdateSecret updates the webhook secret
func (r *Repository) UpdateSecret(ctx context.Context, id uuid.UUID, secret string) error {
	query := `UPDATE webhooks SET secret = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, secret, id)
	return err
}

// Delete deletes a webhook
func (r *Repository) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	query := `DELETE FROM webhooks WHERE id = $1 AND tenant_id = $2`
	result, err := r.db.Exec(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWebhookNotFound
	}

	return nil
}

// CreateDelivery creates a new delivery record
func (r *Repository) CreateDelivery(ctx context.Context, d *Delivery) error {
	now := time.Now()
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	d.CreatedAt = now

	query := `
		INSERT INTO webhook_deliveries (
			id, webhook_id, tenant_id, event_type, payload, status,
			attempt_count, next_retry_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Exec(ctx, query,
		d.ID, d.WebhookID, d.TenantID, d.EventType, d.Payload, d.Status,
		d.AttemptCount, d.NextRetryAt, d.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create delivery: %w", err)
	}

	return nil
}

// UpdateDeliverySuccess marks a delivery as successful
func (r *Repository) UpdateDeliverySuccess(ctx context.Context, id uuid.UUID, responseStatus int, responseBody string, headers interface{}) error {
	now := time.Now()
	query := `
		UPDATE webhook_deliveries SET
			status = 'success', response_status = $1, response_body = $2,
			response_headers = $3, delivered_at = $4, attempt_count = attempt_count + 1
		WHERE id = $5
	`

	_, err := r.db.Exec(ctx, query, responseStatus, responseBody, headers, now, id)
	return err
}

// UpdateDeliveryFailure marks a delivery as failed and schedules retry
func (r *Repository) UpdateDeliveryFailure(ctx context.Context, id uuid.UUID, errorMsg string, responseStatus *int, nextRetryAt *time.Time) error {
	query := `
		UPDATE webhook_deliveries SET
			status = CASE WHEN $4 IS NULL THEN 'failed' ELSE 'pending' END,
			last_error = $1, response_status = $2, next_retry_at = $4,
			attempt_count = attempt_count + 1
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, errorMsg, responseStatus, id, nextRetryAt)
	return err
}

// GetPendingDeliveries retrieves deliveries ready for retry
func (r *Repository) GetPendingDeliveries(ctx context.Context, limit int) ([]*Delivery, error) {
	query := `
		SELECT d.id, d.webhook_id, d.tenant_id, d.event_type, d.payload, d.status,
		       d.response_status, d.response_body, d.response_headers, d.attempt_count,
		       d.last_error, d.next_retry_at, d.delivered_at, d.created_at
		FROM webhook_deliveries d
		WHERE d.status = 'pending' AND (d.next_retry_at IS NULL OR d.next_retry_at <= NOW())
		ORDER BY d.created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get pending deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		d := &Delivery{}
		err := rows.Scan(
			&d.ID, &d.WebhookID, &d.TenantID, &d.EventType, &d.Payload, &d.Status,
			&d.ResponseStatus, &d.ResponseBody, &d.ResponseHeaders, &d.AttemptCount,
			&d.LastError, &d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan delivery: %w", err)
		}
		deliveries = append(deliveries, d)
	}

	return deliveries, rows.Err()
}

// ListDeliveries lists deliveries for a webhook
func (r *Repository) ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*Delivery, int, error) {
	// Count total
	var total int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM webhook_deliveries WHERE webhook_id = $1`, webhookID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count deliveries: %w", err)
	}

	// Fetch rows
	query := `
		SELECT id, webhook_id, tenant_id, event_type, payload, status,
		       response_status, response_body, response_headers, attempt_count,
		       last_error, next_retry_at, delivered_at, created_at
		FROM webhook_deliveries WHERE webhook_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, webhookID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*Delivery
	for rows.Next() {
		d := &Delivery{}
		err := rows.Scan(
			&d.ID, &d.WebhookID, &d.TenantID, &d.EventType, &d.Payload, &d.Status,
			&d.ResponseStatus, &d.ResponseBody, &d.ResponseHeaders, &d.AttemptCount,
			&d.LastError, &d.NextRetryAt, &d.DeliveredAt, &d.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan delivery: %w", err)
		}
		deliveries = append(deliveries, d)
	}

	return deliveries, total, rows.Err()
}
