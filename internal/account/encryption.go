package account

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidKey       = errors.New("encryption key must be 32 bytes")
	ErrInvalidIV        = errors.New("invalid initialization vector")
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Encryptor handles AES-256-GCM encryption/decryption
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor with the given key
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return &Encryptor{key: key}, nil
}

// Encrypt encrypts data using AES-256-GCM and returns ciphertext and IV
func (e *Encryptor) Encrypt(plaintext []byte) (ciphertext, iv []byte, err error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("create GCM: %w", err)
	}

	// Generate random nonce/IV
	iv = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, fmt.Errorf("generate IV: %w", err)
	}

	// Encrypt and authenticate
	ciphertext = gcm.Seal(nil, iv, plaintext, nil)

	return ciphertext, iv, nil
}

// Decrypt decrypts data using AES-256-GCM
func (e *Encryptor) Decrypt(ciphertext, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	if len(iv) != gcm.NonceSize() {
		return nil, ErrInvalidIV
	}

	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptJSON encrypts a struct as JSON
func (e *Encryptor) EncryptJSON(v interface{}) (ciphertext, iv []byte, err error) {
	plaintext, err := json.Marshal(v)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal JSON: %w", err)
	}
	return e.Encrypt(plaintext)
}

// DecryptJSON decrypts ciphertext and unmarshals to a struct
func (e *Encryptor) DecryptJSON(ciphertext, iv []byte, v interface{}) error {
	plaintext, err := e.Decrypt(ciphertext, iv)
	if err != nil {
		return err
	}
	return json.Unmarshal(plaintext, v)
}

// RotateKey re-encrypts data with a new key
func RotateKey(oldEncryptor, newEncryptor *Encryptor, ciphertext, iv []byte) (newCiphertext, newIV []byte, err error) {
	// Decrypt with old key
	plaintext, err := oldEncryptor.Decrypt(ciphertext, iv)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt with old key: %w", err)
	}

	// Encrypt with new key
	newCiphertext, newIV, err = newEncryptor.Encrypt(plaintext)
	if err != nil {
		return nil, nil, fmt.Errorf("encrypt with new key: %w", err)
	}

	return newCiphertext, newIV, nil
}

// GenerateKey generates a random 256-bit key
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return key, nil
}

// KeyRotator handles encryption key rotation for accounts
type KeyRotator struct {
	oldEncryptor *Encryptor
	newEncryptor *Encryptor
}

// NewKeyRotator creates a new key rotator
func NewKeyRotator(oldKey, newKey []byte) (*KeyRotator, error) {
	oldEnc, err := NewEncryptor(oldKey)
	if err != nil {
		return nil, fmt.Errorf("create old encryptor: %w", err)
	}

	newEnc, err := NewEncryptor(newKey)
	if err != nil {
		return nil, fmt.Errorf("create new encryptor: %w", err)
	}

	return &KeyRotator{
		oldEncryptor: oldEnc,
		newEncryptor: newEnc,
	}, nil
}

// RotateCredentials re-encrypts credentials with the new key
func (kr *KeyRotator) RotateCredentials(ciphertext, iv []byte) (newCiphertext, newIV []byte, err error) {
	return RotateKey(kr.oldEncryptor, kr.newEncryptor, ciphertext, iv)
}

// EncryptedData represents encrypted data that can be rotated
type EncryptedData struct {
	Ciphertext []byte
	IV         []byte
}

// BatchRotator handles batch key rotation operations
type BatchRotator struct {
	rotator   *KeyRotator
	batchSize int
}

// NewBatchRotator creates a new batch rotator
func NewBatchRotator(oldKey, newKey []byte, batchSize int) (*BatchRotator, error) {
	rotator, err := NewKeyRotator(oldKey, newKey)
	if err != nil {
		return nil, err
	}

	if batchSize <= 0 {
		batchSize = 100
	}

	return &BatchRotator{
		rotator:   rotator,
		batchSize: batchSize,
	}, nil
}

// RotateBatch rotates a batch of encrypted data
func (br *BatchRotator) RotateBatch(data []EncryptedData) ([]EncryptedData, error) {
	results := make([]EncryptedData, len(data))

	for i, d := range data {
		newCiphertext, newIV, err := br.rotator.RotateCredentials(d.Ciphertext, d.IV)
		if err != nil {
			return nil, fmt.Errorf("rotate item %d: %w", i, err)
		}
		results[i] = EncryptedData{
			Ciphertext: newCiphertext,
			IV:         newIV,
		}
	}

	return results, nil
}
