package idaustria

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionStore defines the interface for storing ID Austria sessions
type SessionStore interface {
	// SaveSession saves a new session
	SaveSession(ctx context.Context, session *Session) error
	// GetSessionByState retrieves a session by its state parameter
	GetSessionByState(ctx context.Context, state string) (*Session, error)
	// GetSessionByID retrieves a session by its ID
	GetSessionByID(ctx context.Context, id string) (*Session, error)
	// UpdateSession updates an existing session
	UpdateSession(ctx context.Context, session *Session) error
	// DeleteSession deletes a session
	DeleteSession(ctx context.Context, id string) error
	// CleanupExpiredSessions removes expired sessions
	CleanupExpiredSessions(ctx context.Context) error
}

// SessionManager manages ID Austria authentication sessions
type SessionManager struct {
	store       SessionStore
	client      *Client
	sessionTTL  time.Duration
}

// NewSessionManager creates a new session manager
func NewSessionManager(store SessionStore, client *Client) *SessionManager {
	return &SessionManager{
		store:      store,
		client:     client,
		sessionTTL: 15 * time.Minute, // Sessions expire after 15 minutes
	}
}

// CreateSession creates a new authentication session
func (m *SessionManager) CreateSession(ctx context.Context, signerID, batchID, redirectAfter string) (*Session, string, error) {
	// Create authorization request with PKCE
	authReq, err := m.client.CreateAuthorizationRequest(redirectAfter)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create auth request: %w", err)
	}

	// Generate authorization URL
	authURL, err := m.client.AuthorizationURL(ctx, authReq)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate auth URL: %w", err)
	}

	// Create session
	session := &Session{
		ID:            uuid.New().String(),
		State:         authReq.State,
		Nonce:         authReq.Nonce,
		CodeVerifier:  authReq.CodeVerifier,
		RedirectAfter: authReq.RedirectAfter,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(m.sessionTTL),
		SignerID:      signerID,
		BatchID:       batchID,
		Status:        SessionStatusPending,
	}

	// Save session
	if err := m.store.SaveSession(ctx, session); err != nil {
		return nil, "", fmt.Errorf("failed to save session: %w", err)
	}

	return session, authURL, nil
}

// HandleCallback processes the OIDC callback
func (m *SessionManager) HandleCallback(ctx context.Context, state, code, errorCode, errorDescription string) (*Session, error) {
	// Get session by state
	session, err := m.store.GetSessionByState(ctx, state)
	if err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		session.Status = SessionStatusExpired
		m.store.UpdateSession(ctx, session)
		return nil, &OIDCError{
			Code:        ErrCodeInvalidRequest,
			Description: "session expired",
		}
	}

	// Check if session is already used
	if session.Status != SessionStatusPending {
		return nil, &OIDCError{
			Code:        ErrCodeInvalidRequest,
			Description: fmt.Sprintf("session already in status: %s", session.Status),
		}
	}

	// Validate callback
	if err := m.client.ValidateCallback(session.State, state, code, errorCode, errorDescription); err != nil {
		session.Status = SessionStatusFailed
		session.Error = err.Error()
		m.store.UpdateSession(ctx, session)
		return nil, err
	}

	// Exchange code for tokens
	token, err := m.client.ExchangeCode(ctx, code, session.CodeVerifier)
	if err != nil {
		session.Status = SessionStatusFailed
		session.Error = err.Error()
		m.store.UpdateSession(ctx, session)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := m.client.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		session.Status = SessionStatusFailed
		session.Error = err.Error()
		m.store.UpdateSession(ctx, session)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Update session with results
	session.Status = SessionStatusAuthenticated
	session.Token = token
	session.UserInfo = userInfo
	session.AuthenticatedAt = time.Now()

	if err := m.store.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return session, nil
}

// MarkSessionUsed marks a session as used (after signing is complete)
func (m *SessionManager) MarkSessionUsed(ctx context.Context, sessionID string) error {
	session, err := m.store.GetSessionByID(ctx, sessionID)
	if err != nil {
		return err
	}

	session.Status = SessionStatusUsed
	return m.store.UpdateSession(ctx, session)
}

// GetSession retrieves a session by ID
func (m *SessionManager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return m.store.GetSessionByID(ctx, sessionID)
}

// CleanupSessions removes expired sessions
func (m *SessionManager) CleanupSessions(ctx context.Context) error {
	return m.store.CleanupExpiredSessions(ctx)
}

// HashBPK creates a SHA-256 hash of a BPK for privacy
// BPK should never be stored in plaintext
func HashBPK(bpk string) string {
	hash := sha256.Sum256([]byte(bpk))
	return hex.EncodeToString(hash[:])
}

// InMemorySessionStore is an in-memory implementation of SessionStore for testing
type InMemorySessionStore struct {
	sessions map[string]*Session
}

// NewInMemorySessionStore creates a new in-memory session store
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

// SaveSession saves a session
func (s *InMemorySessionStore) SaveSession(ctx context.Context, session *Session) error {
	s.sessions[session.ID] = session
	return nil
}

// GetSessionByState retrieves a session by state
func (s *InMemorySessionStore) GetSessionByState(ctx context.Context, state string) (*Session, error) {
	for _, session := range s.sessions {
		if session.State == state {
			return session, nil
		}
	}
	return nil, fmt.Errorf("session not found")
}

// GetSessionByID retrieves a session by ID
func (s *InMemorySessionStore) GetSessionByID(ctx context.Context, id string) (*Session, error) {
	session, ok := s.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// UpdateSession updates a session
func (s *InMemorySessionStore) UpdateSession(ctx context.Context, session *Session) error {
	s.sessions[session.ID] = session
	return nil
}

// DeleteSession deletes a session
func (s *InMemorySessionStore) DeleteSession(ctx context.Context, id string) error {
	delete(s.sessions, id)
	return nil
}

// CleanupExpiredSessions removes expired sessions
func (s *InMemorySessionStore) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.ExpiresAt) {
			delete(s.sessions, id)
		}
	}
	return nil
}
