# Tasks: MCP Tools Expansion

**Input**: Design documents from `/specs/003-mcp-tools-expansion/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included per Constitution (Test-First Development principle IV)

**Organization**: Tasks grouped by user story for independent implementation and testing.

**Status**: ✅ COMPLETE (2025-12-07)

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1=Databox, US2=Firmenbuch, US3=UVA)
- All paths relative to repository root

---

## Phase 1: Setup

**Purpose**: No new setup required - extending existing MCP infrastructure

- [x] T001 Verify existing MCP server builds successfully via `go build ./...`
- [x] T002 Verify existing MCP tests pass via `go test ./tests/unit/... -run TestMCP -v`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Session helper function needed by databox and UVA tools

**⚠️ CRITICAL**: US1 and US3 depend on session lookup; US2 does not need sessions

- [x] T003 Add test for session lookup helper in tests/unit/mcp_test.go
- [x] T004 Implement getSessionForAccount helper in internal/mcp/handlers.go

**Checkpoint**: ✅ Session lookup ready - user story implementation can begin

---

## Phase 3: User Story 1 - Databox Document Access (Priority: P1) 🎯 MVP

**Goal**: Enable AI assistants to list and download documents from FinanzOnline databox

**Independent Test**: Call `fo-databox-list` and `fo-databox-download` via MCP protocol and verify structured responses

### Tests for User Story 1

> **NOTE: Write tests FIRST, ensure they FAIL before implementation**

- [x] T005 [P] [US1] Add test for handleDataboxList in tests/unit/mcp_test.go
- [x] T006 [P] [US1] Add test for handleDataboxDownload in tests/unit/mcp_test.go
- [x] T007 [P] [US1] Add test for databox auth error handling in tests/unit/mcp_test.go

### Implementation for User Story 1

- [x] T008 [US1] Implement handleDataboxList handler in internal/mcp/handlers.go
- [x] T009 [US1] Implement handleDataboxDownload handler in internal/mcp/handlers.go
- [x] T010 [US1] Register fo-databox-list tool in internal/mcp/server.go
- [x] T011 [US1] Register fo-databox-download tool in internal/mcp/server.go
- [x] T012 [US1] Verify tests pass: `go test ./tests/unit/... -run "TestMCP.*Databox" -v`

**Checkpoint**: ✅ Databox tools functional - AI can list and download documents

---

## Phase 4: User Story 2 - Firmenbuch Company Lookup (Priority: P2)

**Goal**: Enable AI assistants to search and retrieve company information from Firmenbuch

**Independent Test**: Call `fo-fb-search` and `fo-fb-extract` via MCP protocol and verify structured responses

### Tests for User Story 2

- [x] T013 [P] [US2] Add test for handleFBSearch in tests/unit/mcp_test.go
- [x] T014 [P] [US2] Add test for handleFBExtract in tests/unit/mcp_test.go
- [x] T015 [P] [US2] Add test for FN validation error in tests/unit/mcp_test.go

### Implementation for User Story 2

- [x] T016 [US2] Implement handleFBSearch handler in internal/mcp/handlers.go
- [x] T017 [US2] Implement handleFBExtract handler in internal/mcp/handlers.go
- [x] T018 [US2] Register fo-fb-search tool in internal/mcp/server.go
- [x] T019 [US2] Register fo-fb-extract tool in internal/mcp/server.go
- [x] T020 [US2] Verify tests pass: `go test ./tests/unit/... -run "TestMCP.*FB" -v`

**Checkpoint**: ✅ Firmenbuch tools functional - AI can search and extract company data

---

## Phase 5: User Story 3 - UVA Submission (Priority: P3)

**Goal**: Enable AI assistants to submit UVA (VAT advance returns) to FinanzOnline

**Independent Test**: Call `fo-uva-submit` via MCP protocol and verify submission response

### Tests for User Story 3

- [x] T021 [P] [US3] Add test for handleUVASubmit in tests/unit/mcp_test.go
- [x] T022 [P] [US3] Add test for UVA validation errors in tests/unit/mcp_test.go
- [x] T023 [P] [US3] Add test for UVA auth error handling in tests/unit/mcp_test.go

### Implementation for User Story 3

- [x] T024 [US3] Implement handleUVASubmit handler in internal/mcp/handlers.go
- [x] T025 [US3] Register fo-uva-submit tool in internal/mcp/server.go
- [x] T026 [US3] Verify tests pass: `go test ./tests/unit/... -run "TestMCP.*UVA" -v`

**Checkpoint**: ✅ UVA tool functional - AI can submit VAT returns

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification and integration

- [x] T027 Verify all 10 MCP tools appear in `fo mcp tools` output
- [x] T028 Run full test suite: `go test ./... -v`
- [x] T029 Build release binary: `go build -o fo.exe ./cmd/fo`
- [x] T030 Validate quickstart.md examples work with built binary

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - verify existing state
- **Phase 2 (Foundational)**: Depends on Phase 1 - adds session helper
- **Phase 3 (US1 Databox)**: Depends on Phase 2 - uses session helper
- **Phase 4 (US2 Firmenbuch)**: Depends on Phase 1 only - no session needed
- **Phase 5 (US3 UVA)**: Depends on Phase 2 - uses session helper
- **Phase 6 (Polish)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (Databox)**: Requires session helper from Phase 2
- **US2 (Firmenbuch)**: Independent - can run in parallel with US1/US3 after Phase 1
- **US3 (UVA)**: Requires session helper from Phase 2

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Handlers before registrations
- Registration after handler complete
- Verify tests pass before checkpoint

### Parallel Opportunities

- T005, T006, T007 can run in parallel (different test functions)
- T013, T014, T015 can run in parallel (different test functions)
- T021, T022, T023 can run in parallel (different test functions)
- US2 (Firmenbuch) can run in parallel with US1 or US3 after Phase 1

---

## Parallel Example: User Story 2

```bash
# Launch all tests for User Story 2 together:
Task: T013 "Add test for handleFBSearch in tests/unit/mcp_test.go"
Task: T014 "Add test for handleFBExtract in tests/unit/mcp_test.go"
Task: T015 "Add test for FN validation error in tests/unit/mcp_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup verification
2. Complete Phase 2: Session helper (CRITICAL for US1)
3. Complete Phase 3: User Story 1 (Databox)
4. **STOP and VALIDATE**: Test databox tools independently
5. Deploy/demo if ready

### Incremental Delivery

1. Phase 1 + 2 → Foundation ready
2. Add US1 (Databox) → Test → MVP deployed
3. Add US2 (Firmenbuch) → Test → Deploy
4. Add US3 (UVA) → Test → Deploy
5. Each story adds value without breaking previous

### Parallel Strategy

With foundation complete:
- Developer A: US1 (Databox) - needs session helper
- Developer B: US2 (Firmenbuch) - independent
- Then: US3 (UVA) - needs session helper

---

## Notes

- [P] tasks = different test functions or files, no dependencies
- [US1/US2/US3] label maps task to specific user story
- Each user story independently testable
- Verify tests fail before implementing handlers
- Commit after each task or logical group
- Constitution: Test-First Development (Principle IV)

---

## Completion Summary

**Completed**: 2025-12-07

### Files Modified
- `internal/mcp/handlers.go` - Added 5 new handler functions + helper utilities
- `internal/mcp/server.go` - Registered 5 new tools
- `tests/unit/mcp_test.go` - Added 11 new test functions

### Tools Implemented
1. `fo-databox-list` - Lists FinanzOnline databox documents
2. `fo-databox-download` - Downloads databox document content
3. `fo-fb-search` - Searches Firmenbuch by company name
4. `fo-fb-extract` - Retrieves full company details by FN
5. `fo-uva-submit` - Submits UVA (VAT return) to FinanzOnline

### Test Results
- All 20 MCP-related tests pass
- Full test suite passes (100+ tests)
- Build succeeds on all platforms
