package elda

import (
	"context"
	"encoding/xml"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// L16Service handles L16 Lohnzettel ELDA protocol operations
type L16Service struct {
	client *Client
}

// NewL16Service creates a new L16 service
func NewL16Service(client *Client) *L16Service {
	return &L16Service{client: client}
}

// SubmitL16 submits an L16 Lohnzettel to ELDA
func (s *L16Service) SubmitL16(ctx context.Context, doc *L16Document) (*L16SubmitResult, error) {
	// Marshal the document to XML
	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal L16 document: %w", err)
	}

	// Add XML header
	fullXML := []byte(xml.Header)
	fullXML = append(fullXML, xmlData...)

	// Create submission request
	type submitRequest struct {
		XMLName  xml.Name `xml:"SubmitLohnzettel"`
		XMLNS    string   `xml:"xmlns,attr"`
		Document string   `xml:"Document"`
	}

	req := submitRequest{
		XMLNS:    ELDANS,
		Document: string(fullXML),
	}

	var resp L16Response
	err = s.client.callWithRetry(ctx, "SubmitLohnzettel", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("ELDA L16 submission failed: %w", err)
	}

	result := &L16SubmitResult{
		Success:         resp.Erfolg,
		Protokollnummer: resp.Protokollnummer,
		RequestXML:      string(fullXML),
		SubmittedAt:     time.Now(),
	}

	if !resp.Erfolg {
		result.ErrorCode = resp.ErrorCode
		result.ErrorMessage = resp.ErrorMessage
		return result, fmt.Errorf("ELDA rejected L16: %s - %s", resp.ErrorCode, resp.ErrorMessage)
	}

	return result, nil
}

// L16SubmitResult contains the result of an L16 submission
type L16SubmitResult struct {
	Success         bool      `json:"success"`
	Protokollnummer string    `json:"protokollnummer,omitempty"`
	ErrorCode       string    `json:"error_code,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
	RequestXML      string    `json:"-"`
	ResponseXML     string    `json:"-"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

// SubmitL16Batch submits multiple L16 documents with controlled concurrency
func (s *L16Service) SubmitL16Batch(ctx context.Context, docs []*L16BatchItem, maxConcurrent int) *L16BatchSubmitResult {
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default max concurrent submissions
	}
	if maxConcurrent > 10 {
		maxConcurrent = 10 // Cap at 10 to avoid overwhelming ELDA
	}

	result := &L16BatchSubmitResult{
		Total:     len(docs),
		Results:   make([]L16ItemResult, len(docs)),
		StartedAt: time.Now(),
	}

	// Use semaphore for concurrency control
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, item := range docs {
		wg.Add(1)
		go func(idx int, batchItem *L16BatchItem) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				result.Results[idx] = L16ItemResult{
					LohnzettelID: batchItem.LohnzettelID,
					Success:      false,
					Error:        "context cancelled",
				}
				mu.Unlock()
				return
			}

			// Submit the L16
			submitResult, err := s.SubmitL16(ctx, batchItem.Document)

			mu.Lock()
			defer mu.Unlock()

			itemResult := L16ItemResult{
				LohnzettelID: batchItem.LohnzettelID,
			}

			if err != nil {
				itemResult.Success = false
				itemResult.Error = err.Error()
				if submitResult != nil {
					itemResult.ErrorCode = submitResult.ErrorCode
				}
				result.Rejected++
			} else {
				itemResult.Success = true
				itemResult.Protokollnummer = submitResult.Protokollnummer
				result.Accepted++
			}

			result.Results[idx] = itemResult
			result.Submitted++
		}(i, item)
	}

	wg.Wait()
	result.CompletedAt = time.Now()

	return result
}

// L16BatchItem represents a single L16 in a batch
type L16BatchItem struct {
	LohnzettelID uuid.UUID
	Document     *L16Document
}

// L16BatchSubmitResult contains the result of a batch submission
type L16BatchSubmitResult struct {
	Total       int             `json:"total"`
	Submitted   int             `json:"submitted"`
	Accepted    int             `json:"accepted"`
	Rejected    int             `json:"rejected"`
	Results     []L16ItemResult `json:"results"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt time.Time       `json:"completed_at"`
}

// L16ItemResult contains the result for a single L16 in a batch
type L16ItemResult struct {
	LohnzettelID    uuid.UUID `json:"lohnzettel_id"`
	Success         bool      `json:"success"`
	Protokollnummer string    `json:"protokollnummer,omitempty"`
	ErrorCode       string    `json:"error_code,omitempty"`
	Error           string    `json:"error,omitempty"`
}

// QueryL16Status queries the status of an L16 submission
func (s *L16Service) QueryL16Status(ctx context.Context, protokollnummer string) (*L16StatusResult, error) {
	type statusRequest struct {
		XMLName         xml.Name `xml:"LohnzettelStatusAbfrage"`
		XMLNS           string   `xml:"xmlns,attr"`
		Protokollnummer string   `xml:"Protokollnummer"`
	}

	type statusResponse struct {
		XMLName         xml.Name `xml:"LohnzettelStatusResponse"`
		Status          string   `xml:"Status"`
		Protokollnummer string   `xml:"Protokollnummer"`
		Verarbeitet     bool     `xml:"Verarbeitet"`
		FehlerCode      string   `xml:"FehlerCode,omitempty"`
		FehlerMeldung   string   `xml:"FehlerMeldung,omitempty"`
		VerarbeitetAm   string   `xml:"VerarbeitetAm,omitempty"`
	}

	req := statusRequest{
		XMLNS:           ELDANS,
		Protokollnummer: protokollnummer,
	}

	var resp statusResponse
	err := s.client.callWithRetry(ctx, "LohnzettelStatusAbfrage", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("L16 status query failed: %w", err)
	}

	result := &L16StatusResult{
		Protokollnummer: resp.Protokollnummer,
		Status:          resp.Status,
		Processed:       resp.Verarbeitet,
	}

	if resp.FehlerCode != "" {
		result.ErrorCode = resp.FehlerCode
		result.ErrorMessage = resp.FehlerMeldung
	}

	if resp.VerarbeitetAm != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", resp.VerarbeitetAm); err == nil {
			result.ProcessedAt = &t
		}
	}

	return result, nil
}

// L16StatusResult contains the status of an L16
type L16StatusResult struct {
	Protokollnummer string     `json:"protokollnummer"`
	Status          string     `json:"status"`
	Processed       bool       `json:"processed"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ErrorCode       string     `json:"error_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
}

// SubmitL16Berichtigung submits a correction L16
func (s *L16Service) SubmitL16Berichtigung(ctx context.Context, doc *L16Document, originalProtokollnummer string) (*L16SubmitResult, error) {
	// Marshal the document to XML
	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal L16 correction: %w", err)
	}

	fullXML := []byte(xml.Header)
	fullXML = append(fullXML, xmlData...)

	// Create correction request
	type correctionRequest struct {
		XMLName             xml.Name `xml:"SubmitLohnzettelBerichtigung"`
		XMLNS               string   `xml:"xmlns,attr"`
		OriginalProtokoll   string   `xml:"OriginalProtokollnummer"`
		BerichtigungDocument string  `xml:"BerichtigungDocument"`
	}

	req := correctionRequest{
		XMLNS:                ELDANS,
		OriginalProtokoll:    originalProtokollnummer,
		BerichtigungDocument: string(fullXML),
	}

	var resp L16Response
	err = s.client.callWithRetry(ctx, "SubmitLohnzettelBerichtigung", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("L16 correction submission failed: %w", err)
	}

	result := &L16SubmitResult{
		Success:         resp.Erfolg,
		Protokollnummer: resp.Protokollnummer,
		RequestXML:      string(fullXML),
		SubmittedAt:     time.Now(),
	}

	if !resp.Erfolg {
		result.ErrorCode = resp.ErrorCode
		result.ErrorMessage = resp.ErrorMessage
		return result, fmt.Errorf("ELDA rejected L16 correction: %s - %s", resp.ErrorCode, resp.ErrorMessage)
	}

	return result, nil
}

// ValidateL16 validates an L16 document with ELDA (dry run)
func (s *L16Service) ValidateL16(ctx context.Context, doc *L16Document) (*L16ValidationResult, error) {
	type validateRequest struct {
		XMLName  xml.Name `xml:"ValidateLohnzettel"`
		XMLNS    string   `xml:"xmlns,attr"`
		Document string   `xml:"Document"`
	}

	type validateResponse struct {
		XMLName   xml.Name `xml:"ValidateLohnzettelResponse"`
		Valid     bool     `xml:"Valid"`
		Fehler    []string `xml:"Fehler>Fehler,omitempty"`
		Warnungen []string `xml:"Warnungen>Warnung,omitempty"`
	}

	// Marshal the document
	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal L16 for validation: %w", err)
	}

	fullXML := []byte(xml.Header)
	fullXML = append(fullXML, xmlData...)

	req := validateRequest{
		XMLNS:    ELDANS,
		Document: string(fullXML),
	}

	var resp validateResponse
	err = s.client.callWithRetry(ctx, "ValidateLohnzettel", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("L16 validation request failed: %w", err)
	}

	return &L16ValidationResult{
		Valid:    resp.Valid,
		Errors:   resp.Fehler,
		Warnings: resp.Warnungen,
	}, nil
}

// L16ValidationResult contains ELDA's validation result
type L16ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// GetL16DeadlineInfo returns deadline information for a given year
func GetL16DeadlineInfo(year int) *L16DeadlineInfo {
	deadline := GetL16Deadline(year)
	now := time.Now()

	daysUntil := DaysUntilL16Deadline(year)

	return &L16DeadlineInfo{
		Year:          year,
		Deadline:      deadline,
		DaysRemaining: daysUntil,
		IsOverdue:     now.After(deadline),
		IsUrgent:      daysUntil > 0 && daysUntil <= 7, // 7 days for L16 (more complex)
		FormattedDate: deadline.Format("02.01.2006"),
	}
}

// L16DeadlineInfo contains deadline information
type L16DeadlineInfo struct {
	Year          int       `json:"year"`
	Deadline      time.Time `json:"deadline"`
	DaysRemaining int       `json:"days_remaining"`
	IsOverdue     bool      `json:"is_overdue"`
	IsUrgent      bool      `json:"is_urgent"`
	FormattedDate string    `json:"formatted_date"`
}

// GetPendingL16Years returns years that need L16 submission
func GetPendingL16Years(submittedYears []int) []L16PendingYear {
	now := time.Now()
	currentYear := now.Year()

	// Build a set of submitted years
	submittedSet := make(map[int]bool)
	for _, y := range submittedYears {
		submittedSet[y] = true
	}

	var pending []L16PendingYear

	// Check from 2020 to current-1
	for y := 2020; y <= currentYear-1; y++ {
		if submittedSet[y] {
			continue
		}

		deadline := GetL16Deadline(y)
		// Only include if deadline hasn't passed more than 5 years ago
		if now.Sub(deadline) > 5*365*24*time.Hour {
			continue
		}

		pending = append(pending, L16PendingYear{
			Year:      y,
			Deadline:  deadline,
			IsOverdue: now.After(deadline),
		})
	}

	return pending
}

// L16PendingYear represents a year that needs L16 submission
type L16PendingYear struct {
	Year      int       `json:"year"`
	Deadline  time.Time `json:"deadline"`
	IsOverdue bool      `json:"is_overdue"`
}

// GetL16Statistics retrieves statistics for L16 submissions
func (s *L16Service) GetL16Statistics(ctx context.Context, year int) (*L16Statistics, error) {
	type statsRequest struct {
		XMLName xml.Name `xml:"LohnzettelStatistik"`
		XMLNS   string   `xml:"xmlns,attr"`
		Jahr    int      `xml:"Jahr"`
	}

	type statsResponse struct {
		XMLName          xml.Name `xml:"LohnzettelStatistikResponse"`
		Jahr             int      `xml:"Jahr"`
		GesamtAnzahl     int      `xml:"GesamtAnzahl"`
		AngenommenAnzahl int      `xml:"AngenommenAnzahl"`
		AbgelehntAnzahl  int      `xml:"AbgelehntAnzahl"`
		PendingAnzahl    int      `xml:"PendingAnzahl"`
	}

	req := statsRequest{
		XMLNS: ELDANS,
		Jahr:  year,
	}

	var resp statsResponse
	err := s.client.callWithRetry(ctx, "LohnzettelStatistik", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("L16 statistics query failed: %w", err)
	}

	return &L16Statistics{
		Year:      resp.Jahr,
		Total:     resp.GesamtAnzahl,
		Accepted:  resp.AngenommenAnzahl,
		Rejected:  resp.AbgelehntAnzahl,
		Pending:   resp.PendingAnzahl,
		QueriedAt: time.Now(),
	}, nil
}

// L16Statistics contains submission statistics
type L16Statistics struct {
	Year      int       `json:"year"`
	Total     int       `json:"total"`
	Accepted  int       `json:"accepted"`
	Rejected  int       `json:"rejected"`
	Pending   int       `json:"pending"`
	QueriedAt time.Time `json:"queried_at"`
}
