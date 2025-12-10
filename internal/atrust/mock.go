package atrust

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// MockClient is a mock implementation of the A-Trust client for testing
type MockClient struct {
	mu sync.Mutex

	// Configuration
	FailureRate     float64 // Probability of request failure (0.0 to 1.0)
	SimulateLatency time.Duration

	// Tracking
	SignCalls      []SignRequest
	BatchSignCalls []BatchSignRequest
	TimestampCalls []TimestampRequest

	// Mock data
	MockCertificate     string
	MockCertificateInfo *CertificateInfo

	// Error injection
	ForceError *ATrustError
}

// NewMockClient creates a new mock A-Trust client
func NewMockClient() *MockClient {
	return &MockClient{
		MockCertificate: base64.StdEncoding.EncodeToString([]byte("MOCK_CERTIFICATE_DATA")),
		MockCertificateInfo: &CertificateInfo{
			Subject:      "CN=Max Mustermann, O=Test Company GmbH, C=AT",
			SubjectCN:    "Max Mustermann",
			Issuer:       "CN=A-Trust Qual-01, O=A-Trust, C=AT",
			IssuerCN:     "A-Trust Qual-01",
			SerialNumber: "1234567890ABCDEF",
			ValidFrom:    time.Now().Add(-365 * 24 * time.Hour),
			ValidTo:      time.Now().Add(365 * 24 * time.Hour),
			IsQualified:  true,
			KeyUsage:     []string{"digitalSignature", "nonRepudiation"},
		},
	}
}

// Sign implements the mock signing operation
func (m *MockClient) Sign(ctx context.Context, req *SignRequest) (*SignResponse, error) {
	m.mu.Lock()
	m.SignCalls = append(m.SignCalls, *req)
	m.mu.Unlock()

	// Simulate latency
	if m.SimulateLatency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.SimulateLatency):
		}
	}

	// Check for forced error
	if m.ForceError != nil {
		return nil, m.ForceError
	}

	// Validate hash
	if _, err := hex.DecodeString(req.DocumentHash); err != nil {
		return nil, &ATrustError{
			StatusCode: 400,
			Code:       ErrCodeInvalidHash,
			Message:    "Invalid document hash format",
		}
	}

	// Generate mock signature
	signature := generateMockSignature(req.DocumentHash)
	timestamp := generateMockTimestamp()

	return &SignResponse{
		Signature:          signature,
		SignedAt:           time.Now(),
		Certificate:        m.MockCertificate,
		CertificateChain:   []string{m.MockCertificate},
		Timestamp:          timestamp,
		TimestampAuthority: "A-Trust Timestamp Service",
	}, nil
}

// BatchSign implements the mock batch signing operation
func (m *MockClient) BatchSign(ctx context.Context, req *BatchSignRequest) (*BatchSignResponse, error) {
	m.mu.Lock()
	m.BatchSignCalls = append(m.BatchSignCalls, *req)
	m.mu.Unlock()

	// Simulate latency
	if m.SimulateLatency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.SimulateLatency):
		}
	}

	// Check for forced error
	if m.ForceError != nil {
		return nil, m.ForceError
	}

	if len(req.Documents) > 100 {
		return nil, &ATrustError{
			StatusCode: 400,
			Code:       ErrCodeBatchTooLarge,
			Message:    "Batch size exceeds maximum of 100 documents",
		}
	}

	results := make([]BatchSignResult, len(req.Documents))
	for i, doc := range req.Documents {
		// Validate hash
		if _, err := hex.DecodeString(doc.DocumentHash); err != nil {
			results[i] = BatchSignResult{
				ID:        doc.ID,
				Success:   false,
				Error:     "Invalid document hash format",
				ErrorCode: ErrCodeInvalidHash,
			}
			continue
		}

		// Generate mock signature
		signature := generateMockSignature(doc.DocumentHash)
		timestamp := generateMockTimestamp()

		results[i] = BatchSignResult{
			ID:          doc.ID,
			Success:     true,
			Signature:   signature,
			Certificate: m.MockCertificate,
			Timestamp:   timestamp,
		}
	}

	return &BatchSignResponse{
		Results:  results,
		SignedAt: time.Now(),
	}, nil
}

// GetTimestamp implements the mock timestamp operation
func (m *MockClient) GetTimestamp(ctx context.Context, req *TimestampRequest) (*TimestampResponse, error) {
	m.mu.Lock()
	m.TimestampCalls = append(m.TimestampCalls, *req)
	m.mu.Unlock()

	// Simulate latency
	if m.SimulateLatency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.SimulateLatency):
		}
	}

	// Check for forced error
	if m.ForceError != nil {
		return nil, m.ForceError
	}

	// Validate hash
	if _, err := hex.DecodeString(req.Hash); err != nil {
		return nil, &ATrustError{
			StatusCode: 400,
			Code:       ErrCodeInvalidHash,
			Message:    "Invalid hash format",
		}
	}

	return &TimestampResponse{
		Token:        generateMockTimestamp(),
		Time:         time.Now(),
		Authority:    "A-Trust Timestamp Service (Mock)",
		SerialNumber: generateRandomHex(16),
	}, nil
}

// GetCertificateInfo implements the mock certificate info operation
func (m *MockClient) GetCertificateInfo(ctx context.Context, certID string) (*CertificateInfo, error) {
	// Simulate latency
	if m.SimulateLatency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(m.SimulateLatency):
		}
	}

	// Check for forced error
	if m.ForceError != nil {
		return nil, m.ForceError
	}

	return m.MockCertificateInfo, nil
}

// HealthCheck implements the mock health check
func (m *MockClient) HealthCheck(ctx context.Context) error {
	if m.ForceError != nil {
		return m.ForceError
	}
	return nil
}

// Reset clears all recorded calls
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SignCalls = nil
	m.BatchSignCalls = nil
	m.TimestampCalls = nil
	m.ForceError = nil
}

// GetSignCallCount returns the number of Sign calls made
func (m *MockClient) GetSignCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.SignCalls)
}

// GetBatchSignCallCount returns the number of BatchSign calls made
func (m *MockClient) GetBatchSignCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.BatchSignCalls)
}

// Helper functions

func generateMockSignature(documentHash string) string {
	// Generate a mock PKCS#7 signature
	// In reality this would be a proper signature, but for mock purposes
	// we just generate some random data that looks like a signature
	data := fmt.Sprintf("MOCK_SIGNATURE:%s:%d", documentHash, time.Now().UnixNano())
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func generateMockTimestamp() string {
	// Generate a mock timestamp token
	data := fmt.Sprintf("MOCK_TIMESTAMP:%d", time.Now().UnixNano())
	return base64.StdEncoding.EncodeToString([]byte(data))
}

func generateRandomHex(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Signer interface for dependency injection
type Signer interface {
	Sign(ctx context.Context, req *SignRequest) (*SignResponse, error)
	BatchSign(ctx context.Context, req *BatchSignRequest) (*BatchSignResponse, error)
	GetTimestamp(ctx context.Context, req *TimestampRequest) (*TimestampResponse, error)
	GetCertificateInfo(ctx context.Context, certID string) (*CertificateInfo, error)
	HealthCheck(ctx context.Context) error
}

// Ensure Client and MockClient implement Signer
var _ Signer = (*Client)(nil)
var _ Signer = (*MockClient)(nil)
