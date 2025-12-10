package unit

import (
	"encoding/base32"
	"testing"
	"time"

	"austrian-business-infrastructure/internal/auth"
	"austrian-business-infrastructure/internal/crypto"
	"github.com/pquerna/otp/totp"
)

// TestTOTP_GenerateSecret tests TOTP secret generation
func TestTOTP_GenerateSecret(t *testing.T) {
	km := crypto.NewKeyManager()
	// Load a test master key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, err := totpManager.GenerateSecret("test@example.com")
	if err != nil {
		t.Fatalf("failed to generate TOTP secret: %v", err)
	}

	// Verify secret is correct size (20 bytes per TOTPSecretSize)
	if len(setupInfo.Secret) != 20 {
		t.Errorf("expected secret size 20, got %d", len(setupInfo.Secret))
	}

	// Verify OTP URL is generated
	if setupInfo.OTPURL == "" {
		t.Error("OTP URL should not be empty")
	}

	// Verify QR code is generated
	if len(setupInfo.QRCode) == 0 {
		t.Error("QR code should not be empty")
	}

	// Verify issuer is correct
	if setupInfo.Issuer != auth.TOTPIssuer {
		t.Errorf("expected issuer '%s', got '%s'", auth.TOTPIssuer, setupInfo.Issuer)
	}

	// Verify account is set
	if setupInfo.Account != "test@example.com" {
		t.Errorf("expected account 'test@example.com', got '%s'", setupInfo.Account)
	}
}

// TestTOTP_VerifyCodePlainSecret tests TOTP verification with plain secret
func TestTOTP_VerifyCodePlainSecret(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, err := totpManager.GenerateSecret("test@example.com")
	if err != nil {
		t.Fatalf("failed to generate TOTP secret: %v", err)
	}

	// Generate a valid TOTP code using the same secret
	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(setupInfo.Secret)
	validCode, err := totp.GenerateCode(secretBase32, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}

	// Verify with valid code
	if !totpManager.VerifyCodePlainSecret(setupInfo.Secret, validCode) {
		t.Error("valid TOTP code should verify successfully")
	}

	// Verify with invalid code
	if totpManager.VerifyCodePlainSecret(setupInfo.Secret, "000000") {
		t.Error("invalid TOTP code should not verify")
	}

	// Verify with wrong format code
	if totpManager.VerifyCodePlainSecret(setupInfo.Secret, "12345") {
		t.Error("wrong format TOTP code should not verify")
	}
}

// TestTOTP_VerifyCodeWithEncryption tests TOTP verification with encrypted secret
func TestTOTP_VerifyCodeWithEncryption(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, err := totpManager.GenerateSecret("test@example.com")
	if err != nil {
		t.Fatalf("failed to generate TOTP secret: %v", err)
	}

	// Encrypt the secret
	tenantKey := make([]byte, 32)
	for i := range tenantKey {
		tenantKey[i] = byte(i + 100)
	}

	encryptedSecret, err := totpManager.EncryptSecret(setupInfo.Secret, tenantKey)
	if err != nil {
		t.Fatalf("failed to encrypt TOTP secret: %v", err)
	}

	// Generate a valid TOTP code
	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(setupInfo.Secret)
	validCode, err := totp.GenerateCode(secretBase32, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}

	// Verify with encrypted secret and valid code
	valid, err := totpManager.VerifyCode(encryptedSecret, validCode, tenantKey)
	if err != nil {
		t.Fatalf("verification failed with error: %v", err)
	}
	if !valid {
		t.Error("valid TOTP code should verify successfully with encrypted secret")
	}

	// Verify with wrong code
	valid, err = totpManager.VerifyCode(encryptedSecret, "000000", tenantKey)
	if err != nil {
		t.Fatalf("verification failed with error: %v", err)
	}
	if valid {
		t.Error("invalid TOTP code should not verify")
	}
}

// TestTOTP_VerifyCodeWrongKey tests rejection with wrong decryption key
func TestTOTP_VerifyCodeWrongKey(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, _ := totpManager.GenerateSecret("test@example.com")

	// Encrypt with one key
	tenantKey := make([]byte, 32)
	for i := range tenantKey {
		tenantKey[i] = byte(i)
	}
	encryptedSecret, _ := totpManager.EncryptSecret(setupInfo.Secret, tenantKey)

	// Generate valid code
	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(setupInfo.Secret)
	validCode, _ := totp.GenerateCode(secretBase32, time.Now())

	// Try to verify with wrong key
	wrongKey := make([]byte, 32)
	for i := range wrongKey {
		wrongKey[i] = byte(i + 50)
	}

	_, err := totpManager.VerifyCode(encryptedSecret, validCode, wrongKey)
	if err == nil {
		t.Error("expected error when verifying with wrong key")
	}
}

// TestTOTP_EncryptDecrypt tests TOTP secret encryption and decryption
func TestTOTP_EncryptDecrypt(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, _ := totpManager.GenerateSecret("test@example.com")

	tenantKey := make([]byte, 32)
	for i := range tenantKey {
		tenantKey[i] = byte(i)
	}

	// Encrypt
	encrypted, err := totpManager.EncryptSecret(setupInfo.Secret, tenantKey)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	// Encrypted should be different from original
	if string(encrypted) == string(setupInfo.Secret) {
		t.Error("encrypted secret should differ from original")
	}

	// Decrypt
	decrypted, err := totpManager.DecryptSecret(encrypted, tenantKey)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	// Decrypted should match original
	if len(decrypted) != len(setupInfo.Secret) {
		t.Errorf("decrypted length mismatch: got %d, expected %d", len(decrypted), len(setupInfo.Secret))
	}
	for i := range decrypted {
		if decrypted[i] != setupInfo.Secret[i] {
			t.Errorf("decrypted byte %d mismatch: got %d, expected %d", i, decrypted[i], setupInfo.Secret[i])
		}
	}
}

// TestTOTP_TimeWindow tests TOTP time window tolerance (skew)
func TestTOTP_TimeWindow(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, _ := totpManager.GenerateSecret("test@example.com")

	secretBase32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(setupInfo.Secret)

	// Test current time
	currentCode, _ := totp.GenerateCode(secretBase32, time.Now())
	if !totpManager.VerifyCodePlainSecret(setupInfo.Secret, currentCode) {
		t.Error("current time code should verify")
	}

	// Test previous time window (30 seconds ago) - should work with skew=1
	previousCode, _ := totp.GenerateCode(secretBase32, time.Now().Add(-30*time.Second))
	if !totpManager.VerifyCodePlainSecret(setupInfo.Secret, previousCode) {
		t.Error("previous window code should verify with skew tolerance")
	}

	// Test next time window (30 seconds ahead) - should work with skew=1
	nextCode, _ := totp.GenerateCode(secretBase32, time.Now().Add(30*time.Second))
	if !totpManager.VerifyCodePlainSecret(setupInfo.Secret, nextCode) {
		t.Error("next window code should verify with skew tolerance")
	}
}

// TestTOTP_QRCodeFormat tests that QR code is valid PNG
func TestTOTP_QRCodeFormat(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, _ := totpManager.GenerateSecret("test@example.com")

	// PNG files start with these magic bytes
	pngMagic := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

	if len(setupInfo.QRCode) < len(pngMagic) {
		t.Fatal("QR code too short to be valid PNG")
	}

	for i, b := range pngMagic {
		if setupInfo.QRCode[i] != b {
			t.Errorf("QR code PNG magic byte %d mismatch: got %02x, expected %02x", i, setupInfo.QRCode[i], b)
		}
	}
}

// TestTOTP_OTPURLFormat tests that OTP URL has correct format
func TestTOTP_OTPURLFormat(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	setupInfo, _ := totpManager.GenerateSecret("user@company.at")

	// OTP URL should start with otpauth://totp/
	expectedPrefix := "otpauth://totp/"
	if len(setupInfo.OTPURL) < len(expectedPrefix) {
		t.Fatal("OTP URL too short")
	}

	if setupInfo.OTPURL[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("OTP URL should start with '%s', got '%s'", expectedPrefix, setupInfo.OTPURL[:20])
	}

	// Should contain the issuer
	if !containsSubstring(setupInfo.OTPURL, "issuer=") {
		t.Error("OTP URL should contain issuer parameter")
	}

	// Should contain the secret
	if !containsSubstring(setupInfo.OTPURL, "secret=") {
		t.Error("OTP URL should contain secret parameter")
	}
}

// TestTOTP_Constants tests TOTP configuration constants
func TestTOTP_Constants(t *testing.T) {
	// Verify constants match expected values
	if auth.TOTPIssuer != "Austrian Business Platform" {
		t.Errorf("unexpected TOTP issuer: %s", auth.TOTPIssuer)
	}

	if auth.TOTPSecretSize != 20 {
		t.Errorf("unexpected TOTP secret size: %d", auth.TOTPSecretSize)
	}

	if auth.TOTPDigits != 6 {
		t.Errorf("unexpected TOTP digits: %d", auth.TOTPDigits)
	}

	if auth.TOTPPeriod != 30 {
		t.Errorf("unexpected TOTP period: %d", auth.TOTPPeriod)
	}
}

// TestTOTP_UniqueSecrets tests that each generation creates unique secrets
func TestTOTP_UniqueSecrets(t *testing.T) {
	km := crypto.NewKeyManager()
	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	totpManager := auth.NewTOTPManager(km)

	// Generate multiple secrets
	secrets := make([][]byte, 5)
	for i := 0; i < 5; i++ {
		setupInfo, err := totpManager.GenerateSecret("test@example.com")
		if err != nil {
			t.Fatalf("failed to generate secret %d: %v", i, err)
		}
		secrets[i] = setupInfo.Secret
	}

	// Verify all secrets are unique
	for i := 0; i < len(secrets); i++ {
		for j := i + 1; j < len(secrets); j++ {
			if bytesAreEqual(secrets[i], secrets[j]) {
				t.Errorf("secrets %d and %d should be different", i, j)
			}
		}
	}
}

// containsSubstring checks if string contains substring (helper for TOTP tests)
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// bytesAreEqual compares two byte slices (helper for TOTP tests)
func bytesAreEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
