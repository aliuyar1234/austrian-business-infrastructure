package fonws

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"time"
)

// UVA period type constants
const (
	PeriodTypeMonthly   = "monthly"
	PeriodTypeQuarterly = "quarterly"
)

// UVA status constants
type UVAStatus string

const (
	UVAStatusDraft     UVAStatus = "draft"
	UVAStatusValidated UVAStatus = "validated"
	UVAStatusSubmitted UVAStatus = "submitted"
	UVAStatusAccepted  UVAStatus = "accepted"
	UVAStatusRejected  UVAStatus = "rejected"
)

// UVAPeriod represents the tax period (monthly or quarterly)
type UVAPeriod struct {
	Type  string // "monthly" or "quarterly"
	Value int    // 1-12 for monthly, 1-4 for quarterly
}

// UVA represents a VAT advance return (Umsatzsteuervoranmeldung)
type UVA struct {
	// Period information
	Year   int       // Tax year (e.g., 2025)
	Period UVAPeriod // Monthly (1-12) or Quarterly (1-4)

	// Tax amounts (in cents to avoid floating point)
	KZ000 int64 // Gesamtbetrag der Lieferungen (total deliveries)
	KZ001 int64 // Innergemeinschaftliche Lieferungen
	KZ011 int64 // Steuerfrei ohne Vorsteuerabzug
	KZ017 int64 // Normalsteuersatz 20%
	KZ018 int64 // Ermäßigter Steuersatz 10%
	KZ019 int64 // Ermäßigter Steuersatz 13%
	KZ020 int64 // Sonstige Steuersätze
	KZ022 int64 // Einfuhrumsatzsteuer
	KZ029 int64 // Innergemeinschaftliche Erwerbe
	KZ060 int64 // Vorsteuer
	KZ065 int64 // Einfuhrumsatzsteuer als Vorsteuer
	KZ066 int64 // Vorsteuern aus IG Erwerben
	KZ070 int64 // Sonstige Berichtigungen
	KZ095 int64 // Zahllast/Gutschrift (calculated)

	// Metadata
	CreatedAt   time.Time
	SubmittedAt *time.Time
	Status      UVAStatus
	Reference   string // FinanzOnline reference number
}

// UVADocument is the XML structure for FinanzOnline U30 submission
type UVADocument struct {
	XMLName      xml.Name      `xml:"Umsatzsteuervoranmeldung"`
	XMLNS        string        `xml:"xmlns,attr"`
	Steuernummer string        `xml:"Steuernummer,omitempty"`
	Zeitraum     UVAZeitraum   `xml:"Zeitraum"`
	Kennzahlen   UVAKennzahlen `xml:"Kennzahlen"`
}

// UVAZeitraum represents the period in XML format
type UVAZeitraum struct {
	Jahr    int    `xml:"Jahr"`
	Monat   string `xml:"Monat,omitempty"`
	Quartal int    `xml:"Quartal,omitempty"`
}

// UVAKennzahlen represents the KZ values in XML format
type UVAKennzahlen struct {
	KZ000 int64 `xml:"KZ000,omitempty"`
	KZ001 int64 `xml:"KZ001,omitempty"`
	KZ011 int64 `xml:"KZ011,omitempty"`
	KZ017 int64 `xml:"KZ017,omitempty"`
	KZ018 int64 `xml:"KZ018,omitempty"`
	KZ019 int64 `xml:"KZ019,omitempty"`
	KZ020 int64 `xml:"KZ020,omitempty"`
	KZ022 int64 `xml:"KZ022,omitempty"`
	KZ029 int64 `xml:"KZ029,omitempty"`
	KZ060 int64 `xml:"KZ060,omitempty"`
	KZ065 int64 `xml:"KZ065,omitempty"`
	KZ066 int64 `xml:"KZ066,omitempty"`
	KZ070 int64 `xml:"KZ070,omitempty"`
	KZ095 int64 `xml:"KZ095,omitempty"`
}

// ValidateUVA validates a UVA struct
func ValidateUVA(uva *UVA) error {
	// Validate year
	if uva.Year < 2000 || uva.Year > 2100 {
		return errors.New("year must be between 2000 and 2100")
	}

	// Validate period
	switch uva.Period.Type {
	case PeriodTypeMonthly:
		if uva.Period.Value < 1 || uva.Period.Value > 12 {
			return errors.New("month must be between 1 and 12")
		}
	case PeriodTypeQuarterly:
		if uva.Period.Value < 1 || uva.Period.Value > 4 {
			return errors.New("quarter must be between 1 and 4")
		}
	default:
		return errors.New("period type must be 'monthly' or 'quarterly'")
	}

	// Validate KZ values are non-negative (except KZ095 which can be negative for refunds)
	if uva.KZ000 < 0 {
		return errors.New("KZ000 must be non-negative")
	}
	if uva.KZ001 < 0 {
		return errors.New("KZ001 must be non-negative")
	}
	if uva.KZ011 < 0 {
		return errors.New("KZ011 must be non-negative")
	}
	if uva.KZ017 < 0 {
		return errors.New("KZ017 must be non-negative")
	}
	if uva.KZ018 < 0 {
		return errors.New("KZ018 must be non-negative")
	}
	if uva.KZ019 < 0 {
		return errors.New("KZ019 must be non-negative")
	}
	if uva.KZ020 < 0 {
		return errors.New("KZ020 must be non-negative")
	}
	if uva.KZ022 < 0 {
		return errors.New("KZ022 must be non-negative")
	}
	if uva.KZ029 < 0 {
		return errors.New("KZ029 must be non-negative")
	}
	if uva.KZ060 < 0 {
		return errors.New("KZ060 must be non-negative")
	}
	if uva.KZ065 < 0 {
		return errors.New("KZ065 must be non-negative")
	}
	if uva.KZ066 < 0 {
		return errors.New("KZ066 must be non-negative")
	}
	if uva.KZ070 < 0 {
		return errors.New("KZ070 must be non-negative")
	}
	// Note: KZ095 (Zahllast/Gutschrift) can be negative (refund)

	return nil
}

// GenerateUVAXML generates the BMF XML document from a UVA struct
func GenerateUVAXML(uva *UVA) ([]byte, error) {
	if err := ValidateUVA(uva); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	doc := UVADocument{
		XMLNS: "http://www.bmf.gv.at/steuern/fon/u30",
		Zeitraum: UVAZeitraum{
			Jahr: uva.Year,
		},
		Kennzahlen: UVAKennzahlen{
			KZ000: uva.KZ000,
			KZ001: uva.KZ001,
			KZ011: uva.KZ011,
			KZ017: uva.KZ017,
			KZ018: uva.KZ018,
			KZ019: uva.KZ019,
			KZ020: uva.KZ020,
			KZ022: uva.KZ022,
			KZ029: uva.KZ029,
			KZ060: uva.KZ060,
			KZ065: uva.KZ065,
			KZ066: uva.KZ066,
			KZ070: uva.KZ070,
			KZ095: uva.KZ095,
		},
	}

	// Set period based on type
	if uva.Period.Type == PeriodTypeMonthly {
		doc.Zeitraum.Monat = fmt.Sprintf("%02d", uva.Period.Value)
	} else {
		doc.Zeitraum.Quartal = uva.Period.Value
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	// Add XML declaration
	result := []byte(xml.Header)
	result = append(result, output...)

	return result, nil
}

// FileUploadRequest represents the SOAP request for file upload
type FileUploadRequest struct {
	XMLName xml.Name `xml:"fon:upload"`
	Xmlns   string   `xml:"xmlns:fon,attr"`
	TID     string   `xml:"tid"`
	BenID   string   `xml:"benid"`
	ID      string   `xml:"id"`
	Art     string   `xml:"art"`
	Data    string   `xml:"uebession>data"`
}

// FileUploadResponse represents the SOAP response from file upload
type FileUploadResponse struct {
	RC          int    `xml:"rc"`
	Msg         string `xml:"msg"`
	Belegnummer string `xml:"belegnummer"`
}

// FileUploadNS is the namespace for the file upload service
const FileUploadNS = "https://finanzonline.bmf.gv.at/fonws/ws/fileUploadService"

// FileUploadService handles UVA and ZM file uploads
type FileUploadService struct {
	client *Client
}

// NewFileUploadService creates a new file upload service
func NewFileUploadService(client *Client) *FileUploadService {
	return &FileUploadService{client: client}
}

// Upload submits a file to FinanzOnline
func (s *FileUploadService) Upload(sessionID, tid, benid, art string, data []byte) (*FileUploadResponse, error) {
	// Base64 encode the data
	encoded := base64.StdEncoding.EncodeToString(data)

	req := FileUploadRequest{
		Xmlns: FileUploadNS,
		TID:   tid,
		BenID: benid,
		ID:    sessionID,
		Art:   art,
		Data:  encoded,
	}

	var resp FileUploadResponse
	err := s.client.Call("fileUpload", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	if resp.RC != 0 {
		return &resp, fmt.Errorf("upload error (code %d): %s", resp.RC, resp.Msg)
	}

	return &resp, nil
}

// SubmitUVA submits a UVA to FinanzOnline
func (s *FileUploadService) SubmitUVA(sessionID, tid, benid string, uva *UVA) (*FileUploadResponse, error) {
	xmlData, err := GenerateUVAXML(uva)
	if err != nil {
		return nil, fmt.Errorf("failed to generate UVA XML: %w", err)
	}

	resp, err := s.Upload(sessionID, tid, benid, "U30", xmlData)
	if err != nil {
		return nil, err
	}

	// Update UVA status and reference
	uva.Status = UVAStatusSubmitted
	now := time.Now()
	uva.SubmittedAt = &now
	uva.Reference = resp.Belegnummer

	return resp, nil
}

// ParseUVAFromXML parses a UVA XML file into a UVA struct
func ParseUVAFromXML(data []byte) (*UVA, error) {
	var doc UVADocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse UVA XML: %w", err)
	}

	uva := &UVA{
		Year: doc.Zeitraum.Jahr,
		KZ000: doc.Kennzahlen.KZ000,
		KZ001: doc.Kennzahlen.KZ001,
		KZ011: doc.Kennzahlen.KZ011,
		KZ017: doc.Kennzahlen.KZ017,
		KZ018: doc.Kennzahlen.KZ018,
		KZ019: doc.Kennzahlen.KZ019,
		KZ020: doc.Kennzahlen.KZ020,
		KZ022: doc.Kennzahlen.KZ022,
		KZ029: doc.Kennzahlen.KZ029,
		KZ060: doc.Kennzahlen.KZ060,
		KZ065: doc.Kennzahlen.KZ065,
		KZ066: doc.Kennzahlen.KZ066,
		KZ070: doc.Kennzahlen.KZ070,
		KZ095: doc.Kennzahlen.KZ095,
		Status: UVAStatusDraft,
		CreatedAt: time.Now(),
	}

	// Determine period type
	if doc.Zeitraum.Monat != "" {
		var month int
		fmt.Sscanf(doc.Zeitraum.Monat, "%d", &month)
		uva.Period = UVAPeriod{Type: PeriodTypeMonthly, Value: month}
	} else if doc.Zeitraum.Quartal > 0 {
		uva.Period = UVAPeriod{Type: PeriodTypeQuarterly, Value: doc.Zeitraum.Quartal}
	}

	return uva, nil
}

// CalculateKZ095 calculates the tax liability/credit (Zahllast/Gutschrift)
func (uva *UVA) CalculateKZ095() int64 {
	// Tax payable: 20% of KZ017 + 10% of KZ018 + 13% of KZ019 + other taxes
	taxPayable := int64(0)
	taxPayable += uva.KZ017 * 20 / 100
	taxPayable += uva.KZ018 * 10 / 100
	taxPayable += uva.KZ019 * 13 / 100
	taxPayable += uva.KZ022 // Import VAT
	taxPayable += uva.KZ029 * 20 / 100 // IC acquisitions at 20%

	// Input tax deductions
	inputTax := uva.KZ060 + uva.KZ065 + uva.KZ066 + uva.KZ070

	// Result: positive = payment due, negative = refund
	return taxPayable - inputTax
}
