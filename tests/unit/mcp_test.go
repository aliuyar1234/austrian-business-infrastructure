package unit

import (
	"testing"

	"github.com/austrian-business-infrastructure/fo/internal/mcp"
)

// T123: Test MCP tool registration
func TestMCPToolRegistration(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})

	// Register tools
	server.RegisterTools()

	// Verify tools are registered
	tools := server.GetRegisteredTools()

	expectedTools := []string{
		"fo-uid-validate",
		"fo-iban-validate",
		"fo-bic-lookup",
		"fo-sv-nummer-validate",
		"fo-fn-validate",
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %s to be registered", expected)
		}
	}
}

// T124: Test MCP tool handler execution
func TestMCPToolHandlerExecution(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	// Test fo-uid-validate tool
	result, err := server.ExecuteTool("fo-uid-validate", map[string]interface{}{
		"uid": "ATU12345678",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-uid-validate: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// Test fo-iban-validate tool
	result, err = server.ExecuteTool("fo-iban-validate", map[string]interface{}{
		"iban": "AT611904300234573201",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-iban-validate: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if valid, ok := resultMap["valid"].(bool); !ok || !valid {
		t.Error("Expected valid IBAN")
	}

	// Test fo-bic-lookup tool
	result, err = server.ExecuteTool("fo-bic-lookup", map[string]interface{}{
		"bank_code": "19043",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-bic-lookup: %v", err)
	}

	resultMap, ok = result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if bic, ok := resultMap["bic"].(string); !ok || bic != "BKAUATWW" {
		t.Errorf("Expected BIC BKAUATWW, got %v", resultMap["bic"])
	}

	// Test unknown tool
	_, err = server.ExecuteTool("unknown-tool", nil)
	if err == nil {
		t.Error("Expected error for unknown tool")
	}
}

// Test MCPTool struct
func TestMCPTool(t *testing.T) {
	tool := &mcp.MCPTool{
		Name:        "test-tool",
		Description: "A test tool",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type":        "string",
					"description": "First parameter",
				},
			},
		},
	}

	if tool.Name != "test-tool" {
		t.Errorf("Expected name test-tool, got %s", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got %s", tool.Description)
	}
}

// Test MCPToolResult struct
func TestMCPToolResult(t *testing.T) {
	result := &mcp.MCPToolResult{
		Content: []mcp.MCPContent{
			{
				Type: "text",
				Text: "Result text",
			},
		},
		IsError: false,
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content, got %d", len(result.Content))
	}
	if result.Content[0].Text != "Result text" {
		t.Errorf("Expected text 'Result text', got %s", result.Content[0].Text)
	}
	if result.IsError {
		t.Error("Expected IsError to be false")
	}
}

// Test ServerConfig
func TestServerConfig(t *testing.T) {
	config := mcp.ServerConfig{
		Name:    "test-server",
		Version: "1.0.0",
	}

	if config.Name != "test-server" {
		t.Errorf("Expected name test-server, got %s", config.Name)
	}
	if config.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", config.Version)
	}
}

// Test NewServer
func TestNewServer(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo",
		Version: "1.0.0",
	})

	if server == nil {
		t.Fatal("Expected non-nil server")
	}
}

// Test tool parameter validation
func TestToolParameterValidation(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	// Test with missing required parameter
	_, err := server.ExecuteTool("fo-uid-validate", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing uid parameter")
	}

	// Test with empty IBAN
	_, err = server.ExecuteTool("fo-iban-validate", map[string]interface{}{
		"iban": "",
	})
	if err == nil {
		t.Error("Expected error for empty iban parameter")
	}
}

// T125: Test SV-Nummer validation via MCP
func TestMCPSVNummerValidation(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	// Test valid SV-Nummer (1234150189 encodes birth date 15.01.89)
	result, err := server.ExecuteTool("fo-sv-nummer-validate", map[string]interface{}{
		"sv_nummer": "1234150189",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-sv-nummer-validate: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if valid, ok := resultMap["valid"].(bool); !ok || !valid {
		t.Errorf("Expected valid SV-Nummer, got: %v", resultMap)
	}

	// Check that birth_date_iso contains the correct date
	if birthDateISO, ok := resultMap["birth_date_iso"].(string); !ok || birthDateISO != "1989-01-15" {
		t.Errorf("Expected birth_date_iso 1989-01-15, got %v", resultMap["birth_date_iso"])
	}

	// Test invalid SV-Nummer
	result, err = server.ExecuteTool("fo-sv-nummer-validate", map[string]interface{}{
		"sv_nummer": "1234567890",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-sv-nummer-validate: %v", err)
	}

	resultMap, ok = result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if valid, ok := resultMap["valid"].(bool); ok && valid {
		t.Error("Expected invalid SV-Nummer")
	}
}

// T126: Test FN validation via MCP
func TestMCPFNValidation(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	// Test valid FN
	result, err := server.ExecuteTool("fo-fn-validate", map[string]interface{}{
		"fn": "FN123456a",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-fn-validate: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if valid, ok := resultMap["valid"].(bool); !ok || !valid {
		t.Errorf("Expected valid FN, got: %v", resultMap)
	}

	// Test invalid FN
	result, err = server.ExecuteTool("fo-fn-validate", map[string]interface{}{
		"fn": "INVALID",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-fn-validate: %v", err)
	}

	resultMap, ok = result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if valid, ok := resultMap["valid"].(bool); ok && valid {
		t.Error("Expected invalid FN")
	}
}

// T127: Test BIC lookup with unknown bank code
func TestMCPBICLookupUnknown(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-bic-lookup", map[string]interface{}{
		"bank_code": "99999",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-bic-lookup: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	if found, ok := resultMap["found"].(bool); ok && found {
		t.Error("Expected not found for unknown bank code")
	}
}

// T128: Test UID validation for different EU countries
func TestMCPUIDValidationCountries(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	testCases := []struct {
		uid          string
		valid        bool
		countryCode  string
	}{
		{"ATU12345678", true, "AT"},
		{"DE123456789", true, "DE"},
		{"INVALID", false, "IN"},
	}

	for _, tc := range testCases {
		t.Run(tc.uid, func(t *testing.T) {
			result, err := server.ExecuteTool("fo-uid-validate", map[string]interface{}{
				"uid": tc.uid,
			})
			if err != nil {
				t.Fatalf("Failed to execute fo-uid-validate: %v", err)
			}

			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Fatal("Expected map result")
			}

			if valid, ok := resultMap["valid"].(bool); ok && valid != tc.valid {
				t.Errorf("Expected valid=%v for UID %s, got %v", tc.valid, tc.uid, valid)
			}
		})
	}
}

// ===== MCP Tools Expansion Tests (003-mcp-tools-expansion) =====

// T129: Test databox list tool returns auth error without session
func TestMCPDataboxListAuthError(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-databox-list", map[string]interface{}{
		"account_id": "test-account",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-databox-list: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return an auth error since no session is available
	if isError, ok := resultMap["error"].(bool); !ok || !isError {
		t.Error("Expected error=true for missing session")
	}
	if errType, ok := resultMap["error_type"].(string); !ok || errType != "authentication" {
		t.Errorf("Expected error_type=authentication, got %v", resultMap["error_type"])
	}
}

// T130: Test databox download tool returns auth error without session
func TestMCPDataboxDownloadAuthError(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-databox-download", map[string]interface{}{
		"account_id":  "test-account",
		"document_id": "ABC123",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-databox-download: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return an auth error since no session is available
	if isError, ok := resultMap["error"].(bool); !ok || !isError {
		t.Error("Expected error=true for missing session")
	}
}

// T131: Test databox list missing account_id parameter
func TestMCPDataboxListMissingParam(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	_, err := server.ExecuteTool("fo-databox-list", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing account_id parameter")
	}
}

// T132: Test FB search tool (no session required)
func TestMCPFBSearch(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-fb-search", map[string]interface{}{
		"query": "Test GmbH",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-fb-search: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return a valid response structure
	if _, ok := resultMap["results"]; !ok {
		t.Error("Expected results field in response")
	}
}

// T133: Test FB extract tool with valid FN format
func TestMCPFBExtract(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-fb-extract", map[string]interface{}{
		"fn": "FN123456a",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-fb-extract: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return a valid response structure (may be mock data or service error)
	if resultMap["fn"] != "FN123456a" {
		t.Errorf("Expected fn=FN123456a in response, got %v", resultMap["fn"])
	}
}

// T134: Test FB extract with invalid FN format
func TestMCPFBExtractInvalidFN(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-fb-extract", map[string]interface{}{
		"fn": "INVALID",
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-fb-extract: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return validation error
	if isError, ok := resultMap["error"].(bool); !ok || !isError {
		t.Error("Expected error=true for invalid FN format")
	}
	if errType, ok := resultMap["error_type"].(string); !ok || errType != "validation" {
		t.Errorf("Expected error_type=validation, got %v", resultMap["error_type"])
	}
}

// T135: Test FB search missing query parameter
func TestMCPFBSearchMissingParam(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	_, err := server.ExecuteTool("fo-fb-search", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing query parameter")
	}
}

// T136: Test UVA submit tool returns auth error without session
func TestMCPUVASubmitAuthError(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-uva-submit", map[string]interface{}{
		"account_id":   "test-account",
		"year":         float64(2025), // JSON numbers are float64
		"period_type":  "monthly",
		"period_value": float64(1),
		"kz_values": map[string]interface{}{
			"kz000": float64(1000000),
			"kz017": float64(800000),
		},
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-uva-submit: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return an auth error since no session is available
	if isError, ok := resultMap["error"].(bool); !ok || !isError {
		t.Error("Expected error=true for missing session")
	}
	if errType, ok := resultMap["error_type"].(string); !ok || errType != "authentication" {
		t.Errorf("Expected error_type=authentication, got %v", resultMap["error_type"])
	}
}

// T137: Test UVA submit with invalid period
func TestMCPUVASubmitValidationError(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	result, err := server.ExecuteTool("fo-uva-submit", map[string]interface{}{
		"account_id":   "test-account",
		"year":         float64(2025),
		"period_type":  "monthly",
		"period_value": float64(13), // Invalid month
		"kz_values":    map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("Failed to execute fo-uva-submit: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("Expected map result")
	}

	// Should return validation error
	if isError, ok := resultMap["error"].(bool); !ok || !isError {
		t.Error("Expected error=true for invalid period")
	}
	if errType, ok := resultMap["error_type"].(string); !ok || errType != "validation" {
		t.Errorf("Expected error_type=validation, got %v", resultMap["error_type"])
	}
}

// T138: Test UVA submit missing required parameters
func TestMCPUVASubmitMissingParams(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	_, err := server.ExecuteTool("fo-uva-submit", map[string]interface{}{})
	if err == nil {
		t.Error("Expected error for missing parameters")
	}
}

// T139: Test all 10 tools are registered
func TestMCPAllToolsRegistered(t *testing.T) {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo-test",
		Version: "1.0.0",
	})
	server.RegisterTools()

	tools := server.GetRegisteredTools()

	expectedTools := []string{
		// Original 5 validation tools
		"fo-uid-validate",
		"fo-iban-validate",
		"fo-bic-lookup",
		"fo-sv-nummer-validate",
		"fo-fn-validate",
		// 5 operational tools
		"fo-databox-list",
		"fo-databox-download",
		"fo-fb-search",
		"fo-fb-extract",
		"fo-uva-submit",
		// 4 AI document intelligence tools (011-ai-document-intelligence)
		"fo-document-classify",
		"fo-deadline-extract",
		"fo-amount-extract",
		"fo-document-summarize",
	}

	for _, expected := range expectedTools {
		found := false
		for _, tool := range tools {
			if tool.Name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tool %s to be registered", expected)
		}
	}

	if len(tools) != 14 {
		t.Errorf("Expected 14 tools, got %d", len(tools))
	}
}
