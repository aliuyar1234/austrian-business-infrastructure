package export

import (
	"fmt"
	"strings"
	"time"

	"austrian-business-infrastructure/internal/foerderung"
)

// GenerateMarkdown generates a Markdown report of search results
func GenerateMarkdown(search *foerderung.FoerderungsSuche, matches []foerderung.FoerderungsMatch) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Förderungsradar - Suchergebnis\n\n")
	sb.WriteString(fmt.Sprintf("**Suche-ID:** %s  \n", search.ID.String()[:8]))
	sb.WriteString(fmt.Sprintf("**Datum:** %s  \n", search.CreatedAt.Format("02.01.2006 15:04")))
	sb.WriteString(fmt.Sprintf("**Status:** %s  \n\n", translateStatus(search.Status)))

	// Summary
	sb.WriteString("## Zusammenfassung\n\n")
	sb.WriteString(fmt.Sprintf("- **Geprüfte Förderungen:** %d\n", search.TotalFoerderungen))
	sb.WriteString(fmt.Sprintf("- **Passende Förderungen:** %d\n", search.TotalMatches))
	if search.LLMTokensUsed > 0 {
		sb.WriteString(fmt.Sprintf("- **KI-Analyse:** Ja (Kosten: €%.2f)\n", float64(search.LLMCostCents)/100))
	} else {
		sb.WriteString("- **KI-Analyse:** Nein (Regel-basiert)\n")
	}
	sb.WriteString("\n")

	// Matches
	if len(matches) > 0 {
		sb.WriteString("## Passende Förderungen\n\n")

		for i, match := range matches {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, match.FoerderungName))
			sb.WriteString(fmt.Sprintf("**Fördergeber:** %s  \n", match.Provider))
			sb.WriteString(fmt.Sprintf("**Gesamtbewertung:** %.0f%%  \n", match.TotalScore*100))
			sb.WriteString(fmt.Sprintf("- Regel-Score: %.0f%%\n", match.RuleScore*100))
			sb.WriteString(fmt.Sprintf("- KI-Score: %.0f%%\n", match.LLMScore*100))
			sb.WriteString("\n")

			// LLM insights
			if match.LLMResult != nil {
				llm := match.LLMResult

				// Eligibility
				eligibility := "Nicht geeignet"
				if llm.Eligible {
					eligibility = "Geeignet"
				}
				sb.WriteString(fmt.Sprintf("**Eignung:** %s (Konfidenz: %s)  \n\n", eligibility, translateConfidence(llm.Confidence)))

				// Matched criteria
				if len(llm.MatchedCriteria) > 0 {
					sb.WriteString("**Erfüllte Kriterien:**\n")
					for _, c := range llm.MatchedCriteria {
						sb.WriteString(fmt.Sprintf("- %s\n", c))
					}
					sb.WriteString("\n")
				}

				// Implicit matches
				if len(llm.ImplicitMatches) > 0 {
					sb.WriteString("**Zusätzlich erkannt:**\n")
					for _, m := range llm.ImplicitMatches {
						sb.WriteString(fmt.Sprintf("- %s\n", m))
					}
					sb.WriteString("\n")
				}

				// Concerns
				if len(llm.Concerns) > 0 {
					sb.WriteString("**Bedenken:**\n")
					for _, c := range llm.Concerns {
						sb.WriteString(fmt.Sprintf("- ⚠️ %s\n", c))
					}
					sb.WriteString("\n")
				}

				// Estimated amount
				if llm.EstimatedAmount != nil {
					sb.WriteString(fmt.Sprintf("**Geschätzte Förderhöhe:** bis zu €%s  \n\n", formatAmount(*llm.EstimatedAmount)))
				}

				// Combination hint
				if llm.CombinationHint != nil && *llm.CombinationHint != "" {
					sb.WriteString(fmt.Sprintf("**💡 Kombinations-Tipp:** %s  \n\n", *llm.CombinationHint))
				}

				// Next steps
				if len(llm.NextSteps) > 0 {
					sb.WriteString("**Nächste Schritte:**\n")
					for i, step := range llm.NextSteps {
						sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
					}
					sb.WriteString("\n")
				}

				// Insider tip
				if llm.InsiderTip != nil && *llm.InsiderTip != "" {
					sb.WriteString(fmt.Sprintf("**🎯 Insider-Tipp:** %s  \n\n", *llm.InsiderTip))
				}
			}

			sb.WriteString("---\n\n")
		}
	} else {
		sb.WriteString("## Keine passenden Förderungen gefunden\n\n")
		sb.WriteString("Leider konnten keine Förderungen gefunden werden, die zu Ihrem Profil passen.\n\n")
		sb.WriteString("**Empfehlungen:**\n")
		sb.WriteString("- Überprüfen Sie Ihr Unternehmensprofil auf Vollständigkeit\n")
		sb.WriteString("- Erweitern Sie Ihre Projektthemen\n")
		sb.WriteString("- Kontaktieren Sie uns für eine persönliche Beratung\n\n")
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("*Generiert am %s durch Förderungsradar*\n", time.Now().Format("02.01.2006 15:04")))
	sb.WriteString("\n*Hinweis: Diese Analyse dient als Orientierung. Die endgültige Förderfähigkeit wird vom jeweiligen Fördergeber geprüft.*\n")

	return sb.String()
}

// Helper functions

func translateStatus(status string) string {
	switch status {
	case foerderung.SearchStatusPending:
		return "Ausstehend"
	case foerderung.SearchStatusRuleFiltering:
		return "Regel-Filterung"
	case foerderung.SearchStatusLLMAnalysis:
		return "KI-Analyse"
	case foerderung.SearchStatusCompleted:
		return "Abgeschlossen"
	case foerderung.SearchStatusFailed:
		return "Fehlgeschlagen"
	default:
		return status
	}
}

func translateConfidence(confidence string) string {
	switch confidence {
	case "high":
		return "Hoch"
	case "medium":
		return "Mittel"
	case "low":
		return "Niedrig"
	default:
		return confidence
	}
}

func formatAmount(amount int) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1f Mio", float64(amount)/1000000)
	}
	if amount >= 1000 {
		return fmt.Sprintf("%d.%03d", amount/1000, amount%1000)
	}
	return fmt.Sprintf("%d", amount)
}
