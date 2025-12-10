package unit

import (
	"os"
	"path/filepath"
	"testing"

	"austrian-business-infrastructure/internal/crypto"
	"github.com/google/uuid"
)

// ============================================================
// KeyManager Tests
// ============================================================

// TestKeyManager_LoadMasterKey tests direct master key loading
func TestKeyManager_LoadMasterKey(t *testing.T) {
	km := crypto.NewKeyManager()

	// Generate a valid 32-byte key
	testKey := make([]byte, 32)
	for i := range testKey {
		testKey[i] = byte(i)
	}

	// Before loading
	if km.IsLoaded() {
		t.Error("key should not be loaded initially")
	}

	// Load key
	err := km.LoadMasterKey(testKey)
	if err != nil {
		t.Fatalf("failed to load master key: %v", err)
	}

	// After loading
	if !km.IsLoaded() {
		t.Error("key should be loaded after LoadMasterKey")
	}

	// Get key
	key, err := km.GetMasterKey()
	if err != nil {
		t.Fatalf("failed to get master key: %v", err)
	}

	// Verify key matches (but is a copy)
	for i := range key {
		if key[i] != testKey[i] {
			t.Errorf("key byte %d mismatch: got %d, expected %d", i, key[i], testKey[i])
		}
	}
}

// TestKeyManager_LoadMasterKeyInvalidLength tests rejection of wrong-length keys
func TestKeyManager_LoadMasterKeyInvalidLength(t *testing.T) {
	km := crypto.NewKeyManager()

	// Try loading keys of various wrong lengths
	wrongLengths := []int{0, 16, 24, 31, 33, 64}

	for _, length := range wrongLengths {
		testKey := make([]byte, length)
		err := km.LoadMasterKey(testKey)

		if err == nil {
			t.Errorf("expected error for key length %d", length)
		}
	}
}

// TestKeyManager_Clear tests clearing key from memory
func TestKeyManager_Clear(t *testing.T) {
	km := crypto.NewKeyManager()

	testKey := make([]byte, 32)
	km.LoadMasterKey(testKey)

	if !km.IsLoaded() {
		t.Error("key should be loaded")
	}

	km.Clear()

	if km.IsLoaded() {
		t.Error("key should not be loaded after Clear")
	}

	_, err := km.GetMasterKey()
	if err != crypto.ErrNoMasterKey {
		t.Errorf("expected ErrNoMasterKey, got %v", err)
	}
}

// TestKeyManager_LoadFromEnv tests loading from environment variable
func TestKeyManager_LoadFromEnv(t *testing.T) {
	km := crypto.NewKeyManager()

	// Set environment variable with valid hex key (64 hex chars = 32 bytes)
	hexKey := "0001020304050607080910111213141516171819202122232425262728293031"
	os.Setenv("MASTER_KEY", hexKey)
	defer os.Unsetenv("MASTER_KEY")

	err := km.LoadMasterKeyFromEnv()
	if err != nil {
		t.Fatalf("failed to load from env: %v", err)
	}

	if !km.IsLoaded() {
		t.Error("key should be loaded")
	}
}

// TestKeyManager_LoadFromEnvMissing tests error when env var is missing
func TestKeyManager_LoadFromEnvMissing(t *testing.T) {
	km := crypto.NewKeyManager()

	os.Unsetenv("MASTER_KEY")

	err := km.LoadMasterKeyFromEnv()
	if err == nil {
		t.Error("expected error when env var is missing")
	}
}

// TestKeyManager_LoadFromEnvInvalidHex tests error for invalid hex
func TestKeyManager_LoadFromEnvInvalidHex(t *testing.T) {
	km := crypto.NewKeyManager()

	os.Setenv("MASTER_KEY", "not-valid-hex-string!")
	defer os.Unsetenv("MASTER_KEY")

	err := km.LoadMasterKeyFromEnv()
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}

// TestKeyManager_LoadFromFile tests loading from file
func TestKeyManager_LoadFromFile(t *testing.T) {
	km := crypto.NewKeyManager()

	// Create temp file with valid hex key
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "master.key")

	hexKey := "0001020304050607080910111213141516171819202122232425262728293031"
	err := os.WriteFile(keyFile, []byte(hexKey), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err = km.LoadMasterKeyFromFile(keyFile)
	if err != nil {
		t.Fatalf("failed to load from file: %v", err)
	}

	if !km.IsLoaded() {
		t.Error("key should be loaded")
	}
}

// TestKeyManager_LoadFromFileMissing tests error when file is missing
func TestKeyManager_LoadFromFileMissing(t *testing.T) {
	km := crypto.NewKeyManager()

	err := km.LoadMasterKeyFromFile("/nonexistent/path/to/key")
	if err == nil {
		t.Error("expected error when file is missing")
	}
}

// TestGenerateMasterKey tests master key generation
func TestGenerateMasterKey(t *testing.T) {
	hexKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("failed to generate master key: %v", err)
	}

	// Should be 64 hex chars (32 bytes)
	if len(hexKey) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(hexKey))
	}

	// Should be valid hex
	for _, c := range hexKey {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("invalid hex char: %c", c)
		}
	}

	// Should be loadable
	km := crypto.NewKeyManager()
	os.Setenv("MASTER_KEY", hexKey)
	defer os.Unsetenv("MASTER_KEY")

	err = km.LoadMasterKeyFromEnv()
	if err != nil {
		t.Fatalf("generated key should be loadable: %v", err)
	}
}

// TestGenerateNonce tests nonce generation
func TestGenerateNonce(t *testing.T) {
	nonce, err := crypto.GenerateNonce()
	if err != nil {
		t.Fatalf("failed to generate nonce: %v", err)
	}

	// Should be 12 bytes
	if len(nonce) != crypto.NonceSize {
		t.Errorf("expected %d bytes, got %d", crypto.NonceSize, len(nonce))
	}

	// Generate another and verify they're different
	nonce2, _ := crypto.GenerateNonce()
	same := true
	for i := range nonce {
		if nonce[i] != nonce2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("nonces should be unique")
	}
}

// ============================================================
// Key Derivation (HKDF) Tests
// ============================================================

// TestDeriveKey_Basic tests basic HKDF key derivation
func TestDeriveKey_Basic(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	salt := []byte("test-salt")
	info := "test-info"

	derivedKey, err := crypto.DeriveKey(masterKey, salt, info)
	if err != nil {
		t.Fatalf("failed to derive key: %v", err)
	}

	// Should be 32 bytes
	if len(derivedKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(derivedKey))
	}

	// Derived key should be different from master key
	same := true
	for i := range derivedKey {
		if derivedKey[i] != masterKey[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("derived key should differ from master key")
	}
}

// TestDeriveKey_Deterministic tests that HKDF is deterministic
func TestDeriveKey_Deterministic(t *testing.T) {
	masterKey := make([]byte, 32)
	salt := []byte("same-salt")
	info := "same-info"

	derived1, _ := crypto.DeriveKey(masterKey, salt, info)
	derived2, _ := crypto.DeriveKey(masterKey, salt, info)

	for i := range derived1 {
		if derived1[i] != derived2[i] {
			t.Error("same inputs should produce same output")
			break
		}
	}
}

// TestDeriveKey_DifferentSalt tests that different salts produce different keys
func TestDeriveKey_DifferentSalt(t *testing.T) {
	masterKey := make([]byte, 32)

	derived1, _ := crypto.DeriveKey(masterKey, []byte("salt1"), "info")
	derived2, _ := crypto.DeriveKey(masterKey, []byte("salt2"), "info")

	same := true
	for i := range derived1 {
		if derived1[i] != derived2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different salts should produce different keys")
	}
}

// TestDeriveKey_DifferentInfo tests that different info produces different keys
func TestDeriveKey_DifferentInfo(t *testing.T) {
	masterKey := make([]byte, 32)
	salt := []byte("same-salt")

	derived1, _ := crypto.DeriveKey(masterKey, salt, "info1")
	derived2, _ := crypto.DeriveKey(masterKey, salt, "info2")

	same := true
	for i := range derived1 {
		if derived1[i] != derived2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different info should produce different keys")
	}
}

// TestDeriveKey_InvalidMasterKey tests error for wrong-length master key
func TestDeriveKey_InvalidMasterKey(t *testing.T) {
	shortKey := make([]byte, 16)
	_, err := crypto.DeriveKey(shortKey, []byte("salt"), "info")

	if err == nil {
		t.Error("expected error for invalid key length")
	}
}

// TestDeriveTenantKey tests tenant key derivation
func TestDeriveTenantKey(t *testing.T) {
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}

	tenantID := uuid.New()

	tenantKey, err := crypto.DeriveTenantKey(masterKey, tenantID)
	if err != nil {
		t.Fatalf("failed to derive tenant key: %v", err)
	}

	if len(tenantKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(tenantKey))
	}
}

// TestDeriveTenantKey_DifferentTenants tests that different tenants get different keys
func TestDeriveTenantKey_DifferentTenants(t *testing.T) {
	masterKey := make([]byte, 32)

	tenant1 := uuid.New()
	tenant2 := uuid.New()

	key1, _ := crypto.DeriveTenantKey(masterKey, tenant1)
	key2, _ := crypto.DeriveTenantKey(masterKey, tenant2)

	same := true
	for i := range key1 {
		if key1[i] != key2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different tenants should have different keys")
	}
}

// TestDeriveCredentialKey tests credential key derivation
func TestDeriveCredentialKey(t *testing.T) {
	tenantKey := make([]byte, 32)
	for i := range tenantKey {
		tenantKey[i] = byte(i)
	}

	credentialID := uuid.New()

	credKey, err := crypto.DeriveCredentialKey(tenantKey, credentialID)
	if err != nil {
		t.Fatalf("failed to derive credential key: %v", err)
	}

	if len(credKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(credKey))
	}
}

// TestDeriveFullCredentialKey tests full key derivation chain
func TestDeriveFullCredentialKey(t *testing.T) {
	km := crypto.NewKeyManager()

	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	km.LoadMasterKey(masterKey)

	tenantID := uuid.New()
	credentialID := uuid.New()

	credKey, err := crypto.DeriveFullCredentialKey(km, tenantID, credentialID)
	if err != nil {
		t.Fatalf("failed to derive full credential key: %v", err)
	}

	if len(credKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(credKey))
	}

	// Verify key is deterministic
	credKey2, _ := crypto.DeriveFullCredentialKey(km, tenantID, credentialID)
	for i := range credKey {
		if credKey[i] != credKey2[i] {
			t.Error("same inputs should produce same key")
			break
		}
	}
}

// TestDeriveFullCredentialKey_NoMasterKey tests error when master key not loaded
func TestDeriveFullCredentialKey_NoMasterKey(t *testing.T) {
	km := crypto.NewKeyManager()
	// Don't load master key

	_, err := crypto.DeriveFullCredentialKey(km, uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error when master key not loaded")
	}
}

// TestDeriveTOTPKey tests TOTP key derivation
func TestDeriveTOTPKey(t *testing.T) {
	tenantKey := make([]byte, 32)
	userID := uuid.New()

	totpKey, err := crypto.DeriveTOTPKey(tenantKey, userID)
	if err != nil {
		t.Fatalf("failed to derive TOTP key: %v", err)
	}

	if len(totpKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(totpKey))
	}
}

// TestDeriveRecoveryKey tests recovery key derivation
func TestDeriveRecoveryKey(t *testing.T) {
	tenantKey := make([]byte, 32)
	userID := uuid.New()

	recoveryKey, err := crypto.DeriveRecoveryKey(tenantKey, userID)
	if err != nil {
		t.Fatalf("failed to derive recovery key: %v", err)
	}

	if len(recoveryKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(recoveryKey))
	}
}

// TestDeriveExportKey tests export key derivation
func TestDeriveExportKey(t *testing.T) {
	tenantKey := make([]byte, 32)
	exportID := uuid.New()

	exportKey, err := crypto.DeriveExportKey(tenantKey, exportID)
	if err != nil {
		t.Fatalf("failed to derive export key: %v", err)
	}

	if len(exportKey) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(exportKey))
	}
}

// ============================================================
// KeyDeriver Tests
// ============================================================

// TestKeyDeriver_GetCredentialKey tests KeyDeriver.GetCredentialKey
func TestKeyDeriver_GetCredentialKey(t *testing.T) {
	km := crypto.NewKeyManager()
	masterKey := make([]byte, 32)
	for i := range masterKey {
		masterKey[i] = byte(i)
	}
	km.LoadMasterKey(masterKey)

	kd := crypto.NewKeyDeriver(km)

	tenantID := uuid.New()
	credentialID := uuid.New()

	key, err := kd.GetCredentialKey(tenantID, credentialID)
	if err != nil {
		t.Fatalf("failed to get credential key: %v", err)
	}

	if len(key) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(key))
	}
}

// TestKeyDeriver_GetTOTPKey tests KeyDeriver.GetTOTPKey
func TestKeyDeriver_GetTOTPKey(t *testing.T) {
	km := crypto.NewKeyManager()
	masterKey := make([]byte, 32)
	km.LoadMasterKey(masterKey)

	kd := crypto.NewKeyDeriver(km)

	tenantID := uuid.New()
	userID := uuid.New()

	key, err := kd.GetTOTPKey(tenantID, userID)
	if err != nil {
		t.Fatalf("failed to get TOTP key: %v", err)
	}

	if len(key) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(key))
	}
}

// TestKeyDeriver_GetRecoveryKey tests KeyDeriver.GetRecoveryKey
func TestKeyDeriver_GetRecoveryKey(t *testing.T) {
	km := crypto.NewKeyManager()
	masterKey := make([]byte, 32)
	km.LoadMasterKey(masterKey)

	kd := crypto.NewKeyDeriver(km)

	tenantID := uuid.New()
	userID := uuid.New()

	key, err := kd.GetRecoveryKey(tenantID, userID)
	if err != nil {
		t.Fatalf("failed to get recovery key: %v", err)
	}

	if len(key) != crypto.KeySize {
		t.Errorf("expected %d bytes, got %d", crypto.KeySize, len(key))
	}
}

// ============================================================
// Zero Tests
// ============================================================

// TestZero tests secure memory zeroing
func TestZero(t *testing.T) {
	// Create a buffer with data
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	// Zero it
	crypto.Zero(data)

	// Verify all bytes are zero
	for i, b := range data {
		if b != 0 {
			t.Errorf("byte %d should be zero, got %d", i, b)
		}
	}
}

// TestZero_EmptySlice tests zeroing empty slice (should not panic)
func TestZero_EmptySlice(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Zero panicked on empty slice: %v", r)
		}
	}()

	crypto.Zero([]byte{})
	crypto.Zero(nil)
}
