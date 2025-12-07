# Austrian Business Infrastructure CLI

A Go CLI toolkit for Austrian government and business API integrations.

## Modules

| Module | Description |
|--------|-------------|
| **FinanzOnline** | Session management, databox access, UVA/ZM tax submission |
| **ELDA** | Employee registration/deregistration (Anmeldung/Abmeldung) |
| **Firmenbuch** | Company registry search, extract, watchlist monitoring |
| **E-Rechnung** | XRechnung/ZUGFeRD invoice generation (EN16931 compliant) |
| **SEPA** | pain.001/pain.008 generation, camt.053 parsing, IBAN/BIC validation |

## Requirements

- Go 1.23+
- FinanzOnline WebService credentials
- ELDA credentials (for social insurance)
- Firmenbuch API key (for company registry)

## Build

```bash
go build -o fo.exe ./cmd/fo
```

## Quick Start

### Account Management

```bash
# Add FinanzOnline account
fo account add --name "Muster GmbH" --tid 123456789 --benid USER01

# Add ELDA account
fo account add --type elda --name "My Company" --elda-serial "123456" --bknr "1234567890"

# List accounts
fo account list
```

### FinanzOnline

```bash
# Login and check databox
fo session login "Muster GmbH"
fo databox list "Muster GmbH"

# Submit UVA (VAT advance return)
fo uva submit --input uva.json --account "Muster GmbH"

# Submit ZM (summary declaration)
fo zm submit --input zm.json --account "Muster GmbH"
```

### ELDA (Social Insurance)

```bash
# Register employee
fo elda anmeldung --employee-file employee.json --account "My Company"

# Deregister employee
fo elda abmeldung --sv-nummer 1234010190 --end-date 2025-12-31 --account "My Company"
```

### Firmenbuch (Company Registry)

```bash
# Search companies
fo fb search "Muster GmbH" --limit 10

# Get company details
fo fb extract FN123456b --json

# Watchlist
fo fb watch add FN123456b
fo fb watch list
```

### E-Rechnung (E-Invoice)

```bash
# Generate XRechnung
fo erechnung create --input invoice.json --format xrechnung --output invoice.xml

# Validate invoice
fo erechnung validate invoice.xml
```

### SEPA

```bash
# Generate credit transfer (pain.001)
fo sepa pain001 payments.csv --debtor-name "Payer GmbH" --debtor-iban AT611904300234573201 -o payments.xml

# Parse bank statement (camt.053)
fo sepa camt053 statement.xml --json

# Validate IBAN
fo sepa validate AT611904300234573201

# BIC lookup
fo sepa bic 19043
```

### Dashboard

```bash
# Check all accounts
fo dashboard --all

# Check specific services
fo dashboard --services fo,elda
```

### MCP Server (AI Integration)

```bash
# Start MCP server
fo mcp serve --stdio
```

Claude Desktop config (`claude_desktop_config.json`):
```json
{
  "mcpServers": {
    "austrian-business": {
      "command": "fo.exe",
      "args": ["mcp", "serve", "--stdio"]
    }
  }
}
```

## Global Flags

| Flag | Description |
|------|-------------|
| `--config, -c` | Custom config directory |
| `--json, -j` | Output in JSON format |
| `--verbose, -v` | Enable debug logging |

## Testing

```bash
go test ./...
```

## Project Structure

```
cmd/fo/          # CLI entry point
internal/
  cli/           # Cobra commands
  config/        # Configuration paths
  elda/          # ELDA client & types
  erechnung/     # E-invoice generation
  fb/            # Firmenbuch client
  fonws/         # FinanzOnline WebService
  mcp/           # MCP server
  sepa/          # SEPA payment files
  store/         # Encrypted credential storage
tests/
  unit/          # Unit tests
  integration/   # Integration tests
```

## License

MIT
