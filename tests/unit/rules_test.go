package unit

import (
	"testing"
	"time"
)

// T122: Unit tests for individual rule implementations

// ======================
// COMBINED SCORING TESTS
// ======================

func TestCombinedScoring_AllRulesPass(t *testing.T) {
	// Profile with good match
	employees := 50
	profile := ProfileInput{
		CompanyName:    "Test GmbH",
		State:          "Wien",
		Industry:       "IT & Software",
		EmployeesCount: employees,
		IsKMU:          true,
		Topics:         []string{"Digitalisierung", "Innovation"},
	}

	deadline := time.Now().AddDate(0, 3, 0)
	foerderung := Foerderung{
		Name:                "Digital Wien",
		TargetStates:        []string{"Wien"},
		TargetSizes:         []CompanySize{SizeKMU},
		Topics:              []string{"Digitalisierung", "Innovation"},
		ApplicationDeadline: &deadline,
	}

	// All rules should score high
	regionScore := checkRegion(profile, foerderung)
	sizeScore := checkSize(profile, foerderung)
	topicScore := checkTopics(profile, foerderung)

	totalPossible := WeightRegion + WeightSize + WeightTopics
	totalActual := regionScore + sizeScore + topicScore

	// Should score at least 80% when all match
	if float64(totalActual)/float64(totalPossible) < 0.8 {
		t.Errorf("Expected >= 80%% score, got %.2f%%", float64(totalActual)/float64(totalPossible)*100)
	}
}

func TestCombinedScoring_PartialMatch(t *testing.T) {
	employees := 50
	profile := ProfileInput{
		CompanyName:    "Test GmbH",
		State:          "Salzburg",
		EmployeesCount: employees,
		IsKMU:          true,
		Topics:         []string{"Export"}, // Doesn't match
	}

	foerderung := Foerderung{
		Name:         "Wien Digital",
		TargetStates: []string{"Wien"}, // Won't match
		TargetSizes:  []CompanySize{SizeKMU},
		Topics:       []string{"Digitalisierung"},
	}

	regionScore := checkRegion(profile, foerderung)
	sizeScore := checkSize(profile, foerderung)
	topicScore := checkTopics(profile, foerderung)

	// Region should fail, size pass, topics fail
	if regionScore != 0 {
		t.Errorf("Expected region score 0 for wrong state, got %d", regionScore)
	}

	if sizeScore != WeightSize {
		t.Errorf("Expected size score %d for KMU match, got %d", WeightSize, sizeScore)
	}

	if topicScore != 0 {
		t.Errorf("Expected topic score 0 for no match, got %d", topicScore)
	}
}

// ======================
// EDGE CASES
// ======================

func TestEdgeCase_EmptyProfile(t *testing.T) {
	profile := ProfileInput{}
	foerderung := Foerderung{
		Name:        "General Förderung",
		TargetSizes: []CompanySize{SizeAll},
	}

	// Should not panic and should return some scores
	regionScore := checkRegion(profile, foerderung)
	sizeScore := checkSize(profile, foerderung)

	// Empty state = all states accepted
	if regionScore != WeightRegion {
		t.Errorf("Empty state should match (nationwide), got score %d", regionScore)
	}

	// All sizes = match
	if sizeScore != WeightSize {
		t.Errorf("All sizes should match, got score %d", sizeScore)
	}
}

func TestEdgeCase_EmptyFoerderung(t *testing.T) {
	employees := 100
	profile := ProfileInput{
		State:          "Wien",
		EmployeesCount: employees,
		Topics:         []string{"Innovation"},
	}

	foerderung := Foerderung{
		Name: "Minimal Förderung",
		// No restrictions
	}

	regionScore := checkRegion(profile, foerderung)
	sizeScore := checkSize(profile, foerderung)
	topicScore := checkTopics(profile, foerderung)

	// All should be maximum since no restrictions
	if regionScore != WeightRegion {
		t.Errorf("No region restriction should give max score, got %d", regionScore)
	}
	if sizeScore != WeightSize {
		t.Errorf("No size restriction should give max score, got %d", sizeScore)
	}
	if topicScore != WeightTopics {
		t.Errorf("No topic restriction should give max score, got %d", topicScore)
	}
}

func TestEdgeCase_DeadlineToday(t *testing.T) {
	// Deadline exactly today - should still pass
	today := time.Now()
	foerderung := Foerderung{
		ApplicationDeadline: &today,
	}

	passed := checkDeadline(foerderung)
	if !passed {
		t.Error("Deadline today should still be valid")
	}
}

func TestEdgeCase_DeadlineYesterday(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	foerderung := Foerderung{
		ApplicationDeadline: &yesterday,
	}

	passed := checkDeadline(foerderung)
	if passed {
		t.Error("Deadline yesterday should be invalid")
	}
}

// ======================
// COMPANY SIZE CLASSIFICATION
// ======================

func TestCompanySizeClassification_Startup(t *testing.T) {
	tests := []struct {
		name       string
		isStartup  bool
		employees  int
		foundedYr  int
		matchSize  CompanySize
		shouldPass bool
	}{
		{"Explicit startup", true, 10, 2023, SizeStartup, true},
		{"Young company", false, 5, 2023, SizeStartup, false}, // Not marked as startup
		{"Old startup flag", true, 100, 2010, SizeStartup, true}, // Flag overrides age
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := ProfileInput{
				IsStartup:      tt.isStartup,
				EmployeesCount: tt.employees,
				FoundedYear:    tt.foundedYr,
			}

			foerderung := Foerderung{
				TargetSizes: []CompanySize{tt.matchSize},
			}

			score := checkSize(profile, foerderung)
			passed := score > 0

			if passed != tt.shouldPass {
				t.Errorf("Expected pass=%v for %s, got pass=%v (score=%d)",
					tt.shouldPass, tt.name, passed, score)
			}
		})
	}
}

func TestCompanySizeClassification_EmployeeThresholds(t *testing.T) {
	tests := []struct {
		name      string
		employees int
		matchSize CompanySize
		expected  bool
	}{
		{"Mikro 1", 1, SizeMikro, true},
		{"Mikro 9", 9, SizeMikro, true},
		{"Mikro 10 boundary", 10, SizeMikro, false}, // >= 10 is Klein
		{"Klein 10", 10, SizeKlein, true},
		{"Klein 49", 49, SizeKlein, true},
		{"Klein 50 boundary", 50, SizeKlein, false}, // >= 50 is Mittel
		{"Mittel 50", 50, SizeMittel, true},
		{"Mittel 249", 249, SizeMittel, true},
		{"Mittel 250 boundary", 250, SizeMittel, false}, // >= 250 is Gross
		{"Gross 250", 250, SizeGross, true},
		{"Gross 1000", 1000, SizeGross, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := ProfileInput{
				EmployeesCount: tt.employees,
			}

			foerderung := Foerderung{
				TargetSizes: []CompanySize{tt.matchSize},
			}

			score := checkSize(profile, foerderung)
			passed := score > 0

			if passed != tt.expected {
				t.Errorf("Expected %v for %d employees with target %s, got %v",
					tt.expected, tt.employees, tt.matchSize, passed)
			}
		})
	}
}

// ======================
// TOPIC MATCHING
// ======================

func TestTopicMatching_ExactMatch(t *testing.T) {
	profile := ProfileInput{
		Topics: []string{"Digitalisierung"},
	}

	foerderung := Foerderung{
		Topics: []string{"Digitalisierung"},
	}

	score := checkTopics(profile, foerderung)
	if score != WeightTopics {
		t.Errorf("Exact topic match should give max score %d, got %d", WeightTopics, score)
	}
}

func TestTopicMatching_MultipleMatches(t *testing.T) {
	profile := ProfileInput{
		Topics: []string{"Digitalisierung", "Innovation", "KI"},
	}

	foerderung := Foerderung{
		Topics: []string{"Digitalisierung", "Innovation", "Nachhaltigkeit"},
	}

	score := checkTopics(profile, foerderung)
	// 2/3 match = ~66% * WeightTopics
	expectedMin := int(float64(WeightTopics) * 0.5)
	if score < expectedMin {
		t.Errorf("Expected score >= %d for 2/3 match, got %d", expectedMin, score)
	}
}

func TestTopicMatching_NoOverlap(t *testing.T) {
	profile := ProfileInput{
		Topics: []string{"Export", "International"},
	}

	foerderung := Foerderung{
		Topics: []string{"Digitalisierung", "Innovation"},
	}

	score := checkTopics(profile, foerderung)
	if score != 0 {
		t.Errorf("No topic overlap should give 0, got %d", score)
	}
}

// ======================
// AGE RESTRICTIONS
// ======================

func TestAgeRestrictions(t *testing.T) {
	currentYear := time.Now().Year()

	tests := []struct {
		name      string
		founded   int
		minAge    *int
		maxAge    *int
		shouldPass bool
	}{
		{"No age restriction", 2010, nil, nil, true},
		{"Within range", currentYear - 3, intPtr(0), intPtr(5), true},
		{"Too old", currentYear - 10, intPtr(0), intPtr(5), false},
		{"Too young", currentYear - 1, intPtr(3), intPtr(10), false},
		{"At min boundary", currentYear - 3, intPtr(3), nil, true},
		{"At max boundary", currentYear - 5, nil, intPtr(5), true},
		{"Brand new", currentYear, intPtr(0), intPtr(3), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := ProfileInput{
				FoundedYear: tt.founded,
			}

			foerderung := Foerderung{
				TargetAgeMin: tt.minAge,
				TargetAgeMax: tt.maxAge,
			}

			passed := checkAge(profile, foerderung)
			if passed != tt.shouldPass {
				t.Errorf("Expected pass=%v for founded %d with min=%v max=%v, got %v",
					tt.shouldPass, tt.founded, tt.minAge, tt.maxAge, passed)
			}
		})
	}
}

// ======================
// KMU DEFINITION
// ======================

func TestKMUDefinition_EUCriteria(t *testing.T) {
	// EU KMU definition:
	// - < 250 employees AND
	// - (< €50M revenue OR < €43M balance sheet total)

	tests := []struct {
		name      string
		employees int
		revenue   int
		balance   int
		isKMU     bool
	}{
		{"Small KMU", 50, 5000000, 4000000, true},                      // All under limits
		{"Max employees KMU", 249, 40000000, 40000000, true},           // At employee limit
		{"Too many employees", 250, 10000000, 10000000, false},         // Over employee limit
		{"High revenue, low balance", 100, 60000000, 30000000, true},   // Revenue over, balance under
		{"Low revenue, high balance", 100, 30000000, 50000000, true},   // Revenue under, balance over
		{"Both limits exceeded", 100, 60000000, 50000000, false},       // Both over
		{"Large by all metrics", 300, 100000000, 100000000, false},     // All over limits
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateIsKMU(&tt.employees, &tt.revenue, &tt.balance)
			if result != tt.isKMU {
				t.Errorf("Expected isKMU=%v for emp=%d rev=%d bal=%d, got %v",
					tt.isKMU, tt.employees, tt.revenue, tt.balance, result)
			}
		})
	}
}

func TestKMUDefinition_PartialData(t *testing.T) {
	t.Run("Only employees known - KMU", func(t *testing.T) {
		employees := 100
		result := calculateIsKMU(&employees, nil, nil)
		if !result {
			t.Error("Expected KMU with only 100 employees known")
		}
	})

	t.Run("Only employees known - not KMU", func(t *testing.T) {
		employees := 300
		result := calculateIsKMU(&employees, nil, nil)
		if result {
			t.Error("Expected not KMU with 300 employees")
		}
	})

	t.Run("No data - assume KMU", func(t *testing.T) {
		result := calculateIsKMU(nil, nil, nil)
		if !result {
			t.Error("Expected KMU assumption when no data available")
		}
	})
}

// ======================
// HELPERS
// ======================

func intPtr(v int) *int {
	return &v
}
