# Tasks: Austrian Business Infrastructure - Complete Product Suite

**Input**: Design documents from `/specs/002-vision-completion-roadmap/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Test-First Development per Constitution (Principle IV). Tests MUST fail before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

**Status**: 🟡 NEARLY COMPLETE - Only T045 (UID rate limiting) remains

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US8)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `cmd/fo/`, `internal/`, `tests/` at repository root
- Paths follow existing Go conventions from Spec 001

---

## Phase 1: Setup

**Purpose**: Extend existing project structure for new modules

- [x] T001 Create internal/elda/ directory for ELDA social security module
- [x] T002 [P] Create internal/fb/ directory for Firmenbuch module
- [x] T003 [P] Create internal/erechnung/ directory for E-Rechnung module
- [x] T004 [P] Create internal/sepa/ directory for SEPA module
- [x] T005 [P] Create internal/mcp/ directory for MCP server module
- [x] T006 Add mcp-go dependency `go get github.com/mark3labs/mcp-go` (only new external dependency)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure extensions that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T007 Extend AccountType in internal/store/accounts.go to support "elda" and "firmenbuch" types
- [x] T008 Add ExtendedAccount struct with type-specific fields in internal/store/accounts.go
- [x] T009 [P] Implement SV-Nummer validation function in internal/elda/validation.go
- [x] T010 [P] Implement IBAN validation (ISO 7064 Mod 97) in internal/sepa/iban.go
- [x] T011 [P] Implement Austrian bank code to BIC lookup in internal/sepa/bic.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - UVA-Einreichung (Priority: P1)

**Goal**: Tax advisor can create, validate, and submit VAT advance returns (UVA) to FinanzOnline

**Independent Test**: Run `fo uva submit <account> --file uva.xml`, verify submission confirmation with reference number

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T012 [P] [US1] Write unit test for UVA XML generation in tests/unit/uva_test.go
- [x] T013 [P] [US1] Write unit test for UVA XML validation in tests/unit/uva_test.go
- [x] T014 [P] [US1] Write unit test for FileUpload SOAP request serialization in tests/unit/uva_test.go
- [x] T015 [P] [US1] Write unit test for FileUpload SOAP response parsing in tests/unit/uva_test.go
- [x] T016 [US1] Write integration test for UVA submission flow in tests/unit/uva_test.go

### Implementation for User Story 1

- [x] T017 [P] [US1] Implement UVA struct with XML tags in internal/fonws/uva.go
- [x] T018 [P] [US1] Implement UVAPeriod and UVAStatus types in internal/fonws/uva.go
- [x] T019 [P] [US1] Implement UVADocument struct for BMF XML schema in internal/fonws/uva.go
- [x] T020 [US1] Implement GenerateUVAXML() function in internal/fonws/uva.go
- [x] T021 [US1] Implement ValidateUVA() function in internal/fonws/uva.go
- [x] T022 [P] [US1] Implement FileUploadRequest/Response structs in internal/fonws/uva.go
- [x] T023 [US1] Implement FileUploadService.Upload() SOAP call in internal/fonws/uva.go
- [x] T024 [US1] Implement SubmitUVA() function in internal/fonws/uva.go
- [x] T025 [US1] Implement `fo uva validate` command in internal/cli/uva.go
- [x] T026 [US1] Implement `fo uva submit` command in internal/cli/uva.go
- [x] T027 [US1] Add table output for UVA submission result in internal/cli/uva.go
- [x] T028 [US1] Add JSON output for UVA commands (--json flag) in internal/cli/uva.go
- [x] T029 [US1] Implement `fo uva submit --all` for batch submission in internal/cli/uva.go

**Checkpoint**: User Story 1 complete - can submit UVA to FinanzOnline

---

## Phase 4: User Story 2 - UID-Validierung (Priority: P1)

**Goal**: Validate EU VAT identification numbers with company details

**Independent Test**: Run `fo uid check ATU12345678`, verify company name and address displayed

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T030 [P] [US2] Write unit test for UID validation request serialization in tests/unit/uid_test.go
- [x] T031 [P] [US2] Write unit test for UID validation response parsing in tests/unit/uid_test.go
- [x] T032 [P] [US2] Write unit test for UID format validation in tests/unit/uid_test.go
- [x] T033 [P] [US2] Write unit test for CSV batch processing in tests/unit/uid_test.go
- [x] T034 [US2] Write integration test for UID validation flow in tests/unit/uid_test.go

### Implementation for User Story 2

- [x] T035 [P] [US2] Implement UIDValidationRequest struct in internal/fonws/uid.go
- [x] T036 [P] [US2] Implement UIDValidationResult struct in internal/fonws/uid.go
- [x] T037 [P] [US2] Implement UIDAddress struct in internal/fonws/uid.go
- [x] T038 [US2] Implement UIDService.Validate() SOAP call in internal/fonws/uid.go
- [x] T039 [US2] Implement ValidateUIDFormat() function in internal/fonws/uid.go
- [x] T040 [US2] Implement `fo uid check <uid>` command in internal/cli/uid.go
- [x] T041 [US2] Add table output for UID validation result in internal/cli/uid.go
- [x] T042 [US2] Add JSON output for UID commands (--json flag) in internal/cli/uid.go
- [x] T043 [US2] Implement ParseCSV() for batch processing in internal/fonws/uid.go
- [x] T044 [US2] Implement `fo uid batch <file.csv>` command in internal/cli/uid.go
- [ ] T045 [US2] Add rate limit tracking (2 per UID per day) in internal/fonws/uid.go

**Checkpoint**: User Story 2 complete - can validate UID numbers

---

## Phase 5: User Story 3 - ELDA Dienstnehmer-Anmeldung (Priority: P2)

**Goal**: HR can register new employees with social security (ELDA)

**Independent Test**: Run `elda anmeldung` with employee data, verify reference number returned

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T046 [P] [US3] Write unit test for ELDA Anmeldung XML generation in tests/unit/elda_test.go
- [x] T047 [P] [US3] Write unit test for SV-Nummer validation in tests/unit/elda_test.go
- [x] T048 [P] [US3] Write unit test for ELDA response parsing in tests/unit/elda_test.go
- [x] T049 [US3] Write integration test for ELDA submission flow in tests/integration/elda_test.go

### Implementation for User Story 3

- [x] T050 [P] [US3] Implement ELDAKopf struct in internal/elda/types.go
- [x] T051 [P] [US3] Implement ELDAAnmeldung struct in internal/elda/types.go
- [x] T052 [P] [US3] Implement ELDABeschaeftigung struct in internal/elda/types.go
- [x] T053 [P] [US3] Implement ELDAEntgelt struct in internal/elda/types.go
- [x] T054 [P] [US3] Implement ELDAResponse struct in internal/elda/types.go
- [x] T055 [US3] Implement ELDAClient SOAP client in internal/elda/client.go
- [x] T056 [US3] Implement GenerateAnmeldungXML() in internal/elda/anmeldung.go
- [x] T057 [US3] Implement SubmitAnmeldung() in internal/elda/anmeldung.go
- [x] T058 [US3] Implement `elda anmeldung` command in internal/cli/elda.go
- [x] T059 [US3] Add table output for ELDA submission in internal/cli/elda.go
- [x] T060 [US3] Add JSON output for ELDA commands (--json flag) in internal/cli/elda.go
- [x] T061 [P] [US3] Implement ELDAAbmeldung struct in internal/elda/types.go
- [x] T062 [US3] Implement SubmitAbmeldung() in internal/elda/abmeldung.go
- [x] T063 [US3] Implement `elda abmeldung` command in internal/cli/elda.go
- [x] T064 [US3] Implement QueryStatus() in internal/elda/status.go
- [x] T065 [US3] Implement `elda status <reference>` command in internal/cli/elda.go
- [x] T066 [US3] Add ELDA account type support to `fo account add --type elda` in internal/cli/account.go

**Checkpoint**: User Story 3 complete - can register/deregister employees via ELDA

---

## Phase 6: User Story 4 - Firmenbuch-Auszug (Priority: P2)

**Goal**: Query Austrian company register for company information

**Independent Test**: Run `fb extract FN123456a`, verify structured company data displayed

### Tests for User Story 4

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T067 [P] [US4] Write unit test for FBSearchRequest serialization in tests/unit/fb_test.go
- [x] T068 [P] [US4] Write unit test for FBExtractResponse parsing in tests/unit/fb_test.go
- [x] T069 [P] [US4] Write unit test for FN number format validation in tests/unit/fb_test.go
- [x] T070 [US4] Write integration test for Firmenbuch query flow in tests/integration/fb_test.go

### Implementation for User Story 4

- [x] T071 [P] [US4] Implement FBSearchRequest struct in internal/fb/types.go
- [x] T072 [P] [US4] Implement FBSearchResponse struct in internal/fb/types.go
- [x] T073 [P] [US4] Implement FBExtract struct in internal/fb/types.go
- [x] T074 [P] [US4] Implement FBPerson struct in internal/fb/types.go
- [x] T075 [P] [US4] Implement FBGesellschafter struct in internal/fb/types.go
- [x] T076 [US4] Implement FBClient with API key auth in internal/fb/client.go
- [x] T077 [US4] Implement Search() SOAP call in internal/fb/search.go
- [x] T078 [US4] Implement Extract() SOAP call in internal/fb/extract.go
- [x] T079 [US4] Implement `fb search <name>` command in internal/cli/fb.go
- [x] T080 [US4] Implement `fb extract <FN>` command in internal/cli/fb.go
- [x] T081 [US4] Add table output for Firmenbuch results in internal/cli/fb.go
- [x] T082 [US4] Add JSON output for FB commands (--json flag) in internal/cli/fb.go
- [x] T083 [P] [US4] Implement FBWatchlist struct in internal/fb/types.go
- [x] T084 [US4] Implement watchlist persistence in internal/fb/monitor.go
- [x] T085 [US4] Implement `fb watch add/list/remove/check` commands in internal/cli/fb.go
- [x] T086 [US4] Add Firmenbuch account type to `fo account add --type firmenbuch` in internal/cli/account.go

**Checkpoint**: User Story 4 complete - can query and monitor Firmenbuch

---

## Phase 7: User Story 5 - E-Rechnung Erstellung (Priority: P2)

**Goal**: Create EN16931-compliant electronic invoices (XRechnung/ZUGFeRD)

**Independent Test**: Run `erechnung create invoice.json --format xrechnung`, verify valid XML output

### Tests for User Story 5

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T087 [P] [US5] Write unit test for Invoice struct to XRechnung XML in tests/unit/erechnung_test.go
- [x] T088 [P] [US5] Write unit test for Invoice struct to ZUGFeRD XML in tests/unit/erechnung_test.go
- [x] T089 [P] [US5] Write unit test for EN16931 validation rules in tests/unit/erechnung_test.go
- [x] T090 [P] [US5] Write unit test for invoice calculation (totals, tax) in tests/unit/erechnung_test.go

### Implementation for User Story 5

- [x] T091 [P] [US5] Implement Invoice struct in internal/erechnung/invoice.go
- [x] T092 [P] [US5] Implement InvoiceParty struct in internal/erechnung/invoice.go
- [x] T093 [P] [US5] Implement InvoiceLine struct in internal/erechnung/invoice.go
- [x] T094 [P] [US5] Implement TaxSubtotal struct in internal/erechnung/invoice.go
- [x] T095 [US5] Implement GenerateXRechnung() UBL XML output in internal/erechnung/xrechnung.go
- [x] T096 [US5] Implement GenerateZUGFeRD() CII XML output in internal/erechnung/zugferd.go
- [x] T097 [P] [US5] Implement InvoiceValidationResult struct in internal/erechnung/validate.go
- [x] T098 [US5] Implement ValidateEN16931() business rules in internal/erechnung/validate.go
- [x] T099 [US5] Implement ParseInvoiceJSON() from JSON input in internal/erechnung/invoice.go
- [x] T100 [US5] Implement `erechnung create <file.json>` command in internal/cli/erechnung.go
- [x] T101 [US5] Implement `erechnung validate <file.xml>` command in internal/cli/erechnung.go
- [x] T102 [US5] Add --format flag (xrechnung/zugferd) in internal/cli/erechnung.go
- [x] T103 [US5] Implement ExtractFromPDF() for ZUGFeRD PDFs in internal/erechnung/extract.go
- [x] T104 [US5] Implement `erechnung extract <file.pdf>` command in internal/cli/erechnung.go

**Checkpoint**: User Story 5 complete - can create and validate e-invoices

---

## Phase 8: User Story 6 - SEPA-Zahlungsdatei Erstellung (Priority: P3)

**Goal**: Generate SEPA pain.001 credit transfer files from payment data

**Independent Test**: Run `sepa pain001 payments.csv`, verify valid pain.001 XML output

### Tests for User Story 6

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T105 [P] [US6] Write unit test for pain.001 XML generation in tests/unit/sepa_test.go
- [x] T106 [P] [US6] Write unit test for camt.053 XML parsing in tests/unit/sepa_test.go
- [x] T107 [P] [US6] Write unit test for IBAN validation in tests/unit/sepa_test.go
- [x] T108 [P] [US6] Write unit test for BIC lookup in tests/unit/sepa_test.go

### Implementation for User Story 6

- [x] T109 [P] [US6] Implement SEPACreditTransfer struct in internal/sepa/pain001.go
- [x] T110 [P] [US6] Implement SEPACreditTransaction struct in internal/sepa/pain001.go
- [x] T111 [P] [US6] Implement SEPAAddress struct in internal/sepa/types.go
- [x] T112 [US6] Implement GeneratePain001() XML output in internal/sepa/pain001.go
- [x] T113 [US6] Implement ParseCSV() for payment input in internal/sepa/pain001.go
- [x] T114 [US6] Implement `sepa pain001 <file.csv>` command in internal/cli/sepa.go
- [x] T115 [P] [US6] Implement SEPAStatement struct in internal/sepa/camt053.go
- [x] T116 [P] [US6] Implement SEPAStatementEntry struct in internal/sepa/camt053.go
- [x] T117 [US6] Implement ParseCamt053() XML parsing in internal/sepa/camt053.go
- [x] T118 [US6] Implement `sepa camt053 <file.xml>` command in internal/cli/sepa.go
- [x] T119 [US6] Implement `sepa validate <IBAN>` command in internal/cli/sepa.go
- [x] T120 [US6] Add table/JSON output for SEPA commands in internal/cli/sepa.go
- [x] T121 [P] [US6] Implement SEPADirectDebit struct in internal/sepa/pain008.go
- [x] T122 [US6] Implement GeneratePain008() XML output in internal/sepa/pain008.go

**Checkpoint**: User Story 6 complete - can generate SEPA payment files

---

## Phase 9: User Story 7 - MCP Server für AI-Integration (Priority: P3)

**Goal**: Expose Austrian Business Infrastructure tools via MCP protocol for AI assistants

**Independent Test**: Start `fo mcp serve`, connect MCP client, call `fo-uid-validate` tool

### Tests for User Story 7

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T123 [P] [US7] Write unit test for MCP tool registration in tests/unit/mcp_test.go
- [x] T124 [P] [US7] Write unit test for MCP tool handler execution in tests/unit/mcp_test.go
- [x] T125 [US7] Write integration test for MCP server protocol in tests/integration/mcp_test.go

### Implementation for User Story 7

- [x] T126 [P] [US7] Implement MCPTool struct in internal/mcp/tools.go
- [x] T127 [P] [US7] Implement MCPToolResult struct in internal/mcp/tools.go
- [x] T128 [P] [US7] Implement MCPServerConfig struct in internal/mcp/server.go
- [x] T129 [US7] Implement MCP Server with stdio transport in internal/mcp/server.go
- [x] T130 [US7] Implement RegisterTools() with all available tools in internal/mcp/server.go
- [x] T131 [US7] Implement fo-databox-list tool handler in internal/mcp/handlers.go
- [x] T132 [US7] Implement fo-databox-download tool handler in internal/mcp/handlers.go
- [x] T133 [US7] Implement fo-uid-validate tool handler in internal/mcp/handlers.go
- [x] T134 [US7] Implement fo-uva-submit tool handler in internal/mcp/handlers.go
- [x] T135 [US7] Implement fb-search tool handler in internal/mcp/handlers.go
- [x] T136 [US7] Implement fb-extract tool handler in internal/mcp/handlers.go
- [x] T137 [US7] Implement `fo mcp serve` command in internal/cli/mcp.go
- [x] T138 [US7] Implement `fo mcp tools` command to list available tools in internal/cli/mcp.go

**Checkpoint**: User Story 7 complete - MCP server operational

---

## Phase 10: User Story 8 - Zusammenfassende Meldung (Priority: P3)

**Goal**: Submit quarterly recapitulative statements (ZM) to FinanzOnline

**Independent Test**: Run `fo zm submit`, verify submission confirmation with reference number

### Tests for User Story 8

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T139 [P] [US8] Write unit test for ZM XML generation in tests/unit/zm_test.go
- [x] T140 [P] [US8] Write unit test for ZM entry validation in tests/unit/zm_test.go
- [x] T141 [US8] Write integration test for ZM submission flow in tests/integration/zm_test.go

### Implementation for User Story 8

- [x] T142 [P] [US8] Implement ZM struct in internal/fonws/zm.go
- [x] T143 [P] [US8] Implement ZMEntry struct in internal/fonws/zm.go
- [x] T144 [P] [US8] Implement ZMDeliveryType constants in internal/fonws/zm.go
- [x] T145 [US8] Implement GenerateZMXML() function in internal/fonws/zm.go
- [x] T146 [US8] Implement SubmitZM() function using FileUploadService in internal/fonws/zm.go
- [x] T147 [US8] Implement `fo zm submit` command in internal/cli/zm.go
- [x] T148 [US8] Implement `fo zm generate --period Q1-2025` command in internal/cli/zm.go
- [x] T149 [US8] Add table/JSON output for ZM commands in internal/cli/zm.go

**Checkpoint**: User Story 8 complete - can submit ZM to FinanzOnline

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T150 [P] Add verbose logging throughout with --verbose flag check
- [x] T151 [P] Ensure no sensitive data logged (PIN, token, master password) per FR-065
- [x] T152 [P] Add shell completion for new commands (bash, zsh, fish, powershell)
- [x] T153 Run all tests and verify 100% of acceptance scenarios pass
- [x] T154 Run quickstart.md validation (manual test of documented workflows)
- [x] T155 Cross-platform build verification (Windows, Linux, macOS)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (SOAP client already exists from Spec 001)
- **User Story 2 (Phase 4)**: Depends on Foundational - can run parallel to US1
- **User Story 3 (Phase 5)**: Depends on Foundational + account type extension (T007-T008)
- **User Story 4 (Phase 6)**: Depends on Foundational + account type extension (T007-T008)
- **User Story 5 (Phase 7)**: Depends on Foundational only - no external service dependency
- **User Story 6 (Phase 8)**: Depends on Foundational (T010-T011 IBAN/BIC validation)
- **User Story 7 (Phase 9)**: Depends on US1, US2, US4 (exposes their functionality)
- **User Story 8 (Phase 10)**: Depends on US1 (reuses FileUploadService from UVA)
- **Polish (Phase 11)**: Depends on all user stories complete

### User Story Dependencies

```
US1 (UVA) ────────────────────────────────────────────────┐
                                                           │
US2 (UID) ────────────────────────────────────────────────┤
                                                           ▼
US3 (ELDA) ──────────────────────────────────────► US7 (MCP Server)
                                                           ▲
US4 (FB) ─────────────────────────────────────────────────┤
                                                           │
US5 (E-Rechnung) ─────────────────────────────────────────┘

US6 (SEPA) ───────────────────────────────────────────────┐
                                                           │
US8 (ZM) ─────────────── Depends on US1 ──────────────────┘
```

### Within Each User Story

1. Tests MUST be written and FAIL before implementation
2. Structs before functions
3. SOAP/service layer before CLI commands
4. Table output before JSON output
5. Core functionality before batch/advanced features

### Parallel Opportunities

**Phase 1 (all parallel)**:
- T001, T002, T003, T004, T005

**Phase 2 (can run in parallel)**:
- T009, T010, T011 (different files)

**User Story 1 (tests parallel, then structs parallel)**:
- T012, T013, T014, T015 (all unit tests)
- T017, T018, T019, T022 (all structs)

**User Story 2 (tests parallel, then structs parallel)**:
- T030, T031, T032, T033 (all unit tests)
- T035, T036, T037 (all structs)

**User Story 3 (tests parallel, then structs parallel)**:
- T046, T047, T048 (all unit tests)
- T050, T051, T052, T053, T054, T061 (all structs)

**User Story 4 (tests parallel, then structs parallel)**:
- T067, T068, T069 (all unit tests)
- T071, T072, T073, T074, T075, T083 (all structs)

**User Story 5 (tests parallel, then structs parallel)**:
- T087, T088, T089, T090 (all unit tests)
- T091, T092, T093, T094, T097 (all structs)

**User Story 6 (tests parallel, then structs parallel)**:
- T105, T106, T107, T108 (all unit tests)
- T109, T110, T111, T115, T116, T121 (all structs)

---

## Parallel Execution Examples

### Phase 2 Parallel Launch

```bash
# Launch foundational tasks together:
Task: "Implement SV-Nummer validation in internal/elda/validation.go"
Task: "Implement IBAN validation in internal/sepa/iban.go"
Task: "Implement Austrian bank code to BIC lookup in internal/sepa/bic.go"
```

### User Story 1 Test Launch

```bash
# Launch all US1 unit tests together:
Task: "Write unit test for UVA XML generation in tests/unit/uva_test.go"
Task: "Write unit test for UVA XML validation in tests/unit/uva_test.go"
Task: "Write unit test for FileUpload SOAP request in tests/unit/fonws_test.go"
Task: "Write unit test for FileUpload SOAP response in tests/unit/fonws_test.go"
```

### User Story 1 Struct Launch

```bash
# Launch all US1 structs together:
Task: "Implement UVA struct in internal/fonws/uva.go"
Task: "Implement UVAPeriod and UVAStatus in internal/fonws/uva.go"
Task: "Implement UVADocument struct in internal/fonws/uva.go"
Task: "Implement FileUploadRequest/Response in internal/fonws/upload.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (UVA)
4. Complete Phase 4: User Story 2 (UID)
5. **STOP and VALIDATE**: Can submit UVA, validate UID
6. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 (UVA) → Can submit tax returns → Demo
3. Add US2 (UID) → Can validate partners → Demo
4. Add US3 (ELDA) → Can register employees → Demo
5. Add US4 (FB) → Can query companies → Demo
6. Add US5 (E-Rechnung) → Can create invoices → Demo
7. Add US6 (SEPA) → Can create payments → Demo
8. Add US7 (MCP) → AI integration ready → Demo
9. Add US8 (ZM) → Full FinanzOnline suite → Demo (Feature complete!)

### Parallel Team Strategy

With 4 developers after Foundational:
- Developer A: User Story 1 (UVA) + User Story 8 (ZM)
- Developer B: User Story 2 (UID) + User Story 4 (FB)
- Developer C: User Story 3 (ELDA) + User Story 5 (E-Rechnung)
- Developer D: User Story 6 (SEPA) → then User Story 7 (MCP after US1, US2, US4)

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story is independently testable at its checkpoint
- Tests MUST fail before implementation (Constitution Principle IV)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- mcp-go is the only new external dependency (Constitution Principle V)
