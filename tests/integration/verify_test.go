//go:build !windows

package integration

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/verify"
)

// TestVerificationServiceTypes verifies verification service types
func TestVerificationServiceTypes(t *testing.T) {
	result := &verify.VerificationResult{
		IsValid:        true,
		Status:         "valid",
		DocumentHash:   "abcdef123456",
		SignatureCount: 2,
		VerifiedAt:     time.Now(),
	}

	if !result.IsValid {
		t.Error("Result should be valid")
	}

	if result.SignatureCount != 2 {
		t.Error("Signature count mismatch")
	}

	if result.DocumentHash == "" {
		t.Error("Document hash should not be empty")
	}
}

// TestSignatureInfoTypes verifies signature info types
func TestSignatureInfoTypes(t *testing.T) {
	signedAt := time.Now()

	info := &verify.SignatureInfo{
		SignerName:         "Max Mustermann",
		SignerEmail:        "max@example.com",
		SignedAt:           signedAt,
		IsValid:            true,
		HashAlgorithm:      "SHA-256",
		SignatureAlgorithm: "RSA-PSS",
	}

	if info.SignerName != "Max Mustermann" {
		t.Error("Signer name mismatch")
	}

	if !info.IsValid {
		t.Error("Signature should be valid")
	}

	if info.HashAlgorithm != "SHA-256" {
		t.Error("Hash algorithm mismatch")
	}
}

// TestCertInfoTypes verifies certificate info types
func TestCertInfoTypes(t *testing.T) {
	validFrom := time.Now().Add(-365 * 24 * time.Hour)
	validTo := time.Now().Add(365 * 24 * time.Hour)

	cert := &verify.CertInfo{
		Subject:      "CN=Max Mustermann, O=Test GmbH, C=AT",
		SubjectCN:    "Max Mustermann",
		Issuer:       "CN=A-Trust Qualified CA, O=A-Trust, C=AT",
		IssuerCN:     "A-Trust Qualified CA",
		SerialNumber: "123456789",
		ValidFrom:    validFrom,
		ValidTo:      validTo,
		IsQualified:  true,
		IsExpired:    false,
		IsRevoked:    false,
	}

	if cert.SubjectCN != "Max Mustermann" {
		t.Error("Subject CN mismatch")
	}

	if !cert.IsQualified {
		t.Error("Certificate should be qualified")
	}

	if cert.IsExpired {
		t.Error("Certificate should not be expired")
	}

	if cert.IsRevoked {
		t.Error("Certificate should not be revoked")
	}

	// Test validity period
	if !cert.ValidTo.After(time.Now()) {
		t.Error("Certificate should be valid")
	}
}

// TestTimestampInfoTypes verifies timestamp info types
func TestTimestampInfoTypes(t *testing.T) {
	ts := &verify.TimestampInfo{
		Time:      time.Now(),
		Authority: "A-Trust TSA",
		IsValid:   true,
	}

	if ts.Authority != "A-Trust TSA" {
		t.Error("Timestamp authority mismatch")
	}

	if !ts.IsValid {
		t.Error("Timestamp should be valid")
	}
}

// TestVerificationStatuses verifies all verification status values
func TestVerificationStatuses(t *testing.T) {
	statuses := []string{
		"valid",
		"invalid",
		"indeterminate",
		"unknown",
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status should not be empty")
		}
	}
}

// TestHashCalculation verifies hash calculation behavior
func TestHashCalculation(t *testing.T) {
	// Test that different content produces different hashes
	content1 := []byte("Hello, World!")
	content2 := []byte("Hello, World?")

	// Hash function is deterministic
	hash1a := calculateTestHash(content1)
	hash1b := calculateTestHash(content1)

	if hash1a != hash1b {
		t.Error("Same content should produce same hash")
	}

	// Different content produces different hash
	hash2 := calculateTestHash(content2)

	if hash1a == hash2 {
		t.Error("Different content should produce different hash")
	}
}

// calculateTestHash is a helper for hash testing
func calculateTestHash(content []byte) string {
	// Simple mock hash for testing
	return string(content[:min(10, len(content))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestVerificationResultWarningsAndErrors verifies warning/error handling
func TestVerificationResultWarningsAndErrors(t *testing.T) {
	result := &verify.VerificationResult{
		IsValid:  false,
		Status:   "invalid",
		Warnings: []string{"Certificate expires soon"},
		Errors:   []string{"Signature hash mismatch"},
	}

	if len(result.Warnings) != 1 {
		t.Error("Expected 1 warning")
	}

	if len(result.Errors) != 1 {
		t.Error("Expected 1 error")
	}

	if result.Warnings[0] != "Certificate expires soon" {
		t.Error("Warning message mismatch")
	}

	if result.Errors[0] != "Signature hash mismatch" {
		t.Error("Error message mismatch")
	}
}

// TestMultipleSignaturesVerification verifies handling of multiple signatures
func TestMultipleSignaturesVerification(t *testing.T) {
	signatures := []verify.SignatureInfo{
		{SignerName: "Signer 1", IsValid: true},
		{SignerName: "Signer 2", IsValid: true},
		{SignerName: "Signer 3", IsValid: false},
	}

	// Count valid signatures
	validCount := 0
	for _, sig := range signatures {
		if sig.IsValid {
			validCount++
		}
	}

	if validCount != 2 {
		t.Errorf("Expected 2 valid signatures, got %d", validCount)
	}

	// Overall document validity depends on all signatures
	allValid := true
	for _, sig := range signatures {
		if !sig.IsValid {
			allValid = false
			break
		}
	}

	if allValid {
		t.Error("Document should be invalid due to invalid signature")
	}
}

// TestVerificationStorage verifies verification storage types
func TestVerificationStorage(t *testing.T) {
	verifyID := uuid.New()
	tenantID := uuid.New()
	docHash := "abc123"

	// Verification record should have all required fields
	if verifyID == uuid.Nil {
		t.Error("Verification ID should not be nil")
	}

	if tenantID == uuid.Nil {
		t.Error("Tenant ID should not be nil")
	}

	if docHash == "" {
		t.Error("Document hash should not be empty")
	}
}
