package mcp

import (
	"errors"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
	"github.com/austrian-business-infrastructure/fo/internal/fb"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/austrian-business-infrastructure/fo/internal/sepa"
)

// handleUIDValidate validates an Austrian/EU VAT number
func handleUIDValidate(params map[string]interface{}) (interface{}, error) {
	uidStr, ok := params["uid"].(string)
	if !ok || uidStr == "" {
		return nil, errors.New("missing required parameter: uid")
	}

	result := fonws.ValidateUIDFormat(uidStr)
	return map[string]interface{}{
		"uid":           uidStr,
		"valid":         result.Valid,
		"country_code":  result.CountryCode,
		"error_message": result.Error,
	}, nil
}

// handleIBANValidate validates an IBAN
func handleIBANValidate(params map[string]interface{}) (interface{}, error) {
	ibanStr, ok := params["iban"].(string)
	if !ok || ibanStr == "" {
		return nil, errors.New("missing required parameter: iban")
	}

	result := sepa.ValidateIBANWithDetails(ibanStr)
	return map[string]interface{}{
		"iban":          result.IBAN,
		"valid":         result.Valid,
		"country_code":  result.CountryCode,
		"bank_code":     result.BankCode,
		"bic":           result.BIC,
		"bank_name":     result.BankName,
		"error_message": result.ErrorMessage,
	}, nil
}

// handleBICLookup looks up BIC for an Austrian bank code
func handleBICLookup(params map[string]interface{}) (interface{}, error) {
	bankCode, ok := params["bank_code"].(string)
	if !ok || bankCode == "" {
		return nil, errors.New("missing required parameter: bank_code")
	}

	bic, name := sepa.LookupAustrianBank(bankCode)
	if bic == "" {
		return map[string]interface{}{
			"bank_code": bankCode,
			"found":     false,
			"bic":       "",
			"bank_name": "",
		}, nil
	}

	return map[string]interface{}{
		"bank_code": bankCode,
		"found":     true,
		"bic":       bic,
		"bank_name": name,
	}, nil
}

// handleSVNummerValidate validates an Austrian social security number
func handleSVNummerValidate(params map[string]interface{}) (interface{}, error) {
	svNummer, ok := params["sv_nummer"].(string)
	if !ok || svNummer == "" {
		return nil, errors.New("missing required parameter: sv_nummer")
	}

	// Use the ELDA package for SV-Nummer validation
	err := elda.ValidateSVNummer(svNummer)
	if err != nil {
		return map[string]interface{}{
			"sv_nummer":     svNummer,
			"valid":         false,
			"error_message": err.Error(),
		}, nil
	}

	// Extract birth date using ELDA helper
	birthDate, _ := elda.ExtractBirthDateFromSVNummer(svNummer)

	// Format birth date components
	day := svNummer[4:6]
	month := svNummer[6:8]
	year := svNummer[8:10]

	return map[string]interface{}{
		"sv_nummer":      svNummer,
		"valid":          true,
		"serial_number":  svNummer[0:4],
		"birth_date":     day + "." + month + "." + year,
		"birth_date_iso": birthDate.Format("2006-01-02"),
	}, nil
}

// handleFNValidate validates an Austrian Firmenbuch number
func handleFNValidate(params map[string]interface{}) (interface{}, error) {
	fnStr, ok := params["fn"].(string)
	if !ok || fnStr == "" {
		return nil, errors.New("missing required parameter: fn")
	}

	err := fb.ValidateFN(fnStr)
	if err != nil {
		return map[string]interface{}{
			"fn":            fnStr,
			"valid":         false,
			"error_message": err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"fn":    fnStr,
		"valid": true,
	}, nil
}

// ===== MCP Tools Expansion (003-mcp-tools-expansion) =====

// authError returns a structured authentication error response
func authError(message string) map[string]interface{} {
	return map[string]interface{}{
		"error":      true,
		"error_type": "authentication",
		"message":    message,
	}
}

// validationError returns a structured validation error response
func validationError(message string) map[string]interface{} {
	return map[string]interface{}{
		"error":      true,
		"error_type": "validation",
		"message":    message,
	}
}

// serviceError returns a structured service error response
func serviceError(message string) map[string]interface{} {
	return map[string]interface{}{
		"error":      true,
		"error_type": "service",
		"message":    message,
	}
}

// handleDataboxList lists documents in a FinanzOnline databox
func handleDataboxList(params map[string]interface{}) (interface{}, error) {
	accountID, ok := params["account_id"].(string)
	if !ok || accountID == "" {
		return nil, errors.New("missing required parameter: account_id")
	}

	// MCP tools cannot access sessions directly (would require credential storage access)
	// Return auth error with guidance to use CLI
	return authError("No active session available. Use 'fo session login " + accountID + "' to authenticate, then use 'fo databox list " + accountID + "' to access the databox."), nil
}

// handleDataboxDownload downloads a document from a FinanzOnline databox
func handleDataboxDownload(params map[string]interface{}) (interface{}, error) {
	accountID, ok := params["account_id"].(string)
	if !ok || accountID == "" {
		return nil, errors.New("missing required parameter: account_id")
	}

	documentID, ok := params["document_id"].(string)
	if !ok || documentID == "" {
		return nil, errors.New("missing required parameter: document_id")
	}

	// MCP tools cannot access sessions directly
	return authError("No active session available. Use 'fo session login " + accountID + "' to authenticate, then use 'fo databox download " + accountID + " " + documentID + "' to download the document."), nil
}

// handleFBSearch searches for companies in the Austrian Firmenbuch
func handleFBSearch(params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, errors.New("missing required parameter: query")
	}

	// Extract optional parameters
	location, _ := params["location"].(string)
	maxResults := 20
	if max, ok := params["max_results"].(float64); ok {
		maxResults = int(max)
	}

	// Build search request
	searchReq := &fb.FBSearchRequest{
		Name:    query,
		Ort:     location,
		MaxHits: maxResults,
	}

	// Note: In a real scenario, we'd call the FB client here
	// For now, return a mock response structure since we don't have API keys configured
	// The structure matches what the real API would return
	_ = searchReq // Used to build the request

	return map[string]interface{}{
		"query":       query,
		"total_count": 0,
		"results":     []interface{}{},
		"message":     "Firmenbuch search completed. No API key configured for actual queries.",
	}, nil
}

// handleFBExtract retrieves full company details from the Austrian Firmenbuch
func handleFBExtract(params map[string]interface{}) (interface{}, error) {
	fnStr, ok := params["fn"].(string)
	if !ok || fnStr == "" {
		return nil, errors.New("missing required parameter: fn")
	}

	// Validate FN format first
	if err := fb.ValidateFN(fnStr); err != nil {
		return validationError("Invalid Firmenbuch number format. Expected format: FN followed by 1-9 digits and a lowercase letter (e.g., FN123456a)"), nil
	}

	// Note: In a real scenario, we'd call the FB client here
	// For now, return a structure indicating the FN was validated but no data available
	return map[string]interface{}{
		"fn":      fnStr,
		"message": "FN format validated. Full company data requires configured API access.",
	}, nil
}

// handleUVASubmit submits a UVA (VAT advance return) to FinanzOnline
func handleUVASubmit(params map[string]interface{}) (interface{}, error) {
	accountID, ok := params["account_id"].(string)
	if !ok || accountID == "" {
		return nil, errors.New("missing required parameter: account_id")
	}

	// Extract and validate year
	yearFloat, ok := params["year"].(float64)
	if !ok {
		return nil, errors.New("missing required parameter: year")
	}
	year := int(yearFloat)
	if year < 2000 || year > 2100 {
		return validationError("Invalid year. Must be between 2000 and 2100."), nil
	}

	// Extract and validate period type
	periodType, ok := params["period_type"].(string)
	if !ok || periodType == "" {
		return nil, errors.New("missing required parameter: period_type")
	}
	if periodType != "monthly" && periodType != "quarterly" {
		return validationError("Invalid period_type. Must be 'monthly' or 'quarterly'."), nil
	}

	// Extract and validate period value
	periodValueFloat, ok := params["period_value"].(float64)
	if !ok {
		return nil, errors.New("missing required parameter: period_value")
	}
	periodValue := int(periodValueFloat)

	if periodType == "monthly" && (periodValue < 1 || periodValue > 12) {
		return validationError("Invalid period_value for monthly period. Must be between 1 and 12."), nil
	}
	if periodType == "quarterly" && (periodValue < 1 || periodValue > 4) {
		return validationError("Invalid period_value for quarterly period. Must be between 1 and 4."), nil
	}

	// Extract kz_values (optional validation)
	_, ok = params["kz_values"].(map[string]interface{})
	if !ok {
		return nil, errors.New("missing required parameter: kz_values")
	}

	// MCP tools cannot access sessions directly
	return authError("No active session available. Use 'fo session login " + accountID + "' to authenticate, then use 'fo uva submit' to submit the UVA."), nil
}

// ===== AI Document Intelligence Tools (011-ai-document-intelligence) =====

// handleDocumentClassify classifies document text into categories
func handleDocumentClassify(params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, errors.New("missing required parameter: text")
	}

	title, _ := params["title"].(string)

	// Use heuristic classification for MCP (AI client requires config)
	result := classifyDocumentHeuristic(text, title)

	return map[string]interface{}{
		"document_type":    result.DocumentType,
		"document_subtype": result.DocumentSubtype,
		"confidence":       result.Confidence,
		"urgency":          result.Urgency,
		"requires_action":  result.RequiresAction,
		"keywords":         result.Keywords,
	}, nil
}

// handleDeadlineExtract extracts deadlines from document text
func handleDeadlineExtract(params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, errors.New("missing required parameter: text")
	}

	deadlines := extractDeadlinesHeuristic(text)

	return map[string]interface{}{
		"deadlines": deadlines,
		"count":     len(deadlines),
	}, nil
}

// handleAmountExtract extracts monetary amounts from document text
func handleAmountExtract(params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, errors.New("missing required parameter: text")
	}

	amounts := extractAmountsHeuristic(text)

	return map[string]interface{}{
		"amounts": amounts,
		"count":   len(amounts),
	}, nil
}

// handleDocumentSummarize generates a summary of document text
func handleDocumentSummarize(params map[string]interface{}) (interface{}, error) {
	text, ok := params["text"].(string)
	if !ok || text == "" {
		return nil, errors.New("missing required parameter: text")
	}

	// For MCP without AI, provide a basic summary
	summary := generateBasicSummary(text)

	return map[string]interface{}{
		"summary":       summary.Text,
		"word_count":    summary.WordCount,
		"key_sentences": summary.KeySentences,
	}, nil
}

// Helper structures and functions for heuristic analysis

type classificationResult struct {
	DocumentType    string   `json:"document_type"`
	DocumentSubtype string   `json:"document_subtype"`
	Confidence      float64  `json:"confidence"`
	Urgency         string   `json:"urgency"`
	RequiresAction  bool     `json:"requires_action"`
	Keywords        []string `json:"keywords"`
}

func classifyDocumentHeuristic(text, title string) classificationResult {
	result := classificationResult{
		DocumentType:    "sonstige",
		DocumentSubtype: "sonstige",
		Confidence:      0.5,
		Urgency:         "normal",
		RequiresAction:  false,
		Keywords:        []string{},
	}

	combined := text + " " + title
	lower := toLower(combined)

	// Check for document types by keywords
	if containsAnyMCP(lower, []string{"ergänzungsersuchen", "ersuchen um ergänzung", "werden sie ersucht"}) {
		result.DocumentType = "ersuchen"
		result.RequiresAction = true
		result.Urgency = "high"
		result.Confidence = 0.8
		result.Keywords = append(result.Keywords, "ersuchen")
	} else if containsAnyMCP(lower, []string{"bescheid", "abgabenbescheid", "einkommensteuerbescheid", "umsatzsteuerbescheid"}) {
		result.DocumentType = "bescheid"
		result.Confidence = 0.7
		result.Keywords = append(result.Keywords, "bescheid")
	} else if containsAnyMCP(lower, []string{"mahnung", "zahlungserinnerung", "säumniszuschlag"}) {
		result.DocumentType = "mahnung"
		result.RequiresAction = true
		result.Urgency = "high"
		result.Confidence = 0.8
		result.Keywords = append(result.Keywords, "mahnung")
	} else if containsAnyMCP(lower, []string{"vorhalt", "vorhaltsbeantwortung"}) {
		result.DocumentType = "vorhalt"
		result.RequiresAction = true
		result.Confidence = 0.7
		result.Keywords = append(result.Keywords, "vorhalt")
	} else if containsAnyMCP(lower, []string{"rechnung", "faktura", "invoice"}) {
		result.DocumentType = "rechnung"
		result.Confidence = 0.7
		result.Keywords = append(result.Keywords, "rechnung")
	} else if containsAnyMCP(lower, []string{"mitteilung", "information", "benachrichtigung"}) {
		result.DocumentType = "mitteilung"
		result.Confidence = 0.6
		result.Keywords = append(result.Keywords, "mitteilung")
	} else if containsAnyMCP(lower, []string{"zahlungsbefehl", "exekution"}) {
		result.DocumentType = "zahlungsbefehl"
		result.RequiresAction = true
		result.Urgency = "critical"
		result.Confidence = 0.9
		result.Keywords = append(result.Keywords, "zahlungsbefehl")
	}

	// Check for subtypes
	if containsAnyMCP(lower, []string{"einkommensteuer", "est"}) {
		result.DocumentSubtype = "einkommensteuer"
	} else if containsAnyMCP(lower, []string{"umsatzsteuer", "ust", "uva"}) {
		result.DocumentSubtype = "umsatzsteuer"
	} else if containsAnyMCP(lower, []string{"körperschaftsteuer", "köst"}) {
		result.DocumentSubtype = "körperschaftsteuer"
	} else if containsAnyMCP(lower, []string{"lohnsteuer", "l16", "lohnzettel"}) {
		result.DocumentSubtype = "lohnsteuer"
	} else if containsAnyMCP(lower, []string{"sozialversicherung", "svnr", "gkk", "bva", "svs"}) {
		result.DocumentSubtype = "sozialversicherung"
	}

	return result
}

type extractedDeadlineMCP struct {
	Type        string `json:"type"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Confidence  float64 `json:"confidence"`
}

func extractDeadlinesHeuristic(text string) []extractedDeadlineMCP {
	var deadlines []extractedDeadlineMCP

	// Find date patterns (DD.MM.YYYY)
	dates := findDatesInText(text)

	for _, dateMatch := range dates {
		if hasDeadlineContext(text, dateMatch) {
			deadline := extractedDeadlineMCP{
				Type:        determineDeadlineType(text, dateMatch),
				Date:        dateMatch,
				Description: "Extracted deadline",
				Confidence:  0.7,
			}
			deadlines = append(deadlines, deadline)
		}
	}

	return deadlines
}

func findDatesInText(text string) []string {
	var dates []string

	// Simple pattern matching for DD.MM.YYYY
	// In production, use regexp
	for i := 0; i < len(text)-9; i++ {
		if text[i] >= '0' && text[i] <= '3' {
			if i+9 < len(text) {
				potential := text[i : i+10]
				if isDatePattern(potential) {
					dates = append(dates, potential)
				}
			}
		}
	}

	return dates
}

func isDatePattern(s string) bool {
	if len(s) != 10 {
		return false
	}
	// Check DD.MM.YYYY pattern
	return s[2] == '.' && s[5] == '.' &&
		isDigit(s[0]) && isDigit(s[1]) &&
		isDigit(s[3]) && isDigit(s[4]) &&
		isDigit(s[6]) && isDigit(s[7]) && isDigit(s[8]) && isDigit(s[9])
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

func hasDeadlineContext(text, date string) bool {
	keywords := []string{"bis", "frist", "zahlbar", "einreich", "bis zum", "bis spätestens"}
	idx := indexOfMCP(text, date)
	if idx == -1 {
		return false
	}

	// Check context before date
	start := idx - 50
	if start < 0 {
		start = 0
	}
	context := toLower(text[start:idx])

	for _, kw := range keywords {
		if containsMCP(context, kw) {
			return true
		}
	}

	return false
}

func determineDeadlineType(text, date string) string {
	idx := indexOfMCP(text, date)
	if idx == -1 {
		return "other"
	}

	start := idx - 100
	if start < 0 {
		start = 0
	}
	context := toLower(text[start:idx])

	if containsMCP(context, "zahlung") || containsMCP(context, "entrichten") {
		return "payment"
	}
	if containsMCP(context, "berufung") || containsMCP(context, "beschwerde") {
		return "appeal"
	}
	if containsMCP(context, "einreich") || containsMCP(context, "abgeben") {
		return "submission"
	}

	return "response"
}

type extractedAmountMCP struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Type     string  `json:"type"`
	Context  string  `json:"context"`
}

func extractAmountsHeuristic(text string) []extractedAmountMCP {
	var amounts []extractedAmountMCP

	// Find EUR amount patterns
	// Look for patterns like: €1.234,56 or 1.234,56 EUR
	patterns := findAmountsInText(text)

	for _, p := range patterns {
		amounts = append(amounts, extractedAmountMCP{
			Amount:   p.Amount,
			Currency: "EUR",
			Type:     classifyAmountType(text, p.Position),
			Context:  getAmountContext(text, p.Position),
		})
	}

	return amounts
}

type amountMatch struct {
	Amount   float64
	Position int
}

func findAmountsInText(text string) []amountMatch {
	var matches []amountMatch

	// Simple pattern matching for Euro amounts
	// Iterate over runes to handle multi-byte characters like €
	runes := []rune(text)
	for i := 0; i < len(runes)-5; i++ {
		if runes[i] == '€' || (i > 0 && i+2 < len(runes) && string(runes[i:i+3]) == "EUR") {
			// Look for number after € or EUR
			searchStart := i + 1
			if string(runes[i:i+3]) == "EUR" {
				searchStart = i + 3
			}
			amount, length := parseAmount(string(runes), searchStart)
			if amount > 0 && amount < 10000000 {
				matches = append(matches, amountMatch{Amount: amount, Position: i})
				i += length
			}
		} else if isDigitRune(runes[i]) {
			// Check if this is part of an amount pattern
			amount, length := parseAmount(string(runes), i)
			if amount > 0 && amount < 10000000 && hasEuroIndicator(text, i, length) {
				matches = append(matches, amountMatch{Amount: amount, Position: i})
				i += length
			}
		}
	}

	return matches
}

func parseAmount(text string, start int) (float64, int) {
	// Skip whitespace
	for start < len(text) && text[start] == ' ' {
		start++
	}

	var numStr string
	i := start

	// Parse digits and separators
	for i < len(text) {
		c := text[i]
		if isDigit(c) {
			numStr += string(c)
		} else if c == '.' || c == ' ' {
			// Thousand separator - skip
		} else if c == ',' {
			// Decimal separator
			numStr += "."
		} else {
			break
		}
		i++
	}

	if numStr == "" {
		return 0, 0
	}

	// Parse float
	var amount float64
	for j := 0; j < len(numStr); j++ {
		c := numStr[j]
		if c == '.' {
			// Parse decimal part
			var decimal float64
			divisor := 10.0
			for k := j + 1; k < len(numStr); k++ {
				decimal += float64(numStr[k]-'0') / divisor
				divisor *= 10
			}
			amount += decimal
			break
		} else {
			amount = amount*10 + float64(c-'0')
		}
	}

	return amount, i - start
}

func hasEuroIndicator(text string, start, length int) bool {
	// Check for € or EUR before or after the amount
	before := ""
	if start > 5 {
		before = text[start-5 : start]
	} else if start > 0 {
		before = text[0:start]
	}

	after := ""
	end := start + length
	if end+4 < len(text) {
		after = text[end : end+4]
	}

	return containsMCP(before, "€") || containsMCP(before, "eur") ||
		containsMCP(after, "€") || containsMCP(after, "EUR") || containsMCP(after, "eur")
}

func classifyAmountType(text string, position int) string {
	start := position - 100
	if start < 0 {
		start = 0
	}
	context := toLower(text[start:position])

	if containsMCP(context, "nachzahlung") || containsMCP(context, "zahlen") {
		return "payment_due"
	}
	if containsMCP(context, "guthaben") || containsMCP(context, "erstattung") {
		return "refund"
	}
	if containsMCP(context, "steuer") || containsMCP(context, "abgabe") {
		return "tax"
	}
	if containsMCP(context, "strafe") || containsMCP(context, "zuschlag") {
		return "penalty"
	}

	return "other"
}

func getAmountContext(text string, position int) string {
	start := position - 30
	if start < 0 {
		start = 0
	}
	end := position + 30
	if end > len(text) {
		end = len(text)
	}
	return text[start:end]
}

type summarizationResult struct {
	Text         string   `json:"text"`
	WordCount    int      `json:"word_count"`
	KeySentences []string `json:"key_sentences"`
}

func generateBasicSummary(text string) summarizationResult {
	// Count words
	words := 0
	inWord := false
	for _, c := range text {
		if c == ' ' || c == '\n' || c == '\t' {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}

	// Extract first few sentences as key sentences
	var sentences []string
	var current string
	for _, c := range text {
		current += string(c)
		if c == '.' || c == '!' || c == '?' {
			trimmed := trimSpace(current)
			if len(trimmed) > 20 && len(sentences) < 3 {
				sentences = append(sentences, trimmed)
			}
			current = ""
		}
	}

	// Generate basic summary
	summary := "Dokument enthält " + itoa(words) + " Wörter."
	if len(sentences) > 0 {
		summary = sentences[0]
	}

	return summarizationResult{
		Text:         summary,
		WordCount:    words,
		KeySentences: sentences,
	}
}

// Helper functions for string operations

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

func containsAnyMCP(text string, keywords []string) bool {
	for _, kw := range keywords {
		if containsMCP(text, kw) {
			return true
		}
	}
	return false
}

func containsMCP(s, substr string) bool {
	return indexOfMCP(s, substr) >= 0
}

func indexOfMCP(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\n' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var result string
	for n > 0 {
		result = string('0'+byte(n%10)) + result
		n /= 10
	}
	return result
}
