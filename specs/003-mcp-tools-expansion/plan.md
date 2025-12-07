# Implementation Plan: MCP Tools Expansion

**Branch**: `003-mcp-tools-expansion` | **Date**: 2025-12-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-mcp-tools-expansion/spec.md`

## Summary

Extend the existing MCP server with 5 operational tools (`fo-databox-list`, `fo-databox-download`, `fo-fb-search`, `fo-fb-extract`, `fo-uva-submit`) that enable AI assistants to perform business operations beyond validation. All infrastructure exists - databox, Firmenbuch, and UVA services are implemented. This feature adds MCP handlers that bridge these services to the MCP protocol.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: stdlib only (encoding/json, encoding/xml, net/http) - no new dependencies
**Storage**: Existing SQLite for accounts/sessions via `internal/store`
**Testing**: `go test ./...` with unit tests in `tests/unit/`
**Target Platform**: Windows/Linux/macOS CLI
**Project Type**: Single CLI application
**Performance Goals**: Tool responses within 5-30 seconds (per spec SC-001 through SC-005)
**Constraints**: Must use existing session management; no credential exposure in responses
**Scale/Scope**: 5 new MCP tool handlers, extending existing 5 validation tools

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Delete Over Add | PASS | Adding minimal handler code; all service logic exists |
| II. Deferred Abstraction | PASS | No new abstractions; reusing existing patterns |
| III. Strict Scope Adherence | PASS | 5 tools exactly as specified |
| IV. Test-First Development | PASS | Tests will be written before handlers |
| V. Minimal Dependencies | PASS | No new dependencies required |
| VI. Obvious Over Clever | PASS | Handlers follow existing pattern exactly |
| VII. Single Responsibility | PASS | Each handler does one thing |
| VIII. Shallow Nesting | PASS | Handlers are flat call-and-return |

## Project Structure

### Documentation (this feature)

```text
specs/003-mcp-tools-expansion/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   └── mcp-tools.json   # Tool schemas
├── checklists/
│   └── requirements.md  # Completed checklist
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── mcp/
│   ├── server.go        # Existing - add 5 new RegisterTool calls
│   ├── handlers.go      # Existing - add 5 new handler functions
│   └── tools.go         # Existing - types (no changes needed)
├── fonws/
│   ├── databox.go       # Existing - DataboxService used by handlers
│   ├── uva.go           # Existing - FileUploadService used by handlers
│   └── session.go       # Existing - Session management
├── fb/
│   ├── client.go        # Existing - Search/Extract used by handlers
│   └── types.go         # Existing - types (no changes needed)
└── store/
    └── accounts.go      # Existing - Account lookup

tests/
└── unit/
    └── mcp_test.go      # Add tests for 5 new tools
```

**Structure Decision**: Extend existing `internal/mcp/` package. No new packages needed.

## Complexity Tracking

No violations. Implementation follows existing patterns exactly.
