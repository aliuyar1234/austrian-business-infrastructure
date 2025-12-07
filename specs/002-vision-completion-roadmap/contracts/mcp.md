# Contracts: MCP Server - AI Integration

**Module**: mcp
**Date**: 2025-12-07

## 1. Overview

The MCP (Model Context Protocol) server exposes Austrian Business Infrastructure functionality to AI assistants (Claude, GPT, etc.) through a standardized protocol.

**Transport**: stdio (for Claude Desktop integration)
**Protocol**: JSON-RPC 2.0 over stdio
**SDK**: mcp-go or official Go SDK

---

## 2. Server Capabilities

### 2.1 Server Info

```json
{
    "protocolVersion": "2024-11-05",
    "capabilities": {
        "tools": {
            "listChanged": false
        }
    },
    "serverInfo": {
        "name": "austrian-business-infrastructure",
        "version": "1.0.0"
    }
}
```

---

## 3. Tool Definitions

### 3.1 FinanzOnline Tools

#### fo-databox-list

```json
{
    "name": "fo-databox-list",
    "description": "List documents in the FinanzOnline Databox for an account",
    "inputSchema": {
        "type": "object",
        "properties": {
            "account": {
                "type": "string",
                "description": "Account name stored in credentials"
            },
            "from": {
                "type": "string",
                "description": "Filter from date (YYYY-MM-DD)",
                "format": "date"
            },
            "to": {
                "type": "string",
                "description": "Filter to date (YYYY-MM-DD)",
                "format": "date"
            }
        },
        "required": ["account"]
    }
}
```

**Response**:
```json
{
    "content": [
        {
            "type": "text",
            "text": "Found 3 documents in databox for account 'firma-xyz':\n\n1. 2025-01-15 - Bescheid (⚠️ Action Required)\n   Applkey: ABC123\n\n2. 2025-01-10 - Mitteilung\n   Applkey: DEF456\n\n3. 2025-01-05 - Bescheid\n   Applkey: GHI789"
        }
    ]
}
```

#### fo-databox-download

```json
{
    "name": "fo-databox-download",
    "description": "Download a document from the FinanzOnline Databox",
    "inputSchema": {
        "type": "object",
        "properties": {
            "account": {
                "type": "string",
                "description": "Account name"
            },
            "applkey": {
                "type": "string",
                "description": "Document identifier (applkey)"
            },
            "output_dir": {
                "type": "string",
                "description": "Output directory (default: current directory)"
            }
        },
        "required": ["account", "applkey"]
    }
}
```

#### fo-uid-validate

```json
{
    "name": "fo-uid-validate",
    "description": "Validate an EU VAT identification number (UID)",
    "inputSchema": {
        "type": "object",
        "properties": {
            "uid": {
                "type": "string",
                "description": "UID number to validate (e.g., ATU12345678)"
            }
        },
        "required": ["uid"]
    }
}
```

**Response (valid)**:
```json
{
    "content": [
        {
            "type": "text",
            "text": "✅ UID ATU12345678 is VALID\n\nCompany: Musterfirma GmbH\nAddress: Musterstraße 1, 1010 Wien\nValidated at: 2025-01-15 14:30:00"
        }
    ]
}
```

**Response (invalid)**:
```json
{
    "content": [
        {
            "type": "text",
            "text": "❌ UID ATU99999999 is INVALID\n\nError: UID number not found in registry"
        }
    ],
    "isError": false
}
```

#### fo-uva-submit

```json
{
    "name": "fo-uva-submit",
    "description": "Submit a VAT advance return (UVA) to FinanzOnline",
    "inputSchema": {
        "type": "object",
        "properties": {
            "account": {
                "type": "string",
                "description": "Account name"
            },
            "year": {
                "type": "integer",
                "description": "Tax year"
            },
            "month": {
                "type": "integer",
                "description": "Tax month (1-12) for monthly filing"
            },
            "quarter": {
                "type": "integer",
                "description": "Tax quarter (1-4) for quarterly filing"
            },
            "kz017": {
                "type": "number",
                "description": "KZ017 - 20% VAT taxable amount"
            },
            "kz060": {
                "type": "number",
                "description": "KZ060 - Input tax (Vorsteuer)"
            }
        },
        "required": ["account", "year"]
    }
}
```

### 3.2 ELDA Tools

#### elda-anmeldung

```json
{
    "name": "elda-anmeldung",
    "description": "Register an employee with social security (ELDA)",
    "inputSchema": {
        "type": "object",
        "properties": {
            "account": {
                "type": "string",
                "description": "ELDA account name"
            },
            "svnummer": {
                "type": "string",
                "description": "10-digit social security number"
            },
            "vorname": {
                "type": "string",
                "description": "First name"
            },
            "nachname": {
                "type": "string",
                "description": "Last name"
            },
            "geburtsdatum": {
                "type": "string",
                "description": "Date of birth (YYYY-MM-DD)",
                "format": "date"
            },
            "eintrittsdatum": {
                "type": "string",
                "description": "Start date (YYYY-MM-DD)",
                "format": "date"
            },
            "brutto": {
                "type": "number",
                "description": "Monthly gross salary in EUR"
            },
            "art": {
                "type": "string",
                "enum": ["vollzeit", "teilzeit", "geringfuegig"],
                "description": "Employment type"
            }
        },
        "required": ["account", "svnummer", "vorname", "nachname", "geburtsdatum", "eintrittsdatum", "brutto", "art"]
    }
}
```

#### elda-status

```json
{
    "name": "elda-status",
    "description": "Check status of an ELDA submission",
    "inputSchema": {
        "type": "object",
        "properties": {
            "account": {
                "type": "string"
            },
            "reference": {
                "type": "string",
                "description": "ELDA reference number"
            }
        },
        "required": ["account", "reference"]
    }
}
```

### 3.3 Firmenbuch Tools

#### fb-search

```json
{
    "name": "fb-search",
    "description": "Search the Austrian company register (Firmenbuch)",
    "inputSchema": {
        "type": "object",
        "properties": {
            "name": {
                "type": "string",
                "description": "Company name (fuzzy search)"
            },
            "fn": {
                "type": "string",
                "description": "Firmenbuchnummer (exact search)"
            },
            "ort": {
                "type": "string",
                "description": "Location filter"
            },
            "rechtsform": {
                "type": "string",
                "description": "Legal form filter (GmbH, AG, etc.)"
            }
        }
    }
}
```

**Response**:
```json
{
    "content": [
        {
            "type": "text",
            "text": "Found 3 companies matching 'Musterfirma':\n\n1. FN 123456a - Musterfirma GmbH (Wien) - AKTIV\n2. FN 234567b - Musterfirma KG (Graz) - AKTIV\n3. FN 345678c - Musterfirma OG (Linz) - GELÖSCHT"
        }
    ]
}
```

#### fb-extract

```json
{
    "name": "fb-extract",
    "description": "Get detailed company extract from Firmenbuch",
    "inputSchema": {
        "type": "object",
        "properties": {
            "fn": {
                "type": "string",
                "description": "Firmenbuchnummer (e.g., FN123456a)"
            }
        },
        "required": ["fn"]
    }
}
```

### 3.4 E-Rechnung Tools

#### erechnung-create

```json
{
    "name": "erechnung-create",
    "description": "Create an electronic invoice (XRechnung/ZUGFeRD)",
    "inputSchema": {
        "type": "object",
        "properties": {
            "format": {
                "type": "string",
                "enum": ["xrechnung", "zugferd"],
                "description": "Output format"
            },
            "invoice_number": {
                "type": "string"
            },
            "seller_name": {
                "type": "string"
            },
            "seller_uid": {
                "type": "string"
            },
            "buyer_name": {
                "type": "string"
            },
            "buyer_reference": {
                "type": "string",
                "description": "Leitweg-ID for public authorities"
            },
            "lines": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "description": { "type": "string" },
                        "quantity": { "type": "number" },
                        "unit_price": { "type": "number" },
                        "tax_percent": { "type": "number" }
                    }
                }
            },
            "output_path": {
                "type": "string"
            }
        },
        "required": ["invoice_number", "seller_name", "buyer_name", "lines"]
    }
}
```

#### erechnung-validate

```json
{
    "name": "erechnung-validate",
    "description": "Validate an electronic invoice against EN16931",
    "inputSchema": {
        "type": "object",
        "properties": {
            "file_path": {
                "type": "string",
                "description": "Path to invoice XML or PDF"
            }
        },
        "required": ["file_path"]
    }
}
```

### 3.5 SEPA Tools

#### sepa-pain001

```json
{
    "name": "sepa-pain001",
    "description": "Generate a SEPA credit transfer file (pain.001)",
    "inputSchema": {
        "type": "object",
        "properties": {
            "debtor_name": {
                "type": "string",
                "description": "Sender name"
            },
            "debtor_iban": {
                "type": "string",
                "description": "Sender IBAN"
            },
            "execution_date": {
                "type": "string",
                "format": "date"
            },
            "transactions": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "creditor_name": { "type": "string" },
                        "creditor_iban": { "type": "string" },
                        "amount": { "type": "number" },
                        "reference": { "type": "string" }
                    },
                    "required": ["creditor_name", "creditor_iban", "amount"]
                }
            },
            "output_path": {
                "type": "string"
            }
        },
        "required": ["debtor_name", "debtor_iban", "transactions"]
    }
}
```

#### sepa-validate-iban

```json
{
    "name": "sepa-validate-iban",
    "description": "Validate an IBAN and get bank information",
    "inputSchema": {
        "type": "object",
        "properties": {
            "iban": {
                "type": "string"
            }
        },
        "required": ["iban"]
    }
}
```

---

## 4. Error Handling

### 4.1 Tool Errors

```json
{
    "content": [
        {
            "type": "text",
            "text": "Error: Account 'unknown-account' not found in credential store.\n\nAvailable accounts: firma-xyz, firma-abc"
        }
    ],
    "isError": true
}
```

### 4.2 Authentication Required

When master password is needed:

```json
{
    "content": [
        {
            "type": "text",
            "text": "⚠️ Authentication required.\n\nThe credential store is locked. Please run `fo unlock` in a terminal to unlock it, then retry this operation."
        }
    ],
    "isError": true
}
```

---

## 5. Go Implementation

### 5.1 Server Structure

```go
// internal/mcp/server.go

type Server struct {
    name        string
    version     string
    tools       map[string]Tool
    credStore   *store.CredentialStore
}

type Tool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"inputSchema"`
    Handler     func(args map[string]interface{}) (*Result, error)
}

type Result struct {
    Content []Content `json:"content"`
    IsError bool      `json:"isError,omitempty"`
}

type Content struct {
    Type string `json:"type"`
    Text string `json:"text,omitempty"`
    URI  string `json:"uri,omitempty"`
}
```

### 5.2 Tool Registration

```go
func (s *Server) RegisterTools() {
    s.tools["fo-databox-list"] = Tool{
        Name:        "fo-databox-list",
        Description: "List documents in FinanzOnline Databox",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "account": map[string]interface{}{
                    "type":        "string",
                    "description": "Account name",
                },
            },
            "required": []string{"account"},
        },
        Handler: s.handleDataboxList,
    }

    s.tools["fo-uid-validate"] = Tool{
        Name:        "fo-uid-validate",
        Description: "Validate an EU VAT ID",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "uid": map[string]interface{}{
                    "type":        "string",
                    "description": "UID number (e.g., ATU12345678)",
                },
            },
            "required": []string{"uid"},
        },
        Handler: s.handleUIDValidate,
    }

    // ... more tools
}
```

### 5.3 Handler Example

```go
func (s *Server) handleUIDValidate(args map[string]interface{}) (*Result, error) {
    uid, ok := args["uid"].(string)
    if !ok {
        return nil, fmt.Errorf("uid must be a string")
    }

    // Call UID validation service
    result, err := fonws.ValidateUID(uid)
    if err != nil {
        return &Result{
            Content: []Content{{
                Type: "text",
                Text: fmt.Sprintf("Error validating UID: %v", err),
            }},
            IsError: true,
        }, nil
    }

    var text string
    if result.Valid {
        text = fmt.Sprintf("✅ UID %s is VALID\n\nCompany: %s\nAddress: %s, %s %s\nValidated at: %s",
            result.UID,
            result.CompanyName,
            result.Address.Street,
            result.Address.PostCode,
            result.Address.City,
            result.QueryTime.Format("2006-01-02 15:04:05"),
        )
    } else {
        text = fmt.Sprintf("❌ UID %s is INVALID\n\nError: %s",
            result.UID,
            result.ErrorMessage,
        )
    }

    return &Result{
        Content: []Content{{Type: "text", Text: text}},
    }, nil
}
```

---

## 6. CLI Commands

```bash
# Start MCP server (stdio)
fo mcp serve

# Start with specific capabilities
fo mcp serve --enable-elda --enable-fb

# Test tool locally
fo mcp test fo-uid-validate '{"uid": "ATU12345678"}'

# List available tools
fo mcp tools
```

---

## 7. Claude Desktop Configuration

```json
{
    "mcpServers": {
        "austrian-business-infrastructure": {
            "command": "fo",
            "args": ["mcp", "serve"],
            "env": {
                "FO_CONFIG_DIR": "/path/to/config"
            }
        }
    }
}
```

---

## 8. Security Considerations

1. **Credential Handling**: MCP server does NOT expose credentials. All operations use stored credentials referenced by account name.

2. **Master Password**: The credential store must be unlocked before MCP operations. Consider a session-based unlock mechanism.

3. **Audit Logging**: All tool invocations should be logged with timestamp and parameters (excluding sensitive data).

4. **Rate Limiting**: Apply same rate limits as CLI (e.g., UID queries per day).

5. **Input Validation**: All inputs validated against JSON Schema before processing.
