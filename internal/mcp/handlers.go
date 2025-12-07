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
