package matcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/foerderung"
)

// ClaudeLLMClient implements LLMClient using Claude API
type ClaudeLLMClient struct {
	apiKey     string
	model      string
	maxTokens  int
	temperature float64
	httpClient *http.Client
}

// NewClaudeLLMClient creates a new Claude LLM client
func NewClaudeLLMClient(apiKey string, cfg *config.FoerderungConfig) *ClaudeLLMClient {
	return &ClaudeLLMClient{
		apiKey:      apiKey,
		model:       cfg.LLMModel,
		maxTokens:   cfg.LLMMaxTokens,
		temperature: cfg.LLMTemperature,
		httpClient: &http.Client{
			Timeout: cfg.LLMTimeout,
		},
	}
}

// ClaudeRequest represents the Claude API request format
type ClaudeRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system"`
	Messages    []ClaudeMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
}

// ClaudeMessage represents a message in the Claude API
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the Claude API response
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// AnalystResponse represents the expected JSON response from the LLM
type AnalystResponse struct {
	Eligible         bool     `json:"eligible"`
	Confidence       string   `json:"confidence"`
	Score            int      `json:"score"`
	MatchedCriteria  []string `json:"matchedCriteria"`
	ImplicitMatches  []string `json:"implicitMatches"`
	Concerns         []string `json:"concerns"`
	EstimatedAmount  *struct {
		Min   int    `json:"min"`
		Max   int    `json:"max"`
		Basis string `json:"basis"`
	} `json:"estimatedAmount"`
	KombinierbarMit []string `json:"kombinierbarMit"`
	NextSteps       []struct {
		Schritt string `json:"schritt"`
		URL     string `json:"url,omitempty"`
		Frist   string `json:"frist,omitempty"`
	} `json:"nextSteps"`
	InsiderTipp *string `json:"insiderTipp"`
}

// AnalyzeEligibility implements the LLMClient interface
func (c *ClaudeLLMClient) AnalyzeEligibility(
	ctx context.Context,
	profile *ProfileInput,
	fd *foerderung.Foerderung,
) (*foerderung.LLMEligibilityResult, error) {
	analyst, dataGuard := GetSystemPrompts()

	// Create user prompt with sanitized data
	userPrompt := c.createUserPrompt(profile, fd)

	// Combine system prompts
	systemPrompt := analyst + "\n\n" + dataGuard

	// Create request
	req := ClaudeRequest{
		Model:       c.model,
		MaxTokens:   c.maxTokens,
		System:      systemPrompt,
		Temperature: c.temperature,
		Messages: []ClaudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API error (status %d): %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if claudeResp.Error != nil {
		return nil, fmt.Errorf("Claude API error: %s", claudeResp.Error.Message)
	}

	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("no content in Claude response")
	}

	// Parse the JSON response
	responseText := claudeResp.Content[0].Text
	result, err := c.parseAnalystResponse(responseText, fd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analyst response: %w", err)
	}

	return result, nil
}

// createUserPrompt creates the user prompt with sanitized data
// Matches TypeScript: erstelleUserPrompt()
func (c *ClaudeLLMClient) createUserPrompt(profile *ProfileInput, fd *foerderung.Foerderung) string {
	currentYear := time.Now().Year()
	companyAge := 0
	if profile.FoundedYear != nil {
		companyAge = currentYear - *profile.FoundedYear
	}

	data := map[string]interface{}{
		"unternehmen": map[string]interface{}{
			"name":             sanitizeString(profile.CompanyName),
			"rechtsform":       sanitizeString(profile.LegalForm),
			"gruendungsjahr":   profile.FoundedYear,
			"alter":            companyAge,
			"bundesland":       sanitizeString(profile.State),
			"mitarbeiter":      profile.EmployeesCount,
			"jahresumsatz":     profile.AnnualRevenue,
			"branche":          sanitizeString(profile.Industry),
			"onaceCode":        sanitizeStrings(profile.OnaceCodes),
			"vorhaben":         sanitizeString(profile.ProjectDescription),
			"themen":           sanitizeStrings(profile.ProjectTopics),
			"investitionsSumme": profile.InvestmentAmount,
		},
		"foerderung": map[string]interface{}{
			"id":              fd.ID.String(),
			"name":            sanitizeString(fd.Name),
			"traeger":         sanitizeString(fd.Provider),
			"bundesland":      fd.TargetStates,
			"art":             string(fd.Type),
			"maxBetrag":       fd.MaxAmount,
			"foerderquote":    fd.FundingRateMax,
			"minProjektkosten": fd.MinAmount,
			"beschreibung":    sanitizeStringPtr(fd.Description),
			"detailkriterien": sanitizeStringPtr(fd.Requirements),
			"themen":          fd.Topics,
			"zielgruppe": map[string]interface{}{
				"groessen":          fd.TargetSizes,
				"maxAlterJahre":     fd.TargetAgeMax,
				"minAlterJahre":     fd.TargetAgeMin,
				"branchen":          fd.TargetIndustries,
				"branchenAusschluss": fd.ExcludedIndustries,
			},
			"einreichfrist": formatDeadline(fd.ApplicationDeadline),
			"quelle":        sanitizeStringPtr(fd.URL),
		},
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

// parseAnalystResponse parses the LLM JSON response into LLMEligibilityResult
func (c *ClaudeLLMClient) parseAnalystResponse(responseText string, fd *foerderung.Foerderung) (*foerderung.LLMEligibilityResult, error) {
	// Try to extract JSON from response (might have markdown code blocks)
	jsonStr := extractJSON(responseText)

	var analyst AnalystResponse
	if err := json.Unmarshal([]byte(jsonStr), &analyst); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Convert to LLMEligibilityResult
	result := &foerderung.LLMEligibilityResult{
		Eligible:        analyst.Eligible,
		Confidence:      analyst.Confidence,
		MatchedCriteria: analyst.MatchedCriteria,
		ImplicitMatches: analyst.ImplicitMatches,
		Concerns:        analyst.Concerns,
	}

	// Convert estimated amount
	if analyst.EstimatedAmount != nil {
		// Use max as the estimate
		result.EstimatedAmount = &analyst.EstimatedAmount.Max
	}

	// Convert next steps
	if len(analyst.NextSteps) > 0 {
		steps := make([]string, 0, len(analyst.NextSteps))
		for _, step := range analyst.NextSteps {
			steps = append(steps, step.Schritt)
		}
		result.NextSteps = steps
	}

	// Copy insider tip
	if analyst.InsiderTipp != nil && *analyst.InsiderTipp != "" {
		result.InsiderTip = analyst.InsiderTipp
	}

	// Create combination hint from kombinierbarMit
	if len(analyst.KombinierbarMit) > 0 {
		hint := "Kombinierbar mit: " + strings.Join(analyst.KombinierbarMit, ", ")
		result.CombinationHint = &hint
	}

	return result, nil
}

// Helper functions

func sanitizeString(s string) string {
	// Remove control characters except newlines/tabs
	result := strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(result)
}

func sanitizeStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return sanitizeString(*s)
}

func sanitizeStrings(ss []string) []string {
	result := make([]string, 0, len(ss))
	for _, s := range ss {
		if sanitized := sanitizeString(s); sanitized != "" {
			result = append(result, sanitized)
		}
	}
	return result
}

func formatDeadline(t *time.Time) string {
	if t == nil {
		return "laufend"
	}
	return t.Format("02.01.2006")
}

func extractJSON(text string) string {
	// Try to find JSON in markdown code blocks
	if start := strings.Index(text, "```json"); start != -1 {
		start += 7
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Try to find JSON in plain code blocks
	if start := strings.Index(text, "```"); start != -1 {
		start += 3
		if end := strings.Index(text[start:], "```"); end != -1 {
			return strings.TrimSpace(text[start : start+end])
		}
	}

	// Find JSON object directly
	if start := strings.Index(text, "{"); start != -1 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(text); i++ {
			if text[i] == '{' {
				depth++
			} else if text[i] == '}' {
				depth--
				if depth == 0 {
					return text[start : i+1]
				}
			}
		}
	}

	return text
}

// DeepSeekLLMClient implements LLMClient using DeepSeek API
// This matches the TypeScript implementation exactly
type DeepSeekLLMClient struct {
	apiKey      string
	model       string
	maxTokens   int
	temperature float64
	httpClient  *http.Client
}

// NewDeepSeekLLMClient creates a new DeepSeek LLM client
func NewDeepSeekLLMClient(apiKey string, cfg *config.FoerderungConfig) *DeepSeekLLMClient {
	return &DeepSeekLLMClient{
		apiKey:      apiKey,
		model:       "deepseek-chat", // TypeScript default
		maxTokens:   cfg.LLMMaxTokens,
		temperature: cfg.LLMTemperature,
		httpClient: &http.Client{
			Timeout: cfg.LLMTimeout,
		},
	}
}

// DeepSeekRequest represents the DeepSeek API request format
type DeepSeekRequest struct {
	Model          string            `json:"model"`
	Messages       []DeepSeekMessage `json:"messages"`
	Temperature    float64           `json:"temperature"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

// DeepSeekMessage represents a message in the DeepSeek API
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekResponse represents the DeepSeek API response
type DeepSeekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// AnalyzeEligibility implements the LLMClient interface for DeepSeek
func (c *DeepSeekLLMClient) AnalyzeEligibility(
	ctx context.Context,
	profile *ProfileInput,
	fd *foerderung.Foerderung,
) (*foerderung.LLMEligibilityResult, error) {
	analyst, dataGuard := GetSystemPrompts()

	// Create user prompt
	userPrompt := c.createUserPrompt(profile, fd)

	// Create request (TypeScript style: two system messages)
	req := DeepSeekRequest{
		Model:       c.model,
		Temperature: c.temperature,
		Messages: []DeepSeekMessage{
			{Role: "system", Content: analyst},
			{Role: "system", Content: dataGuard},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &struct {
			Type string `json:"type"`
		}{Type: "json_object"},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DeepSeek API error (status %d): %s", resp.StatusCode, string(body))
	}

	var dsResp DeepSeekResponse
	if err := json.Unmarshal(body, &dsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if dsResp.Error != nil {
		return nil, fmt.Errorf("DeepSeek API error: %s", dsResp.Error.Message)
	}

	if len(dsResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from DeepSeek")
	}

	// Parse the JSON response
	responseText := dsResp.Choices[0].Message.Content
	result, err := c.parseAnalystResponse(responseText, fd)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analyst response: %w", err)
	}

	return result, nil
}

// createUserPrompt creates the user prompt with sanitized data
func (c *DeepSeekLLMClient) createUserPrompt(profile *ProfileInput, fd *foerderung.Foerderung) string {
	currentYear := time.Now().Year()
	companyAge := 0
	if profile.FoundedYear != nil {
		companyAge = currentYear - *profile.FoundedYear
	}

	data := map[string]interface{}{
		"unternehmen": map[string]interface{}{
			"name":             sanitizeString(profile.CompanyName),
			"rechtsform":       sanitizeString(profile.LegalForm),
			"gruendungsjahr":   profile.FoundedYear,
			"alter":            companyAge,
			"bundesland":       sanitizeString(profile.State),
			"mitarbeiter":      profile.EmployeesCount,
			"jahresumsatz":     profile.AnnualRevenue,
			"branche":          sanitizeString(profile.Industry),
			"onaceCode":        sanitizeStrings(profile.OnaceCodes),
			"vorhaben":         sanitizeString(profile.ProjectDescription),
			"themen":           sanitizeStrings(profile.ProjectTopics),
			"investitionsSumme": profile.InvestmentAmount,
		},
		"foerderung": map[string]interface{}{
			"id":              fd.ID.String(),
			"name":            sanitizeString(fd.Name),
			"traeger":         sanitizeString(fd.Provider),
			"bundesland":      fd.TargetStates,
			"art":             string(fd.Type),
			"maxBetrag":       fd.MaxAmount,
			"foerderquote":    fd.FundingRateMax,
			"minProjektkosten": fd.MinAmount,
			"beschreibung":    sanitizeStringPtr(fd.Description),
			"detailkriterien": sanitizeStringPtr(fd.Requirements),
			"themen":          fd.Topics,
			"zielgruppe": map[string]interface{}{
				"groessen":          fd.TargetSizes,
				"maxAlterJahre":     fd.TargetAgeMax,
				"minAlterJahre":     fd.TargetAgeMin,
				"branchen":          fd.TargetIndustries,
				"branchenAusschluss": fd.ExcludedIndustries,
			},
			"einreichfrist": formatDeadline(fd.ApplicationDeadline),
			"quelle":        sanitizeStringPtr(fd.URL),
		},
	}

	jsonData, _ := json.MarshalIndent(data, "", "  ")
	return string(jsonData)
}

// parseAnalystResponse parses the LLM JSON response into LLMEligibilityResult
func (c *DeepSeekLLMClient) parseAnalystResponse(responseText string, fd *foerderung.Foerderung) (*foerderung.LLMEligibilityResult, error) {
	jsonStr := extractJSON(responseText)

	var analyst AnalystResponse
	if err := json.Unmarshal([]byte(jsonStr), &analyst); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result := &foerderung.LLMEligibilityResult{
		Eligible:        analyst.Eligible,
		Confidence:      analyst.Confidence,
		MatchedCriteria: analyst.MatchedCriteria,
		ImplicitMatches: analyst.ImplicitMatches,
		Concerns:        analyst.Concerns,
	}

	if analyst.EstimatedAmount != nil {
		result.EstimatedAmount = &analyst.EstimatedAmount.Max
	}

	if len(analyst.NextSteps) > 0 {
		steps := make([]string, 0, len(analyst.NextSteps))
		for _, step := range analyst.NextSteps {
			steps = append(steps, step.Schritt)
		}
		result.NextSteps = steps
	}

	if analyst.InsiderTipp != nil && *analyst.InsiderTipp != "" {
		result.InsiderTip = analyst.InsiderTipp
	}

	if len(analyst.KombinierbarMit) > 0 {
		hint := "Kombinierbar mit: " + strings.Join(analyst.KombinierbarMit, ", ")
		result.CombinationHint = &hint
	}

	return result, nil
}

// CreateFallbackResult creates a fallback result when LLM fails
// Matches TypeScript: erstelleFallbackErgebnis()
func CreateFallbackResult() *foerderung.LLMEligibilityResult {
	return &foerderung.LLMEligibilityResult{
		Eligible:        true,
		Confidence:      ConfidenceLow,
		MatchedCriteria: []string{},
		ImplicitMatches: []string{},
		Concerns:        []string{"Automatische Analyse nicht verfügbar"},
		NextSteps:       []string{"Förderung manuell auf Eignung prüfen"},
	}
}
