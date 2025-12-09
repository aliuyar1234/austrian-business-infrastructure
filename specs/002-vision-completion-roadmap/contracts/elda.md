# Contracts: ELDA - Social Security Integration

**Module**: elda
**Date**: 2025-12-07

## 1. Overview

ELDA (Elektronischer Datenaustausch mit den österreichischen Sozialversicherungsträgern) uses a different SOAP endpoint and credentials than FinanzOnline. All transmissions follow the DM-Org specification.

**Endpoint**: ELDA portal submission
**Documentation**: DM-Org Version 41.1.0

---

## 2. Anmeldung (Employee Registration)

### 2.1 Request Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ELDA xmlns="http://www.elda.at/dm">
    <Kopf>
        <DienstgeberNr>12345678</DienstgeberNr>
        <BenutzerNr>WS001</BenutzerNr>
        <Meldungsart>AN</Meldungsart>
        <Erstelldatum>2025-01-15</Erstelldatum>
    </Kopf>
    <Anmeldung>
        <SVNummer>1234150185</SVNummer>
        <Vorname>Max</Vorname>
        <Nachname>Mustermann</Nachname>
        <Geburtsdatum>1985-01-15</Geburtsdatum>
        <Geschlecht>M</Geschlecht>
        <Eintrittsdatum>2025-02-01</Eintrittsdatum>
        <Beschaeftigung>
            <Art>V</Art>
            <Arbeitszeit>38.5</Arbeitszeit>
            <Kollektiv>HAN</Kollektiv>
        </Beschaeftigung>
        <Entgelt>
            <Brutto>350000</Brutto>
            <Sonderzahlung>700000</Sonderzahlung>
        </Entgelt>
    </Anmeldung>
</ELDA>
```

**Parameters**:
- `DienstgeberNr`: 8-digit employer number
- `Meldungsart`: "AN" for Anmeldung
- `SVNummer`: 10-digit social security number
- `Art`: V=Vollzeit, T=Teilzeit, G=Geringfügig
- `Brutto`: Monthly gross in cents
- `Sonderzahlung`: Annual special payments in cents

### 2.2 Response Structure

```xml
<ELDAResponse>
    <Status>OK</Status>
    <Referenznummer>ELDA-2025-AN-12345678</Referenznummer>
    <Timestamp>2025-01-15T14:30:00+01:00</Timestamp>
</ELDAResponse>
```

### 2.3 Error Response

```xml
<ELDAResponse>
    <Status>ERROR</Status>
    <Fehler>
        <Code>E1001</Code>
        <Feld>SVNummer</Feld>
        <Meldung>SVNummer ungültig - Prüfziffer stimmt nicht</Meldung>
    </Fehler>
</ELDAResponse>
```

---

## 3. Abmeldung (Employee Deregistration)

### 3.1 Request Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ELDA xmlns="http://www.elda.at/dm">
    <Kopf>
        <DienstgeberNr>12345678</DienstgeberNr>
        <BenutzerNr>WS001</BenutzerNr>
        <Meldungsart>AB</Meldungsart>
        <Erstelldatum>2025-06-30</Erstelldatum>
    </Kopf>
    <Abmeldung>
        <SVNummer>1234150185</SVNummer>
        <Austrittsdatum>2025-06-30</Austrittsdatum>
        <Grund>K</Grund>
        <Abrechnung>
            <Urlaubsersatz>85000</Urlaubsersatz>
            <Abfertigung>0</Abfertigung>
        </Abrechnung>
    </Abmeldung>
</ELDA>
```

**Grund Codes**:
- `K`: Kündigung durch Dienstgeber
- `KN`: Kündigung durch Dienstnehmer
- `E`: Einvernehmlich
- `EN`: Entlassung
- `A`: Vorzeitiger Austritt
- `B`: Befristung abgelaufen
- `P`: Pensionierung

---

## 4. Status Query

### 4.1 Request Structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<ELDA xmlns="http://www.elda.at/dm">
    <Kopf>
        <DienstgeberNr>12345678</DienstgeberNr>
        <BenutzerNr>WS001</BenutzerNr>
        <Meldungsart>ST</Meldungsart>
    </Kopf>
    <StatusAbfrage>
        <Referenznummer>ELDA-2025-AN-12345678</Referenznummer>
    </StatusAbfrage>
</ELDA>
```

### 4.2 Response Structure

```xml
<ELDAResponse>
    <Status>VERARBEITET</Status>
    <Referenznummer>ELDA-2025-AN-12345678</Referenznummer>
    <Details>
        <Eingangsdatum>2025-01-15T14:30:00+01:00</Eingangsdatum>
        <Verarbeitungsdatum>2025-01-15T15:00:00+01:00</Verarbeitungsdatum>
        <Ergebnis>ANGENOMMEN</Ergebnis>
    </Details>
</ELDAResponse>
```

**Status Values**:
- `EINGEGANGEN`: Received, not yet processed
- `IN_VERARBEITUNG`: Being processed
- `VERARBEITET`: Processed
- `FEHLER`: Error occurred

---

## 5. Error Codes

| Code | Description | Action |
|------|-------------|--------|
| E1001 | SVNummer ungültig | Verify check digit |
| E1002 | DienstgeberNr ungültig | Verify employer number |
| E1003 | Eintrittsdatum in Vergangenheit | Must be future date |
| E1004 | Pflichtfeld fehlt | Add required field |
| E1005 | Doppelte Meldung | Already registered |
| E2001 | Authentifizierung fehlgeschlagen | Check credentials |
| E2002 | Keine Berechtigung | Contact ELDA |

---

## 6. Go Struct Definitions

```go
// internal/elda/types.go

// ELDAKopf is the header for all ELDA messages
type ELDAKopf struct {
    XMLName       xml.Name `xml:"Kopf"`
    DienstgeberNr string   `xml:"DienstgeberNr"`
    BenutzerNr    string   `xml:"BenutzerNr"`
    Meldungsart   string   `xml:"Meldungsart"`
    Erstelldatum  string   `xml:"Erstelldatum"`
}

// ELDAAnmeldung for employee registration
type ELDAAnmeldung struct {
    XMLName       xml.Name            `xml:"Anmeldung"`
    SVNummer      string              `xml:"SVNummer"`
    Vorname       string              `xml:"Vorname"`
    Nachname      string              `xml:"Nachname"`
    Geburtsdatum  string              `xml:"Geburtsdatum"`
    Geschlecht    string              `xml:"Geschlecht"`
    Eintrittsdatum string             `xml:"Eintrittsdatum"`
    Beschaeftigung ELDABeschaeftigung `xml:"Beschaeftigung"`
    Entgelt       ELDAEntgelt         `xml:"Entgelt"`
}

type ELDABeschaeftigung struct {
    Art        string  `xml:"Art"`
    Arbeitszeit float64 `xml:"Arbeitszeit"`
    Kollektiv  string  `xml:"Kollektiv"`
}

type ELDAEntgelt struct {
    Brutto        int64 `xml:"Brutto"`
    Sonderzahlung int64 `xml:"Sonderzahlung"`
}

// ELDAAbmeldung for employee deregistration
type ELDAAbmeldung struct {
    XMLName       xml.Name        `xml:"Abmeldung"`
    SVNummer      string          `xml:"SVNummer"`
    Austrittsdatum string         `xml:"Austrittsdatum"`
    Grund         string          `xml:"Grund"`
    Abrechnung    ELDAAbrechnung  `xml:"Abrechnung"`
}

type ELDAAbrechnung struct {
    Urlaubsersatz int64 `xml:"Urlaubsersatz"`
    Abfertigung   int64 `xml:"Abfertigung"`
}

// ELDAResponse for all responses
type ELDAResponse struct {
    XMLName        xml.Name     `xml:"ELDAResponse"`
    Status         string       `xml:"Status"`
    Referenznummer string       `xml:"Referenznummer,omitempty"`
    Timestamp      string       `xml:"Timestamp,omitempty"`
    Fehler         *ELDAFehler  `xml:"Fehler,omitempty"`
    Details        *ELDADetails `xml:"Details,omitempty"`
}

type ELDAFehler struct {
    Code    string `xml:"Code"`
    Feld    string `xml:"Feld,omitempty"`
    Meldung string `xml:"Meldung"`
}

type ELDADetails struct {
    Eingangsdatum     string `xml:"Eingangsdatum"`
    Verarbeitungsdatum string `xml:"Verarbeitungsdatum,omitempty"`
    Ergebnis          string `xml:"Ergebnis,omitempty"`
}
```

---

## 7. SV-Nummer Validation

The Austrian social security number (SVNummer) has a check digit:

```go
// ValidateSVNummer validates an Austrian social security number
func ValidateSVNummer(svnr string) bool {
    if len(svnr) != 10 {
        return false
    }

    // Weights for check digit calculation
    weights := []int{3, 7, 9, 0, 5, 8, 4, 2, 1, 6}

    sum := 0
    for i, c := range svnr {
        digit := int(c - '0')
        if digit < 0 || digit > 9 {
            return false
        }
        sum += digit * weights[i]
    }

    return sum % 11 == 0
}
```

---

## 8. CLI Commands

```bash
# ELDA Account Management
fo account add elda-mycompany --type elda
# Prompts: DienstgeberNr, BenutzerNr, PIN

# Anmeldung
elda anmeldung --account mycompany \
    --svnummer 1234150185 \
    --vorname Max \
    --nachname Mustermann \
    --geburtsdatum 1985-01-15 \
    --eintrittsdatum 2025-02-01 \
    --brutto 3500 \
    --art vollzeit

elda anmeldung --account mycompany --file anmeldung.json

# Abmeldung
elda abmeldung --account mycompany \
    --svnummer 1234150185 \
    --austrittsdatum 2025-06-30 \
    --grund kuendigung

# Status
elda status ELDA-2025-AN-12345678 --account mycompany
```

---

## 9. Credential Storage

ELDA credentials stored separately in the credential store:

```json
{
    "accounts": {
        "fo-personal": {
            "type": "finanzonline",
            "tid": "123456789",
            "benid": "USER1",
            "pin": "encrypted..."
        },
        "elda-mycompany": {
            "type": "elda",
            "dienstgeber_nr": "12345678",
            "benutzer_nr": "WS001",
            "pin": "encrypted..."
        }
    }
}
```
