# Firmenbuch Integration

Access the Austrian company register (Firmenbuch) for company lookups, extracts, and monitoring.

## Features

- Company search by name or FN
- Official extract retrieval
- Company watchlist with change notifications
- Historical data access

## Usage

### Search Companies

```bash
# Search by name
curl "http://localhost:8080/api/v1/firmenbuch/search?query=Musterfirma" \
  -H "Authorization: Bearer $TOKEN"

# Search by Firmenbuchnummer
curl "http://localhost:8080/api/v1/firmenbuch/search?fn=FN123456a" \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "results": [
    {
      "fn": "FN123456a",
      "name": "Musterfirma GmbH",
      "legal_form": "GmbH",
      "address": "Musterstraße 1, 1010 Wien",
      "status": "active"
    }
  ]
}
```

### Get Company Details

```bash
curl "http://localhost:8080/api/v1/firmenbuch/FN123456a" \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "fn": "FN123456a",
  "name": "Musterfirma GmbH",
  "legal_form": "GmbH",
  "registered_address": {
    "street": "Musterstraße 1",
    "postal_code": "1010",
    "city": "Wien"
  },
  "share_capital": 35000.00,
  "registration_date": "2020-05-15",
  "representatives": [
    {
      "name": "Max Mustermann",
      "role": "Geschäftsführer",
      "since": "2020-05-15",
      "sole_representation": true
    }
  ],
  "shareholders": [
    {
      "name": "Max Mustermann",
      "share_percentage": 100
    }
  ],
  "status": "active"
}
```

### Get Official Extract

```bash
# Order extract
curl -X POST "http://localhost:8080/api/v1/firmenbuch/FN123456a/extract" \
  -H "Authorization: Bearer $TOKEN"

# Download when ready
curl "http://localhost:8080/api/v1/firmenbuch/FN123456a/extract/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o extract.pdf
```

### Watchlist

Monitor companies for changes:

```bash
# Add to watchlist
curl -X POST http://localhost:8080/api/v1/firmenbuch/watchlist \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "fn": "FN123456a",
    "notify_email": true
  }'

# List watched companies
curl http://localhost:8080/api/v1/firmenbuch/watchlist \
  -H "Authorization: Bearer $TOKEN"

# Remove from watchlist
curl -X DELETE "http://localhost:8080/api/v1/firmenbuch/watchlist/FN123456a" \
  -H "Authorization: Bearer $TOKEN"
```

You'll receive notifications when:
- Company name changes
- Address changes
- Representatives change
- Capital changes
- Insolvency proceedings
- Company dissolution

## Legal Forms

Common Austrian legal forms:

| Code | Name |
|------|------|
| `GmbH` | Gesellschaft mit beschränkter Haftung |
| `AG` | Aktiengesellschaft |
| `KG` | Kommanditgesellschaft |
| `OG` | Offene Gesellschaft |
| `e.U.` | eingetragener Unternehmer |
| `GesbR` | Gesellschaft bürgerlichen Rechts |

## Error Handling

| Code | Meaning |
|------|---------|
| `FB_NOT_FOUND` | Company not found |
| `FB_INVALID_FN` | Invalid Firmenbuchnummer format |
| `FB_UNAVAILABLE` | Firmenbuch service unavailable |
