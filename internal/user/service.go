package user

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/austrian-business-infrastructure/fo/pkg/crypto"
	"github.com/google/uuid"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

var (
	ErrInvalidEmail          = errors.New("invalid email format")
	ErrUserInactive          = errors.New("user is inactive")
	ErrCannotRemoveLastOwner = errors.New("cannot remove the last owner of the tenant")
)

// CreateUserInput contains input for creating a user
type CreateUserInput struct {
	TenantID uuid.UUID
	Email    string
	Password string
	Name     string
	Role     Role
}

// Service provides user business logic
type Service struct {
	repo *Repository
}

// NewService creates a new user service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new user with password
func (s *Service) Create(ctx context.Context, input *CreateUserInput) (*User, error) {
	// Validate email
	if !isValidEmail(input.Email) {
		return nil, ErrInvalidEmail
	}

	// Validate role
	if !IsValidRole(string(input.Role)) {
		return nil, ErrInvalidRole
	}

	// Validate and hash password
	if err := crypto.ValidatePassword(input.Password, nil); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	passwordHash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		TenantID:     input.TenantID,
		Email:        normalizeEmail(input.Email),
		PasswordHash: &passwordHash,
		Name:         input.Name,
		Role:         input.Role,
		IsActive:     true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// CreateOAuthUser creates a user via OAuth
func (s *Service) CreateOAuthUser(ctx context.Context, tenantID uuid.UUID, email, name, provider, oauthID string, avatarURL *string) (*User, error) {
	if !isValidEmail(email) {
		return nil, ErrInvalidEmail
	}

	user := &User{
		TenantID:      tenantID,
		Email:         normalizeEmail(email),
		Name:          name,
		Role:          RoleMember,
		OAuthProvider: &provider,
		OAuthID:       &oauthID,
		AvatarURL:     avatarURL,
		EmailVerified: true, // OAuth emails are considered verified
		IsActive:      true,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByEmail retrieves a user by email within a tenant
func (s *Service) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*User, error) {
	return s.repo.GetByEmail(ctx, tenantID, normalizeEmail(email))
}

// GetByEmailGlobal retrieves a user by email across all tenants
func (s *Service) GetByEmailGlobal(ctx context.Context, email string) (*User, error) {
	return s.repo.GetByEmailGlobal(ctx, normalizeEmail(email))
}

// GetByOAuth retrieves a user by OAuth credentials
func (s *Service) GetByOAuth(ctx context.Context, provider, oauthID string) (*User, error) {
	return s.repo.GetByOAuth(ctx, provider, oauthID)
}

// ListByTenant returns all users for a tenant
func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID) ([]*User, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// Authenticate validates credentials and returns the user
func (s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	user, err := s.repo.GetByEmailGlobal(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, crypto.ErrPasswordInvalid
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	if user.PasswordHash == nil {
		return nil, crypto.ErrPasswordInvalid // OAuth-only user
	}

	if err := crypto.VerifyPassword(password, *user.PasswordHash); err != nil {
		return nil, err
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	return user, nil
}

// UpdatePassword changes a user's password
func (s *Service) UpdatePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.PasswordHash == nil {
		return errors.New("cannot change password for OAuth-only user")
	}

	// Verify current password
	if err := crypto.VerifyPassword(currentPassword, *user.PasswordHash); err != nil {
		return err
	}

	// Validate new password
	if err := crypto.ValidatePassword(newPassword, nil); err != nil {
		return err
	}

	// Hash new password
	newHash, err := crypto.HashPassword(newPassword)
	if err != nil {
		return err
	}

	return s.repo.UpdatePassword(ctx, userID, newHash)
}

// UpdateRole changes a user's role
func (s *Service) UpdateRole(ctx context.Context, userID uuid.UUID, newRole Role, actorID uuid.UUID) error {
	if !IsValidRole(string(newRole)) {
		return ErrInvalidRole
	}

	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// If demoting from owner, ensure there's another owner
	if user.Role == RoleOwner && newRole != RoleOwner {
		count, err := s.repo.CountOwners(ctx, user.TenantID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	user.Role = newRole
	return s.repo.Update(ctx, user)
}

// Deactivate deactivates a user
func (s *Service) Deactivate(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// If deactivating an owner, ensure there's another owner
	if user.Role == RoleOwner {
		count, err := s.repo.CountOwners(ctx, user.TenantID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return ErrCannotRemoveLastOwner
		}
	}

	return s.repo.Deactivate(ctx, userID)
}

// VerifyEmail marks a user's email as verified
func (s *Service) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	return s.repo.VerifyEmail(ctx, userID)
}

// ============== 2FA Methods (Security Layer) ==============

// SetTOTPSecret stores an encrypted TOTP secret for a user
func (s *Service) SetTOTPSecret(ctx context.Context, userID uuid.UUID, encryptedSecret []byte) error {
	return s.repo.SetTOTPSecret(ctx, userID, encryptedSecret)
}

// EnableTOTP enables TOTP 2FA for a user
func (s *Service) EnableTOTP(ctx context.Context, userID uuid.UUID) error {
	return s.repo.EnableTOTP(ctx, userID)
}

// DisableTOTP disables TOTP 2FA for a user (clears secret and recovery codes)
func (s *Service) DisableTOTP(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DisableTOTP(ctx, userID)
}

// SetRecoveryCodes stores encrypted recovery codes for a user
func (s *Service) SetRecoveryCodes(ctx context.Context, userID uuid.UUID, encryptedCodes []byte) error {
	return s.repo.SetRecoveryCodes(ctx, userID, encryptedCodes)
}

// IncrementRecoveryCodesUsed increments the used recovery codes count
func (s *Service) IncrementRecoveryCodesUsed(ctx context.Context, userID uuid.UUID) error {
	return s.repo.IncrementRecoveryCodesUsed(ctx, userID)
}

// VerifyPassword verifies a user's password without performing login
func (s *Service) VerifyPassword(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if user.PasswordHash == nil {
		return crypto.ErrPasswordInvalid // OAuth-only user
	}

	return crypto.VerifyPassword(password, *user.PasswordHash)
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// normalizeEmail converts email to lowercase
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
