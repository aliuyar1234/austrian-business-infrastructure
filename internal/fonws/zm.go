package fonws

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ZMStatus represents the status of a ZM submission
type ZMStatus string

const (
	ZMStatusDraft     ZMStatus = "draft"
	ZMStatusSubmitted ZMStatus = "submitted"
	ZMStatusAccepted  ZMStatus = "accepted"
	ZMStatusRejected  ZMStatus = "rejected"
)

// ZMDeliveryType represents the type of delivery for ZM entries
type ZMDeliveryType string

const (
	ZMDeliveryTypeGoods      ZMDeliveryType = "L" // Lieferungen (goods)
	ZMDeliveryTypeTriangular ZMDeliveryType = "D" // Dreiecksgeschäfte (triangular transactions)
	ZMDeliveryTypeServices   ZMDeliveryType = "S" // Sonstige Leistungen (services)
)

// ZM represents a recapitulative statement (Zusammenfassende Meldung)
type ZM struct {
	Year    int
	Quarter int // 1-4

	Entries []ZMEntry

	// Metadata
	CreatedAt   time.Time
	SubmittedAt *time.Time
	Status      ZMStatus
	Reference   string
}

// ZMEntry represents a single entry in the ZM
type ZMEntry struct {
	PartnerUID   string         // EU partner UID number
	CountryCode  string         // 2-letter country code
	DeliveryType ZMDeliveryType // L, D, or S
	Amount       int64          // In cents
}

// ZM XML structures for FinanzOnline
type zmXML struct {
	XMLName    xml.Name       `xml:"ZM"`
	Jahr       int            `xml:"Jahr"`
	Quartal    int            `xml:"Quartal"`
	Positionen []zmPositionXML `xml:"Position"`
}

type zmPositionXML struct {
	PartnerUID  string `xml:"PartnerUID"`
	LandCode    string `xml:"LandCode"`
	Lieferart   string `xml:"Lieferart"`
	Bemessungsgrundlage int64 `xml:"Bemessungsgrundlage"`
}

// NewZM creates a new ZM with the given year and quarter
func NewZM(year, quarter int) *ZM {
	return &ZM{
		Year:      year,
		Quarter:   quarter,
		Entries:   []ZMEntry{},
		CreatedAt: time.Now(),
		Status:    ZMStatusDraft,
	}
}

// PeriodString returns the period in format "Q1/2025"
func (zm *ZM) PeriodString() string {
	return fmt.Sprintf("Q%d/%d", zm.Quarter, zm.Year)
}

// TotalAmount returns the total amount in cents
func (zm *ZM) TotalAmount() int64 {
	var total int64
	for _, entry := range zm.Entries {
		total += entry.Amount
	}
	return total
}

// TotalAmountEUR returns the total amount in EUR
func (zm *ZM) TotalAmountEUR() float64 {
	return float64(zm.TotalAmount()) / 100.0
}

// Validate validates the ZM
func (zm *ZM) Validate() error {
	if zm.Year < 2000 || zm.Year > 2100 {
		return errors.New("year must be between 2000 and 2100")
	}
	if zm.Quarter < 1 || zm.Quarter > 4 {
		return errors.New("quarter must be between 1 and 4")
	}
	if len(zm.Entries) == 0 {
		return errors.New("ZM must have at least one entry")
	}

	for i, entry := range zm.Entries {
		if err := entry.Validate(); err != nil {
			return fmt.Errorf("entry %d: %w", i+1, err)
		}
	}

	return nil
}

// AddEntry adds an entry to the ZM
func (zm *ZM) AddEntry(entry ZMEntry) error {
	if err := entry.Validate(); err != nil {
		return err
	}
	zm.Entries = append(zm.Entries, entry)
	return nil
}

// Validate validates a ZM entry
func (e *ZMEntry) Validate() error {
	if e.PartnerUID == "" {
		return errors.New("partner_uid is required")
	}
	if e.CountryCode == "" {
		return errors.New("country_code is required")
	}
	if len(e.CountryCode) != 2 {
		return errors.New("country_code must be 2 characters")
	}
	// Austrian UIDs are not allowed in ZM (intra-community only)
	if strings.ToUpper(e.CountryCode) == "AT" {
		return errors.New("country_code: Austrian partners are not allowed in ZM (intra-community only)")
	}
	if e.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if e.DeliveryType != ZMDeliveryTypeGoods &&
		e.DeliveryType != ZMDeliveryTypeTriangular &&
		e.DeliveryType != ZMDeliveryTypeServices {
		return errors.New("invalid delivery type (must be L, D, or S)")
	}

	return nil
}

// AmountEUR returns the amount in EUR
func (e *ZMEntry) AmountEUR() float64 {
	return float64(e.Amount) / 100.0
}

// GenerateZMXML generates XML for a ZM submission
func GenerateZMXML(zm *ZM) ([]byte, error) {
	if err := zm.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	zmXMLData := zmXML{
		Jahr:    zm.Year,
		Quartal: zm.Quarter,
	}

	for _, entry := range zm.Entries {
		zmXMLData.Positionen = append(zmXMLData.Positionen, zmPositionXML{
			PartnerUID:          entry.PartnerUID,
			LandCode:            entry.CountryCode,
			Lieferart:           string(entry.DeliveryType),
			Bemessungsgrundlage: entry.Amount / 100, // Convert to EUR for XML
		})
	}

	data, err := xml.MarshalIndent(zmXMLData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	return append([]byte(xml.Header), data...), nil
}

// ZMSubmissionResult represents the result of a ZM submission
type ZMSubmissionResult struct {
	Success   bool      `json:"success"`
	Reference string    `json:"reference,omitempty"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Period    string    `json:"period"`
}

// SubmitZM submits a ZM to FinanzOnline using the FileUploadService
func (s *FileUploadService) SubmitZM(sessionID, tid, benid string, zm *ZM) (*ZMSubmissionResult, error) {
	// Generate XML
	xmlData, err := GenerateZMXML(zm)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ZM XML: %w", err)
	}

	// Submit using file upload service
	resp, err := s.Upload(sessionID, tid, benid, "ZM", xmlData)
	if err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	result := &ZMSubmissionResult{
		Success:   resp.RC == 0,
		Reference: resp.Belegnummer,
		Timestamp: time.Now(),
		Period:    zm.PeriodString(),
	}

	if resp.RC != 0 {
		result.Message = resp.Msg
	} else {
		result.Message = "ZM submitted successfully"
		zm.Status = ZMStatusSubmitted
		now := time.Now()
		zm.SubmittedAt = &now
		zm.Reference = resp.Belegnummer
	}

	return result, nil
}

// ParseZMFromCSV parses ZM entries from CSV data
// Expected format: partner_uid,country_code,delivery_type,amount
func ParseZMFromCSV(data []byte) ([]ZMEntry, error) {
	// Use the same CSV parsing approach as other modules
	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return nil, errors.New("CSV must have header and at least one data row")
	}

	var entries []ZMEntry
	for i, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, ",")
		if len(fields) < 4 {
			return nil, fmt.Errorf("line %d: expected 4 fields, got %d", i+2, len(fields))
		}

		var amount int64
		fmt.Sscanf(strings.TrimSpace(fields[3]), "%d", &amount)

		entry := ZMEntry{
			PartnerUID:   strings.TrimSpace(fields[0]),
			CountryCode:  strings.ToUpper(strings.TrimSpace(fields[1])),
			DeliveryType: ZMDeliveryType(strings.ToUpper(strings.TrimSpace(fields[2]))),
			Amount:       amount,
		}

		if err := entry.Validate(); err != nil {
			return nil, fmt.Errorf("line %d: %w", i+2, err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
