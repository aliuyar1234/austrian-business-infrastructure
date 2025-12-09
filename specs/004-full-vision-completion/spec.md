# Feature Specification: Full Vision Completion

**Feature Branch**: `004-full-vision-completion`
**Created**: 2025-12-07
**Status**: Draft
**Input**: User description: "the remaining phases till the full vision (all 5 modules fully) are operational"

## Overview

This specification covers the completion of all 5 core modules to achieve the full Austrian Business Infrastructure vision: a comprehensive open-source SDK for Austrian government and business system integrations.

**Current State**:
- Module 1 (FinanzOnline): MVP complete - session, databox, multi-account
- Module 2 (ELDA): Skeleton only
- Module 3 (Firmenbuch): Skeleton only
- Module 4 (E-Rechnung): Core logic exists, needs CLI integration
- Module 5 (SEPA): Core logic exists, needs CLI integration

**Target State**: All 5 modules fully operational with CLI commands, MCP tools, and comprehensive functionality.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tax Accountant Multi-Service Dashboard (Priority: P1)

A tax accountant managing 30+ Austrian companies needs a unified dashboard to check FinanzOnline databox, validate UIDs, submit UVA returns, and monitor company registry changes - all from one CLI tool.

**Why this priority**: This is the core value proposition - saving hours of manual login/checking across multiple government portals. Directly addresses the "2.5 hours daily just for logins" pain point.

**Independent Test**: Can be fully tested by running `fo dashboard --all` and verifying it displays status from all configured services (FinanzOnline accounts, pending UVA deadlines, company registry alerts).

**Acceptance Scenarios**:

1. **Given** a user has configured 30 FinanzOnline accounts and 10 Firmenbuch watch items, **When** they run `fo dashboard --all`, **Then** they see a consolidated view of all pending items within 30 seconds
2. **Given** a user has new documents in 3 accounts, **When** the dashboard runs, **Then** it highlights which accounts need attention with item counts
3. **Given** a network timeout occurs on one service, **When** the dashboard runs, **Then** it shows results from successful services and marks failed ones with error status

---

### User Story 2 - ELDA Employee Registration (Priority: P1)

An HR manager needs to electronically register new employees with Austrian social insurance (ÖGK) without logging into the ELDA web portal manually.

**Why this priority**: ELDA registration is legally required within 7 days of employment start. Manual process is time-consuming and error-prone. High-value automation target.

**Independent Test**: Can be fully tested by running `fo elda anmeldung` with employee data and verifying the registration confirmation is received.

**Acceptance Scenarios**:

1. **Given** valid employee data (name, SV-Nummer, start date, employer VSNR), **When** user runs `fo elda anmeldung --employee-file data.json`, **Then** registration is submitted and confirmation number is returned
2. **Given** an invalid SV-Nummer, **When** user attempts registration, **Then** system rejects with clear error message before submission
3. **Given** successful registration, **When** user queries status, **Then** system shows registration status with timestamp

---

### User Story 3 - Firmenbuch Company Due Diligence (Priority: P2)

A compliance officer needs to perform due diligence on potential business partners by querying the Austrian company registry (Firmenbuch) for company details, ownership structure, and insolvency status.

**Why this priority**: Critical for KYC/AML compliance and business risk assessment. Currently requires manual portal access or expensive third-party services.

**Independent Test**: Can be fully tested by running `fo fb search "Company Name"` and `fo fb extract FN123456x` to retrieve company details.

**Acceptance Scenarios**:

1. **Given** a company name search term, **When** user runs `fo fb search "Muster GmbH"`, **Then** system returns matching companies with FN numbers
2. **Given** a valid Firmenbuch number, **When** user runs `fo fb extract FN123456x`, **Then** system returns company details (name, address, directors, capital, founding date)
3. **Given** a company with insolvency proceedings, **When** user extracts details, **Then** insolvency status is clearly indicated

---

### User Story 4 - E-Invoice Generation and Validation (Priority: P2)

A business owner needs to generate EN16931-compliant e-invoices (XRechnung/ZUGFeRD) for B2B customers and validate incoming invoices against the standard.

**Why this priority**: EU B2B e-invoice mandate is approaching. Businesses need tools to generate and validate compliant invoices without expensive ERP upgrades.

**Independent Test**: Can be fully tested by running `fo erechnung create --input invoice.json --format xrechnung` and validating the output passes EN16931 compliance checks.

**Acceptance Scenarios**:

1. **Given** invoice data in JSON format, **When** user runs `fo erechnung create`, **Then** system generates valid XRechnung XML
2. **Given** an existing PDF invoice, **When** user runs `fo erechnung embed --pdf invoice.pdf --data invoice.json`, **Then** system creates ZUGFeRD PDF with embedded XML
3. **Given** an incoming XRechnung file, **When** user runs `fo erechnung validate invoice.xml`, **Then** system reports compliance status with specific errors if any

---

### User Story 5 - SEPA Payment File Generation (Priority: P2)

A finance manager needs to generate SEPA payment files (pain.001 for credit transfers, pain.008 for direct debits) from their accounting data for bank upload.

**Why this priority**: Every business needs to make payments. SEPA XML generation is tedious and error-prone manually. Banks require specific formats.

**Independent Test**: Can be fully tested by running `fo sepa pain001 --input payments.json` and uploading the resulting XML to a bank portal.

**Acceptance Scenarios**:

1. **Given** a list of payments in JSON format, **When** user runs `fo sepa pain001 --input payments.json`, **Then** system generates valid pain.001.001.03 XML
2. **Given** payment data with invalid IBAN, **When** user generates payment file, **Then** system rejects with clear validation error
3. **Given** a bank statement in camt.053 format, **When** user runs `fo sepa parse statement.xml`, **Then** system extracts transactions into JSON format

---

### User Story 6 - UVA Submission via CLI (Priority: P1)

A tax accountant needs to submit monthly/quarterly VAT advance returns (Umsatzsteuervoranmeldung) directly through the CLI without using the FinanzOnline web portal.

**Why this priority**: UVA submission is a recurring monthly/quarterly task for all VAT-registered businesses. Direct submission saves significant time.

**Independent Test**: Can be fully tested by running `fo uva submit --period 2025-01 --account "Company GmbH"` with prepared UVA data.

**Acceptance Scenarios**:

1. **Given** UVA data for a period, **When** user runs `fo uva submit`, **Then** system submits to FinanzOnline and returns confirmation
2. **Given** a submission with calculation errors, **When** user submits, **Then** FinanzOnline validation errors are returned clearly
3. **Given** successful submission, **When** user queries status, **Then** system shows submission timestamp and reference number

---

### User Story 7 - MCP Server for AI Integration (Priority: P3)

A developer building AI-powered business automation needs MCP tools to access all Austrian business infrastructure services programmatically through Claude or other AI assistants.

**Why this priority**: AI-first workflows are the differentiator. MCP integration enables natural language interaction with all services.

**Independent Test**: Can be fully tested by starting `fo mcp serve` and calling tools via JSON-RPC from an MCP client.

**Acceptance Scenarios**:

1. **Given** the MCP server is running, **When** a client calls `fo-uid-validate` tool, **Then** UID validation result is returned in structured JSON
2. **Given** configured FinanzOnline credentials, **When** client calls `fo-databox-list`, **Then** databox contents are returned
3. **Given** ELDA credentials configured, **When** client calls `fo-elda-status`, **Then** pending registrations are returned

---

### Edge Cases

- What happens when ELDA API is in maintenance window (typically weekends)?
- How does system handle expired FinanzOnline sessions mid-batch operation?
- What happens when Firmenbuch returns no results for a valid search?
- How does system handle rate limiting from government APIs?
- What happens when ZUGFeRD PDF embedding fails due to PDF format issues?
- How does system handle SEPA payments with special characters in beneficiary names?

---

## Requirements *(mandatory)*

### Functional Requirements

#### ELDA Module (Social Insurance)

- **FR-001**: System MUST support employee registration (Anmeldung) submission to ELDA
- **FR-002**: System MUST support employee deregistration (Abmeldung) submission
- **FR-003**: System MUST support change notifications (Änderungsmeldung)
- **FR-004**: System MUST validate SV-Nummer (social security number) before submission
- **FR-005**: System MUST store ELDA credentials encrypted alongside FinanzOnline credentials
- **FR-006**: System MUST support querying registration status
- **FR-007**: System MUST handle ELDA error codes and return meaningful messages

#### Firmenbuch Module (Company Registry)

- **FR-008**: System MUST support company search by name
- **FR-009**: System MUST support company detail extraction by FN number
- **FR-010**: System MUST return company directors and authorized signatories
- **FR-011**: System MUST indicate insolvency status when present
- **FR-012**: System MUST support company monitoring (watch list)
- **FR-013**: System MUST store Firmenbuch API credentials encrypted

#### E-Rechnung Module (E-Invoice)

- **FR-014**: System MUST generate XRechnung XML compliant with EN16931
- **FR-015**: System MUST generate ZUGFeRD PDFs with embedded XML
- **FR-016**: System MUST validate incoming invoices against EN16931
- **FR-017**: System MUST support all Austrian tax rates (20%, 13%, 10%, 0%)
- **FR-018**: System MUST handle reverse charge scenarios
- **FR-019**: System MUST extract invoice data from existing ZUGFeRD PDFs

#### SEPA Module (Banking)

- **FR-020**: System MUST generate pain.001.001.03 credit transfer files
- **FR-021**: System MUST generate pain.008.001.02 direct debit files
- **FR-022**: System MUST parse camt.053 bank statements
- **FR-023**: System MUST validate IBAN with checksum verification
- **FR-024**: System MUST validate BIC format
- **FR-025**: System MUST support Austrian bank code (BLZ) to BIC lookup

#### FinanzOnline Completion

- **FR-026**: System MUST support UVA submission via fileUploadService
- **FR-027**: System MUST support ZM (Zusammenfassende Meldung) submission
- **FR-028**: System MUST support batch operations across multiple accounts
- **FR-029**: System MUST provide unified dashboard showing all services

#### MCP Integration

- **FR-030**: System MUST expose all validation tools via MCP (UID, IBAN, BIC, SV-Nummer, FN)
- **FR-031**: System MUST expose databox operations via MCP
- **FR-032**: System MUST expose Firmenbuch search/extract via MCP
- **FR-033**: System MUST expose UVA submission via MCP
- **FR-034**: System MUST expose ELDA registration status via MCP

#### CLI & Infrastructure

- **FR-035**: All modules MUST support JSON output mode for scripting
- **FR-036**: All modules MUST support verbose/debug logging
- **FR-037**: All credentials MUST be stored using AES-256-GCM encryption
- **FR-038**: System MUST handle network timeouts gracefully with retry logic
- **FR-039**: System MUST support configuration via environment variables

### Key Entities

- **Account**: Represents credentials for a service (FinanzOnline, ELDA, Firmenbuch) with type, identifier, and encrypted credentials
- **Employee**: Person being registered/deregistered with ELDA - includes SV-Nummer, name, employment dates, employer reference
- **Company**: Firmenbuch entity with FN number, name, address, directors, capital, status
- **Invoice**: E-invoice with parties, line items, tax calculations, payment terms
- **Payment**: SEPA payment instruction with debtor, creditor, amount, reference
- **Session**: Active authenticated session with a government service, including token and expiry

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can check status across 30 accounts from all services in under 60 seconds
- **SC-002**: ELDA employee registration completes in under 30 seconds (excluding API response time)
- **SC-003**: Firmenbuch company search returns results in under 5 seconds
- **SC-004**: E-invoice generation produces EN16931-compliant output that passes official validation tools
- **SC-005**: SEPA payment file generation supports batches of 1000+ payments
- **SC-006**: All 5 modules have CLI commands accessible via `fo <module> <command>` pattern
- **SC-007**: All validation tools are accessible via MCP server with consistent JSON responses
- **SC-008**: System handles intermittent network failures with automatic retry (up to 3 attempts)
- **SC-009**: Credential encryption/decryption adds less than 100ms overhead
- **SC-010**: Documentation covers all CLI commands with examples

---

## Assumptions

- ELDA SOAP API credentials can be obtained through standard application process
- Firmenbuch API access requires Justizministerium API key (fee-based)
- Users have valid credentials for each service they want to use
- Network connectivity to Austrian government endpoints is available
- PDF processing for ZUGFeRD uses standard PDF/A-3 format

---

## Out of Scope

- REST API wrapper (planned for future phase)
- Web UI / Dashboard (planned for SaaS phase)
- Hosted/cloud version of the tools
- EST/KEST tax return submission (future enhancement)
- Peppol network integration for e-invoices
- Real-time company monitoring alerts (webhook-based)
