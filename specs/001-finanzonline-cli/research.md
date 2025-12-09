# Research: FinanzOnline CLI

**Date**: 2025-12-07
**Feature**: 001-finanzonline-cli

## Decision Summary

| Topic | Decision | Rationale |
|-------|----------|-----------|
| SOAP Implementation | stdlib `encoding/xml` | Constitution V (Minimal Dependencies), no external SOAP lib needed |
| HTTP Client | stdlib `net/http` | Constitution V, sufficient for SOAP over HTTPS |
| CLI Framework | Cobra | Only justified external dep, stdlib flag lacks subcommands |
| Credential Encryption | AES-256-GCM with Argon2id KDF | Industry standard, master password derived key |
| Config Paths | XDG (Linux/macOS), AppData (Windows) | Platform conventions, cross-platform support |
| Parallel Requests | errgroup | Go team maintained, coordinated error handling |

## FinanzOnline WebService API

### Endpoints

| Service | WSDL URL |
|---------|----------|
| Session | `https://finanzonline.bmf.gv.at/fonws/ws/sessionService.wsdl` |
| Databox | `https://finanzonline.bmf.gv.at/fonws/ws/databoxService.wsdl` |
| Upload | `https://finanzonline.bmf.gv.at/fonws/ws/fileUploadService.wsdl` |
| UID | `https://finanzonline.bmf.gv.at/fonws/ws/uidService.wsdl` |

**Note**: Only Session and Databox services are in scope for this feature.

### Session WebService

#### Login Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Login xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <tid>123456789012</tid>
      <benid>WSUSER001</benid>
      <pin>geheim123</pin>
      <heression>false</heression>
    </Login>
  </soap:Body>
</soap:Envelope>
```

#### Login Response (Success)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LoginResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>0</rc>
      <msg></msg>
      <id>SESSION_TOKEN_STRING</id>
    </LoginResponse>
  </soap:Body>
</soap:Envelope>
```

#### Logout Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Logout xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <id>SESSION_TOKEN_STRING</id>
      <tid>123456789012</tid>
      <benid>WSUSER001</benid>
    </Logout>
  </soap:Body>
</soap:Envelope>
```

#### Response Codes

| Code | Meaning | User Message |
|------|---------|--------------|
| 0 | Success | Login successful |
| -1 | Session expired | Session has expired. Please log in again. |
| -2 | Maintenance | FinanzOnline is currently under maintenance. |
| -3 | Technical error | A technical error occurred. Please try again later. |
| -4 | Invalid credentials | Invalid credentials. Check Teilnehmer-ID, Benutzer-ID, and PIN. |
| -5 | User locked (temp) | User account is temporarily locked due to failed login attempts. |
| -6 | User locked (perm) | User account is permanently locked. Contact FinanzOnline support. |
| -7 | Not WebService user | This user is not enabled for WebService access. |
| -8 | Participant locked | The participant (organization) is locked. Contact FinanzOnline support. |

### Databox WebService

#### GetDataboxInfo Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfo xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <id>SESSION_TOKEN_STRING</id>
      <tid>123456789012</tid>
      <benid>WSUSER001</benid>
      <ts_zust_von></ts_zust_von>
      <ts_zust_bis></ts_zust_bis>
    </GetDataboxInfo>
  </soap:Body>
</soap:Envelope>
```

#### GetDataboxInfo Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfoResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <databox>
          <applkey>APP123</applkey>
          <fiession>FI456</fiession>
          <ts_zust>2025-12-01T10:30:00</ts_zust>
          <erlession>E</erlession>
          <veression>B</veression>
        </databox>
        <!-- more databox entries -->
      </result>
    </GetDataboxInfoResponse>
  </soap:Body>
</soap:Envelope>
```

#### Document Types

| Code | Type | Description | Action Required |
|------|------|-------------|-----------------|
| B | Bescheid | Tax assessment notice | No |
| E | Ergänzungsersuchen | Supplementary request | **Yes** |
| M | Mitteilung | Notification | No |
| V | Vorhalt | Preliminary request | Yes |

### Authentication Flow

```
1. User provides master password
2. Decrypt credential store
3. Select account (by name or --all)
4. Build SOAP Login request with tid/benid/pin
5. POST to sessionService endpoint
6. Parse response, extract session token or error
7. Store session token in memory for subsequent calls
8. On session expiry (-1), re-authenticate automatically
9. On logout or exit, call Logout endpoint
```

## Credential Storage Design

### Storage Location

| Platform | Path |
|----------|------|
| Linux | `$XDG_CONFIG_HOME/fo/accounts.enc` (default: `~/.config/fo/accounts.enc`) |
| macOS | `$XDG_CONFIG_HOME/fo/accounts.enc` (default: `~/.config/fo/accounts.enc`) |
| Windows | `%APPDATA%\fo\accounts.enc` |

### Encryption Scheme

```
Algorithm: AES-256-GCM
Key Derivation: Argon2id (memory-hard, recommended for password hashing)
  - Time: 1 iteration
  - Memory: 64 MB
  - Parallelism: 4 threads
  - Salt: 16 bytes random, stored with encrypted data
  - Key length: 32 bytes

File Format:
  - Bytes 0-15: Salt (16 bytes)
  - Bytes 16-27: Nonce (12 bytes)
  - Bytes 28+: Ciphertext with GCM tag
```

### Plaintext Format (before encryption)

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

## Alternatives Considered

### SOAP Libraries

| Option | Rejected Because |
|--------|-----------------|
| gowsdl | External dep, code generation complexity, Constitution V violation |
| gosoap | External dep, unmaintained, Constitution V violation |
| **stdlib encoding/xml** | **Selected**: No deps, sufficient for simple SOAP requests |

### Credential Storage

| Option | Rejected Because |
|--------|-----------------|
| OS Keychain | Platform-specific APIs, complex cross-platform code |
| SQLite + sqlcipher | External dep (CGO), Constitution V violation |
| Plain file + permissions | Insufficient security for multi-user systems |
| **Encrypted JSON file** | **Selected**: Simple, cross-platform, no external deps |

### Key Derivation

| Option | Rejected Because |
|--------|-----------------|
| PBKDF2 | Less resistant to GPU attacks |
| bcrypt | Designed for hashing, not key derivation |
| scrypt | Good but Argon2id is newer standard |
| **Argon2id** | **Selected**: PHC winner, memory-hard, recommended for KDF |

**Note**: Argon2 requires `golang.org/x/crypto/argon2` which is Go team maintained with 0 transitive deps.

## Open Questions (Resolved)

| Question | Resolution |
|----------|-----------|
| Is FinanzOnline sandbox available? | No public sandbox; use mocked SOAP responses for tests |
| Session token expiry time? | Unknown; handle -1 error code for expiration |
| Rate limiting on API? | Unknown; implement exponential backoff for errors |
| Concurrent session limit? | Unknown; parallel requests from single session likely allowed |

## Dependencies Final List

| Dependency | Type | Justification |
|------------|------|---------------|
| `github.com/spf13/cobra` | External | CLI subcommands, help generation, completions |
| `golang.org/x/sync/errgroup` | Go extended | Parallel request coordination with error handling |
| `golang.org/x/crypto/argon2` | Go extended | Password-based key derivation (memory-hard) |

All other functionality uses stdlib: `net/http`, `encoding/xml`, `encoding/json`, `crypto/aes`, `crypto/cipher`, `crypto/rand`, `os`, `path/filepath`, `fmt`, `io`, `time`.
