package foerderung

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// JSONFoerderung represents a Förderung in the JSON import format
// Matches the TypeScript project format
type JSONFoerderung struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ShortName   string  `json:"shortName,omitempty"`
	Description string  `json:"description,omitempty"`
	Provider    string  `json:"provider"`
	Type        string  `json:"type"`

	FundingRateMin float64 `json:"fundingRateMin,omitempty"`
	FundingRateMax float64 `json:"fundingRateMax,omitempty"`
	MaxAmount      int     `json:"maxAmount,omitempty"`
	MinAmount      int     `json:"minAmount,omitempty"`

	TargetSize       string   `json:"targetSize,omitempty"`
	TargetAge        string   `json:"targetAge,omitempty"`
	TargetLegalForms []string `json:"targetLegalForms,omitempty"`
	TargetIndustries []string `json:"targetIndustries,omitempty"`
	TargetStates     []string `json:"targetStates,omitempty"`

	Topics     []string `json:"topics"`
	Categories []string `json:"categories,omitempty"`

	Requirements        string                 `json:"requirements,omitempty"`
	EligibilityCriteria map[string]interface{} `json:"eligibilityCriteria,omitempty"`

	ApplicationDeadline string `json:"applicationDeadline,omitempty"`
	DeadlineType        string `json:"deadlineType,omitempty"`
	CallStart           string `json:"callStart,omitempty"`
	CallEnd             string `json:"callEnd,omitempty"`

	URL            string `json:"url,omitempty"`
	ApplicationURL string `json:"applicationUrl,omitempty"`
	GuidelineURL   string `json:"guidelineUrl,omitempty"`

	CombinableWith    []string `json:"combinableWith,omitempty"`
	NotCombinableWith []string `json:"notCombinableWith,omitempty"`

	Status        string `json:"status,omitempty"`
	IsHighlighted bool   `json:"isHighlighted,omitempty"`
}

// Importer handles importing Förderungen from JSON files
type Importer struct {
	repo *Repository
}

// NewImporter creates a new importer
func NewImporter(repo *Repository) *Importer {
	return &Importer{repo: repo}
}

// ImportResult contains the result of an import operation
type ImportResult struct {
	TotalRecords int      `json:"total_records"`
	Imported     int      `json:"imported"`
	Updated      int      `json:"updated"`
	Failed       int      `json:"failed"`
	Errors       []string `json:"errors,omitempty"`
}

// ImportFromFile imports Förderungen from a JSON file
func (i *Importer) ImportFromFile(ctx context.Context, filePath string) (*ImportResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return i.ImportFromJSON(ctx, data, filepath.Base(filePath))
}

// ImportFromJSON imports Förderungen from JSON data
func (i *Importer) ImportFromJSON(ctx context.Context, data []byte, source string) (*ImportResult, error) {
	var foerderungen []JSONFoerderung
	if err := json.Unmarshal(data, &foerderungen); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &ImportResult{
		TotalRecords: len(foerderungen),
		Errors:       []string{},
	}

	for idx, jf := range foerderungen {
		f, err := i.convertJSONToFoerderung(&jf, source)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("record %d: %s", idx, err.Error()))
			continue
		}

		// Check if exists by source ID
		existing, err := i.repo.GetBySourceID(ctx, source, jf.ID)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("record %d: %s", idx, err.Error()))
			continue
		}

		if existing != nil {
			// Update existing
			f.ID = existing.ID
			f.CreatedAt = existing.CreatedAt
			if err := i.repo.Update(ctx, f); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("record %d: %s", idx, err.Error()))
				continue
			}
			result.Updated++
		} else {
			// Create new
			if err := i.repo.Create(ctx, f); err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("record %d: %s", idx, err.Error()))
				continue
			}
			result.Imported++
		}
	}

	return result, nil
}

// ImportFromDirectory imports all JSON files from a directory
func (i *Importer) ImportFromDirectory(ctx context.Context, dirPath string) (*ImportResult, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to glob directory: %w", err)
	}

	totalResult := &ImportResult{
		Errors: []string{},
	}

	for _, file := range files {
		result, err := i.ImportFromFile(ctx, file)
		if err != nil {
			totalResult.Errors = append(totalResult.Errors, fmt.Sprintf("%s: %s", filepath.Base(file), err.Error()))
			continue
		}

		totalResult.TotalRecords += result.TotalRecords
		totalResult.Imported += result.Imported
		totalResult.Updated += result.Updated
		totalResult.Failed += result.Failed
		totalResult.Errors = append(totalResult.Errors, result.Errors...)
	}

	return totalResult, nil
}

func (i *Importer) convertJSONToFoerderung(jf *JSONFoerderung, source string) (*Foerderung, error) {
	if jf.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if jf.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if jf.Type == "" {
		return nil, fmt.Errorf("type is required")
	}

	f := &Foerderung{
		ID:       uuid.New(),
		Name:     jf.Name,
		Provider: jf.Provider,
		Type:     FoerderungType(strings.ToLower(jf.Type)),
		Topics:   jf.Topics,
		Status:   StatusActive,
	}

	// Optional string fields
	if jf.ShortName != "" {
		f.ShortName = &jf.ShortName
	}
	if jf.Description != "" {
		f.Description = &jf.Description
	}
	if jf.Requirements != "" {
		f.Requirements = &jf.Requirements
	}
	if jf.URL != "" {
		f.URL = &jf.URL
	}
	if jf.ApplicationURL != "" {
		f.ApplicationURL = &jf.ApplicationURL
	}
	if jf.GuidelineURL != "" {
		f.GuidelineURL = &jf.GuidelineURL
	}

	// Numeric fields
	if jf.FundingRateMin > 0 {
		f.FundingRateMin = &jf.FundingRateMin
	}
	if jf.FundingRateMax > 0 {
		f.FundingRateMax = &jf.FundingRateMax
	}
	if jf.MaxAmount > 0 {
		f.MaxAmount = &jf.MaxAmount
	}
	if jf.MinAmount > 0 {
		f.MinAmount = &jf.MinAmount
	}

	// Target fields
	if jf.TargetSize != "" {
		ts := TargetSize(strings.ToLower(jf.TargetSize))
		f.TargetSize = &ts
	}
	if jf.TargetAge != "" {
		f.TargetAge = &jf.TargetAge
	}
	f.TargetLegalForms = jf.TargetLegalForms
	f.TargetIndustries = jf.TargetIndustries
	f.TargetStates = jf.TargetStates
	f.Categories = jf.Categories

	// Eligibility criteria
	if len(jf.EligibilityCriteria) > 0 {
		data, err := json.Marshal(jf.EligibilityCriteria)
		if err == nil {
			f.EligibilityCriteria = data
		}
	}

	// Dates
	if jf.ApplicationDeadline != "" {
		t, err := parseImportDate(jf.ApplicationDeadline)
		if err == nil {
			f.ApplicationDeadline = &t
		}
	}
	if jf.CallStart != "" {
		t, err := parseImportDate(jf.CallStart)
		if err == nil {
			f.CallStart = &t
		}
	}
	if jf.CallEnd != "" {
		t, err := parseImportDate(jf.CallEnd)
		if err == nil {
			f.CallEnd = &t
		}
	}

	// Deadline type
	if jf.DeadlineType != "" {
		dt := DeadlineType(strings.ToLower(jf.DeadlineType))
		f.DeadlineType = &dt
	}

	// Status
	if jf.Status != "" {
		f.Status = FoerderungStatus(strings.ToLower(jf.Status))
	}
	f.IsHighlighted = jf.IsHighlighted

	// Source tracking
	sourceStr := source
	f.Source = &sourceStr
	if jf.ID != "" {
		f.SourceID = &jf.ID
	}
	now := time.Now()
	f.LastUpdatedAt = &now

	return f, nil
}

func parseImportDate(s string) (time.Time, error) {
	// Try different date formats
	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// GetSeedData returns seed data for the initial 74 Förderungen
// This can be used when the JSON files are not available
func GetSeedData() []JSONFoerderung {
	return []JSONFoerderung{
		// AWS Programs
		{
			ID:             "aws-garantie",
			Name:           "AWS Garantie für Investitionen",
			ShortName:      "AWS Garantie",
			Provider:       "AWS",
			Type:           "garantie",
			FundingRateMax: 0.80,
			MaxAmount:      2500000,
			TargetSize:     "kmu",
			TargetStates:   []string{"alle"},
			Topics:         []string{"investition", "wachstum", "innovation"},
			URL:            "https://www.aws.at/garantien/",
			DeadlineType:   "rolling",
			Description:    "Haftungen für Investitionskredite bis zu 80% der Kreditsumme",
		},
		{
			ID:             "aws-erp-kredit",
			Name:           "AWS ERP-Kredit",
			ShortName:      "ERP-Kredit",
			Provider:       "AWS",
			Type:           "kredit",
			FundingRateMax: 1.00,
			MaxAmount:      7500000,
			TargetSize:     "kmu",
			TargetStates:   []string{"alle"},
			Topics:         []string{"investition", "innovation", "digitalisierung"},
			URL:            "https://www.aws.at/erp-kredite/",
			DeadlineType:   "rolling",
			Description:    "Zinsgünstige Kredite für Investitionen in Innovation und Digitalisierung",
		},
		{
			ID:             "aws-jungunternehmer",
			Name:           "AWS Jungunternehmer-Förderung",
			ShortName:      "Jungunternehmer",
			Provider:       "AWS",
			Type:           "zuschuss",
			FundingRateMax: 0.50,
			MaxAmount:      100000,
			TargetSize:     "startup",
			TargetAge:      "gruendung",
			TargetStates:   []string{"alle"},
			Topics:         []string{"gruendung", "innovation"},
			URL:            "https://www.aws.at/jungunternehmer/",
			DeadlineType:   "rolling",
			Description:    "Förderung für innovative Unternehmensgründungen bis 5 Jahre",
		},
		// FFG Programs
		{
			ID:             "ffg-basisprogramm",
			Name:           "FFG Basisprogramm",
			ShortName:      "FFG Basis",
			Provider:       "FFG",
			Type:           "zuschuss",
			FundingRateMax: 0.50,
			MaxAmount:      500000,
			TargetSize:     "alle",
			TargetStates:   []string{"alle"},
			Topics:         []string{"forschung", "entwicklung", "innovation"},
			URL:            "https://www.ffg.at/basisprogramm",
			DeadlineType:   "rolling",
			Description:    "Förderung für Forschungs- und Entwicklungsprojekte",
		},
		{
			ID:             "ffg-innovationsscheck",
			Name:           "FFG Innovationsscheck",
			ShortName:      "Innovationsscheck",
			Provider:       "FFG",
			Type:           "zuschuss",
			FundingRateMax: 0.80,
			MaxAmount:      10000,
			TargetSize:     "kmu",
			TargetStates:   []string{"alle"},
			Topics:         []string{"innovation", "forschung"},
			URL:            "https://www.ffg.at/innovationsscheck",
			DeadlineType:   "rolling",
			Description:    "Kleine Förderung für den Einstieg in F&E",
		},
		{
			ID:             "ffg-talente",
			Name:           "FFG Talente Programm",
			ShortName:      "FFG Talente",
			Provider:       "FFG",
			Type:           "zuschuss",
			FundingRateMax: 0.50,
			MaxAmount:      200000,
			TargetSize:     "alle",
			TargetStates:   []string{"alle"},
			Topics:         []string{"forschung", "arbeitsplaetze", "bildung"},
			URL:            "https://www.ffg.at/talente",
			DeadlineType:   "fixed",
			Description:    "Förderung für Nachwuchsförderung und Praktika in F&E",
		},
		// WKO Programs
		{
			ID:             "wko-go-international",
			Name:           "go-international Programm",
			ShortName:      "go-international",
			Provider:       "WKO",
			Type:           "zuschuss",
			FundingRateMax: 0.50,
			MaxAmount:      50000,
			TargetSize:     "kmu",
			TargetStates:   []string{"alle"},
			Topics:         []string{"export", "internationalisierung"},
			URL:            "https://www.go-international.at/",
			DeadlineType:   "rolling",
			Description:    "Förderung für Markteinstieg in neue Exportmärkte",
		},
		// AMS Programs
		{
			ID:             "ams-eingliederungsbeihilfe",
			Name:           "AMS Eingliederungsbeihilfe",
			ShortName:      "AMS EGB",
			Provider:       "AMS",
			Type:           "zuschuss",
			FundingRateMax: 0.66,
			TargetSize:     "alle",
			TargetStates:   []string{"alle"},
			Topics:         []string{"arbeitsplaetze"},
			URL:            "https://www.ams.at/unternehmen/personalsuche-und-foerderungen/eingliederungsbeihilfe",
			DeadlineType:   "rolling",
			Description:    "Zuschuss zu Lohnkosten bei Einstellung von AMS-Kunden",
		},
		// Bundesländer
		{
			ID:             "wien-wirtschaft-innovation",
			Name:           "Wirtschaftsagentur Wien - Innovationsförderung",
			ShortName:      "Wien Innovation",
			Provider:       "Wien",
			Type:           "zuschuss",
			FundingRateMax: 0.50,
			MaxAmount:      200000,
			TargetSize:     "kmu",
			TargetStates:   []string{"Wien"},
			Topics:         []string{"innovation", "digitalisierung"},
			URL:            "https://wirtschaftsagentur.at/",
			DeadlineType:   "rolling",
			Description:    "Innovationsförderung für Wiener Unternehmen",
		},
		{
			ID:             "noe-wirtschaftsfoerderung",
			Name:           "ecoplus NÖ Wirtschaftsförderung",
			ShortName:      "NÖ Wirtschaft",
			Provider:       "Niederösterreich",
			Type:           "zuschuss",
			FundingRateMax: 0.30,
			MaxAmount:      500000,
			TargetSize:     "kmu",
			TargetStates:   []string{"Niederösterreich"},
			Topics:         []string{"investition", "wachstum"},
			URL:            "https://www.ecoplus.at/",
			DeadlineType:   "rolling",
			Description:    "Investitionsförderung für niederösterreichische Unternehmen",
		},
		// EU Programs
		{
			ID:             "eu-horizon",
			Name:           "Horizon Europe",
			ShortName:      "Horizon",
			Provider:       "EU",
			Type:           "zuschuss",
			FundingRateMax: 1.00,
			MaxAmount:      10000000,
			TargetSize:     "alle",
			TargetStates:   []string{"alle"},
			Topics:         []string{"forschung", "innovation", "entwicklung"},
			URL:            "https://ec.europa.eu/info/funding-tenders/opportunities/portal/",
			DeadlineType:   "fixed",
			Description:    "EU-Rahmenprogramm für Forschung und Innovation",
		},
		{
			ID:             "eu-efre",
			Name:           "EFRE Österreich",
			ShortName:      "EFRE",
			Provider:       "EU",
			Type:           "zuschuss",
			FundingRateMax: 0.50,
			MaxAmount:      2000000,
			TargetSize:     "alle",
			TargetStates:   []string{"alle"},
			Topics:         []string{"entwicklung", "innovation", "nachhaltigkeit"},
			URL:            "https://www.efre.gv.at/",
			DeadlineType:   "rolling",
			Description:    "EU-Strukturfonds für regionale Entwicklung",
		},
		// OeKB
		{
			ID:             "oekb-exportgarantie",
			Name:           "OeKB Exportgarantie",
			ShortName:      "Exportgarantie",
			Provider:       "OeKB",
			Type:           "garantie",
			FundingRateMax: 0.95,
			TargetSize:     "alle",
			TargetStates:   []string{"alle"},
			Topics:         []string{"export", "internationalisierung"},
			URL:            "https://www.oekb.at/",
			DeadlineType:   "rolling",
			Description:    "Absicherung von Exportgeschäften gegen Zahlungsausfall",
		},
	}
}
