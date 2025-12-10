package imports

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"

	"austrian-business-infrastructure/internal/account"
)

var (
	ErrEmptyFile         = errors.New("empty CSV file")
	ErrMissingHeaders    = errors.New("missing required CSV headers")
	ErrTooManyRows       = errors.New("CSV file exceeds maximum allowed rows")
	ErrInvalidColumnCount = errors.New("row has invalid column count")
)

// Required CSV columns
var requiredColumns = []string{"name", "type", "tid", "ben_id", "pin"}

// ParsedRow represents a parsed CSV row
type ParsedRow struct {
	RowNumber   int               `json:"row_number"`
	Name        string            `json:"name"`
	Type        string            `json:"type"`
	TID         string            `json:"tid,omitempty"`
	BenID       string            `json:"ben_id,omitempty"`
	PIN         string            `json:"pin,omitempty"`
	DienstgeberNr string          `json:"dienstgeber_nr,omitempty"`
	CertPath    string            `json:"cert_path,omitempty"`
	CertPassword string           `json:"cert_password,omitempty"`
	Username    string            `json:"username,omitempty"`
	Password    string            `json:"password,omitempty"`
	Errors      []string          `json:"errors,omitempty"`
	Valid       bool              `json:"valid"`
}

// ParseResult contains the result of parsing a CSV file
type ParseResult struct {
	Rows       []*ParsedRow `json:"rows"`
	ValidCount int          `json:"valid_count"`
	ErrorCount int          `json:"error_count"`
	TotalRows  int          `json:"total_rows"`
}

// Parser handles CSV parsing for account imports
type Parser struct {
	maxRows int
}

// NewParser creates a new CSV parser
func NewParser(maxRows int) *Parser {
	if maxRows <= 0 {
		maxRows = 500 // Default limit
	}
	return &Parser{maxRows: maxRows}
}

// Parse parses a CSV file and validates rows
func (p *Parser) Parse(reader io.Reader) (*ParseResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// Read header
	headers, err := csvReader.Read()
	if err == io.EOF {
		return nil, ErrEmptyFile
	}
	if err != nil {
		return nil, err
	}

	// Normalize headers
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Validate required columns
	for _, col := range requiredColumns {
		if _, ok := headerMap[col]; !ok {
			return nil, ErrMissingHeaders
		}
	}

	result := &ParseResult{
		Rows: make([]*ParsedRow, 0),
	}

	rowNum := 1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		rowNum++
		if rowNum > p.maxRows+1 { // +1 for header
			return nil, ErrTooManyRows
		}

		row := &ParsedRow{
			RowNumber: rowNum,
			Valid:     true,
		}

		// Extract values by column name
		row.Name = getColumn(record, headerMap, "name")
		row.Type = strings.ToLower(getColumn(record, headerMap, "type"))
		row.TID = getColumn(record, headerMap, "tid")
		row.BenID = getColumn(record, headerMap, "ben_id")
		row.PIN = getColumn(record, headerMap, "pin")
		row.DienstgeberNr = getColumn(record, headerMap, "dienstgeber_nr")
		row.CertPath = getColumn(record, headerMap, "cert_path")
		row.CertPassword = getColumn(record, headerMap, "cert_password")
		row.Username = getColumn(record, headerMap, "username")
		row.Password = getColumn(record, headerMap, "password")

		// Validate row
		p.validateRow(row)

		if row.Valid {
			result.ValidCount++
		} else {
			result.ErrorCount++
		}

		result.Rows = append(result.Rows, row)
	}

	result.TotalRows = len(result.Rows)
	return result, nil
}

func (p *Parser) validateRow(row *ParsedRow) {
	// Validate name
	if row.Name == "" {
		row.Errors = append(row.Errors, "name is required")
		row.Valid = false
	}

	// Validate type
	if err := account.ValidateAccountType(row.Type); err != nil {
		row.Errors = append(row.Errors, "invalid account type")
		row.Valid = false
	}

	// Validate type-specific fields
	switch row.Type {
	case account.AccountTypeFinanzOnline:
		if err := account.ValidateTID(row.TID); err != nil {
			row.Errors = append(row.Errors, err.Error())
			row.Valid = false
		}
		if err := account.ValidateBenID(row.BenID); err != nil {
			row.Errors = append(row.Errors, err.Error())
			row.Valid = false
		}
		if err := account.ValidatePIN(row.PIN); err != nil {
			row.Errors = append(row.Errors, err.Error())
			row.Valid = false
		}

	case account.AccountTypeELDA:
		if err := account.ValidateDienstgebernummer(row.DienstgeberNr); err != nil {
			row.Errors = append(row.Errors, err.Error())
			row.Valid = false
		}
		if row.CertPath == "" {
			row.Errors = append(row.Errors, "certificate path is required for ELDA")
			row.Valid = false
		}
		if err := account.ValidatePIN(row.PIN); err != nil {
			row.Errors = append(row.Errors, err.Error())
			row.Valid = false
		}

	case account.AccountTypeFirmenbuch:
		if row.Username == "" {
			row.Errors = append(row.Errors, "username is required for Firmenbuch")
			row.Valid = false
		}
		if row.Password == "" {
			row.Errors = append(row.Errors, "password is required for Firmenbuch")
			row.Valid = false
		}
	}
}

func getColumn(record []string, headerMap map[string]int, column string) string {
	if idx, ok := headerMap[column]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}
