# CLI Command Contracts

**Feature Branch**: `004-full-vision-completion`
**Date**: 2025-12-07

---

## Command Structure

All commands follow the pattern: `fo <module> <action> [flags]`

Global flags available on all commands:
- `--config, -c`: Custom config directory
- `--json, -j`: Output in JSON format
- `--verbose, -v`: Enable debug logging

---

## 1. ELDA Commands

### fo elda anmeldung
Register employee with social insurance.

```bash
fo elda anmeldung --employee-file <path> [--account <name>] [--test]
```

### fo elda abmeldung
Deregister employee from social insurance.

```bash
fo elda abmeldung --sv-nummer <number> --end-date <date> [--account <name>]
```

### fo elda status
Query registration status.

```bash
fo elda status --sv-nummer <number> [--account <name>]
```

---

## 2. Firmenbuch Commands

### fo fb search
Search companies by name.

```bash
fo fb search <query> [--limit <n>] [--legal-form <form>]
```

### fo fb extract
Get company details (Firmenbuchauszug).

```bash
fo fb extract <fn-nummer> [--include-history]
```

### fo fb monitor
Manage company watch list.

```bash
fo fb monitor add <fn-nummer>
fo fb monitor remove <fn-nummer>
fo fb monitor list
```

---

## 3. E-Rechnung Commands

### fo erechnung create
Generate EN16931-compliant invoice.

```bash
fo erechnung create --input <path> --format <xrechnung|zugferd|ebinterface> [--output <path>]
```

### fo erechnung validate
Validate invoice against EN16931.

```bash
fo erechnung validate <file>
```

### fo erechnung embed
Create ZUGFeRD PDF with embedded XML.

```bash
fo erechnung embed --pdf <path> --data <path> --output <path>
```

---

## 4. SEPA Commands

### fo sepa pain001
Generate credit transfer file.

```bash
fo sepa pain001 --input <path> [--output <path>] [--version 03|09]
```

### fo sepa pain008
Generate direct debit file.

```bash
fo sepa pain008 --input <path> [--output <path>] [--version 02|08]
```

### fo sepa parse
Parse bank statement (camt.053).

```bash
fo sepa parse <file>
```

### fo sepa validate
Validate IBAN.

```bash
fo sepa validate <iban>
```

---

## 5. UVA/ZM Commands

### fo uva submit
Submit VAT advance return.

```bash
fo uva submit --input <path> --account <name> [--test]
```

### fo zm submit
Submit summary declaration.

```bash
fo zm submit --input <path> --account <name> [--test]
```

---

## 6. Dashboard & MCP

### fo dashboard
Unified status across all services.

```bash
fo dashboard [--all] [--services fo,elda,fb]
```

### fo mcp serve
Start MCP server for AI integration.

```bash
fo mcp serve [--stdio]
```

---

## Error Codes

| Code | Description |
|------|-------------|
| `VALIDATION_ERROR` | Input validation failed |
| `AUTH_ERROR` | Authentication failed |
| `SESSION_EXPIRED` | Session needs refresh |
| `NETWORK_ERROR` | Connection failed |
| `API_ERROR` | Remote API returned error |
