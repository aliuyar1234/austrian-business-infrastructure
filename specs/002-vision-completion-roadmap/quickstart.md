# Quickstart: Austrian Business Infrastructure - Complete Product Suite

**Feature**: 002-vision-completion-roadmap
**Date**: 2025-12-07

This guide walks through common workflows for each module.

---

## Prerequisites

Before starting, ensure:
1. `fo` CLI is installed and in PATH
2. Master password is set up: `fo account add <name>`
3. FinanzOnline WebService user exists (for UVA/UID/ZM)

---

## 1. FinanzOnline Extensions

### 1.1 UID Validation

**Single UID Check**:
```bash
# Check if a UID number is valid
fo uid check ATU12345678

# Output:
# ✅ Valid
# Company: Musterfirma GmbH
# Address: Musterstraße 1, 1010 Wien
```

**Batch Validation**:
```bash
# Prepare CSV file (uids.csv):
# uid
# ATU12345678
# DE123456789
# IT12345678901

fo uid batch uids.csv --output results.csv

# Results CSV contains: uid, valid, company_name, address, error
```

**JSON Output**:
```bash
fo uid check ATU12345678 --json
```

### 1.2 UVA Submission

**Validate UVA before submission**:
```bash
# From XML file
fo uva validate uva-january.xml

# From command line values
fo uva validate --year 2025 --month 1 --kz017 80000 --kz060 16000
```

**Submit UVA**:
```bash
# Using stored account
fo uva submit firma-xyz --file uva-january.xml

# Or directly with values
fo uva submit firma-xyz \
    --year 2025 \
    --month 1 \
    --kz017 80000 \
    --kz060 16000

# Output:
# ✅ UVA submitted successfully
# Reference: FON-2025-12345678
```

**Check Submission Status**:
```bash
fo uva status FON-2025-12345678 --account firma-xyz
```

### 1.3 Zusammenfassende Meldung (ZM)

```bash
# Create and submit ZM
fo zm submit firma-xyz \
    --year 2025 \
    --quarter 1 \
    --entry DE123456789:L:50000 \
    --entry IT12345678901:S:25000

# L = Lieferungen (goods)
# S = Sonstige Leistungen (services)
# D = Dreiecksgeschäfte (triangular)
```

---

## 2. ELDA - Social Security

### 2.1 Setup ELDA Account

```bash
# Add ELDA credentials (separate from FinanzOnline)
fo account add mycompany-elda --type elda

# Prompts for:
# - Dienstgeber-Nr (8 digits)
# - Benutzer-Nr
# - PIN
```

### 2.2 Register Employee

```bash
# Register new employee
elda anmeldung --account mycompany-elda \
    --svnummer 1234150185 \
    --vorname Max \
    --nachname Mustermann \
    --geburtsdatum 1985-01-15 \
    --eintrittsdatum 2025-02-01 \
    --brutto 3500 \
    --art vollzeit

# Output:
# ✅ Anmeldung submitted
# Reference: ELDA-2025-AN-12345678
```

### 2.3 Check Status

```bash
elda status ELDA-2025-AN-12345678 --account mycompany-elda

# Output:
# Status: VERARBEITET
# Submitted: 2025-01-15 14:30:00
# Processed: 2025-01-15 15:00:00
# Result: ANGENOMMEN
```

### 2.4 Deregister Employee

```bash
elda abmeldung --account mycompany-elda \
    --svnummer 1234150185 \
    --austrittsdatum 2025-06-30 \
    --grund kuendigung
```

---

## 3. Firmenbuch - Company Register

### 3.1 Setup API Key

```bash
# Add Firmenbuch credentials
fo account add fb-default --type firmenbuch

# Prompts for API key from api.auszug.at or similar provider
```

### 3.2 Search Companies

```bash
# Search by name (fuzzy)
fb search "Musterfirma"

# Output:
# FN 123456a - Musterfirma GmbH (Wien) - AKTIV
# FN 234567b - Musterfirma KG (Graz) - AKTIV

# Search with filters
fb search "Musterfirma" --rechtsform GmbH --ort Wien
```

### 3.3 Get Company Extract

```bash
fb extract FN123456a

# Output:
# Musterfirma GmbH (FN 123456a)
# ═══════════════════════════════════════
# Rechtsform: GmbH
# Sitz: Wien
# Adresse: Musterstraße 1, 1010 Wien
# Stammkapital: 35.000 EUR
#
# Geschäftsführer:
#   - Max Mustermann (seit 2010-06-01, selbständig)
#
# Gesellschafter:
#   - Max Mustermann: 70% (24.500 EUR)
#   - Holding GmbH (FN 654321b): 30% (10.500 EUR)

# JSON output
fb extract FN123456a --json --output musterfirma.json
```

### 3.4 Monitor Changes

```bash
# Add to watchlist
fb watch add FN123456a
fb watch add FN234567b

# Check for changes
fb watch check

# Output:
# FN 123456a - Musterfirma GmbH
#   ⚠️ Change detected: 2025-01-10
#   - Geschäftsführer-Wechsel: Maria Musterfrau bestellt
#
# FN 234567b - Musterfirma KG
#   No changes since last check
```

---

## 4. E-Rechnung - Electronic Invoicing

### 4.1 Create Invoice

**Prepare invoice data (invoice.json)**:
```json
{
    "invoice_number": "INV-2025-001234",
    "issue_date": "2025-01-15",
    "due_date": "2025-02-14",
    "buyer_reference": "04011000-12345-67",

    "seller": {
        "name": "Lieferant GmbH",
        "street": "Lieferantenstraße 1",
        "city": "Wien",
        "post_code": "1010",
        "country": "AT",
        "tax_id": "ATU12345678"
    },

    "buyer": {
        "name": "Kunde AG",
        "street": "Kundenweg 5",
        "city": "Graz",
        "post_code": "8010",
        "country": "AT"
    },

    "payment": {
        "iban": "AT611904300234573201"
    },

    "lines": [
        {
            "description": "Consulting Service",
            "quantity": 10,
            "unit": "C62",
            "unit_price": 100.00,
            "tax_category": "S",
            "tax_percent": 20
        }
    ]
}
```

**Generate Invoice**:
```bash
# XRechnung (XML only)
erechnung create invoice.json --format xrechnung --output invoice.xml

# ZUGFeRD (PDF with embedded XML)
erechnung create invoice.json --format zugferd --output invoice.pdf
```

### 4.2 Validate Invoice

```bash
erechnung validate invoice.xml

# Output:
# ✅ Invoice is valid (EN16931 compliant)
# Warnings: 0
# Errors: 0
```

### 4.3 Extract from PDF

```bash
erechnung extract invoice.pdf --output extracted.json
```

---

## 5. SEPA - Banking

### 5.1 Create Credit Transfer

**Prepare payments (payments.csv)**:
```csv
end_to_end_id,creditor_name,creditor_iban,amount,currency,remittance_info
E2E-001,Empfänger 1 GmbH,AT021100000012345678,1000.00,EUR,Rechnung RE-2025-001
E2E-002,Empfänger 2 AG,AT301200000098765432,500.00,EUR,Rechnung RE-2025-002
```

**Generate pain.001**:
```bash
sepa pain001 payments.csv \
    --debtor-name "Auftraggeber GmbH" \
    --debtor-iban AT611904300234573201 \
    --execution-date 2025-01-20 \
    --output payments.xml

# Output:
# ✅ Created payments.xml
# Transactions: 2
# Total: 1.500,00 EUR
```

### 5.2 Validate IBAN

```bash
sepa validate AT611904300234573201

# Output:
# ✅ Valid IBAN
# Country: AT (Austria)
# Bank Code: 19043
# BIC: BKAUATWW
# Bank: Bank Austria
```

### 5.3 Parse Account Statement

```bash
sepa camt053 statement.xml

# Output:
# Statement: 2025-01-15
# Account: AT611904300234573201
# Opening Balance: 10.000,00 EUR
# Closing Balance: 9.500,00 EUR
#
# Transactions:
#   2025-01-15  -500,00 EUR  Empfänger GmbH  Rechnung RE-2025-001

# Export to CSV
sepa camt053 statement.xml --output transactions.csv
```

---

## 6. MCP Server - AI Integration

### 6.1 Start MCP Server

```bash
# Start for Claude Desktop
fo mcp serve
```

### 6.2 Configure Claude Desktop

Add to Claude Desktop settings (`claude_desktop_config.json`):
```json
{
    "mcpServers": {
        "austrian-business-infrastructure": {
            "command": "fo",
            "args": ["mcp", "serve"]
        }
    }
}
```

### 6.3 Use in Claude

After configuration, ask Claude:
- "Check if UID ATU12345678 is valid"
- "List databox documents for account firma-xyz"
- "Search for Musterfirma in the Firmenbuch"
- "Create a SEPA transfer for 1000 EUR to AT021100000012345678"

---

## Common Workflows

### Accountant Monthly Workflow

```bash
# 1. Check all databoxes for new documents
fo databox list --all

# 2. Download action-required documents
fo databox download firma-xyz ABC123 --output ./documents/

# 3. Validate UIDs for new clients
fo uid batch new-clients.csv --output uid-results.csv

# 4. Submit UVAs for all clients
for account in firma-xyz firma-abc; do
    fo uva submit $account --file ./uva/$account.xml
done
```

### HR New Employee Workflow

```bash
# 1. Validate SV-Nummer format
elda validate-svnummer 1234150185

# 2. Submit registration
elda anmeldung --account mycompany-elda \
    --svnummer 1234150185 \
    --vorname Max \
    --nachname Mustermann \
    --geburtsdatum 1985-01-15 \
    --eintrittsdatum 2025-02-01 \
    --brutto 3500 \
    --art vollzeit

# 3. Check status next day
elda status ELDA-2025-AN-12345678 --account mycompany-elda
```

### Due Diligence Workflow

```bash
# 1. Search for company
fb search "Target GmbH" --ort Wien

# 2. Get detailed extract
fb extract FN123456a --output target.json

# 3. Add to monitoring watchlist
fb watch add FN123456a

# 4. Validate company UID
fo uid check ATU12345678
```

---

## Troubleshooting

### Authentication Errors

```bash
# Re-enter master password
fo unlock

# List available accounts
fo account list
```

### Rate Limits

```bash
# UID query limit (2 per day per UID via FinanzOnline)
# Use VIES fallback for bulk validation
fo uid batch uids.csv --source vies
```

### API Errors

```bash
# Enable verbose logging
fo --verbose uid check ATU12345678

# Check specific error codes in documentation
fo help errors
```
