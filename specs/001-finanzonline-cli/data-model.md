# Data Model: FinanzOnline CLI

**Date**: 2025-12-07
**Feature**: 001-finanzonline-cli

## Entities

### Account

Represents a stored FinanzOnline WebService account.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| name | string | unique, non-empty, max 100 chars | User-assigned friendly name |
| tid | string | exactly 12 digits | Teilnehmer-ID (participant ID) |
| benid | string | non-empty, max 50 chars | Benutzer-ID (WebService user ID) |
| pin | string | non-empty | WebService PIN (stored encrypted) |

**Validation Rules**:
- `name` must be unique across all accounts
- `tid` must match pattern `^\d{12}$`
- `benid` and `pin` must not be empty
- Combination `tid + benid` should be unique (same account, different name is allowed)

**Lifecycle**:
- Created: When user adds account via `fo account add`
- Updated: Not supported in v1 (delete + re-add)
- Deleted: When user removes account via `fo account remove`

### Session

Represents an active authenticated connection to FinanzOnline.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| token | string | non-empty | Session token from Login response |
| accountName | string | references Account.name | Which account this session belongs to |
| tid | string | exactly 12 digits | Teilnehmer-ID for this session |
| benid | string | non-empty | Benutzer-ID for this session |
| createdAt | timestamp | | When session was established |
| valid | boolean | | Whether session is still valid |

**Validation Rules**:
- `token` must be non-empty after successful login
- `accountName` must reference existing stored account

**Lifecycle**:
- Created: On successful Login response (rc=0)
- Invalidated: On Logout, on error code -1, or on application exit
- Not persisted: Session exists only in memory during CLI execution

**State Transitions**:
```
[none] --login--> [active] --logout--> [none]
                     |
                     +--expired (-1)--> [invalid] --re-auth--> [active]
```

### DataboxItem

Represents a document in the FinanzOnline Databox.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| applkey | string | non-empty | Application key (unique identifier) |
| filebez | string | | File identifier/reference |
| ts_zust | timestamp | | Document delivery timestamp |
| erledigungsart | string | enum: B, E, M, V | Document type code |
| veression | string | | Processing status |
| actionRequired | boolean | computed | True if erledigungsart in [E, V] |

**Validation Rules**:
- `applkey` must be non-empty (used for download)
- `ts_zust` should be parseable as ISO timestamp

**Document Type Mapping**:
| Code | Type | Action Required |
|------|------|-----------------|
| B | Bescheid (Assessment) | No |
| E | Ergänzungsersuchen (Supplementary Request) | **Yes** |
| M | Mitteilung (Notification) | No |
| V | Vorhalt (Preliminary Request) | **Yes** |

**Lifecycle**:
- Fetched: Retrieved from Databox WebService
- Not persisted: Exists only in memory during CLI execution

### CredentialStore

Represents the encrypted file containing all accounts.

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| version | int | must be 1 | Schema version for future migrations |
| accounts | []Account | | List of stored accounts |

**File Format** (after decryption):
```json
{
  "version": 1,
  "accounts": [
    {
      "name": "Holding GmbH",
      "tid": "123456789012",
      "benid": "WSUSER001",
      "pin": "geheim123"
    }
  ]
}
```

**Encryption Envelope**:
```
| Salt (16 bytes) | Nonce (12 bytes) | Ciphertext + GCM Tag |
```

## Relationships

```
CredentialStore 1──* Account
Account 1──0..1 Session (in memory only)
Session 1──* DataboxItem (fetched per request)
```

## Data Flow

```
User Input (master password)
    │
    ▼
┌─────────────────┐
│ CredentialStore │ ◄── Encrypted file on disk
└────────┬────────┘
         │ decrypt
         ▼
    ┌─────────┐
    │ Account │ (1..n)
    └────┬────┘
         │ login
         ▼
    ┌─────────┐
    │ Session │ (in memory)
    └────┬────┘
         │ getDatabox
         ▼
  ┌─────────────┐
  │ DataboxItem │ (1..n, in memory)
  └─────────────┘
         │
         ▼
    CLI Output (table/JSON)
```

## Storage Locations

| Platform | Config Directory | Credential File |
|----------|-----------------|-----------------|
| Linux | `$XDG_CONFIG_HOME/fo` or `~/.config/fo` | `accounts.enc` |
| macOS | `$XDG_CONFIG_HOME/fo` or `~/.config/fo` | `accounts.enc` |
| Windows | `%APPDATA%\fo` | `accounts.enc` |

## Validation Summary

| Entity | Field | Validation |
|--------|-------|------------|
| Account | name | unique, non-empty, max 100 chars |
| Account | tid | regex `^\d{12}$` |
| Account | benid | non-empty |
| Account | pin | non-empty |
| Session | token | non-empty (runtime) |
| DataboxItem | applkey | non-empty |
| CredentialStore | version | must equal 1 |
