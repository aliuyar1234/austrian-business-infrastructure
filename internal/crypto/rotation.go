package crypto

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrRotationInProgress indicates a key rotation is already in progress
	ErrRotationInProgress = errors.New("key rotation already in progress")
	// ErrNoOldKey indicates the old key is not available for migration
	ErrNoOldKey = errors.New("old key not available for migration")
)

// KeyRotationStatus represents the current state of a key rotation
type KeyRotationStatus struct {
	InProgress    bool      `json:"in_progress"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	OldKeyVersion int       `json:"old_key_version"`
	NewKeyVersion int       `json:"new_key_version"`
	Progress      int       `json:"progress_percent"`
}

// KeyRotationLog represents a log entry for key rotation
type KeyRotationLog struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	OldKeyVersion int        `json:"old_key_version"`
	NewKeyVersion int        `json:"new_key_version"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	Status        string     `json:"status"` // pending, in_progress, completed, failed
	ItemsTotal    int        `json:"items_total"`
	ItemsMigrated int        `json:"items_migrated"`
	ErrorMessage  *string    `json:"error_message,omitempty"`
}

// KeyRotator handles key rotation for a tenant
type KeyRotator struct {
	keyManager *KeyManager
	// In production, this would have database access to:
	// - List all credentials for a tenant
	// - Re-encrypt each one with new key
	// - Update key_version column
	// - Log rotation progress
}

// NewKeyRotator creates a new key rotator
func NewKeyRotator(keyManager *KeyManager) *KeyRotator {
	if keyManager == nil {
		keyManager = GetKeyManager()
	}
	return &KeyRotator{keyManager: keyManager}
}

// RotateCredentialKey re-encrypts a single credential with a new key version
// This is called during key rotation for each credential
//
// Parameters:
//   - oldCiphertext: The credential encrypted with the old key
//   - tenantID: The tenant that owns the credential
//   - credentialID: The unique ID of the credential
//   - oldMasterKey: The previous master key (for decryption)
//   - newMasterKey: The new master key (for re-encryption)
//
// Returns:
//   - newCiphertext: The credential encrypted with the new key
func (kr *KeyRotator) RotateCredentialKey(
	ctx context.Context,
	oldCiphertext []byte,
	tenantID, credentialID uuid.UUID,
	oldMasterKey, newMasterKey []byte,
) ([]byte, error) {
	// Derive old credential key
	oldTenantKey, err := DeriveTenantKey(oldMasterKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive old tenant key: %w", err)
	}
	defer Zero(oldTenantKey)

	oldCredKey, err := DeriveCredentialKey(oldTenantKey, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive old credential key: %w", err)
	}
	defer Zero(oldCredKey)

	// Decrypt with old key
	plaintext, err := Decrypt(oldCiphertext, oldCredKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt with old key: %w", err)
	}
	defer Zero(plaintext)

	// Derive new credential key
	newTenantKey, err := DeriveTenantKey(newMasterKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive new tenant key: %w", err)
	}
	defer Zero(newTenantKey)

	newCredKey, err := DeriveCredentialKey(newTenantKey, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive new credential key: %w", err)
	}
	defer Zero(newCredKey)

	// Re-encrypt with new key
	newCiphertext, err := Encrypt(plaintext, newCredKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with new key: %w", err)
	}

	return newCiphertext, nil
}

// RotateTenantKey re-encrypts data encrypted with tenant key (not per-credential key)
// Used for TOTP secrets, recovery codes, etc.
func (kr *KeyRotator) RotateTenantKey(
	ctx context.Context,
	oldCiphertext []byte,
	tenantID uuid.UUID,
	oldMasterKey, newMasterKey []byte,
) ([]byte, error) {
	// Derive old tenant key
	oldTenantKey, err := DeriveTenantKey(oldMasterKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive old tenant key: %w", err)
	}
	defer Zero(oldTenantKey)

	// Decrypt with old key
	plaintext, err := Decrypt(oldCiphertext, oldTenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt with old key: %w", err)
	}
	defer Zero(plaintext)

	// Derive new tenant key
	newTenantKey, err := DeriveTenantKey(newMasterKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive new tenant key: %w", err)
	}
	defer Zero(newTenantKey)

	// Re-encrypt with new key
	newCiphertext, err := Encrypt(plaintext, newTenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with new key: %w", err)
	}

	return newCiphertext, nil
}

// KeyRotationCallback is called for each item during rotation
type KeyRotationCallback func(ctx context.Context, itemID uuid.UUID, oldCiphertext []byte) (newCiphertext []byte, err error)

// KeyRotationJob represents a background job for key rotation
type KeyRotationJob struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	OldKeyVersion int
	NewKeyVersion int
	StartedAt     time.Time
}

// This would be implemented as a background job that:
// 1. Lists all credentials for the tenant
// 2. For each credential:
//    a. Decrypt with old key
//    b. Re-encrypt with new key
//    c. Update in database with new key_version
//    d. Log progress
// 3. Mark rotation as complete
//
// The actual implementation would be in the job worker package
// and would use the database to track progress for fault tolerance
