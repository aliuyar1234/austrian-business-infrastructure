package mbgm

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/elda"
)

// Service handles mBGM business logic
type Service struct {
	repo        *Repository
	validator   *Validator
	builder     *Builder
	eldaService *elda.MBGMService
	logger      *slog.Logger
}

// ServiceConfig contains configuration for the mBGM service
type ServiceConfig struct {
	Repository  *Repository
	ELDAClient  *elda.Client
	Logger      *slog.Logger
}

// NewService creates a new mBGM service
func NewService(cfg ServiceConfig) *Service {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Service{
		repo:        cfg.Repository,
		validator:   NewValidator(cfg.Repository),
		builder:     NewBuilder(),
		eldaService: elda.NewMBGMService(cfg.ELDAClient),
		logger:      logger,
	}
}

// Create creates a new mBGM from a request
func (s *Service) Create(ctx context.Context, req *elda.MBGMCreateRequest) (*elda.MBGM, error) {
	// Validate the request
	validationResult := ValidateCreateRequest(req)
	if !validationResult.Valid {
		return nil, &ValidationError{Result: validationResult}
	}

	// Create the mBGM
	mbgm := &elda.MBGM{
		ID:            uuid.New(),
		ELDAAccountID: req.ELDAAccountID,
		Year:          req.Year,
		Month:         req.Month,
		Status:        elda.MBGMStatusDraft,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Create positions
	positionen := make([]*elda.MBGMPosition, 0, len(req.Positionen))
	for i, posReq := range req.Positionen {
		pos := s.createPositionFromRequest(mbgm.ID, &posReq, i)
		positionen = append(positionen, pos)
	}
	mbgm.Positionen = positionen

	// Calculate totals
	mbgm.TotalDienstnehmer = len(positionen)
	mbgm.TotalBeitragsgrundlage = s.calculateTotalBeitragsgrundlage(positionen)

	// Save to database
	if err := s.repo.Create(ctx, mbgm); err != nil {
		return nil, fmt.Errorf("failed to create mBGM: %w", err)
	}

	// Save positions
	if err := s.repo.CreatePositions(ctx, positionen); err != nil {
		return nil, fmt.Errorf("failed to create positions: %w", err)
	}

	s.logger.Info("mBGM created",
		"id", mbgm.ID,
		"year", mbgm.Year,
		"month", mbgm.Month,
		"positions", len(positionen))

	return mbgm, nil
}

// createPositionFromRequest creates an MBGMPosition from a request
func (s *Service) createPositionFromRequest(mbgmID uuid.UUID, req *elda.MBGMPositionCreateRequest, index int) *elda.MBGMPosition {
	pos := &elda.MBGMPosition{
		ID:                uuid.New(),
		MBGMID:            mbgmID,
		SVNummer:          req.SVNummer,
		Familienname:      req.Familienname,
		Vorname:           req.Vorname,
		Beitragsgruppe:    req.Beitragsgruppe,
		Beitragsgrundlage: req.Beitragsgrundlage,
		Sonderzahlung:     req.Sonderzahlung,
		Wochenstunden:     req.Wochenstunden,
		IsValid:           true,
		PositionIndex:     index,
		CreatedAt:         time.Now(),
	}

	// Parse dates
	if req.Geburtsdatum != "" {
		if t, err := time.Parse("2006-01-02", req.Geburtsdatum); err == nil {
			pos.Geburtsdatum = &t
		}
	}
	if req.VonDatum != "" {
		if t, err := time.Parse("2006-01-02", req.VonDatum); err == nil {
			pos.VonDatum = &t
		}
	}
	if req.BisDatum != "" {
		if t, err := time.Parse("2006-01-02", req.BisDatum); err == nil {
			pos.BisDatum = &t
		}
	}

	return pos
}

// calculateTotalBeitragsgrundlage calculates the sum of all Beitragsgrundlagen
func (s *Service) calculateTotalBeitragsgrundlage(positionen []*elda.MBGMPosition) float64 {
	var total float64
	for _, pos := range positionen {
		total += pos.Beitragsgrundlage
	}
	return total
}

// GetByID retrieves an mBGM by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*elda.MBGM, error) {
	return s.repo.GetByIDWithPositions(ctx, id)
}

// List retrieves mBGMs with optional filters
func (s *Service) List(ctx context.Context, accountID uuid.UUID, filter ServiceListFilter) ([]*elda.MBGM, error) {
	repoFilter := &ListFilter{
		Year:  filter.Year,
		Month: filter.Month,
	}
	if filter.Status != nil {
		repoFilter.Status = *filter.Status
	}
	return s.repo.ListByAccount(ctx, accountID, repoFilter)
}

// ServiceListFilter contains filter options for listing mBGMs (service layer)
type ServiceListFilter struct {
	Year   *int
	Month  *int
	Status *elda.MBGMStatus
}

// Validate validates an mBGM and updates validation status
func (s *Service) Validate(ctx context.Context, id uuid.UUID) (*ValidationResult, error) {
	mbgm, err := s.repo.GetByIDWithPositions(ctx, id)
	if err != nil {
		return nil, err
	}

	result := s.validator.ValidateMBGM(mbgm)

	// Update position validation status
	for i, pos := range mbgm.Positionen {
		posResult := s.validator.ValidatePosition(pos, mbgm.Year, mbgm.Month)
		pos.IsValid = posResult.Valid
		if !posResult.Valid {
			pos.ValidationErrors = posResult.ErrorMessages()
		}
		if err := s.repo.UpdatePosition(ctx, pos); err != nil {
			s.logger.Warn("failed to update position validation", "position", i, "error", err)
		}
	}

	// Update mBGM status if valid
	if result.Valid && mbgm.Status == elda.MBGMStatusDraft {
		mbgm.Status = elda.MBGMStatusValidated
		mbgm.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, mbgm); err != nil {
			return result, fmt.Errorf("failed to update mBGM status: %w", err)
		}
	}

	return result, nil
}

// PreviewXML generates an XML preview for an mBGM
func (s *Service) PreviewXML(ctx context.Context, id uuid.UUID, dienstgeberNr string) (*XMLPreview, error) {
	mbgm, err := s.repo.GetByIDWithPositions(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.builder.PreviewXML(mbgm, dienstgeberNr)
}

// Submit submits an mBGM to ELDA
func (s *Service) Submit(ctx context.Context, id uuid.UUID, dienstgeberNr string) (*SubmitResult, error) {
	mbgm, err := s.repo.GetByIDWithPositions(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate before submission
	validationResult := s.validator.ValidateMBGM(mbgm)
	if !validationResult.Valid {
		return nil, &ValidationError{Result: validationResult}
	}

	// Build XML document
	doc := s.buildDocument(mbgm, dienstgeberNr)

	// Submit to ELDA
	eldaResult, err := s.eldaService.SubmitMBGM(ctx, doc)

	// Update mBGM with result
	now := time.Now()
	mbgm.SubmittedAt = &now
	mbgm.RequestXML = eldaResult.RequestXML

	if err != nil {
		mbgm.Status = elda.MBGMStatusRejected
		mbgm.ErrorCode = eldaResult.ErrorCode
		mbgm.ErrorMessage = eldaResult.ErrorMessage
		mbgm.UpdatedAt = now

		if updateErr := s.repo.Update(ctx, mbgm); updateErr != nil {
			s.logger.Error("failed to update rejected mBGM", "id", id, "error", updateErr)
		}

		return &SubmitResult{
			Success:      false,
			ErrorCode:    eldaResult.ErrorCode,
			ErrorMessage: eldaResult.ErrorMessage,
		}, err
	}

	// Success
	mbgm.Status = elda.MBGMStatusSubmitted
	mbgm.Protokollnummer = eldaResult.Protokollnummer
	mbgm.UpdatedAt = now

	if updateErr := s.repo.Update(ctx, mbgm); updateErr != nil {
		s.logger.Error("failed to update submitted mBGM", "id", id, "error", updateErr)
	}

	s.logger.Info("mBGM submitted successfully",
		"id", id,
		"protokollnummer", eldaResult.Protokollnummer)

	return &SubmitResult{
		Success:         true,
		Protokollnummer: eldaResult.Protokollnummer,
		Warnings:        eldaResult.Warnings,
		SubmittedAt:     eldaResult.SubmittedAt,
	}, nil
}

// buildDocument creates an ELDA MBGMDocument from the mBGM
func (s *Service) buildDocument(mbgm *elda.MBGM, dienstgeberNr string) *elda.MBGMDocument {
	doc := &elda.MBGMDocument{
		XMLNS: elda.ELDANS,
		Kopf: elda.MBGMKopf{
			DienstgeberNummer: dienstgeberNr,
			Jahr:              mbgm.Year,
			Monat:             mbgm.Month,
			Erstellungsdatum:  time.Now().Format("2006-01-02"),
			IsKorrektur:       mbgm.IsCorrection,
		},
		Positionen: make([]elda.MBGMXMLPos, 0, len(mbgm.Positionen)),
	}

	for _, pos := range mbgm.Positionen {
		xmlPos := elda.MBGMXMLPos{
			SVNummer:          pos.SVNummer,
			Familienname:      pos.Familienname,
			Vorname:           pos.Vorname,
			Beitragsgruppe:    pos.Beitragsgruppe,
			Beitragsgrundlage: fmt.Sprintf("%.2f", pos.Beitragsgrundlage),
		}

		if pos.Geburtsdatum != nil {
			xmlPos.Geburtsdatum = pos.Geburtsdatum.Format("2006-01-02")
		}
		if pos.Sonderzahlung > 0 {
			xmlPos.Sonderzahlung = fmt.Sprintf("%.2f", pos.Sonderzahlung)
		}
		if pos.VonDatum != nil {
			xmlPos.VonDatum = pos.VonDatum.Format("2006-01-02")
		}
		if pos.BisDatum != nil {
			xmlPos.BisDatum = pos.BisDatum.Format("2006-01-02")
		}
		if pos.Wochenstunden != nil {
			wh := fmt.Sprintf("%.2f", *pos.Wochenstunden)
			xmlPos.Wochenstunden = &wh
		}

		doc.Positionen = append(doc.Positionen, xmlPos)
	}

	return doc
}

// SubmitResult contains the result of submitting an mBGM
type SubmitResult struct {
	Success         bool      `json:"success"`
	Protokollnummer string    `json:"protokollnummer,omitempty"`
	ErrorCode       string    `json:"error_code,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

// CreateCorrection creates a correction mBGM based on an existing one
func (s *Service) CreateCorrection(ctx context.Context, originalID uuid.UUID, req *elda.MBGMCreateRequest) (*elda.MBGM, error) {
	// Get original mBGM
	original, err := s.repo.GetByID(ctx, originalID)
	if err != nil {
		return nil, fmt.Errorf("original mBGM not found: %w", err)
	}

	// Verify original was submitted
	if original.Protokollnummer == "" {
		return nil, fmt.Errorf("can only correct submitted mBGMs")
	}

	// Use same year/month as original
	req.Year = original.Year
	req.Month = original.Month

	// Create the correction
	mbgm, err := s.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	// Mark as correction
	mbgm.IsCorrection = true
	mbgm.CorrectsID = &originalID
	mbgm.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, mbgm); err != nil {
		return nil, fmt.Errorf("failed to update correction mBGM: %w", err)
	}

	s.logger.Info("mBGM correction created",
		"id", mbgm.ID,
		"corrects", originalID)

	return mbgm, nil
}

// Delete deletes an mBGM (only draft status)
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	mbgm, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if mbgm.Status != elda.MBGMStatusDraft {
		return fmt.Errorf("can only delete mBGMs in draft status")
	}

	// Delete positions first
	if err := s.repo.DeletePositions(ctx, id); err != nil {
		return fmt.Errorf("failed to delete positions: %w", err)
	}

	// Delete mBGM
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete mBGM: %w", err)
	}

	s.logger.Info("mBGM deleted", "id", id)
	return nil
}

// GetSummary returns a summary of an mBGM
func (s *Service) GetSummary(ctx context.Context, id uuid.UUID, dienstgeberNr string) (*MBGMSummary, error) {
	mbgm, err := s.repo.GetByIDWithPositions(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.builder.BuildSummary(mbgm, dienstgeberNr)
}

// GetDeadlineInfo returns deadline information for an mBGM
func (s *Service) GetDeadlineInfo(year, month int) *elda.MBGMDeadlineInfo {
	return elda.GetMBGMDeadlineInfo(year, month)
}

// GetBeitragsgruppen returns available Beitragsgruppen
func (s *Service) GetBeitragsgruppen(ctx context.Context) ([]*elda.Beitragsgruppe, error) {
	return s.repo.GetBeitragsgruppen(ctx)
}

// ValidationError wraps a validation result as an error
type ValidationError struct {
	Result *ValidationResult
}

func (e *ValidationError) Error() string {
	if e.Result == nil || len(e.Result.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s", e.Result.Errors[0].Message)
}

// GetValidationErrors returns all validation errors
func (e *ValidationError) GetValidationErrors() []FieldValidationError {
	if e.Result == nil {
		return nil
	}
	return e.Result.Errors
}
