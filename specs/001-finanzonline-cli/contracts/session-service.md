# Session Service Contract

**Endpoint**: `https://finanzonline.bmf.gv.at/fonws/ws/sessionService`
**Protocol**: SOAP 1.1 over HTTPS
**Content-Type**: `text/xml; charset=utf-8`

## Operations

### Login

Authenticates a WebService user and returns a session token.

#### Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Login xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <tid>{teilnehmer_id}</tid>
      <benid>{benutzer_id}</benid>
      <pin>{pin}</pin>
      <heression>false</heression>
    </Login>
  </soap:Body>
</soap:Envelope>
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| tid | string(12) | Yes | Teilnehmer-ID (exactly 12 digits) |
| benid | string | Yes | Benutzer-ID (WebService user identifier) |
| pin | string | Yes | WebService PIN |
| heression | boolean | Yes | Always "false" for WebService |

#### Response (Success)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LoginResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>0</rc>
      <msg></msg>
      <id>{session_token}</id>
    </LoginResponse>
  </soap:Body>
</soap:Envelope>
```

#### Response (Error)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LoginResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>{error_code}</rc>
      <msg>{error_message}</msg>
      <id></id>
    </LoginResponse>
  </soap:Body>
</soap:Envelope>
```

| Field | Type | Description |
|-------|------|-------------|
| rc | int | Response code (0 = success, negative = error) |
| msg | string | Error message (empty on success) |
| id | string | Session token (empty on error) |

#### Error Codes

| Code | Constant | User Message |
|------|----------|--------------|
| 0 | `ErrNone` | (success) |
| -1 | `ErrSessionExpired` | Session expired. Please log in again. |
| -2 | `ErrMaintenance` | FinanzOnline is under maintenance. Try again later. |
| -3 | `ErrTechnical` | Technical error. Please try again later. |
| -4 | `ErrInvalidCredentials` | Invalid credentials. Check Teilnehmer-ID, Benutzer-ID, and PIN. |
| -5 | `ErrUserLockedTemp` | User temporarily locked. Too many failed attempts. |
| -6 | `ErrUserLockedPerm` | User permanently locked. Contact FinanzOnline support. |
| -7 | `ErrNotWebServiceUser` | Not a WebService user. Enable WebService access in FinanzOnline. |
| -8 | `ErrParticipantLocked` | Participant locked. Contact FinanzOnline support. |

---

### Logout

Terminates an active session.

#### Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <Logout xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <id>{session_token}</id>
      <tid>{teilnehmer_id}</tid>
      <benid>{benutzer_id}</benid>
    </Logout>
  </soap:Body>
</soap:Envelope>
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | Yes | Session token from Login |
| tid | string(12) | Yes | Teilnehmer-ID |
| benid | string | Yes | Benutzer-ID |

#### Response

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <LogoutResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/sessionService">
      <rc>0</rc>
      <msg></msg>
    </LogoutResponse>
  </soap:Body>
</soap:Envelope>
```

| Field | Type | Description |
|-------|------|-------------|
| rc | int | Response code (0 = success) |
| msg | string | Error message (empty on success) |

---

## Go Structs

```go
// LoginRequest represents a SOAP Login request
type LoginRequest struct {
    XMLName   xml.Name `xml:"Login"`
    Xmlns     string   `xml:"xmlns,attr"`
    TID       string   `xml:"tid"`
    BenID     string   `xml:"benid"`
    PIN       string   `xml:"pin"`
    Herstellerkennung string `xml:"herstellung"`
}

// LoginResponse represents a SOAP Login response
type LoginResponse struct {
    XMLName xml.Name `xml:"LoginResponse"`
    RC      int      `xml:"rc"`
    Msg     string   `xml:"msg"`
    ID      string   `xml:"id"`
}

// LogoutRequest represents a SOAP Logout request
type LogoutRequest struct {
    XMLName xml.Name `xml:"Logout"`
    Xmlns   string   `xml:"xmlns,attr"`
    ID      string   `xml:"id"`
    TID     string   `xml:"tid"`
    BenID   string   `xml:"benid"`
}

// LogoutResponse represents a SOAP Logout response
type LogoutResponse struct {
    XMLName xml.Name `xml:"LogoutResponse"`
    RC      int      `xml:"rc"`
    Msg     string   `xml:"msg"`
}
```
