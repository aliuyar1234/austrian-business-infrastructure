package account

import (
	"context"
	"errors"
	"time"

	"austrian-business-infrastructure/internal/account/types"
	"github.com/google/uuid"
)

var (
	ErrDuplicateTID       = errors.New("TID already exists for this tenant")
	ErrInvalidCredentials = errors.New("invalid credentials for account type")
	ErrTestRateLimited    = errors.New("connection test rate limited, try again later")
)

// Connector defines the interface for testing external service connections
type Connector interface {
	TestConnection(ctx context.Context, creds interface{}) (*ConnectionTestResult, error)
}

// ConnectionTestResult represents the result of a connection test
type ConnectionTestResult struct {
	Success      bool
	DurationMs   int
	ErrorCode    string
	ErrorMessage string
}

// Service handles account business logic
type Service struct {
	repo       *Repository
	encryptor  *Encryptor
	connectors map[string]Connector
}

// NewService creates a new account service
func NewService(repo *Repository, encryptionKey []byte) (*Service, error) {
	enc, err := NewEncryptor(encryptionKey)
	if err != nil {
		return nil, err
	}

	return &Service{
		repo:       repo,
		encryptor:  enc,
		connectors: make(map[string]Connector),
	}, nil
}

// RegisterConnector registers a connector for an account type
func (s *Service) RegisterConnector(accountType string, connector Connector) {
	s.connectors[accountType] = connector
}

// CreateAccountInput defines input for creating an account
type CreateAccountInput struct {
	TenantID    uuid.UUID
	Name        string
	Type        string
	Credentials interface{}
}

// CreateAccount creates a new account with encrypted credentials
func (s *Service) CreateAccount(ctx context.Context, input *CreateAccountInput) (*Account, error) {
	// Validate account type
	if err := ValidateAccountType(input.Type); err != nil {
		return nil, err
	}

	// Validate credentials based on type
	if err := s.validateCredentials(input.Type, input.Credentials); err != nil {
		return nil, err
	}

	// Check for duplicate TID for FO accounts
	if input.Type == AccountTypeFinanzOnline {
		creds := input.Credentials.(*types.FinanzOnlineCredentials)
		exists, err := s.repo.CheckDuplicateTID(ctx, input.TenantID, creds.TID, nil)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrDuplicateTID
		}
	}

	// Encrypt credentials
	credBytes, iv, err := s.encryptor.EncryptJSON(input.Credentials)
	if err != nil {
		return nil, err
	}

	account := &Account{
		TenantID:      input.TenantID,
		Name:          input.Name,
		Type:          input.Type,
		Credentials:   credBytes,
		CredentialsIV: iv,
	}

	created, err := s.repo.Create(ctx, account)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// GetAccount retrieves an account by ID
func (s *Service) GetAccount(ctx context.Context, id, tenantID uuid.UUID) (*Account, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// GetAccountWithCredentials retrieves an account and decrypts credentials
func (s *Service) GetAccountWithCredentials(ctx context.Context, id, tenantID uuid.UUID) (*Account, interface{}, error) {
	account, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}

	// Decrypt credentials
	creds, err := s.decryptCredentials(account)
	if err != nil {
		return nil, nil, err
	}

	return account, creds, nil
}

// GetAccountWithMaskedCredentials retrieves an account with masked credentials
func (s *Service) GetAccountWithMaskedCredentials(ctx context.Context, id, tenantID uuid.UUID) (*Account, interface{}, error) {
	account, creds, err := s.GetAccountWithCredentials(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}

	masked := types.MaskCredentials(account.Type, creds)
	return account, masked, nil
}

// ListAccounts retrieves accounts with filtering
func (s *Service) ListAccounts(ctx context.Context, filter ListFilter) ([]*Account, int, error) {
	return s.repo.List(ctx, filter)
}

// UpdateAccount updates account metadata (not credentials)
func (s *Service) UpdateAccount(ctx context.Context, account *Account) error {
	return s.repo.Update(ctx, account)
}

// UpdateCredentials updates account credentials
func (s *Service) UpdateCredentials(ctx context.Context, id, tenantID uuid.UUID, accountType string, creds interface{}) error {
	// Validate credentials
	if err := s.validateCredentials(accountType, creds); err != nil {
		return err
	}

	// Encrypt new credentials
	credBytes, iv, err := s.encryptor.EncryptJSON(creds)
	if err != nil {
		return err
	}

	return s.repo.UpdateCredentials(ctx, id, tenantID, credBytes, iv)
}

// DeleteAccount soft-deletes an account
func (s *Service) DeleteAccount(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.repo.SoftDelete(ctx, id, tenantID)
}

// ForceDeleteAccount permanently deletes an account (GDPR)
func (s *Service) ForceDeleteAccount(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.repo.HardDelete(ctx, id, tenantID)
}

// TestConnection tests the connection for an account
func (s *Service) TestConnection(ctx context.Context, id, tenantID uuid.UUID) (*ConnectionTest, error) {
	// Get account with credentials
	account, creds, err := s.GetAccountWithCredentials(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Get connector for account type
	connector, ok := s.connectors[account.Type]
	if !ok {
		// No connector registered, return mock success for now
		return s.saveTestResult(ctx, account.ID, &ConnectionTestResult{
			Success:    true,
			DurationMs: 0,
		})
	}

	// Run connection test
	start := time.Now()
	result, err := connector.TestConnection(ctx, creds)
	if err != nil {
		result = &ConnectionTestResult{
			Success:      false,
			DurationMs:   int(time.Since(start).Milliseconds()),
			ErrorMessage: err.Error(),
		}
	} else {
		result.DurationMs = int(time.Since(start).Milliseconds())
	}

	// Save test result and update account status
	return s.saveTestResult(ctx, account.ID, result)
}

func (s *Service) saveTestResult(ctx context.Context, accountID uuid.UUID, result *ConnectionTestResult) (*ConnectionTest, error) {
	// Save connection test
	test := &ConnectionTest{
		AccountID:    accountID,
		Success:      result.Success,
		DurationMs:   &result.DurationMs,
	}

	if result.ErrorCode != "" {
		test.ErrorCode = &result.ErrorCode
	}
	if result.ErrorMessage != "" {
		test.ErrorMessage = &result.ErrorMessage
	}

	saved, err := s.repo.SaveConnectionTest(ctx, test)
	if err != nil {
		return nil, err
	}

	// Update account status
	var status string
	var errMsg *string
	if result.Success {
		status = "verified"
	} else {
		status = "error"
		if result.ErrorMessage != "" {
			errMsg = &result.ErrorMessage
		}
	}

	if err := s.repo.UpdateStatus(ctx, accountID, status, errMsg); err != nil {
		return saved, err
	}

	return saved, nil
}

func (s *Service) validateCredentials(accountType string, creds interface{}) error {
	switch accountType {
	case AccountTypeFinanzOnline:
		c, ok := creds.(*types.FinanzOnlineCredentials)
		if !ok {
			return ErrInvalidCredentials
		}
		return ValidateFinanzOnlineCredentials(c.TID, c.BenID, c.PIN)

	case AccountTypeELDA:
		c, ok := creds.(*types.ELDACredentials)
		if !ok {
			return ErrInvalidCredentials
		}
		return ValidateELDACredentials(c.DienstgeberNr, c.PIN, c.CertificatePath)

	case AccountTypeFirmenbuch:
		c, ok := creds.(*types.FirmenbuchCredentials)
		if !ok {
			return ErrInvalidCredentials
		}
		return ValidateFirmenbuchCredentials(c.Username, c.Password)

	default:
		return ErrInvalidAccountType
	}
}

func (s *Service) decryptCredentials(account *Account) (interface{}, error) {
	plaintext, err := s.encryptor.Decrypt(account.Credentials, account.CredentialsIV)
	if err != nil {
		return nil, err
	}

	return types.UnmarshalCredentials(account.Type, plaintext)
}

// GetConnectionTests retrieves connection test history
func (s *Service) GetConnectionTests(ctx context.Context, accountID uuid.UUID, limit int) ([]*ConnectionTest, error) {
	return s.repo.GetConnectionTests(ctx, accountID, limit)
}
