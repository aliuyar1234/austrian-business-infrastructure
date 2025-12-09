# SEPA Payment Processing

Generate SEPA payment files and parse bank statements.

## Features

- SEPA Credit Transfer (pain.001) - outgoing payments
- SEPA Direct Debit (pain.008) - incoming payments
- Bank Statement parsing (camt.053)
- IBAN/BIC validation

## Usage

### SEPA Credit Transfer (pain.001)

Generate payment files for your bank:

```bash
curl -X POST http://localhost:8080/api/v1/sepa/pain001 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message_id": "MSG-2024-001",
    "creation_date": "2024-01-15",
    "debtor": {
      "name": "Meine Firma GmbH",
      "iban": "AT611904300234573201",
      "bic": "BKAUATWW"
    },
    "payments": [
      {
        "payment_id": "PMT-001",
        "creditor_name": "Lieferant AG",
        "creditor_iban": "DE89370400440532013000",
        "creditor_bic": "COBADEFFXXX",
        "amount": 1500.00,
        "currency": "EUR",
        "reference": "INV-2024-123",
        "execution_date": "2024-01-20"
      },
      {
        "payment_id": "PMT-002",
        "creditor_name": "Dienstleister",
        "creditor_iban": "AT483200000012345678",
        "amount": 750.50,
        "currency": "EUR",
        "reference": "Honorar Jänner"
      }
    ]
  }'
```

**Response:**
```json
{
  "id": "uuid",
  "message_id": "MSG-2024-001",
  "status": "generated",
  "total_amount": 2250.50,
  "payment_count": 2,
  "download_url": "/api/v1/sepa/pain001/{id}/download"
}
```

### SEPA Direct Debit (pain.008)

Collect payments from customers:

```bash
curl -X POST http://localhost:8080/api/v1/sepa/pain008 \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message_id": "DD-2024-001",
    "creditor": {
      "name": "Meine Firma GmbH",
      "iban": "AT611904300234573201",
      "creditor_id": "AT12ZZZ00000012345"
    },
    "debits": [
      {
        "debit_id": "DD-001",
        "debtor_name": "Kunde Mustermann",
        "debtor_iban": "AT123456789012345678",
        "amount": 99.00,
        "mandate_id": "MNDT-2023-001",
        "mandate_date": "2023-06-01",
        "reference": "Abo Jänner 2024",
        "sequence_type": "RCUR"
      }
    ]
  }'
```

### Parse Bank Statement (camt.053)

```bash
curl -X POST http://localhost:8080/api/v1/sepa/camt053/parse \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/xml" \
  --data-binary @statement.xml
```

**Response:**
```json
{
  "account": {
    "iban": "AT611904300234573201",
    "currency": "EUR"
  },
  "statement_id": "12345/2024",
  "from_date": "2024-01-01",
  "to_date": "2024-01-31",
  "opening_balance": 10000.00,
  "closing_balance": 12500.00,
  "entries": [
    {
      "amount": 1500.00,
      "credit_debit": "CRDT",
      "booking_date": "2024-01-15",
      "value_date": "2024-01-15",
      "reference": "INV-2024-100",
      "debtor_name": "Kunde AG",
      "debtor_iban": "DE89370400440532013000"
    }
  ]
}
```

### IBAN Validation

```bash
curl "http://localhost:8080/api/v1/sepa/validate/iban?iban=AT611904300234573201" \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "valid": true,
  "iban": "AT611904300234573201",
  "country": "AT",
  "checksum": "61",
  "bban": "1904300234573201",
  "bank_code": "19043",
  "account_number": "00234573201"
}
```

### BIC Lookup

```bash
curl "http://localhost:8080/api/v1/sepa/bic?iban=AT611904300234573201" \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "bic": "BKAUATWW",
  "bank_name": "UniCredit Bank Austria AG",
  "city": "Wien",
  "country": "AT"
}
```

## Sequence Types (Direct Debit)

| Code | Description |
|------|-------------|
| `FRST` | First debit of mandate |
| `RCUR` | Recurring debit |
| `OOFF` | One-off debit |
| `FNAL` | Final debit of mandate |

## File Download

```bash
# Download pain.001
curl "http://localhost:8080/api/v1/sepa/pain001/{id}/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o payment.xml

# Download pain.008
curl "http://localhost:8080/api/v1/sepa/pain008/{id}/download" \
  -H "Authorization: Bearer $TOKEN" \
  -o directdebit.xml
```

## Bank Upload

Generated files are compatible with:
- Austrian banks (ELBA, George Business, etc.)
- EBICS systems
- Most EU banking portals

## Error Handling

| Code | Meaning |
|------|---------|
| `SEPA_INVALID_IBAN` | Invalid IBAN format/checksum |
| `SEPA_INVALID_BIC` | Invalid BIC format |
| `SEPA_AMOUNT_INVALID` | Invalid amount |
| `SEPA_PARSE_ERROR` | Cannot parse camt.053 |
