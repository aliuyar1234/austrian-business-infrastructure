package unit

import (
	"bytes"
	"encoding/json"
	"testing"

	"austrian-business-infrastructure/internal/account"
)

// T066: Unit tests for encryption

func TestNewEncryptor(t *testing.T) {
	t.Run("Valid 32-byte key", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}

		enc, err := account.NewEncryptor(key)
		if err != nil {
			t.Fatalf("Failed to create encryptor with valid key: %v", err)
		}
		if enc == nil {
			t.Fatal("Encryptor is nil")
		}
	})

	t.Run("Key too short", func(t *testing.T) {
		key := make([]byte, 16)
		_, err := account.NewEncryptor(key)
		if err != account.ErrInvalidKey {
			t.Errorf("Expected ErrInvalidKey, got %v", err)
		}
	})

	t.Run("Key too long", func(t *testing.T) {
		key := make([]byte, 64)
		_, err := account.NewEncryptor(key)
		if err != account.ErrInvalidKey {
			t.Errorf("Expected ErrInvalidKey, got %v", err)
		}
	})

	t.Run("Empty key", func(t *testing.T) {
		_, err := account.NewEncryptor([]byte{})
		if err != account.ErrInvalidKey {
			t.Errorf("Expected ErrInvalidKey, got %v", err)
		}
	})
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, _ := account.NewEncryptor(key)

	t.Run("Round trip encryption", func(t *testing.T) {
		plaintext := []byte("This is secret data!")

		ciphertext, iv, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt failed: %v", err)
		}

		if len(ciphertext) == 0 {
			t.Error("Ciphertext is empty")
		}
		if len(iv) == 0 {
			t.Error("IV is empty")
		}

		// Ciphertext should not equal plaintext
		if bytes.Equal(ciphertext, plaintext) {
			t.Error("Ciphertext equals plaintext")
		}

		// Decrypt
		decrypted, err := enc.Decrypt(ciphertext, iv)
		if err != nil {
			t.Fatalf("Decrypt failed: %v", err)
		}

		if !bytes.Equal(decrypted, plaintext) {
			t.Errorf("Decrypted data does not match: got %s, want %s", decrypted, plaintext)
		}
	})

	t.Run("Encrypt empty data", func(t *testing.T) {
		plaintext := []byte{}

		ciphertext, iv, err := enc.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("Encrypt failed: %v", err)
		}

		decrypted, err := enc.Decrypt(ciphertext, iv)
		if err != nil {
			t.Fatalf("Decrypt failed: %v", err)
		}

		if len(decrypted) != 0 {
			t.Error("Decrypted empty data should be empty")
		}
	})

	t.Run("Different IVs for same plaintext", func(t *testing.T) {
		plaintext := []byte("Same data twice")

		ciphertext1, iv1, _ := enc.Encrypt(plaintext)
		ciphertext2, iv2, _ := enc.Encrypt(plaintext)

		// IVs should be different (random)
		if bytes.Equal(iv1, iv2) {
			t.Error("IVs should be different for each encryption")
		}

		// Ciphertexts should be different due to different IVs
		if bytes.Equal(ciphertext1, ciphertext2) {
			t.Error("Ciphertexts should be different with different IVs")
		}
	})

	t.Run("Wrong IV fails decryption", func(t *testing.T) {
		plaintext := []byte("Secret message")

		ciphertext, iv, _ := enc.Encrypt(plaintext)

		// Modify IV
		wrongIV := make([]byte, len(iv))
		copy(wrongIV, iv)
		wrongIV[0] ^= 0xFF

		_, err := enc.Decrypt(ciphertext, wrongIV)
		if err != account.ErrDecryptionFailed {
			t.Errorf("Expected ErrDecryptionFailed with wrong IV, got %v", err)
		}
	})

	t.Run("Tampered ciphertext fails", func(t *testing.T) {
		plaintext := []byte("Secret message")

		ciphertext, iv, _ := enc.Encrypt(plaintext)

		// Tamper with ciphertext
		ciphertext[0] ^= 0xFF

		_, err := enc.Decrypt(ciphertext, iv)
		if err != account.ErrDecryptionFailed {
			t.Errorf("Expected ErrDecryptionFailed with tampered ciphertext, got %v", err)
		}
	})

	t.Run("Invalid IV length", func(t *testing.T) {
		plaintext := []byte("Secret message")

		ciphertext, _, _ := enc.Encrypt(plaintext)

		// Use wrong IV length
		shortIV := []byte{1, 2, 3}
		_, err := enc.Decrypt(ciphertext, shortIV)
		if err != account.ErrInvalidIV {
			t.Errorf("Expected ErrInvalidIV, got %v", err)
		}
	})
}

func TestEncryptDecryptJSON(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	enc, _ := account.NewEncryptor(key)

	type TestStruct struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
		Value  int    `json:"value"`
	}

	t.Run("Round trip JSON", func(t *testing.T) {
		original := &TestStruct{
			Name:   "Test",
			Secret: "super-secret",
			Value:  42,
		}

		ciphertext, iv, err := enc.EncryptJSON(original)
		if err != nil {
			t.Fatalf("EncryptJSON failed: %v", err)
		}

		var decrypted TestStruct
		err = enc.DecryptJSON(ciphertext, iv, &decrypted)
		if err != nil {
			t.Fatalf("DecryptJSON failed: %v", err)
		}

		if decrypted.Name != original.Name {
			t.Errorf("Name mismatch: got %s, want %s", decrypted.Name, original.Name)
		}
		if decrypted.Secret != original.Secret {
			t.Errorf("Secret mismatch: got %s, want %s", decrypted.Secret, original.Secret)
		}
		if decrypted.Value != original.Value {
			t.Errorf("Value mismatch: got %d, want %d", decrypted.Value, original.Value)
		}
	})

	t.Run("JSON with special characters", func(t *testing.T) {
		original := &TestStruct{
			Name:   "Test with \"quotes\" and \\ backslash",
			Secret: "äöü€ Unicode",
			Value:  -999,
		}

		ciphertext, iv, err := enc.EncryptJSON(original)
		if err != nil {
			t.Fatalf("EncryptJSON failed: %v", err)
		}

		var decrypted TestStruct
		err = enc.DecryptJSON(ciphertext, iv, &decrypted)
		if err != nil {
			t.Fatalf("DecryptJSON failed: %v", err)
		}

		if decrypted.Name != original.Name {
			t.Errorf("Name mismatch with special chars")
		}
		if decrypted.Secret != original.Secret {
			t.Errorf("Secret mismatch with Unicode")
		}
	})

	t.Run("JSON map type", func(t *testing.T) {
		original := map[string]interface{}{
			"key1": "value1",
			"key2": float64(42), // JSON numbers are float64
		}

		ciphertext, iv, err := enc.EncryptJSON(original)
		if err != nil {
			t.Fatalf("EncryptJSON failed: %v", err)
		}

		var decrypted map[string]interface{}
		err = enc.DecryptJSON(ciphertext, iv, &decrypted)
		if err != nil {
			t.Fatalf("DecryptJSON failed: %v", err)
		}

		if decrypted["key1"] != original["key1"] {
			t.Error("Map value mismatch")
		}
	})

	t.Run("Invalid JSON fails marshal", func(t *testing.T) {
		// Channel can't be marshaled to JSON
		ch := make(chan int)
		_, _, err := enc.EncryptJSON(ch)
		if err == nil {
			t.Error("Expected error marshaling channel")
		}
	})

	t.Run("Invalid JSON fails unmarshal", func(t *testing.T) {
		// Encrypt valid JSON
		original := map[string]string{"key": "value"}
		ciphertext, iv, _ := enc.EncryptJSON(original)

		// Try to unmarshal into wrong type
		var decrypted int
		err := enc.DecryptJSON(ciphertext, iv, &decrypted)
		if err == nil {
			t.Error("Expected error unmarshaling to wrong type")
		}
	})
}

func TestKeyRotation(t *testing.T) {
	oldKey := make([]byte, 32)
	newKey := make([]byte, 32)
	for i := range oldKey {
		oldKey[i] = byte(i)
		newKey[i] = byte(i + 1)
	}

	t.Run("Rotate key successfully", func(t *testing.T) {
		oldEnc, _ := account.NewEncryptor(oldKey)
		newEnc, _ := account.NewEncryptor(newKey)

		// Encrypt with old key
		plaintext := []byte("Secret data to rotate")
		ciphertext, iv, _ := oldEnc.Encrypt(plaintext)

		// Rotate to new key
		newCiphertext, newIV, err := account.RotateKey(oldEnc, newEnc, ciphertext, iv)
		if err != nil {
			t.Fatalf("RotateKey failed: %v", err)
		}

		// Verify old key can't decrypt new ciphertext
		_, err = oldEnc.Decrypt(newCiphertext, newIV)
		if err != account.ErrDecryptionFailed {
			t.Error("Old key should not decrypt rotated ciphertext")
		}

		// Verify new key can decrypt
		decrypted, err := newEnc.Decrypt(newCiphertext, newIV)
		if err != nil {
			t.Fatalf("New key should decrypt rotated ciphertext: %v", err)
		}

		if !bytes.Equal(decrypted, plaintext) {
			t.Error("Decrypted data doesn't match original")
		}
	})

	t.Run("Rotation with invalid old ciphertext", func(t *testing.T) {
		oldEnc, _ := account.NewEncryptor(oldKey)
		newEnc, _ := account.NewEncryptor(newKey)

		// Use invalid ciphertext
		invalidCiphertext := []byte("not valid ciphertext")
		iv := make([]byte, 12) // GCM nonce size

		_, _, err := account.RotateKey(oldEnc, newEnc, invalidCiphertext, iv)
		if err == nil {
			t.Error("Expected error rotating invalid ciphertext")
		}
	})
}

func TestKeyRotator(t *testing.T) {
	oldKey := make([]byte, 32)
	newKey := make([]byte, 32)
	for i := range oldKey {
		oldKey[i] = byte(i)
		newKey[i] = byte(255 - i)
	}

	t.Run("Create rotator with valid keys", func(t *testing.T) {
		rotator, err := account.NewKeyRotator(oldKey, newKey)
		if err != nil {
			t.Fatalf("Failed to create rotator: %v", err)
		}
		if rotator == nil {
			t.Fatal("Rotator is nil")
		}
	})

	t.Run("Create rotator with invalid old key", func(t *testing.T) {
		_, err := account.NewKeyRotator([]byte("short"), newKey)
		if err == nil {
			t.Error("Expected error with invalid old key")
		}
	})

	t.Run("Create rotator with invalid new key", func(t *testing.T) {
		_, err := account.NewKeyRotator(oldKey, []byte("short"))
		if err == nil {
			t.Error("Expected error with invalid new key")
		}
	})

	t.Run("RotateCredentials", func(t *testing.T) {
		rotator, _ := account.NewKeyRotator(oldKey, newKey)

		// Encrypt with old key
		oldEnc, _ := account.NewEncryptor(oldKey)
		plaintext := []byte(`{"tid":"123456789","pin":"secret"}`)
		ciphertext, iv, _ := oldEnc.Encrypt(plaintext)

		// Rotate
		newCiphertext, newIV, err := rotator.RotateCredentials(ciphertext, iv)
		if err != nil {
			t.Fatalf("RotateCredentials failed: %v", err)
		}

		// Decrypt with new key
		newEnc, _ := account.NewEncryptor(newKey)
		decrypted, err := newEnc.Decrypt(newCiphertext, newIV)
		if err != nil {
			t.Fatalf("Decrypt with new key failed: %v", err)
		}

		if !bytes.Equal(decrypted, plaintext) {
			t.Error("Decrypted data doesn't match")
		}
	})
}

func TestBatchRotator(t *testing.T) {
	oldKey := make([]byte, 32)
	newKey := make([]byte, 32)
	for i := range oldKey {
		oldKey[i] = byte(i)
		newKey[i] = byte(i + 100)
	}

	t.Run("Rotate batch of credentials", func(t *testing.T) {
		batchRotator, err := account.NewBatchRotator(oldKey, newKey, 10)
		if err != nil {
			t.Fatalf("Failed to create batch rotator: %v", err)
		}

		// Create test data encrypted with old key
		oldEnc, _ := account.NewEncryptor(oldKey)
		testData := []account.EncryptedData{}
		originalValues := []string{}

		for i := 0; i < 5; i++ {
			value := []byte("Secret " + string(rune('A'+i)))
			originalValues = append(originalValues, string(value))
			ciphertext, iv, _ := oldEnc.Encrypt(value)
			testData = append(testData, account.EncryptedData{
				Ciphertext: ciphertext,
				IV:         iv,
			})
		}

		// Rotate batch
		rotatedData, err := batchRotator.RotateBatch(testData)
		if err != nil {
			t.Fatalf("RotateBatch failed: %v", err)
		}

		if len(rotatedData) != len(testData) {
			t.Fatalf("Rotated batch size mismatch: got %d, want %d", len(rotatedData), len(testData))
		}

		// Verify all items can be decrypted with new key
		newEnc, _ := account.NewEncryptor(newKey)
		for i, data := range rotatedData {
			decrypted, err := newEnc.Decrypt(data.Ciphertext, data.IV)
			if err != nil {
				t.Errorf("Failed to decrypt rotated item %d: %v", i, err)
			}
			if string(decrypted) != originalValues[i] {
				t.Errorf("Item %d value mismatch: got %s, want %s", i, decrypted, originalValues[i])
			}
		}
	})

	t.Run("Default batch size", func(t *testing.T) {
		batchRotator, err := account.NewBatchRotator(oldKey, newKey, 0)
		if err != nil {
			t.Fatalf("Failed to create batch rotator with zero batch size: %v", err)
		}
		if batchRotator == nil {
			t.Fatal("BatchRotator is nil")
		}
	})

	t.Run("Batch rotation with invalid item fails", func(t *testing.T) {
		batchRotator, _ := account.NewBatchRotator(oldKey, newKey, 10)

		testData := []account.EncryptedData{
			{Ciphertext: []byte("invalid"), IV: make([]byte, 12)},
		}

		_, err := batchRotator.RotateBatch(testData)
		if err == nil {
			t.Error("Expected error with invalid ciphertext")
		}
	})
}

func TestGenerateKey(t *testing.T) {
	t.Run("Generate valid key", func(t *testing.T) {
		key, err := account.GenerateKey()
		if err != nil {
			t.Fatalf("GenerateKey failed: %v", err)
		}

		if len(key) != 32 {
			t.Errorf("Key length wrong: got %d, want 32", len(key))
		}

		// Should be usable
		enc, err := account.NewEncryptor(key)
		if err != nil {
			t.Fatalf("Generated key not usable: %v", err)
		}

		// Round trip test
		plaintext := []byte("Test with generated key")
		ciphertext, iv, _ := enc.Encrypt(plaintext)
		decrypted, _ := enc.Decrypt(ciphertext, iv)

		if !bytes.Equal(decrypted, plaintext) {
			t.Error("Round trip failed with generated key")
		}
	})

	t.Run("Generated keys are unique", func(t *testing.T) {
		key1, _ := account.GenerateKey()
		key2, _ := account.GenerateKey()

		if bytes.Equal(key1, key2) {
			t.Error("Generated keys should be unique")
		}
	})
}

func TestCredentialEncryption(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i * 3)
	}
	enc, _ := account.NewEncryptor(key)

	t.Run("Encrypt FinanzOnline credentials", func(t *testing.T) {
		creds := map[string]string{
			"tid":    "123456789",
			"ben_id": "TESTUSER",
			"pin":    "secretpin123",
		}

		ciphertext, iv, err := enc.EncryptJSON(creds)
		if err != nil {
			t.Fatalf("Failed to encrypt FO credentials: %v", err)
		}

		var decrypted map[string]string
		err = enc.DecryptJSON(ciphertext, iv, &decrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt FO credentials: %v", err)
		}

		if decrypted["tid"] != creds["tid"] {
			t.Error("TID mismatch")
		}
		if decrypted["pin"] != creds["pin"] {
			t.Error("PIN mismatch")
		}
	})

	t.Run("Encrypt ELDA credentials", func(t *testing.T) {
		creds := map[string]string{
			"dienstgeber_nr":       "123456",
			"pin":                  "eldapin",
			"certificate_path":     "/path/to/cert.p12",
			"certificate_password": "certpass",
		}

		ciphertext, iv, err := enc.EncryptJSON(creds)
		if err != nil {
			t.Fatalf("Failed to encrypt ELDA credentials: %v", err)
		}

		var decrypted map[string]string
		err = enc.DecryptJSON(ciphertext, iv, &decrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt ELDA credentials: %v", err)
		}

		if decrypted["dienstgeber_nr"] != creds["dienstgeber_nr"] {
			t.Error("Dienstgebernummer mismatch")
		}
		if decrypted["certificate_password"] != creds["certificate_password"] {
			t.Error("Certificate password mismatch")
		}
	})

	t.Run("Large credential payload", func(t *testing.T) {
		// Create large payload
		largeValue := make([]byte, 10000)
		for i := range largeValue {
			largeValue[i] = byte(i % 256)
		}

		creds := map[string]interface{}{
			"large_field": string(largeValue),
			"small_field": "small",
		}

		data, _ := json.Marshal(creds)
		ciphertext, iv, err := enc.Encrypt(data)
		if err != nil {
			t.Fatalf("Failed to encrypt large payload: %v", err)
		}

		decrypted, err := enc.Decrypt(ciphertext, iv)
		if err != nil {
			t.Fatalf("Failed to decrypt large payload: %v", err)
		}

		if !bytes.Equal(decrypted, data) {
			t.Error("Large payload mismatch")
		}
	})
}
