# Quickstart: MCP Tools Expansion

**Feature**: 003-mcp-tools-expansion
**Date**: 2025-12-07

## Prerequisites

- Existing `fo` CLI built and working
- FinanzOnline account configured via `fo account add`
- Active session via `fo session login` (for databox/UVA tools)
- Firmenbuch API key configured (for fb tools)

## Using the New MCP Tools

### 1. Start the MCP Server

```bash
fo mcp serve
```

The server runs on stdio transport, communicating via JSON-RPC.

### 2. List Available Tools

Send to stdin:
```json
{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}
```

Response includes all 10 tools (5 existing + 5 new).

### 3. Tool Examples

#### Databox List

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "fo-databox-list",
    "arguments": {
      "account_id": "my-company"
    }
  }
}
```

Response:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "content": [{
      "type": "text",
      "text": "{\"documents\": [...], \"count\": 5}"
    }],
    "isError": false
  }
}
```

#### Databox Download

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "fo-databox-download",
    "arguments": {
      "account_id": "my-company",
      "document_id": "ABC123XYZ"
    }
  }
}
```

#### Firmenbuch Search

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "fo-fb-search",
    "arguments": {
      "query": "Österreichische Post"
    }
  }
}
```

#### Firmenbuch Extract

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "fo-fb-extract",
    "arguments": {
      "fn": "FN169379k"
    }
  }
}
```

#### UVA Submit

```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "fo-uva-submit",
    "arguments": {
      "account_id": "my-company",
      "year": 2025,
      "period_type": "monthly",
      "period_value": 1,
      "kz_values": {
        "kz000": 10000000,
        "kz017": 8000000,
        "kz060": 500000
      }
    }
  }
}
```

## Error Handling

All tools return consistent error structure:

```json
{
  "error": true,
  "error_type": "authentication",
  "message": "No active session. Use 'fo session login <account>' first."
}
```

Error types:
- `authentication`: No active session or session expired
- `validation`: Invalid input parameters
- `service`: External service error (FinanzOnline, Firmenbuch)
- `not_found`: Document or company not found

## Testing Locally

Run the MCP test suite:

```bash
go test ./tests/unit/... -run TestMCP -v
```

## Claude Desktop Integration

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "austrian-business": {
      "command": "fo",
      "args": ["mcp", "serve"]
    }
  }
}
```

Then Claude can use all 10 tools directly.
