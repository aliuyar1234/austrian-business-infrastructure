# Research: Austrian Business Infrastructure - Complete Product Suite

**Feature**: 002-vision-completion-roadmap
**Date**: 2025-12-07
**Status**: Complete

## Executive Summary

This research documents the technical landscape for extending the Austrian Business Infrastructure CLI suite beyond the FinanzOnline basis (Spec 001) to include UVA/UID/ZM extensions, ELDA integration, Firmenbuch queries, E-Rechnung generation, SEPA file creation, and MCP server integration.

**Key Finding**: All required APIs are well-documented with SOAP/XML interfaces. The existing Go codebase architecture (SOAP client, encrypted store, Cobra CLI) can be extended without major changes.

---

## Module Research

### 1. FinanzOnline Extensions (UVA, UID, ZM)

#### 1.1 UID-Abfrage WebService

**WSDL**: `https://finanzonline.bmf.gv.at/fonuid/ws/uidAbfrageService.wsdl`

**Operations**:
- UID validation against Austrian/EU database
- Returns company name, address, validity status

**Rate Limits** (since April 2023):
- Each UID can only be queried **twice per day per participant** via webservice
- Return code 1513 if limit exceeded
- No limit for manual queries in FinanzOnline portal

**Alternative EU VIES**:
- WSDL: `http://ec.europa.eu/taxation_customs/vies/services/checkVatService.wsdl`
- No rate limits, supports all EU countries
- Consider as fallback for batch validation

**Implementation Approach**:
- Extend existing `internal/fonws/` module with `uid.go`
- Use existing session authentication
- Add rate limit tracking per UID per day

#### 1.2 File-Upload WebService (UVA/ZM)

**WSDL**: Documented in BMF File-Upload-Webservice specification

**Supported File Types**:
- U30 (Umsatzsteuervoranmeldung)
- ZM (Zusammenfassende Meldung)
- Various tax declarations

**Prerequisites**:
- WebService user (separate from regular FinanzOnline user)
- All requests must be UTF-8 encoded
- Session established via Session WebService

**Submission Flow**:
1. Login via Session WebService
2. Upload XML file via File-Upload Service
3. Receive confirmation number
4. Check status via status query

**UVA Deadlines**:
- Monthly: 15th of second following month
- Quarterly: May 15, Aug 15, Nov 15, Feb 15

**Implementation Approach**:
- Add `uva.go` and `zm.go` to `internal/fonws/`
- Generate XML from structured data (Go encoding/xml)
- Validate against BMF XML schema before upload

#### 1.3 Session WebService (Existing)

**WSDL**: `https://finanzonline.bmf.gv.at/fonws/ws/sessionService.wsdl`

Already implemented in Spec 001. SessionID used for all other services.

---

### 2. ELDA - Social Security Integration

**Portal**: [elda.at](https://www.elda.at)
**Documentation**: DM-Org Version 41.1.0 (September 2024)

#### 2.1 System Overview

ELDA (Elektronischer Datenaustausch mit den österreichischen Sozialversicherungsträgern) is the mandatory electronic reporting system for:
- Employee registrations (AN-Meldung)
- Employee deregistrations (AB-Meldung)
- Change notifications
- Annual wage statements (L16)
- Contribution proofs

**Key Requirement**: Paper/email/phone notifications are **not valid**. Only ELDA submissions count.

#### 2.2 Registration

- One-time registration at [elda.at](https://www.elda.at/cdscontent/?contentid=10007.838915&portal=eldaportal)
- Separate credentials from FinanzOnline
- Employer account (Dienstgeberkonto) required

#### 2.3 Technical Details

**Transmission Methods**:
- Online via ELDA portal
- File upload via ELDA software
- API integration via DM-Org specification

**Data Format**: XML according to DM-Org specification

**Validation**: Pre-validation before submission, detailed error messages

#### 2.4 2025/2026 Changes

- New L16 field for employee bonus (§ 124b Z 478)
- New version mandatory from 01.02.2026

**Implementation Approach**:
- New `internal/elda/` module
- Separate SOAP client (different endpoint)
- Separate credential storage (different namespace in store)
- Focus on: Anmeldung, Abmeldung, Status query

**Competence Center**: 05 0766-14502700 (technical support)

---

### 3. Firmenbuch - Company Register

**Official Portal**: [justiz.gv.at](https://www.justiz.gv.at/service/datenbanken/firmenbuch.36f.de.html)

#### 3.1 SOAP API

**WSDL**: `https://justizonline.gv.at/jop/api/at.gv.justiz.fbw/ws/fbw.wsdl`

**Authentication**:
- Header: `X-API-KEY: [your_issued_key]`
- Header: `Content-Type: application/soap+xml; charset=utf-8`

**Protocol**: SOAP 1.2 (default), SOAP 1.1 supported

#### 3.2 Available Operations

| Operation | Description |
|-----------|-------------|
| SUCHEFIRMAREQUEST | Search by company name (fuzzy/exact) |
| AUSZUGREQUEST | Get detailed extract by FN number |
| URKUNDEREQUEST | Fetch official documents (PDF) |
| VERAENDERUNGENFIRMAREQUEST | Get modification history |

#### 3.3 Response Structure

AUSZUGRESPONSE contains:
- FIRMA: Main company details
- FUN: Officers/representatives
- PER: Persons
- ZWL: Branches
- VOLLZ: Registration acts
- IDENT: Identifiers
- KUR: Insolvency information

#### 3.4 Access & Costs

**Verrechnungsstellen** (authorized providers):
- api.auszug.at - State-authorized partner since 2015
- Wirtschafts-Compass API (hfdata.at) - Largest Austrian provider

**Costs**: ~15€ per extract (regulated by GGG)

**High Value Datasets**: EU regulation requires free access to certain data (under discussion with BMJ)

**Implementation Approach**:
- New `internal/fb/` module
- REST/SOAP hybrid (API key auth)
- Support FN search, extract, monitoring
- Consider watchlist feature for due diligence

---

### 4. E-Rechnung - Electronic Invoicing

#### 4.1 Standards Overview

**EN 16931**: European standard for e-invoice data exchange
- CEN (European Committee for Standardization)
- Mandatory for B2G in EU

**ZUGFeRD**: German/EU hybrid format
- Current version: 2.3.3 (May 7, 2025)
- Embeds UN/CEFACT-XML in PDF/A-3
- Profiles: Minimum, Basic WL, Basic, EN16931, Extended, XRechnung

**XRechnung**: German standard (KoSIT)
- Implements EU Directive 2014/55/EU
- Pure XML format

#### 4.2 Austria Context

- E-invoicing obligation for B2G already active
- B2B obligation planned (following EU trend)
- USP (Unternehmensserviceportal) and Peppol network for delivery

#### 4.3 Validation

**Types**:
- Syntactic: XML Schema (XSD) validation
- Semantic: Business rule validation (Schematron)

**Tools**:
- CEN official tools
- valitool.org
- KoSIT validator

#### 4.4 2025 Requirements

- Germany: E-invoice reception mandatory since Jan 1, 2025
- Austria: Similar timeline expected

**Implementation Approach**:
- New `internal/erechnung/` module
- Use stdlib `encoding/xml` for generation
- Embed XSD schemas for validation
- Support: XRechnung generation, ZUGFeRD generation, validation, PDF extraction

---

### 5. SEPA - Banking Integration

#### 5.1 ISO 20022 Message Types

| Type | Name | Purpose |
|------|------|---------|
| pain.001 | CustomerCreditTransferInitiation | Credit transfers |
| pain.002 | CustomerPaymentStatusReport | Status reports |
| pain.008 | CustomerDirectDebitInitiation | Direct debits |
| camt.052 | BankToCustomerAccountReport | Intraday reports |
| camt.053 | BankToCustomerStatement | Account statements |
| camt.054 | BankToCustomerDebitCreditNotification | Individual notifications |

#### 5.2 Version Requirements (2025-2026)

**Critical Deadlines**:
- **November 2025**: Structured addresses mandatory
- **October 2025**: camt V2 end of life, switch to V8
- **November 2026**: DTAZV format discontinued, pain.001 required
- **November 2026**: Unstructured addresses no longer supported

**Current Versions**:
- pain.001.001.03 (ISO 2009) - extended support until Nov 2026
- camt.053.001.13 (current)

#### 5.3 Key Requirements

- UTF-8 encoding mandatory
- Structured addresses: Street, Building, PostCode, TownName, Country
- IBAN validation with BIC derivation

**Implementation Approach**:
- New `internal/sepa/` module
- Generate pain.001 (credit transfer) - primary use case
- Generate pain.008 (direct debit) - secondary
- Parse camt.053/054 for reconciliation
- IBAN/BIC validation with Austrian bank database
- Use stdlib `encoding/xml`, no external dependencies

---

### 6. MCP Server - AI Integration

#### 6.1 Protocol Overview

**Model Context Protocol (MCP)**: Open standard by Anthropic (Nov 2024)
- Standardizes LLM integration with external tools
- Adopted by OpenAI, Google DeepMind

**Concepts**:
- **Tools**: LLM-callable functions (side effects allowed)
- **Resources**: Read-only data exposure
- **Prompts**: Reusable templates

#### 6.2 Go SDK

**Official SDK**: [github.com/modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk)
- Maintained with Google collaboration
- Stable release: August 2025

**Community SDK**: [github.com/mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- Mature, well-documented
- Simpler API surface

#### 6.3 Transport Options

- **stdio**: For local integrations (recommended for CLI tools)
- **Streamable HTTP**: For remote servers
- **SSE**: Legacy, backwards compatibility

#### 6.4 Planned Tools

| Tool | Module | Description |
|------|--------|-------------|
| fo-databox-list | fonws | List databox entries |
| fo-databox-download | fonws | Download document |
| fo-uid-validate | fonws | Validate UID number |
| fo-uva-submit | fonws | Submit UVA |
| elda-anmeldung | elda | Register employee |
| fb-search | fb | Search company register |
| fb-extract | fb | Get company extract |

**Implementation Approach**:
- New `internal/mcp/` module
- Use official Go SDK (when stable) or mcp-go
- Expose existing CLI functionality as MCP tools
- stdio transport for Claude Desktop integration
- Secure credential handling (prompt for master password)

---

## Technical Recommendations

### 1. Dependency Strategy

| Need | Recommendation | Justification |
|------|----------------|---------------|
| SOAP/XML | stdlib `encoding/xml` | Existing pattern, no new deps |
| HTTP client | stdlib `net/http` | Existing pattern |
| CLI framework | Cobra (existing) | Already in use |
| Parallelism | errgroup (existing) | Already in use |
| MCP | mcp-go or official SDK | Only new dependency required |

### 2. Module Priority

1. **Phase A (P1)**: UVA, UID - High value, low complexity, extends existing
2. **Phase B (P2)**: ELDA, Firmenbuch, E-Rechnung - Medium complexity, new modules
3. **Phase C (P3)**: SEPA, MCP, ZM - Lower priority, specialized use cases

### 3. Risk Mitigation

| Risk | Mitigation |
|------|------------|
| UID rate limits | Implement local cache, warn user |
| ELDA credential separation | Namespace credentials in store |
| Firmenbuch API costs | Cache results, watchlist batching |
| E-Rechnung complexity | Start with XRechnung only |
| SEPA version changes | Target November 2025 format |
| MCP SDK stability | Use mcp-go community SDK initially |

### 4. Testing Strategy

| Module | Test Approach |
|--------|---------------|
| FinanzOnline extensions | Mock SOAP (like Spec 001) |
| ELDA | Mock SOAP, no sandbox available |
| Firmenbuch | Mock SOAP, consider paid sandbox |
| E-Rechnung | Unit tests with XSD validation |
| SEPA | Unit tests with sample files |
| MCP | Integration tests with test client |

---

## Research Sources

### Official Documentation
- [BMF UID-Abfrage WebService](https://www.bmf.gv.at/dam/jcr:e6acfe5b-f4a5-44f6-8a57-28256efdb850/BMF_UID_Abfrage_Webservice_2.pdf)
- [BMF Session WebService](https://www.bmf.gv.at/dam/jcr:570753b2-d511-4194-a03e-33f0ac7371ec/BMF_Session_Webservice_2.pdf)
- [ELDA Portal](https://www.elda.at)
- [Firmenbuch API Documentation](https://github.com/Open-Justiz-Online/companyregister-api-documentation)
- [ÖGK ELDA Documentation](https://www.gesundheitskasse.at/cdscontent/?contentid=10007.904704&portal=oegkdgportal)

### Standards
- [EN 16931 Overview](https://www.d-velop.de/blog/compliance/en-16931/)
- [ZUGFeRD Specification](https://mind-forms.de/knowhow/zugferd/)
- [ISO 20022 Message Definitions](https://www.iso20022.org/iso-20022-message-definitions)
- [Swiss Payment Standards (ISO 20022 reference)](https://www.six-group.com/en/products-services/banking-services/payment-standardization/standards/iso-20022.html)

### MCP
- [MCP Official GitHub](https://github.com/modelcontextprotocol)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [MCP TypeScript SDK](https://github.com/modelcontextprotocol/typescript-sdk)
- [mcp-go Community Implementation](https://github.com/mark3labs/mcp-go)

### Regulatory
- [USP Umsatzsteuer](https://www.usp.gv.at/themen/steuern-finanzen/umsatzsteuer-ueberblick/umsatzsteuervoranmeldung-und-umsatzsteuererklaerung.html)
- [WKO UVA Guide](https://www.wko.at/steuern/umsatzsteuervoranmeldung-umsatzsteuerjahreserklaerung)
- [Justiz Firmenbuch](https://www.justiz.gv.at/service/datenbanken/firmenbuch.36f.de.html)

---

## Next Steps

1. **Phase 1 Design**: Create data-model.md with entities for all modules
2. **Contracts**: Define SOAP request/response structures
3. **Quickstart**: Document getting started flow for each module
4. **Tasks**: Generate implementation tasks following Spec 001 patterns
