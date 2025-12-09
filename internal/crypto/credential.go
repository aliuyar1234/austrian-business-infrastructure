package crypto

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CredentialEncryption provides high-level credential encryption using the key hierarchy.
// This implements FR-101: AES-256-GCM encryption with HKDF-derived keys.
//
// Key Hierarchy:
//   Master Key (from env/vault)
//   └── Tenant Key (HKDF-derived)
//       └── Credential Key (HKDF-derived)
//
// Each credential has a unique derived key using:
// - Master key (32 bytes)
// - Tenant ID (16 bytes)
// - Credential ID (16 bytes) - typically the account ID
type CredentialEncryption struct {
	keyManager *KeyManager
}

// NewCredentialEncryption creates a new credential encryption helper
func NewCredentialEncryption(keyManager *KeyManager) *CredentialEncryption {
	if keyManager == nil {
		keyManager = GetKeyManager()
	}
	return &CredentialEncryption{keyManager: keyManager}
}

// EncryptedCredential holds an encrypted credential with metadata
type EncryptedCredential struct {
	Ciphertext []byte // Encrypted PIN/password
	KeyVersion int    // Key rotation version (for future rotation)
}

// EncryptPIN encrypts a PIN/password using the full key hierarchy
// The PIN is encrypted with a key derived specifically for this credential
func (ce *CredentialEncryption) EncryptPIN(ctx context.Context, pin string, tenantID, credentialID uuid.UUID) (*EncryptedCredential, error) {
	// Get the credential-specific key
	credKey, err := ce.getCredentialKey(tenantID, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive credential key: %w", err)
	}
	defer Zero(credKey)

	// Encrypt the PIN
	plaintext := []byte(pin)
	defer Zero(plaintext)

	ciphertext, err := Encrypt(plaintext, credKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt PIN: %w", err)
	}

	return &EncryptedCredential{
		Ciphertext: ciphertext,
		KeyVersion: 1, // Current key version
	}, nil
}

// DecryptPIN decrypts a PIN/password and returns it for immediate use
// IMPORTANT: The caller MUST zero the returned slice immediately after use
// using crypto.Zero() or crypto.WithSecureBytes()
func (ce *CredentialEncryption) DecryptPIN(ctx context.Context, encrypted *EncryptedCredential, tenantID, credentialID uuid.UUID) ([]byte, error) {
	if encrypted == nil || len(encrypted.Ciphertext) == 0 {
		return nil, fmt.Errorf("no encrypted credential provided")
	}

	// Get the credential-specific key
	credKey, err := ce.getCredentialKey(tenantID, credentialID)
	if err != nil {
		return nil, fmt.Errorf("failed to derive credential key: %w", err)
	}
	defer Zero(credKey)

	// Decrypt the PIN
	plaintext, err := Decrypt(encrypted.Ciphertext, credKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt PIN: %w", err)
	}

	return plaintext, nil
}

// DecryptPINWithCallback decrypts a PIN and passes it to a callback, then zeros it
// This is the RECOMMENDED way to use decrypted credentials as it ensures
// the sensitive data is zeroed immediately after use.
//
// Example:
//
//	err := credEnc.DecryptPINWithCallback(ctx, encrypted, tenantID, credID, func(pin []byte) error {
//	    return foClient.Login(teilnehmerID, benutzerID, string(pin))
//	})
func (ce *CredentialEncryption) DecryptPINWithCallback(
	ctx context.Context,
	encrypted *EncryptedCredential,
	tenantID, credentialID uuid.UUID,
	fn func(pin []byte) error,
) error {
	pin, err := ce.DecryptPIN(ctx, encrypted, tenantID, credentialID)
	if err != nil {
		return err
	}
	defer Zero(pin)

	return fn(pin)
}

// getCredentialKey derives the full credential key using the key hierarchy
func (ce *CredentialEncryption) getCredentialKey(tenantID, credentialID uuid.UUID) ([]byte, error) {
	return DeriveFullCredentialKey(ce.keyManager, tenantID, credentialID)
}

// EncryptWithTenantKey encrypts data using only the tenant key (for non-credential data)
func (ce *CredentialEncryption) EncryptWithTenantKey(ctx context.Context, plaintext []byte, tenantID uuid.UUID) ([]byte, error) {
	masterKey, err := ce.keyManager.GetMasterKey()
	if err != nil {
		return nil, err
	}

	tenantKey, err := DeriveTenantKey(masterKey, tenantID)
	if err != nil {
		return nil, err
	}
	defer Zero(tenantKey)

	return Encrypt(plaintext, tenantKey)
}

// DecryptWithTenantKey decrypts data using the tenant key
func (ce *CredentialEncryption) DecryptWithTenantKey(ctx context.Context, ciphertext []byte, tenantID uuid.UUID) ([]byte, error) {
	masterKey, err := ce.keyManager.GetMasterKey()
	if err != nil {
		return nil, err
	}

	tenantKey, err := DeriveTenantKey(masterKey, tenantID)
	if err != nil {
		return nil, err
	}
	defer Zero(tenantKey)

	return Decrypt(ciphertext, tenantKey)
}

// Global credential encryption instance
var globalCredEnc *CredentialEncryption

// GetCredentialEncryption returns the global credential encryption instance
func GetCredentialEncryption() *CredentialEncryption {
	if globalCredEnc == nil {
		globalCredEnc = NewCredentialEncryption(nil)
	}
	return globalCredEnc
}
