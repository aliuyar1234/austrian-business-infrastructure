# Data Model: Full Vision Completion

**Feature Branch**: `004-full-vision-completion`
**Date**: 2025-12-07

---

## Entity Overview

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Account   │     │   Session   │     │  Employee   │
│  (store)    │────▶│  (fonws)    │     │   (elda)    │
└─────────────┘     └─────────────┘     └─────────────┘
       │                   │
       │                   ▼
       │            ┌─────────────┐
       │            │   Databox   │
       │            │  Document   │
       │            └─────────────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Company   │     │   Invoice   │     │   Payment   │
│    (fb)     │     │ (erechnung) │     │   (sepa)    │
└─────────────┘     └─────────────┘     └─────────────┘
```

---

## 1. Account (Credential Storage)

**Module**: `internal/store`
**Purpose**: Encrypted credentials for all services

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| ID | string | Yes | Unique identifier (UUID) |
| Name | string | Yes | Display name (e.g., "Muster GmbH") |
| Type | AccountType | Yes | Service type enum |
| TID | string | Conditional | FinanzOnline Teilnehmer-ID (9-12 chars) |
| BenID | string | Conditional | FinanzOnline Benutzer-ID |
| PIN | string | Conditional | Encrypted PIN/password |
| ELDASerial | string | Conditional | ELDA Seriennummer |
| BKNR | string | Conditional | ELDA Beitragskontonummer |
| FBAPIKey | string | Conditional | Firmenbuch API key |
| CreatedAt | time.Time | Yes | Creation timestamp |
| UpdatedAt | time.Time | Yes | Last update timestamp |

### AccountType Enum

| Value | Description |
|-------|-------------|
| `finanzonline` | FinanzOnline WebService account |
| `elda` | ELDA social insurance account |
| `firmenbuch` | Firmenbuch API account |

### Validation Rules

- Name: 1-100 characters, non-empty
- TID: 9-12 alphanumeric characters (FinanzOnline)
- BenID: 5-12 alphanumeric characters (FinanzOnline)
- ELDASerial: Format per ELDA specification
- BKNR: 10-digit contribution account number
- FBAPIKey: Non-empty string

---

## 2. Session (Active Authentication)

**Module**: `internal/fonws`
**Purpose**: Active session with government service

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| ID | string | Yes | Session token from service |
| AccountID | string | Yes | Reference to Account |
| TID | string | Yes | Teilnehmer-ID used for login |
| BenID | string | Yes | Benutzer-ID used for login |
| CreatedAt | time.Time | Yes | Login timestamp |
| ExpiresAt | time.Time | Yes | Session expiry (typically 30 min) |
| LastUsed | time.Time | Yes | Last API call timestamp |

### State Transitions

```
[None] ──login()──▶ [Active] ──expire/logout()──▶ [Expired]
                        │
                        └──refresh()──▶ [Active]
```

### Validation Rules

- Session ID: 10-24 alphanumeric characters
- ExpiresAt must be after CreatedAt
- Session considered expired if current time > ExpiresAt

---

## 3. Employee (ELDA Registration)

**Module**: `internal/elda`
**Purpose**: Employee data for social insurance registration

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| SVNummer | string | Yes | Social security number (10 digits) |
| FirstName | string | Yes | Employee first name |
| LastName | string | Yes | Employee last name |
| DateOfBirth | time.Time | Yes | Date of birth |
| EmployerVSNR | string | Yes | Employer insurance number |
| StartDate | time.Time | Yes | Employment start date |
| EndDate | time.Time | No | Employment end date (for Abmeldung) |
| WeeklyHours | float64 | No | Weekly working hours |
| MonthlyGross | int64 | No | Monthly gross salary (cents) |
| JobTitle | string | No | Job description |
| InsuranceType | string | Yes | Insurance category code |

### SVNummer Validation

```
Format: SSSSDDMMYY (10 digits)
- SSSS: Sequential number (4th digit = check digit)
- DDMMYY: Date of birth
- Month 13/14 are valid (administrative purposes)

Checksum weights: [6, 1, 2, 4, 8, 5, 0, 9, 7, 3]
Valid if: sum(digit[i] * weight[i]) mod 11 != 10
```

### Registration Types

| Type | German | Purpose |
|------|--------|---------|
| Anmeldung | Registration | New employee start |
| Abmeldung | Deregistration | Employee termination |
| Änderung | Change | Update existing registration |

---

## 4. Company (Firmenbuch)

**Module**: `internal/fb`
**Purpose**: Company registry data

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| FNNummer | string | Yes | Firmenbuch number (e.g., "123456b") |
| Name | string | Yes | Company name |
| LegalForm | string | Yes | Legal form (GmbH, AG, OG, KG, etc.) |
| RegisteredOffice | string | Yes | Sitz (registered city) |
| BusinessAddress | Address | Yes | Full business address |
| BusinessSector | string | No | Line of business |
| ShareCapital | int64 | No | Share capital in cents |
| FoundingDate | time.Time | No | Date of incorporation |
| UID | string | No | Austrian VAT number |
| InsolvencyStatus | string | No | Insolvency indicator |
| Directors | []Director | Yes | Managing directors |
| Prokuristen | []Director | No | Commercial attorneys |
| LastUpdated | time.Time | Yes | Last registry update |

### Director (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Name | string | Yes | Full name |
| DateOfBirth | time.Time | Yes | Date of birth |
| Address | Address | No | Residential address |
| Position | string | Yes | Role (Geschäftsführer, Vorstand, etc.) |
| RepresentationType | string | Yes | Solo, joint, etc. |
| Since | time.Time | Yes | Appointment date |

### Address (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Street | string | Yes | Street and number |
| PostalCode | string | Yes | Postal code |
| City | string | Yes | City name |
| Country | string | Yes | ISO country code (default: AT) |

### FN-Nummer Validation

```
Format: [FN ]?[0-9]{1,6}[a-z]
Examples: "FN 123456 b", "123456b"
Check letter calculated from digits
```

---

## 5. Invoice (E-Rechnung)

**Module**: `internal/erechnung`
**Purpose**: EN16931-compliant e-invoice

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| ID | string | Yes | Invoice number |
| IssueDate | time.Time | Yes | Invoice date |
| DueDate | time.Time | No | Payment due date |
| InvoiceType | InvoiceType | Yes | Invoice type code |
| Currency | string | Yes | ISO currency code (default: EUR) |
| Seller | InvoiceParty | Yes | Seller/supplier details |
| Buyer | InvoiceParty | Yes | Buyer/customer details |
| Lines | []InvoiceLine | Yes | Invoice line items |
| TaxSubtotals | []TaxSubtotal | Yes | Tax breakdown |
| TotalNetAmount | int64 | Yes | Net total (cents) |
| TotalTaxAmount | int64 | Yes | Total tax (cents) |
| TotalGrossAmount | int64 | Yes | Gross total (cents) |
| PaymentTerms | string | No | Payment terms text |
| PaymentReference | string | No | Payment reference |
| BankAccount | BankAccount | No | Seller bank details |
| Notes | []string | No | Additional notes |

### InvoiceParty (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| Name | string | Yes | Party name |
| UID | string | Conditional | VAT number (required for B2B) |
| Address | Address | Yes | Postal address |
| Contact | Contact | No | Contact information |
| GLN | string | No | Global Location Number |

### InvoiceLine (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| LineNumber | int | Yes | Sequential line number |
| Description | string | Yes | Item description |
| Quantity | float64 | Yes | Quantity |
| Unit | string | Yes | Unit code (C62=piece, KGM=kg, etc.) |
| UnitPrice | int64 | Yes | Unit price (cents) |
| NetAmount | int64 | Yes | Line net amount (cents) |
| TaxCategory | TaxCategory | Yes | Tax category |
| TaxRate | float64 | Yes | Tax rate percentage |

### TaxCategory Enum (UNCL 5305)

| Code | Description | Rate |
|------|-------------|------|
| S | Standard rated | 20%, 13%, 10% |
| Z | Zero rated | 0% |
| E | Exempt | 0% |
| AE | Reverse charge | 0% |
| K | Intra-community | 0% |

### InvoiceType Enum

| Code | Description |
|------|-------------|
| 380 | Commercial invoice |
| 381 | Credit note |
| 389 | Self-billed invoice |

---

## 6. Payment (SEPA)

**Module**: `internal/sepa`
**Purpose**: SEPA payment instruction

### SEPACreditTransfer (pain.001)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| MessageID | string | Yes | Unique message identifier |
| CreationDateTime | time.Time | Yes | Creation timestamp |
| NumberOfTransactions | int | Yes | Transaction count |
| ControlSum | int64 | Yes | Sum of amounts (cents) |
| Initiator | SEPAParty | Yes | Initiating party |
| PaymentInfo | []PaymentInfo | Yes | Payment batches |

### PaymentInfo (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| PaymentInfoID | string | Yes | Batch identifier |
| PaymentMethod | string | Yes | "TRF" for credit transfer |
| RequestedExecutionDate | time.Time | Yes | Execution date |
| Debtor | SEPAParty | Yes | Payer details |
| DebtorAccount | SEPAAccount | Yes | Payer account |
| DebtorAgent | string | No | Payer bank BIC |
| Transactions | []CreditTransferTx | Yes | Individual payments |

### CreditTransferTx (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| EndToEndID | string | Yes | Unique transaction ID |
| Amount | int64 | Yes | Amount in cents |
| Currency | string | Yes | ISO currency (EUR) |
| Creditor | SEPAParty | Yes | Payee details |
| CreditorAccount | SEPAAccount | Yes | Payee account |
| CreditorAgent | string | No | Payee bank BIC |
| RemittanceInfo | string | No | Payment reference (140 chars max) |

### SEPAAccount (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| IBAN | string | Yes | International Bank Account Number |
| BIC | string | No | Bank Identifier Code (optional since 2016) |
| Name | string | No | Account holder name |

### IBAN Validation

```
Austrian IBAN: AT + 2 check digits + 16 BBAN
Length: 20 characters
Algorithm: ISO 7064 MOD 97-10
1. Move first 4 chars to end
2. Convert letters: A=10, B=11... Z=35
3. Calculate: number mod 97
4. Valid if result = 1
```

---

## 7. UVA (VAT Advance Return)

**Module**: `internal/fonws`
**Purpose**: Umsatzsteuervoranmeldung data

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| TaxNumber | string | Yes | 9-digit tax number |
| Year | int | Yes | Reporting year |
| Period | UVAPeriod | Yes | Month (1-12) or Quarter (1-4) |
| IsQuarterly | bool | Yes | Monthly or quarterly filing |
| Kennzahlen | map[string]int64 | Yes | KZ values in cents |
| SubmittedAt | time.Time | No | Submission timestamp |
| ConfirmationNumber | string | No | BMF confirmation |

### Key Kennzahlen

| KZ | Description |
|----|-------------|
| KZ000 | Total deliveries |
| KZ001 | Intra-community deliveries |
| KZ017 | 20% tax base |
| KZ018 | 10% tax base |
| KZ019 | 13% tax base |
| KZ060 | Input tax deduction |
| KZ095 | Tax liability/credit (calculated) |

---

## 8. ZM (Summary Declaration)

**Module**: `internal/fonws`
**Purpose**: Zusammenfassende Meldung for intra-EU transactions

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| TaxNumber | string | Yes | 9-digit tax number |
| Year | int | Yes | Reporting year |
| Quarter | int | Yes | Quarter (1-4) |
| Entries | []ZMEntry | Yes | Transaction entries |
| SubmittedAt | time.Time | No | Submission timestamp |
| ConfirmationNumber | string | No | BMF confirmation |

### ZMEntry (Nested)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| PartnerUID | string | Yes | Partner VAT number |
| CountryCode | string | Yes | ISO country code |
| TransactionType | ZMType | Yes | L, D, or S |
| Amount | int64 | Yes | Amount in EUR (whole euros) |

### ZMType Enum

| Code | German | English |
|------|--------|---------|
| L | Lieferungen | Goods deliveries |
| D | Dreiecksgeschäfte | Triangular transactions |
| S | Sonstige Leistungen | Services |

---

## Relationships

```
Account 1──────* Session
Account 1──────* Employee (via employer reference)
Account 1──────* Company (via API key)
Company 1──────* Director
Invoice 1──────* InvoiceLine
Invoice 1──────* TaxSubtotal
SEPACreditTransfer 1──────* PaymentInfo
PaymentInfo 1──────* CreditTransferTx
UVA 1──────1 Account (via TaxNumber)
ZM 1──────* ZMEntry
```

---

## Storage Strategy

| Entity | Storage | Encryption |
|--------|---------|------------|
| Account | File (accounts.enc) | AES-256-GCM |
| Session | Memory only | N/A |
| Employee | Input file (JSON) | N/A |
| Company | API response (cached) | N/A |
| Invoice | Output file (XML/PDF) | N/A |
| Payment | Output file (XML) | N/A |
| UVA | Generated XML | N/A |
| ZM | Generated XML | N/A |

---

## Amount Handling

All monetary amounts stored as **int64 in cents** to avoid floating-point precision issues.

```go
// Convert to cents
cents := int64(euros * 100)

// Format for display
euros := float64(cents) / 100
formatted := fmt.Sprintf("%.2f", euros)
```

Exception: ZM amounts are in whole EUR per BMF specification.
