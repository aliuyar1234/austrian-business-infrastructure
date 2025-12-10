package uid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"austrian-business-infrastructure/internal/account"
	"austrian-business-infrastructure/internal/account/types"
	"austrian-business-infrastructure/internal/fonws"
	"github.com/google/uuid"
)

var (
	ErrInvalidLevel    = errors.New("level must be 1 or 2")
	ErrAccountNotFound = errors.New("account not found")
	ErrDailyLimit      = errors.New("daily validation limit exceeded")
	ErrInvalidUID      = errors.New("invalid UID format")
)

// Default cache duration for UID validations (24 hours)
const DefaultCacheDuration = 24 * time.Hour

// Daily limit per tenant
const DailyValidationLimit = 1000

// Service handles UID validation business logic
type Service struct {
	repo           *Repository
	accountService *account.Service
	fonwsClient    *fonws.Client
	cacheDuration  time.Duration
}

// NewService creates a new UID validation service
func NewService(repo *Repository, accountService *account.Service) *Service {
	return &Service{
		repo:           repo,
		accountService: accountService,
		fonwsClient:    fonws.NewClient(),
		cacheDuration:  DefaultCacheDuration,
	}
}

// SetCacheDuration sets the cache duration for UID validations
func (s *Service) SetCacheDuration(d time.Duration) {
	s.cacheDuration = d
}

// Validate validates a UID
func (s *Service) Validate(ctx context.Context, tenantID, userID uuid.UUID, input *ValidateInput) (*Validation, error) {
	// Validate level
	if input.Level != Level1 && input.Level != Level2 {
		input.Level = Level1 // Default to level 1
	}

	// Normalize UID
	uid := strings.ToUpper(strings.TrimSpace(input.UID))

	// Check format first
	formatResult := fonws.ValidateUIDFormat(uid)
	if !formatResult.Valid {
		return s.createValidation(ctx, tenantID, userID, input.AccountID, uid, formatResult.CountryCode, false, input.Level, nil, formatResult.Error)
	}

	// Check cache
	cached, err := s.repo.GetRecentByUID(ctx, tenantID, uid, s.cacheDuration)
	if err != nil {
		return nil, err
	}
	if cached != nil && cached.Level >= input.Level {
		// Return cached result
		return cached, nil
	}

	// Check daily limit
	count, err := s.repo.CountToday(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if count >= DailyValidationLimit {
		return nil, ErrDailyLimit
	}

	// Verify account exists and get credentials
	_, creds, err := s.accountService.GetAccountWithCredentials(ctx, input.AccountID, tenantID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	foCreds, ok := creds.(*types.FinanzOnlineCredentials)
	if !ok {
		return nil, errors.New("invalid account credentials")
	}

	// Login to FinanzOnline
	sessionService := fonws.NewSessionService(s.fonwsClient)
	session, err := sessionService.Login(foCreds.TID, foCreds.BenID, foCreds.PIN)
	if err != nil {
		return nil, fmt.Errorf("failed to login to FinanzOnline: %w", err)
	}
	defer sessionService.Logout(session)

	// Validate UID
	uidService := fonws.NewUIDService(s.fonwsClient)
	result, err := uidService.Validate(session.Token, foCreds.TID, foCreds.BenID, uid, input.Level)
	if err != nil {
		return nil, fmt.Errorf("UID validation failed: %w", err)
	}

	// Store result
	return s.createValidationFromResult(ctx, tenantID, userID, input.AccountID, input.Level, result)
}

// ValidateBatch validates multiple UIDs
func (s *Service) ValidateBatch(ctx context.Context, tenantID, userID uuid.UUID, input *ValidateBatchInput) ([]*Validation, error) {
	// Validate level
	if input.Level != Level1 && input.Level != Level2 {
		input.Level = Level1
	}

	// Check daily limit
	count, err := s.repo.CountToday(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if count+len(input.UIDs) > DailyValidationLimit {
		return nil, ErrDailyLimit
	}

	// Verify account exists and get credentials
	_, creds, err := s.accountService.GetAccountWithCredentials(ctx, input.AccountID, tenantID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	foCreds, ok := creds.(*types.FinanzOnlineCredentials)
	if !ok {
		return nil, errors.New("invalid account credentials")
	}

	// Login to FinanzOnline
	sessionService := fonws.NewSessionService(s.fonwsClient)
	session, err := sessionService.Login(foCreds.TID, foCreds.BenID, foCreds.PIN)
	if err != nil {
		return nil, fmt.Errorf("failed to login to FinanzOnline: %w", err)
	}
	defer sessionService.Logout(session)

	uidService := fonws.NewUIDService(s.fonwsClient)

	var validations []*Validation
	for _, uid := range input.UIDs {
		uid = strings.ToUpper(strings.TrimSpace(uid))

		// Check cache first
		cached, _ := s.repo.GetRecentByUID(ctx, tenantID, uid, s.cacheDuration)
		if cached != nil && cached.Level >= input.Level {
			validations = append(validations, cached)
			continue
		}

		// Check format
		formatResult := fonws.ValidateUIDFormat(uid)
		if !formatResult.Valid {
			v, _ := s.createValidation(ctx, tenantID, userID, input.AccountID, uid, formatResult.CountryCode, false, input.Level, nil, formatResult.Error)
			if v != nil {
				validations = append(validations, v)
			}
			continue
		}

		// Validate against FO
		result, err := uidService.Validate(session.Token, foCreds.TID, foCreds.BenID, uid, input.Level)
		if err != nil {
			v, _ := s.createValidation(ctx, tenantID, userID, input.AccountID, uid, formatResult.CountryCode, false, input.Level, nil, err.Error())
			if v != nil {
				validations = append(validations, v)
			}
			continue
		}

		v, _ := s.createValidationFromResult(ctx, tenantID, userID, input.AccountID, input.Level, result)
		if v != nil {
			validations = append(validations, v)
		}
	}

	return validations, nil
}

// ValidateFormat validates a UID format without querying FinanzOnline
func (s *Service) ValidateFormat(uid string) *FormatValidationResult {
	uid = strings.ToUpper(strings.TrimSpace(uid))
	result := fonws.ValidateUIDFormat(uid)

	return &FormatValidationResult{
		UID:         uid,
		Valid:       result.Valid,
		CountryCode: result.CountryCode,
		Error:       result.Error,
	}
}

// Get retrieves a validation by ID
func (s *Service) Get(ctx context.Context, id, tenantID uuid.UUID) (*Validation, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// List lists validations with filtering
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*Validation, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.List(ctx, filter)
}

// Helper methods

func (s *Service) createValidation(ctx context.Context, tenantID, userID, accountID uuid.UUID, uid, countryCode string, valid bool, level int, result *fonws.UIDValidationResult, errorMsg string) (*Validation, error) {
	v := &Validation{
		TenantID:    tenantID,
		UID:         uid,
		CountryCode: countryCode,
		Valid:       valid,
		Level:       level,
		Source:      "finanzonline",
		ValidatedBy: &userID,
		AccountID:   &accountID,
	}

	if errorMsg != "" {
		v.ErrorMessage = &errorMsg
	}

	return s.repo.Create(ctx, v)
}

func (s *Service) createValidationFromResult(ctx context.Context, tenantID, userID, accountID uuid.UUID, level int, result *fonws.UIDValidationResult) (*Validation, error) {
	v := &Validation{
		TenantID:    tenantID,
		UID:         result.UID,
		CountryCode: result.CountryCode,
		Valid:       result.Valid,
		Level:       level,
		Source:      result.Source,
		ValidatedBy: &userID,
		AccountID:   &accountID,
	}

	if result.CompanyName != "" {
		v.CompanyName = &result.CompanyName
	}
	if result.Address.Street != "" {
		v.Street = &result.Address.Street
	}
	if result.Address.PostCode != "" {
		v.PostCode = &result.Address.PostCode
	}
	if result.Address.City != "" {
		v.City = &result.Address.City
	}
	if result.Address.Country != "" {
		v.Country = &result.Address.Country
	}
	if result.ErrorCode != 0 {
		v.ErrorCode = &result.ErrorCode
	}
	if result.ErrorMessage != "" {
		v.ErrorMessage = &result.ErrorMessage
	}

	return s.repo.Create(ctx, v)
}

// ExportCSV exports validations to CSV format
func (s *Service) ExportCSV(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]byte, error) {
	// Get all matching validations
	filter.Limit = 10000 // Max export limit
	validations, _, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Convert to fonws results for CSV export
	var results []*fonws.UIDValidationResult
	for _, v := range validations {
		r := &fonws.UIDValidationResult{
			UID:         v.UID,
			Valid:       v.Valid,
			CountryCode: v.CountryCode,
			Source:      v.Source,
			QueryTime:   v.ValidatedAt,
		}
		if v.CompanyName != nil {
			r.CompanyName = *v.CompanyName
		}
		if v.Street != nil {
			r.Address.Street = *v.Street
		}
		if v.PostCode != nil {
			r.Address.PostCode = *v.PostCode
		}
		if v.City != nil {
			r.Address.City = *v.City
		}
		if v.ErrorMessage != nil {
			r.ErrorMessage = *v.ErrorMessage
		}
		results = append(results, r)
	}

	return fonws.WriteUIDResultsCSV(results)
}
