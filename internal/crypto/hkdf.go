package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/google/uuid"
	"golang.org/x/crypto/hkdf"
)

// DeriveKey derives a new key using HKDF (RFC 5869).
// masterKey: the input key material (32 bytes)
// salt: context-specific salt (e.g., tenant ID, credential ID as bytes)
// info: optional context info string
// Returns: derived 32-byte key
func DeriveKey(masterKey []byte, salt []byte, info string) ([]byte, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("%w: master key", ErrInvalidKeyLength)
	}

	// Create HKDF reader with SHA-256
	reader := hkdf.New(sha256.New, masterKey, salt, []byte(info))

	// Read derived key
	derivedKey := make([]byte, KeySize)
	if _, err := io.ReadFull(reader, derivedKey); err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	return derivedKey, nil
}

// DeriveTenantKey derives a tenant-specific key from the master key.
// Key hierarchy: Master Key -> Tenant Key
func DeriveTenantKey(masterKey []byte, tenantID uuid.UUID) ([]byte, error) {
	return DeriveKey(masterKey, tenantID[:], "tenant-key")
}

// DeriveCredentialKey derives a credential-specific key from a tenant key.
// Key hierarchy: Tenant Key -> Credential Key
func DeriveCredentialKey(tenantKey []byte, credentialID uuid.UUID) ([]byte, error) {
	return DeriveKey(tenantKey, credentialID[:], "credential-key")
}

// DeriveFullCredentialKey derives a credential key directly from master key.
// This is a convenience function that performs the full key derivation chain:
// Master Key -> Tenant Key -> Credential Key
//
// The caller should zero the returned key after use.
func DeriveFullCredentialKey(km *KeyManager, tenantID, credentialID uuid.UUID) ([]byte, error) {
	// Get master key
	masterKey, err := km.GetMasterKey()
	if err != nil {
		return nil, err
	}
	defer Zero(masterKey)

	// Derive tenant key
	tenantKey, err := DeriveTenantKey(masterKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive tenant key: %w", err)
	}
	defer Zero(tenantKey)

	// Derive credential key
	credKey, err := DeriveCredentialKey(tenantKey, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive credential key: %w", err)
	}

	return credKey, nil
}

// DeriveExportKey derives a key for DSGVO data exports.
// Each export gets a unique key derived from tenant key.
func DeriveExportKey(tenantKey []byte, exportID uuid.UUID) ([]byte, error) {
	return DeriveKey(tenantKey, exportID[:], "export-key")
}

// DeriveTOTPKey derives a key for encrypting TOTP secrets.
// Key hierarchy: Tenant Key -> TOTP Key (per user)
func DeriveTOTPKey(tenantKey []byte, userID uuid.UUID) ([]byte, error) {
	return DeriveKey(tenantKey, userID[:], "totp-key")
}

// DeriveRecoveryKey derives a key for encrypting recovery codes.
// Key hierarchy: Tenant Key -> Recovery Key (per user)
func DeriveRecoveryKey(tenantKey []byte, userID uuid.UUID) ([]byte, error) {
	return DeriveKey(tenantKey, userID[:], "recovery-key")
}

// KeyDeriver provides a higher-level interface for key derivation.
// It caches the master key reference to avoid repeated lookups.
type KeyDeriver struct {
	km *KeyManager
}

// NewKeyDeriver creates a new key deriver
func NewKeyDeriver(km *KeyManager) *KeyDeriver {
	return &KeyDeriver{km: km}
}

// GetCredentialKey returns a derived credential key.
// Caller must zero the returned key after use.
func (kd *KeyDeriver) GetCredentialKey(tenantID, credentialID uuid.UUID) ([]byte, error) {
	return DeriveFullCredentialKey(kd.km, tenantID, credentialID)
}

// GetTOTPKey returns a derived TOTP encryption key.
// Caller must zero the returned key after use.
func (kd *KeyDeriver) GetTOTPKey(tenantID, userID uuid.UUID) ([]byte, error) {
	masterKey, err := kd.km.GetMasterKey()
	if err != nil {
		return nil, err
	}
	defer Zero(masterKey)

	tenantKey, err := DeriveTenantKey(masterKey, tenantID)
	if err != nil {
		return nil, err
	}
	defer Zero(tenantKey)

	return DeriveTOTPKey(tenantKey, userID)
}

// GetRecoveryKey returns a derived recovery code encryption key.
// Caller must zero the returned key after use.
func (kd *KeyDeriver) GetRecoveryKey(tenantID, userID uuid.UUID) ([]byte, error) {
	masterKey, err := kd.km.GetMasterKey()
	if err != nil {
		return nil, err
	}
	defer Zero(masterKey)

	tenantKey, err := DeriveTenantKey(masterKey, tenantID)
	if err != nil {
		return nil, err
	}
	defer Zero(tenantKey)

	return DeriveRecoveryKey(tenantKey, userID)
}
