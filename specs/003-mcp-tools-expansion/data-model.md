# Data Model: MCP Tools Expansion

**Feature**: 003-mcp-tools-expansion
**Date**: 2025-12-07

## Overview

This feature adds no new data models. All required types exist in the codebase. This document maps existing types to MCP tool responses.

## MCP Response Mapping

### 1. fo-databox-list Response

**Source**: `internal/fonws/databox.go:DataboxEntry`

```
MCP Response Field     → Source Field         → Go Type
─────────────────────────────────────────────────────────
document_id            → Applkey              → string
description            → Filebez              → string
date                   → TsZust               → string (ISO format)
type_code              → Erlession            → string
type_name              → TypeName()           → string
action_required        → ActionRequired()     → bool
```

### 2. fo-databox-download Response

**Source**: `internal/fonws/databox.go:DataboxDownload`

```
MCP Response Field     → Source              → Go Type
─────────────────────────────────────────────────────────
document_id            → applkey param       → string
filename               → resp.Filename       → string
content_base64         → base64(content)     → string
content_length         → len(content)        → int
```

### 3. fo-fb-search Response

**Source**: `internal/fb/types.go:FBSearchResponse`, `FBSearchResult`

```
MCP Response Field     → Source Field         → Go Type
─────────────────────────────────────────────────────────
total_count            → TotalCount           → int
results[]              → Results              → []FBSearchResult
  .fn                  → FN                   → string
  .company_name        → Firma                → string
  .legal_form          → Rechtsform           → Rechtsform (string)
  .location            → Sitz                 → string
  .status              → Status               → FBStatus (string)
```

### 4. fo-fb-extract Response

**Source**: `internal/fb/types.go:FBExtract`, `FBPerson`, `FBGesellschafter`

```
MCP Response Field     → Source Field             → Go Type
─────────────────────────────────────────────────────────
fn                     → FN                       → string
company_name           → Firma                    → string
legal_form             → Rechtsform               → string
location               → Sitz                     → string
address                → Adresse                  → FBAdresse
  .street              → .Strasse                 → string
  .postal_code         → .PLZ                     → string
  .city                → .Ort                     → string
  .country             → .Land                    → string
share_capital_cents    → Stammkapital             → int64
share_capital_eur      → StammkapitalEUR()        → float64
currency               → Waehrung                 → string
status                 → Status                   → string
founding_date          → GruendungsdatumString()  → string
last_change            → LetzteAenderungString()  → string
business_purpose       → Gegenstand               → string
vat_id                 → UID                      → string
directors[]            → Geschaeftsfuehrer        → []FBPerson
  .first_name          → .Vorname                 → string
  .last_name           → .Nachname                → string
  .function            → .Funktion                → string
  .representation      → .VertretungsArt          → string
  .since               → .SeitString()            → string
shareholders[]         → Gesellschafter           → []FBGesellschafter
  .name                → .Name                    → string
  .fn                  → .FN                      → string
  .share_percent       → .AnteilProzent()         → float64
  .contribution_eur    → .StammeinlageEUR()       → float64
```

### 5. fo-uva-submit Response

**Source**: `internal/fonws/uva.go:FileUploadResponse`

```
MCP Response Field     → Source Field         → Go Type
─────────────────────────────────────────────────────────
success                → RC == 0              → bool
reference              → Belegnummer          → string
message                → Msg                  → string
period                 → calculated           → string (e.g., "01/2025")
submitted_at           → time.Now()           → string (ISO format)
```

## Error Response Structure

All tools use consistent error structure:

```
MCP Error Field        → Description
─────────────────────────────────────────────────────────
error                  → true
error_type             → "validation" | "authentication" | "service" | "not_found"
message                → Human-readable error message
details                → Optional: field-level errors for validation
```

## Existing Type Locations

| Type | Package | File | Line |
|------|---------|------|------|
| DataboxEntry | fonws | databox.go | 36-42 |
| DataboxDownload | fonws | databox.go | 84-87 |
| DataboxService | fonws | databox.go | 90-92 |
| FBSearchRequest | fb | types.go | 79-85 |
| FBSearchResult | fb | types.go | 88-94 |
| FBSearchResponse | fb | types.go | 97-101 |
| FBExtract | fb | types.go | 104-120 |
| FBPerson | fb | types.go | 137-146 |
| FBGesellschafter | fb | types.go | 163-171 |
| UVA | fonws | uva.go | 35-61 |
| FileUploadResponse | fonws | uva.go | 222-227 |

## No New Types Required

This feature maps existing types to JSON responses. No new Go types are needed.
