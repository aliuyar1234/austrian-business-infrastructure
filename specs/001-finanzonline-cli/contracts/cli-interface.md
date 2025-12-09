# CLI Interface Contract

**Binary**: `fo`
**Description**: FinanzOnline CLI for multi-account session management and databox access

## Command Structure

```
fo [global-flags] <command> [command-flags] [args]
```

## Global Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--config` | `-c` | string | (platform default) | Path to config directory |
| `--json` | `-j` | bool | false | Output in JSON format |
| `--verbose` | `-v` | bool | false | Enable verbose logging |
| `--help` | `-h` | bool | false | Show help |
| `--version` | | bool | false | Show version |

## Commands

### fo account

Manage stored FinanzOnline accounts.

#### fo account add

Add a new account to the credential store.

```bash
fo account add <name>
```

| Argument | Required | Description |
|----------|----------|-------------|
| name | Yes | Friendly name for the account |

**Interactive prompts**:
- Teilnehmer-ID (12 digits)
- Benutzer-ID
- PIN (hidden input)
- Master password (if first account or store locked)

**Output (success)**:
```
Account "Holding GmbH" added successfully.
```

**Output (JSON)**:
```json
{"status": "success", "account": "Holding GmbH"}
```

**Exit codes**:
- 0: Success
- 1: Account name already exists
- 2: Invalid input (TID format, empty fields)
- 3: Encryption error

---

#### fo account list

List all stored accounts.

```bash
fo account list
```

**Output (table)**:
```
NAME              TID             BENUTZER-ID
Holding GmbH      123456789012    WSUSER001
Tochter GmbH 1    234567890123    WSUSER002
Tochter GmbH 2    345678901234    WSUSER003
```

**Output (JSON)**:
```json
{
  "accounts": [
    {"name": "Holding GmbH", "tid": "123456789012", "benid": "WSUSER001"},
    {"name": "Tochter GmbH 1", "tid": "234567890123", "benid": "WSUSER002"}
  ]
}
```

**Note**: PIN is never displayed.

**Exit codes**:
- 0: Success
- 3: Decryption error (wrong master password)

---

#### fo account remove

Remove an account from the credential store.

```bash
fo account remove <name>
```

| Argument | Required | Description |
|----------|----------|-------------|
| name | Yes | Name of account to remove |

**Output (success)**:
```
Account "Holding GmbH" removed.
```

**Exit codes**:
- 0: Success
- 1: Account not found
- 3: Encryption error

---

### fo session

Manage FinanzOnline sessions.

#### fo session login

Authenticate with FinanzOnline.

```bash
fo session login <account-name>
```

| Argument | Required | Description |
|----------|----------|-------------|
| account-name | Yes | Name of stored account |

**Output (success)**:
```
Logged in as "Holding GmbH" (123456789012)
```

**Output (JSON)**:
```json
{"status": "success", "account": "Holding GmbH", "tid": "123456789012"}
```

**Exit codes**:
- 0: Success
- 1: Account not found
- 4: Authentication failed (with specific error from FinanzOnline)
- 5: Network error

---

#### fo session logout

Terminate active session.

```bash
fo session logout
```

**Output (success)**:
```
Logged out.
```

**Exit codes**:
- 0: Success
- 4: No active session

---

### fo databox

Access FinanzOnline Databox documents.

#### fo databox list

List documents in databox for current session or specified account.

```bash
fo databox list [account-name]
fo databox list --all
```

| Argument/Flag | Required | Description |
|---------------|----------|-------------|
| account-name | No | Specific account (uses active session if omitted) |
| `--all` | No | Check all stored accounts |
| `--from` | No | Filter: documents from date (YYYY-MM-DD) |
| `--to` | No | Filter: documents until date (YYYY-MM-DD) |

**Output (single account, table)**:
```
TYPE                    DATE         ACTION    REFERENCE
Bescheid               2025-12-01              APP123456
Ergänzungsersuchen     2025-12-05   ⚠️         APP789012
```

**Output (--all, table)**:
```
Checking 3 accounts... done in 4s

ACCOUNT           NEW ITEMS   ACTION REQUIRED
Holding GmbH      0
Tochter GmbH 1    2
Tochter GmbH 2    1           ⚠️ Ergänzungsersuchen
```

**Output (JSON)**:
```json
{
  "accounts": [
    {
      "name": "Holding GmbH",
      "tid": "123456789012",
      "items": [],
      "newCount": 0,
      "actionRequired": false
    },
    {
      "name": "Tochter GmbH 2",
      "tid": "345678901234",
      "items": [
        {
          "applkey": "APP789012",
          "type": "Ergänzungsersuchen",
          "date": "2025-12-05T14:15:00",
          "actionRequired": true
        }
      ],
      "newCount": 1,
      "actionRequired": true
    }
  ]
}
```

**Exit codes**:
- 0: Success
- 1: Account not found
- 4: Authentication failed
- 5: Network error

---

#### fo databox download

Download a specific document.

```bash
fo databox download <applkey> [--output <path>]
```

| Argument/Flag | Required | Description |
|---------------|----------|-------------|
| applkey | Yes | Document identifier from `databox list` |
| `--output` / `-o` | No | Output path (default: current directory with original filename) |

**Output (success)**:
```
Downloaded: Bescheid_ESt_2024.pdf (42 KB)
```

**Exit codes**:
- 0: Success
- 1: Document not found
- 4: Authentication failed
- 5: Network error
- 6: File write error

---

## Exit Code Summary

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Resource not found (account, document) |
| 2 | Invalid input (validation error) |
| 3 | Encryption/decryption error |
| 4 | Authentication/session error |
| 5 | Network error |
| 6 | File I/O error |

## Output Formats

### Table (default)

Human-readable tabular output for terminal use. Includes:
- Column headers
- Aligned columns
- Unicode symbols for status (⚠️ for action required)
- Progress indicators for batch operations

### JSON (`--json`)

Machine-readable JSON output for scripting. Includes:
- Consistent field names
- ISO 8601 timestamps
- Boolean flags instead of symbols
- Error details in `error` field on failure

## Environment Variables

| Variable | Description |
|----------|-------------|
| `FO_CONFIG_DIR` | Override config directory path |
| `FO_MASTER_PASSWORD` | Master password (avoid, prefer interactive prompt) |
| `NO_COLOR` | Disable colored output |

## Examples

```bash
# Add first account (will prompt for master password setup)
fo account add "Holding GmbH"

# Add more accounts
fo account add "Tochter GmbH 1"
fo account add "Tochter GmbH 2"

# List all accounts
fo account list

# Check single account databox
fo databox list "Holding GmbH"

# Check ALL accounts (the killer feature)
fo databox list --all

# JSON output for scripting
fo databox list --all --json | jq '.accounts[] | select(.actionRequired)'

# Download a document
fo databox download APP789012 -o ~/Downloads/

# Verbose mode for debugging
fo -v databox list "Holding GmbH"
```
