package auth

import (
	"bytes"
	"encoding/base32"
	"errors"
	"fmt"
	"image/png"
	"time"

	"austrian-business-infrastructure/internal/crypto"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	// ErrTOTPSecretGeneration indicates TOTP secret generation failed
	ErrTOTPSecretGeneration = errors.New("failed to generate TOTP secret")
	// ErrTOTPInvalidCode indicates the TOTP code is invalid
	ErrTOTPInvalidCode = errors.New("invalid TOTP code")
	// ErrTOTPNotEnabled indicates 2FA is not enabled for the user
	ErrTOTPNotEnabled = errors.New("2FA is not enabled for this user")
	// ErrTOTPAlreadyEnabled indicates 2FA is already enabled
	ErrTOTPAlreadyEnabled = errors.New("2FA is already enabled for this user")
)

const (
	// TOTPIssuer is the issuer name shown in authenticator apps
	TOTPIssuer = "Austrian Business Platform"
	// TOTPSecretSize is the size of TOTP secrets in bytes
	TOTPSecretSize = 20
	// TOTPDigits is the number of digits in TOTP codes
	TOTPDigits = 6
	// TOTPPeriod is the period in seconds for TOTP codes
	TOTPPeriod = 30
)

// TOTPManager handles TOTP operations with encryption
type TOTPManager struct {
	keyManager *crypto.KeyManager
}

// NewTOTPManager creates a new TOTP manager
func NewTOTPManager(keyManager *crypto.KeyManager) *TOTPManager {
	return &TOTPManager{
		keyManager: keyManager,
	}
}

// TOTPSetupInfo contains the information needed to set up TOTP
type TOTPSetupInfo struct {
	Secret   []byte // Plain text secret (encrypt before storing)
	OTPURL   string // otpauth:// URL for QR code
	QRCode   []byte // PNG image of QR code
	Issuer   string
	Account  string
}

// GenerateSecret generates a new TOTP secret for a user
func (m *TOTPManager) GenerateSecret(email string) (*TOTPSetupInfo, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      TOTPIssuer,
		AccountName: email,
		Period:      TOTPPeriod,
		SecretSize:  TOTPSecretSize,
		Digits:      otp.Digits(TOTPDigits),
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTOTPSecretGeneration, err)
	}

	// Generate QR code as PNG
	var qrBuf bytes.Buffer
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}
	if err := png.Encode(&qrBuf, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	// Decode the base32 secret to raw bytes for encryption
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(key.Secret())
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret: %w", err)
	}

	return &TOTPSetupInfo{
		Secret:  secret,
		OTPURL:  key.URL(),
		QRCode:  qrBuf.Bytes(),
		Issuer:  TOTPIssuer,
		Account: email,
	}, nil
}

// VerifyCode verifies a TOTP code against an encrypted secret
// Allows 1 step tolerance (previous and next 30-second window)
func (m *TOTPManager) VerifyCode(encryptedSecret []byte, code string, tenantKey []byte) (bool, error) {
	// Decrypt the secret
	secret, err := crypto.Decrypt(encryptedSecret, tenantKey)
	if err != nil {
		return false, fmt.Errorf("failed to decrypt TOTP secret: %w", err)
	}
	defer crypto.Zero(secret)

	// Re-encode to base32 for verification
	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	// Validate with 1 step tolerance (skew = 1)
	valid, err := totp.ValidateCustom(code, secretBase32, time.Now(), totp.ValidateOpts{
		Period:    TOTPPeriod,
		Skew:      1, // Allow 1 step tolerance per FR-108
		Digits:    otp.Digits(TOTPDigits),
		Algorithm: otp.AlgorithmSHA1,
	})

	if err != nil {
		return false, fmt.Errorf("TOTP validation error: %w", err)
	}

	return valid, nil
}

// VerifyCodePlainSecret verifies a TOTP code against a plain secret (for setup verification)
func (m *TOTPManager) VerifyCodePlainSecret(secret []byte, code string) bool {
	// Encode to base32 for verification
	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)

	valid, _ := totp.ValidateCustom(code, secretBase32, time.Now(), totp.ValidateOpts{
		Period:    TOTPPeriod,
		Skew:      1,
		Digits:    otp.Digits(TOTPDigits),
		Algorithm: otp.AlgorithmSHA1,
	})

	return valid
}

// EncryptSecret encrypts a TOTP secret for storage
func (m *TOTPManager) EncryptSecret(secret, tenantKey []byte) ([]byte, error) {
	encrypted, err := crypto.Encrypt(secret, tenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt TOTP secret: %w", err)
	}
	return encrypted, nil
}

// DecryptSecret decrypts a stored TOTP secret
func (m *TOTPManager) DecryptSecret(encryptedSecret, tenantKey []byte) ([]byte, error) {
	secret, err := crypto.Decrypt(encryptedSecret, tenantKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt TOTP secret: %w", err)
	}
	return secret, nil
}
