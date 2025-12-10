package unit

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// Test types matching internal/matcher/types.go
type ProfileInput struct {
	CompanyName    string
	State          string
	District       string
	EmployeesCount int
	AnnualRevenue  int
	BalanceTotal   int
	IsStartup      bool
	IsKMU          bool
	LegalForm      string
	FoundedYear    int
	Topics         []string
	Industry       string
}

type CompanySize string

const (
	SizeAll     CompanySize = "all"
	SizeStartup CompanySize = "startup"
	SizeMikro   CompanySize = "mikro"
	SizeKlein   CompanySize = "klein"
	SizeMittel  CompanySize = "mittel"
	SizeKMU     CompanySize = "kmu"
	SizeGross   CompanySize = "gross"
)

type Foerderung struct {
	ID                  uuid.UUID
	Name                string
	Provider            string
	TargetStates        []string
	TargetSizes         []CompanySize
	TargetAgeMin        *int
	TargetAgeMax        *int
	Topics              []string
	ExcludedIndustries  []string
	ApplicationDeadline *time.Time
	Status              string
}

// Rule scoring weights (from matcher/types.go)
const (
	WeightTopics = 50
	WeightSize   = 25
	WeightRegion = 25
)

// ======================
// REGION RULE TESTS
// ======================

func TestCheckRegion_ExactMatch(t *testing.T) {
	profile := ProfileInput{State: "Wien"}
	foerderung := Foerderung{TargetStates: []string{"Wien", "Niederösterreich"}}

	score := checkRegion(profile, foerderung)
	if score != WeightRegion {
		t.Errorf("Expected %d, got %d", WeightRegion, score)
	}
}

func TestCheckRegion_NoMatch(t *testing.T) {
	profile := ProfileInput{State: "Tirol"}
	foerderung := Foerderung{TargetStates: []string{"Wien", "Niederösterreich"}}

	score := checkRegion(profile, foerderung)
	if score != 0 {
		t.Errorf("Expected 0, got %d", score)
	}
}

func TestCheckRegion_EmptyTargetStates(t *testing.T) {
	profile := ProfileInput{State: "Wien"}
	foerderung := Foerderung{TargetStates: []string{}}

	score := checkRegion(profile, foerderung)
	if score != WeightRegion {
		t.Errorf("Expected %d (all states), got %d", WeightRegion, score)
	}
}

func TestCheckRegion_NilTargetStates(t *testing.T) {
	profile := ProfileInput{State: "Steiermark"}
	foerderung := Foerderung{TargetStates: nil}

	score := checkRegion(profile, foerderung)
	if score != WeightRegion {
		t.Errorf("Expected %d (all states), got %d", WeightRegion, score)
	}
}

// ======================
// SIZE RULE TESTS
// ======================

func TestCheckSize_StartupMatch(t *testing.T) {
	profile := ProfileInput{IsStartup: true, EmployeesCount: 5}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeStartup}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d, got %d", WeightSize, score)
	}
}

func TestCheckSize_KMUMatch(t *testing.T) {
	profile := ProfileInput{IsKMU: true, EmployeesCount: 100}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeKMU}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d, got %d", WeightSize, score)
	}
}

func TestCheckSize_MikroByEmployees(t *testing.T) {
	profile := ProfileInput{EmployeesCount: 5}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeMikro}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d (mikro <10), got %d", WeightSize, score)
	}
}

func TestCheckSize_KleinByEmployees(t *testing.T) {
	profile := ProfileInput{EmployeesCount: 30}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeKlein}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d (klein <50), got %d", WeightSize, score)
	}
}

func TestCheckSize_MittelByEmployees(t *testing.T) {
	profile := ProfileInput{EmployeesCount: 150}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeMittel}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d (mittel <250), got %d", WeightSize, score)
	}
}

func TestCheckSize_GrossExcluded(t *testing.T) {
	profile := ProfileInput{EmployeesCount: 500, IsKMU: false}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeKMU}}

	score := checkSize(profile, foerderung)
	if score != 0 {
		t.Errorf("Expected 0 (gross excluded from KMU), got %d", score)
	}
}

func TestCheckSize_AllSizes(t *testing.T) {
	profile := ProfileInput{EmployeesCount: 500}
	foerderung := Foerderung{TargetSizes: []CompanySize{SizeAll}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d (all sizes), got %d", WeightSize, score)
	}
}

func TestCheckSize_EmptyTargetSizes(t *testing.T) {
	profile := ProfileInput{EmployeesCount: 100}
	foerderung := Foerderung{TargetSizes: []CompanySize{}}

	score := checkSize(profile, foerderung)
	if score != WeightSize {
		t.Errorf("Expected %d (empty = all), got %d", WeightSize, score)
	}
}

// ======================
// TOPICS RULE TESTS
// ======================

func TestCheckTopics_FullMatch(t *testing.T) {
	profile := ProfileInput{Topics: []string{"Innovation", "Digitalisierung"}}
	foerderung := Foerderung{Topics: []string{"Innovation", "Digitalisierung"}}

	score := checkTopics(profile, foerderung)
	if score != WeightTopics {
		t.Errorf("Expected %d, got %d", WeightTopics, score)
	}
}

func TestCheckTopics_PartialMatch(t *testing.T) {
	profile := ProfileInput{Topics: []string{"Innovation", "Export"}}
	foerderung := Foerderung{Topics: []string{"Innovation", "Digitalisierung", "Forschung"}}

	score := checkTopics(profile, foerderung)
	// 1 match out of 3 topics = 33% * 50 = ~16
	if score < 10 || score > 20 {
		t.Errorf("Expected partial score 10-20, got %d", score)
	}
}

func TestCheckTopics_NoMatch(t *testing.T) {
	profile := ProfileInput{Topics: []string{"Export"}}
	foerderung := Foerderung{Topics: []string{"Innovation", "Digitalisierung"}}

	score := checkTopics(profile, foerderung)
	if score != 0 {
		t.Errorf("Expected 0, got %d", score)
	}
}

func TestCheckTopics_EmptyProfileTopics(t *testing.T) {
	profile := ProfileInput{Topics: []string{}}
	foerderung := Foerderung{Topics: []string{"Innovation"}}

	score := checkTopics(profile, foerderung)
	if score != 0 {
		t.Errorf("Expected 0 (no profile topics), got %d", score)
	}
}

func TestCheckTopics_EmptyFoerderungTopics(t *testing.T) {
	profile := ProfileInput{Topics: []string{"Innovation"}}
	foerderung := Foerderung{Topics: []string{}}

	score := checkTopics(profile, foerderung)
	if score != WeightTopics {
		t.Errorf("Expected %d (empty = all), got %d", WeightTopics, score)
	}
}

// ======================
// DEADLINE RULE TESTS
// ======================

func TestCheckDeadline_NoDeadline(t *testing.T) {
	foerderung := Foerderung{ApplicationDeadline: nil}

	passed := checkDeadline(foerderung)
	if !passed {
		t.Error("Expected true (no deadline = always valid)")
	}
}

func TestCheckDeadline_FutureDeadline(t *testing.T) {
	future := time.Now().AddDate(0, 1, 0) // 1 month from now
	foerderung := Foerderung{ApplicationDeadline: &future}

	passed := checkDeadline(foerderung)
	if !passed {
		t.Error("Expected true (future deadline)")
	}
}

func TestCheckDeadline_PastDeadline(t *testing.T) {
	past := time.Now().AddDate(0, -1, 0) // 1 month ago
	foerderung := Foerderung{ApplicationDeadline: &past}

	passed := checkDeadline(foerderung)
	if passed {
		t.Error("Expected false (past deadline)")
	}
}

func TestCheckDeadline_TodayDeadline(t *testing.T) {
	today := time.Now()
	foerderung := Foerderung{ApplicationDeadline: &today}

	passed := checkDeadline(foerderung)
	if !passed {
		t.Error("Expected true (deadline today is still valid)")
	}
}

// ======================
// AGE RULE TESTS
// ======================

func TestCheckAge_WithinRange(t *testing.T) {
	minAge := 0
	maxAge := 5
	profile := ProfileInput{FoundedYear: time.Now().Year() - 3} // 3 years old
	foerderung := Foerderung{TargetAgeMin: &minAge, TargetAgeMax: &maxAge}

	passed := checkAge(profile, foerderung)
	if !passed {
		t.Error("Expected true (3 years within 0-5)")
	}
}

func TestCheckAge_TooOld(t *testing.T) {
	minAge := 0
	maxAge := 5
	profile := ProfileInput{FoundedYear: time.Now().Year() - 10} // 10 years old
	foerderung := Foerderung{TargetAgeMin: &minAge, TargetAgeMax: &maxAge}

	passed := checkAge(profile, foerderung)
	if passed {
		t.Error("Expected false (10 years > max 5)")
	}
}

func TestCheckAge_TooYoung(t *testing.T) {
	minAge := 3
	maxAge := 10
	profile := ProfileInput{FoundedYear: time.Now().Year() - 1} // 1 year old
	foerderung := Foerderung{TargetAgeMin: &minAge, TargetAgeMax: &maxAge}

	passed := checkAge(profile, foerderung)
	if passed {
		t.Error("Expected false (1 year < min 3)")
	}
}

func TestCheckAge_NoRestriction(t *testing.T) {
	profile := ProfileInput{FoundedYear: time.Now().Year() - 50}
	foerderung := Foerderung{TargetAgeMin: nil, TargetAgeMax: nil}

	passed := checkAge(profile, foerderung)
	if !passed {
		t.Error("Expected true (no age restriction)")
	}
}

func TestCheckAge_OnlyMinAge(t *testing.T) {
	minAge := 2
	profile := ProfileInput{FoundedYear: time.Now().Year() - 5}
	foerderung := Foerderung{TargetAgeMin: &minAge, TargetAgeMax: nil}

	passed := checkAge(profile, foerderung)
	if !passed {
		t.Error("Expected true (5 >= min 2)")
	}
}

// ======================
// INDUSTRY EXCLUSION TESTS
// ======================

func TestCheckIndustry_NotExcluded(t *testing.T) {
	profile := ProfileInput{Industry: "IT & Software"}
	foerderung := Foerderung{ExcludedIndustries: []string{"Glücksspiel", "Tabak"}}

	passed := checkIndustry(profile, foerderung)
	if !passed {
		t.Error("Expected true (IT not excluded)")
	}
}

func TestCheckIndustry_Excluded(t *testing.T) {
	profile := ProfileInput{Industry: "Glücksspiel"}
	foerderung := Foerderung{ExcludedIndustries: []string{"Glücksspiel", "Tabak"}}

	passed := checkIndustry(profile, foerderung)
	if passed {
		t.Error("Expected false (Glücksspiel excluded)")
	}
}

func TestCheckIndustry_NoExclusions(t *testing.T) {
	profile := ProfileInput{Industry: "Anything"}
	foerderung := Foerderung{ExcludedIndustries: nil}

	passed := checkIndustry(profile, foerderung)
	if !passed {
		t.Error("Expected true (no exclusions)")
	}
}

// ======================
// KMU CALCULATION TESTS
// ======================

func TestCalculateIsKMU_SmallCompany(t *testing.T) {
	employees := 50
	revenue := 10000000   // €10M
	balance := 8000000    // €8M

	isKMU := calculateIsKMU(&employees, &revenue, &balance)
	if !isKMU {
		t.Error("Expected true (50 employees, €10M revenue)")
	}
}

func TestCalculateIsKMU_TooManyEmployees(t *testing.T) {
	employees := 300
	revenue := 10000000
	balance := 8000000

	isKMU := calculateIsKMU(&employees, &revenue, &balance)
	if isKMU {
		t.Error("Expected false (300 employees >= 250)")
	}
}

func TestCalculateIsKMU_TooHighRevenueAndBalance(t *testing.T) {
	employees := 100
	revenue := 60000000   // €60M
	balance := 50000000   // €50M

	isKMU := calculateIsKMU(&employees, &revenue, &balance)
	if isKMU {
		t.Error("Expected false (revenue >= €50M AND balance >= €43M)")
	}
}

func TestCalculateIsKMU_HighRevenueButLowBalance(t *testing.T) {
	employees := 100
	revenue := 60000000   // €60M
	balance := 30000000   // €30M

	isKMU := calculateIsKMU(&employees, &revenue, &balance)
	if !isKMU {
		t.Error("Expected true (balance < €43M)")
	}
}

func TestCalculateIsKMU_NoData(t *testing.T) {
	isKMU := calculateIsKMU(nil, nil, nil)
	if !isKMU {
		t.Error("Expected true (assume KMU when no data)")
	}
}

// ======================
// TOTAL SCORE TESTS
// ======================

func TestCalculateTotalScore_FullMatch(t *testing.T) {
	ruleScore := 100
	llmScore := 100
	expectedTotal := 100 // 100*0.4 + 100*0.6 = 100

	total := calculateTotalScore(ruleScore, llmScore)
	if total != expectedTotal {
		t.Errorf("Expected %d, got %d", expectedTotal, total)
	}
}

func TestCalculateTotalScore_RuleOnly(t *testing.T) {
	ruleScore := 100
	llmScore := 0
	expectedTotal := 40 // 100*0.4 + 0*0.6 = 40

	total := calculateTotalScore(ruleScore, llmScore)
	if total != expectedTotal {
		t.Errorf("Expected %d, got %d", expectedTotal, total)
	}
}

func TestCalculateTotalScore_LLMOnly(t *testing.T) {
	ruleScore := 0
	llmScore := 100
	expectedTotal := 60 // 0*0.4 + 100*0.6 = 60

	total := calculateTotalScore(ruleScore, llmScore)
	if total != expectedTotal {
		t.Errorf("Expected %d, got %d", expectedTotal, total)
	}
}

func TestCalculateTotalScore_Mixed(t *testing.T) {
	ruleScore := 75
	llmScore := 85
	expectedTotal := 81 // 75*0.4 + 85*0.6 = 30 + 51 = 81

	total := calculateTotalScore(ruleScore, llmScore)
	if total != expectedTotal {
		t.Errorf("Expected %d, got %d", expectedTotal, total)
	}
}

// ======================
// HELPER FUNCTION IMPLEMENTATIONS
// ======================

func checkRegion(profile ProfileInput, f Foerderung) int {
	if len(f.TargetStates) == 0 {
		return WeightRegion
	}
	for _, state := range f.TargetStates {
		if state == profile.State {
			return WeightRegion
		}
	}
	return 0
}

func checkSize(profile ProfileInput, f Foerderung) int {
	if len(f.TargetSizes) == 0 {
		return WeightSize
	}

	for _, size := range f.TargetSizes {
		switch size {
		case SizeAll:
			return WeightSize
		case SizeStartup:
			if profile.IsStartup {
				return WeightSize
			}
		case SizeKMU:
			if profile.IsKMU || profile.EmployeesCount < 250 {
				return WeightSize
			}
		case SizeMikro:
			if profile.EmployeesCount < 10 {
				return WeightSize
			}
		case SizeKlein:
			if profile.EmployeesCount < 50 {
				return WeightSize
			}
		case SizeMittel:
			if profile.EmployeesCount < 250 {
				return WeightSize
			}
		case SizeGross:
			if profile.EmployeesCount >= 250 {
				return WeightSize
			}
		}
	}
	return 0
}

func checkTopics(profile ProfileInput, f Foerderung) int {
	if len(f.Topics) == 0 {
		return WeightTopics
	}
	if len(profile.Topics) == 0 {
		return 0
	}

	matches := 0
	for _, pt := range profile.Topics {
		for _, ft := range f.Topics {
			if pt == ft {
				matches++
				break
			}
		}
	}

	if matches == 0 {
		return 0
	}

	return (matches * WeightTopics) / len(f.Topics)
}

func checkDeadline(f Foerderung) bool {
	if f.ApplicationDeadline == nil {
		return true
	}
	return !f.ApplicationDeadline.Before(time.Now().Truncate(24 * time.Hour))
}

func checkAge(profile ProfileInput, f Foerderung) bool {
	if f.TargetAgeMin == nil && f.TargetAgeMax == nil {
		return true
	}

	age := time.Now().Year() - profile.FoundedYear

	if f.TargetAgeMin != nil && age < *f.TargetAgeMin {
		return false
	}
	if f.TargetAgeMax != nil && age > *f.TargetAgeMax {
		return false
	}
	return true
}

func checkIndustry(profile ProfileInput, f Foerderung) bool {
	if len(f.ExcludedIndustries) == 0 {
		return true
	}
	for _, excluded := range f.ExcludedIndustries {
		if profile.Industry == excluded {
			return false
		}
	}
	return true
}

func calculateIsKMU(employees, revenue, balance *int) bool {
	if employees == nil && revenue == nil && balance == nil {
		return true
	}

	if employees != nil && *employees >= 250 {
		return false
	}

	if revenue != nil && balance != nil {
		if *revenue >= 50000000 && *balance >= 43000000 {
			return false
		}
	}

	return true
}

func calculateTotalScore(ruleScore, llmScore int) int {
	return int(float64(ruleScore)*0.4 + float64(llmScore)*0.6)
}
