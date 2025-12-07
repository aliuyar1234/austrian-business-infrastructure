package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"
)

const (
	// Argon2 parameters (recommended for key derivation)
	argon2Time    = 1
	argon2Memory  = 64 * 1024 // 64 MB
	argon2Threads = 4
	argon2KeyLen  = 32 // 256 bits for AES-256

	// Salt and nonce sizes
	saltSize  = 16
	nonceSize = 12 // GCM standard nonce size
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: invalid master password or corrupted data")
	ErrDataTooShort     = errors.New("encrypted data too short")
)

// DeriveKey derives a 256-bit key from a password using Argon2id
func DeriveKey(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argon2Time,
		argon2Memory,
		argon2Threads,
		argon2KeyLen,
	)
}

// Encrypt encrypts plaintext using AES-256-GCM with a key derived from the master password.
// Returns: salt (16 bytes) || nonce (12 bytes) || ciphertext with GCM tag
func Encrypt(plaintext []byte, masterPassword string) ([]byte, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from password
	key := DeriveKey(masterPassword, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and append GCM tag
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Combine: salt || nonce || ciphertext
	result := make([]byte, 0, saltSize+nonceSize+len(ciphertext))
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)

	return result, nil
}

// Decrypt decrypts data that was encrypted with Encrypt.
// Input format: salt (16 bytes) || nonce (12 bytes) || ciphertext with GCM tag
func Decrypt(data []byte, masterPassword string) ([]byte, error) {
	minLen := saltSize + nonceSize + 16 // minimum: salt + nonce + GCM tag
	if len(data) < minLen {
		return nil, ErrDataTooShort
	}

	// Extract salt, nonce, and ciphertext
	salt := data[:saltSize]
	nonce := data[saltSize : saltSize+nonceSize]
	ciphertext := data[saltSize+nonceSize:]

	// Derive key from password
	key := DeriveKey(masterPassword, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt and verify GCM tag
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
