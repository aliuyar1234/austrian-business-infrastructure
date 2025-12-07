package fonws

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalidUIDFormat = errors.New("invalid UID format")
	ErrUIDNotFound      = errors.New("UID not found in registry")
	ErrUIDDailyLimit    = errors.New("daily query limit exceeded for this UID")
)

// UID format patterns by country
var uidPatterns = map[string]*regexp.Regexp{
	"AT": regexp.MustCompile(`^ATU\d{8}$`),      // Austria: ATU + 8 digits
	"DE": regexp.MustCompile(`^DE\d{9}$`),       // Germany: DE + 9 digits
	"IT": regexp.MustCompile(`^IT\d{11}$`),      // Italy: IT + 11 digits
	"FR": regexp.MustCompile(`^FR[A-Z0-9]{2}\d{9}$`), // France: FR + 2 chars + 9 digits
	"NL": regexp.MustCompile(`^NL\d{9}B\d{2}$`), // Netherlands: NL + 9 digits + B + 2 digits
	"BE": regexp.MustCompile(`^BE0?\d{9,10}$`),  // Belgium: BE + 9-10 digits
	"ES": regexp.MustCompile(`^ES[A-Z0-9]\d{7}[A-Z0-9]$`), // Spain: ES + 9 chars
	"GB": regexp.MustCompile(`^GB\d{9}(\d{3})?$`), // UK: GB + 9 or 12 digits
	"PL": regexp.MustCompile(`^PL\d{10}$`),      // Poland: PL + 10 digits
	"CH": regexp.MustCompile(`^CHE\d{9}$`),      // Switzerland: CHE + 9 digits
}

// UIDAbfrageRequest represents a SOAP request for UID validation
type UIDAbfrageRequest struct {
	XMLName xml.Name `xml:"uid:uidAbfrage"`
	Xmlns   string   `xml:"xmlns:uid,attr"`
	TID     string   `xml:"tid"`
	BenID   string   `xml:"benid"`
	ID      string   `xml:"id"`
	UIDTN   string   `xml:"uid_tn"`
	Stufe   int      `xml:"stufe"`
}

// UIDAbfrageResponse represents a SOAP response from UID validation
type UIDAbfrageResponse struct {
	RC         int    `xml:"rc"`
	Msg        string `xml:"msg"`
	UIDTN      string `xml:"uid_tn"`
	Gueltig    string `xml:"gueltig"`
	Name       string `xml:"name"`
	AdrStrasse string `xml:"adr_strasse"`
	AdrPLZ     string `xml:"adr_plz"`
	AdrOrt     string `xml:"adr_ort"`
}

// UIDValidationResult contains the validation result
type UIDValidationResult struct {
	UID          string     `json:"uid"`
	Valid        bool       `json:"valid"`
	CompanyName  string     `json:"company_name,omitempty"`
	Address      UIDAddress `json:"address,omitempty"`
	ValidAt      time.Time  `json:"valid_at,omitempty"`
	QueryTime    time.Time  `json:"query_time"`
	Source       string     `json:"source"` // "finanzonline" or "vies"
	CountryCode  string     `json:"country_code"`
	ErrorCode    int        `json:"error_code,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
}

// UIDAddress represents an address from UID validation
type UIDAddress struct {
	Street   string `json:"street,omitempty"`
	PostCode string `json:"post_code,omitempty"`
	City     string `json:"city,omitempty"`
	Country  string `json:"country,omitempty"`
}

// FormattedAddress returns a formatted address string
func (r *UIDValidationResult) FormattedAddress() string {
	if r.Address.Street == "" {
		return ""
	}
	return fmt.Sprintf("%s, %s %s", r.Address.Street, r.Address.PostCode, r.Address.City)
}

// UIDFormatResult contains the result of format validation
type UIDFormatResult struct {
	Valid       bool
	CountryCode string
	Error       string
}

// ValidateUIDFormat validates the format of a UID number
func ValidateUIDFormat(uid string) *UIDFormatResult {
	uid = strings.ToUpper(strings.TrimSpace(uid))
	result := &UIDFormatResult{}

	if len(uid) < 4 {
		result.Error = "UID too short"
		return result
	}

	// Extract country code
	countryCode := uid[:2]
	result.CountryCode = countryCode

	// Check if country is supported
	pattern, ok := uidPatterns[countryCode]
	if !ok {
		// Try with 3-character country code (e.g., CHE for Switzerland)
		if len(uid) >= 3 {
			countryCode = uid[:3]
			pattern, ok = uidPatterns[countryCode]
			if ok {
				result.CountryCode = countryCode
			}
		}
		if !ok {
			result.Error = fmt.Sprintf("unsupported country code: %s", uid[:2])
			return result
		}
	}

	// Validate against pattern
	if !pattern.MatchString(uid) {
		result.Error = fmt.Sprintf("invalid format for country %s", countryCode)
		return result
	}

	result.Valid = true
	return result
}

// ConvertUIDResponse converts a SOAP response to a validation result
func ConvertUIDResponse(resp *UIDAbfrageResponse) *UIDValidationResult {
	result := &UIDValidationResult{
		UID:       resp.UIDTN,
		QueryTime: time.Now(),
		Source:    "finanzonline",
	}

	// Extract country code from UID
	if len(resp.UIDTN) >= 2 {
		result.CountryCode = resp.UIDTN[:2]
	}

	// Check if valid
	if resp.RC == 0 && (resp.Gueltig == "true" || resp.Gueltig == "1") {
		result.Valid = true
		result.CompanyName = resp.Name
		result.Address = UIDAddress{
			Street:   resp.AdrStrasse,
			PostCode: resp.AdrPLZ,
			City:     resp.AdrOrt,
			Country:  result.CountryCode,
		}
		result.ValidAt = time.Now()
	} else {
		result.Valid = false
		result.ErrorCode = resp.RC
		result.ErrorMessage = resp.Msg

		// Map error codes to friendly messages
		switch resp.RC {
		case 1513:
			result.ErrorMessage = "daily query limit exceeded for this UID"
		case 1514:
			result.ErrorMessage = "UID number not found in registry"
		case -2:
			result.ErrorMessage = "session expired or invalid"
		}
	}

	return result
}

// ParseUIDCSV parses a CSV file containing UID numbers
// Returns a slice of UID strings
func ParseUIDCSV(data []byte) ([]string, error) {
	reader := csv.NewReader(bytes.NewReader(data))

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Find uid column index
	uidColIdx := -1
	for i, col := range header {
		if strings.ToLower(strings.TrimSpace(col)) == "uid" {
			uidColIdx = i
			break
		}
	}

	if uidColIdx == -1 {
		return nil, errors.New("CSV must contain a 'uid' column")
	}

	var uids []string
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV row: %w", err)
		}

		if len(record) > uidColIdx {
			uid := strings.TrimSpace(record[uidColIdx])
			if uid != "" {
				uids = append(uids, strings.ToUpper(uid))
			}
		}
	}

	return uids, nil
}

// WriteUIDResultsCSV writes validation results to CSV format
func WriteUIDResultsCSV(results []*UIDValidationResult) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"uid", "valid", "company_name", "street", "post_code", "city", "error"}
	if err := writer.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write rows
	for _, r := range results {
		row := []string{
			r.UID,
			fmt.Sprintf("%t", r.Valid),
			r.CompanyName,
			r.Address.Street,
			r.Address.PostCode,
			r.Address.City,
			r.ErrorMessage,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// ParseUIDList parses a list of UIDs from various formats (newline, comma, semicolon)
func ParseUIDList(input string) []string {
	// Replace common separators with newlines
	input = strings.ReplaceAll(input, ",", "\n")
	input = strings.ReplaceAll(input, ";", "\n")

	var uids []string
	scanner := bufio.NewScanner(strings.NewReader(input))
	for scanner.Scan() {
		uid := strings.TrimSpace(scanner.Text())
		if uid != "" {
			uids = append(uids, strings.ToUpper(uid))
		}
	}

	return uids
}

// UIDNS is the namespace for the UID service
const UIDNS = "http://finanzonline.bmf.gv.at/fonuid"

// UIDService handles UID validation queries
type UIDService struct {
	client *Client
}

// NewUIDService creates a new UID service
func NewUIDService(client *Client) *UIDService {
	return &UIDService{client: client}
}

// Validate validates a UID number against FinanzOnline
func (s *UIDService) Validate(sessionID, tid, benid, uid string, level int) (*UIDValidationResult, error) {
	// First validate format
	formatResult := ValidateUIDFormat(uid)
	if !formatResult.Valid {
		return &UIDValidationResult{
			UID:          uid,
			Valid:        false,
			QueryTime:    time.Now(),
			ErrorMessage: formatResult.Error,
		}, nil
	}

	// Build request
	req := UIDAbfrageRequest{
		Xmlns: UIDNS,
		TID:   tid,
		BenID: benid,
		ID:    sessionID,
		UIDTN: uid,
		Stufe: level,
	}

	var resp UIDAbfrageResponse
	err := s.client.Call("uidAbfrage", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("UID validation failed: %w", err)
	}

	return ConvertUIDResponse(&resp), nil
}

// ValidateBatch validates multiple UIDs
func (s *UIDService) ValidateBatch(sessionID, tid, benid string, uids []string, level int) ([]*UIDValidationResult, error) {
	results := make([]*UIDValidationResult, 0, len(uids))

	for _, uid := range uids {
		result, err := s.Validate(sessionID, tid, benid, uid, level)
		if err != nil {
			// Record error but continue with other UIDs
			results = append(results, &UIDValidationResult{
				UID:          uid,
				Valid:        false,
				QueryTime:    time.Now(),
				ErrorMessage: err.Error(),
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}
