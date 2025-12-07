# Implementation Plan: Austrian Business Infrastructure - Complete Product Suite

**Branch**: `002-vision-completion-roadmap` | **Date**: 2025-12-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-vision-completion-roadmap/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Extends the existing FinanzOnline CLI (Spec 001) to a comprehensive Austrian business infrastructure suite covering:

1. **FinanzOnline Extensions**: UVA submission, UID validation, Zusammenfassende Meldung via SOAP services
2. **ELDA Suite**: Social security reporting (employee registration, deregistration, changes)
3. **Firmenbuch Suite**: Company registry queries and monitoring
4. **E-Rechnung Suite**: ZUGFeRD/XRechnung creation, validation, extraction
5. **SEPA Toolkit**: pain.001/008 generation, camt.053/054 parsing, IBAN validation
6. **MCP Server**: Model Context Protocol integration for AI assistants

Technical approach: Extend existing Go CLI architecture with new modules following established patterns (SOAP client, encrypted credential store, Cobra commands, JSON/table output).

## Technical Context

**Language/Version**: Go 1.23+
**Primary Dependencies**: Cobra (CLI), errgroup (parallelism), argon2 (KDF), encoding/xml (SOAP/XML)
**Storage**: AES-256-GCM encrypted file-based credential store (existing from Spec 001)
**Testing**: Go testing with mocked SOAP servers, integration tests for workflows
**Target Platform**: Windows, Linux, macOS
**Project Type**: Single project (extending existing `cmd/fo/`, `internal/`)
**Performance Goals**: 50 UVA submissions in <30min, 1000 UID validations in <5min, 500 SEPA payments in <30sec
**Constraints**: No cloud dependency, credentials never logged, cross-platform identical behavior
**Scale/Scope**: 6 new modules, ~35 functional requirements, 8 user stories

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Delete Over Add | ✅ PASS | Extends existing foundation, no duplicate infrastructure |
| II. Deferred Abstraction | ✅ PASS | No premature abstractions - each module handles its own service |
| III. Strict Scope Adherence | ✅ PASS | 35 requirements from spec, no "nice to have" features |
| IV. Test-First Development | ✅ REQUIRED | All implementations require failing tests first |
| V. Minimal Dependencies | ✅ PASS | Stdlib preferred: encoding/xml, net/http, crypto/aes |
| VI. Obvious Over Clever | ✅ PASS | Follow existing patterns from Spec 001 |
| VII. Single Responsibility | ✅ PASS | Each module (fonws, elda, fb, erechnung, sepa, mcp) isolated |
| VIII. Shallow Nesting | ✅ PASS | Follow existing code patterns, max 3 levels |

**Pre-Phase 0 Gate**: PASSED - No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/002-vision-completion-roadmap/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
└── fo/
    └── main.go           # CLI entry point (existing)

internal/
├── cli/                  # Cobra commands (existing)
│   ├── root.go           # Root command with global flags
│   ├── account.go        # Account management (existing)
│   ├── session.go        # Session login/logout (existing)
│   ├── databox.go        # Databox commands (existing)
│   ├── dashboard.go      # Parallel account check (existing)
│   ├── uva.go            # NEW: UVA submission commands
│   ├── uid.go            # NEW: UID validation commands
│   └── zm.go             # NEW: Zusammenfassende Meldung commands
├── config/               # Configuration paths (existing)
│   └── paths.go          # XDG/AppData resolution
├── store/                # Encrypted credential store (existing)
│   ├── accounts.go       # Account CRUD operations
│   └── crypto.go         # AES-256-GCM, Argon2id
├── fonws/                # FinanzOnline WebService (existing + extensions)
│   ├── client.go         # SOAP client
│   ├── errors.go         # Error code mapping
│   ├── session.go        # Login/Logout
│   ├── databox.go        # Databox operations (existing)
│   ├── uva.go            # NEW: UVA SOAP operations
│   ├── uid.go            # NEW: UID validation SOAP
│   └── zm.go             # NEW: ZM SOAP operations
├── elda/                 # NEW: ELDA social security module
│   ├── client.go         # ELDA SOAP client
│   ├── anmeldung.go      # Employee registration
│   ├── abmeldung.go      # Employee deregistration
│   └── status.go         # Status queries
├── fb/                   # NEW: Firmenbuch module
│   ├── client.go         # Firmenbuch API client
│   ├── search.go         # Company search
│   ├── extract.go        # Extract data
│   └── monitor.go        # Change monitoring
├── erechnung/            # NEW: E-Rechnung module
│   ├── xrechnung.go      # XRechnung generation
│   ├── zugferd.go        # ZUGFeRD generation
│   ├── validate.go       # EN16931 validation
│   └── extract.go        # PDF/XML extraction
├── sepa/                 # NEW: SEPA module
│   ├── pain001.go        # Credit Transfer generation
│   ├── pain008.go        # Direct Debit generation
│   ├── camt053.go        # Statement parsing
│   ├── camt054.go        # Detail parsing
│   └── iban.go           # IBAN/BIC validation
└── mcp/                  # NEW: MCP Server module
    ├── server.go         # MCP protocol server
    ├── tools.go          # Tool definitions
    └── handlers.go       # Tool handlers

tests/
├── unit/
│   ├── fonws_test.go     # SOAP tests (existing)
│   ├── store_test.go     # Credential store tests (existing)
│   ├── dashboard_test.go # Dashboard tests (existing)
│   ├── uva_test.go       # NEW: UVA tests
│   ├── uid_test.go       # NEW: UID tests
│   ├── elda_test.go      # NEW: ELDA tests
│   ├── fb_test.go        # NEW: Firmenbuch tests
│   ├── erechnung_test.go # NEW: E-Rechnung tests
│   └── sepa_test.go      # NEW: SEPA tests
└── integration/
    ├── session_test.go   # Session flow tests (existing)
    ├── store_test.go     # Store flow tests (existing)
    ├── databox_test.go   # Databox flow tests (existing)
    ├── uva_test.go       # NEW: UVA workflow tests
    ├── uid_test.go       # NEW: UID workflow tests
    ├── elda_test.go      # NEW: ELDA workflow tests
    └── mcp_test.go       # NEW: MCP server tests
```

**Structure Decision**: Single project extending existing Spec 001 structure. All new modules follow the established pattern: `internal/<module>/` for business logic, `internal/cli/<command>.go` for CLI commands, `tests/unit/` and `tests/integration/` for testing.

## Complexity Tracking

> **No violations requiring justification**

All new modules follow established patterns from Spec 001. No new abstractions beyond what's needed for each service integration.

