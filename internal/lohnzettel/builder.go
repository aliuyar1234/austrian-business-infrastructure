package lohnzettel

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

// Builder generates L16 XML documents for ELDA submission
type Builder struct{}

// NewBuilder creates a new L16 XML builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildXML generates the complete L16 XML document
func (b *Builder) BuildXML(l *elda.Lohnzettel) ([]byte, error) {
	doc := b.buildDocument(l)

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal L16 XML: %w", err)
	}

	// Add XML header
	result := []byte(xml.Header)
	result = append(result, output...)

	return result, nil
}

// BuildXMLString generates the L16 XML as a string
func (b *Builder) BuildXMLString(l *elda.Lohnzettel) (string, error) {
	data, err := b.BuildXML(l)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// buildDocument creates the L16Document structure
func (b *Builder) buildDocument(l *elda.Lohnzettel) *elda.L16Document {
	doc := &elda.L16Document{
		XMLNS: elda.ELDANS,
		Jahr:  l.Year,
		Arbeitnehmer: elda.L16Arbeitnehmer{
			SVNummer:     l.SVNummer,
			Familienname: l.Familienname,
			Vorname:      l.Vorname,
		},
		Bezuege: b.buildBezuege(&l.L16Data),
		Zeiten:  b.buildZeiten(&l.L16Data),
		Abzuege: b.buildAbzuege(&l.L16Data),
	}

	if l.Geburtsdatum != nil {
		doc.Arbeitnehmer.Geburtsdatum = l.Geburtsdatum.Format("2006-01-02")
	}

	return doc
}

// buildBezuege creates the income section
func (b *Builder) buildBezuege(data *elda.L16Data) elda.L16Bezuege {
	bezuege := elda.L16Bezuege{
		KZ210: formatAmount(data.KZ210),
	}

	if data.KZ215 > 0 {
		bezuege.KZ215 = formatAmount(data.KZ215)
	}
	if data.KZ220 > 0 {
		bezuege.KZ220 = formatAmount(data.KZ220)
	}
	if data.KZ230 > 0 {
		bezuege.KZ230 = formatAmount(data.KZ230)
	}
	if data.KZ243 > 0 {
		bezuege.KZ243 = formatAmount(data.KZ243)
	}
	if data.KZ245 > 0 {
		bezuege.KZ245 = formatAmount(data.KZ245)
	}
	if data.KZ250 > 0 {
		bezuege.KZ250 = formatAmount(data.KZ250)
	}
	if data.KZ260 > 0 {
		bezuege.KZ260 = formatAmount(data.KZ260)
	}

	return bezuege
}

// buildZeiten creates the employment period section
func (b *Builder) buildZeiten(data *elda.L16Data) elda.L16Zeiten {
	zeiten := elda.L16Zeiten{}

	if data.BeschaeftigungVon != "" {
		zeiten.BeschaeftigungVon = data.BeschaeftigungVon
	}
	if data.BeschaeftigungBis != "" {
		zeiten.BeschaeftigungBis = data.BeschaeftigungBis
	}
	if data.ArbeitsTage > 0 {
		zeiten.ArbeitsTage = data.ArbeitsTage
	}

	return zeiten
}

// buildAbzuege creates the deduction section
func (b *Builder) buildAbzuege(data *elda.L16Data) elda.L16Abzuege {
	abzuege := elda.L16Abzuege{}

	if data.KZ226 > 0 {
		abzuege.KZ226 = formatAmount(data.KZ226)
	}
	if data.KZ231 > 0 {
		abzuege.KZ231 = formatAmount(data.KZ231)
	}
	if data.KZ235 > 0 {
		abzuege.KZ235 = formatAmount(data.KZ235)
	}
	if data.KZ240 > 0 {
		abzuege.KZ240 = formatAmount(data.KZ240)
	}

	return abzuege
}

// formatAmount formats a float64 as a decimal string for XML
func formatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

// PreviewXML generates a preview of the XML
func (b *Builder) PreviewXML(l *elda.Lohnzettel) (*XMLPreview, error) {
	xmlData, err := b.BuildXML(l)
	if err != nil {
		return nil, err
	}

	return &XMLPreview{
		XML:         string(xmlData),
		Year:        l.Year,
		SVNummer:    l.SVNummer,
		Name:        fmt.Sprintf("%s %s", l.Vorname, l.Familienname),
		Bruttobezug: l.L16Data.KZ210,
		Lohnsteuer:  l.L16Data.KZ220,
		SVBeitraege: l.L16Data.KZ230,
		GeneratedAt: time.Now(),
	}, nil
}

// XMLPreview contains a preview of the generated XML
type XMLPreview struct {
	XML         string    `json:"xml"`
	Year        int       `json:"year"`
	SVNummer    string    `json:"sv_nummer"`
	Name        string    `json:"name"`
	Bruttobezug float64   `json:"bruttobezug"`
	Lohnsteuer  float64   `json:"lohnsteuer"`
	SVBeitraege float64   `json:"sv_beitraege"`
	GeneratedAt time.Time `json:"generated_at"`
}

// BuildBerichtigungXML generates a correction L16 XML
func (b *Builder) BuildBerichtigungXML(original, corrected *elda.Lohnzettel) ([]byte, error) {
	// Mark as Berichtigung
	corrected.IsBerichtigung = true
	return b.BuildXML(corrected)
}

// L16Summary provides a summary of an L16
type L16Summary struct {
	Year           int       `json:"year"`
	SVNummer       string    `json:"sv_nummer"`
	Name           string    `json:"name"`
	Bruttobezug    float64   `json:"bruttobezug"`
	SonstigeBezuge float64   `json:"sonstige_bezuege"`
	Lohnsteuer     float64   `json:"lohnsteuer"`
	SVBeitraege    float64   `json:"sv_beitraege"`
	XMLSize        int       `json:"xml_size_bytes"`
	Deadline       time.Time `json:"deadline"`
	DaysUntil      int       `json:"days_until_deadline"`
}

// BuildSummary creates a summary of the L16
func (b *Builder) BuildSummary(l *elda.Lohnzettel) (*L16Summary, error) {
	xmlData, err := b.BuildXML(l)
	if err != nil {
		return nil, err
	}

	deadline := elda.GetL16Deadline(l.Year)
	daysUntil := elda.DaysUntilL16Deadline(l.Year)

	return &L16Summary{
		Year:           l.Year,
		SVNummer:       l.SVNummer,
		Name:           fmt.Sprintf("%s %s", l.Vorname, l.Familienname),
		Bruttobezug:    l.L16Data.KZ210,
		SonstigeBezuge: l.L16Data.KZ215,
		Lohnsteuer:     l.L16Data.KZ220,
		SVBeitraege:    l.L16Data.KZ230,
		XMLSize:        len(xmlData),
		Deadline:       deadline,
		DaysUntil:      daysUntil,
	}, nil
}

// BatchXMLResult contains the result of building batch XML
type BatchXMLResult struct {
	LohnzettelID uuid.UUID `json:"lohnzettel_id"`
	XML          []byte    `json:"-"`
	Error        error     `json:"error,omitempty"`
}

// BuildBatchXML generates XML for multiple L16 documents
func (b *Builder) BuildBatchXML(lohnzettel []*elda.Lohnzettel) []BatchXMLResult {
	results := make([]BatchXMLResult, len(lohnzettel))

	for i, l := range lohnzettel {
		xmlData, err := b.BuildXML(l)
		results[i] = BatchXMLResult{
			LohnzettelID: l.ID,
			XML:          xmlData,
			Error:        err,
		}
	}

	return results
}
