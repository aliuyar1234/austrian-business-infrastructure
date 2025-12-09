// Package crypto provides cryptographic utilities for the security layer.
// It implements the key hierarchy: HSM/Vault -> Master Key -> Tenant Key -> Credential Key
package crypto

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
)

var (
	// ErrNoMasterKey indicates master key is not loaded
	ErrNoMasterKey = errors.New("master key not loaded")
	// ErrInvalidKeyLength indicates key has wrong length
	ErrInvalidKeyLength = errors.New("invalid key length: must be 32 bytes")
	// ErrKeyLoadFailed indicates key loading failed
	ErrKeyLoadFailed = errors.New("failed to load master key")
)

const (
	// KeySize is the size of AES-256 keys in bytes
	KeySize = 32
	// NonceSize is the size of GCM nonces in bytes
	NonceSize = 12
)

// KeyManager handles the master key and key derivation
type KeyManager struct {
	mu        sync.RWMutex
	masterKey []byte
	loaded    bool
}

// NewKeyManager creates a new key manager
func NewKeyManager() *KeyManager {
	return &KeyManager{}
}

// globalKeyManager is the singleton key manager instance
var globalKeyManager = NewKeyManager()

// GetKeyManager returns the global key manager instance
func GetKeyManager() *KeyManager {
	return globalKeyManager
}

// LoadMasterKeyFromEnv loads the master key from an environment variable.
// The key must be a 64-character hex string (32 bytes).
// Environment variable: MASTER_KEY
func (km *KeyManager) LoadMasterKeyFromEnv() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	keyHex := os.Getenv("MASTER_KEY")
	if keyHex == "" {
		return fmt.Errorf("%w: MASTER_KEY environment variable not set", ErrKeyLoadFailed)
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("%w: invalid hex encoding: %v", ErrKeyLoadFailed, err)
	}

	if len(key) != KeySize {
		return fmt.Errorf("%w: got %d bytes, expected %d", ErrInvalidKeyLength, len(key), KeySize)
	}

	km.masterKey = key
	km.loaded = true
	return nil
}

// LoadMasterKeyFromFile loads the master key from a file.
// The file must contain a 64-character hex string (32 bytes).
func (km *KeyManager) LoadMasterKeyFromFile(path string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrKeyLoadFailed, err)
	}

	// Trim whitespace and newlines
	keyHex := string(data)
	for len(keyHex) > 0 && (keyHex[len(keyHex)-1] == '\n' || keyHex[len(keyHex)-1] == '\r' || keyHex[len(keyHex)-1] == ' ') {
		keyHex = keyHex[:len(keyHex)-1]
	}

	key, err := hex.DecodeString(keyHex)
	if err != nil {
		return fmt.Errorf("%w: invalid hex encoding: %v", ErrKeyLoadFailed, err)
	}

	if len(key) != KeySize {
		return fmt.Errorf("%w: got %d bytes, expected %d", ErrInvalidKeyLength, len(key), KeySize)
	}

	km.masterKey = key
	km.loaded = true
	return nil
}

// LoadMasterKey sets the master key directly.
// Use this for testing or when key comes from Vault/HSM.
func (km *KeyManager) LoadMasterKey(key []byte) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(key) != KeySize {
		return fmt.Errorf("%w: got %d bytes, expected %d", ErrInvalidKeyLength, len(key), KeySize)
	}

	// Make a copy to prevent external modification
	km.masterKey = make([]byte, KeySize)
	copy(km.masterKey, key)
	km.loaded = true
	return nil
}

// GetMasterKey returns a copy of the master key.
// The caller is responsible for zeroing the returned key after use.
func (km *KeyManager) GetMasterKey() ([]byte, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if !km.loaded {
		return nil, ErrNoMasterKey
	}

	// Return a copy
	key := make([]byte, KeySize)
	copy(key, km.masterKey)
	return key, nil
}

// IsLoaded returns true if master key is loaded
func (km *KeyManager) IsLoaded() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.loaded
}

// Clear zeros and removes the master key from memory
func (km *KeyManager) Clear() {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.masterKey != nil {
		Zero(km.masterKey)
		km.masterKey = nil
	}
	km.loaded = false
}

// GenerateMasterKey generates a new random 32-byte master key.
// Returns the key as a hex string for storage.
// WARNING: Store this securely! Loss means all encrypted data is unrecoverable.
func GenerateMasterKey() (string, error) {
	key := make([]byte, KeySize)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate master key: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// GenerateNonce generates a random 12-byte nonce for AES-GCM
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}
