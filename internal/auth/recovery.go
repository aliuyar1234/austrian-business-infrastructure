package auth

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"austrian-business-infrastructure/internal/crypto"
)

var (
	// ErrRecoveryCodeInvalid indicates the recovery code is invalid or already used
	ErrRecoveryCodeInvalid = errors.New("invalid or already used recovery code")
	// ErrNoRecoveryCodes indicates no recovery codes are configured
	ErrNoRecoveryCodes = errors.New("no recovery codes configured")
	// ErrAllRecoveryCodesUsed indicates all recovery codes have been used
	ErrAllRecoveryCodesUsed = errors.New("all recovery codes have been used")
)

const (
	// RecoveryCodeCount is the number of recovery codes to generate
	RecoveryCodeCount = 10
	// RecoveryCodeLength is the length of each recovery code in bytes (before encoding)
	RecoveryCodeLength = 10
)

// RecoveryCodes represents the stored recovery codes
type RecoveryCodes struct {
	Codes []RecoveryCode `json:"codes"`
}

// RecoveryCode represents a single recovery code
type RecoveryCode struct {
	Code string `json:"code"`
	Used bool   `json:"used"`
}

// RecoveryCodeManager handles recovery code operations
type RecoveryCodeManager struct{}

// NewRecoveryCodeManager creates a new recovery code manager
func NewRecoveryCodeManager() *RecoveryCodeManager {
	return &RecoveryCodeManager{}
}

// GenerateCodes generates 10 recovery codes
// Returns the plain text codes (display to user) and the encrypted codes (store in DB)
func (m *RecoveryCodeManager) GenerateCodes(tenantKey []byte) (plainCodes []string, encryptedData []byte, err error) {
	codes := make([]RecoveryCode, RecoveryCodeCount)
	plainCodes = make([]string, RecoveryCodeCount)

	for i := 0; i < RecoveryCodeCount; i++ {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate recovery code: %w", err)
		}
		codes[i] = RecoveryCode{
			Code: code,
			Used: false,
		}
		plainCodes[i] = code
	}

	// Serialize to JSON
	rc := RecoveryCodes{Codes: codes}
	jsonData, err := json.Marshal(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize recovery codes: %w", err)
	}

	// Encrypt
	encrypted, err := crypto.Encrypt(jsonData, tenantKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encrypt recovery codes: %w", err)
	}

	return plainCodes, encrypted, nil
}

// ValidateCode validates a recovery code and marks it as used if valid
// Returns updated encrypted data and error
func (m *RecoveryCodeManager) ValidateCode(encryptedData []byte, code string, tenantKey []byte) (updatedEncrypted []byte, err error) {
	if len(encryptedData) == 0 {
		return nil, ErrNoRecoveryCodes
	}

	// Decrypt
	jsonData, err := crypto.Decrypt(encryptedData, tenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt recovery codes: %w", err)
	}
	defer crypto.Zero(jsonData)

	// Parse
	var rc RecoveryCodes
	if err := json.Unmarshal(jsonData, &rc); err != nil {
		return nil, fmt.Errorf("failed to parse recovery codes: %w", err)
	}

	// Find and validate the code
	found := false
	for i := range rc.Codes {
		if rc.Codes[i].Code == code {
			if rc.Codes[i].Used {
				return nil, ErrRecoveryCodeInvalid
			}
			rc.Codes[i].Used = true
			found = true
			break
		}
	}

	if !found {
		return nil, ErrRecoveryCodeInvalid
	}

	// Re-serialize
	newJsonData, err := json.Marshal(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to re-serialize recovery codes: %w", err)
	}

	// Re-encrypt
	encrypted, err := crypto.Encrypt(newJsonData, tenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed to re-encrypt recovery codes: %w", err)
	}

	return encrypted, nil
}

// GetRemainingCount returns the number of unused recovery codes
func (m *RecoveryCodeManager) GetRemainingCount(encryptedData []byte, tenantKey []byte) (int, error) {
	if len(encryptedData) == 0 {
		return 0, nil
	}

	// Decrypt
	jsonData, err := crypto.Decrypt(encryptedData, tenantKey)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt recovery codes: %w", err)
	}
	defer crypto.Zero(jsonData)

	// Parse
	var rc RecoveryCodes
	if err := json.Unmarshal(jsonData, &rc); err != nil {
		return 0, fmt.Errorf("failed to parse recovery codes: %w", err)
	}

	// Count unused
	count := 0
	for _, c := range rc.Codes {
		if !c.Used {
			count++
		}
	}

	return count, nil
}

// generateRecoveryCode generates a single recovery code
// Format: XXXX-XXXX-XXXX (alphanumeric, easily readable)
func generateRecoveryCode() (string, error) {
	const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ" // Exclude 0, 1, I, O for readability
	bytes := make([]byte, RecoveryCodeLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	code := make([]byte, 14) // XXXX-XXXX-XXXX = 14 chars
	codeIdx := 0
	for i := 0; i < RecoveryCodeLength; i++ {
		if i == 4 || i == 8 {
			code[codeIdx] = '-'
			codeIdx++
		}
		if i < RecoveryCodeLength {
			code[codeIdx] = alphabet[int(bytes[i])%len(alphabet)]
			codeIdx++
		}
	}

	return string(code), nil
}

// CountUsedCodes returns the number of used recovery codes
func (m *RecoveryCodeManager) CountUsedCodes(encryptedData []byte, tenantKey []byte) (int, error) {
	if len(encryptedData) == 0 {
		return 0, nil
	}

	// Decrypt
	jsonData, err := crypto.Decrypt(encryptedData, tenantKey)
	if err != nil {
		return 0, fmt.Errorf("failed to decrypt recovery codes: %w", err)
	}
	defer crypto.Zero(jsonData)

	// Parse
	var rc RecoveryCodes
	if err := json.Unmarshal(jsonData, &rc); err != nil {
		return 0, fmt.Errorf("failed to parse recovery codes: %w", err)
	}

	// Count used
	count := 0
	for _, c := range rc.Codes {
		if c.Used {
			count++
		}
	}

	return count, nil
}
