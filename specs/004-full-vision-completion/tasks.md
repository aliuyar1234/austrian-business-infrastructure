# Tasks: Full Vision Completion

**Input**: Design documents from `/specs/004-full-vision-completion/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Organization**: Tasks grouped by user story for independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1-US7)
- Exact file paths included

---

## Phase 1: Setup

**Purpose**: Verify existing infrastructure, no new setup needed

- [x] T001 Verify Go 1.23+ installed and project builds with `go build ./...`
- [x] T002 [P] Verify all existing tests pass with `go test ./...`
- [x] T003 [P] Review existing module skeletons in internal/elda/, internal/fb/, internal/erechnung/, internal/sepa/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Extend core infrastructure to support new account types

**⚠️ CRITICAL**: Must complete before user stories

- [x] T004 Extend AccountType enum to include `elda` and `firmenbuch` types in internal/store/accounts.go
- [x] T005 Add ELDA credential fields (ELDASerial, BKNR) to Account struct in internal/store/accounts.go
- [x] T006 Add Firmenbuch credential field (FBAPIKey) to Account struct in internal/store/accounts.go
- [x] T007 Update `fo account add` command to accept --type elda/firmenbuch flags in internal/cli/account.go
- [x] T008 [P] Implement SV-Nummer validation algorithm in internal/elda/validation.go
- [x] T009 [P] Implement FN-Nummer validation (regex + check letter) in internal/fb/validation.go
- [x] T010 [P] Add retry logic helper for network operations in internal/fonws/client.go

**Checkpoint**: Foundation ready - account types support all services ✅

---

## Phase 3: User Story 1 - Multi-Service Dashboard (Priority: P1) 🎯 MVP

**Goal**: Unified dashboard showing status across all configured accounts (FinanzOnline, ELDA, Firmenbuch)

**Independent Test**: Run `fo dashboard --all` and verify consolidated status from all services

### Implementation for User Story 1

- [x] T011 [US1] Define DashboardResult struct with service statuses in internal/cli/dashboard.go
- [x] T012 [US1] Implement parallel account checking using errgroup in internal/cli/dashboard.go
- [x] T013 [US1] Add FinanzOnline databox count to dashboard result in internal/cli/dashboard.go
- [x] T014 [US1] Add ELDA pending registrations placeholder to dashboard in internal/cli/dashboard.go
- [x] T015 [US1] Add Firmenbuch watch list placeholder to dashboard in internal/cli/dashboard.go
- [x] T016 [US1] Implement table output format for dashboard in internal/cli/dashboard.go
- [x] T017 [US1] Implement JSON output format for dashboard (--json flag) in internal/cli/dashboard.go
- [x] T018 [US1] Add --services filter flag to dashboard command in internal/cli/dashboard.go
- [x] T019 [US1] Handle partial failures gracefully (show successful + mark failed) in internal/cli/dashboard.go
- [x] T020 [US1] Add unit test for dashboard aggregation in tests/unit/dashboard_test.go

**Checkpoint**: Dashboard shows consolidated view across all account types ✅

---

## Phase 4: User Story 2 - ELDA Employee Registration (Priority: P1)

**Goal**: Register/deregister employees with Austrian social insurance via CLI

**Independent Test**: Run `fo elda anmeldung --employee-file data.json` and verify confirmation

### Implementation for User Story 2

- [x] T021 [P] [US2] Define Employee struct with all fields in internal/elda/types.go
- [x] T022 [P] [US2] Define ELDAMessage struct for XML generation in internal/elda/types.go
- [x] T023 [P] [US2] Define ELDAResponse struct for parsing responses in internal/elda/types.go
- [x] T024 [US2] Implement GenerateAnmeldungXML function in internal/elda/client.go
- [x] T025 [US2] Implement GenerateAbmeldungXML function in internal/elda/client.go
- [x] T026 [US2] Implement ELDA FTPS submission client in internal/elda/client.go
- [x] T027 [US2] Implement ParseELDAResponse function in internal/elda/client.go
- [x] T028 [US2] Map ELDA error codes to meaningful messages in internal/elda/errors.go
- [x] T029 [US2] Implement `fo elda anmeldung` command in internal/cli/elda.go
- [x] T030 [US2] Implement `fo elda abmeldung` command in internal/cli/elda.go
- [x] T031 [US2] Implement `fo elda status` command in internal/cli/elda.go
- [x] T032 [US2] Add JSON input parsing for employee data in internal/cli/elda.go
- [x] T033 [US2] Add --test flag for ELDA test environment in internal/cli/elda.go
- [x] T034 [US2] Add unit tests for SV-Nummer validation in tests/unit/elda_test.go
- [x] T035 [US2] Add unit tests for XML generation in tests/unit/elda_test.go

**Checkpoint**: ELDA registration works independently via CLI ✅

---

## Phase 5: User Story 6 - UVA Submission (Priority: P1)

**Goal**: Submit VAT advance returns directly via CLI

**Independent Test**: Run `fo uva submit --input uva.json --account "Test"` and verify confirmation

### Implementation for User Story 6

- [x] T036 [P] [US6] Define UVA struct with all Kennzahlen in internal/fonws/uva.go
- [x] T037 [P] [US6] Define UVAResponse struct in internal/fonws/uva.go
- [x] T038 [US6] Implement GenerateUVAXML function per BMF spec in internal/fonws/uva.go
- [x] T039 [US6] Implement ValidateUVA function (required fields, calculations) in internal/fonws/uva.go
- [x] T040 [US6] Implement SubmitUVA via fileUploadService in internal/fonws/uva.go
- [x] T041 [US6] Add Base64 encoding for XML data in internal/fonws/uva.go
- [x] T042 [US6] Implement `fo uva submit` command in internal/cli/uva.go
- [x] T043 [US6] Implement `fo uva calculate` preview command in internal/cli/uva.go
- [x] T044 [US6] Add --test flag for test submission (uebermittlung=T) in internal/cli/uva.go
- [x] T045 [US6] Add JSON input parsing for UVA data in internal/cli/uva.go
- [x] T046 [P] [US6] Define ZM struct with entries in internal/fonws/zm.go
- [x] T047 [US6] Implement GenerateZMXML function in internal/fonws/zm.go
- [x] T048 [US6] Implement SubmitZM via fileUploadService in internal/fonws/zm.go
- [x] T049 [US6] Implement `fo zm submit` command in internal/cli/zm.go
- [x] T050 [US6] Add unit tests for UVA XML generation in tests/unit/uva_test.go
- [x] T051 [US6] Add unit tests for ZM XML generation in tests/unit/zm_test.go

**Checkpoint**: UVA and ZM submission works via CLI ✅

---

## Phase 6: User Story 3 - Firmenbuch Due Diligence (Priority: P2)

**Goal**: Query Austrian company registry for company details and insolvency status

**Independent Test**: Run `fo fb search "Muster GmbH"` and `fo fb extract FN123456b`

### Implementation for User Story 3

- [x] T052 [P] [US3] Define Company struct with all fields in internal/fb/types.go
- [x] T053 [P] [US3] Define Director struct in internal/fb/types.go
- [x] T054 [P] [US3] Define Address struct (shared) in internal/fb/types.go
- [x] T055 [P] [US3] Define FBSearchRequest/Response structs in internal/fb/types.go
- [x] T056 [P] [US3] Define FBAuszugRequest/Response structs in internal/fb/types.go
- [x] T057 [US3] Implement SOAP envelope builder for Firmenbuch in internal/fb/client.go
- [x] T058 [US3] Implement SearchCompany function in internal/fb/client.go
- [x] T059 [US3] Implement ExtractCompany (Auszug) function in internal/fb/client.go
- [x] T060 [US3] Add X-API-KEY header authentication in internal/fb/client.go
- [x] T061 [US3] Parse SOAP response to Company struct in internal/fb/client.go
- [x] T062 [US3] Implement `fo fb search` command in internal/cli/fb.go
- [x] T063 [US3] Implement `fo fb extract` command in internal/cli/fb.go
- [x] T064 [US3] Implement `fo fb monitor add/remove/list` commands in internal/cli/fb.go
- [x] T065 [US3] Add table output format for company data in internal/cli/fb.go
- [x] T066 [US3] Add JSON output format for company data in internal/cli/fb.go
- [x] T067 [US3] Add unit tests for FN-Nummer validation in tests/unit/fb_test.go
- [x] T068 [US3] Add unit tests for SOAP request building in tests/unit/fb_test.go

**Checkpoint**: Firmenbuch search and extract work via CLI ✅

---

## Phase 7: User Story 4 - E-Invoice Generation (Priority: P2)

**Goal**: Generate EN16931-compliant XRechnung/ZUGFeRD invoices

**Independent Test**: Run `fo erechnung create --input invoice.json --format xrechnung`

### Implementation for User Story 4

- [x] T069 [P] [US4] Define Invoice struct with all fields in internal/erechnung/invoice.go
- [x] T070 [P] [US4] Define InvoiceParty struct in internal/erechnung/invoice.go
- [x] T071 [P] [US4] Define InvoiceLine struct in internal/erechnung/invoice.go
- [x] T072 [P] [US4] Define TaxSubtotal struct in internal/erechnung/invoice.go
- [x] T073 [P] [US4] Define TaxCategory enum (S, Z, E, AE, K) in internal/erechnung/invoice.go
- [x] T074 [US4] Implement CalculateTotals function in internal/erechnung/invoice.go
- [x] T075 [US4] Implement GenerateXRechnungXML (UBL format) in internal/erechnung/xrechnung.go
- [x] T076 [US4] Implement tax rate mapping (20%, 13%, 10%, 0%) in internal/erechnung/xrechnung.go
- [x] T077 [US4] Implement ValidateInvoice function against EN16931 rules in internal/erechnung/validate.go
- [x] T078 [US4] Implement ParseXRechnungXML for validation in internal/erechnung/validate.go
- [x] T079 [US4] Implement `fo erechnung create` command in internal/cli/erechnung.go
- [x] T080 [US4] Implement `fo erechnung validate` command in internal/cli/erechnung.go
- [x] T081 [US4] Add --format flag (xrechnung, zugferd, ebinterface) in internal/cli/erechnung.go
- [x] T082 [US4] Add JSON input parsing for invoice data in internal/cli/erechnung.go
- [x] T083 [US4] Add unit tests for invoice calculation in tests/unit/erechnung_test.go
- [x] T084 [US4] Add unit tests for XRechnung generation in tests/unit/erechnung_test.go

**Checkpoint**: E-invoice generation and validation work via CLI ✅

---

## Phase 8: User Story 5 - SEPA Payment Files (Priority: P2)

**Goal**: Generate SEPA payment files and parse bank statements

**Independent Test**: Run `fo sepa pain001 --input payments.json`

### Implementation for User Story 5

- [x] T085 [P] [US5] Define SEPACreditTransfer struct in internal/sepa/types.go
- [x] T086 [P] [US5] Define PaymentInfo struct in internal/sepa/types.go
- [x] T087 [P] [US5] Define CreditTransferTx struct in internal/sepa/types.go
- [x] T088 [P] [US5] Define SEPAParty and SEPAAccount structs in internal/sepa/types.go
- [x] T089 [P] [US5] Define SEPAStatement struct for camt.053 in internal/sepa/types.go
- [x] T090 [US5] Implement IBAN validation (MOD 97-10) in internal/sepa/iban.go
- [x] T091 [US5] Implement BIC validation in internal/sepa/bic.go
- [x] T092 [US5] Implement Austrian BLZ to BIC lookup table in internal/sepa/bic.go
- [x] T093 [US5] Implement GeneratePain001XML (version 03 and 09) in internal/sepa/pain001.go
- [x] T094 [US5] Implement GeneratePain008XML (version 02 and 08) in internal/sepa/pain008.go
- [x] T095 [US5] Implement ParseCamt053 bank statement parser in internal/sepa/camt053.go
- [x] T096 [US5] Implement `fo sepa pain001` command in internal/cli/sepa.go
- [x] T097 [US5] Implement `fo sepa pain008` command in internal/cli/sepa.go
- [x] T098 [US5] Implement `fo sepa parse` command for camt.053 in internal/cli/sepa.go
- [x] T099 [US5] Implement `fo sepa validate` command for IBAN in internal/cli/sepa.go
- [x] T100 [US5] Add --version flag for SEPA version selection in internal/cli/sepa.go
- [x] T101 [US5] Add unit tests for IBAN validation in tests/unit/sepa_test.go
- [x] T102 [US5] Add unit tests for pain.001 generation in tests/unit/sepa_test.go

**Checkpoint**: SEPA payment file generation works via CLI ✅

---

## Phase 9: User Story 7 - MCP Server Extension (Priority: P3)

**Goal**: Expose all tools via MCP for AI integration

**Independent Test**: Start `fo mcp serve` and call tools via JSON-RPC

### Implementation for User Story 7

- [x] T103 [P] [US7] Add fo-elda-register tool definition in internal/mcp/tools.go
- [x] T104 [P] [US7] Add fo-elda-status tool definition in internal/mcp/tools.go
- [x] T105 [P] [US7] Add fo-zm-submit tool definition in internal/mcp/tools.go
- [x] T106 [P] [US7] Add fo-invoice-create tool definition in internal/mcp/tools.go
- [x] T107 [P] [US7] Add fo-invoice-validate tool definition in internal/mcp/tools.go
- [x] T108 [P] [US7] Add fo-sepa-pain001 tool definition in internal/mcp/tools.go
- [x] T109 [P] [US7] Add fo-sepa-pain008 tool definition in internal/mcp/tools.go
- [x] T110 [US7] Implement handler for fo-elda-register in internal/mcp/handlers.go
- [x] T111 [US7] Implement handler for fo-elda-status in internal/mcp/handlers.go
- [x] T112 [US7] Implement handler for fo-zm-submit in internal/mcp/handlers.go
- [x] T113 [US7] Implement handler for fo-invoice-create in internal/mcp/handlers.go
- [x] T114 [US7] Implement handler for fo-invoice-validate in internal/mcp/handlers.go
- [x] T115 [US7] Implement handler for fo-sepa-pain001 in internal/mcp/handlers.go
- [x] T116 [US7] Implement handler for fo-sepa-pain008 in internal/mcp/handlers.go
- [x] T117 [US7] Register all new tools in MCP server in internal/mcp/server.go
- [x] T118 [US7] Add unit tests for new MCP tools in tests/unit/mcp_test.go

**Checkpoint**: All services accessible via MCP server ✅

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Final improvements across all stories

- [x] T119 [P] Add environment variable support (FO_CONFIG_DIR, FO_JSON_OUTPUT) in internal/config/paths.go
- [x] T120 [P] Add verbose logging to all commands (--verbose flag) in internal/cli/root.go
- [x] T121 Integrate ELDA status into dashboard (replace placeholder) in internal/cli/dashboard.go
- [x] T122 Integrate Firmenbuch watch list into dashboard in internal/cli/dashboard.go
- [x] T123 [P] Run all tests and fix any failures with `go test ./...`
- [x] T124 [P] Build and test binary with `go build -o fo.exe ./cmd/fo`
- [x] T125 Validate quickstart.md examples work end-to-end

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies
- **Phase 2 (Foundational)**: Depends on Phase 1 - BLOCKS all user stories
- **Phases 3-9 (User Stories)**: All depend on Phase 2 completion
- **Phase 10 (Polish)**: Depends on all desired user stories

### User Story Dependencies

| Story | Priority | Can Start After | Depends On |
|-------|----------|-----------------|------------|
| US1 (Dashboard) | P1 | Phase 2 | None |
| US2 (ELDA) | P1 | Phase 2 | None |
| US6 (UVA/ZM) | P1 | Phase 2 | None |
| US3 (Firmenbuch) | P2 | Phase 2 | None |
| US4 (E-Rechnung) | P2 | Phase 2 | None |
| US5 (SEPA) | P2 | Phase 2 | None |
| US7 (MCP) | P3 | US2, US3, US4, US5, US6 | All other stories |

### Parallel Opportunities

**Within Phase 2 (Foundational):**
```
T008, T009, T010 can run in parallel
```

**All P1 stories can run in parallel after Phase 2:**
```
US1 (Dashboard) || US2 (ELDA) || US6 (UVA/ZM)
```

**All P2 stories can run in parallel:**
```
US3 (Firmenbuch) || US4 (E-Rechnung) || US5 (SEPA)
```

**Within User Story 2 (ELDA):**
```
T021, T022, T023 (types) can run in parallel
```

**Within User Story 5 (SEPA):**
```
T085, T086, T087, T088, T089 (types) can run in parallel
```

---

## Implementation Strategy

### MVP First (P1 Stories Only)

1. Complete Phase 1: Setup (verify)
2. Complete Phase 2: Foundational (account types)
3. Complete Phase 3: US1 Dashboard → **Deploy/Demo**
4. Complete Phase 4: US2 ELDA → **Deploy/Demo**
5. Complete Phase 5: US6 UVA/ZM → **Deploy/Demo**

### Incremental Delivery

| Increment | Stories | Value Delivered |
|-----------|---------|-----------------|
| MVP | US1 + US6 | Dashboard + UVA submission |
| +ELDA | US2 | Employee registration |
| +Firmenbuch | US3 | Company due diligence |
| +E-Rechnung | US4 | Invoice generation |
| +SEPA | US5 | Payment files |
| +MCP | US7 | AI integration |

### Parallel Team Strategy

With 3 developers after Phase 2:
- Dev A: US1 (Dashboard) → US3 (Firmenbuch)
- Dev B: US2 (ELDA) → US4 (E-Rechnung)
- Dev C: US6 (UVA/ZM) → US5 (SEPA)
- All: US7 (MCP) after dependencies complete

---

## Summary

| Metric | Value |
|--------|-------|
| Total Tasks | 125 |
| Phase 1 (Setup) | 3 |
| Phase 2 (Foundational) | 7 |
| US1 (Dashboard) | 10 |
| US2 (ELDA) | 15 |
| US6 (UVA/ZM) | 16 |
| US3 (Firmenbuch) | 17 |
| US4 (E-Rechnung) | 16 |
| US5 (SEPA) | 18 |
| US7 (MCP) | 16 |
| Phase 10 (Polish) | 7 |
| Parallel Tasks | 41 |
| MVP Scope | US1 + US2 + US6 (48 tasks) |

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to user story
- Each story is independently testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story
