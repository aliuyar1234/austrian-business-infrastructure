# Contracts: E-Rechnung - Electronic Invoicing

**Module**: erechnung
**Date**: 2025-12-07

## 1. Overview

The E-Rechnung module generates and validates electronic invoices according to EN 16931 (European norm). Primary formats:

- **XRechnung**: Pure XML format (German standard, compatible with Austrian requirements)
- **ZUGFeRD**: PDF/A-3 with embedded XML (hybrid format)

---

## 2. XRechnung Format (UBL 2.1)

### 2.1 Invoice Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
         xmlns:cac="urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"
         xmlns:cbc="urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2">

    <!-- Business Document Header -->
    <cbc:CustomizationID>urn:cen.eu:en16931:2017#compliant#urn:xeinkauf.de:kosit:xrechnung_3.0</cbc:CustomizationID>
    <cbc:ProfileID>urn:fdc:peppol.eu:2017:poacc:billing:01:1.0</cbc:ProfileID>

    <!-- Invoice Identification -->
    <cbc:ID>INV-2025-001234</cbc:ID>
    <cbc:IssueDate>2025-01-15</cbc:IssueDate>
    <cbc:DueDate>2025-02-14</cbc:DueDate>
    <cbc:InvoiceTypeCode>380</cbc:InvoiceTypeCode>
    <cbc:DocumentCurrencyCode>EUR</cbc:DocumentCurrencyCode>

    <!-- Buyer Reference (Leitweg-ID) -->
    <cbc:BuyerReference>04011000-12345-67</cbc:BuyerReference>

    <!-- Seller -->
    <cac:AccountingSupplierParty>
        <cac:Party>
            <cac:PartyName>
                <cbc:Name>Lieferant GmbH</cbc:Name>
            </cac:PartyName>
            <cac:PostalAddress>
                <cbc:StreetName>Lieferantenstraße 1</cbc:StreetName>
                <cbc:CityName>Wien</cbc:CityName>
                <cbc:PostalZone>1010</cbc:PostalZone>
                <cac:Country>
                    <cbc:IdentificationCode>AT</cbc:IdentificationCode>
                </cac:Country>
            </cac:PostalAddress>
            <cac:PartyTaxScheme>
                <cbc:CompanyID>ATU12345678</cbc:CompanyID>
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:PartyTaxScheme>
        </cac:Party>
    </cac:AccountingSupplierParty>

    <!-- Buyer -->
    <cac:AccountingCustomerParty>
        <cac:Party>
            <cac:PartyName>
                <cbc:Name>Kunde AG</cbc:Name>
            </cac:PartyName>
            <cac:PostalAddress>
                <cbc:StreetName>Kundenweg 5</cbc:StreetName>
                <cbc:CityName>Graz</cbc:CityName>
                <cbc:PostalZone>8010</cbc:PostalZone>
                <cac:Country>
                    <cbc:IdentificationCode>AT</cbc:IdentificationCode>
                </cac:Country>
            </cac:PostalAddress>
        </cac:Party>
    </cac:AccountingCustomerParty>

    <!-- Payment Terms -->
    <cac:PaymentMeans>
        <cbc:PaymentMeansCode>30</cbc:PaymentMeansCode>
        <cac:PayeeFinancialAccount>
            <cbc:ID>AT611904300234573201</cbc:ID>
        </cac:PayeeFinancialAccount>
    </cac:PaymentMeans>

    <!-- Tax Summary -->
    <cac:TaxTotal>
        <cbc:TaxAmount currencyID="EUR">200.00</cbc:TaxAmount>
        <cac:TaxSubtotal>
            <cbc:TaxableAmount currencyID="EUR">1000.00</cbc:TaxableAmount>
            <cbc:TaxAmount currencyID="EUR">200.00</cbc:TaxAmount>
            <cac:TaxCategory>
                <cbc:ID>S</cbc:ID>
                <cbc:Percent>20</cbc:Percent>
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:TaxCategory>
        </cac:TaxSubtotal>
    </cac:TaxTotal>

    <!-- Document Totals -->
    <cac:LegalMonetaryTotal>
        <cbc:LineExtensionAmount currencyID="EUR">1000.00</cbc:LineExtensionAmount>
        <cbc:TaxExclusiveAmount currencyID="EUR">1000.00</cbc:TaxExclusiveAmount>
        <cbc:TaxInclusiveAmount currencyID="EUR">1200.00</cbc:TaxInclusiveAmount>
        <cbc:PayableAmount currencyID="EUR">1200.00</cbc:PayableAmount>
    </cac:LegalMonetaryTotal>

    <!-- Invoice Lines -->
    <cac:InvoiceLine>
        <cbc:ID>1</cbc:ID>
        <cbc:InvoicedQuantity unitCode="C62">10</cbc:InvoicedQuantity>
        <cbc:LineExtensionAmount currencyID="EUR">1000.00</cbc:LineExtensionAmount>
        <cac:Item>
            <cbc:Description>Beratungsleistung</cbc:Description>
            <cbc:Name>Consulting Service</cbc:Name>
            <cac:ClassifiedTaxCategory>
                <cbc:ID>S</cbc:ID>
                <cbc:Percent>20</cbc:Percent>
                <cac:TaxScheme>
                    <cbc:ID>VAT</cbc:ID>
                </cac:TaxScheme>
            </cac:ClassifiedTaxCategory>
        </cac:Item>
        <cac:Price>
            <cbc:PriceAmount currencyID="EUR">100.00</cbc:PriceAmount>
        </cac:Price>
    </cac:InvoiceLine>
</Invoice>
```

---

## 3. ZUGFeRD Format (CII)

ZUGFeRD uses UN/CEFACT Cross Industry Invoice (CII) format:

### 3.1 Embedded XML Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rsm:CrossIndustryInvoice xmlns:rsm="urn:un:unece:uncefact:data:standard:CrossIndustryInvoice:100"
    xmlns:ram="urn:un:unece:uncefact:data:standard:ReusableAggregateBusinessInformationEntity:100"
    xmlns:udt="urn:un:unece:uncefact:data:standard:UnqualifiedDataType:100">

    <rsm:ExchangedDocumentContext>
        <ram:GuidelineSpecifiedDocumentContextParameter>
            <ram:ID>urn:factur-x.eu:1p0:en16931</ram:ID>
        </ram:GuidelineSpecifiedDocumentContextParameter>
    </rsm:ExchangedDocumentContext>

    <rsm:ExchangedDocument>
        <ram:ID>INV-2025-001234</ram:ID>
        <ram:TypeCode>380</ram:TypeCode>
        <ram:IssueDateTime>
            <udt:DateTimeString format="102">20250115</udt:DateTimeString>
        </ram:IssueDateTime>
    </rsm:ExchangedDocument>

    <rsm:SupplyChainTradeTransaction>
        <!-- Trade Agreement (Parties) -->
        <ram:ApplicableHeaderTradeAgreement>
            <ram:SellerTradeParty>
                <ram:Name>Lieferant GmbH</ram:Name>
                <ram:PostalTradeAddress>
                    <ram:LineOne>Lieferantenstraße 1</ram:LineOne>
                    <ram:PostcodeCode>1010</ram:PostcodeCode>
                    <ram:CityName>Wien</ram:CityName>
                    <ram:CountryID>AT</ram:CountryID>
                </ram:PostalTradeAddress>
                <ram:SpecifiedTaxRegistration>
                    <ram:ID schemeID="VA">ATU12345678</ram:ID>
                </ram:SpecifiedTaxRegistration>
            </ram:SellerTradeParty>
            <ram:BuyerTradeParty>
                <ram:Name>Kunde AG</ram:Name>
            </ram:BuyerTradeParty>
        </ram:ApplicableHeaderTradeAgreement>

        <!-- Trade Settlement (Payment, Taxes, Totals) -->
        <ram:ApplicableHeaderTradeSettlement>
            <ram:InvoiceCurrencyCode>EUR</ram:InvoiceCurrencyCode>
            <ram:SpecifiedTradeSettlementPaymentMeans>
                <ram:TypeCode>30</ram:TypeCode>
                <ram:PayeePartyCreditorFinancialAccount>
                    <ram:IBANID>AT611904300234573201</ram:IBANID>
                </ram:PayeePartyCreditorFinancialAccount>
            </ram:SpecifiedTradeSettlementPaymentMeans>
            <ram:ApplicableTradeTax>
                <ram:CalculatedAmount>200.00</ram:CalculatedAmount>
                <ram:BasisAmount>1000.00</ram:BasisAmount>
                <ram:CategoryCode>S</ram:CategoryCode>
                <ram:RateApplicablePercent>20</ram:RateApplicablePercent>
            </ram:ApplicableTradeTax>
            <ram:SpecifiedTradeSettlementHeaderMonetarySummation>
                <ram:LineTotalAmount>1000.00</ram:LineTotalAmount>
                <ram:TaxBasisTotalAmount>1000.00</ram:TaxBasisTotalAmount>
                <ram:TaxTotalAmount currencyID="EUR">200.00</ram:TaxTotalAmount>
                <ram:GrandTotalAmount>1200.00</ram:GrandTotalAmount>
                <ram:DuePayableAmount>1200.00</ram:DuePayableAmount>
            </ram:SpecifiedTradeSettlementHeaderMonetarySummation>
        </ram:ApplicableHeaderTradeSettlement>

        <!-- Line Items -->
        <ram:IncludedSupplyChainTradeLineItem>
            <ram:AssociatedDocumentLineDocument>
                <ram:LineID>1</ram:LineID>
            </ram:AssociatedDocumentLineDocument>
            <ram:SpecifiedTradeProduct>
                <ram:Name>Consulting Service</ram:Name>
            </ram:SpecifiedTradeProduct>
            <ram:SpecifiedLineTradeAgreement>
                <ram:NetPriceProductTradePrice>
                    <ram:ChargeAmount>100.00</ram:ChargeAmount>
                </ram:NetPriceProductTradePrice>
            </ram:SpecifiedLineTradeAgreement>
            <ram:SpecifiedLineTradeDelivery>
                <ram:BilledQuantity unitCode="C62">10</ram:BilledQuantity>
            </ram:SpecifiedLineTradeDelivery>
            <ram:SpecifiedLineTradeSettlement>
                <ram:ApplicableTradeTax>
                    <ram:CategoryCode>S</ram:CategoryCode>
                    <ram:RateApplicablePercent>20</ram:RateApplicablePercent>
                </ram:ApplicableTradeTax>
                <ram:SpecifiedTradeSettlementLineMonetarySummation>
                    <ram:LineTotalAmount>1000.00</ram:LineTotalAmount>
                </ram:SpecifiedTradeSettlementLineMonetarySummation>
            </ram:SpecifiedLineTradeSettlement>
        </ram:IncludedSupplyChainTradeLineItem>
    </rsm:SupplyChainTradeTransaction>
</rsm:CrossIndustryInvoice>
```

---

## 4. Invoice Type Codes

| Code | Description |
|------|-------------|
| 380 | Commercial invoice |
| 381 | Credit note |
| 384 | Corrected invoice |
| 389 | Self-billed invoice |
| 751 | Invoice information for accounting purposes |

---

## 5. Tax Category Codes

| Code | Description | Rate |
|------|-------------|------|
| S | Standard rate | 20% (AT), varies by country |
| Z | Zero rated | 0% |
| E | Exempt | N/A |
| AE | Reverse charge | N/A |
| K | Intra-community supply | 0% |
| G | Export outside EU | 0% |

---

## 6. Unit Codes (UN/ECE Rec 20)

| Code | Description |
|------|-------------|
| C62 | One (piece) |
| HUR | Hour |
| DAY | Day |
| MON | Month |
| KGM | Kilogram |
| MTR | Metre |
| LTR | Litre |

---

## 7. Validation Rules (EN 16931)

### 7.1 Business Rules (BR)

| Rule | Description |
|------|-------------|
| BR-01 | Invoice must have a specification identifier |
| BR-02 | Invoice must have an invoice number |
| BR-03 | Invoice must have an issue date |
| BR-04 | Invoice must have an invoice type code |
| BR-05 | Invoice must have a currency code |
| BR-06 | Invoice must have seller name |
| BR-07 | Invoice must have buyer name |

### 7.2 Calculation Rules

```
TaxAmount = TaxableAmount × TaxPercent / 100
TaxInclusiveAmount = TaxExclusiveAmount + TaxAmount
LineExtensionAmount = Quantity × UnitPrice
```

---

## 8. Go Struct Definitions

```go
// internal/erechnung/invoice.go

// Invoice represents an EN16931-compliant invoice
type Invoice struct {
    // Header
    CustomizationID string
    ProfileID       string
    ID              string
    IssueDate       time.Time
    DueDate         time.Time
    TypeCode        string
    CurrencyCode    string
    BuyerReference  string

    // Parties
    Seller InvoiceParty
    Buyer  InvoiceParty

    // Payment
    PaymentMeansCode string
    PaymentIBAN      string
    PaymentBIC       string
    PaymentTerms     string

    // Tax
    TaxTotal     int64  // In cents
    TaxSubtotals []TaxSubtotal

    // Totals
    LineExtensionAmount int64 // Net total
    TaxExclusiveAmount  int64 // Before tax
    TaxInclusiveAmount  int64 // After tax
    PayableAmount       int64 // Due amount

    // Lines
    Lines []InvoiceLine

    // Notes
    Notes []string
}

type InvoiceParty struct {
    Name        string
    Street      string
    City        string
    PostCode    string
    CountryCode string
    TaxID       string // UID number
    GLN         string // Global Location Number
    Contact     *PartyContact
}

type PartyContact struct {
    Name  string
    Phone string
    Email string
}

type TaxSubtotal struct {
    TaxableAmount int64   // In cents
    TaxAmount     int64   // In cents
    CategoryCode  string  // S, Z, E, etc.
    Percent       float64 // 20, 10, 0, etc.
}

type InvoiceLine struct {
    ID              string
    Quantity        float64
    QuantityUnit    string // UN/ECE code
    UnitPrice       int64  // In cents
    LineAmount      int64  // In cents
    Description     string
    TaxCategoryCode string
    TaxPercent      float64
}
```

---

## 9. Validation Result

```go
// ValidationResult from EN16931 validation
type ValidationResult struct {
    Valid    bool
    Errors   []ValidationError
    Warnings []ValidationError
}

type ValidationError struct {
    Rule     string // e.g., "BR-01"
    Location string // XPath or field name
    Message  string
    Severity string // "error" or "warning"
}

// Example errors
var ErrMissingInvoiceNumber = ValidationError{
    Rule:     "BR-02",
    Location: "Invoice/ID",
    Message:  "An Invoice shall have an Invoice number",
    Severity: "error",
}
```

---

## 10. CLI Commands

```bash
# Create invoice from JSON
erechnung create invoice.json
erechnung create invoice.json --format xrechnung --output invoice.xml
erechnung create invoice.json --format zugferd --output invoice.pdf

# Validate invoice
erechnung validate invoice.xml
erechnung validate invoice.pdf --format zugferd

# Extract from PDF
erechnung extract invoice.pdf --output invoice.json

# Convert between formats
erechnung convert invoice.xml --to zugferd --output invoice.pdf
erechnung convert invoice.pdf --to xrechnung --output invoice.xml
```

---

## 11. Input JSON Format

```json
{
    "invoice_number": "INV-2025-001234",
    "issue_date": "2025-01-15",
    "due_date": "2025-02-14",
    "buyer_reference": "04011000-12345-67",

    "seller": {
        "name": "Lieferant GmbH",
        "street": "Lieferantenstraße 1",
        "city": "Wien",
        "post_code": "1010",
        "country": "AT",
        "tax_id": "ATU12345678"
    },

    "buyer": {
        "name": "Kunde AG",
        "street": "Kundenweg 5",
        "city": "Graz",
        "post_code": "8010",
        "country": "AT"
    },

    "payment": {
        "iban": "AT611904300234573201"
    },

    "lines": [
        {
            "description": "Consulting Service",
            "quantity": 10,
            "unit": "C62",
            "unit_price": 100.00,
            "tax_category": "S",
            "tax_percent": 20
        }
    ]
}
```
