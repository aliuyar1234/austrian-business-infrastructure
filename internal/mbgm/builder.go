package mbgm

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"austrian-business-infrastructure/internal/elda"
)

// Builder generates mBGM XML documents for ELDA submission
type Builder struct{}

// NewBuilder creates a new mBGM XML builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildXML generates the complete mBGM XML document
func (b *Builder) BuildXML(mbgm *elda.MBGM, dienstgeberNr string) ([]byte, error) {
	doc := b.buildDocument(mbgm, dienstgeberNr)

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mBGM XML: %w", err)
	}

	// Add XML header
	result := []byte(xml.Header)
	result = append(result, output...)

	return result, nil
}

// BuildXMLString generates the mBGM XML as a string
func (b *Builder) BuildXMLString(mbgm *elda.MBGM, dienstgeberNr string) (string, error) {
	data, err := b.BuildXML(mbgm, dienstgeberNr)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// buildDocument creates the MBGMDocument structure
func (b *Builder) buildDocument(mbgm *elda.MBGM, dienstgeberNr string) *elda.MBGMDocument {
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
		xmlPos := b.buildPosition(pos)
		doc.Positionen = append(doc.Positionen, xmlPos)
	}

	return doc
}

// buildPosition converts an MBGMPosition to XML format
func (b *Builder) buildPosition(pos *elda.MBGMPosition) elda.MBGMXMLPos {
	xmlPos := elda.MBGMXMLPos{
		SVNummer:          pos.SVNummer,
		Familienname:      pos.Familienname,
		Vorname:           pos.Vorname,
		Beitragsgruppe:    pos.Beitragsgruppe,
		Beitragsgrundlage: formatAmount(pos.Beitragsgrundlage),
	}

	// Optional fields
	if pos.Geburtsdatum != nil {
		xmlPos.Geburtsdatum = pos.Geburtsdatum.Format("2006-01-02")
	}

	if pos.Sonderzahlung > 0 {
		xmlPos.Sonderzahlung = formatAmount(pos.Sonderzahlung)
	}

	if pos.VonDatum != nil {
		xmlPos.VonDatum = pos.VonDatum.Format("2006-01-02")
	}

	if pos.BisDatum != nil {
		xmlPos.BisDatum = pos.BisDatum.Format("2006-01-02")
	}

	if pos.Wochenstunden != nil {
		wh := formatAmount(*pos.Wochenstunden)
		xmlPos.Wochenstunden = &wh
	}

	return xmlPos
}

// formatAmount formats a float64 as a decimal string for XML
func formatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

// BuildCorrectionXML generates a correction mBGM XML
func (b *Builder) BuildCorrectionXML(original *elda.MBGM, corrected *elda.MBGM, dienstgeberNr string) ([]byte, error) {
	// Mark as correction
	corrected.IsCorrection = true

	// Build the corrected document
	return b.BuildXML(corrected, dienstgeberNr)
}

// PreviewXML generates a preview of the XML without actually submitting
// This is useful for validation and user review
func (b *Builder) PreviewXML(mbgm *elda.MBGM, dienstgeberNr string) (*XMLPreview, error) {
	xmlData, err := b.BuildXML(mbgm, dienstgeberNr)
	if err != nil {
		return nil, err
	}

	preview := &XMLPreview{
		XML:               string(xmlData),
		PositionCount:     len(mbgm.Positionen),
		TotalBeitragsBase: calculateTotalBeitragsgrundlage(mbgm),
		Year:              mbgm.Year,
		Month:             mbgm.Month,
		IsCorrection:      mbgm.IsCorrection,
		GeneratedAt:       time.Now(),
	}

	return preview, nil
}

// XMLPreview contains a preview of the generated XML
type XMLPreview struct {
	XML               string    `json:"xml"`
	PositionCount     int       `json:"position_count"`
	TotalBeitragsBase float64   `json:"total_beitragsgrundlage"`
	Year              int       `json:"year"`
	Month             int       `json:"month"`
	IsCorrection      bool      `json:"is_correction"`
	GeneratedAt       time.Time `json:"generated_at"`
}

// calculateTotalBeitragsgrundlage sums up all Beitragsgrundlagen
func calculateTotalBeitragsgrundlage(mbgm *elda.MBGM) float64 {
	var total float64
	for _, pos := range mbgm.Positionen {
		total += pos.Beitragsgrundlage
	}
	return total
}

// BuildBatchXML generates XML for multiple mBGMs (if ELDA supports batch)
func (b *Builder) BuildBatchXML(mbgms []*elda.MBGM, dienstgeberNr string) ([][]byte, error) {
	results := make([][]byte, 0, len(mbgms))

	for _, mbgm := range mbgms {
		xmlData, err := b.BuildXML(mbgm, dienstgeberNr)
		if err != nil {
			return nil, fmt.Errorf("failed to build XML for mBGM %s: %w", mbgm.ID, err)
		}
		results = append(results, xmlData)
	}

	return results, nil
}

// ValidateGeneratedXML validates the generated XML against schema rules
func (b *Builder) ValidateGeneratedXML(xmlData []byte) error {
	// Parse back to verify structure
	var doc elda.MBGMDocument
	if err := xml.Unmarshal(xmlData, &doc); err != nil {
		return fmt.Errorf("generated XML is invalid: %w", err)
	}

	// Basic structure validation
	if doc.Kopf.DienstgeberNummer == "" {
		return fmt.Errorf("DienstgeberNummer is required")
	}

	if doc.Kopf.Jahr < 2020 {
		return fmt.Errorf("invalid year: %d", doc.Kopf.Jahr)
	}

	if doc.Kopf.Monat < 1 || doc.Kopf.Monat > 12 {
		return fmt.Errorf("invalid month: %d", doc.Kopf.Monat)
	}

	if len(doc.Positionen) == 0 {
		return fmt.Errorf("at least one position is required")
	}

	return nil
}

// MBGMSummary provides a summary of an mBGM for logging/display
type MBGMSummary struct {
	Year              int       `json:"year"`
	Month             int       `json:"month"`
	PositionCount     int       `json:"position_count"`
	TotalBeitragsBase float64   `json:"total_beitragsgrundlage"`
	IsCorrection      bool      `json:"is_correction"`
	XMLSize           int       `json:"xml_size_bytes"`
	Deadline          time.Time `json:"deadline"`
	DaysUntilDeadline int       `json:"days_until_deadline"`
}

// BuildSummary creates a summary of the mBGM
func (b *Builder) BuildSummary(mbgm *elda.MBGM, dienstgeberNr string) (*MBGMSummary, error) {
	xmlData, err := b.BuildXML(mbgm, dienstgeberNr)
	if err != nil {
		return nil, err
	}

	deadline := elda.GetMBGMDeadline(mbgm.Year, mbgm.Month)
	daysUntil := int(time.Until(deadline).Hours() / 24)
	if daysUntil < 0 {
		daysUntil = 0
	}

	return &MBGMSummary{
		Year:              mbgm.Year,
		Month:             mbgm.Month,
		PositionCount:     len(mbgm.Positionen),
		TotalBeitragsBase: calculateTotalBeitragsgrundlage(mbgm),
		IsCorrection:      mbgm.IsCorrection,
		XMLSize:           len(xmlData),
		Deadline:          deadline,
		DaysUntilDeadline: daysUntil,
	}, nil
}
