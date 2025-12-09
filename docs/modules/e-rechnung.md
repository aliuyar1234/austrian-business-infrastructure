# E-Rechnung (Electronic Invoicing)

Generate EN16931-compliant electronic invoices in XRechnung and ZUGFeRD formats.

## Features

- XRechnung XML generation (for B2G)
- ZUGFeRD PDF/A-3 with embedded XML
- EN16931 compliance validation
- PDF generation with Austrian layout

## Supported Formats

| Format | Use Case | Output |
|--------|----------|--------|
| XRechnung | B2G (public sector) | XML |
| ZUGFeRD Basic | Simple B2B | PDF + XML |
| ZUGFeRD Comfort | Standard B2B | PDF + XML |
| ZUGFeRD Extended | Complex B2B | PDF + XML |

## Usage

### Create Invoice

```bash
curl -X POST http://localhost:8080/api/v1/invoices \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "xrechnung",
    "invoice_number": "RE-2024-001",
    "issue_date": "2024-01-15",
    "due_date": "2024-02-15",
    "seller": {
      "name": "Meine Firma GmbH",
      "street": "Musterstraße 1",
      "city": "Wien",
      "postal_code": "1010",
      "country": "AT",
      "uid": "ATU12345678",
      "iban": "AT123456789012345678",
      "bic": "BKAUATWW"
    },
    "buyer": {
      "name": "Kunde AG",
      "street": "Kundenstraße 5",
      "city": "Graz",
      "postal_code": "8010",
      "country": "AT",
      "uid": "ATU87654321"
    },
    "lines": [
      {
        "description": "Beratungsleistung Jänner 2024",
        "quantity": 10,
        "unit": "HUR",
        "unit_price": 150.00,
        "vat_rate": 20,
        "vat_category": "S"
      },
      {
        "description": "Softwarelizenz",
        "quantity": 1,
        "unit": "C62",
        "unit_price": 500.00,
        "vat_rate": 20,
        "vat_category": "S"
      }
    ],
    "payment_terms": "Zahlbar innerhalb 30 Tagen",
    "note": "Vielen Dank für Ihren Auftrag"
  }'
```

**Response:**
```json
{
  "id": "uuid",
  "invoice_number": "RE-2024-001",
  "status": "draft",
  "totals": {
    "net_amount": 2000.00,
    "vat_amount": 400.00,
    "gross_amount": 2400.00
  }
}
```

### Download Formats

```bash
# XML (XRechnung/ZUGFeRD)
curl "http://localhost:8080/api/v1/invoices/{id}/xml" \
  -H "Authorization: Bearer $TOKEN" \
  -o invoice.xml

# PDF
curl "http://localhost:8080/api/v1/invoices/{id}/pdf" \
  -H "Authorization: Bearer $TOKEN" \
  -o invoice.pdf

# ZUGFeRD (PDF with embedded XML)
curl "http://localhost:8080/api/v1/invoices/{id}/zugferd" \
  -H "Authorization: Bearer $TOKEN" \
  -o invoice-zugferd.pdf
```

### Validate Invoice

```bash
curl -X POST http://localhost:8080/api/v1/invoices/{id}/validate \
  -H "Authorization: Bearer $TOKEN"
```

**Response:**
```json
{
  "valid": true,
  "schema_valid": true,
  "schematron_valid": true,
  "warnings": [],
  "errors": []
}
```

## VAT Categories

| Code | Description | Rate |
|------|-------------|------|
| `S` | Standard rate | 20% |
| `AA` | Reduced rate | 10% |
| `Z` | Zero rated | 0% |
| `E` | Exempt | 0% |
| `AE` | Reverse charge | 0% |
| `K` | Intra-community | 0% |
| `G` | Export | 0% |

## Unit Codes (UN/ECE Rec 20)

| Code | Description |
|------|-------------|
| `C62` | One (piece) |
| `HUR` | Hour |
| `DAY` | Day |
| `MON` | Month |
| `KGM` | Kilogram |
| `MTR` | Meter |
| `LTR` | Liter |

## B2G Requirements

For Austrian public sector (Bund):
- Use XRechnung format
- Include Leitweg-ID in buyer reference
- Submit via USP (Unternehmensserviceportal)

```json
{
  "type": "xrechnung",
  "buyer_reference": "991-12345-67",
  ...
}
```

## Error Handling

| Code | Meaning |
|------|---------|
| `INV_VALIDATION_FAILED` | EN16931 validation error |
| `INV_INVALID_UID` | Invalid UID number |
| `INV_MISSING_FIELD` | Required field missing |
