# Research: MCP Tools Expansion

**Feature**: 003-mcp-tools-expansion
**Date**: 2025-12-07

## Executive Summary

This research confirms that all required infrastructure for the 5 new MCP tools already exists in the codebase. The implementation requires only adding handler functions and tool registrations to the existing MCP server.

## Existing Infrastructure Analysis

### 1. MCP Server (`internal/mcp/`)

The MCP server is fully implemented with:

| Component | File | Status |
|-----------|------|--------|
| Server struct | `server.go:18-23` | Ready |
| Tool registration | `server.go:38-43` | Pattern established |
| Handler type | `server.go:26` | `ToolHandler func(params map[string]interface{}) (interface{}, error)` |
| JSON-RPC handling | `server.go:169-207` | Ready |
| Existing tools | `server.go:71-166` | 5 validation tools registered |

**Pattern for new tools**: Each tool needs:
1. `RegisterTool()` call with `MCPTool` definition in `server.go`
2. Handler function in `handlers.go`

### 2. Databox Service (`internal/fonws/databox.go`)

| Method | Line | Description | MCP Tool |
|--------|------|-------------|----------|
| `GetInfo()` | 100-127 | Lists databox documents | `fo-databox-list` |
| `DownloadToBytes()` | 177-208 | Downloads document content | `fo-databox-download` |

**Dependencies**: Requires active `*Session` from session management.

**Return types**:
- `GetInfo()`: `[]DataboxEntry` with Applkey, Filebez, TsZust, Erlession
- `DownloadToBytes()`: `([]byte, string, error)` - content, filename, error

### 3. Firmenbuch Client (`internal/fb/client.go`)

| Method | Line | Description | MCP Tool |
|--------|------|-------------|----------|
| `Search()` | 129-136 | Searches by name/criteria | `fo-fb-search` |
| `Extract()` | 139-156 | Gets full company details | `fo-fb-extract` |

**No session required**: Uses API key authentication (configured in client).

**Return types**:
- `Search()`: `*FBSearchResponse` with Results array
- `Extract()`: `*FBExtract` with full company data

### 4. UVA Submission (`internal/fonws/uva.go`)

| Method | Line | Description | MCP Tool |
|--------|------|-------------|----------|
| `SubmitUVA()` | 270-288 | Submits UVA to FinanzOnline | `fo-uva-submit` |

**Dependencies**: Requires sessionID, tid, benid from active session.

**Return type**: `*FileUploadResponse` with RC, Msg, Belegnummer

### 5. Session Management (`internal/fonws/session.go`)

Sessions are managed externally. MCP tools will need to:
- Accept account identifier as parameter
- Look up session from store
- Return auth error if no active session

This mirrors the existing databox CLI pattern in `internal/cli/databox.go`.

## Implementation Design

### Handler Pattern

All new handlers follow the existing pattern from `handlers.go`:

```go
func handleToolName(params map[string]interface{}) (interface{}, error) {
    // 1. Extract and validate parameters
    param, ok := params["param_name"].(string)
    if !ok || param == "" {
        return nil, errors.New("missing required parameter: param_name")
    }

    // 2. Call existing service
    result, err := service.Method(...)
    if err != nil {
        return nil, err
    }

    // 3. Return structured result
    return map[string]interface{}{
        "field1": result.Field1,
        "field2": result.Field2,
    }, nil
}
```

### Session Handling for Databox/UVA Tools

These tools require authentication. The MCP handler will:
1. Accept `account_id` parameter
2. Load account from store
3. Get active session for account
4. Return structured error if no session (FR-019)

Error structure for auth failures:
```json
{
    "error": true,
    "error_type": "authentication",
    "message": "No active session. Use 'fo session login <account>' first."
}
```

### Tool Input Schemas

Each tool requires a JSON Schema for MCP discovery:

| Tool | Required Params | Optional Params |
|------|-----------------|-----------------|
| fo-databox-list | account_id | from_date, to_date |
| fo-databox-download | account_id, document_id | - |
| fo-fb-search | query | max_results |
| fo-fb-extract | fn | - |
| fo-uva-submit | account_id, year, period_type, period_value, kz_values | - |

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Session expiry mid-request | Medium | Existing `IsSessionExpired()` check returns clear error |
| Large databox documents | Low | Existing `DownloadToBytes()` handles; spec allows declining >10MB |
| Invalid FN format | Low | Existing `ValidateFN()` provides validation |
| UVA validation failures | Medium | Existing `ValidateUVA()` provides pre-submit validation |

## Conclusion

Implementation is straightforward:
1. Add 5 handler functions to `handlers.go` (~20-30 lines each)
2. Add 5 `RegisterTool()` calls to `server.go` (~15-20 lines each)
3. Add tests to `mcp_test.go` (~50-100 lines)

Total estimated code: ~250-350 lines of Go.

No new dependencies. No architectural changes. No new abstractions.
