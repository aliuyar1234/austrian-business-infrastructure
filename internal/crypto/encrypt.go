package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
)

var (
	// ErrInvalidCiphertext indicates ciphertext is too short or malformed
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	// ErrDecryptionFailed indicates decryption failed (likely wrong key or tampered data)
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Encrypt encrypts plaintext using AES-256-GCM with the provided key.
// A random nonce is generated and prepended to the ciphertext.
// Returns: nonce || ciphertext || tag
//
// The key must be exactly 32 bytes (256 bits).
// The caller should zero the key after use.
func Encrypt(plaintext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := GenerateNonce()
	if err != nil {
		return nil, err
	}

	// Seal appends the ciphertext to the nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// EncryptWithNonce encrypts plaintext using a provided nonce.
// This is useful when the nonce needs to be stored separately (e.g., in a database column).
// Returns: ciphertext || tag (nonce NOT included)
//
// WARNING: Never reuse a nonce with the same key!
func EncryptWithNonce(plaintext, key, nonce []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeyLength
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce length: expected %d, got %d", NonceSize, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt.
// Expects: nonce || ciphertext || tag
//
// The key must be exactly 32 bytes (256 bits).
// The caller should zero the key and returned plaintext after use.
func Decrypt(ciphertext, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Minimum length: nonce (12) + tag (16)
	if len(ciphertext) < NonceSize+gcm.Overhead() {
		return nil, ErrInvalidCiphertext
	}

	nonce := ciphertext[:NonceSize]
	ciphertextWithTag := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextWithTag, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// DecryptWithNonce decrypts ciphertext with a separate nonce.
// Use this when the nonce was stored separately (e.g., from EncryptWithNonce).
// Expects: ciphertext || tag (nonce provided separately)
//
// The caller should zero the key and returned plaintext after use.
func DecryptWithNonce(ciphertext, key, nonce []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeyLength
	}
	if len(nonce) != NonceSize {
		return nil, fmt.Errorf("invalid nonce length: expected %d, got %d", NonceSize, len(nonce))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Minimum length: tag (16)
	if len(ciphertext) < gcm.Overhead() {
		return nil, ErrInvalidCiphertext
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// CredentialEncryptor provides a high-level interface for encrypting credentials.
// It handles the full key derivation chain and proper cleanup.
type CredentialEncryptor struct {
	deriver *KeyDeriver
}

// NewCredentialEncryptor creates a new credential encryptor
func NewCredentialEncryptor(km *KeyManager) *CredentialEncryptor {
	return &CredentialEncryptor{
		deriver: NewKeyDeriver(km),
	}
}

// EncryptCredential encrypts a credential (e.g., PIN) using the full key hierarchy.
// Returns the encrypted data and nonce for storage.
//
// The plaintext is zeroed after encryption.
func (ce *CredentialEncryptor) EncryptCredential(plaintext []byte, tenantID, credentialID [16]byte) (ciphertext, nonce []byte, err error) {
	// Get derived key
	key, err := ce.deriver.GetCredentialKey(tenantID, credentialID)
	if err != nil {
		return nil, nil, err
	}
	defer Zero(key)

	// Generate nonce
	nonce, err = GenerateNonce()
	if err != nil {
		return nil, nil, err
	}

	// Encrypt
	ciphertext, err = EncryptWithNonce(plaintext, key, nonce)
	if err != nil {
		return nil, nil, err
	}

	// Zero plaintext
	Zero(plaintext)

	return ciphertext, nonce, nil
}

// DecryptCredential decrypts a credential using the full key hierarchy.
// The caller MUST zero the returned plaintext after use.
func (ce *CredentialEncryptor) DecryptCredential(ciphertext, nonce []byte, tenantID, credentialID [16]byte) ([]byte, error) {
	// Get derived key
	key, err := ce.deriver.GetCredentialKey(tenantID, credentialID)
	if err != nil {
		return nil, err
	}
	defer Zero(key)

	// Decrypt
	return DecryptWithNonce(ciphertext, key, nonce)
}
