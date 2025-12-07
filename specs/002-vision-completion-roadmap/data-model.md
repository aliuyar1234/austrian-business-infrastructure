# Data Model: Austrian Business Infrastructure - Complete Product Suite

**Feature**: 002-vision-completion-roadmap
**Date**: 2025-12-07
**Input**: spec.md, research.md

## Overview

This document defines the data structures for all modules in the Austrian Business Infrastructure suite. All entities use Go-idiomatic naming and are designed for JSON/XML serialization.

---

## Module 1: FinanzOnline Extensions

### 1.1 UVA (Umsatzsteuervoranmeldung)

```go
// UVA represents a VAT advance return (Umsatzsteuervoranmeldung)
type UVA struct {
    // Period information
    Year    int       // Tax year (e.g., 2025)
    Period  UVAPeriod // Monthly (1-12) or Quarterly (1-4)

    // Tax amounts (in cents to avoid floating point)
    KZ000   int64 // Gesamtbetrag der Lieferungen (total deliveries)
    KZ001   int64 // Innergemeinschaftliche Lieferungen
    KZ011   int64 // Steuerfrei ohne Vorsteuerabzug
    KZ017   int64 // Normalsteuersatz 20%
    KZ018   int64 // Ermäßigter Steuersatz 10%
    KZ019   int64 // Ermäßigter Steuersatz 13%
    KZ020   int64 // Sonstige Steuersätze
    KZ022   int64 // Einfuhrumsatzsteuer
    KZ029   int64 // Innergemeinschaftliche Erwerbe
    KZ060   int64 // Vorsteuer
    KZ065   int64 // Einfuhrumsatzsteuer als Vorsteuer
    KZ066   int64 // Vorsteuern aus IG Erwerben
    KZ070   int64 // Sonstige Berichtigungen
    KZ095   int64 // Zahllast/Gutschrift (calculated)

    // Metadata
    CreatedAt   time.Time
    SubmittedAt *time.Time
    Status      UVAStatus
    Reference   string // FinanzOnline reference number
}

type UVAPeriod struct {
    Type  string // "monthly" or "quarterly"
    Value int    // 1-12 for monthly, 1-4 for quarterly
}

type UVAStatus string

const (
    UVAStatusDraft     UVAStatus = "draft"
    UVAStatusValidated UVAStatus = "validated"
    UVAStatusSubmitted UVAStatus = "submitted"
    UVAStatusAccepted  UVAStatus = "accepted"
    UVAStatusRejected  UVAStatus = "rejected"
)
```

### 1.2 UID Validation

```go
// UIDValidationRequest represents a UID validation query
type UIDValidationRequest struct {
    UID         string // Format: ATU12345678 (2-letter country + up to 12 chars)
    RequesterID string // Own UID for Level 2 queries (optional)
}

// UIDValidationResult contains the validation response
type UIDValidationResult struct {
    UID          string
    Valid        bool
    CompanyName  string
    Address      UIDAddress
    ValidAt      time.Time
    QueryTime    time.Time
    Source       string // "finanzonline" or "vies"
    ErrorCode    int
    ErrorMessage string
}

type UIDAddress struct {
    Street   string
    PostCode string
    City     string
    Country  string
}
```

### 1.3 Zusammenfassende Meldung (ZM)

```go
// ZM represents a recapitulative statement (Zusammenfassende Meldung)
type ZM struct {
    Year    int
    Quarter int // 1-4

    Entries []ZMEntry

    // Metadata
    CreatedAt   time.Time
    SubmittedAt *time.Time
    Status      ZMStatus
    Reference   string
}

type ZMEntry struct {
    PartnerUID   string // EU partner UID number
    CountryCode  string // 2-letter country code
    DeliveryType ZMDeliveryType
    Amount       int64  // In cents
}

type ZMDeliveryType string

const (
    ZMDeliveryTypeGoods              ZMDeliveryType = "L" // Lieferungen
    ZMDeliveryTypeTriangular         ZMDeliveryType = "D" // Dreiecksgeschäfte
    ZMDeliveryTypeServices           ZMDeliveryType = "S" // Sonstige Leistungen
)

type ZMStatus string

const (
    ZMStatusDraft     ZMStatus = "draft"
    ZMStatusSubmitted ZMStatus = "submitted"
    ZMStatusAccepted  ZMStatus = "accepted"
    ZMStatusRejected  ZMStatus = "rejected"
)
```

---

## Module 2: ELDA - Social Security

### 2.1 Employee Registration (Anmeldung)

```go
// ELDAAnmeldung represents an employee registration
type ELDAAnmeldung struct {
    // Employee data
    SVNummer      string    // 10-digit social security number
    Vorname       string
    Nachname      string
    Geburtsdatum  time.Time
    Geschlecht    string    // "M" or "W"

    // Employment data
    Eintrittsdatum    time.Time
    Beschaeftigung    ELDABeschaeftigung
    Arbeitszeit       ELDAArbeitszeit
    Entgelt           ELDAEntgelt

    // Employer reference
    DienstgeberNr string // ELDA employer number

    // Metadata
    Status    ELDAMeldungStatus
    Reference string
}

type ELDABeschaeftigung struct {
    Art          string // "vollzeit", "teilzeit", "geringfuegig"
    Taetigkeit   string // Job description
    Kollektiv    string // Collective agreement code
    Einstufung   string // Grading
}

type ELDAArbeitszeit struct {
    Stunden      float64 // Weekly hours
    Tage         int     // Days per week
}

type ELDAEntgelt struct {
    Brutto       int64   // Monthly gross in cents
    Netto        int64   // Monthly net in cents (optional)
    Sonderzahl   int64   // Special payments per year in cents
}

type ELDAMeldungStatus string

const (
    ELDAStatusDraft     ELDAMeldungStatus = "draft"
    ELDAStatusSubmitted ELDAMeldungStatus = "submitted"
    ELDAStatusProcessed ELDAMeldungStatus = "processed"
    ELDAStatusRejected  ELDAMeldungStatus = "rejected"
)
```

### 2.2 Employee Deregistration (Abmeldung)

```go
// ELDAAbmeldung represents an employee deregistration
type ELDAAbmeldung struct {
    SVNummer       string
    Austrittsdatum time.Time
    Grund          ELDAAustrittGrund

    // Final settlement
    Abfertigung    int64 // Severance pay in cents
    Urlaubsersatz  int64 // Vacation compensation in cents

    DienstgeberNr string
    Status        ELDAMeldungStatus
    Reference     string
}

type ELDAAustrittGrund string

const (
    ELDAGrundKuendigung      ELDAAustrittGrund = "K"  // Kündigung
    ELDAGrundEinvernehmlich  ELDAAustrittGrund = "E"  // Einvernehmlich
    ELDAGrundEntlassung      ELDAAustrittGrund = "EN" // Entlassung
    ELDAGrundAustritt        ELDAAustrittGrund = "A"  // Vorzeitiger Austritt
    ELDAGrundBefristet       ELDAAustrittGrund = "B"  // Befristung
)
```

### 2.3 ELDA Credentials

```go
// ELDACredentials stores ELDA-specific authentication
type ELDACredentials struct {
    DienstgeberNr string // Employer number
    BenutzerNr    string // User number
    PIN           string // PIN (encrypted in store)
}
```

---

## Module 3: Firmenbuch - Company Register

### 3.1 Company Search

```go
// FBSearchRequest represents a company search query
type FBSearchRequest struct {
    Firmenname  string // Company name (fuzzy search)
    FN          string // Firmenbuchnummer (exact search)
    Ort         string // Location filter
    Rechtsform  string // Legal form filter (GmbH, AG, etc.)
    Gericht     string // Court code filter
}

// FBSearchResult contains search results
type FBSearchResult struct {
    Entries []FBSearchEntry
    Total   int
}

type FBSearchEntry struct {
    FN         string // e.g., "FN 123456a"
    Firma      string // Company name
    Rechtsform string // Legal form
    Sitz       string // Registered office
    Status     string // "aktiv", "gelöscht", "in Liquidation"
}
```

### 3.2 Company Extract

```go
// FBExtract represents a full Firmenbuch extract
type FBExtract struct {
    FN             string
    Firma          string
    Rechtsform     string
    Sitz           string
    Geschaeftsadresse FBAdresse

    // Capital
    Stammkapital   int64  // In cents (for GmbH)
    Grundkapital   int64  // In cents (for AG)
    Waehrung       string // "EUR"

    // People
    Geschaeftsfuehrer []FBPerson
    Prokuristen       []FBPerson
    Gesellschafter    []FBGesellschafter
    Aufsichtsrat      []FBPerson

    // Metadata
    Gruendungsdatum  time.Time
    Eintragungsdatum time.Time
    Letztaktualisiert time.Time

    // History
    Aenderungen []FBAenderung
}

type FBAdresse struct {
    Strasse  string
    PLZ      string
    Ort      string
    Land     string
}

type FBPerson struct {
    Vorname      string
    Nachname     string
    Geburtsdatum time.Time
    Funktion     string // "Geschäftsführer", "Prokurist", etc.
    Seit         time.Time
    Bis          *time.Time // nil if still active
    Vertretung   string     // "selbständig", "gemeinsam"
}

type FBGesellschafter struct {
    Name         string // Person or company name
    FN           string // FN if company (optional)
    Anteil       string // e.g., "50%" or "12.500 EUR"
    Stammeinlage int64  // In cents
}

type FBAenderung struct {
    Datum       time.Time
    Art         string // Type of change
    Beschreibung string
}
```

### 3.3 Watchlist

```go
// FBWatchlist tracks companies for change monitoring
type FBWatchlist struct {
    Entries []FBWatchEntry
}

type FBWatchEntry struct {
    FN              string
    Firma           string
    AddedAt         time.Time
    LastChecked     time.Time
    LastChange      *time.Time
    ChangeCount     int
}
```

---

## Module 4: E-Rechnung - Electronic Invoicing

### 4.1 Invoice

```go
// Invoice represents an EN16931-compliant electronic invoice
type Invoice struct {
    // Header
    InvoiceNumber    string
    InvoiceDate      time.Time
    DueDate          time.Time
    InvoiceTypeCode  string // "380" = Commercial Invoice
    CurrencyCode     string // "EUR"

    // Period
    BillingPeriodStart *time.Time
    BillingPeriodEnd   *time.Time

    // Parties
    Seller InvoiceParty
    Buyer  InvoiceParty

    // Routing (for Peppol/USP)
    BuyerReference string // Leitweg-ID
    OrderReference string // Purchase order number

    // Lines
    Lines []InvoiceLine

    // Totals
    TotalNetAmount   int64 // In cents
    TotalTaxAmount   int64
    TotalGrossAmount int64

    // Tax breakdown
    TaxSubtotals []InvoiceTaxSubtotal

    // Payment
    PaymentTerms  string
    PaymentMethod string // "30" = Credit transfer
    PaymentIBAN   string
    PaymentBIC    string

    // Notes
    Notes []string
}

type InvoiceParty struct {
    Name           string
    UID            string // VAT number (optional)
    GLN            string // Global Location Number (optional)
    Street         string
    PostCode       string
    City           string
    CountryCode    string // ISO 3166-1 alpha-2
    ContactName    string
    ContactEmail   string
    ContactPhone   string
}

type InvoiceLine struct {
    LineID          string
    Description     string
    Quantity        float64
    QuantityUnit    string // UN/ECE Recommendation 20
    UnitPrice       int64  // In cents
    NetAmount       int64
    TaxCategoryCode string // "S" = Standard, "Z" = Zero, "E" = Exempt
    TaxPercent      float64
    TaxAmount       int64
}

type InvoiceTaxSubtotal struct {
    TaxableAmount   int64
    TaxAmount       int64
    TaxCategoryCode string
    TaxPercent      float64
}
```

### 4.2 Invoice Format

```go
type InvoiceFormat string

const (
    FormatXRechnung    InvoiceFormat = "xrechnung"
    FormatZUGFeRD      InvoiceFormat = "zugferd"
    FormatZUGFeRDBasic InvoiceFormat = "zugferd_basic"
)
```

### 4.3 Validation Result

```go
type InvoiceValidationResult struct {
    Valid    bool
    Errors   []InvoiceValidationError
    Warnings []InvoiceValidationError
}

type InvoiceValidationError struct {
    Code     string // e.g., "BR-01"
    Field    string // e.g., "InvoiceNumber"
    Message  string
    Severity string // "error" or "warning"
}
```

---

## Module 5: SEPA - Banking

### 5.1 Credit Transfer (pain.001)

```go
// SEPACreditTransfer represents a SEPA credit transfer batch
type SEPACreditTransfer struct {
    MessageID       string
    CreationTime    time.Time

    // Initiator
    InitiatorName   string
    InitiatorID     string // Optional, e.g., creditor ID

    // Debtor (sender)
    DebtorName      string
    DebtorIBAN      string
    DebtorBIC       string

    // Payment information
    PaymentMethod   string    // "TRF"
    BatchBooking    bool
    RequestedDate   time.Time // Execution date

    // Transactions
    Transactions    []SEPACreditTransaction

    // Totals (calculated)
    NumberOfTx      int
    ControlSum      int64 // In cents
}

type SEPACreditTransaction struct {
    EndToEndID      string // Unique reference
    InstructedAmount int64 // In cents
    Currency        string // "EUR"

    // Creditor (recipient)
    CreditorName    string
    CreditorIBAN    string
    CreditorBIC     string // Optional, can be derived
    CreditorAddress *SEPAAddress

    // Purpose
    RemittanceInfo  string // Verwendungszweck (max 140 chars)
    PurposeCode     string // Optional ISO purpose code
}

type SEPAAddress struct {
    Street      string
    Building    string
    PostCode    string
    TownName    string
    CountryCode string // ISO 3166-1 alpha-2
}
```

### 5.2 Direct Debit (pain.008)

```go
// SEPADirectDebit represents a SEPA direct debit batch
type SEPADirectDebit struct {
    MessageID       string
    CreationTime    time.Time

    // Creditor (collector)
    CreditorName    string
    CreditorID      string // Gläubiger-ID
    CreditorIBAN    string
    CreditorBIC     string

    // Collection details
    SequenceType    string    // "FRST", "RCUR", "OOFF", "FNAL"
    CollectionDate  time.Time

    // Transactions
    Transactions    []SEPADebitTransaction

    NumberOfTx      int
    ControlSum      int64
}

type SEPADebitTransaction struct {
    EndToEndID        string
    InstructedAmount  int64
    Currency          string

    // Debtor (payer)
    DebtorName        string
    DebtorIBAN        string
    DebtorBIC         string

    // Mandate
    MandateID         string
    MandateSignDate   time.Time

    RemittanceInfo    string
}
```

### 5.3 Account Statement (camt.053)

```go
// SEPAStatement represents a bank statement
type SEPAStatement struct {
    StatementID     string
    CreationTime    time.Time
    AccountIBAN     string
    AccountBIC      string

    // Period
    FromDate        time.Time
    ToDate          time.Time

    // Balances
    OpeningBalance  SEPABalance
    ClosingBalance  SEPABalance

    // Entries
    Entries         []SEPAStatementEntry
}

type SEPABalance struct {
    Amount   int64
    Currency string
    Date     time.Time
    Type     string // "OPBD" = Opening, "CLBD" = Closing
}

type SEPAStatementEntry struct {
    EntryRef        string
    Amount          int64
    Currency        string
    CreditDebit     string // "CRDT" or "DBIT"
    BookingDate     time.Time
    ValueDate       time.Time

    // Counterparty
    CounterpartyName string
    CounterpartyIBAN string

    // Details
    RemittanceInfo  string
    EndToEndID      string
    MandateID       string // For direct debits
}
```

### 5.4 IBAN Validation

```go
// IBANValidationResult contains IBAN validation results
type IBANValidationResult struct {
    IBAN         string
    Valid        bool
    CountryCode  string
    BankCode     string
    BIC          string // Derived BIC
    BankName     string
    ErrorMessage string
}
```

---

## Module 6: MCP Server

### 6.1 Tool Definitions

```go
// MCPTool defines an MCP tool exposed by the server
type MCPTool struct {
    Name        string
    Description string
    InputSchema map[string]interface{} // JSON Schema
}

// MCPToolResult represents a tool execution result
type MCPToolResult struct {
    Content     []MCPContent
    IsError     bool
}

type MCPContent struct {
    Type string // "text" or "resource"
    Text string
    URI  string // For resources
}
```

### 6.2 Server Configuration

```go
// MCPServerConfig configures the MCP server
type MCPServerConfig struct {
    Name        string
    Version     string
    Transport   string // "stdio" or "http"

    // Feature flags
    EnableFonws bool
    EnableElda  bool
    EnableFB    bool

    // Security
    RequireAuth bool
}
```

---

## Cross-Cutting: Extended Credential Store

### Account Extensions

```go
// AccountType distinguishes credential types
type AccountType string

const (
    AccountTypeFinanzOnline AccountType = "finanzonline"
    AccountTypeELDA         AccountType = "elda"
    AccountTypeFirmenbuch   AccountType = "firmenbuch"
)

// ExtendedAccount extends the base account with type
type ExtendedAccount struct {
    Name        string
    Type        AccountType

    // FinanzOnline
    TID         string
    BenID       string
    PIN         string

    // ELDA (if Type == AccountTypeELDA)
    DienstgeberNr string
    ELDABenutzer  string
    ELDAPIN       string

    // Firmenbuch (if Type == AccountTypeFirmenbuch)
    APIKey        string
}
```

---

## Validation Rules

### UVA
- Year: 2000-2100
- Period: Monthly 1-12 or Quarterly 1-4
- All KZ values: >= 0 (non-negative)
- KZ095: Calculated, can be negative (refund)

### UID
- Format: 2-letter country code + up to 12 alphanumeric characters
- Austrian: ATU + 8 digits

### ELDA
- SVNummer: Exactly 10 digits
- Geburtsdatum: Must match SVNummer validation digit

### IBAN
- Austrian: AT + 2 check digits + 16 digits (total 20 chars)
- Check digit validation per ISO 7064

### Invoice
- InvoiceNumber: Non-empty, unique per seller
- TotalGrossAmount: TotalNetAmount + TotalTaxAmount
- At least one InvoiceLine required

---

## Next Steps

1. Create contracts/ with SOAP XML structures
2. Create quickstart.md with usage examples
3. Generate tasks.md from this data model
