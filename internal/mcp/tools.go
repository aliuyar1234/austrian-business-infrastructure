package mcp

// MCPTool represents a tool that can be called via MCP
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPContent represents content in an MCP response
type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// MCPToolResult represents the result of a tool call
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an outgoing MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an error in the MCP protocol
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
