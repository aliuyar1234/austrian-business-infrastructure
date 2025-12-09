package client

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvitationExpired = errors.New("invitation has expired")
	ErrInvitationUsed    = errors.New("invitation has already been used")
	ErrInvitationInvalid = errors.New("invalid invitation token")
	ErrNotActive         = errors.New("client is not active")
)

// Invitation represents a client invitation
type Invitation struct {
	ID          uuid.UUID  `json:"id"`
	ClientID    uuid.UUID  `json:"client_id"`
	Token       string     `json:"-"` // Never expose raw token
	TokenHash   string     `json:"-"`
	ExpiresAt   time.Time  `json:"expires_at"`
	Used        bool       `json:"used"`
	UsedAt      *time.Time `json:"used_at,omitempty"`
	InvitedBy   uuid.UUID  `json:"invited_by"`
	EmailSentAt *time.Time `json:"email_sent_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// InviteRequest contains data for inviting a client
type InviteRequest struct {
	Email       string      `json:"email"`
	Name        string      `json:"name"`
	CompanyName *string     `json:"company_name,omitempty"`
	AccountIDs  []uuid.UUID `json:"account_ids"`
}

// InviteResponse contains the result of an invitation
type InviteResponse struct {
	ClientID     uuid.UUID `json:"client_id"`
	InvitationID uuid.UUID `json:"invitation_id"`
	Status       string    `json:"status"`
	InviteSent   bool      `json:"invitation_sent"`
}

// ActivationInfo contains info shown during activation
type ActivationInfo struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	TenantName string `json:"tenant_name"`
}

// Service provides client business logic
type Service struct {
	repo              *Repository
	pool              *pgxpool.Pool
	invitationExpiry  time.Duration
}

// NewService creates a new client service
func NewService(pool *pgxpool.Pool, invitationExpiryHours int) *Service {
	return &Service{
		repo:             NewRepository(pool),
		pool:             pool,
		invitationExpiry: time.Duration(invitationExpiryHours) * time.Hour,
	}
}

// Repository returns the underlying repository
func (s *Service) Repository() *Repository {
	return s.repo
}

// GetByID retrieves a client by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Client, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByUserID retrieves a client by user ID
func (s *Service) GetByUserID(ctx context.Context, userID uuid.UUID) (*Client, error) {
	return s.repo.GetByUserID(ctx, userID)
}

// List returns all clients for a tenant
func (s *Service) List(ctx context.Context, tenantID uuid.UUID, status *Status, limit, offset int) ([]*Client, int, error) {
	return s.repo.ListByTenant(ctx, tenantID, status, limit, offset)
}

// Invite creates a new client invitation
func (s *Service) Invite(ctx context.Context, tenantID uuid.UUID, invitedBy uuid.UUID, req *InviteRequest) (*InviteResponse, error) {
	now := time.Now()

	// Create client record
	client := &Client{
		TenantID:           tenantID,
		Email:              req.Email,
		Name:               req.Name,
		CompanyName:        req.CompanyName,
		Status:             StatusInvited,
		InvitedAt:          &now,
		NotificationEmail:  true,
		NotificationPortal: true,
		Language:           "de",
	}

	if err := s.repo.Create(ctx, client); err != nil {
		return nil, err
	}

	// Grant account access
	for _, accountID := range req.AccountIDs {
		access := &ClientAccountAccess{
			ClientID:         client.ID,
			AccountID:        accountID,
			CanUpload:        true,
			CanViewDocuments: true,
			CanApprove:       true,
		}
		if err := s.repo.AddAccountAccess(ctx, access); err != nil {
			return nil, err
		}
	}

	// Create invitation
	invitation, err := s.createInvitation(ctx, client.ID, invitedBy)
	if err != nil {
		return nil, err
	}

	return &InviteResponse{
		ClientID:     client.ID,
		InvitationID: invitation.ID,
		Status:       string(client.Status),
		InviteSent:   false, // Will be set by handler after sending email
	}, nil
}

// createInvitation creates a new invitation for a client
func (s *Service) createInvitation(ctx context.Context, clientID uuid.UUID, invitedBy uuid.UUID) (*Invitation, error) {
	// Generate secure token
	token, err := generateSecureToken(32)
	if err != nil {
		return nil, err
	}

	tokenHash := hashToken(token)

	invitation := &Invitation{
		ID:        uuid.New(),
		ClientID:  clientID,
		Token:     token,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(s.invitationExpiry),
		Used:      false,
		InvitedBy: invitedBy,
	}

	query := `
		INSERT INTO client_invitations (id, client_id, token, token_hash, expires_at, invited_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at
	`

	err = s.pool.QueryRow(ctx, query,
		invitation.ID,
		invitation.ClientID,
		invitation.Token,
		invitation.TokenHash,
		invitation.ExpiresAt,
		invitation.InvitedBy,
	).Scan(&invitation.CreatedAt)

	if err != nil {
		return nil, err
	}

	return invitation, nil
}

// GetInvitationToken returns the raw token for an invitation (for email sending)
func (s *Service) GetInvitationToken(ctx context.Context, invitationID uuid.UUID) (string, error) {
	query := `SELECT token FROM client_invitations WHERE id = $1 AND NOT used`

	var token string
	err := s.pool.QueryRow(ctx, query, invitationID).Scan(&token)
	if err != nil {
		return "", ErrInvitationInvalid
	}

	return token, nil
}

// ValidateInvitation validates an invitation token and returns activation info
func (s *Service) ValidateInvitation(ctx context.Context, token string) (*ActivationInfo, *Client, error) {
	tokenHash := hashToken(token)

	query := `
		SELECT ci.id, ci.client_id, ci.expires_at, ci.used,
			   c.email, c.name, t.name as tenant_name
		FROM client_invitations ci
		JOIN clients c ON ci.client_id = c.id
		JOIN tenants t ON c.tenant_id = t.id
		WHERE ci.token_hash = $1
	`

	var invitationID, clientID uuid.UUID
	var expiresAt time.Time
	var used bool
	var email, name, tenantName string

	err := s.pool.QueryRow(ctx, query, tokenHash).Scan(
		&invitationID, &clientID, &expiresAt, &used,
		&email, &name, &tenantName,
	)
	if err != nil {
		return nil, nil, ErrInvitationInvalid
	}

	if used {
		return nil, nil, ErrInvitationUsed
	}

	if time.Now().After(expiresAt) {
		return nil, nil, ErrInvitationExpired
	}

	client, err := s.repo.GetByID(ctx, clientID)
	if err != nil {
		return nil, nil, err
	}

	return &ActivationInfo{
		Email:      email,
		Name:       name,
		TenantName: tenantName,
	}, client, nil
}

// ActivateClient completes the activation process
func (s *Service) ActivateClient(ctx context.Context, token string, userID uuid.UUID) (*Client, error) {
	tokenHash := hashToken(token)

	// Validate and get client ID
	query := `
		SELECT ci.client_id
		FROM client_invitations ci
		WHERE ci.token_hash = $1 AND NOT ci.used AND ci.expires_at > NOW()
	`

	var clientID uuid.UUID
	err := s.pool.QueryRow(ctx, query, tokenHash).Scan(&clientID)
	if err != nil {
		return nil, ErrInvitationInvalid
	}

	// Mark invitation as used
	updateQuery := `
		UPDATE client_invitations
		SET used = true, used_at = NOW()
		WHERE token_hash = $1
	`
	_, err = s.pool.Exec(ctx, updateQuery, tokenHash)
	if err != nil {
		return nil, err
	}

	// Activate client
	err = s.repo.Activate(ctx, clientID, userID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, clientID)
}

// ResendInvitation creates a new invitation for an existing invited client
func (s *Service) ResendInvitation(ctx context.Context, clientID uuid.UUID, invitedBy uuid.UUID) (*Invitation, error) {
	client, err := s.repo.GetByID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	if client.Status != StatusInvited {
		return nil, errors.New("client is not in invited status")
	}

	// Invalidate old invitations
	_, err = s.pool.Exec(ctx, `
		UPDATE client_invitations
		SET used = true, used_at = NOW()
		WHERE client_id = $1 AND NOT used
	`, clientID)
	if err != nil {
		return nil, err
	}

	// Create new invitation
	return s.createInvitation(ctx, clientID, invitedBy)
}

// Deactivate deactivates a client
func (s *Service) Deactivate(ctx context.Context, clientID uuid.UUID) error {
	return s.repo.Deactivate(ctx, clientID)
}

// UpdateAccountAccess updates which accounts a client can access
func (s *Service) UpdateAccountAccess(ctx context.Context, clientID uuid.UUID, accountIDs []uuid.UUID) error {
	// Remove all existing access
	_, err := s.pool.Exec(ctx, `DELETE FROM client_account_access WHERE client_id = $1`, clientID)
	if err != nil {
		return err
	}

	// Add new access
	for _, accountID := range accountIDs {
		access := &ClientAccountAccess{
			ClientID:         clientID,
			AccountID:        accountID,
			CanUpload:        true,
			CanViewDocuments: true,
			CanApprove:       true,
		}
		if err := s.repo.AddAccountAccess(ctx, access); err != nil {
			return err
		}
	}

	return nil
}

// GetClientWithAccounts returns a client with their account access info
func (s *Service) GetClientWithAccounts(ctx context.Context, clientID uuid.UUID) (*ClientWithAccounts, error) {
	client, err := s.repo.GetByID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	accounts, err := s.repo.GetAccountAccess(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// Get pending counts
	var pendingUploads, pendingApprovals int
	err = s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM client_uploads WHERE client_id = $1 AND status = 'new'),
			(SELECT COUNT(*) FROM approval_requests WHERE client_id = $1 AND status = 'pending')
	`, clientID).Scan(&pendingUploads, &pendingApprovals)
	if err != nil {
		// Non-critical, continue without counts
		pendingUploads = 0
		pendingApprovals = 0
	}

	return &ClientWithAccounts{
		Client:           *client,
		Accounts:         accounts,
		PendingUploads:   pendingUploads,
		PendingApprovals: pendingApprovals,
	}, nil
}

// ============== Helper Functions ==============

// generateSecureToken generates a cryptographically secure token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.StdEncoding.EncodeToString(hash[:])
}

// MarkInvitationEmailSent marks an invitation as having had its email sent
func (s *Service) MarkInvitationEmailSent(ctx context.Context, invitationID uuid.UUID) error {
	query := `UPDATE client_invitations SET email_sent_at = NOW() WHERE id = $1`
	_, err := s.pool.Exec(ctx, query, invitationID)
	return err
}

// UpdateLastLogin updates the last login time for a client
func (s *Service) UpdateLastLogin(ctx context.Context, clientID uuid.UUID) error {
	return s.repo.UpdateLastLogin(ctx, clientID)
}
