package atrust

import (
	"time"
)

// SignRequest represents a request to sign a document hash
type SignRequest struct {
	// DocumentHash is the SHA-256 hash of the document to sign (hex encoded)
	DocumentHash string `json:"documentHash"`
	// HashAlgorithm is the algorithm used (always SHA256)
	HashAlgorithm string `json:"hashAlgorithm"`
	// SignerCertID is the certificate ID from ID Austria authentication
	SignerCertID string `json:"signerCertId"`
	// Reason is an optional reason for signing
	Reason string `json:"reason,omitempty"`
	// Location is an optional location of signing
	Location string `json:"location,omitempty"`
	// ContactInfo is optional contact information
	ContactInfo string `json:"contactInfo,omitempty"`
}

// SignResponse represents the response from a signing operation
type SignResponse struct {
	// Signature is the PKCS#7 signature (Base64 encoded)
	Signature string `json:"signature"`
	// SignedAt is the timestamp of signing
	SignedAt time.Time `json:"signedAt"`
	// Certificate is the signer's certificate (Base64 encoded DER)
	Certificate string `json:"certificate"`
	// CertificateChain contains the certificate chain (Base64 encoded DER)
	CertificateChain []string `json:"certificateChain,omitempty"`
	// Timestamp is the qualified timestamp (Base64 encoded)
	Timestamp string `json:"timestamp,omitempty"`
	// TimestampAuthority is the TSA that issued the timestamp
	TimestampAuthority string `json:"timestampAuthority,omitempty"`
}

// BatchSignRequest represents a request to sign multiple documents
type BatchSignRequest struct {
	// Documents is a list of document hashes to sign
	Documents []BatchDocument `json:"documents"`
	// SignerCertID is the certificate ID from ID Austria authentication
	SignerCertID string `json:"signerCertId"`
	// Reason is an optional reason for signing (applied to all)
	Reason string `json:"reason,omitempty"`
}

// BatchDocument represents a single document in a batch sign request
type BatchDocument struct {
	// ID is a client-provided identifier for correlation
	ID string `json:"id"`
	// DocumentHash is the SHA-256 hash of the document (hex encoded)
	DocumentHash string `json:"documentHash"`
}

// BatchSignResponse represents the response from a batch signing operation
type BatchSignResponse struct {
	// Results contains the signing result for each document
	Results []BatchSignResult `json:"results"`
	// SignedAt is the timestamp of the batch signing
	SignedAt time.Time `json:"signedAt"`
}

// BatchSignResult represents the result for a single document in a batch
type BatchSignResult struct {
	// ID is the client-provided identifier
	ID string `json:"id"`
	// Success indicates if signing was successful
	Success bool `json:"success"`
	// Signature is the PKCS#7 signature (Base64 encoded) if successful
	Signature string `json:"signature,omitempty"`
	// Certificate is the signer's certificate (Base64 encoded DER) if successful
	Certificate string `json:"certificate,omitempty"`
	// Timestamp is the qualified timestamp (Base64 encoded) if successful
	Timestamp string `json:"timestamp,omitempty"`
	// Error contains the error message if unsuccessful
	Error string `json:"error,omitempty"`
	// ErrorCode contains the error code if unsuccessful
	ErrorCode string `json:"errorCode,omitempty"`
}

// CertificateInfo contains information about a certificate
type CertificateInfo struct {
	// Subject is the certificate subject (CN, O, etc.)
	Subject string `json:"subject"`
	// SubjectCN is just the Common Name
	SubjectCN string `json:"subjectCN"`
	// Issuer is the certificate issuer
	Issuer string `json:"issuer"`
	// IssuerCN is just the issuer's Common Name
	IssuerCN string `json:"issuerCN"`
	// SerialNumber is the certificate serial number
	SerialNumber string `json:"serialNumber"`
	// ValidFrom is the certificate validity start
	ValidFrom time.Time `json:"validFrom"`
	// ValidTo is the certificate validity end
	ValidTo time.Time `json:"validTo"`
	// IsQualified indicates if this is a qualified certificate
	IsQualified bool `json:"isQualified"`
	// KeyUsage describes the allowed uses
	KeyUsage []string `json:"keyUsage,omitempty"`
}

// TimestampRequest represents a request to get a qualified timestamp
type TimestampRequest struct {
	// Hash is the SHA-256 hash to timestamp (hex encoded)
	Hash string `json:"hash"`
	// HashAlgorithm is the algorithm used (always SHA256)
	HashAlgorithm string `json:"hashAlgorithm"`
}

// TimestampResponse represents a timestamp response
type TimestampResponse struct {
	// Token is the timestamp token (Base64 encoded)
	Token string `json:"token"`
	// Time is the timestamp time
	Time time.Time `json:"time"`
	// Authority is the TSA that issued the timestamp
	Authority string `json:"authority"`
	// SerialNumber is the timestamp serial number
	SerialNumber string `json:"serialNumber"`
}

// ErrorResponse represents an error from the A-Trust API
type ErrorResponse struct {
	// Code is the error code
	Code string `json:"code"`
	// Message is the error message
	Message string `json:"message"`
	// Details contains additional error details
	Details string `json:"details,omitempty"`
}

// Error codes returned by A-Trust
const (
	ErrCodeInvalidHash        = "INVALID_HASH"
	ErrCodeInvalidCertificate = "INVALID_CERTIFICATE"
	ErrCodeCertificateExpired = "CERTIFICATE_EXPIRED"
	ErrCodeCertificateRevoked = "CERTIFICATE_REVOKED"
	ErrCodeSignatureFailed    = "SIGNATURE_FAILED"
	ErrCodeTimestampFailed    = "TIMESTAMP_FAILED"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeRateLimited        = "RATE_LIMITED"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeBatchTooLarge      = "BATCH_TOO_LARGE"
)

// Hash algorithms
const (
	HashAlgoSHA256 = "SHA256"
	HashAlgoSHA384 = "SHA384"
	HashAlgoSHA512 = "SHA512"
)
