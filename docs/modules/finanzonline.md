# FinanzOnline Integration

FinanzOnline is the Austrian tax authority's online portal. This module provides programmatic access to its web services.

## Features

- Session management with automatic renewal
- Databox access (retrieve official documents)
- UVA submission (Umsatzsteuervoranmeldung)
- ZM submission (Zusammenfassende Meldung)
- UID validation

## Prerequisites

You need FinanzOnline credentials:
- **Teilnehmer-ID**: Your participant ID
- **Benutzer-ID**: Your user ID
- **PIN**: Your personal PIN

These are the same credentials you use to log into finanzonline.bmf.gv.at.

## Setup

### 1. Create an Account

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "finanzonline",
    "name": "Main FO Account",
    "credentials": {
      "teilnehmer_id": "your-tid",
      "benutzer_id": "your-uid",
      "pin": "your-pin"
    }
  }'
```

### 2. Test Connection

```bash
curl -X POST http://localhost:8080/api/v1/accounts/{id}/test \
  -H "Authorization: Bearer $TOKEN"
```

## Usage

### Databox

Retrieve official documents from your FinanzOnline databox:

```bash
# List messages
curl "http://localhost:8080/api/v1/finanzonline/databox?account_id={id}" \
  -H "Authorization: Bearer $TOKEN"

# Download document
curl "http://localhost:8080/api/v1/finanzonline/databox/{msg_id}/download?account_id={id}" \
  -H "Authorization: Bearer $TOKEN" \
  -o document.pdf
```

### UVA (VAT Return)

Submit monthly/quarterly VAT returns:

```bash
# Create UVA
curl -X POST http://localhost:8080/api/v1/uva \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "period": "2024-01",
    "data": {
      "kz000": 50000.00,
      "kz022": 10000.00,
      "kz029": 1500.00
    }
  }'

# Submit to FinanzOnline
curl -X POST http://localhost:8080/api/v1/uva/{uva_id}/submit \
  -H "Authorization: Bearer $TOKEN"
```

### ZM (EC Sales List)

Report intra-community supplies:

```bash
curl -X POST http://localhost:8080/api/v1/zm \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "period": "2024-Q1",
    "entries": [
      {
        "uid": "DE123456789",
        "amount": 25000.00,
        "type": "L"
      }
    ]
  }'
```

### UID Validation

Validate EU VAT numbers:

```bash
curl "http://localhost:8080/api/v1/uid/validate?uid=ATU12345678" \
  -H "Authorization: Bearer $TOKEN"
```

## Automatic Sync

The system automatically syncs your databox every 6 hours. You can also trigger manual sync:

```bash
curl -X POST http://localhost:8080/api/v1/finanzonline/sync \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"account_id": "{id}"}'
```

## Error Handling

Common FinanzOnline errors:

| Code | Meaning | Solution |
|------|---------|----------|
| `FO_AUTH_FAILED` | Invalid credentials | Check TID/UID/PIN |
| `FO_SESSION_EXPIRED` | Session timed out | System auto-renews |
| `FO_RATE_LIMITED` | Too many requests | Wait and retry |
| `FO_SERVICE_UNAVAILABLE` | BMF maintenance | Try again later |

## Security

- Credentials are encrypted at rest (AES-256-GCM)
- Sessions are not persisted
- All API calls use TLS
- Audit logging for all operations
