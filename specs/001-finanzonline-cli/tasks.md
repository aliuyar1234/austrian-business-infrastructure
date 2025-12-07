# Tasks: FinanzOnline CLI

**Input**: Design documents from `/specs/001-finanzonline-cli/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Test-First Development per Constitution. Tests MUST fail before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

**Status**: ✅ COMPLETE (2025-12-07)

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `cmd/`, `internal/`, `tests/` at repository root
- Paths follow Go conventions from plan.md

---

## Phase 1: Setup

**Purpose**: Project initialization and basic structure

- [x] T001 Initialize Go module with `go mod init github.com/your-org/austrian-business-infrastructure`
- [x] T002 Create project directory structure per plan.md (cmd/fo/, internal/cli/, internal/fonws/, internal/store/, internal/config/, tests/)
- [x] T003 [P] Add Cobra dependency `go get github.com/spf13/cobra`
- [x] T004 [P] Add errgroup dependency `go get golang.org/x/sync/errgroup`
- [x] T005 [P] Add argon2 dependency `go get golang.org/x/crypto/argon2`
- [x] T006 Create CLI entry point skeleton in cmd/fo/main.go
- [x] T007 Create Cobra root command with global flags (--config, --json, --verbose) in internal/cli/root.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T008 Implement platform path resolution (XDG/AppData) in internal/config/paths.go
- [x] T009 [P] Implement AES-256-GCM encryption/decryption in internal/store/crypto.go
- [x] T010 [P] Implement Argon2id key derivation from master password in internal/store/crypto.go
- [x] T011 Implement SOAP envelope builder in internal/fonws/client.go
- [x] T012 [P] Implement HTTP POST with XML content-type in internal/fonws/client.go
- [x] T013 Implement SOAP response parser in internal/fonws/client.go
- [x] T014 Implement FinanzOnline error code mapping (codes 0 to -8) in internal/fonws/errors.go
- [x] T015 Implement human-readable error messages for all error codes in internal/fonws/errors.go

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Single Account Login (Priority: P1)

**Goal**: Authenticate with FinanzOnline WebService and maintain session state

**Independent Test**: Run `fo session login` with valid credentials, verify session established. Run with invalid credentials, verify appropriate error message.

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T016 [P] [US1] Write unit test for Login SOAP request serialization in tests/unit/fonws_test.go
- [x] T017 [P] [US1] Write unit test for Login SOAP response parsing (success) in tests/unit/fonws_test.go
- [x] T018 [P] [US1] Write unit test for Login SOAP response parsing (all error codes) in tests/unit/fonws_test.go
- [x] T019 [P] [US1] Write unit test for Logout SOAP request/response in tests/unit/fonws_test.go
- [x] T020 [US1] Write integration test for session login flow with mocked SOAP in tests/integration/session_test.go

### Implementation for User Story 1

- [x] T021 [P] [US1] Implement LoginRequest struct with XML tags in internal/fonws/session.go
- [x] T022 [P] [US1] Implement LoginResponse struct with XML tags in internal/fonws/session.go
- [x] T023 [P] [US1] Implement LogoutRequest/LogoutResponse structs in internal/fonws/session.go
- [x] T024 [US1] Implement Login() function calling SOAP endpoint in internal/fonws/session.go
- [x] T025 [US1] Implement Logout() function calling SOAP endpoint in internal/fonws/session.go
- [x] T026 [US1] Implement Session struct (token, accountName, valid) in internal/fonws/session.go
- [x] T027 [US1] Implement `fo session login <account>` command in internal/cli/session.go
- [x] T028 [US1] Implement `fo session logout` command in internal/cli/session.go
- [x] T029 [US1] Add table output for login success in internal/cli/session.go
- [x] T030 [US1] Add JSON output for login success (--json flag) in internal/cli/session.go

**Checkpoint**: User Story 1 complete - can authenticate with FinanzOnline

---

## Phase 4: User Story 2 - Multi-Account Management (Priority: P2)

**Goal**: Store, list, and remove multiple account credentials with encrypted storage

**Independent Test**: Run `fo account add`, verify account stored. Run `fo account list`, verify accounts displayed. Run `fo account remove`, verify account deleted.

### Tests for User Story 2

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T031 [P] [US2] Write unit test for Account struct validation (TID format, unique name) in tests/unit/store_test.go
- [x] T032 [P] [US2] Write unit test for CredentialStore JSON serialization in tests/unit/store_test.go
- [x] T033 [P] [US2] Write unit test for encrypt/decrypt roundtrip with master password in tests/unit/store_test.go
- [x] T034 [P] [US2] Write unit test for wrong master password error in tests/unit/store_test.go
- [x] T035 [US2] Write integration test for add/list/remove account flow in tests/integration/store_test.go

### Implementation for User Story 2

- [x] T036 [P] [US2] Implement Account struct with validation in internal/store/accounts.go
- [x] T037 [P] [US2] Implement CredentialStore struct in internal/store/accounts.go
- [x] T038 [US2] Implement Load() to decrypt and parse credential file in internal/store/accounts.go
- [x] T039 [US2] Implement Save() to encrypt and write credential file in internal/store/accounts.go
- [x] T040 [US2] Implement AddAccount() with duplicate name check in internal/store/accounts.go
- [x] T041 [US2] Implement RemoveAccount() in internal/store/accounts.go
- [x] T042 [US2] Implement GetAccount() by name in internal/store/accounts.go
- [x] T043 [US2] Implement ListAccounts() returning names without PINs in internal/store/accounts.go
- [x] T044 [US2] Implement `fo account add <name>` command with interactive prompts in internal/cli/account.go
- [x] T045 [US2] Implement `fo account list` command in internal/cli/account.go
- [x] T046 [US2] Implement `fo account remove <name>` command in internal/cli/account.go
- [x] T047 [US2] Add table output for account list in internal/cli/account.go
- [x] T048 [US2] Add JSON output for account commands (--json flag) in internal/cli/account.go

**Checkpoint**: User Story 2 complete - can manage multiple accounts

---

## Phase 5: User Story 3 - Databox Retrieval (Priority: P3)

**Goal**: List databox documents and download individual files

**Independent Test**: Run `fo databox list <account>`, verify document list displayed. Run `fo databox download <applkey>`, verify file saved locally.

### Tests for User Story 3

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T049 [P] [US3] Write unit test for GetDataboxInfo SOAP request serialization in tests/unit/fonws_test.go
- [x] T050 [P] [US3] Write unit test for GetDataboxInfo SOAP response parsing in tests/unit/fonws_test.go
- [x] T051 [P] [US3] Write unit test for DataboxEntry.ActionRequired() helper in tests/unit/fonws_test.go
- [x] T052 [P] [US3] Write unit test for GetDatabox (download) request/response in tests/unit/fonws_test.go
- [x] T053 [US3] Write integration test for databox list with mocked SOAP in tests/integration/databox_test.go

### Implementation for User Story 3

- [x] T054 [P] [US3] Implement GetDataboxInfoRequest struct in internal/fonws/databox.go
- [x] T055 [P] [US3] Implement GetDataboxInfoResponse struct in internal/fonws/databox.go
- [x] T056 [P] [US3] Implement DataboxEntry struct with ActionRequired() helper in internal/fonws/databox.go
- [x] T057 [US3] Implement GetDataboxInfo() function calling SOAP endpoint in internal/fonws/databox.go
- [x] T058 [P] [US3] Implement GetDataboxRequest/Response structs in internal/fonws/databox.go
- [x] T059 [US3] Implement GetDatabox() function for document download in internal/fonws/databox.go
- [x] T060 [US3] Implement base64 decoding and file write for downloads in internal/fonws/databox.go
- [x] T061 [US3] Implement `fo databox list <account>` command in internal/cli/databox.go
- [x] T062 [US3] Implement `fo databox download <applkey>` command in internal/cli/databox.go
- [x] T063 [US3] Add table output with action-required flag (⚠️) in internal/cli/databox.go
- [x] T064 [US3] Add JSON output for databox commands (--json flag) in internal/cli/databox.go
- [x] T065 [US3] Add --from and --to date filters in internal/cli/databox.go

**Checkpoint**: User Story 3 complete - can retrieve and download databox documents

---

## Phase 6: User Story 4 - Batch Operations Dashboard (Priority: P4)

**Goal**: Check all accounts in parallel and display aggregated summary

**Independent Test**: Run `fo databox list --all`, verify all accounts checked and summary displayed with action-required flags.

### Tests for User Story 4

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T066 [P] [US4] Write unit test for parallel account processing with errgroup in tests/unit/dashboard_test.go
- [x] T067 [P] [US4] Write unit test for aggregated result sorting in tests/unit/dashboard_test.go
- [x] T068 [US4] Write integration test for batch operation with multiple mocked accounts in tests/integration/databox_test.go

### Implementation for User Story 4

- [x] T069 [US4] Implement parallel login for multiple accounts using errgroup in internal/cli/dashboard.go
- [x] T070 [US4] Implement parallel databox fetch for multiple accounts in internal/cli/dashboard.go
- [x] T071 [US4] Implement progress indicator ("Checking N accounts...") in internal/cli/dashboard.go
- [x] T072 [US4] Implement aggregated summary table (account, new items, action required) in internal/cli/dashboard.go
- [x] T073 [US4] Implement `fo databox list --all` flag handling in internal/cli/databox.go
- [x] T074 [US4] Add JSON output for batch operation results in internal/cli/dashboard.go
- [x] T075 [US4] Add error handling for partial failures (some accounts succeed, some fail) in internal/cli/dashboard.go

**Checkpoint**: User Story 4 complete - batch dashboard operational

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [x] T076 [P] Add verbose logging throughout with --verbose flag check
- [x] T077 [P] Ensure no sensitive data logged (PIN, token, master password) per FR-012
- [x] T078 [P] Add shell completion generation (bash, zsh, fish, powershell) in internal/cli/root.go
- [x] T079 [P] Add --version flag with build info in internal/cli/root.go
- [x] T080 Run all tests and verify 100% of acceptance scenarios pass
- [x] T081 Run quickstart.md validation (manual test of documented workflow)
- [x] T082 Cross-platform build verification (Windows, Linux, macOS)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (SOAP client, error codes)
- **User Story 2 (Phase 4)**: Depends on Foundational (crypto, paths) - can run parallel to US1
- **User Story 3 (Phase 5)**: Depends on US1 (session) + US2 (accounts)
- **User Story 4 (Phase 6)**: Depends on US3 (databox) + US2 (multi-account)
- **Polish (Phase 7)**: Depends on all user stories complete

### User Story Dependencies

```
US1 (Session) ─────────────────────────────┐
                                           ▼
US2 (Accounts) ───────────────────────────► US3 (Databox) ──► US4 (Dashboard)
```

### Within Each User Story

1. Tests MUST be written and FAIL before implementation
2. Structs before functions
3. Functions before CLI commands
4. Table output before JSON output

### Parallel Opportunities

**Phase 2 (can run in parallel):**
- T009, T010 (crypto)
- T011, T012, T013 (SOAP client)

**User Story 1 (can run in parallel):**
- T016, T017, T018, T019 (all unit tests)
- T021, T022, T023 (all structs)

**User Story 2 (can run in parallel):**
- T031, T032, T033, T034 (all unit tests)
- T036, T037 (structs)

**User Story 3 (can run in parallel):**
- T049, T050, T051, T052 (all unit tests)
- T054, T055, T056, T058 (all structs)

**User Story 4 (can run in parallel):**
- T066, T067 (unit tests)

---

## Parallel Execution Examples

### Phase 2 Parallel Launch

```bash
# Launch crypto and SOAP tasks together:
Task: "Implement AES-256-GCM in internal/store/crypto.go"
Task: "Implement Argon2id KDF in internal/store/crypto.go"
Task: "Implement SOAP envelope builder in internal/fonws/client.go"
```

### User Story 3 Test Launch

```bash
# Launch all US3 unit tests together:
Task: "Write unit test for GetDataboxInfo request in tests/unit/fonws_test.go"
Task: "Write unit test for GetDataboxInfo response in tests/unit/fonws_test.go"
Task: "Write unit test for ActionRequired() helper in tests/unit/fonws_test.go"
Task: "Write unit test for GetDatabox download in tests/unit/fonws_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 (Session login/logout)
4. Complete Phase 4: User Story 2 (Account management)
5. **STOP and VALIDATE**: Can add accounts, login, logout
6. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → Can authenticate → Demo
3. Add US2 → Can manage accounts → Demo
4. Add US3 → Can list/download docs → Demo
5. Add US4 → Full dashboard → Demo (Feature complete!)

### Parallel Team Strategy

With 2 developers after Foundational:
- Developer A: User Story 1 (Session)
- Developer B: User Story 2 (Accounts)

Then both converge on US3 and US4.

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Each user story is independently testable at its checkpoint
- Tests MUST fail before implementation (Constitution principle IV)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
