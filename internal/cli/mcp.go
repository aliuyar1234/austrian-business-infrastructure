package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/austrian-business-infrastructure/fo/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP (Model Context Protocol) server for AI integration",
	Long: `MCP server allows AI assistants like Claude to interact with fo tools.

Commands:
  serve  - Start the MCP server (stdio transport)
  tools  - List available MCP tools`,
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP server using stdio transport.

The server exposes fo validation tools to MCP-compatible AI clients.
This is typically invoked by an AI client, not manually.

Available tools:
  fo-uid-validate       - Validate EU VAT identification numbers
  fo-iban-validate      - Validate IBANs
  fo-bic-lookup         - Look up BIC for Austrian bank codes
  fo-sv-nummer-validate - Validate Austrian social security numbers
  fo-fn-validate        - Validate Austrian Firmenbuch numbers`,
	RunE: runMCPServe,
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List available MCP tools",
	Long:  `List all tools available through the MCP server.`,
	RunE:  runMCPTools,
}

func init() {
	mcpCmd.AddCommand(mcpServeCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServe(cmd *cobra.Command, args []string) error {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo",
		Version: Version,
	})
	server.RegisterTools()

	LogVerbose("Starting MCP server...")
	return server.RunStdio()
}

func runMCPTools(cmd *cobra.Command, args []string) error {
	server := mcp.NewServer(mcp.ServerConfig{
		Name:    "fo",
		Version: Version,
	})
	server.RegisterTools()

	tools := server.GetRegisteredTools()

	if IsJSONOutput() {
		toolList := make([]map[string]interface{}, len(tools))
		for i, tool := range tools {
			toolList[i] = map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": tool.InputSchema,
			}
		}
		data, _ := json.MarshalIndent(toolList, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	fmt.Printf("Available MCP Tools (%d)\n\n", len(tools))
	for _, tool := range tools {
		fmt.Printf("  %s\n", tool.Name)
		fmt.Printf("    %s\n\n", tool.Description)
	}

	fmt.Println("Use `fo mcp serve` to start the MCP server.")
	fmt.Println("Configure your AI client with:")
	fmt.Println()
	fmt.Println(`  {`)
	fmt.Println(`    "mcpServers": {`)
	fmt.Println(`      "fo": {`)
	fmt.Println(`        "command": "fo",`)
	fmt.Println(`        "args": ["mcp", "serve"]`)
	fmt.Println(`      }`)
	fmt.Println(`    }`)
	fmt.Println(`  }`)

	return nil
}

// GenerateMCPConfig generates the MCP configuration for fo
func GenerateMCPConfig() map[string]interface{} {
	exePath, _ := os.Executable()
	if exePath == "" {
		exePath = "fo"
	}

	return map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"fo": map[string]interface{}{
				"command": exePath,
				"args":    []string{"mcp", "serve"},
			},
		},
	}
}
