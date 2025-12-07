# Implementation Plan: FinanzOnline CLI

**Branch**: `001-finanzonline-cli` | **Date**: 2025-12-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-finanzonline-cli/spec.md`

## Summary

Build a CLI tool for FinanzOnline WebService integration enabling multi-account session management and databox access. Primary users are tax accountants managing 30+ client accounts who need to check all databoxes for new documents in under 1 minute (vs. 2+ hours manually).

Technical approach: Go CLI with Cobra, stdlib SOAP via `encoding/xml`, encrypted credential storage with AES-256-GCM, platform-appropriate config paths (XDG/AppData).

## Technical Context

**Language/Version**: Go 1.23+
**Primary Dependencies**: Cobra (CLI framework), encoding/xml (SOAP), crypto/aes + crypto/cipher (encryption)
**Storage**: Encrypted JSON file with AES-256-GCM (master password derived key)
**Testing**: go test with integration tests against FinanzOnline sandbox (if available) or mocked SOAP responses
**Target Platform**: Windows, Linux, macOS (cross-compiled binaries)
**Project Type**: Single CLI application
**Performance Goals**: 30 accounts checked in <30 seconds (SC-001), single login <5 seconds (SC-002)
**Constraints**: No external SOAP libraries, stdlib net/http only, parallel requests via errgroup
**Scale/Scope**: 30-100 accounts per user, single-user CLI tool

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Delete Over Add | ✅ PASS | Minimal scope: session, accounts, databox only |
| II. Deferred Abstraction | ✅ PASS | No premature abstractions planned |
| III. Strict Scope Adherence | ✅ PASS | Scope matches spec exactly (4 user stories) |
| IV. Test-First Development | ✅ WILL COMPLY | Tests written before implementation |
| V. Minimal Dependencies | ✅ PASS | Only Cobra justified (see below) |
| VI. Obvious Over Clever | ✅ WILL COMPLY | Clear naming, no clever tricks |
| VII. Single Responsibility | ✅ PASS | Clear module separation planned |
| VIII. Shallow Nesting | ✅ WILL COMPLY | Early returns, max 3 levels |

### Dependency Justifications

| Dependency | Why stdlib insufficient | Maintained? | Transitive deps |
|------------|------------------------|-------------|-----------------|
| `github.com/spf13/cobra` | stdlib `flag` lacks subcommands, help generation, shell completion | Yes (6k+ stars, active) | pflag only |
| `golang.org/x/sync/errgroup` | stdlib has no coordinated goroutine error handling | Yes (Go team maintained) | 0 |

**All other functionality uses stdlib**: `net/http`, `encoding/xml`, `encoding/json`, `crypto/aes`, `crypto/cipher`, `os`, `path/filepath`.

## Project Structure

### Documentation (this feature)

```text
specs/001-finanzonline-cli/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (SOAP request/response schemas)
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
cmd/
└── fo/
    └── main.go              # CLI entry point

internal/
├── cli/
│   ├── root.go              # Cobra root command
│   ├── session.go           # fo session login/logout
│   ├── account.go           # fo account add/list/remove
│   ├── databox.go           # fo databox list/download
│   └── dashboard.go         # fo databox --all (batch)
├── fonws/
│   ├── client.go            # SOAP HTTP client
│   ├── session.go           # Session WebService operations
│   └── databox.go           # Databox WebService operations
├── store/
│   ├── accounts.go          # Account credential storage
│   └── crypto.go            # AES-256-GCM encryption
└── config/
    └── paths.go             # XDG/AppData path resolution

tests/
├── integration/
│   ├── session_test.go      # Live or mocked session tests
│   └── databox_test.go      # Live or mocked databox tests
└── unit/
    ├── store_test.go        # Credential encryption tests
    └── fonws_test.go        # SOAP parsing tests
```

**Structure Decision**: Single CLI project. Go convention with `cmd/` for binaries, `internal/` for private packages, `tests/` at root for all test types.

## Complexity Tracking

No violations requiring justification. Design follows all constitution principles.
