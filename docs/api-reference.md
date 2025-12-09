# API Reference

Base URL: `http://localhost:8080/api/v1`

All endpoints require authentication unless noted. Include the access token:
```
Authorization: Bearer <access_token>
```

## Authentication

### POST /auth/register
Register a new user account.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword",
  "name": "John Doe"
}
```

**Response:** `201 Created`
```json
{
  "user_id": "uuid",
  "email": "user@example.com"
}
```

### POST /auth/login
Authenticate and receive tokens.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepassword"
}
```

**Response:** `200 OK`
```json
{
  "access_token": "eyJ...",
  "refresh_token": "eyJ...",
  "expires_at": "2024-01-01T12:15:00Z",
  "token_type": "Bearer"
}
```

### POST /auth/refresh
Refresh access token.

**Request:**
```json
{
  "refresh_token": "eyJ..."
}
```

### POST /auth/logout
Invalidate current session.

### POST /auth/2fa/enable
Enable TOTP two-factor authentication.

**Response:**
```json
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code": "data:image/png;base64,..."
}
```

### POST /auth/2fa/verify
Verify TOTP code.

**Request:**
```json
{
  "code": "123456"
}
```

---

## Accounts

### GET /accounts
List all connected service accounts.

**Response:**
```json
{
  "accounts": [
    {
      "id": "uuid",
      "type": "finanzonline",
      "name": "My FO Account",
      "status": "active",
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### POST /accounts
Create a new service account.

**Request:**
```json
{
  "type": "finanzonline",
  "name": "My FO Account",
  "credentials": {
    "teilnehmer_id": "...",
    "benutzer_id": "...",
    "pin": "..."
  }
}
```

### GET /accounts/:id
Get account details.

### PUT /accounts/:id
Update account.

### DELETE /accounts/:id
Delete account.

### POST /accounts/:id/test
Test account connection.

---

## FinanzOnline

### GET /finanzonline/databox
List databox messages.

**Query Parameters:**
- `account_id` - Account UUID (required)
- `from` - Start date (ISO 8601)
- `to` - End date (ISO 8601)

### GET /finanzonline/databox/:id
Get specific databox message.

### GET /finanzonline/databox/:id/download
Download databox document.

---

## UVA (VAT Returns)

### GET /uva
List UVA submissions.

### POST /uva
Create/submit UVA.

**Request:**
```json
{
  "account_id": "uuid",
  "period": "2024-01",
  "data": {
    "kz000": 10000.00,
    "kz022": 2000.00
  }
}
```

### GET /uva/:id
Get UVA details.

### POST /uva/:id/submit
Submit UVA to FinanzOnline.

---

## ZM (EC Sales List)

### GET /zm
List ZM submissions.

### POST /zm
Create ZM submission.

### POST /zm/:id/submit
Submit ZM to FinanzOnline.

---

## ELDA

### GET /elda/employees
List registered employees.

### POST /elda/anmeldung
Register new employee (Anmeldung).

**Request:**
```json
{
  "account_id": "uuid",
  "employee": {
    "sv_nummer": "1234010188",
    "vorname": "Max",
    "nachname": "Mustermann",
    "geburtsdatum": "1988-01-01",
    "eintrittsdatum": "2024-01-15"
  }
}
```

### POST /elda/abmeldung
Deregister employee (Abmeldung).

### GET /elda/databox
Get ELDA databox messages.

---

## Firmenbuch

### GET /firmenbuch/search
Search company register.

**Query Parameters:**
- `query` - Search term
- `fn` - Firmenbuchnummer (exact)

### GET /firmenbuch/:fn
Get company details.

### GET /firmenbuch/:fn/extract
Get official extract.

### POST /firmenbuch/watchlist
Add company to watchlist.

### GET /firmenbuch/watchlist
List watched companies.

---

## Invoices (E-Rechnung)

### GET /invoices
List invoices.

### POST /invoices
Create invoice.

**Request:**
```json
{
  "type": "xrechnung",
  "seller": {
    "name": "My Company",
    "address": "...",
    "uid": "ATU12345678"
  },
  "buyer": {
    "name": "Customer",
    "address": "..."
  },
  "lines": [
    {
      "description": "Service",
      "quantity": 1,
      "unit_price": 100.00,
      "vat_rate": 20
    }
  ]
}
```

### GET /invoices/:id
Get invoice details.

### GET /invoices/:id/xml
Download invoice XML.

### GET /invoices/:id/pdf
Download invoice PDF.

---

## SEPA

### POST /sepa/pain001
Generate SEPA Credit Transfer (pain.001).

**Request:**
```json
{
  "debtor": {
    "name": "My Company",
    "iban": "AT..."
  },
  "payments": [
    {
      "creditor_name": "Supplier",
      "creditor_iban": "DE...",
      "amount": 1000.00,
      "reference": "INV-001"
    }
  ]
}
```

### POST /sepa/pain008
Generate SEPA Direct Debit (pain.008).

### POST /sepa/camt053/parse
Parse bank statement (camt.053).

### POST /sepa/validate/iban
Validate IBAN.

---

## Documents

### GET /documents
List documents.

### POST /documents/upload
Upload document.

### GET /documents/:id
Get document metadata.

### GET /documents/:id/download
Download document.

### POST /documents/:id/analyze
Trigger AI analysis.

---

## System

### GET /health
Health check (no auth required).

### GET /system/version
Get system version.

### GET /system/metrics
Get system metrics (admin only).

---

## Error Responses

All errors follow this format:

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input",
    "details": [
      {"field": "email", "message": "Invalid email format"}
    ]
  }
}
```

Common error codes:
- `UNAUTHORIZED` - Missing or invalid token
- `FORBIDDEN` - Insufficient permissions
- `NOT_FOUND` - Resource not found
- `VALIDATION_ERROR` - Invalid input
- `RATE_LIMITED` - Too many requests
- `INTERNAL_ERROR` - Server error
