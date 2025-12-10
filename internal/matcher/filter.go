package matcher

import (
	"sort"
	"strings"
	"time"

	"austrian-business-infrastructure/internal/foerderung"
)

// Filter applies rule-based filtering to Förderungen
type Filter struct{}

// NewFilter creates a new Filter
func NewFilter() *Filter {
	return &Filter{}
}

// FilterAll applies all rules to all Förderungen
func (f *Filter) FilterAll(profile *ProfileInput, foerderungen []*foerderung.Foerderung) []*FilterResult {
	results := make([]*FilterResult, 0, len(foerderungen))

	for _, fd := range foerderungen {
		result := f.FilterOne(profile, fd)
		results = append(results, result)
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})

	return results
}

// FilterOne applies all rules to a single Förderung
func (f *Filter) FilterOne(profile *ProfileInput, fd *foerderung.Foerderung) *FilterResult {
	result := &FilterResult{
		FoerderungID:   fd.ID.String(),
		FoerderungName: fd.Name,
		Provider:       fd.Provider,
		RuleResults:    make([]RuleResult, 0, 5),
	}

	// Apply each rule
	result.RuleResults = append(result.RuleResults, f.checkRegion(profile, fd))
	result.RuleResults = append(result.RuleResults, f.checkSize(profile, fd))
	result.RuleResults = append(result.RuleResults, f.checkTopics(profile, fd))
	result.RuleResults = append(result.RuleResults, f.checkDeadline(fd))
	result.RuleResults = append(result.RuleResults, f.checkType(profile, fd))

	// Calculate total score (weighted average)
	// Only rules with weight > 0 contribute to score (Themen, Größe, Standort)
	// Deadline and Type are hard filters (weight 0)
	totalWeight := 0.0
	totalScore := 0.0
	allPassed := true

	for _, rr := range result.RuleResults {
		// Hard filters: if weight is 0 but score is 0 and passed is false, it's a hard fail
		if rr.Weight == 0 {
			if !rr.Passed {
				allPassed = false
			}
			continue
		}

		// Scored rules
		totalWeight += rr.Weight
		totalScore += rr.Score * rr.Weight

		// A scored rule with score 0 is a hard fail
		if rr.Score == 0 {
			allPassed = false
		}
	}

	if totalWeight > 0 {
		result.TotalScore = totalScore / totalWeight
	}
	result.Passed = allPassed && result.TotalScore >= MinScoreForLLM

	return result
}

// GetCandidates returns the top candidates for LLM analysis
func (f *Filter) GetCandidates(profile *ProfileInput, foerderungen []*foerderung.Foerderung) []*MatchCandidate {
	results := f.FilterAll(profile, foerderungen)

	candidates := make([]*MatchCandidate, 0)
	foerderungMap := make(map[string]*foerderung.Foerderung)
	for _, fd := range foerderungen {
		foerderungMap[fd.ID.String()] = fd
	}

	for _, result := range results {
		if result.Passed && len(candidates) < MaxLLMCandidates {
			candidates = append(candidates, &MatchCandidate{
				Foerderung:   foerderungMap[result.FoerderungID],
				FilterResult: result,
			})
		}
	}

	return candidates
}

// ============================================
// RULE IMPLEMENTATIONS
// ============================================

// checkRegion checks if the Förderung is available in the company's region
func (f *Filter) checkRegion(profile *ProfileInput, fd *foerderung.Foerderung) RuleResult {
	result := RuleResult{
		RuleName: "region",
		Weight:   WeightRegion,
		Reasons:  []string{},
	}

	// No state specified - assume nationwide
	if profile.State == "" {
		result.Passed = true
		result.Score = 0.8 // Slight penalty for unknown region
		result.Confidence = ConfidenceMedium
		result.Reasons = append(result.Reasons, "Bundesland nicht angegeben, Förderung angenommen")
		return result
	}

	// Check if Förderung is available in the state
	if len(fd.TargetStates) == 0 {
		result.Passed = true
		result.Score = 1.0
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Förderung österreichweit verfügbar")
		return result
	}

	// Check for "alle" (all states)
	for _, state := range fd.TargetStates {
		if strings.ToLower(state) == "alle" {
			result.Passed = true
			result.Score = 1.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Förderung österreichweit verfügbar")
			return result
		}
	}

	// Check specific states
	profileState := strings.ToLower(profile.State)
	for _, state := range fd.TargetStates {
		if strings.ToLower(state) == profileState ||
			strings.Contains(strings.ToLower(state), profileState) ||
			strings.Contains(profileState, strings.ToLower(state)) {
			result.Passed = true
			result.Score = 1.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Förderung in "+profile.State+" verfügbar")
			return result
		}
	}

	// Not available in region
	result.Passed = false
	result.Score = 0.0
	result.Confidence = ConfidenceHigh
	result.Reasons = append(result.Reasons, "Förderung nur in: "+strings.Join(fd.TargetStates, ", "))
	return result
}

// checkSize checks if the company size matches the target (TypeScript: groesse.ts)
func (f *Filter) checkSize(profile *ProfileInput, fd *foerderung.Foerderung) RuleResult {
	result := RuleResult{
		RuleName: "size",
		Weight:   WeightSize,
		Reasons:  []string{},
	}

	// Get company's granular size
	companySize := profile.DetermineCompanySize()

	// Check against granular TargetSizes array (TypeScript style)
	if len(fd.TargetSizes) > 0 {
		for _, targetSize := range fd.TargetSizes {
			if targetSize == companySize {
				result.Passed = true
				result.Score = 1.0
				result.Confidence = ConfidenceHigh
				result.Reasons = append(result.Reasons, "Unternehmensgröße ("+string(companySize)+") passt")
				return result
			}
		}
		// No match
		result.Passed = false
		result.Score = 0.0
		result.Confidence = ConfidenceHigh
		allowedSizes := make([]string, len(fd.TargetSizes))
		for i, s := range fd.TargetSizes {
			allowedSizes[i] = string(s)
		}
		result.Reasons = append(result.Reasons, "Nur für: "+strings.Join(allowedSizes, ", "))
		return result
	}

	// Fallback to legacy TargetSize (single value)
	if fd.TargetSize == nil {
		result.Passed = true
		result.Score = 1.0
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Keine Größenbeschränkung")
		return result
	}

	targetSize := *fd.TargetSize
	isKMU := profile.DetermineIsKMU()
	companyAge := profile.DetermineCompanyAge(time.Now().Year())

	switch targetSize {
	case foerderung.TargetSizeAlle:
		result.Passed = true
		result.Score = 1.0
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Offen für alle Unternehmensgrößen")

	case foerderung.TargetSizeKMU:
		if isKMU {
			result.Passed = true
			result.Score = 1.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Unternehmen erfüllt KMU-Kriterien")
		} else {
			result.Passed = false
			result.Score = 0.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Nur für KMU (< 250 MA, < €50M Umsatz)")
		}

	case foerderung.TargetSizeStartup:
		if profile.IsStartup || companyAge == "gruendung" {
			result.Passed = true
			result.Score = 1.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Unternehmen qualifiziert als Startup")
		} else {
			result.Passed = false
			result.Score = 0.2 // Partial score - might still apply
			result.Confidence = ConfidenceMedium
			result.Reasons = append(result.Reasons, "Primär für Startups (Gründung < 5 Jahre)")
		}

	case foerderung.TargetSizeGrossunternehmen:
		if !isKMU {
			result.Passed = true
			result.Score = 1.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Für Großunternehmen geeignet")
		} else {
			result.Passed = false
			result.Score = 0.0
			result.Confidence = ConfidenceHigh
			result.Reasons = append(result.Reasons, "Nur für Großunternehmen")
		}

	default:
		result.Passed = true
		result.Score = 0.7
		result.Confidence = ConfidenceLow
		result.Reasons = append(result.Reasons, "Unbekannte Zielgröße: "+string(targetSize))
	}

	return result
}

// checkTopics checks topic overlap between profile and Förderung
func (f *Filter) checkTopics(profile *ProfileInput, fd *foerderung.Foerderung) RuleResult {
	result := RuleResult{
		RuleName: "topics",
		Weight:   WeightTopics,
		Reasons:  []string{},
	}

	if len(fd.Topics) == 0 {
		result.Passed = true
		result.Score = 0.5 // Neutral score
		result.Confidence = ConfidenceLow
		result.Reasons = append(result.Reasons, "Keine Themen für Förderung definiert")
		return result
	}

	if len(profile.ProjectTopics) == 0 {
		result.Passed = true
		result.Score = 0.5 // Neutral score
		result.Confidence = ConfidenceLow
		result.Reasons = append(result.Reasons, "Keine Projekt-Themen angegeben")
		return result
	}

	// Count matching topics
	matchCount := 0
	matchedTopics := []string{}

	profileTopicsLower := make(map[string]bool)
	for _, t := range profile.ProjectTopics {
		profileTopicsLower[strings.ToLower(t)] = true
	}

	for _, fdTopic := range fd.Topics {
		fdTopicLower := strings.ToLower(fdTopic)
		// Direct match or partial match
		for profileTopic := range profileTopicsLower {
			if fdTopicLower == profileTopic ||
				strings.Contains(fdTopicLower, profileTopic) ||
				strings.Contains(profileTopic, fdTopicLower) {
				matchCount++
				matchedTopics = append(matchedTopics, fdTopic)
				break
			}
		}
	}

	// Calculate score based on overlap
	if matchCount == 0 {
		result.Passed = false
		result.Score = 0.1 // Very low score for no matches
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Keine Themenübereinstimmung")
	} else {
		// Score based on match ratio
		matchRatio := float64(matchCount) / float64(len(fd.Topics))
		result.Score = 0.3 + (0.7 * matchRatio) // 30% base + up to 70% for matches
		result.Passed = result.Score >= 0.5

		if matchRatio >= 0.5 {
			result.Confidence = ConfidenceHigh
		} else {
			result.Confidence = ConfidenceMedium
		}

		result.Reasons = append(result.Reasons, "Übereinstimmende Themen: "+strings.Join(matchedTopics, ", "))
	}

	return result
}

// checkDeadline checks if the Förderung deadline is still valid
func (f *Filter) checkDeadline(fd *foerderung.Foerderung) RuleResult {
	result := RuleResult{
		RuleName: "deadline",
		Weight:   WeightDeadline,
		Reasons:  []string{},
	}

	// No deadline - rolling application
	if fd.ApplicationDeadline == nil {
		result.Passed = true
		result.Score = 1.0
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Laufende Einreichung möglich")
		return result
	}

	now := time.Now()
	deadline := *fd.ApplicationDeadline

	if deadline.Before(now) {
		result.Passed = false
		result.Score = 0.0
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Einreichfrist abgelaufen: "+deadline.Format("02.01.2006"))
		return result
	}

	// Check how much time is left
	daysLeft := int(deadline.Sub(now).Hours() / 24)

	if daysLeft <= 7 {
		result.Passed = true
		result.Score = 0.6 // Lower score for tight deadline
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Frist in "+string(rune(daysLeft))+" Tagen - Eile geboten!")
	} else if daysLeft <= 30 {
		result.Passed = true
		result.Score = 0.8
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Einreichfrist: "+deadline.Format("02.01.2006"))
	} else {
		result.Passed = true
		result.Score = 1.0
		result.Confidence = ConfidenceHigh
		result.Reasons = append(result.Reasons, "Ausreichend Zeit bis "+deadline.Format("02.01.2006"))
	}

	return result
}

// checkType checks if the funding type matches the company needs
func (f *Filter) checkType(profile *ProfileInput, fd *foerderung.Foerderung) RuleResult {
	result := RuleResult{
		RuleName: "type",
		Weight:   WeightType,
		Reasons:  []string{},
	}

	// All types are generally acceptable
	result.Passed = true
	result.Confidence = ConfidenceHigh

	switch fd.Type {
	case foerderung.TypeZuschuss:
		result.Score = 1.0
		result.Reasons = append(result.Reasons, "Zuschuss - nicht rückzahlbar")

	case foerderung.TypeGarantie:
		result.Score = 0.9
		result.Reasons = append(result.Reasons, "Garantie - erleichtert Kreditaufnahme")

	case foerderung.TypeKredit:
		result.Score = 0.8
		result.Reasons = append(result.Reasons, "Kredit - zinsgünstig, rückzahlbar")

	case foerderung.TypeBeratung:
		result.Score = 0.7
		result.Reasons = append(result.Reasons, "Beratungsförderung")

	case foerderung.TypeKombination:
		result.Score = 0.95
		result.Reasons = append(result.Reasons, "Kombinierte Förderung (Zuschuss + Kredit/Garantie)")

	default:
		result.Score = 0.6
		result.Reasons = append(result.Reasons, "Unbekannter Förderungstyp")
	}

	return result
}
