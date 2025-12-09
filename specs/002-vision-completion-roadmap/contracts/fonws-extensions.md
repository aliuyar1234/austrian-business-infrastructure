# Contracts: FinanzOnline WebService Extensions

**Module**: fonws (extensions to existing)
**Date**: 2025-12-07

## 1. UID-Abfrage Service

**WSDL**: `https://finanzonline.bmf.gv.at/fonuid/ws/uidAbfrageService.wsdl`

### 1.1 UID Validation Request

```xml
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
                  xmlns:uid="http://finanzonline.bmf.gv.at/fonuid">
    <soapenv:Body>
        <uid:uidAbfrage>
            <uid:tid>123456789</uid:tid>
            <uid:benid>WEBSERVICE</uid:benid>
            <uid:id>SESSION_ID</uid:id>
            <uid:uid_tn>ATU12345678</uid:uid_tn>
            <uid:stuession>1</uid:stuession>
        </uid:uidAbfrage>
    </soapenv:Body>
</soapenv:Envelope>
```

**Parameters**:
- `tid`: Teilnehmer-ID (participant ID)
- `benid`: Benutzer-ID (user ID)
- `id`: Session ID from login
- `uid_tn`: UID number to validate
- `stuession`: Level (1 = basic, 2 = with requester info)

### 1.2 UID Validation Response (Success)

```xml
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
    <soapenv:Body>
        <uidAbfrageResponse>
            <rc>0</rc>
            <msg>OK</msg>
            <uid_tn>ATU12345678</uid_tn>
            <gueltig>true</gueltig>
            <name>Musterfirma GmbH</name>
            <adr_strasse>Musterstraße 1</adr_strasse>
            <adr_plz>1010</adr_plz>
            <adr_ort>Wien</adr_ort>
        </uidAbfrageResponse>
    </soapenv:Body>
</soapenv:Envelope>
```

### 1.3 UID Validation Response (Error)

```xml
<uidAbfrageResponse>
    <rc>1513</rc>
    <msg>Tageslimit für diese UID überschritten</msg>
    <uid_tn>ATU12345678</uid_tn>
    <gueltig></gueltig>
</uidAbfrageResponse>
```

### 1.4 Error Codes

| Code | Message | Action |
|------|---------|--------|
| 0 | OK | Success |
| -1 | Fehler im Webservice | Retry |
| -2 | Keine Session | Re-login |
| 1513 | Tageslimit überschritten | Use VIES or wait |
| 1514 | UID ungültig | Return invalid |

---

## 2. File-Upload Service (UVA/ZM)

**WSDL**: `https://finanzonline.bmf.gv.at/fonws/ws/fileUploadService.wsdl`

### 2.1 UVA Upload Request

```xml
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
                  xmlns:fon="http://finanzonline.bmf.gv.at/fon">
    <soapenv:Body>
        <fon:upload>
            <fon:tid>123456789</fon:tid>
            <fon:benid>WEBSERVICE</fon:benid>
            <fon:id>SESSION_ID</fon:id>
            <fon:art>U30</fon:art>
            <fon:uebession>
                <fon:data><![CDATA[BASE64_ENCODED_XML]]></fon:data>
            </fon:uebession>
        </fon:upload>
    </soapenv:Body>
</soapenv:Envelope>
```

**Parameters**:
- `art`: Document type ("U30" for UVA, "ZM" for Zusammenfassende Meldung)
- `data`: Base64-encoded XML content

### 2.2 UVA XML Format (U30)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Umsatzsteuervoranmeldung xmlns="http://www.bmf.gv.at/steuern/fon/u30">
    <Steuernummer>12-345/6789</Steuernummer>
    <Zeitraum>
        <Jahr>2025</Jahr>
        <Monat>01</Monat>
    </Zeitraum>
    <Kennzahlen>
        <KZ000>100000</KZ000>
        <KZ017>80000</KZ017>
        <KZ060>16000</KZ060>
        <KZ095>16000</KZ095>
    </Kennzahlen>
</Umsatzsteuervoranmeldung>
```

### 2.3 Upload Response

```xml
<uploadResponse>
    <rc>0</rc>
    <msg>Übermittlung erfolgreich</msg>
    <belegnummer>FON-2025-12345678</belegnummer>
</uploadResponse>
```

### 2.4 Error Codes

| Code | Message | Action |
|------|---------|--------|
| 0 | OK | Success |
| -1 | Webservice Fehler | Retry |
| -2 | Session ungültig | Re-login |
| -3 | XML ungültig | Fix XML |
| -4 | Steuernummer nicht berechtigt | Check account |

---

## 3. ZM XML Format

### 3.1 Zusammenfassende Meldung XML

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ZusammenfassendeMeldung xmlns="http://www.bmf.gv.at/steuern/fon/zm">
    <Steuernummer>12-345/6789</Steuernummer>
    <Zeitraum>
        <Jahr>2025</Jahr>
        <Quartal>1</Quartal>
    </Zeitraum>
    <Meldungen>
        <Meldung>
            <PartnerUID>DE123456789</PartnerUID>
            <Laendercode>DE</Laendercode>
            <Art>L</Art>
            <Betrag>50000</Betrag>
        </Meldung>
        <Meldung>
            <PartnerUID>IT12345678901</PartnerUID>
            <Laendercode>IT</Laendercode>
            <Art>S</Art>
            <Betrag>25000</Betrag>
        </Meldung>
    </Meldungen>
</ZusammenfassendeMeldung>
```

**Art Codes**:
- `L`: Lieferungen (goods)
- `D`: Dreiecksgeschäfte (triangular transactions)
- `S`: Sonstige Leistungen (services)

---

## 4. Go Struct Definitions

```go
// internal/fonws/uid.go

// UIDAbfrageRequest for SOAP request
type UIDAbfrageRequest struct {
    XMLName xml.Name `xml:"uid:uidAbfrage"`
    TID     string   `xml:"uid:tid"`
    BenID   string   `xml:"uid:benid"`
    ID      string   `xml:"uid:id"`
    UIDTN   string   `xml:"uid:uid_tn"`
    Stufe   int      `xml:"uid:stuession"`
}

// UIDAbfrageResponse for SOAP response
type UIDAbfrageResponse struct {
    RC         int    `xml:"rc"`
    Msg        string `xml:"msg"`
    UIDTN      string `xml:"uid_tn"`
    Gueltig    string `xml:"gueltig"`
    Name       string `xml:"name"`
    AdrStrasse string `xml:"adr_strasse"`
    AdrPLZ     string `xml:"adr_plz"`
    AdrOrt     string `xml:"adr_ort"`
}
```

```go
// internal/fonws/uva.go

// FileUploadRequest for SOAP request
type FileUploadRequest struct {
    XMLName    xml.Name `xml:"fon:upload"`
    TID        string   `xml:"fon:tid"`
    BenID      string   `xml:"fon:benid"`
    ID         string   `xml:"fon:id"`
    Art        string   `xml:"fon:art"`
    Data       string   `xml:"fon:uebession>fon:data"`
}

// FileUploadResponse for SOAP response
type FileUploadResponse struct {
    RC          int    `xml:"rc"`
    Msg         string `xml:"msg"`
    Belegnummer string `xml:"belegnummer"`
}

// UVA XML structure for U30
type UVADocument struct {
    XMLName      xml.Name       `xml:"Umsatzsteuervoranmeldung"`
    XMLNS        string         `xml:"xmlns,attr"`
    Steuernummer string         `xml:"Steuernummer"`
    Zeitraum     UVAZeitraum    `xml:"Zeitraum"`
    Kennzahlen   UVAKennzahlen  `xml:"Kennzahlen"`
}

type UVAZeitraum struct {
    Jahr   int `xml:"Jahr"`
    Monat  int `xml:"Monat,omitempty"`
    Quartal int `xml:"Quartal,omitempty"`
}

type UVAKennzahlen struct {
    KZ000 int64 `xml:"KZ000,omitempty"`
    KZ001 int64 `xml:"KZ001,omitempty"`
    KZ017 int64 `xml:"KZ017,omitempty"`
    KZ018 int64 `xml:"KZ018,omitempty"`
    KZ019 int64 `xml:"KZ019,omitempty"`
    KZ060 int64 `xml:"KZ060,omitempty"`
    KZ095 int64 `xml:"KZ095,omitempty"`
}
```

---

## 5. CLI Commands

```bash
# UID Validation
fo uid check ATU12345678
fo uid check ATU12345678 --json
fo uid batch uids.csv --output results.csv

# UVA Submission
fo uva validate uva.xml
fo uva submit <account> --file uva.xml
fo uva submit <account> --year 2025 --month 1 --kz017 80000 --kz060 16000
fo uva status <belegnummer>

# ZM Submission
fo zm submit <account> --file zm.xml
fo zm submit <account> --year 2025 --quarter 1
```
