package export

import (
	"bytes"
	"fmt"
	"time"

	"austrian-business-infrastructure/internal/foerderung"
)

// GeneratePDF generates a PDF report of search results
// This is a simple text-based PDF implementation
// For production, consider using a library like gofpdf or pdfcpu
func GeneratePDF(search *foerderung.FoerderungsSuche, matches []foerderung.FoerderungsMatch) ([]byte, error) {
	var buf bytes.Buffer

	// PDF Header
	buf.WriteString("%PDF-1.4\n")

	// Objects
	objects := make([]string, 0)

	// Catalog (object 1)
	objects = append(objects, "1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// Pages (object 2)
	objects = append(objects, "2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// Content stream
	content := generatePDFContent(search, matches)

	// Page (object 3)
	pageObj := fmt.Sprintf("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")
	objects = append(objects, pageObj)

	// Content stream (object 4)
	contentObj := fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(content), content)
	objects = append(objects, contentObj)

	// Font (object 5)
	fontObj := "5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>\nendobj\n"
	objects = append(objects, fontObj)

	// Write objects and track offsets
	offsets := make([]int, 0)
	currentOffset := buf.Len()

	for _, obj := range objects {
		offsets = append(offsets, currentOffset)
		buf.WriteString(obj)
		currentOffset = buf.Len()
	}

	// Cross-reference table
	xrefOffset := buf.Len()
	buf.WriteString("xref\n")
	buf.WriteString(fmt.Sprintf("0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}

	// Trailer
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	buf.WriteString("startxref\n")
	buf.WriteString(fmt.Sprintf("%d\n", xrefOffset))
	buf.WriteString("%%EOF\n")

	return buf.Bytes(), nil
}

// generatePDFContent generates the PDF content stream
func generatePDFContent(search *foerderung.FoerderungsSuche, matches []foerderung.FoerderungsMatch) string {
	var buf bytes.Buffer

	// Start text object
	buf.WriteString("BT\n")

	y := 800 // Start from top

	// Title
	buf.WriteString("/F1 18 Tf\n")
	buf.WriteString(fmt.Sprintf("50 %d Td\n", y))
	buf.WriteString("(Foerderungsradar - Suchergebnis) Tj\n")
	y -= 30

	// Metadata
	buf.WriteString("/F1 10 Tf\n")
	buf.WriteString(fmt.Sprintf("0 -%d Td\n", 30))
	buf.WriteString(fmt.Sprintf("(Datum: %s) Tj\n", search.CreatedAt.Format("02.01.2006 15:04")))
	y -= 15

	buf.WriteString(fmt.Sprintf("0 -%d Td\n", 15))
	buf.WriteString(fmt.Sprintf("(Status: %s) Tj\n", translateStatus(search.Status)))
	y -= 25

	// Summary
	buf.WriteString("/F1 14 Tf\n")
	buf.WriteString(fmt.Sprintf("0 -%d Td\n", 25))
	buf.WriteString("(Zusammenfassung) Tj\n")
	y -= 20

	buf.WriteString("/F1 10 Tf\n")
	buf.WriteString(fmt.Sprintf("0 -%d Td\n", 20))
	buf.WriteString(fmt.Sprintf("(Geprueft: %d Foerderungen) Tj\n", search.TotalFoerderungen))
	y -= 15

	buf.WriteString(fmt.Sprintf("0 -%d Td\n", 15))
	buf.WriteString(fmt.Sprintf("(Passend: %d Foerderungen) Tj\n", search.TotalMatches))
	y -= 30

	// Matches
	if len(matches) > 0 {
		buf.WriteString("/F1 14 Tf\n")
		buf.WriteString(fmt.Sprintf("0 -%d Td\n", 30))
		buf.WriteString("(Passende Foerderungen) Tj\n")

		for i, match := range matches {
			if i >= 10 { // Limit to first 10 on first page
				break
			}

			buf.WriteString("/F1 11 Tf\n")
			buf.WriteString(fmt.Sprintf("0 -%d Td\n", 25))
			buf.WriteString(fmt.Sprintf("(%d. %s) Tj\n", i+1, escapePDFString(match.FoerderungName)))

			buf.WriteString("/F1 9 Tf\n")
			buf.WriteString(fmt.Sprintf("0 -%d Td\n", 15))
			buf.WriteString(fmt.Sprintf("(   Foerdergeber: %s | Bewertung: %.0f%%) Tj\n",
				match.Provider, match.TotalScore*100))

			if match.LLMResult != nil && len(match.LLMResult.MatchedCriteria) > 0 {
				buf.WriteString(fmt.Sprintf("0 -%d Td\n", 12))
				criteria := match.LLMResult.MatchedCriteria[0]
				if len(criteria) > 60 {
					criteria = criteria[:60] + "..."
				}
				buf.WriteString(fmt.Sprintf("(   %s) Tj\n", escapePDFString(criteria)))
			}
		}
	} else {
		buf.WriteString("/F1 12 Tf\n")
		buf.WriteString(fmt.Sprintf("0 -%d Td\n", 30))
		buf.WriteString("(Keine passenden Foerderungen gefunden.) Tj\n")
	}

	// Footer
	buf.WriteString("/F1 8 Tf\n")
	buf.WriteString(fmt.Sprintf("0 -%d Td\n", 50))
	buf.WriteString(fmt.Sprintf("(Generiert am %s durch Foerderungsradar) Tj\n", time.Now().Format("02.01.2006")))

	// End text object
	buf.WriteString("ET\n")

	return buf.String()
}

// escapePDFString escapes special characters for PDF strings
func escapePDFString(s string) string {
	// Replace special PDF characters
	s = replaceAll(s, "\\", "\\\\")
	s = replaceAll(s, "(", "\\(")
	s = replaceAll(s, ")", "\\)")
	// Replace German umlauts with ASCII equivalents for basic PDF
	s = replaceAll(s, "ä", "ae")
	s = replaceAll(s, "ö", "oe")
	s = replaceAll(s, "ü", "ue")
	s = replaceAll(s, "Ä", "Ae")
	s = replaceAll(s, "Ö", "Oe")
	s = replaceAll(s, "Ü", "Ue")
	s = replaceAll(s, "ß", "ss")
	s = replaceAll(s, "€", "EUR")
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for _, r := range s {
		if string(r) == old {
			result += new
		} else {
			result += string(r)
		}
	}
	// Simple string replacement
	for i := 0; i < len(s); {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result = result[:len(result)-len(old)] + new
			i += len(old)
		} else {
			i++
		}
	}
	// Use standard approach
	result = ""
	remaining := s
	for {
		idx := indexOf(remaining, old)
		if idx == -1 {
			result += remaining
			break
		}
		result += remaining[:idx] + new
		remaining = remaining[idx+len(old):]
	}
	return result
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
