# Research Findings: Full Vision Completion

**Feature Branch**: `004-full-vision-completion`
**Date**: 2025-12-07

---

## 1. ELDA (Social Insurance) Integration

### Decision
Use XML file-based submission via FTPS (current) with planned migration to Transfer WebService (February 2026).

### Rationale
- ELDA uses custom XML formats per DM-ORG specification, not standard SOAP
- New Transfer WebService launching February 2026 provides modern API
- FTPS interface (ftps.elda.at) is current production method
- XML format defined by Hauptverband specification v42.3.0

### Technical Details

**Endpoints:**
- Production FTPS: `ftps.elda.at` (Port 21, Passive Mode)
- Test: `https://online-test.elda.at/cgi/webtrans.cgi`
- New WebService (Feb 2026): Documentation at elda.at/cdscontent/?contentid=10007.838875

**Authentication:**
- ID Austria (primary) or EU Login via eIDAS
- ELDA Seriennummer + BKNR linked via USP portal
- API key for new WebService (request from ELDA Competence Center)

**Message Types:**
- Anmeldung: Employee registration
- Abmeldung: Employee deregistration
- Änderungsmeldung: Change notifications
- mBGM: Monthly contribution basis (mandatory since 2019)

**SV-Nummer Validation Algorithm:**
```
Format: 10 digits (4-digit sequence + 6-digit DOB as DDMMYY)
Weights (reversed): [6, 1, 2, 4, 8, 5, 0, 9, 7, 3]
Checksum = sum(digit[i] * weight[i]) mod 11
Valid if checksum != 10; position 4 is check digit
```

### Alternatives Considered
- ELDA Batch Mode (Windows-only, requires eldawin.exe) - Rejected
- Manual portal entry - Rejected (not automatable)

---

## 2. Firmenbuch (Company Registry) Integration

### Decision
Use official JustizOnline SOAP API with X-API-KEY authentication.

### Rationale
- Official data source from Austrian Ministry of Justice
- Legal validity for extracts
- Comprehensive company data including directors, capital, insolvency status
- Well-defined SOAP operations

### Technical Details

**Endpoint:**
```
https://justizonline.gv.at/jop/api/at.gv.justiz.fbw/ws/fbw.wsdl
```

**Authentication:**
```
Header: X-API-KEY: your_api_key_here
```

**SOAP Operations:**
1. `SucheFirmaRequest` - Company search by name
2. `AuszugRequest` - Company extract (Firmenbuchauszug)
3. `UrkundeRequest` - Document retrieval
4. `VeraenderungenFirmaRequest` - Change history

**FN-Nummer Format:**
```
Pattern: [FN]? [0-9]{1,6} [a-z]
Example: FN 123456 b
Validation: Regex ^(FN\s?)?[0-9]{1,6}\s?[a-z]$
```

**Response Data:**
- Company name, legal form, registered office
- Directors (Geschäftsführer) with DOB, addresses
- Share capital, business sector
- Insolvency status, branch offices
- UID number, founding date

### Alternatives Considered
- Third-party REST APIs (api.auszug.at) - Considered as fallback
- data.gv.at HVD bulk download - For analytics only, not real-time
- Screen scraping - Rejected (ToS violation)

---

## 3. FinanzOnline File Upload (UVA/ZM)

### Decision
Use SOAP-based fileUploadService with existing session authentication.

### Rationale
- Official BMF method for UVA and ZM submission
- Reuses session from existing FinanzOnline login
- Single endpoint for both document types
- Well-documented XML formats with XSD schemas

### Technical Details

**Endpoint:**
```
https://finanzonline.bmf.gv.at/fon/ws/fileuploadService.wsdl
```

**Request Structure:**
```xml
<fon:fileuploadRequest>
  <fon:tid>123456789</fon:tid>
  <fon:benid>WEBSERVICE</fon:benid>
  <fon:id>SESSION_ID</fon:id>
  <fon:art>U30</fon:art>          <!-- U30=UVA, U13=ZM -->
  <fon:uebermittlung>P</fon:uebermittlung>  <!-- T=Test, P=Prod -->
  <fon:data><![CDATA[BASE64_XML]]></fon:data>
</fon:fileuploadRequest>
```

**UVA (U30) Key Kennzahlen:**
- KZ000: Total deliveries
- KZ017: 20% tax base
- KZ018: 10% tax base
- KZ019: 13% tax base
- KZ060: Input tax deduction
- KZ095: Tax liability/credit

**ZM (U13) Structure:**
- Quarterly only (Q1-Q4)
- Partner UID, country code, transaction type (L/D/S), amount
- Art codes: L=Goods, D=Triangular, S=Services

**Response Codes:**
- 0: Success
- -1: WebService error
- -2: Session invalid (re-login required)
- -3: XML validation failed
- -4: Tax number not authorized

### Alternatives Considered
- Manual portal upload - Rejected (not automated)
- Third-party accounting software - Rejected (vendor lock-in)

---

## 4. E-Rechnung (E-Invoice) Standards

### Decision
Prioritize UBL with EN16931 compliance; ZUGFeRD/CII as secondary format.

### Rationale
- Austria uses ebInterface and UBL/Peppol for B2G
- Germany mandates XRechnung (UBL or CII) for B2B from 2025
- UBL is dominant EU/global syntax via Peppol network
- ZUGFeRD provides hybrid PDF+XML for human readability

### Technical Details

**EN16931 Syntaxes:**
- UBL 2.1: `urn:oasis:names:specification:ubl:schema:xsd:Invoice-2`
- CII D16B: UN/CEFACT Cross Industry Invoice

**Austrian Tax Rates (UNCL 5305 codes):**
| Rate | Code | Usage |
|------|------|-------|
| 20% | S | Standard rate |
| 13% | S | Reduced (hotels, wine, cultural) |
| 10% | S | Reduced (food, books, transport) |
| 0% | Z | Zero-rated (exports) |
| Exempt | E | Financial services |
| Reverse | AE | VAT reverse charge |

**ZUGFeRD PDF/A-3 Embedding:**
- PDF standard: PDF/A-3 (ISO 19005-3)
- XML file: `factur-x.xml` (ZUGFeRD 2.1)
- Embedded via `/AF` key with `/AFRelationship = Source`

**Validation:**
- Schematron files: github.com/ConnectingEurope/eInvoicing-EN16931
- Latest: v1.3.15 (2025-10-20)

**Go Libraries:**
- XML generation: Custom structs with `encoding/xml`
- PDF embedding: Ghostscript for PDF/A-3 conversion
- Format conversion: `github.com/invopop/gobl.xinvoice` (optional)

### Alternatives Considered
- ebInterface only - Limited international acceptance
- Commercial UniPDF - High cost ($1500+)
- Pure XRechnung - Not human-readable

---

## 5. SEPA Banking Standards

### Decision
Use pain.001.001.09, pain.008.001.08, camt.053.001.08 with fallback to v03/v02.

### Rationale
- New versions mandatory by November 2026
- Structured addresses required from Nov 15, 2026
- Enhanced features: UETR, LEI, DateTime precision
- Legacy versions supported until transition complete

### Technical Details

**Namespaces:**
```xml
<!-- Credit Transfer -->
pain.001.001.03: urn:iso:std:iso:20022:tech:xsd:pain.001.001.03
pain.001.001.09: urn:iso:std:iso:20022:tech:xsd:pain.001.001.09

<!-- Direct Debit -->
pain.008.001.02: urn:iso:std:iso:20022:tech:xsd:pain.008.001.02
pain.008.001.08: urn:iso:std:iso:20022:tech:xsd:pain.008.001.08

<!-- Bank Statement -->
camt.053.001.02: urn:iso:std:iso:20022:tech:xsd:camt.053.001.02
camt.053.001.08: urn:iso:std:iso:20022:tech:xsd:camt.053.001.08
```

**Austrian IBAN Format:**
```
AT kk bbbb bccc cccc cccc
   │  └─┬─┘└─────┬──────┘
   │   BLZ    Account Number
   Check Digits
```

**IBAN Validation (ISO 7064 MOD 97-10):**
1. Check length (Austria: 20 chars)
2. Move first 4 chars to end
3. Convert letters to numbers (A=10...Z=35)
4. Calculate: number MOD 97
5. Valid if remainder = 1

**Austrian Specifics:**
- BIC not required since Feb 2016 for SEPA in EEA
- STUZZA/PSA define national adaptations
- Creditor ID format: AT + check + ZZZ + 11-digit OeNB ID

**Go Libraries:**
- IBAN validation: `github.com/jbub/banking/iban`
- ISO 7064: `github.com/digitorus/iso7064`

### Alternatives Considered
- Legacy formats only - Deprecated Nov 2026
- MT940 statements - Being replaced by camt.053

---

## 6. MCP Server Extension

### Decision
Extend existing MCP server with tools for all modules using mark3labs/mcp-go.

### Rationale
- MCP server already implemented with 10 tools
- Consistent JSON-RPC interface for all services
- Enables AI-powered workflows
- No new dependencies required

### Technical Details

**New Tools to Add:**
| Tool | Module | Purpose |
|------|--------|---------|
| fo-elda-register | ELDA | Submit employee registration |
| fo-elda-status | ELDA | Query registration status |
| fo-fb-search | Firmenbuch | Search companies |
| fo-fb-extract | Firmenbuch | Get company details |
| fo-uva-submit | FinanzOnline | Submit UVA |
| fo-zm-submit | FinanzOnline | Submit ZM |
| fo-sepa-pain001 | SEPA | Generate credit transfer |
| fo-sepa-pain008 | SEPA | Generate direct debit |
| fo-invoice-create | E-Rechnung | Generate invoice |
| fo-invoice-validate | E-Rechnung | Validate invoice |

**Response Format:**
All tools return structured JSON with:
- `success`: boolean
- `data`: result object
- `error`: error message if failed

---

## Summary of Decisions

| Module | Approach | Key Dependency |
|--------|----------|----------------|
| ELDA | XML via FTPS → WebService (2026) | None (stdlib) |
| Firmenbuch | SOAP API with X-API-KEY | None (stdlib) |
| FinanzOnline | fileUploadService SOAP | Existing session |
| E-Rechnung | UBL primary, ZUGFeRD secondary | Ghostscript (optional) |
| SEPA | pain.001.001.09, camt.053.001.08 | github.com/jbub/banking |
| MCP | Extend existing server | mark3labs/mcp-go |

**Constitution Compliance:**
- Minimal Dependencies: Only stdlib + existing deps + optional Ghostscript
- No new abstractions: Extending existing patterns from fonws module
- Test-first: All modules will have unit tests before implementation
