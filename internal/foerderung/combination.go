package foerderung

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// CombinationService handles Förderung combination logic
type CombinationService struct {
	repo *Repository
}

// NewCombinationService creates a new combination service
func NewCombinationService(repo *Repository) *CombinationService {
	return &CombinationService{repo: repo}
}

// CombinationResult represents a combinable Förderung with analysis
type CombinationResult struct {
	Foerderung        *Foerderung `json:"foerderung"`
	CombinationType   string      `json:"combination_type"` // "explicit", "inferred"
	CombinedMaxAmount int         `json:"combined_max_amount,omitempty"`
	Notes             string      `json:"notes,omitempty"`
}

// CombinationAnalysis represents the full combination analysis
type CombinationAnalysis struct {
	PrimaryFoerderung *Foerderung          `json:"primary_foerderung"`
	CombinableWith    []*CombinationResult `json:"combinable_with"`
	NotCombinableWith []*Foerderung        `json:"not_combinable_with"`
	TotalMaxAmount    int                  `json:"total_max_amount"`
	Warnings          []string             `json:"warnings,omitempty"`
}

// GetCombinablePrograms retrieves all programs that can be combined with the given one
func (s *CombinationService) GetCombinablePrograms(ctx context.Context, foerderungID uuid.UUID) (*CombinationAnalysis, error) {
	// Get the primary Förderung
	primary, err := s.repo.GetByID(ctx, foerderungID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary foerderung: %w", err)
	}

	analysis := &CombinationAnalysis{
		PrimaryFoerderung: primary,
		CombinableWith:    make([]*CombinationResult, 0),
		NotCombinableWith: make([]*Foerderung, 0),
	}

	// Add primary amount to total
	if primary.MaxAmount != nil {
		analysis.TotalMaxAmount = *primary.MaxAmount
	}

	// Get explicitly combinable programs
	for _, combID := range primary.CombinableWith {
		f, err := s.repo.GetByID(ctx, combID)
		if err != nil {
			continue // Skip if not found
		}

		if f.Status != StatusActive {
			continue // Skip inactive
		}

		result := &CombinationResult{
			Foerderung:      f,
			CombinationType: "explicit",
		}

		if f.MaxAmount != nil {
			result.CombinedMaxAmount = analysis.TotalMaxAmount + *f.MaxAmount
			analysis.TotalMaxAmount = result.CombinedMaxAmount
		}

		// Check for provider overlap (common combination)
		if f.Provider == primary.Provider {
			result.Notes = "Gleicher Fördergeber - kombinierbar"
		}

		analysis.CombinableWith = append(analysis.CombinableWith, result)
	}

	// Get explicitly NOT combinable programs
	for _, notCombID := range primary.NotCombinableWith {
		f, err := s.repo.GetByID(ctx, notCombID)
		if err != nil {
			continue
		}
		analysis.NotCombinableWith = append(analysis.NotCombinableWith, f)
	}

	// Infer additional combinations based on rules
	inferred, warnings := s.inferCombinations(ctx, primary)
	analysis.CombinableWith = append(analysis.CombinableWith, inferred...)
	analysis.Warnings = warnings

	return analysis, nil
}

// inferCombinations infers potential combinations based on business rules
func (s *CombinationService) inferCombinations(ctx context.Context, primary *Foerderung) ([]*CombinationResult, []string) {
	var results []*CombinationResult
	var warnings []string

	// Get all active Förderungen
	all, err := s.repo.ListActive(ctx)
	if err != nil {
		return results, []string{"Fehler beim Laden der Förderungen für Kombinationsanalyse"}
	}

	// Already processed IDs
	processed := make(map[uuid.UUID]bool)
	processed[primary.ID] = true
	for _, id := range primary.CombinableWith {
		processed[id] = true
	}
	for _, id := range primary.NotCombinableWith {
		processed[id] = true
	}

	for _, f := range all {
		if processed[f.ID] {
			continue
		}

		// Rule 1: Different provider levels (Bund + Land) are often combinable
		if s.areDifferentLevels(primary.Provider, f.Provider) {
			// Check if primary explicitly excludes this
			if s.isExplicitlyExcluded(primary, f) {
				continue
			}

			results = append(results, &CombinationResult{
				Foerderung:      f,
				CombinationType: "inferred",
				Notes:           "Bundes- und Landesförderung oft kombinierbar",
			})

			warnings = append(warnings, fmt.Sprintf(
				"Kombination von %s mit %s sollte im Einzelfall geprüft werden",
				primary.Name, f.Name,
			))
		}

		// Rule 2: EU + National programs
		if s.isEUProgram(primary) && s.isNationalProgram(f) || s.isEUProgram(f) && s.isNationalProgram(primary) {
			if s.isExplicitlyExcluded(primary, f) {
				continue
			}

			results = append(results, &CombinationResult{
				Foerderung:      f,
				CombinationType: "inferred",
				Notes:           "EU + nationale Förderung kombinierbar (Beihilfengrenzen beachten)",
			})

			warnings = append(warnings, "EU-Beihilfengrenzen bei Kombination beachten (max. 60-80% je nach Region)")
		}
	}

	return results, warnings
}

// areDifferentLevels checks if providers are from different government levels
func (s *CombinationService) areDifferentLevels(provider1, provider2 string) bool {
	federalProviders := map[string]bool{
		"AWS": true, "FFG": true, "OeKB": true, "WKO": true, "AMS": true,
	}

	stateProviders := map[string]bool{
		"SFG": true, "WIBAG": true, "WIST": true, "KWF": true, "WKB": true,
		"WF_Sbg": true, "Standortagentur_Tirol": true, "WLV": true,
		"WWFF": true, "EcoPlus": true,
	}

	isFederal1 := federalProviders[provider1]
	isState1 := stateProviders[provider1]

	isFederal2 := federalProviders[provider2]
	isState2 := stateProviders[provider2]

	return (isFederal1 && isState2) || (isState1 && isFederal2)
}

// isEUProgram checks if a Förderung is an EU program
func (s *CombinationService) isEUProgram(f *Foerderung) bool {
	return f.Provider == "EU" || contains(f.Categories, "eu")
}

// isNationalProgram checks if a Förderung is a national program
func (s *CombinationService) isNationalProgram(f *Foerderung) bool {
	return f.Provider == "AWS" || f.Provider == "FFG" || f.Provider == "WKO" || f.Provider == "OeKB"
}

// isExplicitlyExcluded checks if f2 is in f1's not_combinable_with list
func (s *CombinationService) isExplicitlyExcluded(f1, f2 *Foerderung) bool {
	for _, id := range f1.NotCombinableWith {
		if id == f2.ID {
			return true
		}
	}
	return false
}

// contains checks if a slice contains a string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// ValidateCombination validates if two Förderungen can be combined
func (s *CombinationService) ValidateCombination(ctx context.Context, foerderungID1, foerderungID2 uuid.UUID) (*CombinationValidation, error) {
	f1, err := s.repo.GetByID(ctx, foerderungID1)
	if err != nil {
		return nil, fmt.Errorf("failed to get foerderung 1: %w", err)
	}

	f2, err := s.repo.GetByID(ctx, foerderungID2)
	if err != nil {
		return nil, fmt.Errorf("failed to get foerderung 2: %w", err)
	}

	validation := &CombinationValidation{
		Foerderung1: f1,
		Foerderung2: f2,
		Warnings:    make([]string, 0),
	}

	// Check explicit exclusion
	if s.isExplicitlyExcluded(f1, f2) || s.isExplicitlyExcluded(f2, f1) {
		validation.IsValid = false
		validation.Reason = "Diese Förderungen sind explizit nicht kombinierbar"
		return validation, nil
	}

	// Check explicit inclusion
	for _, id := range f1.CombinableWith {
		if id == f2.ID {
			validation.IsValid = true
			validation.Reason = "Diese Förderungen sind explizit kombinierbar"
			return validation, nil
		}
	}

	// Apply inference rules
	if s.areDifferentLevels(f1.Provider, f2.Provider) {
		validation.IsValid = true
		validation.Reason = "Bundes- und Landesförderungen sind in der Regel kombinierbar"
		validation.Warnings = append(validation.Warnings, "Im Einzelfall prüfen - Förderbedingungen können abweichen")
		return validation, nil
	}

	// Default: uncertain
	validation.IsValid = false
	validation.Reason = "Keine Kombinationsinformation verfügbar - bitte im Einzelfall prüfen"
	validation.Warnings = append(validation.Warnings, "Kontaktieren Sie die Förderstellen für verbindliche Auskunft")

	return validation, nil
}

// CombinationValidation represents the validation result
type CombinationValidation struct {
	Foerderung1 *Foerderung `json:"foerderung_1"`
	Foerderung2 *Foerderung `json:"foerderung_2"`
	IsValid     bool        `json:"is_valid"`
	Reason      string      `json:"reason"`
	Warnings    []string    `json:"warnings,omitempty"`
}
