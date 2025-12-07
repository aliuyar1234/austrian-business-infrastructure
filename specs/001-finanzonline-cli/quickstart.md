# Quickstart: FinanzOnline CLI

## Prerequisites

1. **FinanzOnline WebService credentials** for each account:
   - Teilnehmer-ID (12-digit participant ID)
   - Benutzer-ID (WebService user ID, not your regular login)
   - WebService PIN (not your regular password)

   > **Note**: WebService users bypass 2FA. You must enable WebService access for each user in FinanzOnline portal under "Benutzer verwalten".

2. **Go 1.23+** installed (for building from source)

## Installation

### From Binary (Recommended)

Download the latest release for your platform:

```bash
# Linux (amd64)
curl -LO https://github.com/your-org/fo/releases/latest/download/fo-linux-amd64
chmod +x fo-linux-amd64
sudo mv fo-linux-amd64 /usr/local/bin/fo

# macOS (arm64)
curl -LO https://github.com/your-org/fo/releases/latest/download/fo-darwin-arm64
chmod +x fo-darwin-arm64
sudo mv fo-darwin-arm64 /usr/local/bin/fo

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/your-org/fo/releases/latest/download/fo-windows-amd64.exe" -OutFile fo.exe
Move-Item fo.exe "$env:LOCALAPPDATA\Microsoft\WindowsApps\fo.exe"
```

### From Source

```bash
git clone https://github.com/your-org/austrian-business-infrastructure.git
cd austrian-business-infrastructure
go build -o fo ./cmd/fo
```

## First-Time Setup

### 1. Add Your First Account

```bash
fo account add "Holding GmbH"
```

You will be prompted for:
- **Teilnehmer-ID**: Enter your 12-digit participant ID
- **Benutzer-ID**: Enter your WebService user ID
- **PIN**: Enter your WebService PIN (hidden)
- **Master password**: Create a password to encrypt your credentials (first time only)

### 2. Add More Accounts

```bash
fo account add "Tochter GmbH 1"
fo account add "Tochter GmbH 2"
# ... repeat for all accounts
```

### 3. Verify Your Accounts

```bash
fo account list
```

Expected output:
```
NAME              TID             BENUTZER-ID
Holding GmbH      123456789012    WSUSER001
Tochter GmbH 1    234567890123    WSUSER002
Tochter GmbH 2    345678901234    WSUSER003
```

## Daily Workflow

### Check All Accounts (The Killer Feature)

```bash
fo databox list --all
```

This checks all stored accounts in parallel and shows:

```
Checking 30 accounts... done in 12s

ACCOUNT             NEW ITEMS   ACTION REQUIRED
Holding GmbH        0
Tochter GmbH 1      2
Tochter GmbH 3      1           ⚠️ Ergänzungsersuchen
Tochter GmbH 17     1
...

Total: 4 new items across 30 accounts
Action required: 1 account(s)
```

**Time saved**: 2+ hours → 12 seconds

### Check Single Account

```bash
fo databox list "Holding GmbH"
```

Output:
```
TYPE                    DATE         ACTION    REFERENCE
Bescheid               2025-12-01              APP123456
Ergänzungsersuchen     2025-12-05   ⚠️         APP789012
```

### Download a Document

```bash
fo databox download APP789012
```

Downloads to current directory:
```
Downloaded: Ergaenzungsersuchen_2025.pdf (156 KB)
```

Or specify output path:
```bash
fo databox download APP789012 -o ~/Documents/Tax/
```

## Scripting & Automation

### JSON Output

Add `--json` to any command for machine-readable output:

```bash
fo databox list --all --json
```

```json
{
  "accounts": [
    {
      "name": "Holding GmbH",
      "tid": "123456789012",
      "newCount": 0,
      "actionRequired": false,
      "items": []
    },
    {
      "name": "Tochter GmbH 3",
      "tid": "345678901234",
      "newCount": 1,
      "actionRequired": true,
      "items": [
        {
          "applkey": "APP789012",
          "type": "Ergänzungsersuchen",
          "date": "2025-12-05T14:15:00",
          "actionRequired": true
        }
      ]
    }
  ]
}
```

### Example: Daily Check Script

```bash
#!/bin/bash
# daily-check.sh - Run daily via cron

result=$(fo databox list --all --json)
action_count=$(echo "$result" | jq '[.accounts[] | select(.actionRequired)] | length')

if [ "$action_count" -gt 0 ]; then
    echo "⚠️ $action_count account(s) require action!"
    echo "$result" | jq '.accounts[] | select(.actionRequired) | .name'
    # Send notification, email, etc.
fi
```

### Example: Filter Accounts Requiring Action

```bash
fo databox list --all --json | jq '.accounts[] | select(.actionRequired)'
```

## Troubleshooting

### "Not a WebService user" Error

Your user account doesn't have WebService access enabled. In FinanzOnline:
1. Log in with admin account
2. Go to "Benutzer verwalten"
3. Select the user
4. Enable "WebService-Zugang"
5. Note the WebService-PIN (different from portal password)

### "Invalid credentials" Error

Check:
- Teilnehmer-ID is exactly 12 digits
- You're using the WebService Benutzer-ID (not portal username)
- You're using the WebService PIN (not portal password)

### "Session expired" Error

Your session timed out. The CLI will automatically re-authenticate on next command.

### Master Password Forgotten

If you forget your master password, you must delete the credential store and re-add all accounts:

```bash
# Linux/macOS
rm ~/.config/fo/accounts.enc

# Windows
del %APPDATA%\fo\accounts.enc
```

## Configuration

### Config File Location

| Platform | Path |
|----------|------|
| Linux | `~/.config/fo/` |
| macOS | `~/.config/fo/` |
| Windows | `%APPDATA%\fo\` |

### Override Config Path

```bash
fo --config /custom/path account list
```

Or set environment variable:
```bash
export FO_CONFIG_DIR=/custom/path
```

## Getting Help

```bash
fo --help           # General help
fo account --help   # Account commands help
fo databox --help   # Databox commands help
```

## Next Steps

- Add all your client accounts
- Set up a daily cron job to check for new documents
- Use JSON output for integration with your workflow tools
