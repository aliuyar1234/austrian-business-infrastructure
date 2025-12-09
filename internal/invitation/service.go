package invitation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/user"
	"github.com/austrian-business-infrastructure/fo/pkg/crypto"
	"github.com/google/uuid"
)

const (
	// DefaultExpiryDuration is the default invitation expiry (7 days)
	DefaultExpiryDuration = 7 * 24 * time.Hour
	// TokenLength is the length of generated tokens
	TokenLength = 32
)

var (
	ErrCannotInviteOwner = errors.New("cannot invite as owner role")
	ErrEmailAlreadyInTenant = errors.New("email is already a member of this tenant")
	ErrPendingInvitationExists = errors.New("a pending invitation already exists for this email")
)

// CreateInvitationInput contains input for creating an invitation
type CreateInvitationInput struct {
	TenantID  uuid.UUID
	Email     string
	Role      string
	InvitedBy uuid.UUID
}

// CreateInvitationResult contains the result of creating an invitation
type CreateInvitationResult struct {
	Invitation *Invitation
	Token      string // Plain text token (only returned once)
}

// Service provides invitation business logic
type Service struct {
	repo     *Repository
	userRepo *user.Repository
}

// NewService creates a new invitation service
func NewService(repo *Repository, userRepo *user.Repository) *Service {
	return &Service{
		repo:     repo,
		userRepo: userRepo,
	}
}

// Create creates a new invitation
func (s *Service) Create(ctx context.Context, input *CreateInvitationInput) (*CreateInvitationResult, error) {
	email := normalizeEmail(input.Email)

	// Validate role - cannot invite as owner
	if input.Role == string(user.RoleOwner) {
		return nil, ErrCannotInviteOwner
	}

	if !user.IsValidRole(input.Role) {
		return nil, user.ErrInvalidRole
	}

	// Check if user already exists in tenant
	existingUser, err := s.userRepo.GetByEmail(ctx, input.TenantID, email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailAlreadyInTenant
	}
	if err != nil && !errors.Is(err, user.ErrUserNotFound) {
		return nil, err
	}

	// Check for existing pending invitation
	pending, err := s.repo.ListPendingByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	for _, inv := range pending {
		if inv.TenantID == input.TenantID {
			return nil, ErrPendingInvitationExists
		}
	}

	// Generate token
	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	invitation := &Invitation{
		ID:        uuid.New(),
		TenantID:  input.TenantID,
		Email:     email,
		Role:      input.Role,
		TokenHash: hashTokenForStorage(token),
		InvitedBy: input.InvitedBy,
		ExpiresAt: time.Now().Add(DefaultExpiryDuration),
	}

	if err := s.repo.Create(ctx, invitation); err != nil {
		return nil, err
	}

	return &CreateInvitationResult{
		Invitation: invitation,
		Token:      token,
	}, nil
}

// GetByToken retrieves and validates an invitation by token
func (s *Service) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	invitation, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(invitation.ExpiresAt) {
		return nil, ErrInvitationExpired
	}

	// Check if already accepted
	if invitation.AcceptedAt != nil {
		return nil, ErrInvitationUsed
	}

	return invitation, nil
}

// Accept accepts an invitation and creates the user
func (s *Service) Accept(ctx context.Context, token string, name string, password string) (*user.User, error) {
	invitation, err := s.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Validate and hash password
	if err := crypto.ValidatePassword(password, nil); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	passwordHash, err := crypto.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	newUser, err := s.userRepo.CreateWithPassword(ctx, &user.User{
		TenantID: invitation.TenantID,
		Email:    invitation.Email,
		Name:     name,
		Role:     user.Role(invitation.Role),
		IsActive: true,
	}, passwordHash)

	if err != nil {
		return nil, err
	}

	// Mark invitation as accepted
	if err := s.repo.MarkAccepted(ctx, invitation.ID); err != nil {
		// Log but don't fail - user was created
	}

	return newUser, nil
}

// ListByTenant returns all invitations for a tenant
func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Invitation, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Delete deletes an invitation (cancel)
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// Resend creates a new token for an existing invitation
func (s *Service) Resend(ctx context.Context, id uuid.UUID) (*CreateInvitationResult, error) {
	invitation, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if invitation.AcceptedAt != nil {
		return nil, ErrInvitationUsed
	}

	// Delete old invitation
	if err := s.repo.Delete(ctx, id); err != nil {
		return nil, err
	}

	// Create new invitation with same details
	return s.Create(ctx, &CreateInvitationInput{
		TenantID:  invitation.TenantID,
		Email:     invitation.Email,
		Role:      invitation.Role,
		InvitedBy: invitation.InvitedBy,
	})
}

// generateToken creates a secure random token
func generateToken() (string, error) {
	b := make([]byte, TokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashTokenForStorage creates a SHA-256 hash of a token for storage
func hashTokenForStorage(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// normalizeEmail converts email to lowercase
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
