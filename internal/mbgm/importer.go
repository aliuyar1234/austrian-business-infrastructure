package mbgm

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/elda"
)

// CSVFormat defines the CSV import format
type CSVFormat string

const (
	// FormatGeneric is the default generic CSV format
	FormatGeneric CSVFormat = "generic"
	// FormatBMD is the BMD accounting software export format
	FormatBMD CSVFormat = "bmd"
	// FormatRZL is the RZL accounting software export format
	FormatRZL CSVFormat = "rzl"
)

// Importer handles CSV import for mBGM data
type Importer struct {
	validator *Validator
}

// NewImporter creates a new CSV importer
func NewImporter(validator *Validator) *Importer {
	return &Importer{validator: validator}
}

// ImportResult contains the result of importing a CSV file
type ImportResult struct {
	Success      bool                 `json:"success"`
	TotalRows    int                  `json:"total_rows"`
	ValidRows    int                  `json:"valid_rows"`
	InvalidRows  int                  `json:"invalid_rows"`
	Positions    []*ImportedPosition  `json:"positions"`
	Errors       []ImportError        `json:"errors,omitempty"`
	Warnings     []string             `json:"warnings,omitempty"`
	DetectedFormat CSVFormat          `json:"detected_format"`
}

// ImportedPosition contains a parsed position from CSV
type ImportedPosition struct {
	RowNumber     int                              `json:"row_number"`
	Data          elda.MBGMPositionCreateRequest   `json:"data"`
	Valid         bool                             `json:"valid"`
	Errors        []string                         `json:"errors,omitempty"`
}

// ImportError represents an error during import
type ImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column,omitempty"`
	Message string `json:"message"`
}

// Import imports mBGM positions from CSV data
func (i *Importer) Import(data []byte, format CSVFormat, eldaAccountID uuid.UUID, year, month int) (*ImportResult, error) {
	result := &ImportResult{
		Positions:     make([]*ImportedPosition, 0),
		Errors:        make([]ImportError, 0),
		DetectedFormat: format,
	}

	// Detect BOM and encoding
	data = removeBOM(data)

	// Auto-detect format if generic
	if format == FormatGeneric {
		format = i.detectFormat(data)
		result.DetectedFormat = format
	}

	// Parse CSV
	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = detectDelimiter(data)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Map columns based on format
	colMap := i.mapColumns(header, format)
	if colMap == nil {
		return nil, fmt.Errorf("could not map CSV columns - unknown format")
	}

	// Read rows
	rowNum := 1 // Header is row 0
	for {
		rowNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Row:     rowNum,
				Message: fmt.Sprintf("CSV parse error: %v", err),
			})
			continue
		}

		result.TotalRows++

		// Parse row
		pos, rowErrors := i.parseRow(record, colMap, rowNum)
		if pos != nil {
			pos.SVNummer = strings.TrimSpace(pos.SVNummer)

			// Validate SV-Nummer
			if err := ValidateSVNummer(pos.SVNummer); err != nil {
				rowErrors = append(rowErrors, fmt.Sprintf("SV-Nummer: %v", err))
			}
		}

		importedPos := &ImportedPosition{
			RowNumber: rowNum,
			Valid:     len(rowErrors) == 0,
		}

		if pos != nil {
			importedPos.Data = *pos
		}

		if len(rowErrors) > 0 {
			importedPos.Errors = rowErrors
			result.InvalidRows++
			for _, e := range rowErrors {
				result.Errors = append(result.Errors, ImportError{
					Row:     rowNum,
					Message: e,
				})
			}
		} else {
			result.ValidRows++
		}

		result.Positions = append(result.Positions, importedPos)
	}

	result.Success = result.InvalidRows == 0 && result.ValidRows > 0

	// Add warnings
	if result.InvalidRows > 0 {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("%d von %d Zeilen haben Fehler", result.InvalidRows, result.TotalRows))
	}

	return result, nil
}

// columnMap maps CSV columns to fields
type columnMap struct {
	SVNummer          int
	Familienname      int
	Vorname           int
	Geburtsdatum      int
	Beitragsgruppe    int
	Beitragsgrundlage int
	Sonderzahlung     int
	VonDatum          int
	BisDatum          int
	Wochenstunden     int
}

// mapColumns maps header columns to field indices
func (i *Importer) mapColumns(header []string, format CSVFormat) *columnMap {
	cm := &columnMap{
		SVNummer:          -1,
		Familienname:      -1,
		Vorname:           -1,
		Geburtsdatum:      -1,
		Beitragsgruppe:    -1,
		Beitragsgrundlage: -1,
		Sonderzahlung:     -1,
		VonDatum:          -1,
		BisDatum:          -1,
		Wochenstunden:     -1,
	}

	// Normalize headers
	normalized := make([]string, len(header))
	for i, h := range header {
		normalized[i] = strings.ToLower(strings.TrimSpace(h))
	}

	switch format {
	case FormatBMD:
		return i.mapBMDColumns(normalized, cm)
	case FormatRZL:
		return i.mapRZLColumns(normalized, cm)
	default:
		return i.mapGenericColumns(normalized, cm)
	}
}

// mapGenericColumns maps columns for generic CSV format
func (i *Importer) mapGenericColumns(header []string, cm *columnMap) *columnMap {
	svAliases := []string{"sv_nummer", "svnummer", "sv-nummer", "sozialversicherungsnummer", "sv"}
	familieAliases := []string{"familienname", "nachname", "name", "family_name", "last_name"}
	vornameAliases := []string{"vorname", "first_name", "given_name"}
	geburtAliases := []string{"geburtsdatum", "geburtstag", "birth_date", "dob"}
	beitragsgruppeAliases := []string{"beitragsgruppe", "bg", "gruppe", "contribution_group"}
	beitragsgrundlageAliases := []string{"beitragsgrundlage", "grundlage", "betrag", "amount", "basis"}
	sonderzahlungAliases := []string{"sonderzahlung", "sz", "bonus", "sonder"}
	vonAliases := []string{"von_datum", "von", "from", "start", "beginn"}
	bisAliases := []string{"bis_datum", "bis", "to", "end", "ende"}
	stundenAliases := []string{"wochenstunden", "stunden", "hours", "arbeitszeit"}

	for idx, h := range header {
		if containsAny(h, svAliases) {
			cm.SVNummer = idx
		} else if containsAny(h, familieAliases) {
			cm.Familienname = idx
		} else if containsAny(h, vornameAliases) {
			cm.Vorname = idx
		} else if containsAny(h, geburtAliases) {
			cm.Geburtsdatum = idx
		} else if containsAny(h, beitragsgruppeAliases) {
			cm.Beitragsgruppe = idx
		} else if containsAny(h, beitragsgrundlageAliases) {
			cm.Beitragsgrundlage = idx
		} else if containsAny(h, sonderzahlungAliases) {
			cm.Sonderzahlung = idx
		} else if containsAny(h, vonAliases) {
			cm.VonDatum = idx
		} else if containsAny(h, bisAliases) {
			cm.BisDatum = idx
		} else if containsAny(h, stundenAliases) {
			cm.Wochenstunden = idx
		}
	}

	// Validate required columns
	if cm.SVNummer == -1 || cm.Familienname == -1 || cm.Beitragsgruppe == -1 || cm.Beitragsgrundlage == -1 {
		return nil
	}

	return cm
}

// mapBMDColumns maps columns for BMD format
func (i *Importer) mapBMDColumns(header []string, cm *columnMap) *columnMap {
	// BMD specific column names
	for idx, h := range header {
		switch h {
		case "svnr", "sv-nr":
			cm.SVNummer = idx
		case "zuname", "nachname":
			cm.Familienname = idx
		case "vorname":
			cm.Vorname = idx
		case "gebdat", "geburtsdatum":
			cm.Geburtsdatum = idx
		case "bgr", "beitragsgruppe":
			cm.Beitragsgruppe = idx
		case "bbgl", "beitragsgrundlage":
			cm.Beitragsgrundlage = idx
		case "sz", "sonderzahlung":
			cm.Sonderzahlung = idx
		case "von":
			cm.VonDatum = idx
		case "bis":
			cm.BisDatum = idx
		case "wstd", "wochenstunden":
			cm.Wochenstunden = idx
		}
	}

	if cm.SVNummer == -1 || cm.Familienname == -1 || cm.Beitragsgruppe == -1 || cm.Beitragsgrundlage == -1 {
		return nil
	}

	return cm
}

// mapRZLColumns maps columns for RZL format
func (i *Importer) mapRZLColumns(header []string, cm *columnMap) *columnMap {
	// RZL specific column names
	for idx, h := range header {
		switch h {
		case "sozialversicherungsnummer", "sv_nummer":
			cm.SVNummer = idx
		case "familienname", "name":
			cm.Familienname = idx
		case "vorname":
			cm.Vorname = idx
		case "geburtsdatum":
			cm.Geburtsdatum = idx
		case "beitragsgruppe":
			cm.Beitragsgruppe = idx
		case "beitragsgrundlage_monatlich":
			cm.Beitragsgrundlage = idx
		case "sonderzahlung":
			cm.Sonderzahlung = idx
		case "beschaeftigt_von":
			cm.VonDatum = idx
		case "beschaeftigt_bis":
			cm.BisDatum = idx
		case "arbeitszeit_woche":
			cm.Wochenstunden = idx
		}
	}

	if cm.SVNummer == -1 || cm.Familienname == -1 || cm.Beitragsgruppe == -1 || cm.Beitragsgrundlage == -1 {
		return nil
	}

	return cm
}

// parseRow parses a single CSV row
func (i *Importer) parseRow(record []string, cm *columnMap, rowNum int) (*elda.MBGMPositionCreateRequest, []string) {
	var errors []string
	pos := &elda.MBGMPositionCreateRequest{}

	// Required fields
	if cm.SVNummer >= 0 && cm.SVNummer < len(record) {
		pos.SVNummer = strings.TrimSpace(record[cm.SVNummer])
	} else {
		errors = append(errors, "SV-Nummer fehlt")
	}

	if cm.Familienname >= 0 && cm.Familienname < len(record) {
		pos.Familienname = strings.TrimSpace(record[cm.Familienname])
	} else {
		errors = append(errors, "Familienname fehlt")
	}

	if cm.Vorname >= 0 && cm.Vorname < len(record) {
		pos.Vorname = strings.TrimSpace(record[cm.Vorname])
	}

	if cm.Beitragsgruppe >= 0 && cm.Beitragsgruppe < len(record) {
		pos.Beitragsgruppe = strings.ToUpper(strings.TrimSpace(record[cm.Beitragsgruppe]))
	} else {
		errors = append(errors, "Beitragsgruppe fehlt")
	}

	if cm.Beitragsgrundlage >= 0 && cm.Beitragsgrundlage < len(record) {
		val, err := parseAmount(record[cm.Beitragsgrundlage])
		if err != nil {
			errors = append(errors, fmt.Sprintf("Beitragsgrundlage ungültig: %v", err))
		} else {
			pos.Beitragsgrundlage = val
		}
	} else {
		errors = append(errors, "Beitragsgrundlage fehlt")
	}

	// Optional fields
	if cm.Geburtsdatum >= 0 && cm.Geburtsdatum < len(record) && record[cm.Geburtsdatum] != "" {
		date, err := parseDate(record[cm.Geburtsdatum])
		if err == nil {
			pos.Geburtsdatum = date
		}
	}

	if cm.Sonderzahlung >= 0 && cm.Sonderzahlung < len(record) && record[cm.Sonderzahlung] != "" {
		val, err := parseAmount(record[cm.Sonderzahlung])
		if err == nil {
			pos.Sonderzahlung = val
		}
	}

	if cm.VonDatum >= 0 && cm.VonDatum < len(record) && record[cm.VonDatum] != "" {
		date, err := parseDate(record[cm.VonDatum])
		if err == nil {
			pos.VonDatum = date
		}
	}

	if cm.BisDatum >= 0 && cm.BisDatum < len(record) && record[cm.BisDatum] != "" {
		date, err := parseDate(record[cm.BisDatum])
		if err == nil {
			pos.BisDatum = date
		}
	}

	if cm.Wochenstunden >= 0 && cm.Wochenstunden < len(record) && record[cm.Wochenstunden] != "" {
		val, err := parseAmount(record[cm.Wochenstunden])
		if err == nil {
			pos.Wochenstunden = &val
		}
	}

	return pos, errors
}

// detectFormat auto-detects the CSV format from the data
func (i *Importer) detectFormat(data []byte) CSVFormat {
	reader := bufio.NewReader(bytes.NewReader(data))
	header, _ := reader.ReadString('\n')
	headerLower := strings.ToLower(header)

	// BMD indicators
	if strings.Contains(headerLower, "svnr") || strings.Contains(headerLower, "zuname") ||
		strings.Contains(headerLower, "bgr") || strings.Contains(headerLower, "bbgl") {
		return FormatBMD
	}

	// RZL indicators
	if strings.Contains(headerLower, "sozialversicherungsnummer") ||
		strings.Contains(headerLower, "beitragsgrundlage_monatlich") {
		return FormatRZL
	}

	return FormatGeneric
}

// Helper functions

func removeBOM(data []byte) []byte {
	// UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

func detectDelimiter(data []byte) rune {
	reader := bufio.NewReader(bytes.NewReader(data))
	firstLine, _ := reader.ReadString('\n')

	semicolons := strings.Count(firstLine, ";")
	commas := strings.Count(firstLine, ",")
	tabs := strings.Count(firstLine, "\t")

	if semicolons > commas && semicolons > tabs {
		return ';'
	}
	if tabs > commas && tabs > semicolons {
		return '\t'
	}
	return ','
}

func parseAmount(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	// Handle European number format (1.234,56 -> 1234.56)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")

	// Remove currency symbols
	s = strings.TrimPrefix(s, "€")
	s = strings.TrimPrefix(s, "EUR")
	s = strings.TrimSpace(s)

	return strconv.ParseFloat(s, 64)
}

func parseDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}

	// Try various date formats
	formats := []string{
		"2006-01-02",      // ISO
		"02.01.2006",      // German
		"02/01/2006",      // European
		"01/02/2006",      // US
		"2.1.2006",        // German short
		"2.1.06",          // German very short
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	return "", fmt.Errorf("unbekanntes Datumsformat: %s", s)
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// GetCSVTemplate returns a template CSV for mBGM import
func GetCSVTemplate(format CSVFormat) string {
	switch format {
	case FormatBMD:
		return "SVNR;Zuname;Vorname;GebDat;BGR;BBGL;SZ;Von;Bis;WStd\n" +
			"1234010180;Muster;Max;01.01.1980;D1;3500.00;0.00;;;38.5\n"
	case FormatRZL:
		return "Sozialversicherungsnummer;Familienname;Vorname;Geburtsdatum;Beitragsgruppe;Beitragsgrundlage_monatlich;Sonderzahlung;beschaeftigt_von;beschaeftigt_bis;Arbeitszeit_Woche\n" +
			"1234010180;Muster;Max;01.01.1980;D1;3500.00;0.00;;;38.5\n"
	default:
		return "SV_Nummer;Familienname;Vorname;Geburtsdatum;Beitragsgruppe;Beitragsgrundlage;Sonderzahlung;Von_Datum;Bis_Datum;Wochenstunden\n" +
			"1234010180;Muster;Max;1980-01-01;D1;3500.00;0.00;;;38.5\n"
	}
}

// CSVFormatSpec describes a CSV format
type CSVFormatSpec struct {
	Name        CSVFormat `json:"name"`
	Description string    `json:"description"`
	Delimiter   string    `json:"delimiter"`
	Encoding    string    `json:"encoding"`
	Columns     []CSVColumnSpec `json:"columns"`
}

// CSVColumnSpec describes a single CSV column
type CSVColumnSpec struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Example  string `json:"example"`
}

// GetFormatSpecs returns specifications for all supported CSV formats
func GetFormatSpecs() []CSVFormatSpec {
	return []CSVFormatSpec{
		{
			Name:        FormatGeneric,
			Description: "Generisches CSV Format",
			Delimiter:   "Semikolon oder Komma (auto)",
			Encoding:    "UTF-8",
			Columns: []CSVColumnSpec{
				{Name: "SV_Nummer", Required: true, Type: "text", Example: "1234010180"},
				{Name: "Familienname", Required: true, Type: "text", Example: "Muster"},
				{Name: "Vorname", Required: false, Type: "text", Example: "Max"},
				{Name: "Geburtsdatum", Required: false, Type: "date", Example: "1980-01-01"},
				{Name: "Beitragsgruppe", Required: true, Type: "text", Example: "D1"},
				{Name: "Beitragsgrundlage", Required: true, Type: "number", Example: "3500.00"},
				{Name: "Sonderzahlung", Required: false, Type: "number", Example: "0.00"},
				{Name: "Wochenstunden", Required: false, Type: "number", Example: "38.5"},
			},
		},
		{
			Name:        FormatBMD,
			Description: "BMD Export Format",
			Delimiter:   "Semikolon",
			Encoding:    "Windows-1252 oder UTF-8",
			Columns: []CSVColumnSpec{
				{Name: "SVNR", Required: true, Type: "text", Example: "1234010180"},
				{Name: "Zuname", Required: true, Type: "text", Example: "Muster"},
				{Name: "Vorname", Required: false, Type: "text", Example: "Max"},
				{Name: "GebDat", Required: false, Type: "date", Example: "01.01.1980"},
				{Name: "BGR", Required: true, Type: "text", Example: "D1"},
				{Name: "BBGL", Required: true, Type: "number", Example: "3500,00"},
			},
		},
		{
			Name:        FormatRZL,
			Description: "RZL Export Format",
			Delimiter:   "Semikolon",
			Encoding:    "UTF-8",
			Columns: []CSVColumnSpec{
				{Name: "Sozialversicherungsnummer", Required: true, Type: "text", Example: "1234010180"},
				{Name: "Familienname", Required: true, Type: "text", Example: "Muster"},
				{Name: "Vorname", Required: false, Type: "text", Example: "Max"},
				{Name: "Geburtsdatum", Required: false, Type: "date", Example: "01.01.1980"},
				{Name: "Beitragsgruppe", Required: true, Type: "text", Example: "D1"},
				{Name: "Beitragsgrundlage_monatlich", Required: true, Type: "number", Example: "3500,00"},
			},
		},
	}
}
