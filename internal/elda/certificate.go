package elda

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/pkcs12"
)

var (
	ErrCertificateNotFound   = errors.New("certificate not found")
	ErrCertificateExpired    = errors.New("certificate has expired")
	ErrCertificateInvalid    = errors.New("certificate is invalid")
	ErrCertificatePassword   = errors.New("invalid certificate password")
	ErrCertificateLoad       = errors.New("failed to load certificate")
)

// Certificate represents an ELDA certificate with metadata
type Certificate struct {
	TLSCert   *tls.Certificate
	X509Cert  *x509.Certificate
	Path      string
	ExpiresAt time.Time
	Subject   string
	Issuer    string
	Serial    string
}

// CertificateManager handles ELDA certificate operations
type CertificateManager struct {
	mu    sync.RWMutex
	cache map[string]*Certificate
}

// NewCertificateManager creates a new certificate manager
func NewCertificateManager() *CertificateManager {
	return &CertificateManager{
		cache: make(map[string]*Certificate),
	}
}

// LoadFromFile loads a certificate from a PFX/P12 file
func (m *CertificateManager) LoadFromFile(path, password string) (*Certificate, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCertificateNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrCertificateLoad, err)
	}

	return m.LoadFromBytes(data, password, path)
}

// LoadFromBytes loads a certificate from PFX/P12 bytes
func (m *CertificateManager) LoadFromBytes(data []byte, password, path string) (*Certificate, error) {
	// Try PKCS12 first (PFX/P12 files)
	privateKey, x509Cert, err := pkcs12.Decode(data, password)
	if err != nil {
		// Try PEM format
		return m.loadPEM(data, password, path)
	}

	// Create TLS certificate
	tlsCert := &tls.Certificate{
		Certificate: [][]byte{x509Cert.Raw},
		PrivateKey:  privateKey,
		Leaf:        x509Cert,
	}

	cert := &Certificate{
		TLSCert:   tlsCert,
		X509Cert:  x509Cert,
		Path:      path,
		ExpiresAt: x509Cert.NotAfter,
		Subject:   x509Cert.Subject.CommonName,
		Issuer:    x509Cert.Issuer.CommonName,
		Serial:    x509Cert.SerialNumber.String(),
	}

	// Validate certificate
	if err := m.validate(cert); err != nil {
		return nil, err
	}

	// Cache the certificate
	m.mu.Lock()
	m.cache[path] = cert
	m.mu.Unlock()

	return cert, nil
}

// loadPEM loads a certificate from PEM format
func (m *CertificateManager) loadPEM(data []byte, password, path string) (*Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("%w: not a valid PEM or PKCS12 file", ErrCertificateInvalid)
	}

	var keyData []byte
	if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck // deprecated but still works
		var err error
		keyData, err = x509.DecryptPEMBlock(block, []byte(password)) //nolint:staticcheck
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCertificatePassword, err)
		}
	} else {
		keyData = block.Bytes
	}

	// Parse the certificate
	x509Cert, err := x509.ParseCertificate(keyData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCertificateInvalid, err)
	}

	cert := &Certificate{
		X509Cert:  x509Cert,
		Path:      path,
		ExpiresAt: x509Cert.NotAfter,
		Subject:   x509Cert.Subject.CommonName,
		Issuer:    x509Cert.Issuer.CommonName,
		Serial:    x509Cert.SerialNumber.String(),
	}

	// Validate certificate
	if err := m.validate(cert); err != nil {
		return nil, err
	}

	return cert, nil
}

// validate checks if a certificate is valid
func (m *CertificateManager) validate(cert *Certificate) error {
	now := time.Now()

	// Check expiry
	if now.After(cert.ExpiresAt) {
		return ErrCertificateExpired
	}

	// Check not-before date
	if cert.X509Cert != nil && now.Before(cert.X509Cert.NotBefore) {
		return fmt.Errorf("%w: certificate not yet valid", ErrCertificateInvalid)
	}

	return nil
}

// GetCached returns a cached certificate if available
func (m *CertificateManager) GetCached(path string) (*Certificate, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cert, ok := m.cache[path]
	if !ok {
		return nil, false
	}

	// Check if still valid
	if time.Now().After(cert.ExpiresAt) {
		return nil, false
	}

	return cert, true
}

// ClearCache removes all cached certificates
func (m *CertificateManager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cache = make(map[string]*Certificate)
}

// RemoveFromCache removes a specific certificate from cache
func (m *CertificateManager) RemoveFromCache(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cache, path)
}

// CertificateStatus represents the status of a certificate
type CertificateStatus struct {
	Valid           bool      `json:"valid"`
	ExpiresAt       time.Time `json:"expires_at"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	Warning         string    `json:"warning,omitempty"`
}

// GetStatus returns the status of a certificate
func (c *Certificate) GetStatus(warnDays int) *CertificateStatus {
	now := time.Now()
	daysUntil := int(c.ExpiresAt.Sub(now).Hours() / 24)

	status := &CertificateStatus{
		Valid:           now.Before(c.ExpiresAt),
		ExpiresAt:       c.ExpiresAt,
		DaysUntilExpiry: daysUntil,
		Subject:         c.Subject,
		Issuer:          c.Issuer,
	}

	if daysUntil < 0 {
		status.Warning = "Certificate has expired"
	} else if daysUntil <= warnDays {
		status.Warning = fmt.Sprintf("Certificate expires in %d days", daysUntil)
	}

	return status
}

// IsExpiringSoon checks if the certificate expires within the given days
func (c *Certificate) IsExpiringSoon(days int) bool {
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).After(c.ExpiresAt)
}

// TLSConfig returns a TLS configuration using this certificate
func (c *Certificate) TLSConfig() *tls.Config {
	if c.TLSCert == nil {
		return nil
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*c.TLSCert},
		MinVersion:   tls.VersionTLS12,
	}
}

// ParseCertificateExpiry extracts expiry date from PFX bytes without full parsing
func ParseCertificateExpiry(data []byte, password string) (time.Time, error) {
	_, cert, err := pkcs12.Decode(data, password)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", ErrCertificateLoad, err)
	}
	return cert.NotAfter, nil
}
