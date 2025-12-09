package elda

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

// MBGMService handles mBGM ELDA protocol operations
type MBGMService struct {
	client *Client
}

// NewMBGMService creates a new mBGM service
func NewMBGMService(client *Client) *MBGMService {
	return &MBGMService{client: client}
}

// SubmitMBGM submits an mBGM to ELDA
func (s *MBGMService) SubmitMBGM(ctx context.Context, doc *MBGMDocument) (*MBGMSubmitResult, error) {
	// Marshal the document to XML
	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mBGM document: %w", err)
	}

	// Add XML header
	fullXML := []byte(xml.Header)
	fullXML = append(fullXML, xmlData...)

	// Create submission request
	type submitRequest struct {
		XMLName  xml.Name `xml:"SubmitMBGM"`
		XMLNS    string   `xml:"xmlns,attr"`
		Document string   `xml:"Document"` // Base64 or inline XML depending on ELDA spec
	}

	req := submitRequest{
		XMLNS:    ELDANS,
		Document: string(fullXML),
	}

	var resp MBGMResponse
	err = s.client.callWithRetry(ctx, "SubmitMBGM", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("ELDA submission failed: %w", err)
	}

	result := &MBGMSubmitResult{
		Success:         resp.Erfolg,
		Protokollnummer: resp.Protokollnummer,
		RequestXML:      string(fullXML),
		SubmittedAt:     time.Now(),
	}

	if !resp.Erfolg {
		result.ErrorCode = resp.ErrorCode
		result.ErrorMessage = resp.ErrorMessage
		return result, fmt.Errorf("ELDA rejected mBGM: %s - %s", resp.ErrorCode, resp.ErrorMessage)
	}

	if len(resp.Warnungen) > 0 {
		result.Warnings = resp.Warnungen
	}

	return result, nil
}

// MBGMSubmitResult contains the result of an mBGM submission
type MBGMSubmitResult struct {
	Success         bool      `json:"success"`
	Protokollnummer string    `json:"protokollnummer,omitempty"`
	ErrorCode       string    `json:"error_code,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
	RequestXML      string    `json:"-"`
	ResponseXML     string    `json:"-"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

// QueryMBGMStatus queries the status of an mBGM submission
func (s *MBGMService) QueryMBGMStatus(ctx context.Context, dienstgeberNr, protokollnummer string) (*MBGMStatusResult, error) {
	type statusRequest struct {
		XMLName         xml.Name `xml:"MBGMStatusAbfrage"`
		XMLNS           string   `xml:"xmlns,attr"`
		DienstgeberNr   string   `xml:"DienstgeberNummer"`
		Protokollnummer string   `xml:"Protokollnummer"`
	}

	type statusResponse struct {
		XMLName        xml.Name `xml:"MBGMStatusResponse"`
		Status         string   `xml:"Status"`
		Protokollnummer string  `xml:"Protokollnummer"`
		Verarbeitet    bool     `xml:"Verarbeitet"`
		FehlerCode     string   `xml:"FehlerCode,omitempty"`
		FehlerMeldung  string   `xml:"FehlerMeldung,omitempty"`
		VerarbeitetAm  string   `xml:"VerarbeitetAm,omitempty"`
	}

	req := statusRequest{
		XMLNS:           ELDANS,
		DienstgeberNr:   dienstgeberNr,
		Protokollnummer: protokollnummer,
	}

	var resp statusResponse
	err := s.client.callWithRetry(ctx, "MBGMStatusAbfrage", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("status query failed: %w", err)
	}

	result := &MBGMStatusResult{
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

// MBGMStatusResult contains the status of an mBGM
type MBGMStatusResult struct {
	Protokollnummer string     `json:"protokollnummer"`
	Status          string     `json:"status"`
	Processed       bool       `json:"processed"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	ErrorCode       string     `json:"error_code,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
}

// SubmitCorrection submits a correction mBGM
func (s *MBGMService) SubmitCorrection(ctx context.Context, doc *MBGMDocument, originalProtokollnummer string) (*MBGMSubmitResult, error) {
	// Mark as correction
	doc.Kopf.IsKorrektur = true

	// Add reference to original
	type correctionRequest struct {
		XMLName             xml.Name     `xml:"SubmitMBGMKorrektur"`
		XMLNS               string       `xml:"xmlns,attr"`
		OriginalProtokoll   string       `xml:"OriginalProtokollnummer"`
		KorrekturDocument   MBGMDocument `xml:"KorrekturDocument"`
	}

	req := correctionRequest{
		XMLNS:             ELDANS,
		OriginalProtokoll: originalProtokollnummer,
		KorrekturDocument: *doc,
	}

	var resp MBGMResponse
	err := s.client.callWithRetry(ctx, "SubmitMBGMKorrektur", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("correction submission failed: %w", err)
	}

	result := &MBGMSubmitResult{
		Success:         resp.Erfolg,
		Protokollnummer: resp.Protokollnummer,
		SubmittedAt:     time.Now(),
	}

	if !resp.Erfolg {
		result.ErrorCode = resp.ErrorCode
		result.ErrorMessage = resp.ErrorMessage
		return result, fmt.Errorf("ELDA rejected correction: %s - %s", resp.ErrorCode, resp.ErrorMessage)
	}

	return result, nil
}

// ValidateMBGM validates an mBGM document with ELDA (dry run)
func (s *MBGMService) ValidateMBGM(ctx context.Context, doc *MBGMDocument) (*MBGMValidationResult, error) {
	type validateRequest struct {
		XMLName  xml.Name     `xml:"ValidateMBGM"`
		XMLNS    string       `xml:"xmlns,attr"`
		Document MBGMDocument `xml:"Document"`
	}

	type validateResponse struct {
		XMLName     xml.Name `xml:"ValidateMBGMResponse"`
		Valid       bool     `xml:"Valid"`
		Fehler      []string `xml:"Fehler>Fehler,omitempty"`
		Warnungen   []string `xml:"Warnungen>Warnung,omitempty"`
	}

	req := validateRequest{
		XMLNS:    ELDANS,
		Document: *doc,
	}

	var resp validateResponse
	err := s.client.callWithRetry(ctx, "ValidateMBGM", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("validation request failed: %w", err)
	}

	return &MBGMValidationResult{
		Valid:    resp.Valid,
		Errors:   resp.Fehler,
		Warnings: resp.Warnungen,
	}, nil
}

// MBGMValidationResult contains ELDA's validation result
type MBGMValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// GetMBGMDeadlineInfo returns deadline information for a given period
func GetMBGMDeadlineInfo(year, month int) *MBGMDeadlineInfo {
	deadline := GetMBGMDeadline(year, month)
	now := time.Now()

	daysUntil := int(deadline.Sub(now).Hours() / 24)
	if daysUntil < 0 {
		daysUntil = 0
	}

	return &MBGMDeadlineInfo{
		Year:           year,
		Month:          month,
		Deadline:       deadline,
		DaysRemaining:  daysUntil,
		IsOverdue:      now.After(deadline),
		IsUrgent:       daysUntil > 0 && daysUntil <= 3,
		FormattedDate:  deadline.Format("02.01.2006"),
	}
}

// MBGMDeadlineInfo contains deadline information
type MBGMDeadlineInfo struct {
	Year          int       `json:"year"`
	Month         int       `json:"month"`
	Deadline      time.Time `json:"deadline"`
	DaysRemaining int       `json:"days_remaining"`
	IsOverdue     bool      `json:"is_overdue"`
	IsUrgent      bool      `json:"is_urgent"`
	FormattedDate string    `json:"formatted_date"`
}

// GetPendingMBGMPeriods returns periods that need mBGM submission
func GetPendingMBGMPeriods(lastSubmittedYear, lastSubmittedMonth int) []MBGMPeriod {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := int(now.Month())

	// mBGM is for the previous month
	// If current is January 2025, we need to submit December 2024
	var periods []MBGMPeriod

	// Start from the month after last submitted
	year := lastSubmittedYear
	month := lastSubmittedMonth + 1
	if month > 12 {
		month = 1
		year++
	}

	// Add all pending periods up to previous month
	for {
		// Stop if we've reached current month (can't submit for current month yet)
		if year > currentYear || (year == currentYear && month >= currentMonth) {
			break
		}

		deadline := GetMBGMDeadline(year, month)
		periods = append(periods, MBGMPeriod{
			Year:      year,
			Month:     month,
			Deadline:  deadline,
			IsOverdue: time.Now().After(deadline),
		})

		month++
		if month > 12 {
			month = 1
			year++
		}
	}

	return periods
}

// MBGMPeriod represents a period for mBGM submission
type MBGMPeriod struct {
	Year      int       `json:"year"`
	Month     int       `json:"month"`
	Deadline  time.Time `json:"deadline"`
	IsOverdue bool      `json:"is_overdue"`
}
