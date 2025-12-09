# ELDA Integration

ELDA (Elektronischer Datenaustausch mit den österreichischen Sozialversicherungsträgern) handles employee registration with Austrian social insurance carriers.

## Features

- Employee registration (Anmeldung)
- Employee deregistration (Abmeldung)
- Change notifications (Änderungsmeldung)
- L16 annual wage reports
- mBGM monthly contribution statements
- ELDA databox access

## Prerequisites

You need:
- ELDA client certificate (.p12 file)
- Certificate password
- Employer account number (Dienstgeberkontonummer)

## Setup

### 1. Create an Account

```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "elda",
    "name": "Main ELDA Account",
    "credentials": {
      "certificate_base64": "MIIx...",
      "certificate_password": "your-password",
      "dienstgeber_kontonummer": "123456789"
    }
  }'
```

### 2. Test Connection

```bash
curl -X POST http://localhost:8080/api/v1/accounts/{id}/test \
  -H "Authorization: Bearer $TOKEN"
```

## Usage

### Employee Registration (Anmeldung)

Register a new employee:

```bash
curl -X POST http://localhost:8080/api/v1/elda/anmeldung \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "employee": {
      "sv_nummer": "1234010188",
      "vorname": "Max",
      "nachname": "Mustermann",
      "geburtsdatum": "1988-01-01",
      "geschlecht": "M",
      "staatsbuergerschaft": "AT",
      "eintrittsdatum": "2024-01-15",
      "beschaeftigungsart": "vollzeit",
      "woechentliche_arbeitszeit": 38.5,
      "bruttolohn": 3500.00,
      "taetigkeit": "Software Developer"
    }
  }'
```

### Employee Deregistration (Abmeldung)

```bash
curl -X POST http://localhost:8080/api/v1/elda/abmeldung \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "employee": {
      "sv_nummer": "1234010188",
      "austrittsdatum": "2024-06-30",
      "abmeldegrund": "kuendigung_dn"
    }
  }'
```

### Change Notification (Änderungsmeldung)

```bash
curl -X POST http://localhost:8080/api/v1/elda/aenderung \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "employee": {
      "sv_nummer": "1234010188",
      "aenderungsdatum": "2024-03-01",
      "neuer_bruttolohn": 3800.00
    }
  }'
```

### L16 Annual Report

Generate and submit L16 (Lohnzettel):

```bash
# Generate L16
curl -X POST http://localhost:8080/api/v1/elda/l16 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "year": 2024,
    "employee": {
      "sv_nummer": "1234010188",
      "bruttobezuege": 45000.00,
      "sv_beitraege_dn": 8100.00,
      "lohnsteuer": 9500.00
    }
  }'

# Submit
curl -X POST http://localhost:8080/api/v1/elda/l16/{id}/submit \
  -H "Authorization: Bearer $TOKEN"
```

### mBGM Monthly Statement

```bash
curl -X POST http://localhost:8080/api/v1/elda/mbgm \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "{id}",
    "period": "2024-01",
    "employees": [
      {
        "sv_nummer": "1234010188",
        "beitragsgrundlage": 3500.00,
        "beitragstage": 31
      }
    ]
  }'
```

### ELDA Databox

```bash
# List messages
curl "http://localhost:8080/api/v1/elda/databox?account_id={id}" \
  -H "Authorization: Bearer $TOKEN"

# Download document
curl "http://localhost:8080/api/v1/elda/databox/{msg_id}/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o document.pdf
```

## Validation

The system validates:
- SV-Nummer checksum
- Date plausibility
- Required fields per message type
- Contribution calculation

## Error Handling

| Code | Meaning | Solution |
|------|---------|----------|
| `ELDA_CERT_INVALID` | Certificate error | Check certificate |
| `ELDA_SV_INVALID` | Invalid SV number | Verify SV-Nummer |
| `ELDA_DUPLICATE` | Already registered | Check existing records |
| `ELDA_TIMEOUT` | Service timeout | Retry later |

## Deadlines

The system tracks important deadlines:
- Anmeldung: Before employment start (7 days grace)
- Abmeldung: Within 7 days of end
- L16: End of February following year
- mBGM: 15th of following month
