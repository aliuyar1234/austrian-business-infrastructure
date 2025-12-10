package matcher

import (
	"austrian-business-infrastructure/internal/foerderung"
)

// RuleResult represents the result of a single rule evaluation
type RuleResult struct {
	RuleName    string   `json:"rule_name"`
	Passed      bool     `json:"passed"`
	Score       float64  `json:"score"`        // 0.0 - 1.0
	Weight      float64  `json:"weight"`       // Weight in total score
	Reasons     []string `json:"reasons"`      // Why it matched/didn't match
	Confidence  string   `json:"confidence"`   // high, medium, low
}

// FilterResult represents the result of rule-based filtering
type FilterResult struct {
	FoerderungID   string        `json:"foerderung_id"`
	FoerderungName string        `json:"foerderung_name"`
	Provider       string        `json:"provider"`
	Passed         bool          `json:"passed"`
	TotalScore     float64       `json:"total_score"`    // Weighted average of all rules
	RuleResults    []RuleResult  `json:"rule_results"`
}

// MatchCandidate represents a candidate for LLM analysis
type MatchCandidate struct {
	Foerderung   *foerderung.Foerderung `json:"foerderung"`
	FilterResult *FilterResult          `json:"filter_result"`
}

// ProfileInput represents the profile data for matching
type ProfileInput struct {
	// Company basics
	CompanyName string  `json:"company_name"`
	LegalForm   string  `json:"legal_form,omitempty"`
	FoundedYear *int    `json:"founded_year,omitempty"`
	State       string  `json:"state,omitempty"` // Bundesland

	// Size
	EmployeesCount *int `json:"employees_count,omitempty"`
	AnnualRevenue  *int `json:"annual_revenue,omitempty"` // EUR
	BalanceTotal   *int `json:"balance_total,omitempty"`  // EUR

	// Classification
	Industry   string   `json:"industry,omitempty"`
	OnaceCodes []string `json:"onace_codes,omitempty"`
	IsStartup  bool     `json:"is_startup"`
	IsKMU      *bool    `json:"is_kmu,omitempty"`

	// Project
	ProjectDescription string   `json:"project_description,omitempty"`
	InvestmentAmount   *int     `json:"investment_amount,omitempty"` // EUR
	ProjectTopics      []string `json:"project_topics,omitempty"`
}

// DetermineIsKMU calculates if the company qualifies as KMU
func (p *ProfileInput) DetermineIsKMU() bool {
	if p.IsKMU != nil {
		return *p.IsKMU
	}

	// EU KMU Definition:
	// - Less than 250 employees
	// - Annual revenue < €50M OR Balance total < €43M
	if p.EmployeesCount != nil && *p.EmployeesCount >= 250 {
		return false
	}

	if p.AnnualRevenue != nil && *p.AnnualRevenue >= 50000000 {
		if p.BalanceTotal != nil && *p.BalanceTotal >= 43000000 {
			return false
		}
	}

	return true
}

// DetermineCompanyAge returns the company age category
func (p *ProfileInput) DetermineCompanyAge(currentYear int) string {
	if p.FoundedYear == nil {
		return "unknown"
	}

	age := currentYear - *p.FoundedYear
	if age <= 5 {
		return "gruendung"
	}
	return "etabliert"
}

// DetermineCompanySize returns the granular company size (TypeScript: berechneUnternehmensgroesse)
// Based on EU KMU definition
func (p *ProfileInput) DetermineCompanySize() foerderung.CompanySize {
	employees := 0
	if p.EmployeesCount != nil {
		employees = *p.EmployeesCount
	}

	revenue := 0
	if p.AnnualRevenue != nil {
		revenue = *p.AnnualRevenue
	}

	// EPU: 1 employee (Ein-Personen-Unternehmen)
	if employees <= 1 {
		return foerderung.SizeEPU
	}

	// kleinst: < 10 employees AND < €2M revenue
	if employees < 10 && revenue < 2000000 {
		return foerderung.SizeKleinst
	}

	// klein: < 50 employees AND < €10M revenue
	if employees < 50 && revenue < 10000000 {
		return foerderung.SizeKlein
	}

	// mittel: < 250 employees AND < €50M revenue
	if employees < 250 && revenue < 50000000 {
		return foerderung.SizeMittel
	}

	// gross: >= 250 employees OR >= €50M revenue
	return foerderung.SizeGross
}

// Rule weights for score calculation (matching TypeScript exactly)
// TypeScript uses: Themen 50%, Größe 25%, Standort 25%
const (
	WeightRegion   = 0.25 // 25% - Standort
	WeightSize     = 0.25 // 25% - Größe
	WeightTopics   = 0.50 // 50% - Themen (most important)
	WeightDeadline = 0.00 // Hard filter, not scored
	WeightType     = 0.00 // Hard filter, not scored
)

// Confidence levels
const (
	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// MinScoreForLLM is the minimum rule score to pass to LLM analysis
const MinScoreForLLM = 0.50 // 50%

// MaxLLMCandidates is the maximum number of candidates to analyze with LLM
const MaxLLMCandidates = 20
