# Quickstart Guide: Full Vision Completion

**Feature Branch**: `004-full-vision-completion`
**Date**: 2025-12-07

---

## Prerequisites

- Go 1.23+
- FinanzOnline WebService credentials (existing)
- ELDA credentials (for social insurance features)
- Firmenbuch API key (for company registry features)

---

## Build

```bash
cd austrian-business-infrastructure
go build -o fo.exe ./cmd/fo
```

---

## Module Quick Reference

### 1. ELDA (Social Insurance)

**Setup**: Add ELDA account
```bash
fo account add --type elda --name "My Company" \
  --elda-serial "123456" --bknr "1234567890"
```

**Register employee**:
```bash
# Create employee file
cat > employee.json << 'EOF'
{
  "sv_nummer": "1234010190",
  "first_name": "Max",
  "last_name": "Mustermann",
  "date_of_birth": "1990-01-01",
  "employer_vsnr": "12345678",
  "start_date": "2025-01-15",
  "insurance_type": "A1"
}
EOF

# Submit registration
fo elda anmeldung --employee-file employee.json --account "My Company"
```

---

### 2. Firmenbuch (Company Registry)

**Setup**: Add Firmenbuch account
```bash
fo account add --type firmenbuch --name "FB Access" \
  --fb-api-key "your-api-key"
```

**Search companies**:
```bash
fo fb search "Muster GmbH" --limit 10
```

**Get company details**:
```bash
fo fb extract FN123456b --json
```

---

### 3. E-Rechnung (E-Invoice)

**Create XRechnung**:
```bash
# Create invoice file
cat > invoice.json << 'EOF'
{
  "id": "INV-2025-001",
  "issue_date": "2025-01-15",
  "seller": {
    "name": "Seller GmbH",
    "uid": "ATU12345678",
    "address": {"street": "Musterstr. 1", "postal_code": "1010", "city": "Wien", "country": "AT"}
  },
  "buyer": {
    "name": "Buyer AG",
    "uid": "ATU87654321",
    "address": {"street": "Käuferstr. 2", "postal_code": "1020", "city": "Wien", "country": "AT"}
  },
  "lines": [
    {"description": "Consulting", "quantity": 10, "unit": "HUR", "unit_price": 15000, "tax_rate": 20}
  ]
}
EOF

# Generate XRechnung XML
fo erechnung create --input invoice.json --format xrechnung --output invoice.xml
```

**Validate invoice**:
```bash
fo erechnung validate invoice.xml
```

---

### 4. SEPA (Banking)

**Generate credit transfer (pain.001)**:
```bash
cat > payments.json << 'EOF'
{
  "message_id": "MSG-2025-001",
  "debtor": {"name": "Payer GmbH", "iban": "AT611904300234573201"},
  "execution_date": "2025-01-20",
  "payments": [
    {
      "end_to_end_id": "PAY-001",
      "amount": 100000,
      "creditor": {"name": "Payee AG", "iban": "AT021100000012345678"},
      "reference": "Invoice INV-2025-001"
    }
  ]
}
EOF

fo sepa pain001 --input payments.json --output payments.xml
```

**Parse bank statement**:
```bash
fo sepa parse statement.xml --json
```

**Validate IBAN**:
```bash
fo sepa validate AT611904300234573201
```

---

### 5. UVA/ZM Submission

**Submit UVA**:
```bash
cat > uva.json << 'EOF'
{
  "year": 2025,
  "period": 1,
  "is_quarterly": false,
  "kennzahlen": {
    "KZ000": 10000000,
    "KZ017": 8000000,
    "KZ060": 1600000
  }
}
EOF

fo uva submit --input uva.json --account "Muster GmbH"
```

**Submit ZM**:
```bash
cat > zm.json << 'EOF'
{
  "year": 2025,
  "quarter": 1,
  "entries": [
    {"partner_uid": "DE123456789", "country_code": "DE", "transaction_type": "L", "amount": 50000}
  ]
}
EOF

fo zm submit --input zm.json --account "Muster GmbH"
```

---

### 6. Dashboard

**Check all accounts**:
```bash
fo dashboard --all
```

**Check specific services**:
```bash
fo dashboard --services fo,elda
```

---

### 7. MCP Server (AI Integration)

**Start server**:
```bash
fo mcp serve --stdio
```

**Claude Desktop config** (`claude_desktop_config.json`):
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

---

## Common Workflows

### Multi-Account UVA Check
```bash
# Check all FinanzOnline accounts for pending documents
fo dashboard --services fo --json | jq '.accounts[] | select(.pending_items > 0)'
```

### Batch Employee Registration
```bash
# Register multiple employees from directory
for f in employees/*.json; do
  fo elda anmeldung --employee-file "$f" --account "My Company"
done
```

### Due Diligence Report
```bash
# Get company info and check for insolvency
fo fb extract FN123456b --json | jq '{name, insolvency_status, directors}'
```

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `FO_CONFIG_DIR` | Custom config directory |
| `FO_MASTER_PASSWORD` | Master password (avoid, use prompt) |
| `FO_JSON_OUTPUT` | Default to JSON output |
| `FO_VERBOSE` | Enable debug logging |

---

## Next Steps

1. Run `/speckit.tasks` to generate implementation tasks
2. Implement modules in priority order: P1 → P2 → P3
3. Run tests: `go test ./...`
