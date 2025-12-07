# Databox Service Contract

**Endpoint**: `https://finanzonline.bmf.gv.at/fonws/ws/databoxService`
**Protocol**: SOAP 1.1 over HTTPS
**Content-Type**: `text/xml; charset=utf-8`

## Operations

### GetDataboxInfo

Retrieves list of documents in the FinanzOnline Databox.

#### Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfo xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <id>{session_token}</id>
      <tid>{teilnehmer_id}</tid>
      <benid>{benutzer_id}</benid>
      <ts_zust_von>{from_date}</ts_zust_von>
      <ts_zust_bis>{to_date}</ts_zust_bis>
    </GetDataboxInfo>
  </soap:Body>
</soap:Envelope>
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | Yes | Session token from Login |
| tid | string(12) | Yes | Teilnehmer-ID |
| benid | string | Yes | Benutzer-ID |
| ts_zust_von | string | No | Filter: documents from date (YYYY-MM-DD) |
| ts_zust_bis | string | No | Filter: documents until date (YYYY-MM-DD) |

#### Response (Success)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxInfoResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <databox>
          <applkey>APP123456</applkey>
          <filebez>Bescheid_2025</filebez>
          <ts_zust>2025-12-01T10:30:00</ts_zust>
          <erlession>B</erlession>
          <veression>N</veression>
        </databox>
        <databox>
          <applkey>APP789012</applkey>
          <filebez>Ergaenzungsersuchen_2025</filebez>
          <ts_zust>2025-12-05T14:15:00</ts_zust>
          <erlession>E</erlession>
          <veression>N</veression>
        </databox>
      </result>
    </GetDataboxInfoResponse>
  </soap:Body>
</soap:Envelope>
```

| Field | Type | Description |
|-------|------|-------------|
| rc | int | Response code (0 = success) |
| msg | string | Error message (empty on success) |
| result | element | Container for databox entries |
| databox | element[] | List of document entries |

#### Databox Entry Fields

| Field | Type | Description |
|-------|------|-------------|
| applkey | string | Unique document identifier (for download) |
| filebez | string | File description/name |
| ts_zust | datetime | Document delivery timestamp |
| erlession | string | Document type code (B, E, M, V) |
| veression | string | Processing status (N=new, etc.) |

#### Document Type Codes

| Code | Type | Description | Action Required |
|------|------|-------------|-----------------|
| B | Bescheid | Tax assessment notice | No |
| E | Ergänzungsersuchen | Request for additional information | **Yes** |
| M | Mitteilung | General notification | No |
| V | Vorhalt | Preliminary inquiry/request | **Yes** |

---

### GetDatabox

Downloads a specific document from the Databox.

#### Request

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDatabox xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <id>{session_token}</id>
      <tid>{teilnehmer_id}</tid>
      <benid>{benutzer_id}</benid>
      <applkey>{document_applkey}</applkey>
    </GetDatabox>
  </soap:Body>
</soap:Envelope>
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | Yes | Session token from Login |
| tid | string(12) | Yes | Teilnehmer-ID |
| benid | string | Yes | Benutzer-ID |
| applkey | string | Yes | Document identifier from GetDataboxInfo |

#### Response (Success)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
  <soap:Body>
    <GetDataboxResponse xmlns="https://finanzonline.bmf.gv.at/fonws/ws/databoxService">
      <rc>0</rc>
      <msg></msg>
      <result>
        <filename>Bescheid_ESt_2024.pdf</filename>
        <content>{base64_encoded_content}</content>
      </result>
    </GetDataboxResponse>
  </soap:Body>
</soap:Envelope>
```

| Field | Type | Description |
|-------|------|-------------|
| rc | int | Response code (0 = success) |
| msg | string | Error message (empty on success) |
| filename | string | Suggested filename for the document |
| content | string | Base64-encoded document content |

---

## Error Codes

Same as Session Service:

| Code | Description |
|------|-------------|
| 0 | Success |
| -1 | Session expired |
| -2 | Maintenance |
| -3 | Technical error |

---

## Go Structs

```go
// GetDataboxInfoRequest represents a SOAP GetDataboxInfo request
type GetDataboxInfoRequest struct {
    XMLName    xml.Name `xml:"GetDataboxInfo"`
    Xmlns      string   `xml:"xmlns,attr"`
    ID         string   `xml:"id"`
    TID        string   `xml:"tid"`
    BenID      string   `xml:"benid"`
    TsZustVon  string   `xml:"ts_zust_von,omitempty"`
    TsZustBis  string   `xml:"ts_zust_bis,omitempty"`
}

// GetDataboxInfoResponse represents a SOAP GetDataboxInfo response
type GetDataboxInfoResponse struct {
    XMLName xml.Name       `xml:"GetDataboxInfoResponse"`
    RC      int            `xml:"rc"`
    Msg     string         `xml:"msg"`
    Result  DataboxResult  `xml:"result"`
}

// DataboxResult contains the list of databox entries
type DataboxResult struct {
    Entries []DataboxEntry `xml:"databox"`
}

// DataboxEntry represents a single document in the databox
type DataboxEntry struct {
    Applkey    string `xml:"applkey"`
    Filebez    string `xml:"filebez"`
    TsZust     string `xml:"ts_zust"`
    Erlession  string `xml:"erlession"`
    Veression  string `xml:"verarbeitung"`
}

// GetDataboxRequest represents a SOAP GetDatabox request (download)
type GetDataboxRequest struct {
    XMLName xml.Name `xml:"GetDatabox"`
    Xmlns   string   `xml:"xmlns,attr"`
    ID      string   `xml:"id"`
    TID     string   `xml:"tid"`
    BenID   string   `xml:"benid"`
    Applkey string   `xml:"applkey"`
}

// GetDataboxResponse represents a SOAP GetDatabox response (download)
type GetDataboxResponse struct {
    XMLName  xml.Name        `xml:"GetDataboxResponse"`
    RC       int             `xml:"rc"`
    Msg      string          `xml:"msg"`
    Result   DataboxDownload `xml:"result"`
}

// DataboxDownload contains the downloaded document
type DataboxDownload struct {
    Filename string `xml:"filename"`
    Content  string `xml:"content"` // Base64 encoded
}

// Helper method to check if action is required
func (e DataboxEntry) ActionRequired() bool {
    return e.Erlession == "E" || e.Erlession == "V"
}

// Helper method to get human-readable document type
func (e DataboxEntry) TypeName() string {
    switch e.Erlession {
    case "B":
        return "Bescheid"
    case "E":
        return "Ergänzungsersuchen"
    case "M":
        return "Mitteilung"
    case "V":
        return "Vorhalt"
    default:
        return e.Erlession
    }
}
```
