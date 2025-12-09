package zm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/austrian-business-infrastructure/fo/internal/account"
	"github.com/austrian-business-infrastructure/fo/internal/account/types"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/google/uuid"
)

var (
	ErrInvalidQuarter      = errors.New("quarter must be between 1 and 4")
	ErrInvalidYear         = errors.New("year must be between 2000 and 2100")
	ErrSubmissionNotDraft  = errors.New("submission is not in draft status")
	ErrAccountNotFound     = errors.New("account not found")
	ErrValidationFailed    = errors.New("validation failed")
	ErrSubmissionFailed    = errors.New("submission to FinanzOnline failed")
	ErrNoEntries           = errors.New("ZM must have at least one entry")
)

// Service handles ZM business logic
type Service struct {
	repo           *Repository
	accountService *account.Service
	fonwsClient    *fonws.Client
}

// NewService creates a new ZM service
func NewService(repo *Repository, accountService *account.Service) *Service {
	return &Service{
		repo:           repo,
		accountService: accountService,
		fonwsClient:    fonws.NewClient(),
	}
}

// Create creates a new ZM submission
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, input *CreateSubmissionInput) (*Submission, error) {
	// Validate period
	if err := s.validatePeriod(input.PeriodYear, input.PeriodQuarter); err != nil {
		return nil, err
	}

	// Validate entries
	if len(input.Entries) == 0 {
		return nil, ErrNoEntries
	}

	// Verify account exists and belongs to tenant
	acc, err := s.accountService.GetAccount(ctx, input.AccountID, tenantID)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	if acc.Type != account.AccountTypeFinanzOnline {
		return nil, errors.New("account must be a FinanzOnline account")
	}

	// Check for duplicate
	exists, err := s.repo.CheckDuplicatePeriod(ctx, tenantID, input.AccountID, input.PeriodYear, input.PeriodQuarter, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicatePeriod
	}

	// Calculate total amount
	var totalAmount int64
	for _, e := range input.Entries {
		totalAmount += e.Amount
	}

	// Serialize entries
	entriesJSON, err := json.Marshal(input.Entries)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize entries: %w", err)
	}

	submission := &Submission{
		TenantID:      tenantID,
		AccountID:     input.AccountID,
		PeriodYear:    input.PeriodYear,
		PeriodQuarter: input.PeriodQuarter,
		Entries:       entriesJSON,
		EntryCount:    len(input.Entries),
		TotalAmount:   totalAmount,
	}

	return s.repo.Create(ctx, submission)
}

// Get retrieves a submission by ID
func (s *Service) Get(ctx context.Context, id, tenantID uuid.UUID) (*Submission, error) {
	return s.repo.GetByID(ctx, id, tenantID)
}

// List lists submissions with filtering
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*Submission, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.List(ctx, filter)
}

// Update updates a submission (only for drafts)
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, input *UpdateSubmissionInput) (*Submission, error) {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if submission.Status != StatusDraft {
		return nil, ErrSubmissionNotDraft
	}

	// Validate entries
	if len(input.Entries) == 0 {
		return nil, ErrNoEntries
	}

	// Calculate total amount
	var totalAmount int64
	for _, e := range input.Entries {
		totalAmount += e.Amount
	}

	// Serialize entries
	entriesJSON, err := json.Marshal(input.Entries)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize entries: %w", err)
	}

	submission.Entries = entriesJSON
	submission.EntryCount = len(input.Entries)
	submission.TotalAmount = totalAmount
	submission.ValidationStatus = "pending"
	submission.ValidationErrors = nil

	if err := s.repo.Update(ctx, submission); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id, tenantID)
}

// Delete deletes a submission (only for drafts)
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return err
	}

	if submission.Status != StatusDraft {
		return ErrSubmissionNotDraft
	}

	return s.repo.Delete(ctx, id, tenantID)
}

// Validate validates a ZM submission
func (s *Service) Validate(ctx context.Context, id, tenantID uuid.UUID) (*Submission, error) {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if submission.Status != StatusDraft && submission.Status != StatusValidated {
		return nil, errors.New("submission cannot be validated in current status")
	}

	// Parse entries
	var entries []Entry
	if err := json.Unmarshal(submission.Entries, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse entries: %w", err)
	}

	// Convert to fonws format and validate
	zm := s.entriesToFonwsZM(submission.PeriodYear, submission.PeriodQuarter, entries)
	validationErr := zm.Validate()

	if validationErr != nil {
		validationErrors, _ := json.Marshal(map[string]string{"error": validationErr.Error()})
		submission.ValidationStatus = "failed"
		submission.ValidationErrors = validationErrors
		submission.Status = StatusDraft
	} else {
		submission.ValidationStatus = "passed"
		submission.ValidationErrors = nil
		submission.Status = StatusValidated
	}

	if err := s.repo.Update(ctx, submission); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id, tenantID)
}

// Submit submits a ZM to FinanzOnline
func (s *Service) Submit(ctx context.Context, id, tenantID, userID uuid.UUID, dryRun bool) (*Submission, error) {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Must be validated or draft
	if submission.Status != StatusDraft && submission.Status != StatusValidated {
		return nil, errors.New("submission must be in draft or validated status")
	}

	// Parse entries
	var entries []Entry
	if err := json.Unmarshal(submission.Entries, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse entries: %w", err)
	}

	// Create fonws ZM struct
	zm := s.entriesToFonwsZM(submission.PeriodYear, submission.PeriodQuarter, entries)

	// Validate first
	if err := zm.Validate(); err != nil {
		validationErrors, _ := json.Marshal(map[string]string{"error": err.Error()})
		submission.ValidationStatus = "failed"
		submission.ValidationErrors = validationErrors
		if updateErr := s.repo.Update(ctx, submission); updateErr != nil {
			return nil, updateErr
		}
		return nil, ErrValidationFailed
	}

	// Generate XML
	xmlContent, err := fonws.GenerateZMXML(zm)
	if err != nil {
		return nil, fmt.Errorf("failed to generate XML: %w", err)
	}

	// Save XML content
	if err := s.repo.SaveXMLContent(ctx, id, tenantID, xmlContent); err != nil {
		return nil, err
	}

	// If dry run, just validate and return
	if dryRun {
		submission.ValidationStatus = "passed"
		submission.Status = StatusValidated
		if err := s.repo.Update(ctx, submission); err != nil {
			return nil, err
		}
		return s.repo.GetByID(ctx, id, tenantID)
	}

	// Get account credentials
	_, creds, err := s.accountService.GetAccountWithCredentials(ctx, submission.AccountID, tenantID)
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

	// Submit to FinanzOnline
	uploadService := fonws.NewFileUploadService(s.fonwsClient)
	result, err := uploadService.SubmitZM(session.Token, foCreds.TID, foCreds.BenID, zm)

	var status string
	var foRef string
	var respCode int
	var respMsg string

	if err != nil || !result.Success {
		status = StatusError
		if result != nil {
			respMsg = result.Message
		} else {
			respMsg = err.Error()
		}
	} else {
		status = StatusSubmitted
		foRef = result.Reference
		respMsg = result.Message
	}

	// Update submission with result
	if err := s.repo.UpdateSubmissionResult(ctx, id, tenantID, foRef, respCode, respMsg, status); err != nil {
		return nil, err
	}

	// Set submitted by
	if err := s.repo.SetSubmittedBy(ctx, id, tenantID, userID); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id, tenantID)
}

// GetXML retrieves the generated XML for a submission
func (s *Service) GetXML(ctx context.Context, id, tenantID uuid.UUID) ([]byte, error) {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if len(submission.XMLContent) > 0 {
		return submission.XMLContent, nil
	}

	// Generate XML on the fly
	var entries []Entry
	if err := json.Unmarshal(submission.Entries, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse entries: %w", err)
	}

	zm := s.entriesToFonwsZM(submission.PeriodYear, submission.PeriodQuarter, entries)
	return fonws.GenerateZMXML(zm)
}

// ImportCSV imports ZM entries from CSV
func (s *Service) ImportCSV(ctx context.Context, tenantID, accountID uuid.UUID, year, quarter int, csvData []byte) (*Submission, error) {
	entries, err := fonws.ParseZMFromCSV(csvData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Convert to our format
	zmEntries := make([]Entry, len(entries))
	for i, e := range entries {
		zmEntries[i] = Entry{
			PartnerUID:   e.PartnerUID,
			CountryCode:  e.CountryCode,
			DeliveryType: string(e.DeliveryType),
			Amount:       e.Amount,
		}
	}

	input := &CreateSubmissionInput{
		AccountID:     accountID,
		PeriodYear:    year,
		PeriodQuarter: quarter,
		Entries:       zmEntries,
	}

	return s.Create(ctx, tenantID, input)
}

// Helper methods

func (s *Service) validatePeriod(year, quarter int) error {
	if year < 2000 || year > 2100 {
		return ErrInvalidYear
	}
	if quarter < 1 || quarter > 4 {
		return ErrInvalidQuarter
	}
	return nil
}

func (s *Service) entriesToFonwsZM(year, quarter int, entries []Entry) *fonws.ZM {
	zm := fonws.NewZM(year, quarter)
	for _, e := range entries {
		zm.Entries = append(zm.Entries, fonws.ZMEntry{
			PartnerUID:   e.PartnerUID,
			CountryCode:  e.CountryCode,
			DeliveryType: fonws.ZMDeliveryType(e.DeliveryType),
			Amount:       e.Amount,
		})
	}
	return zm
}

// ParseEntries parses the JSON entries from a submission
func ParseEntries(rawData json.RawMessage) ([]Entry, error) {
	var entries []Entry
	if err := json.Unmarshal(rawData, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}
