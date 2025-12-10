package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/foerderung"
)

// Repository handles monitor database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new monitor repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new monitor
func (r *Repository) Create(ctx context.Context, m *foerderung.ProfilMonitor) error {
	m.ID = uuid.New()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO profil_monitore (
			id, tenant_id, profile_id, is_active, min_score_threshold,
			notification_email, notification_portal, digest_mode,
			last_check_at, last_notification_at, matches_found,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`,
		m.ID, m.TenantID, m.ProfileID, m.IsActive, m.MinScoreThreshold,
		m.NotificationEmail, m.NotificationPortal, m.DigestMode,
		m.LastCheckAt, m.LastNotificationAt, m.MatchesFound,
		m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create monitor: %w", err)
	}

	return nil
}

// GetByID retrieves a monitor by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.ProfilMonitor, error) {
	var m foerderung.ProfilMonitor
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, is_active, min_score_threshold,
			notification_email, notification_portal, digest_mode,
			last_check_at, last_notification_at, matches_found,
			created_at, updated_at
		FROM profil_monitore
		WHERE id = $1
	`, id).Scan(
		&m.ID, &m.TenantID, &m.ProfileID, &m.IsActive, &m.MinScoreThreshold,
		&m.NotificationEmail, &m.NotificationPortal, &m.DigestMode,
		&m.LastCheckAt, &m.LastNotificationAt, &m.MatchesFound,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("monitor not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get monitor: %w", err)
	}

	return &m, nil
}

// GetByIDAndTenant retrieves a monitor ensuring tenant access
func (r *Repository) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.ProfilMonitor, error) {
	var m foerderung.ProfilMonitor
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, profile_id, is_active, min_score_threshold,
			notification_email, notification_portal, digest_mode,
			last_check_at, last_notification_at, matches_found,
			created_at, updated_at
		FROM profil_monitore
		WHERE id = $1 AND tenant_id = $2
	`, id, tenantID).Scan(
		&m.ID, &m.TenantID, &m.ProfileID, &m.IsActive, &m.MinScoreThreshold,
		&m.NotificationEmail, &m.NotificationPortal, &m.DigestMode,
		&m.LastCheckAt, &m.LastNotificationAt, &m.MatchesFound,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("monitor not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get monitor: %w", err)
	}

	return &m, nil
}

// ListByTenant retrieves all monitors for a tenant
func (r *Repository) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*foerderung.ProfilMonitor, int, error) {
	if limit <= 0 {
		limit = 20
	}

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM profil_monitore WHERE tenant_id = $1
	`, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count monitors: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, profile_id, is_active, min_score_threshold,
			notification_email, notification_portal, digest_mode,
			last_check_at, last_notification_at, matches_found,
			created_at, updated_at
		FROM profil_monitore
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]*foerderung.ProfilMonitor, 0)
	for rows.Next() {
		var m foerderung.ProfilMonitor
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.ProfileID, &m.IsActive, &m.MinScoreThreshold,
			&m.NotificationEmail, &m.NotificationPortal, &m.DigestMode,
			&m.LastCheckAt, &m.LastNotificationAt, &m.MatchesFound,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan monitor: %w", err)
		}
		monitors = append(monitors, &m)
	}

	return monitors, total, nil
}

// ListActive retrieves all active monitors
func (r *Repository) ListActive(ctx context.Context) ([]*foerderung.ProfilMonitor, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, profile_id, is_active, min_score_threshold,
			notification_email, notification_portal, digest_mode,
			last_check_at, last_notification_at, matches_found,
			created_at, updated_at
		FROM profil_monitore
		WHERE is_active = true
		ORDER BY last_check_at ASC NULLS FIRST
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list active monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]*foerderung.ProfilMonitor, 0)
	for rows.Next() {
		var m foerderung.ProfilMonitor
		if err := rows.Scan(
			&m.ID, &m.TenantID, &m.ProfileID, &m.IsActive, &m.MinScoreThreshold,
			&m.NotificationEmail, &m.NotificationPortal, &m.DigestMode,
			&m.LastCheckAt, &m.LastNotificationAt, &m.MatchesFound,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan monitor: %w", err)
		}
		monitors = append(monitors, &m)
	}

	return monitors, nil
}

// Update updates a monitor
func (r *Repository) Update(ctx context.Context, m *foerderung.ProfilMonitor) error {
	m.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, `
		UPDATE profil_monitore SET
			is_active = $2, min_score_threshold = $3,
			notification_email = $4, notification_portal = $5, digest_mode = $6,
			last_check_at = $7, last_notification_at = $8, matches_found = $9,
			updated_at = $10
		WHERE id = $1
	`,
		m.ID, m.IsActive, m.MinScoreThreshold,
		m.NotificationEmail, m.NotificationPortal, m.DigestMode,
		m.LastCheckAt, m.LastNotificationAt, m.MatchesFound,
		m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update monitor: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitor not found")
	}

	return nil
}

// Delete deletes a monitor
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Exec(ctx, `DELETE FROM profil_monitore WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete monitor: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("monitor not found")
	}

	return nil
}

// NotificationRepository handles notification database operations
type NotificationRepository struct {
	db *pgxpool.Pool
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new notification
func (r *NotificationRepository) Create(ctx context.Context, n *foerderung.MonitorNotification) error {
	n.ID = uuid.New()
	n.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, `
		INSERT INTO monitor_notifications (
			id, monitor_id, foerderung_id, score, match_summary,
			email_sent, email_sent_at, portal_notified,
			viewed_at, dismissed, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		n.ID, n.MonitorID, n.FoerderungID, n.Score, n.MatchSummary,
		n.EmailSent, n.EmailSentAt, n.PortalNotified,
		n.ViewedAt, n.Dismissed, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	return nil
}

// ListByMonitor retrieves notifications for a monitor
func (r *NotificationRepository) ListByMonitor(ctx context.Context, monitorID uuid.UUID, limit, offset int) ([]*foerderung.MonitorNotification, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, monitor_id, foerderung_id, score, match_summary,
			email_sent, email_sent_at, portal_notified,
			viewed_at, dismissed, created_at
		FROM monitor_notifications
		WHERE monitor_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, monitorID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	notifications := make([]*foerderung.MonitorNotification, 0)
	for rows.Next() {
		var n foerderung.MonitorNotification
		if err := rows.Scan(
			&n.ID, &n.MonitorID, &n.FoerderungID, &n.Score, &n.MatchSummary,
			&n.EmailSent, &n.EmailSentAt, &n.PortalNotified,
			&n.ViewedAt, &n.Dismissed, &n.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, &n)
	}

	return notifications, nil
}

// MarkAsViewed marks a notification as viewed
func (r *NotificationRepository) MarkAsViewed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.db.Exec(ctx, `
		UPDATE monitor_notifications SET viewed_at = $2 WHERE id = $1
	`, id, now)
	return err
}

// Dismiss dismisses a notification
func (r *NotificationRepository) Dismiss(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `
		UPDATE monitor_notifications SET dismissed = true WHERE id = $1
	`, id)
	return err
}
