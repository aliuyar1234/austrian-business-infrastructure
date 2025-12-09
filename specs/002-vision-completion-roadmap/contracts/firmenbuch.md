# Contracts: Firmenbuch - Company Register API

**Module**: fb
**Date**: 2025-12-07

## 1. Overview

The Firmenbuch (Austrian Company Register) provides a SOAP API through JustizOnline.

**WSDL**: `https://justizonline.gv.at/jop/api/at.gv.justiz.fbw/ws/fbw.wsdl`
**Protocol**: SOAP 1.2 (default), SOAP 1.1 supported
**Authentication**: API Key header

---

## 2. Authentication

All requests require:

```http
Content-Type: application/soap+xml; charset=utf-8
X-API-KEY: your_api_key_here
```

For SOAP 1.1:
```http
Content-Type: text/xml; charset=utf-8
SOAPAction: "urn:search"
X-API-KEY: your_api_key_here
```

---

## 3. Company Search (SUCHEFIRMAREQUEST)

### 3.1 Request Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
               xmlns:fbw="http://justiz.gv.at/fbw">
    <soap:Body>
        <fbw:SUCHEFIRMAREQUEST>
            <fbw:FIRMENNAME>Musterfirma</fbw:FIRMENNAME>
            <fbw:SUCHEART>PHONETISCH</fbw:SUCHEART>
            <fbw:GERICHT></fbw:GERICHT>
            <fbw:RECHTSFORM></fbw:RECHTSFORM>
            <fbw:ORT></fbw:ORT>
            <fbw:MAXRESULTS>50</fbw:MAXRESULTS>
        </fbw:SUCHEFIRMAREQUEST>
    </soap:Body>
</soap:Envelope>
```

**Parameters**:
- `FIRMENNAME`: Company name (fuzzy or exact)
- `SUCHEART`: "EXAKT" or "PHONETISCH" (fuzzy)
- `GERICHT`: Court code filter (optional)
- `RECHTSFORM`: Legal form filter (optional)
- `ORT`: Location filter (optional)
- `MAXRESULTS`: Max results (default 50)

### 3.2 Response Structure

```xml
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope">
    <soap:Body>
        <SUCHEFIRMARESPONSE>
            <ANZAHL>3</ANZAHL>
            <FIRMEN>
                <FIRMA>
                    <FN>123456a</FN>
                    <FIRMENNAME>Musterfirma GmbH</FIRMENNAME>
                    <RECHTSFORM>GmbH</RECHTSFORM>
                    <SITZ>Wien</SITZ>
                    <STATUS>AKTIV</STATUS>
                </FIRMA>
                <FIRMA>
                    <FN>234567b</FN>
                    <FIRMENNAME>Musterfirma KG</FIRMENNAME>
                    <RECHTSFORM>KG</RECHTSFORM>
                    <SITZ>Graz</SITZ>
                    <STATUS>AKTIV</STATUS>
                </FIRMA>
            </FIRMEN>
        </SUCHEFIRMARESPONSE>
    </soap:Body>
</soap:Envelope>
```

---

## 4. Company Extract (AUSZUGREQUEST)

### 4.1 Request Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"
               xmlns:fbw="http://justiz.gv.at/fbw">
    <soap:Body>
        <fbw:AUSZUGREQUEST>
            <fbw:FN>123456a</fbw:FN>
            <fbw:VARIANTE>AKTUELL</fbw:VARIANTE>
            <fbw:SPRACHE>DE</fbw:SPRACHE>
        </fbw:AUSZUGREQUEST>
    </soap:Body>
</soap:Envelope>
```

**Parameters**:
- `FN`: Firmenbuchnummer (e.g., "123456a")
- `VARIANTE`: "AKTUELL" (current) or "HISTORISCH" (with history)
- `SPRACHE`: "DE" (German)

### 4.2 Response Structure

```xml
<AUSZUGRESPONSE>
    <FIRMA>
        <FN>123456a</FN>
        <FIRMENNAME>Musterfirma GmbH</FIRMENNAME>
        <RECHTSFORM>GmbH</RECHTSFORM>
        <SITZ>Wien</SITZ>
        <ADRESSE>
            <STRASSE>Musterstraße 1</STRASSE>
            <PLZ>1010</PLZ>
            <ORT>Wien</ORT>
            <LAND>AT</LAND>
        </ADRESSE>
        <KAPITAL>
            <STAMMKAPITAL>35000</STAMMKAPITAL>
            <WAEHRUNG>EUR</WAEHRUNG>
        </KAPITAL>
        <GRUENDUNG>2010-05-15</GRUENDUNG>
        <EINTRAGUNG>2010-06-01</EINTRAGUNG>
    </FIRMA>
    <FUN>
        <PERSON>
            <VORNAME>Max</VORNAME>
            <NACHNAME>Mustermann</NACHNAME>
            <GEBURTSDATUM>1970-03-20</GEBURTSDATUM>
            <FUNKTION>GESCHAEFTSFUEHRER</FUNKTION>
            <VERTRETUNG>SELBSTAENDIG</VERTRETUNG>
            <SEIT>2010-06-01</SEIT>
        </PERSON>
    </FUN>
    <PER>
        <GESELLSCHAFTER>
            <NAME>Max Mustermann</NAME>
            <ANTEIL>70%</ANTEIL>
            <STAMMEINLAGE>24500</STAMMEINLAGE>
        </GESELLSCHAFTER>
        <GESELLSCHAFTER>
            <NAME>Holding GmbH</NAME>
            <FN>654321b</FN>
            <ANTEIL>30%</ANTEIL>
            <STAMMEINLAGE>10500</STAMMEINLAGE>
        </GESELLSCHAFTER>
    </PER>
    <IDENT>
        <UID>ATU12345678</UID>
        <GLN>1234567890123</GLN>
    </IDENT>
</AUSZUGRESPONSE>
```

---

## 5. Document Request (URKUNDEREQUEST)

### 5.1 Request Structure

```xml
<fbw:URKUNDEREQUEST>
    <fbw:FN>123456a</fbw:FN>
    <fbw:DOKUMENT_ID>URK-2024-001234</fbw:DOKUMENT_ID>
</fbw:URKUNDEREQUEST>
```

### 5.2 Response Structure

```xml
<URKUNDERESPONSE>
    <FN>123456a</FN>
    <DOKUMENT_ID>URK-2024-001234</DOKUMENT_ID>
    <DATEINAME>Gesellschaftsvertrag.pdf</DATEINAME>
    <INHALT>BASE64_ENCODED_PDF</INHALT>
    <GROESSE>245678</GROESSE>
</URKUNDERESPONSE>
```

---

## 6. Change History (VERAENDERUNGENFIRMAREQUEST)

### 6.1 Request Structure

```xml
<fbw:VERAENDERUNGENFIRMAREQUEST>
    <fbw:FN>123456a</fbw:FN>
    <fbw:VON>2024-01-01</fbw:VON>
    <fbw:BIS>2025-01-01</fbw:BIS>
</fbw:VERAENDERUNGENFIRMAREQUEST>
```

### 6.2 Response Structure

```xml
<VERAENDERUNGENFIRMARESPONSE>
    <FN>123456a</FN>
    <AENDERUNGEN>
        <AENDERUNG>
            <DATUM>2024-06-15</DATUM>
            <ART>GESCHAEFTSFUEHRER_WECHSEL</ART>
            <BESCHREIBUNG>Neuer Geschäftsführer bestellt: Maria Musterfrau</BESCHREIBUNG>
        </AENDERUNG>
        <AENDERUNG>
            <DATUM>2024-03-01</DATUM>
            <ART>ADRESSE</ART>
            <BESCHREIBUNG>Sitzverlegung nach Wien</BESCHREIBUNG>
        </AENDERUNG>
    </AENDERUNGEN>
</VERAENDERUNGENFIRMARESPONSE>
```

---

## 7. Error Responses

### 7.1 SOAP Fault

```xml
<soap:Fault>
    <soap:Code>
        <soap:Value>soap:Sender</soap:Value>
    </soap:Code>
    <soap:Reason>
        <soap:Text xml:lang="de">Firmenbuchnummer nicht gefunden</soap:Text>
    </soap:Reason>
    <soap:Detail>
        <fbw:ErrorCode>FB-404</fbw:ErrorCode>
    </soap:Detail>
</soap:Fault>
```

### 7.2 Error Codes

| Code | Description |
|------|-------------|
| FB-400 | Invalid request format |
| FB-401 | Authentication failed |
| FB-403 | Access denied |
| FB-404 | Company not found |
| FB-429 | Rate limit exceeded |
| FB-500 | Internal server error |

---

## 8. Go Struct Definitions

```go
// internal/fb/types.go

// FBSearchRequest for company search
type FBSearchRequest struct {
    XMLName    xml.Name `xml:"fbw:SUCHEFIRMAREQUEST"`
    Firmenname string   `xml:"fbw:FIRMENNAME"`
    Sucheart   string   `xml:"fbw:SUCHEART"`
    Gericht    string   `xml:"fbw:GERICHT,omitempty"`
    Rechtsform string   `xml:"fbw:RECHTSFORM,omitempty"`
    Ort        string   `xml:"fbw:ORT,omitempty"`
    MaxResults int      `xml:"fbw:MAXRESULTS,omitempty"`
}

// FBSearchResponse for search results
type FBSearchResponse struct {
    Anzahl int         `xml:"ANZAHL"`
    Firmen []FBFirmaKurz `xml:"FIRMEN>FIRMA"`
}

type FBFirmaKurz struct {
    FN         string `xml:"FN"`
    Firmenname string `xml:"FIRMENNAME"`
    Rechtsform string `xml:"RECHTSFORM"`
    Sitz       string `xml:"SITZ"`
    Status     string `xml:"STATUS"`
}

// FBExtractRequest for detailed extract
type FBExtractRequest struct {
    XMLName  xml.Name `xml:"fbw:AUSZUGREQUEST"`
    FN       string   `xml:"fbw:FN"`
    Variante string   `xml:"fbw:VARIANTE"`
    Sprache  string   `xml:"fbw:SPRACHE"`
}

// FBExtractResponse for full company data
type FBExtractResponse struct {
    Firma   FBFirma        `xml:"FIRMA"`
    Fun     []FBFunktion   `xml:"FUN>PERSON"`
    Per     []FBGesellsch  `xml:"PER>GESELLSCHAFTER"`
    Ident   FBIdent        `xml:"IDENT"`
}

type FBFirma struct {
    FN           string     `xml:"FN"`
    Firmenname   string     `xml:"FIRMENNAME"`
    Rechtsform   string     `xml:"RECHTSFORM"`
    Sitz         string     `xml:"SITZ"`
    Adresse      FBAdresse  `xml:"ADRESSE"`
    Kapital      FBKapital  `xml:"KAPITAL"`
    Gruendung    string     `xml:"GRUENDUNG"`
    Eintragung   string     `xml:"EINTRAGUNG"`
}

type FBAdresse struct {
    Strasse string `xml:"STRASSE"`
    PLZ     string `xml:"PLZ"`
    Ort     string `xml:"ORT"`
    Land    string `xml:"LAND"`
}

type FBKapital struct {
    Stammkapital int64  `xml:"STAMMKAPITAL"`
    Grundkapital int64  `xml:"GRUNDKAPITAL"`
    Waehrung     string `xml:"WAEHRUNG"`
}

type FBFunktion struct {
    Vorname      string `xml:"VORNAME"`
    Nachname     string `xml:"NACHNAME"`
    Geburtsdatum string `xml:"GEBURTSDATUM"`
    Funktion     string `xml:"FUNKTION"`
    Vertretung   string `xml:"VERTRETUNG"`
    Seit         string `xml:"SEIT"`
    Bis          string `xml:"BIS,omitempty"`
}

type FBGesellsch struct {
    Name         string `xml:"NAME"`
    FN           string `xml:"FN,omitempty"`
    Anteil       string `xml:"ANTEIL"`
    Stammeinlage int64  `xml:"STAMMEINLAGE"`
}

type FBIdent struct {
    UID string `xml:"UID"`
    GLN string `xml:"GLN"`
}
```

---

## 9. CLI Commands

```bash
# Search
fb search "Musterfirma"
fb search "Musterfirma" --rechtsform GmbH --ort Wien
fb search "Musterfirma" --exact

# Extract
fb extract FN123456a
fb extract FN123456a --json
fb extract FN123456a --output musterfirma.json

# Watchlist
fb watch add FN123456a
fb watch add FN234567b
fb watch list
fb watch check
fb watch remove FN123456a

# Document
fb document FN123456a URK-2024-001234 --output vertrag.pdf
```

---

## 10. Credential Storage

Firmenbuch API credentials:

```json
{
    "accounts": {
        "fb-default": {
            "type": "firmenbuch",
            "api_key": "encrypted_api_key..."
        }
    }
}
```

---

## 11. Caching Strategy

To reduce API costs:

1. **Search results**: Cache for 24 hours
2. **Extract data**: Cache for 7 days (unless watchlist update)
3. **Documents**: Cache indefinitely (immutable)

```go
type FBCache struct {
    SearchResults map[string]CachedSearch    // key: query hash
    Extracts      map[string]CachedExtract   // key: FN
    Documents     map[string]string          // key: doc_id, value: file path
}

type CachedSearch struct {
    Results   FBSearchResponse
    CachedAt  time.Time
}

type CachedExtract struct {
    Extract   FBExtractResponse
    CachedAt  time.Time
}
```
