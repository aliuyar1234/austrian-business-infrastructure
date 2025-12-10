package verify

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/signature"
)

// Service provides signature verification functionality
type Service struct {
	repo        *signature.Repository
	trustedCAs  *x509.CertPool
}

// NewService creates a new verification service
func NewService(repo *signature.Repository) *Service {
	// Initialize with A-Trust root certificates
	trustedCAs := x509.NewCertPool()
	// TODO: Add A-Trust root certificates to the pool

	return &Service{
		repo:       repo,
		trustedCAs: trustedCAs,
	}
}

// VerificationResult contains the result of verifying a signed PDF
type VerificationResult struct {
	IsValid            bool                   `json:"is_valid"`
	Status             signature.VerificationStatus `json:"status"`
	DocumentHash       string                 `json:"document_hash"`
	SignatureCount     int                    `json:"signature_count"`
	Signatures         []SignatureInfo        `json:"signatures"`
	Warnings           []string               `json:"warnings,omitempty"`
	Errors             []string               `json:"errors,omitempty"`
	VerifiedAt         time.Time              `json:"verified_at"`
}

// SignatureInfo contains information about a single signature
type SignatureInfo struct {
	SignerName         string         `json:"signer_name"`
	SignerEmail        string         `json:"signer_email,omitempty"`
	SignedAt           time.Time      `json:"signed_at"`
	IsValid            bool           `json:"is_valid"`
	ValidationMessages []string       `json:"validation_messages,omitempty"`
	Certificate        *CertInfo      `json:"certificate,omitempty"`
	Timestamp          *TimestampInfo `json:"timestamp,omitempty"`
	HashAlgorithm      string         `json:"hash_algorithm"`
	SignatureAlgorithm string         `json:"signature_algorithm"`
}

// CertInfo contains certificate information
type CertInfo struct {
	Subject      string    `json:"subject"`
	SubjectCN    string    `json:"subject_cn"`
	Issuer       string    `json:"issuer"`
	IssuerCN     string    `json:"issuer_cn"`
	SerialNumber string    `json:"serial_number"`
	ValidFrom    time.Time `json:"valid_from"`
	ValidTo      time.Time `json:"valid_to"`
	IsQualified  bool      `json:"is_qualified"`
	KeyUsage     []string  `json:"key_usage,omitempty"`
	IsExpired    bool      `json:"is_expired"`
	IsRevoked    bool      `json:"is_revoked"`
}

// TimestampInfo contains timestamp information
type TimestampInfo struct {
	Time       time.Time `json:"time"`
	Authority  string    `json:"authority"`
	IsValid    bool      `json:"is_valid"`
}

// VerifyDocument verifies signatures in a PDF document
func (s *Service) VerifyDocument(ctx context.Context, content []byte, filename string, tenantID uuid.UUID, userID *uuid.UUID) (*VerificationResult, error) {
	result := &VerificationResult{
		VerifiedAt:   time.Now(),
		DocumentHash: s.hashDocument(content),
	}

	// Parse PDF and extract signatures
	signatures, err := s.extractSignatures(content)
	if err != nil {
		result.IsValid = false
		result.Status = signature.VerificationStatusUnknown
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to extract signatures: %v", err))
		return result, nil
	}

	if len(signatures) == 0 {
		result.IsValid = false
		result.Status = signature.VerificationStatusInvalid
		result.Errors = append(result.Errors, "No signatures found in document")
		return result, nil
	}

	result.SignatureCount = len(signatures)
	result.Signatures = signatures

	// Validate each signature
	allValid := true
	hasInvalid := false
	for i := range signatures {
		if !signatures[i].IsValid {
			allValid = false
			hasInvalid = true
		}
	}

	if allValid {
		result.IsValid = true
		result.Status = signature.VerificationStatusValid
	} else if hasInvalid {
		result.IsValid = false
		result.Status = signature.VerificationStatusInvalid
	} else {
		result.IsValid = false
		result.Status = signature.VerificationStatusIndeterminate
	}

	// Store verification result
	signaturesJSON, _ := json.Marshal(result.Signatures)
	verification := &signature.Verification{
		TenantID:           tenantID,
		DocumentHash:       result.DocumentHash,
		IsValid:            result.IsValid,
		VerificationStatus: result.Status,
		Signatures:         signaturesJSON,
		SignatureCount:     result.SignatureCount,
		VerifiedBy:         userID,
	}
	if filename != "" {
		verification.OriginalFilename = &filename
	}

	if err := s.repo.CreateVerification(ctx, verification); err != nil {
		// Log but don't fail
	}

	return result, nil
}

// VerifyDocumentByID verifies a document stored in the system
func (s *Service) VerifyDocumentByID(ctx context.Context, documentID, tenantID uuid.UUID, userID *uuid.UUID, docStore signature.DocumentStore) (*VerificationResult, error) {
	// Get document content
	content, err := docStore.GetDocumentContent(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	result, err := s.VerifyDocument(ctx, content, "", tenantID, userID)
	if err != nil {
		return nil, err
	}

	// Update verification with document ID
	// This would be done in the stored verification record
	return result, nil
}

// GetVerification retrieves a stored verification result
func (s *Service) GetVerification(ctx context.Context, verificationID uuid.UUID) (*signature.Verification, error) {
	return s.repo.GetVerificationByID(ctx, verificationID)
}

// extractSignatures extracts signature information from a PDF
func (s *Service) extractSignatures(content []byte) ([]SignatureInfo, error) {
	// TODO: Implement actual PDF signature extraction using pdfcpu or similar
	// This is a placeholder implementation

	// For now, return an empty list (no signatures detected)
	// In a real implementation, this would:
	// 1. Parse the PDF structure
	// 2. Find signature dictionaries
	// 3. Extract PKCS#7/CMS signatures
	// 4. Validate the signatures
	// 5. Extract certificate chains
	// 6. Verify timestamps

	return []SignatureInfo{}, nil
}

// validateSignature validates a single signature
func (s *Service) validateSignature(sig *SignatureInfo, content []byte) error {
	// TODO: Implement actual signature validation
	// 1. Verify the cryptographic signature
	// 2. Validate the certificate chain
	// 3. Check certificate validity period
	// 4. Check certificate revocation status (OCSP/CRL)
	// 5. Validate the timestamp if present

	return nil
}

// validateCertificateChain validates a certificate chain
func (s *Service) validateCertificateChain(certs []*x509.Certificate) error {
	if len(certs) == 0 {
		return fmt.Errorf("empty certificate chain")
	}

	// Build verification options
	opts := x509.VerifyOptions{
		Roots:         s.trustedCAs,
		Intermediates: x509.NewCertPool(),
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	}

	// Add intermediate certificates
	for i := 1; i < len(certs); i++ {
		opts.Intermediates.AddCert(certs[i])
	}

	// Verify the leaf certificate
	_, err := certs[0].Verify(opts)
	return err
}

// checkCertificateRevocation checks if a certificate is revoked
func (s *Service) checkCertificateRevocation(cert *x509.Certificate) (bool, error) {
	// TODO: Implement OCSP/CRL checking
	// 1. Check OCSP response if available
	// 2. Fall back to CRL if OCSP not available
	// For now, return not revoked
	return false, nil
}

// validateTimestamp validates a qualified timestamp
func (s *Service) validateTimestamp(timestamp []byte) (*TimestampInfo, error) {
	// TODO: Implement timestamp validation
	// 1. Parse the timestamp token
	// 2. Verify the TSA signature
	// 3. Validate the TSA certificate
	// 4. Extract the timestamp time

	return &TimestampInfo{
		Time:      time.Now(),
		Authority: "Unknown",
		IsValid:   false,
	}, nil
}

// hashDocument calculates the SHA-256 hash of a document
func (s *Service) hashDocument(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// hashDocumentReader calculates hash from a reader
func (s *Service) hashDocumentReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// isQualifiedCertificate checks if a certificate is a qualified certificate
func (s *Service) isQualifiedCertificate(cert *x509.Certificate) bool {
	// Check for QC statements extension (OID 1.3.6.1.5.5.7.1.3)
	// This is a simplified check - a full implementation would parse the extension

	for _, ext := range cert.Extensions {
		if ext.Id.String() == "1.3.6.1.5.5.7.1.3" {
			return true
		}
	}
	return false
}

// getKeyUsageStrings converts key usage bits to strings
func (s *Service) getKeyUsageStrings(usage x509.KeyUsage) []string {
	var usages []string
	if usage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "digitalSignature")
	}
	if usage&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "nonRepudiation")
	}
	if usage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "keyEncipherment")
	}
	if usage&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "dataEncipherment")
	}
	if usage&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "keyAgreement")
	}
	if usage&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "keyCertSign")
	}
	if usage&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "cRLSign")
	}
	return usages
}

// LoadATrustRootCertificates loads A-Trust root certificates
func (s *Service) LoadATrustRootCertificates() error {
	// A-Trust root certificate PEM data would be embedded here
	// or loaded from a file/configuration

	// Example of how to add certificates:
	// pemData := []byte(`-----BEGIN CERTIFICATE-----...`)
	// if !s.trustedCAs.AppendCertsFromPEM(pemData) {
	//     return fmt.Errorf("failed to parse A-Trust root certificate")
	// }

	return nil
}
