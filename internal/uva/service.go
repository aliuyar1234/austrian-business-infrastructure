package uva

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"austrian-business-infrastructure/internal/account"
	"austrian-business-infrastructure/internal/account/types"
	"austrian-business-infrastructure/internal/fonws"
	"github.com/google/uuid"
)

var (
	ErrInvalidPeriodType   = errors.New("invalid period type")
	ErrInvalidMonth        = errors.New("month must be between 1 and 12")
	ErrInvalidQuarter      = errors.New("quarter must be between 1 and 4")
	ErrInvalidYear         = errors.New("year must be between 2000 and 2100")
	ErrSubmissionNotDraft  = errors.New("submission is not in draft status")
	ErrAccountNotFound     = errors.New("account not found")
	ErrValidationFailed    = errors.New("validation failed")
	ErrSubmissionFailed    = errors.New("submission to FinanzOnline failed")
)

// Service handles UVA business logic
type Service struct {
	repo           *Repository
	accountService *account.Service
	fonwsClient    *fonws.Client
}

// NewService creates a new UVA service
func NewService(repo *Repository, accountService *account.Service) *Service {
	return &Service{
		repo:           repo,
		accountService: accountService,
		fonwsClient:    fonws.NewClient(),
	}
}

// Create creates a new UVA submission
func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, input *CreateSubmissionInput) (*Submission, error) {
	// Validate period
	if err := s.validatePeriod(input); err != nil {
		return nil, err
	}

	// Verify account exists and belongs to tenant
	acc, err := s.accountService.GetAccount(ctx, input.AccountID, tenantID)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	if acc.Type != account.AccountTypeFinanzOnline {
		return nil, errors.New("account must be a FinanzOnline account")
	}

	// Get period value
	periodValue := 0
	if input.PeriodType == PeriodTypeMonthly && input.PeriodMonth != nil {
		periodValue = *input.PeriodMonth
	} else if input.PeriodType == PeriodTypeQuarterly && input.PeriodQuarter != nil {
		periodValue = *input.PeriodQuarter
	}

	// Check for duplicate
	exists, err := s.repo.CheckDuplicatePeriod(ctx, tenantID, input.AccountID, input.PeriodYear, input.PeriodType, periodValue, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicatePeriod
	}

	// Calculate KZ095 if not provided
	if input.Data.KZ095 == 0 {
		input.Data.KZ095 = s.calculateKZ095(&input.Data)
	}

	// Serialize data
	dataJSON, err := json.Marshal(input.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
	}

	submission := &Submission{
		TenantID:      tenantID,
		AccountID:     input.AccountID,
		PeriodYear:    input.PeriodYear,
		PeriodMonth:   input.PeriodMonth,
		PeriodQuarter: input.PeriodQuarter,
		PeriodType:    input.PeriodType,
		Data:          dataJSON,
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

	// Calculate KZ095 if not provided
	if input.Data.KZ095 == 0 {
		input.Data.KZ095 = s.calculateKZ095(&input.Data)
	}

	// Serialize data
	dataJSON, err := json.Marshal(input.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize data: %w", err)
	}

	submission.Data = dataJSON
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

// Validate validates a UVA submission
func (s *Service) Validate(ctx context.Context, id, tenantID uuid.UUID) (*Submission, error) {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	if submission.Status != StatusDraft && submission.Status != StatusValidated {
		return nil, errors.New("submission cannot be validated in current status")
	}

	// Parse the data
	var data UVAData
	if err := json.Unmarshal(submission.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse submission data: %w", err)
	}

	// Validate using fonws library
	uva := s.dataToFonwsUVA(submission, &data)
	validationErr := fonws.ValidateUVA(uva)

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

// Submit submits a UVA to FinanzOnline
func (s *Service) Submit(ctx context.Context, id, tenantID, userID uuid.UUID, dryRun bool) (*Submission, error) {
	submission, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Must be validated or draft (will validate first)
	if submission.Status != StatusDraft && submission.Status != StatusValidated {
		return nil, errors.New("submission must be in draft or validated status")
	}

	// Parse the data
	var data UVAData
	if err := json.Unmarshal(submission.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse submission data: %w", err)
	}

	// Create fonws UVA struct
	uva := s.dataToFonwsUVA(submission, &data)

	// Validate first
	if err := fonws.ValidateUVA(uva); err != nil {
		validationErrors, _ := json.Marshal(map[string]string{"error": err.Error()})
		submission.ValidationStatus = "failed"
		submission.ValidationErrors = validationErrors
		if updateErr := s.repo.Update(ctx, submission); updateErr != nil {
			return nil, updateErr
		}
		return nil, ErrValidationFailed
	}

	// Generate XML
	xmlContent, err := fonws.GenerateUVAXML(uva)
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
	resp, err := uploadService.SubmitUVA(session.Token, foCreds.TID, foCreds.BenID, uva)

	var status string
	var foRef string
	var respCode int
	var respMsg string

	if err != nil {
		status = StatusError
		respMsg = err.Error()
		if resp != nil {
			respCode = resp.RC
			respMsg = resp.Msg
		}
	} else {
		status = StatusSubmitted
		foRef = resp.Belegnummer
		respCode = resp.RC
		respMsg = resp.Msg
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
	var data UVAData
	if err := json.Unmarshal(submission.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse submission data: %w", err)
	}

	uva := s.dataToFonwsUVA(submission, &data)
	return fonws.GenerateUVAXML(uva)
}

// Batch operations

// CreateBatch creates a new batch UVA submission
func (s *Service) CreateBatch(ctx context.Context, tenantID, userID uuid.UUID, name string, accountIDs []uuid.UUID, periodYear int, periodType string, periodMonth, periodQuarter *int) (*Batch, error) {
	// Validate period
	input := &CreateSubmissionInput{
		PeriodYear:    periodYear,
		PeriodType:    periodType,
		PeriodMonth:   periodMonth,
		PeriodQuarter: periodQuarter,
	}
	if err := s.validatePeriod(input); err != nil {
		return nil, err
	}

	batch := &Batch{
		TenantID:      tenantID,
		Name:          name,
		PeriodYear:    periodYear,
		PeriodMonth:   periodMonth,
		PeriodQuarter: periodQuarter,
		PeriodType:    periodType,
		TotalCount:    len(accountIDs),
		CreatedBy:     userID,
	}

	return s.repo.CreateBatch(ctx, batch)
}

// GetBatch retrieves a batch by ID
func (s *Service) GetBatch(ctx context.Context, id, tenantID uuid.UUID) (*Batch, error) {
	return s.repo.GetBatchByID(ctx, id, tenantID)
}

// ListBatches lists batches for a tenant
func (s *Service) ListBatches(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Batch, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListBatches(ctx, tenantID, limit, offset)
}

// Helper methods

func (s *Service) validatePeriod(input *CreateSubmissionInput) error {
	if input.PeriodYear < 2000 || input.PeriodYear > 2100 {
		return ErrInvalidYear
	}

	switch input.PeriodType {
	case PeriodTypeMonthly:
		if input.PeriodMonth == nil {
			return errors.New("month is required for monthly period")
		}
		if *input.PeriodMonth < 1 || *input.PeriodMonth > 12 {
			return ErrInvalidMonth
		}
	case PeriodTypeQuarterly:
		if input.PeriodQuarter == nil {
			return errors.New("quarter is required for quarterly period")
		}
		if *input.PeriodQuarter < 1 || *input.PeriodQuarter > 4 {
			return ErrInvalidQuarter
		}
	default:
		return ErrInvalidPeriodType
	}

	return nil
}

func (s *Service) calculateKZ095(data *UVAData) int64 {
	// Tax payable: 20% of KZ017 + 10% of KZ018 + 13% of KZ019 + other taxes
	taxPayable := int64(0)
	taxPayable += data.KZ017 * 20 / 100
	taxPayable += data.KZ018 * 10 / 100
	taxPayable += data.KZ019 * 13 / 100
	taxPayable += data.KZ022 // Import VAT
	taxPayable += data.KZ029 * 20 / 100 // IC acquisitions at 20%

	// Input tax deductions
	inputTax := data.KZ060 + data.KZ065 + data.KZ066 + data.KZ070

	// Result: positive = payment due, negative = refund
	return taxPayable - inputTax
}

func (s *Service) dataToFonwsUVA(submission *Submission, data *UVAData) *fonws.UVA {
	uva := &fonws.UVA{
		Year: submission.PeriodYear,
		KZ000: data.KZ000,
		KZ001: data.KZ001,
		KZ011: data.KZ011,
		KZ017: data.KZ017,
		KZ018: data.KZ018,
		KZ019: data.KZ019,
		KZ020: data.KZ020,
		KZ022: data.KZ022,
		KZ029: data.KZ029,
		KZ060: data.KZ060,
		KZ065: data.KZ065,
		KZ066: data.KZ066,
		KZ070: data.KZ070,
		KZ095: data.KZ095,
	}

	if submission.PeriodType == PeriodTypeMonthly && submission.PeriodMonth != nil {
		uva.Period = fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: *submission.PeriodMonth}
	} else if submission.PeriodType == PeriodTypeQuarterly && submission.PeriodQuarter != nil {
		uva.Period = fonws.UVAPeriod{Type: fonws.PeriodTypeQuarterly, Value: *submission.PeriodQuarter}
	}

	return uva
}

// ParseData parses the JSON data from a submission
func ParseData(rawData json.RawMessage) (*UVAData, error) {
	var data UVAData
	if err := json.Unmarshal(rawData, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
