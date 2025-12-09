# Feature Specification: MCP Tools Expansion

**Feature Branch**: `003-mcp-tools-expansion`
**Created**: 2025-12-07
**Status**: Draft
**Input**: User description: "the missing MCP tools (databox-list, databox-download, uva-submit, fb-search, fb-extract)"

## Overview

The MCP (Model Context Protocol) server currently exposes 5 validation tools (UID, IBAN, BIC, SV-Nummer, FN). This specification adds 5 additional tools that enable AI assistants to perform operational tasks: listing and downloading databox documents, submitting UVA returns, and searching/extracting Firmenbuch company data.

These tools transform the MCP server from a validation-only service to a full operational assistant capable of completing end-to-end business workflows.

---

## User Scenarios & Testing

### User Story 1 - Databox Document Access (Priority: P1)

An AI assistant helping a tax advisor needs to check for new documents in a client's FinanzOnline databox and download relevant tax notices for review.

**Why this priority**: Databox access is the most frequently needed operation. Tax advisors check databoxes daily for multiple clients. Enabling AI assistants to list and retrieve documents eliminates manual portal navigation.

**Independent Test**: AI assistant can call `fo-databox-list` to see available documents and `fo-databox-download` to retrieve a specific document's content.

**Acceptance Scenarios**:

1. **Given** a configured FinanzOnline account, **When** AI calls `fo-databox-list` with the account identifier, **Then** the tool returns a list of documents with ID, date, subject, and read status
2. **Given** a document ID from the list, **When** AI calls `fo-databox-download` with that ID, **Then** the tool returns the document content and metadata
3. **Given** a non-existent document ID, **When** AI calls `fo-databox-download`, **Then** the tool returns an error with clear explanation
4. **Given** no active session, **When** AI calls either databox tool, **Then** the tool returns an authentication error with guidance

---

### User Story 2 - Firmenbuch Company Lookup (Priority: P2)

An AI assistant helping with due diligence needs to search for companies in the Austrian company register and retrieve detailed company information.

**Why this priority**: Company lookups are essential for KYC/AML compliance, business partner verification, and M&A research. Enabling AI to perform these searches accelerates due diligence workflows.

**Independent Test**: AI assistant can call `fo-fb-search` with a company name and receive matching results, then call `fo-fb-extract` to get full details.

**Acceptance Scenarios**:

1. **Given** a company name, **When** AI calls `fo-fb-search` with that name, **Then** the tool returns a list of matching companies with FN number, name, and legal form
2. **Given** a valid Firmenbuch number (FN), **When** AI calls `fo-fb-extract` with that FN, **Then** the tool returns structured company data including directors, capital, and shareholders
3. **Given** a search query with no matches, **When** AI calls `fo-fb-search`, **Then** the tool returns an empty result set with appropriate message
4. **Given** an invalid FN format, **When** AI calls `fo-fb-extract`, **Then** the tool returns a validation error explaining the correct format

---

### User Story 3 - UVA Submission (Priority: P3)

An AI assistant helping prepare tax filings needs to submit a completed UVA (VAT advance return) to FinanzOnline on behalf of a client.

**Why this priority**: While important, UVA submission is a sensitive operation with legal implications. It builds on the existing UVA generation capability but requires careful authorization. Less frequent than lookups but high value when needed.

**Independent Test**: AI assistant can call `fo-uva-submit` with prepared UVA data and receive a submission confirmation or validation errors.

**Acceptance Scenarios**:

1. **Given** valid UVA data (period, amounts, account), **When** AI calls `fo-uva-submit`, **Then** the tool submits the UVA and returns a reference number
2. **Given** UVA data with validation errors, **When** AI calls `fo-uva-submit`, **Then** the tool returns detailed validation errors before submission
3. **Given** an unauthorized account, **When** AI calls `fo-uva-submit`, **Then** the tool returns an authorization error
4. **Given** a successful submission, **When** the operation completes, **Then** the tool returns the official submission reference and timestamp

---

### Edge Cases

- What happens when the FinanzOnline session expires mid-operation? → Tool returns session expiry error with instruction to re-authenticate
- How does the system handle rate limits on Firmenbuch queries? → Tool tracks rate limits and returns informative error when exceeded
- What if databox contains very large documents (>10MB)? → Tool returns document metadata with size warning; download may be chunked or declined
- How are concurrent MCP tool calls handled? → Each call is stateless; multiple clients can call simultaneously
- What happens if UVA submission succeeds but confirmation retrieval fails? → Tool returns partial success with submission reference and warning about confirmation status

---

## Requirements

### Functional Requirements

#### Databox Tools

- **FR-001**: System MUST expose `fo-databox-list` tool that returns documents from a FinanzOnline databox
- **FR-002**: System MUST expose `fo-databox-download` tool that retrieves a specific document by ID
- **FR-003**: Databox tools MUST accept account identifier to specify which account to query
- **FR-004**: Databox list MUST return document ID, date, subject, sender, and read status for each entry
- **FR-005**: Databox download MUST return document content, format type, and metadata

#### Firmenbuch Tools

- **FR-006**: System MUST expose `fo-fb-search` tool that searches companies by name or criteria
- **FR-007**: System MUST expose `fo-fb-extract` tool that retrieves full company details by FN number
- **FR-008**: Search results MUST include FN number, company name, legal form, and registered address
- **FR-009**: Extract results MUST include directors, shareholders, capital, and registration dates
- **FR-010**: FN input MUST be validated before querying external services

#### UVA Submission Tool

- **FR-011**: System MUST expose `fo-uva-submit` tool that submits UVA to FinanzOnline
- **FR-012**: UVA submission MUST validate all data before attempting submission
- **FR-013**: Successful submission MUST return the official reference number
- **FR-014**: Failed submission MUST return detailed error information including field-level errors
- **FR-015**: UVA tool MUST require explicit account authorization before submission

#### Cross-Cutting Requirements

- **FR-016**: All tools MUST return structured data suitable for AI processing
- **FR-017**: All tools MUST include clear error messages with actionable guidance
- **FR-018**: All tools MUST be discoverable via `fo mcp tools` command
- **FR-019**: All tools MUST respect existing authentication and authorization mechanisms
- **FR-020**: Tools MUST NOT expose sensitive credentials in responses or error messages

---

### Key Entities

- **MCP Tool**: A callable function exposed via the MCP protocol with name, description, input schema, and handler
- **Databox Document**: A document in FinanzOnline with ID, date, subject, sender, content, and read status
- **Firmenbuch Entry**: A company record with FN number, name, legal form, address, directors, shareholders, and capital
- **UVA Submission**: A VAT advance return with period, tax amounts by category, and submission status
- **Tool Result**: Structured response from any MCP tool containing success status, data payload, and any errors

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: AI assistants can list databox documents for any configured account within 5 seconds
- **SC-002**: AI assistants can download any databox document and receive its content within 10 seconds
- **SC-003**: AI assistants can search for companies and receive results within 3 seconds
- **SC-004**: AI assistants can retrieve full company details within 5 seconds of providing FN
- **SC-005**: AI assistants can submit a valid UVA and receive confirmation within 30 seconds
- **SC-006**: 100% of tool responses include clear success/error indication parseable by AI
- **SC-007**: All 5 new tools appear in `fo mcp tools` output with accurate descriptions
- **SC-008**: Tools work correctly when called via any MCP-compatible client (Claude Desktop, etc.)
- **SC-009**: Error messages are specific enough for AI to provide helpful guidance to users
- **SC-010**: No sensitive data (passwords, tokens, PINs) appears in any tool response

---

## Assumptions

1. FinanzOnline account credentials are already configured and stored securely via existing `fo account` commands
2. The existing MCP server infrastructure (stdio transport, JSON-RPC protocol) is stable and reusable
3. Firmenbuch queries use the existing `fb` module's client implementation
4. UVA submission uses the existing `fonws` FileUploadService implementation
5. AI clients will handle authentication prompts when sessions are not active
6. Rate limits for external services are managed at the client level, not MCP protocol level

---

## Dependencies

- Existing MCP server implementation (`internal/mcp/server.go`)
- Existing databox service (`internal/fonws/databox.go`)
- Existing Firmenbuch client (`internal/fb/client.go`)
- Existing UVA submission service (`internal/fonws/uva.go`)
- Existing account and session management (`internal/store/accounts.go`, `internal/fonws/session.go`)

---

## Out of Scope

- Real-time notifications or webhooks for new databox documents
- Batch operations across multiple accounts in a single tool call
- Document format conversion (PDF to text, etc.)
- Firmenbuch monitoring/watchlist functionality via MCP
- ELDA or E-Rechnung MCP tools (future enhancement)
- Authentication/login operations via MCP (security consideration)
