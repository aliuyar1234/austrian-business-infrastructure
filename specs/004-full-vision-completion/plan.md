# Implementation Plan: Full Vision Completion

**Branch**: `004-full-vision-completion` | **Date**: 2025-12-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-full-vision-completion/spec.md`

## Summary

Complete all 5 Austrian Business Infrastructure modules to full operational status. The project already has Go 1.23+ with Cobra CLI, encrypted credential storage (AES-256-GCM/Argon2), and FinanzOnline MVP. This plan extends the existing architecture to add full ELDA, Firmenbuch, E-Rechnung, and SEPA functionality, plus expanded MCP tools.

## Technical Context

**Language/Version**: Go 1.23+
**Primary Dependencies**: Cobra v1.8.1 (CLI), golang.org/x/crypto (encryption), golang.org/x/sync (parallelism), mark3labs/mcp-go v0.43.2 (MCP server)
**Storage**: Encrypted file-based credential store (`accounts.enc`), no external database
**Testing**: Go standard testing (`go test`), integration tests with mocks for external APIs
**Target Platform**: Windows, Linux, macOS (cross-platform CLI)
**Project Type**: Single CLI application with modular internal packages
**Performance Goals**: 30 accounts in <60s, single operation <5s, 1000+ SEPA payments per batch
**Constraints**: No external database, offline-capable for local operations, <100ms encryption overhead
**Scale/Scope**: 5 modules, ~15 CLI commands, ~15 MCP tools, targeting tax accountants with 30+ clients

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Delete Over Add | PASS | Extending existing modules, not adding new abstractions |
| II. Deferred Abstraction | PASS | Using established patterns from fonws module |
| III. Strict Scope Adherence | PASS | 39 FRs defined, no scope creep |
| IV. Test-First Development | PASS | Integration tests exist, will extend |
| V. Minimal Dependencies | PASS | Only stdlib + existing deps (Cobra, crypto, mcp-go) |
| VI. Obvious Over Clever | PASS | Following existing code patterns |
| VII. Single Responsibility | PASS | Each module has clear boundary |
| VIII. Shallow Nesting | PASS | Existing code uses early returns |

**Gate Result**: PASS - No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/004-full-vision-completion/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (CLI interface contracts)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── fo/
    └── main.go              # CLI entry point

internal/
├── cli/                     # Cobra CLI commands
│   ├── root.go              # Root command, global flags
│   ├── account.go           # Account management
│   ├── dashboard.go         # Multi-service dashboard
│   ├── databox.go           # FinanzOnline databox
│   ├── elda.go              # ELDA commands (EXTEND)
│   ├── erechnung.go         # E-Rechnung commands (EXTEND)
│   ├── fb.go                # Firmenbuch commands (EXTEND)
│   ├── mcp.go               # MCP server command
│   ├── sepa.go              # SEPA commands (EXTEND)
│   ├── session.go           # Session management
│   ├── uid.go               # UID validation
│   ├── uva.go               # UVA submission (EXTEND)
│   └── zm.go                # ZM submission (EXTEND)
├── config/                  # Platform paths
│   └── paths.go
├── elda/                    # ELDA SOAP client (COMPLETE)
│   ├── client.go
│   ├── types.go
│   └── validation.go
├── erechnung/               # E-Invoice toolkit (COMPLETE)
│   ├── invoice.go
│   ├── validate.go
│   ├── xrechnung.go
│   └── zugferd.go
├── fb/                      # Firmenbuch client (COMPLETE)
│   ├── client.go
│   └── types.go
├── fonws/                   # FinanzOnline WebService (EXTEND)
│   ├── client.go
│   ├── databox.go
│   ├── errors.go
│   ├── session.go
│   ├── uid.go
│   ├── uva.go               # Complete submission
│   └── zm.go                # Complete submission
├── mcp/                     # MCP server (EXTEND)
│   ├── server.go
│   ├── tools.go
│   └── handlers.go
├── sepa/                    # SEPA toolkit (COMPLETE)
│   ├── bic.go
│   ├── camt053.go
│   ├── iban.go
│   ├── pain001.go
│   ├── pain008.go
│   └── types.go
└── store/                   # Credential storage
    ├── accounts.go
    └── crypto.go

tests/
├── integration/
│   ├── databox_test.go
│   ├── session_test.go
│   └── store_test.go
└── unit/
    ├── dashboard_test.go
    ├── elda_test.go         # Extend
    ├── erechnung_test.go    # Extend
    ├── fb_test.go           # Extend
    ├── fonws_test.go
    ├── mcp_test.go          # Extend
    ├── sepa_test.go         # Extend
    ├── store_test.go
    ├── uid_test.go
    ├── uva_test.go          # Extend
    └── zm_test.go           # Extend
```

**Structure Decision**: Single project structure maintained. All modules under `internal/` with corresponding CLI commands in `internal/cli/`. Follows established Go project layout.

## Complexity Tracking

> No violations detected. Table not required.
