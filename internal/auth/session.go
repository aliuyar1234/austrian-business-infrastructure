package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/austrian-business-infrastructure/fo/pkg/cache"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExpired  = errors.New("session has expired")
)

// Session represents an active user session
type Session struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash string     `json:"-"`
	UserAgent        *string    `json:"user_agent,omitempty"`
	IPAddress        *string    `json:"ip_address,omitempty"`
	ExpiresAt        time.Time  `json:"expires_at"`
	CreatedAt        time.Time  `json:"created_at"`
	LastUsedAt       time.Time  `json:"last_used_at"`
}

// SessionManager handles session operations
type SessionManager struct {
	pool        *pgxpool.Pool
	redis       *cache.Client
	sessionTTL  time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(pool *pgxpool.Pool, redis *cache.Client, sessionTTL time.Duration) *SessionManager {
	return &SessionManager{
		pool:       pool,
		redis:      redis,
		sessionTTL: sessionTTL,
	}
}

// CreateSession creates a new session for a user
func (m *SessionManager) CreateSession(ctx context.Context, userID uuid.UUID, refreshToken, userAgent, ipAddress string) (*Session, error) {
	session := &Session{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshTokenHash: hashToken(refreshToken),
		ExpiresAt:        time.Now().Add(m.sessionTTL),
	}

	if userAgent != "" {
		session.UserAgent = &userAgent
	}
	if ipAddress != "" {
		session.IPAddress = &ipAddress
	}

	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, last_used_at
	`

	err := m.pool.QueryRow(ctx, query,
		session.ID,
		session.UserID,
		session.RefreshTokenHash,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
	).Scan(&session.CreatedAt, &session.LastUsedAt)

	if err != nil {
		return nil, err
	}

	// Cache session in Redis for fast validation
	sessionKey := "session:" + session.ID.String()
	if err := m.redis.Set(ctx, sessionKey, userID.String(), m.sessionTTL).Err(); err != nil {
		// Log but don't fail - DB is source of truth
	}

	return session, nil
}

// ValidateRefreshToken validates a refresh token and returns the session
func (m *SessionManager) ValidateRefreshToken(ctx context.Context, refreshToken string) (*Session, error) {
	tokenHash := hashToken(refreshToken)

	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at, last_used_at
		FROM sessions
		WHERE refresh_token_hash = $1
	`

	session := &Session{}
	err := m.pool.QueryRow(ctx, query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenHash,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastUsedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		// Clean up expired session
		_ = m.DeleteSession(ctx, session.ID)
		return nil, ErrSessionExpired
	}

	return session, nil
}

// RotateRefreshToken creates a new refresh token for an existing session
func (m *SessionManager) RotateRefreshToken(ctx context.Context, sessionID uuid.UUID, newRefreshToken string) error {
	newHash := hashToken(newRefreshToken)
	newExpiry := time.Now().Add(m.sessionTTL)

	query := `
		UPDATE sessions
		SET refresh_token_hash = $2, expires_at = $3, last_used_at = NOW()
		WHERE id = $1
	`

	result, err := m.pool.Exec(ctx, query, sessionID, newHash, newExpiry)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrSessionNotFound
	}

	// Update Redis TTL
	sessionKey := "session:" + sessionID.String()
	m.redis.Expire(ctx, sessionKey, m.sessionTTL)

	return nil
}

// DeleteSession removes a session
func (m *SessionManager) DeleteSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM sessions WHERE id = $1`

	_, err := m.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return err
	}

	// Remove from Redis
	sessionKey := "session:" + sessionID.String()
	m.redis.Del(ctx, sessionKey)

	return nil
}

// DeleteByRefreshToken removes a session by refresh token
func (m *SessionManager) DeleteByRefreshToken(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)

	// Get session ID first for Redis cleanup
	var sessionID uuid.UUID
	query := `DELETE FROM sessions WHERE refresh_token_hash = $1 RETURNING id`
	err := m.pool.QueryRow(ctx, query, tokenHash).Scan(&sessionID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // Already deleted
		}
		return err
	}

	// Remove from Redis
	sessionKey := "session:" + sessionID.String()
	m.redis.Del(ctx, sessionKey)

	return nil
}

// DeleteAllUserSessions removes all sessions for a user
func (m *SessionManager) DeleteAllUserSessions(ctx context.Context, userID uuid.UUID) error {
	// Get all session IDs first for Redis cleanup
	query := `SELECT id FROM sessions WHERE user_id = $1`
	rows, err := m.pool.Query(ctx, query, userID)
	if err != nil {
		return err
	}

	var sessionIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		sessionIDs = append(sessionIDs, id)
	}
	rows.Close()

	// Delete from database
	deleteQuery := `DELETE FROM sessions WHERE user_id = $1`
	if _, err := m.pool.Exec(ctx, deleteQuery, userID); err != nil {
		return err
	}

	// Remove from Redis
	for _, id := range sessionIDs {
		sessionKey := "session:" + id.String()
		m.redis.Del(ctx, sessionKey)
	}

	return nil
}

// ListUserSessions returns all active sessions for a user
func (m *SessionManager) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at, last_used_at
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY last_used_at DESC
	`

	rows, err := m.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenHash,
			&session.UserAgent,
			&session.IPAddress,
			&session.ExpiresAt,
			&session.CreatedAt,
			&session.LastUsedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// CleanupExpired removes expired sessions
func (m *SessionManager) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at < NOW()`
	result, err := m.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
