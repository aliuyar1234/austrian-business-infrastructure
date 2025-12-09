package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sync"
)

var (
	// ErrNoPrivateKey indicates the private key is not loaded
	ErrNoPrivateKey = errors.New("ECDSA private key not loaded")
	// ErrInvalidKeyFormat indicates the key file format is invalid
	ErrInvalidKeyFormat = errors.New("invalid key format")
	// ErrKeyGenFailed indicates key generation failed
	ErrKeyGenFailed = errors.New("failed to generate ECDSA key pair")
)

// ECDSAKeyManager manages ECDSA P-256 keys for ES256 JWT signing.
// The private key is used for signing, the public key for verification.
type ECDSAKeyManager struct {
	mu         sync.RWMutex
	privateKey *ecdsa.PrivateKey
	loaded     bool
}

// NewECDSAKeyManager creates a new ECDSA key manager
func NewECDSAKeyManager() *ECDSAKeyManager {
	return &ECDSAKeyManager{}
}

// globalECDSAKeyManager is the singleton ECDSA key manager
var globalECDSAKeyManager = NewECDSAKeyManager()

// GetECDSAKeyManager returns the global ECDSA key manager
func GetECDSAKeyManager() *ECDSAKeyManager {
	return globalECDSAKeyManager
}

// LoadFromEnv loads the ECDSA private key from environment variable.
// The key should be in PEM format (ECDSA PRIVATE KEY).
// Environment variable: JWT_ECDSA_PRIVATE_KEY
func (km *ECDSAKeyManager) LoadFromEnv() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	keyPEM := os.Getenv("JWT_ECDSA_PRIVATE_KEY")
	if keyPEM == "" {
		return fmt.Errorf("%w: JWT_ECDSA_PRIVATE_KEY environment variable not set", ErrNoPrivateKey)
	}

	return km.loadFromPEM([]byte(keyPEM))
}

// LoadFromFile loads the ECDSA private key from a PEM file.
func (km *ECDSAKeyManager) LoadFromFile(path string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	return km.loadFromPEM(data)
}

// loadFromPEM parses a PEM-encoded ECDSA private key
func (km *ECDSAKeyManager) loadFromPEM(pemData []byte) error {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return fmt.Errorf("%w: no PEM block found", ErrInvalidKeyFormat)
	}

	var privateKey *ecdsa.PrivateKey
	var err error

	switch block.Type {
	case "EC PRIVATE KEY":
		privateKey, err = x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return fmt.Errorf("%w: %v", ErrInvalidKeyFormat, parseErr)
		}
		var ok bool
		privateKey, ok = key.(*ecdsa.PrivateKey)
		if !ok {
			return fmt.Errorf("%w: not an ECDSA key", ErrInvalidKeyFormat)
		}
	default:
		return fmt.Errorf("%w: unexpected PEM type: %s", ErrInvalidKeyFormat, block.Type)
	}

	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidKeyFormat, err)
	}

	// Verify it's a P-256 curve (required for ES256)
	if privateKey.Curve != elliptic.P256() {
		return fmt.Errorf("%w: key must use P-256 curve for ES256", ErrInvalidKeyFormat)
	}

	km.privateKey = privateKey
	km.loaded = true
	return nil
}

// LoadKey loads an existing ECDSA private key directly.
// Use this for testing or when key comes from external source.
func (km *ECDSAKeyManager) LoadKey(key *ecdsa.PrivateKey) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if key == nil {
		return ErrNoPrivateKey
	}

	if key.Curve != elliptic.P256() {
		return fmt.Errorf("%w: key must use P-256 curve for ES256", ErrInvalidKeyFormat)
	}

	km.privateKey = key
	km.loaded = true
	return nil
}

// GetPrivateKey returns the ECDSA private key for signing.
// Returns an error if no key is loaded.
func (km *ECDSAKeyManager) GetPrivateKey() (*ecdsa.PrivateKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if !km.loaded {
		return nil, ErrNoPrivateKey
	}

	return km.privateKey, nil
}

// GetPublicKey returns the ECDSA public key for verification.
// Returns an error if no key is loaded.
func (km *ECDSAKeyManager) GetPublicKey() (*ecdsa.PublicKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if !km.loaded {
		return nil, ErrNoPrivateKey
	}

	return &km.privateKey.PublicKey, nil
}

// IsLoaded returns true if a key is loaded
func (km *ECDSAKeyManager) IsLoaded() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.loaded
}

// Clear removes the key from memory
func (km *ECDSAKeyManager) Clear() {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.privateKey = nil
	km.loaded = false
}

// GenerateKey generates a new ECDSA P-256 key pair.
// Returns the private key in PEM format for storage.
// WARNING: Store the private key securely!
func GenerateKey() (privatePEM, publicPEM []byte, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrKeyGenFailed, err)
	}

	// Encode private key to PEM
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	privatePEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return privatePEM, publicPEM, nil
}

// GenerateKeyToFiles generates a new ECDSA P-256 key pair and saves to files.
func GenerateKeyToFiles(privateKeyPath, publicKeyPath string) error {
	privatePEM, publicPEM, err := GenerateKey()
	if err != nil {
		return err
	}

	if err := os.WriteFile(privateKeyPath, privatePEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	if err := os.WriteFile(publicKeyPath, publicPEM, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// MustLoadKeys loads ECDSA keys from environment or file, failing fast if unavailable.
// This should be called at server startup to ensure auth will work.
// Priority: 1) JWT_ECDSA_PRIVATE_KEY env var, 2) JWT_ECDSA_KEY_FILE path, 3) Generate new (dev only)
func MustLoadKeys(allowGenerate bool) error {
	km := GetECDSAKeyManager()

	// Try loading from environment first
	if err := km.LoadFromEnv(); err == nil {
		return nil
	}

	// Try loading from file path in environment
	keyFile := os.Getenv("JWT_ECDSA_KEY_FILE")
	if keyFile != "" {
		if err := km.LoadFromFile(keyFile); err != nil {
			return fmt.Errorf("failed to load JWT key from file %s: %w", keyFile, err)
		}
		return nil
	}

	// In development, optionally generate a new key
	if allowGenerate {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return fmt.Errorf("failed to generate development JWT key: %w", err)
		}
		if err := km.LoadKey(privateKey); err != nil {
			return fmt.Errorf("failed to load generated key: %w", err)
		}
		return nil
	}

	return fmt.Errorf("no JWT ECDSA key configured: set JWT_ECDSA_PRIVATE_KEY env var or JWT_ECDSA_KEY_FILE path")
}
