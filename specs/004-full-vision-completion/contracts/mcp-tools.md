# MCP Tools Contract

**Feature Branch**: `004-full-vision-completion`
**Date**: 2025-12-07

---

## Overview

All MCP tools follow JSON-RPC 2.0 via stdio transport. Tools return structured JSON with consistent response format.

---

## Existing Tools (from Spec 003)

| Tool | Purpose |
|------|---------|
| `fo-uid-validate` | Validate EU VAT numbers |
| `fo-iban-validate` | IBAN validation |
| `fo-bic-lookup` | Austrian bank code lookup |
| `fo-sv-nummer-validate` | Social security number validation |
| `fo-fn-validate` | Firmenbuch number validation |
| `fo-databox-list` | List FinanzOnline documents |
| `fo-databox-download` | Download specific document |
| `fo-fb-search` | Search companies |
| `fo-fb-extract` | Get company details |
| `fo-uva-submit` | Submit VAT return |

---

## New Tools (Spec 004)

### fo-elda-register
Submit employee registration to ELDA.

**Input**:
```json
{
  "sv_nummer": "1234010190",
  "first_name": "Max",
  "last_name": "Mustermann",
  "start_date": "2025-01-15",
  "employer_vsnr": "12345678"
}
```

**Output**:
```json
{
  "success": true,
  "confirmation_number": "ELDA-2025-12345678"
}
```

---

### fo-elda-status
Query ELDA registration status.

**Input**:
```json
{
  "sv_nummer": "1234010190"
}
```

**Output**:
```json
{
  "success": true,
  "status": "registered",
  "registered_at": "2025-01-15T10:00:00Z"
}
```

---

### fo-zm-submit
Submit ZM (summary declaration).

**Input**:
```json
{
  "account": "Muster GmbH",
  "year": 2025,
  "quarter": 1,
  "entries": [
    {"partner_uid": "DE123456789", "type": "L", "amount": 50000}
  ]
}
```

---

### fo-invoice-create
Generate EN16931 invoice.

**Input**:
```json
{
  "format": "xrechnung",
  "invoice": {
    "id": "INV-2025-001",
    "seller": {"name": "...", "uid": "ATU..."},
    "buyer": {"name": "...", "uid": "ATU..."},
    "lines": [...]
  }
}
```

**Output**:
```json
{
  "success": true,
  "xml": "<?xml version=\"1.0\"...>",
  "format": "xrechnung"
}
```

---

### fo-invoice-validate
Validate invoice against EN16931.

**Input**:
```json
{
  "xml": "<?xml version=\"1.0\"..."
}
```

**Output**:
```json
{
  "valid": true,
  "format": "xrechnung",
  "errors": [],
  "warnings": []
}
```

---

### fo-sepa-generate
Generate SEPA payment file.

**Input**:
```json
{
  "type": "pain001",
  "version": "09",
  "debtor": {"name": "...", "iban": "AT..."},
  "payments": [...]
}
```

---

## Response Format

All tools return:

```json
{
  "success": true|false,
  "data": {...},
  "error": "message if failed",
  "error_code": "CODE if failed"
}
```
