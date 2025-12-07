# Austrian Business Infrastructure

## The Open-Source Backend for Austrian Business Operations

---

## Vision

Jede österreichische Firma kämpft mit denselben Behörden-Schnittstellen. FinanzOnline, ELDA, Firmenbuch, USP - alle haben APIs, alle sind dokumentiert, niemand baut Open-Source Tools dafür.

**Wir bauen die fehlende Infrastruktur-Schicht.**

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│                    AUSTRIAN BUSINESS INFRASTRUCTURE                         │
│                                                                             │
│         "Stripe für österreichische Behörden-Schnittstellen"               │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │   DEINE APP / ERP / BUCHHALTUNG                                    │   │
│  │                                                                     │   │
│  └───────────────────────────┬─────────────────────────────────────────┘   │
│                              │                                              │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │                    AUSTRIAN BUSINESS SDK                           │   │
│  │                                                                     │   │
│  │   Go Libraries + CLI Tools + MCP Servers + REST Wrapper           │   │
│  │                                                                     │   │
│  └───────────────────────────┬─────────────────────────────────────────┘   │
│                              │                                              │
│          ┌───────────────────┼───────────────────┐                         │
│          │                   │                   │                         │
│          ▼                   ▼                   ▼                         │
│  ┌───────────────┐   ┌───────────────┐   ┌───────────────┐                │
│  │               │   │               │   │               │                │
│  │  TAX & FINANCE│   │    SOCIAL     │   │   COMPANY     │                │
│  │               │   │   INSURANCE   │   │   REGISTRY    │                │
│  │  FinanzOnline │   │     ELDA      │   │  Firmenbuch   │                │
│  │  USP Portal   │   │     ÖGK       │   │    GISA       │                │
│  │  E-Rechnung   │   │     SVS       │   │    WKO        │                │
│  │               │   │               │   │               │                │
│  └───────────────┘   └───────────────┘   └───────────────┘                │
│          │                   │                   │                         │
│          └───────────────────┴───────────────────┘                         │
│                              │                                              │
│                              ▼                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                                                                     │   │
│  │                    ÖSTERREICHISCHE BEHÖRDEN                        │   │
│  │                                                                     │   │
│  │      BMF          Sozialversicherung        Justizministerium      │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Warum das niemand gebaut hat

| Grund | Realität |
|-------|----------|
| SOAP ist eklig | Stimmt. Aber einmal abstrahiert, nie wieder anfassen. |
| Nur Österreich | 600.000+ Unternehmen. €50/Monat = €360M TAM. |
| BMD/RZL existieren | Proprietär, teuer, kein API-First. |
| Kein VC-Case | Perfekt. Bootstrap-fähig, profitabel ab Tag 1. |

**Der echte Grund:** Niemand will SOAP anfassen. Das ist unser Moat.

---

## Die Module

### 1. Tax & Finance (FinanzOnline Suite)

```
fo - FinanzOnline CLI

├── fo session          # Login/Logout, Session Management
├── fo databox          # Bescheide, Ergänzungsersuchen abrufen
├── fo uva              # Umsatzsteuervoranmeldung
├── fo uid              # UID-Validierung
├── fo zm               # Zusammenfassende Meldung
├── fo est              # Einkommensteuererklärung
├── fo kest             # Körperschaftsteuererklärung
└── fo account          # Multi-Account Management
```

**WebService Endpoints:**
- Session: `sessionService.wsdl`
- Databox: `databoxService.wsdl`  
- Upload: `fileUploadService.wsdl`
- UID: `uidService.wsdl`

**Use Cases:**
- Steuerberater mit 50+ Mandanten
- Holdings mit 30 Tochtergesellschaften
- SaaS-Anbieter die FinanzOnline integrieren wollen

---

### 2. Social Insurance (ELDA Suite)

```
elda - Elektronischer Datenaustausch mit Sozialversicherung

├── elda anmeldung      # Dienstnehmer anmelden
├── elda abmeldung      # Dienstnehmer abmelden
├── elda aenderung      # Änderungsmeldungen
├── elda lohnzettel     # Jahreslohnzettel
├── elda beitraege      # Beitragsnachweisung
└── elda status         # Meldungsstatus abfragen
```

**Pain Point:** HR-Abteilungen machen das manuell oder zahlen €€€ für BMD/DATEV.

---

### 3. Company Registry (Firmenbuch Suite)

```
fb - Firmenbuch CLI

├── fb search           # Firmen suchen
├── fb extract          # Firmenbuchauszug
├── fb monitor          # Änderungen überwachen
├── fb insolvenz        # Insolvenzdatei abfragen
└── fb gisa             # Gewerbeinformationssystem
```

**Use Cases:**
- Due Diligence Automation
- KYC/AML Compliance
- Credit Risk Monitoring
- M&A Research

---

### 4. E-Rechnung (ZUGFeRD/XRechnung)

```
erechnung - Austrian E-Invoice Toolkit

├── erechnung validate  # Validierung gegen EN16931
├── erechnung create    # Rechnung erstellen
├── erechnung extract   # Daten aus PDF/XML extrahieren
├── erechnung convert   # Format-Konvertierung
└── erechnung send      # An USP/Peppol senden
```

**Timing:** B2B-Pflicht kommt (DE 2025, AT folgt).

---

### 5. Banking & Payments

```
sepa - SEPA Toolkit

├── sepa pain001        # Überweisungen generieren
├── sepa pain008        # Lastschriften generieren
├── sepa camt053        # Kontoauszüge parsen
├── sepa camt054        # Einzelumsätze parsen
└── sepa validate       # IBAN/BIC Validierung
```

**Integration:** Jedes ERP braucht SEPA. Keiner will die XML-Spec lesen.

---

## Architektur

```
austrian-business-sdk/
│
├── cmd/
│   ├── fo/                 # FinanzOnline CLI
│   ├── elda/               # ELDA CLI
│   ├── fb/                 # Firmenbuch CLI
│   ├── erechnung/          # E-Rechnung CLI
│   └── sepa/               # SEPA CLI
│
├── pkg/
│   ├── finanzonline/       # FinanzOnline Go Library
│   │   ├── session.go
│   │   ├── databox.go
│   │   ├── uva.go
│   │   └── uid.go
│   │
│   ├── elda/               # ELDA Go Library
│   ├── firmenbuch/         # Firmenbuch Go Library
│   ├── erechnung/          # E-Rechnung Go Library
│   └── sepa/               # SEPA Go Library
│
├── mcp/                    # MCP Servers für AI Integration
│   ├── finanzonline-mcp/
│   ├── elda-mcp/
│   └── firmenbuch-mcp/
│
├── api/                    # REST Wrapper (optional)
│   └── gateway/
│
└── internal/
    ├── soap/               # Shared SOAP utilities
    ├── crypto/             # Credential encryption
    └── config/             # XDG/AppData handling
```

---

## Go-To-Market: Trojanisches Pferd Strategie

```
Phase 1: FinanzOnline CLI (Monat 1-3)
         ├── Open Source, MIT License
         ├── Löst akutes Problem (2FA + Multi-Account)
         ├── Baut Community + Credibility
         └── Erste GitHub Stars

Phase 2: Firmenbuch + UID (Monat 4-6)
         ├── Due Diligence Use Case
         ├── Erste Firmenkunden
         └── Consulting-Revenue nebenbei

Phase 3: ELDA + E-Rechnung (Monat 7-12)
         ├── HR-Abteilungen als Kunden
         ├── SaaS-Integration Partnerships
         └── €€€ Support-Verträge

Phase 4: Austrian Business Cloud (Jahr 2)
         ├── Hosted Version der Tools
         ├── Team Collaboration Features
         ├── Compliance Dashboard
         └── €50-500/Monat pro Firma
```

---

## Revenue Model

### Phase 1-2: Open Source + Consulting

| Revenue Stream | Preis |
|----------------|-------|
| GitHub Sponsors | €5-50/Monat |
| Implementation Support | €150/Stunde |
| Custom Connectors | €5-15k einmalig |
| Training/Workshops | €2-5k/Tag |

### Phase 3+: SaaS + Enterprise

| Tier | Features | Preis |
|------|----------|-------|
| **CLI** | Open Source, self-hosted | Gratis |
| **Team** | Hosted, Multi-User, Audit Log | €99/Monat |
| **Business** | + API, Webhooks, Integrations | €299/Monat |
| **Enterprise** | + SLA, On-Prem, Custom | €999+/Monat |

---

## Warum jetzt?

1. **2FA-Pflicht seit 1.10.2025** - Der Schmerz ist akut
2. **E-Rechnung B2B kommt** - Timing ist perfekt
3. **AI braucht strukturierte Daten** - MCP Server als Differentiator
4. **Remote Work normalisiert** - Steuerberater wollen Automation
5. **Null Open-Source Konkurrenz** - First Mover Advantage

---

## Erster Schritt: FinanzOnline CLI

### Das Problem

Buchhalterin mit 30 Accounts. 2.5 Stunden täglich nur für Logins.

### Die Lösung

```bash
$ fo databox --all

Checking 30 accounts... done in 12s

┌─────────────────────┬────────────┬─────────────────────────┐
│ Account             │ Neue Items │ Aktion erforderlich     │
├─────────────────────┼────────────┼─────────────────────────┤
│ Holding GmbH        │ 0          │                         │
│ Tochter GmbH 1      │ 2          │                         │
│ Tochter GmbH 3      │ 1          │ ⚠️  Ergänzungsersuchen   │
│ Tochter GmbH 17     │ 1          │                         │
└─────────────────────┴────────────┴─────────────────────────┘

Zeit: 12 Sekunden statt 2.5 Stunden.
```

### Technische Details

**WebService API (kein 2FA für WebService-User!):**

```
Session:  https://finanzonline.bmf.gv.at/fonws/ws/sessionService.wsdl
Databox:  https://finanzonline.bmf.gv.at/fonws/ws/databoxService.wsdl
Upload:   https://finanzonline.bmf.gv.at/fonws/ws/fileUploadService.wsdl
```

**Login:**
```xml
<login>
  <tid>123456789012</tid>           <!-- Teilnehmer-ID -->
  <benid>WSUSER001</benid>          <!-- WebService-Benutzer-ID -->
  <pin>geheim123</pin>              <!-- WebService-PIN -->
</login>
```

**Response Codes:**
| Code | Bedeutung |
|------|-----------|
| 0 | OK |
| -1 | Session expired |
| -2 | Wartungsarbeiten |
| -3 | Technischer Fehler |
| -4 | Ungültige Credentials |
| -5/-6 | User gesperrt |
| -7 | Kein WebService-User |
| -8 | Teilnehmer gesperrt |

### Architektur

```
finanzonline-cli/
├── cmd/fo/
│   ├── main.go
│   ├── account.go       # Multi-Account Management
│   ├── databox.go       # Databox abfragen/download
│   ├── session.go       # Login/Logout
│   └── dashboard.go     # Übersicht
├── pkg/fonws/
│   ├── client.go        # SOAP Client
│   ├── session.go       # Session WebService
│   └── databox.go       # Databox WebService
├── internal/store/
│   └── accounts.go      # Encrypted Credential Store (AES-256-GCM)
└── go.mod
```

### Roadmap

| Phase | Feature | Wochen |
|-------|---------|--------|
| 1 | Session WebService | 1-2 |
| 2 | Multi-Account Store | 1-2 |
| 3 | Databox Download | 2 |
| 4 | Dashboard | 1 |
| 5 | UVA Upload | 2-3 |
| 6 | UID-Validierung | 1 |
| 7 | MCP Server | 2 |

---

## Konkurrenz

| Wer | Was | Problem |
|-----|-----|---------|
| BMD, RZL | Full Suite | €€€, Vendor Lock-in |
| FreeFinance | SaaS | Nur für deren Produkt |
| GitHub Scripts | OSS | Veraltet, unmaintained |
| **Wir** | Open Source SDK | **Existiert nicht** |

---

## Die Bet

> "In 5 Jahren läuft jede österreichische Business-Automation über unsere Libraries - oder Forks davon."

Open Source gewinnt Infrastruktur-Layer. Immer.

Wir bauen das Terraform für österreichische Behörden.
