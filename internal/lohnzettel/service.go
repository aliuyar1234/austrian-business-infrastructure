package lohnzettel

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

// Service handles L16 Lohnzettel business logic
type Service struct {
	repo      *Repository
	builder   *Builder
	validator *Validator
	eldaL16   *elda.L16Service
}

// NewService creates a new Lohnzettel service
func NewService(pool *pgxpool.Pool, eldaClient *elda.Client) *Service {
	return &Service{
		repo:      NewRepository(pool),
		builder:   NewBuilder(),
		validator: NewValidator(),
		eldaL16:   elda.NewL16Service(eldaClient),
	}
}

// Create creates a new Lohnzettel (L16)
func (s *Service) Create(ctx context.Context, req *elda.LohnzettelCreateRequest) (*elda.Lohnzettel, error) {
	// Validate the request
	validation := ValidateCreateRequest(req)
	if !validation.Valid {
		return nil, &ValidationError{
			Message: "Validierungsfehler",
			Errors:  validation.ErrorMessages(),
		}
	}

	// Create the lohnzettel entity
	lohnzettel := &elda.Lohnzettel{
		ID:            uuid.New(),
		ELDAAccountID: req.ELDAAccountID,
		Year:          req.Year,
		SVNummer:      req.SVNummer,
		Familienname:  req.Familienname,
		Vorname:       req.Vorname,
		L16Data:       req.L16Data,
		Status:        elda.L16StatusDraft,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Parse Geburtsdatum if provided
	if req.Geburtsdatum != "" {
		if t, err := time.Parse("2006-01-02", req.Geburtsdatum); err == nil {
			lohnzettel.Geburtsdatum = &t
		}
	}

	// Save to database
	if err := s.repo.Create(ctx, lohnzettel); err != nil {
		return nil, fmt.Errorf("failed to create lohnzettel: %w", err)
	}

	return lohnzettel, nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
	Errors  []string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Get retrieves a Lohnzettel by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*elda.Lohnzettel, error) {
	return s.repo.Get(ctx, id)
}

// List retrieves Lohnzettel with optional filters
func (s *Service) List(ctx context.Context, filter ServiceListFilter) ([]*elda.Lohnzettel, error) {
	repoFilter := ListFilter{
		ELDAAccountID: filter.ELDAAccountID,
		Year:          filter.Year,
		Status:        filter.Status,
		BatchID:       filter.BatchID,
		Limit:         filter.Limit,
		Offset:        filter.Offset,
	}
	return s.repo.List(ctx, repoFilter)
}

// ServiceListFilter defines filter options for listing Lohnzettel
type ServiceListFilter struct {
	ELDAAccountID *uuid.UUID
	Year          *int
	Status        *elda.L16Status
	BatchID       *uuid.UUID
	Limit         int
	Offset        int
}

// Validate validates a Lohnzettel
func (s *Service) Validate(ctx context.Context, id uuid.UUID) (*ValidationResult, error) {
	lohnzettel, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	result := s.validator.ValidateLohnzettel(lohnzettel)

	// Update status if valid
	if result.Valid && lohnzettel.Status == elda.L16StatusDraft {
		lohnzettel.Status = elda.L16StatusValidated
		lohnzettel.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, lohnzettel); err != nil {
			return nil, fmt.Errorf("failed to update lohnzettel status: %w", err)
		}
	}

	return result, nil
}

// Preview generates an XML preview without submitting
func (s *Service) Preview(ctx context.Context, id uuid.UUID) (*XMLPreview, error) {
	lohnzettel, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.builder.PreviewXML(lohnzettel)
}

// Submit submits a Lohnzettel to ELDA
func (s *Service) Submit(ctx context.Context, id uuid.UUID) (*elda.L16SubmitResult, error) {
	lohnzettel, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate first
	validation := s.validator.ValidateLohnzettel(lohnzettel)
	if !validation.Valid {
		return nil, &ValidationError{
			Message: "Lohnzettel ist nicht valide",
			Errors:  validation.ErrorMessages(),
		}
	}

	// Build the XML document
	doc := s.builder.buildDocument(lohnzettel)

	// Submit to ELDA
	result, err := s.eldaL16.SubmitL16(ctx, doc)

	// Store request XML regardless of result
	xmlData, _ := s.builder.BuildXML(lohnzettel)
	lohnzettel.RequestXML = string(xmlData)
	lohnzettel.UpdatedAt = time.Now()

	if err != nil {
		// Submission failed
		lohnzettel.Status = elda.L16StatusRejected
		if result != nil {
			lohnzettel.ErrorCode = result.ErrorCode
			lohnzettel.ErrorMessage = result.ErrorMessage
		} else {
			lohnzettel.ErrorMessage = err.Error()
		}
	} else {
		// Submission succeeded
		lohnzettel.Status = elda.L16StatusSubmitted
		lohnzettel.Protokollnummer = result.Protokollnummer
		now := time.Now()
		lohnzettel.SubmittedAt = &now
		lohnzettel.ErrorCode = ""
		lohnzettel.ErrorMessage = ""
	}

	// Update database
	if updateErr := s.repo.Update(ctx, lohnzettel); updateErr != nil {
		return result, fmt.Errorf("submitted but failed to update record: %w", updateErr)
	}

	return result, err
}

// CreateBatch creates a new batch for L16 submission
func (s *Service) CreateBatch(ctx context.Context, req *elda.LohnzettelBatchCreateRequest) (*elda.LohnzettelBatch, error) {
	// Validate request
	validation := ValidateBatchCreateRequest(req)
	if !validation.Valid {
		return nil, &ValidationError{
			Message: "Batch-Validierungsfehler",
			Errors:  validation.ErrorMessages(),
		}
	}

	// Create batch
	batch := &elda.LohnzettelBatch{
		ID:              uuid.New(),
		ELDAAccountID:   req.ELDAAccountID,
		Year:            req.Year,
		TotalLohnzettel: len(req.LohnzettelIDs),
		Status:          elda.BatchStatusDraft,
		CreatedAt:       time.Now(),
	}

	// Save batch
	if err := s.repo.CreateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	// Assign lohnzettel to batch
	for _, lzID := range req.LohnzettelIDs {
		lz, err := s.repo.Get(ctx, lzID)
		if err != nil {
			continue // Skip missing lohnzettel
		}
		lz.BatchID = &batch.ID
		lz.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, lz); err != nil {
			continue
		}
	}

	// Reload batch with lohnzettel
	return s.repo.GetBatch(ctx, batch.ID)
}

// GetBatch retrieves a batch by ID
func (s *Service) GetBatch(ctx context.Context, id uuid.UUID) (*elda.LohnzettelBatch, error) {
	return s.repo.GetBatch(ctx, id)
}

// ListBatches lists batches with filters
func (s *Service) ListBatches(ctx context.Context, filter BatchListFilter) ([]*elda.LohnzettelBatch, error) {
	repoFilter := BatchListFilter{
		ELDAAccountID: filter.ELDAAccountID,
		Year:          filter.Year,
		Status:        filter.Status,
		Limit:         filter.Limit,
		Offset:        filter.Offset,
	}
	return s.repo.ListBatches(ctx, repoFilter)
}

// SubmitBatch submits all Lohnzettel in a batch to ELDA
func (s *Service) SubmitBatch(ctx context.Context, batchID uuid.UUID, maxConcurrent int) (*elda.L16BatchSubmitResult, error) {
	batch, err := s.repo.GetBatch(ctx, batchID)
	if err != nil {
		return nil, err
	}

	// Update batch status to submitting
	batch.Status = elda.BatchStatusSubmitting
	now := time.Now()
	batch.StartedAt = &now
	if err := s.repo.UpdateBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to update batch status: %w", err)
	}

	// Get all lohnzettel in batch
	lohnzettel, err := s.repo.List(ctx, ListFilter{BatchID: &batchID})
	if err != nil {
		return nil, fmt.Errorf("failed to get batch lohnzettel: %w", err)
	}

	// Prepare batch items
	var items []*elda.L16BatchItem
	for _, lz := range lohnzettel {
		// Validate each
		validation := s.validator.ValidateLohnzettel(lz)
		if !validation.Valid {
			continue // Skip invalid
		}

		doc := s.builder.buildDocument(lz)
		items = append(items, &elda.L16BatchItem{
			LohnzettelID: lz.ID,
			Document:     doc,
		})
	}

	// Submit batch to ELDA
	result := s.eldaL16.SubmitL16Batch(ctx, items, maxConcurrent)

	// Update individual lohnzettel based on results
	for _, itemResult := range result.Results {
		lz, err := s.repo.Get(ctx, itemResult.LohnzettelID)
		if err != nil {
			continue
		}

		lz.UpdatedAt = time.Now()
		if itemResult.Success {
			lz.Status = elda.L16StatusSubmitted
			lz.Protokollnummer = itemResult.Protokollnummer
			lz.SubmittedAt = &now
		} else {
			lz.Status = elda.L16StatusRejected
			lz.ErrorCode = itemResult.ErrorCode
			lz.ErrorMessage = itemResult.Error
		}

		if err := s.repo.Update(ctx, lz); err != nil {
			continue
		}
	}

	// Update batch final status
	completedAt := time.Now()
	batch.CompletedAt = &completedAt
	batch.SubmittedCount = result.Submitted
	batch.AcceptedCount = result.Accepted
	batch.RejectedCount = result.Rejected

	if result.Rejected > 0 && result.Accepted > 0 {
		batch.Status = elda.BatchStatusPartialFailure
	} else if result.Rejected == result.Total {
		batch.Status = elda.BatchStatusPartialFailure // All failed
	} else {
		batch.Status = elda.BatchStatusCompleted
	}

	if err := s.repo.UpdateBatch(ctx, batch); err != nil {
		return result, fmt.Errorf("batch submitted but failed to update record: %w", err)
	}

	return result, nil
}

// CreateBerichtigung creates a correction Lohnzettel
func (s *Service) CreateBerichtigung(ctx context.Context, originalID uuid.UUID, correctedData *elda.L16Data) (*elda.Lohnzettel, error) {
	original, err := s.repo.Get(ctx, originalID)
	if err != nil {
		return nil, fmt.Errorf("original lohnzettel not found: %w", err)
	}

	if original.Protokollnummer == "" {
		return nil, fmt.Errorf("original lohnzettel has no protokollnummer - was it submitted?")
	}

	// Create correction
	correction := &elda.Lohnzettel{
		ID:             uuid.New(),
		ELDAAccountID:  original.ELDAAccountID,
		Year:           original.Year,
		SVNummer:       original.SVNummer,
		Familienname:   original.Familienname,
		Vorname:        original.Vorname,
		Geburtsdatum:   original.Geburtsdatum,
		L16Data:        *correctedData,
		Status:         elda.L16StatusDraft,
		IsBerichtigung: true,
		BerichtigtID:   &originalID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.repo.Create(ctx, correction); err != nil {
		return nil, fmt.Errorf("failed to create correction: %w", err)
	}

	return correction, nil
}

// SubmitBerichtigung submits a correction Lohnzettel
func (s *Service) SubmitBerichtigung(ctx context.Context, correctionID uuid.UUID) (*elda.L16SubmitResult, error) {
	correction, err := s.repo.Get(ctx, correctionID)
	if err != nil {
		return nil, err
	}

	if !correction.IsBerichtigung || correction.BerichtigtID == nil {
		return nil, fmt.Errorf("not a correction lohnzettel")
	}

	// Get original to get protokollnummer
	original, err := s.repo.Get(ctx, *correction.BerichtigtID)
	if err != nil {
		return nil, fmt.Errorf("original lohnzettel not found: %w", err)
	}

	// Validate
	validation := s.validator.ValidateLohnzettel(correction)
	if !validation.Valid {
		return nil, &ValidationError{
			Message: "Berichtigung ist nicht valide",
			Errors:  validation.ErrorMessages(),
		}
	}

	// Build XML
	doc := s.builder.buildDocument(correction)

	// Submit to ELDA
	result, err := s.eldaL16.SubmitL16Berichtigung(ctx, doc, original.Protokollnummer)

	// Store request XML
	xmlData, _ := s.builder.BuildXML(correction)
	correction.RequestXML = string(xmlData)
	correction.UpdatedAt = time.Now()

	if err != nil {
		correction.Status = elda.L16StatusRejected
		if result != nil {
			correction.ErrorCode = result.ErrorCode
			correction.ErrorMessage = result.ErrorMessage
		} else {
			correction.ErrorMessage = err.Error()
		}
	} else {
		correction.Status = elda.L16StatusSubmitted
		correction.Protokollnummer = result.Protokollnummer
		now := time.Now()
		correction.SubmittedAt = &now
	}

	if updateErr := s.repo.Update(ctx, correction); updateErr != nil {
		return result, fmt.Errorf("submitted but failed to update record: %w", updateErr)
	}

	return result, err
}

// GetDeadlineInfo returns deadline information for L16
func (s *Service) GetDeadlineInfo(year int) *elda.L16DeadlineInfo {
	return elda.GetL16DeadlineInfo(year)
}

// GetDeadlineStatus returns deadline status for all pending years
func (s *Service) GetDeadlineStatus(ctx context.Context, eldaAccountID uuid.UUID) (*ServiceDeadlineStatus, error) {
	// Get submitted years from database
	submittedYears, err := s.repo.GetSubmittedYears(ctx, eldaAccountID)
	if err != nil {
		return nil, err
	}

	pending := elda.GetPendingL16Years(submittedYears)

	status := &ServiceDeadlineStatus{
		CurrentYear:    time.Now().Year(),
		SubmittedYears: submittedYears,
	}

	for _, p := range pending {
		status.PendingYears = append(status.PendingYears, YearStatus{
			Year:      p.Year,
			Deadline:  p.Deadline,
			IsOverdue: p.IsOverdue,
			DaysLeft:  elda.DaysUntilL16Deadline(p.Year),
		})

		if p.IsOverdue {
			status.OverdueCount++
		}
		if !p.IsOverdue && elda.DaysUntilL16Deadline(p.Year) <= 7 {
			status.UrgentCount++
		}
	}

	return status, nil
}

// ServiceDeadlineStatus contains deadline information for L16 (used by service)
type ServiceDeadlineStatus struct {
	CurrentYear    int          `json:"current_year"`
	SubmittedYears []int        `json:"submitted_years"`
	PendingYears   []YearStatus `json:"pending_years"`
	OverdueCount   int          `json:"overdue_count"`
	UrgentCount    int          `json:"urgent_count"`
}

// YearStatus contains status for a specific year
type YearStatus struct {
	Year      int       `json:"year"`
	Deadline  time.Time `json:"deadline"`
	IsOverdue bool      `json:"is_overdue"`
	DaysLeft  int       `json:"days_left"`
}

// GetSummary generates a summary for a Lohnzettel
func (s *Service) GetSummary(ctx context.Context, id uuid.UUID) (*L16Summary, error) {
	lohnzettel, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.builder.BuildSummary(lohnzettel)
}

// Delete deletes a Lohnzettel (only if draft)
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	lohnzettel, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if lohnzettel.Status != elda.L16StatusDraft {
		return fmt.Errorf("kann nur Entwürfe löschen, aktueller Status: %s", lohnzettel.Status)
	}

	return s.repo.Delete(ctx, id)
}

// Count returns the count of Lohnzettel matching the filter
func (s *Service) Count(ctx context.Context, filter ServiceListFilter) (int, error) {
	repoFilter := ListFilter{
		ELDAAccountID: filter.ELDAAccountID,
		Year:          filter.Year,
		Status:        filter.Status,
		BatchID:       filter.BatchID,
	}
	return s.repo.Count(ctx, repoFilter)
}

// GetStatistics retrieves L16 statistics from ELDA
func (s *Service) GetStatistics(ctx context.Context, year int) (*elda.L16Statistics, error) {
	return s.eldaL16.GetL16Statistics(ctx, year)
}

// QueryStatus queries the status of a submitted Lohnzettel
func (s *Service) QueryStatus(ctx context.Context, id uuid.UUID) (*elda.L16StatusResult, error) {
	lohnzettel, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if lohnzettel.Protokollnummer == "" {
		return nil, fmt.Errorf("lohnzettel has no protokollnummer")
	}

	result, err := s.eldaL16.QueryL16Status(ctx, lohnzettel.Protokollnummer)
	if err != nil {
		return nil, err
	}

	// Update local status if processed
	if result.Processed {
		if result.ErrorCode != "" {
			lohnzettel.Status = elda.L16StatusRejected
			lohnzettel.ErrorCode = result.ErrorCode
			lohnzettel.ErrorMessage = result.ErrorMessage
		} else {
			lohnzettel.Status = elda.L16StatusAccepted
		}
		lohnzettel.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, lohnzettel); err != nil {
			return result, fmt.Errorf("status updated but failed to save: %w", err)
		}
	}

	return result, nil
}

// BulkCreate creates multiple Lohnzettel at once
func (s *Service) BulkCreate(ctx context.Context, requests []*elda.LohnzettelCreateRequest) ([]*BulkCreateResult, error) {
	results := make([]*BulkCreateResult, len(requests))

	for i, req := range requests {
		result := &BulkCreateResult{
			Index:    i,
			SVNummer: req.SVNummer,
			Name:     fmt.Sprintf("%s %s", req.Vorname, req.Familienname),
		}

		lz, err := s.Create(ctx, req)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
			result.LohnzettelID = lz.ID
		}

		results[i] = result
	}

	return results, nil
}

// BulkCreateResult contains the result of a bulk create operation
type BulkCreateResult struct {
	Index        int       `json:"index"`
	SVNummer     string    `json:"sv_nummer"`
	Name         string    `json:"name"`
	Success      bool      `json:"success"`
	LohnzettelID uuid.UUID `json:"lohnzettel_id,omitempty"`
	Error        string    `json:"error,omitempty"`
}

// ImportFromCSV imports Lohnzettel from CSV data
func (s *Service) ImportFromCSV(ctx context.Context, eldaAccountID uuid.UUID, year int, csvData []byte, format CSVFormat) (*ImportResult, error) {
	importer := NewImporter()

	// Parse CSV
	records, parseErr := importer.ParseCSV(csvData, format)
	if parseErr != nil {
		return nil, fmt.Errorf("CSV parse error: %w", parseErr)
	}

	result := &ImportResult{
		TotalRows: len(records),
	}

	for i, record := range records {
		rowNum := i + 2 // +2 for header and 0-index

		// Convert to create request
		req := &elda.LohnzettelCreateRequest{
			ELDAAccountID: eldaAccountID,
			Year:          year,
			SVNummer:      record.SVNummer,
			Familienname:  record.Familienname,
			Vorname:       record.Vorname,
			Geburtsdatum:  record.Geburtsdatum,
			L16Data:       record.L16Data,
		}

		// Validate
		validation := ValidateCreateRequest(req)
		if !validation.Valid {
			result.FailedRows++
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Field:   "validation",
				Message: validation.ErrorMessages()[0],
			})
			continue
		}

		// Create
		lz, err := s.Create(ctx, req)
		if err != nil {
			result.FailedRows++
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Field:   "create",
				Message: err.Error(),
			})
			continue
		}

		result.SuccessRows++
		result.CreatedIDs = append(result.CreatedIDs, lz.ID)
	}

	return result, nil
}

// ImportResult contains the result of a CSV import
type ImportResult struct {
	TotalRows   int           `json:"total_rows"`
	SuccessRows int           `json:"success_rows"`
	FailedRows  int           `json:"failed_rows"`
	CreatedIDs  []uuid.UUID   `json:"created_ids"`
	Errors      []ImportError `json:"errors,omitempty"`
}

// ImportError represents an error during import
type ImportError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// CSVRecord represents a parsed CSV record
type CSVRecord struct {
	SVNummer     string
	Familienname string
	Vorname      string
	Geburtsdatum string
	L16Data      elda.L16Data
}

// CSVFormat represents CSV format types
type CSVFormat string

const (
	CSVFormatGeneric CSVFormat = "generic"
	CSVFormatBMD     CSVFormat = "bmd"
	CSVFormatRZL     CSVFormat = "rzl"
)

// Importer handles CSV import
type Importer struct{}

// NewImporter creates a new importer
func NewImporter() *Importer {
	return &Importer{}
}

// ParseCSV parses CSV data into records
func (imp *Importer) ParseCSV(data []byte, format CSVFormat) ([]CSVRecord, error) {
	// For now, implement generic format
	// BMD and RZL formats would have different column mappings

	lines := splitLines(string(data))
	if len(lines) < 2 {
		return nil, fmt.Errorf("CSV must have header and at least one data row")
	}

	var records []CSVRecord

	// Skip header (line 0)
	for i := 1; i < len(lines); i++ {
		if lines[i] == "" {
			continue
		}

		fields := splitCSV(lines[i])
		if len(fields) < 5 {
			continue // Skip malformed rows
		}

		record := CSVRecord{
			SVNummer:     fields[0],
			Familienname: fields[1],
			Vorname:      fields[2],
		}

		if len(fields) > 3 {
			record.Geburtsdatum = fields[3]
		}

		// Parse L16 data fields
		if len(fields) > 4 {
			record.L16Data.KZ210 = parseFloat(fields[4])
		}
		if len(fields) > 5 {
			record.L16Data.KZ215 = parseFloat(fields[5])
		}
		if len(fields) > 6 {
			record.L16Data.KZ220 = parseFloat(fields[6])
		}
		if len(fields) > 7 {
			record.L16Data.KZ230 = parseFloat(fields[7])
		}

		records = append(records, record)
	}

	return records, nil
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitCSV(line string) []string {
	var fields []string
	var field string
	inQuotes := false

	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			inQuotes = !inQuotes
		} else if c == ';' && !inQuotes {
			fields = append(fields, field)
			field = ""
		} else if c == ',' && !inQuotes {
			fields = append(fields, field)
			field = ""
		} else {
			field += string(c)
		}
	}
	fields = append(fields, field)

	return fields
}

func parseFloat(s string) float64 {
	// Handle German number format (comma as decimal separator)
	s = replaceCommaWithDot(s)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func replaceCommaWithDot(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, '.')
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
