package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// ServerConfig holds MCP server configuration
type ServerConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Server represents an MCP server
type Server struct {
	config   ServerConfig
	tools    map[string]*MCPTool
	handlers map[string]ToolHandler
	mu       sync.RWMutex
}

// ToolHandler is a function that executes a tool
type ToolHandler func(params map[string]interface{}) (interface{}, error)

// NewServer creates a new MCP server
func NewServer(config ServerConfig) *Server {
	return &Server{
		config:   config,
		tools:    make(map[string]*MCPTool),
		handlers: make(map[string]ToolHandler),
	}
}

// RegisterTool registers a tool with its handler
func (s *Server) RegisterTool(tool *MCPTool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[tool.Name] = tool
	s.handlers[tool.Name] = handler
}

// GetRegisteredTools returns all registered tools
func (s *Server) GetRegisteredTools() []*MCPTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]*MCPTool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ExecuteTool executes a tool by name with the given parameters
func (s *Server) ExecuteTool(name string, params map[string]interface{}) (interface{}, error) {
	s.mu.RLock()
	handler, ok := s.handlers[name]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}

	return handler(params)
}

// RegisterTools registers all Austrian Business Infrastructure tools
func (s *Server) RegisterTools() {
	// UID validation tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-uid-validate",
			Description: "Validate an Austrian or EU VAT identification number (UID)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"uid": map[string]interface{}{
						"type":        "string",
						"description": "The VAT identification number to validate (e.g., ATU12345678)",
					},
				},
				"required": []string{"uid"},
			},
		},
		handleUIDValidate,
	)

	// IBAN validation tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-iban-validate",
			Description: "Validate an IBAN (International Bank Account Number)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"iban": map[string]interface{}{
						"type":        "string",
						"description": "The IBAN to validate",
					},
				},
				"required": []string{"iban"},
			},
		},
		handleIBANValidate,
	)

	// BIC lookup tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-bic-lookup",
			Description: "Look up BIC for an Austrian bank code",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"bank_code": map[string]interface{}{
						"type":        "string",
						"description": "The Austrian bank code (BLZ)",
					},
				},
				"required": []string{"bank_code"},
			},
		},
		handleBICLookup,
	)

	// SV-Nummer validation tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-sv-nummer-validate",
			Description: "Validate an Austrian social security number (SV-Nummer)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"sv_nummer": map[string]interface{}{
						"type":        "string",
						"description": "The SV-Nummer to validate (10 digits)",
					},
				},
				"required": []string{"sv_nummer"},
			},
		},
		handleSVNummerValidate,
	)

	// FN validation tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-fn-validate",
			Description: "Validate an Austrian Firmenbuch number (FN)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fn": map[string]interface{}{
						"type":        "string",
						"description": "The Firmenbuch number to validate (e.g., FN123456a)",
					},
				},
				"required": []string{"fn"},
			},
		},
		handleFNValidate,
	)

	// ===== MCP Tools Expansion (003-mcp-tools-expansion) =====

	// Databox list tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-databox-list",
			Description: "List documents in a FinanzOnline databox",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"account_id": map[string]interface{}{
						"type":        "string",
						"description": "The account identifier (TID or alias)",
					},
					"from_date": map[string]interface{}{
						"type":        "string",
						"description": "Start date filter (YYYY-MM-DD format)",
					},
					"to_date": map[string]interface{}{
						"type":        "string",
						"description": "End date filter (YYYY-MM-DD format)",
					},
				},
				"required": []string{"account_id"},
			},
		},
		handleDataboxList,
	)

	// Databox download tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-databox-download",
			Description: "Download a document from a FinanzOnline databox",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"account_id": map[string]interface{}{
						"type":        "string",
						"description": "The account identifier (TID or alias)",
					},
					"document_id": map[string]interface{}{
						"type":        "string",
						"description": "The document ID (applkey) from databox list",
					},
				},
				"required": []string{"account_id", "document_id"},
			},
		},
		handleDataboxDownload,
	)

	// Firmenbuch search tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-fb-search",
			Description: "Search for companies in the Austrian Firmenbuch (company register)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Company name or search term",
					},
					"location": map[string]interface{}{
						"type":        "string",
						"description": "Filter by city/location",
					},
					"max_results": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results (default: 20)",
					},
				},
				"required": []string{"query"},
			},
		},
		handleFBSearch,
	)

	// Firmenbuch extract tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-fb-extract",
			Description: "Get full company details from the Austrian Firmenbuch",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fn": map[string]interface{}{
						"type":        "string",
						"description": "Firmenbuch number (e.g., FN123456a)",
					},
				},
				"required": []string{"fn"},
			},
		},
		handleFBExtract,
	)

	// UVA submit tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-uva-submit",
			Description: "Submit a UVA (VAT advance return) to FinanzOnline",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"account_id": map[string]interface{}{
						"type":        "string",
						"description": "The account identifier (TID or alias)",
					},
					"year": map[string]interface{}{
						"type":        "integer",
						"description": "Tax year (e.g., 2025)",
					},
					"period_type": map[string]interface{}{
						"type":        "string",
						"description": "Period type: monthly or quarterly",
					},
					"period_value": map[string]interface{}{
						"type":        "integer",
						"description": "Period value: 1-12 for monthly, 1-4 for quarterly",
					},
					"kz_values": map[string]interface{}{
						"type":        "object",
						"description": "KZ (Kennzahl) values in cents",
					},
				},
				"required": []string{"account_id", "year", "period_type", "period_value", "kz_values"},
			},
		},
		handleUVASubmit,
	)

	// ===== AI Document Intelligence Tools (011-ai-document-intelligence) =====

	// Document classification tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-document-classify",
			Description: "Classify an Austrian official document by type (Bescheid, Ersuchen, Mahnung, etc.)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The document text to classify",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Optional document title for better classification",
					},
				},
				"required": []string{"text"},
			},
		},
		handleDocumentClassify,
	)

	// Deadline extraction tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-deadline-extract",
			Description: "Extract deadlines (Fristen) from an Austrian official document",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The document text to extract deadlines from",
					},
				},
				"required": []string{"text"},
			},
		},
		handleDeadlineExtract,
	)

	// Amount extraction tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-amount-extract",
			Description: "Extract monetary amounts (Beträge) from an Austrian official document",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The document text to extract amounts from",
					},
				},
				"required": []string{"text"},
			},
		},
		handleAmountExtract,
	)

	// Document summarization tool
	s.RegisterTool(
		&MCPTool{
			Name:        "fo-document-summarize",
			Description: "Generate a plain-language summary of an Austrian official document",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The document text to summarize",
					},
				},
				"required": []string{"text"},
			},
		},
		handleDocumentSummarize,
	)
}

// RunStdio runs the MCP server using stdio transport
func (s *Server) RunStdio() error {
	reader := json.NewDecoder(os.Stdin)
	writer := json.NewEncoder(os.Stdout)

	for {
		var request MCPRequest
		if err := reader.Decode(&request); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("failed to read request: %w", err)
		}

		response := s.handleRequest(&request)
		if err := writer.Encode(response); err != nil {
			return fmt.Errorf("failed to write response: %w", err)
		}
	}
}

// handleRequest processes an MCP request
func (s *Server) handleRequest(req *MCPRequest) *MCPResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *MCPRequest) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.config.Name,
				"version": s.config.Version,
			},
		},
	}
}

// handleToolsList returns the list of available tools
func (s *Server) handleToolsList(req *MCPRequest) *MCPResponse {
	tools := s.GetRegisteredTools()

	toolList := make([]map[string]interface{}, len(tools))
	for i, tool := range tools {
		toolList[i] = map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": toolList,
		},
	}
}

// handleToolsCall executes a tool call
func (s *Server) handleToolsCall(req *MCPRequest) *MCPResponse {
	params, ok := req.Params.(map[string]interface{})
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	toolName, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]interface{})

	result, err := s.ExecuteTool(toolName, arguments)
	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: &MCPToolResult{
				Content: []MCPContent{
					{
						Type: "text",
						Text: err.Error(),
					},
				},
				IsError: true,
			},
		}
	}

	// Convert result to JSON text
	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: &MCPToolResult{
			Content: []MCPContent{
				{
					Type: "text",
					Text: string(resultJSON),
				},
			},
			IsError: false,
		},
	}
}
