# MCP Server (AI Integration)

The Model Context Protocol (MCP) server enables AI assistants to interact with Austrian business services.

## Features

- Tool-based interface for AI models
- All core functionality exposed as MCP tools
- Secure credential handling
- Automatic session management

## Setup

### With Claude Desktop

Add to your Claude Desktop configuration (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "austrian-business": {
      "command": "path/to/austrian-business-server",
      "args": ["--mcp"],
      "env": {
        "DATABASE_URL": "postgres://...",
        "REDIS_URL": "redis://...",
        "MASTER_KEY": "..."
      }
    }
  }
}
```

### Standalone MCP Server

```bash
# Run MCP server
./bin/server --mcp --port 3333
```

## Available Tools

### Account Management

#### `list_accounts`
List all connected service accounts.

#### `test_account`
Test connection to a service account.

**Parameters:**
- `account_id` (string): Account UUID

### FinanzOnline

#### `fo_get_databox`
Retrieve FinanzOnline databox messages.

**Parameters:**
- `account_id` (string): FO account UUID
- `from_date` (string, optional): Start date
- `to_date` (string, optional): End date

#### `fo_submit_uva`
Submit UVA (VAT return).

**Parameters:**
- `account_id` (string): FO account UUID
- `period` (string): Period (YYYY-MM)
- `data` (object): UVA field values

#### `fo_validate_uid`
Validate EU VAT number.

**Parameters:**
- `uid` (string): VAT number to validate

### ELDA

#### `elda_register_employee`
Register employee (Anmeldung).

**Parameters:**
- `account_id` (string): ELDA account UUID
- `employee` (object): Employee data

#### `elda_deregister_employee`
Deregister employee (Abmeldung).

**Parameters:**
- `account_id` (string): ELDA account UUID
- `sv_nummer` (string): Social security number
- `end_date` (string): Last day of employment
- `reason` (string): Termination reason

### Firmenbuch

#### `fb_search`
Search company register.

**Parameters:**
- `query` (string, optional): Search term
- `fn` (string, optional): Firmenbuchnummer

#### `fb_get_company`
Get company details.

**Parameters:**
- `fn` (string): Firmenbuchnummer

### Invoicing

#### `create_invoice`
Create electronic invoice.

**Parameters:**
- `type` (string): `xrechnung` or `zugferd`
- `seller` (object): Seller details
- `buyer` (object): Buyer details
- `lines` (array): Invoice lines

#### `get_invoice_pdf`
Get invoice as PDF.

**Parameters:**
- `invoice_id` (string): Invoice UUID

### SEPA

#### `generate_sepa_payment`
Generate SEPA Credit Transfer file.

**Parameters:**
- `debtor` (object): Payer details
- `payments` (array): Payment instructions

#### `validate_iban`
Validate IBAN.

**Parameters:**
- `iban` (string): IBAN to validate

### Documents

#### `analyze_document`
Analyze document with AI.

**Parameters:**
- `document_id` (string): Document UUID
- `analysis_type` (string): `classify`, `extract`, `summarize`

## Example Usage

When integrated with an AI assistant, you can use natural language:

- "Check my FinanzOnline databox for new messages"
- "Submit my January UVA with these values..."
- "Register a new employee starting next Monday"
- "Look up company FN 123456a in the Firmenbuch"
- "Create an invoice for Customer AG"
- "Validate this IBAN: AT61..."

The AI will automatically use the appropriate MCP tools.

## Security

- All credentials remain server-side
- AI never sees raw passwords/PINs
- Operations are logged for audit
- Rate limiting applies