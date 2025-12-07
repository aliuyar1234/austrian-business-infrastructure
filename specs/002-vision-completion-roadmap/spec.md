# Feature Specification: Austrian Business Infrastructure - Complete Product Suite

**Feature Branch**: `002-vision-completion-roadmap`
**Created**: 2025-12-07
**Status**: Draft
**Input**: Complete Austrian Business Infrastructure vision - full product for market launch (not just MVP)

## Overview

Diese Spezifikation definiert die verbleibenden Module, um die vollständige Austrian Business Infrastructure Vision zu realisieren. Das Ziel ist ein marktreifes Produkt-Suite, das österreichischen Unternehmen eine umfassende Open-Source-Lösung für alle Behördenschnittstellen bietet.

### Bereits implementiert (Spec 001)
- ✅ FinanzOnline CLI Basis (Session, Databox, Account Management)

### In dieser Spezifikation
- FinanzOnline Erweiterungen (UVA, UID, ZM)
- ELDA Suite (Sozialversicherung)
- Firmenbuch Suite (Company Registry)
- E-Rechnung Suite (ZUGFeRD/XRechnung)
- SEPA Toolkit (Banking)
- MCP Server Integration (AI-Ready)

---

## User Scenarios & Testing

### User Story 1 - UVA-Einreichung (Priority: P1)

Ein Steuerberater muss für seine Mandanten monatlich/quartalsweise Umsatzsteuervoranmeldungen (UVA) erstellen und an FinanzOnline übermitteln. Derzeit erfolgt dies manuell über das Web-Portal für jeden Mandanten einzeln.

**Why this priority**: UVA ist die häufigste wiederkehrende Pflicht (monatlich/quartalsweise für alle Unternehmen mit >35.000€ Umsatz). Höchste Zeitersparnis und direkter Mehrwert.

**Independent Test**: Steuerberater kann eine UVA-XML-Datei für einen Mandanten erstellen, validieren und an FinanzOnline übermitteln. Erhält Bestätigung der erfolgreichen Einreichung.

**Acceptance Scenarios**:

1. **Given** ein authentifizierter Account mit UVA-Daten, **When** der Nutzer `fo uva submit` ausführt, **Then** wird die UVA an FinanzOnline übermittelt und eine Bestätigungsnummer angezeigt
2. **Given** eine UVA-XML-Datei, **When** der Nutzer `fo uva validate` ausführt, **Then** werden Validierungsfehler oder Erfolg angezeigt
3. **Given** mehrere Mandanten-Accounts, **When** der Nutzer `fo uva submit --all` ausführt, **Then** werden alle fälligen UVAs parallel eingereicht

---

### User Story 2 - UID-Validierung (Priority: P1)

Ein Unternehmen muss vor Geschäftsabschlüssen mit EU-Partnern deren UID-Nummer validieren (gesetzliche Pflicht für innergemeinschaftliche Lieferungen). Dies ist aktuell ein manueller Prozess über das VIES-Portal.

**Why this priority**: Compliance-kritisch, hohe Frequenz bei international tätigen Unternehmen, einfache Integration.

**Independent Test**: Nutzer kann eine einzelne UID validieren und erhält sofort das Ergebnis mit Firmendaten.

**Acceptance Scenarios**:

1. **Given** eine gültige österreichische UID, **When** der Nutzer `fo uid check ATU12345678` ausführt, **Then** werden Firmenname und Adresse angezeigt
2. **Given** eine ungültige UID, **When** der Nutzer die Validierung ausführt, **Then** wird eine klare Fehlermeldung angezeigt
3. **Given** eine CSV-Datei mit UIDs, **When** der Nutzer `fo uid batch uids.csv` ausführt, **Then** werden alle UIDs validiert und Ergebnisse als Report ausgegeben

---

### User Story 3 - ELDA Dienstnehmer-Anmeldung (Priority: P2)

Eine HR-Abteilung muss neue Mitarbeiter bei der Sozialversicherung anmelden. Dies geschieht aktuell manuell über das ELDA-Portal oder teure Softwarelösungen.

**Why this priority**: Jede Neueinstellung erfordert ELDA-Meldung. Häufiger Use Case mit hohem Automatisierungspotenzial.

**Independent Test**: HR-Mitarbeiter kann einen neuen Dienstnehmer mit allen Pflichtdaten anmelden und erhält Bestätigung.

**Acceptance Scenarios**:

1. **Given** vollständige Mitarbeiterdaten (SV-Nummer, Name, Eintrittsdatum, Entgelt), **When** der Nutzer `elda anmeldung` ausführt, **Then** wird die Anmeldung übermittelt und Referenznummer angezeigt
2. **Given** unvollständige Daten, **When** der Nutzer die Anmeldung versucht, **Then** werden fehlende Pflichtfelder aufgelistet
3. **Given** eine erfolgreiche Anmeldung, **When** der Nutzer `elda status <referenz>` ausführt, **Then** wird der aktuelle Bearbeitungsstatus angezeigt

---

### User Story 4 - Firmenbuch-Auszug (Priority: P2)

Ein M&A-Berater muss für Due Diligence regelmäßig Firmenbuchauszüge abrufen. Manuelles Abrufen über das Justizportal ist zeitaufwändig bei vielen Unternehmen.

**Why this priority**: Due Diligence, KYC/AML-Compliance sind gesetzliche Pflichten. Hoher Zeitaufwand bei mehreren Unternehmen.

**Independent Test**: Nutzer kann per Firmenbuchnummer oder Name einen aktuellen Auszug abrufen.

**Acceptance Scenarios**:

1. **Given** eine Firmenbuchnummer (z.B. FN 123456a), **When** der Nutzer `fb extract FN123456a` ausführt, **Then** wird der strukturierte Firmenbuchauszug angezeigt
2. **Given** ein Firmenname, **When** der Nutzer `fb search "Muster GmbH"` ausführt, **Then** werden alle passenden Firmen mit FN aufgelistet
3. **Given** eine Watchlist mit Firmenbuchnummern, **When** der Nutzer `fb monitor --watchlist` ausführt, **Then** werden Änderungen seit letzter Prüfung angezeigt

---

### User Story 5 - E-Rechnung Erstellung (Priority: P2)

Ein Lieferant muss Rechnungen im ZUGFeRD/XRechnung-Format an öffentliche Auftraggeber senden. Die manuelle Erstellung dieser XML-Rechnungen ist komplex und fehleranfällig.

**Why this priority**: B2G-Pflicht bereits aktiv, B2B-Pflicht kommt. Frühzeitige Adoption verschafft Wettbewerbsvorteil.

**Independent Test**: Nutzer kann aus Rechnungsdaten eine valide XRechnung erstellen und als PDF/XML exportieren.

**Acceptance Scenarios**:

1. **Given** Rechnungsdaten (Empfänger, Positionen, Beträge), **When** der Nutzer `erechnung create invoice.json` ausführt, **Then** wird eine EN16931-konforme XML-Rechnung erstellt
2. **Given** eine bestehende XML-Rechnung, **When** der Nutzer `erechnung validate invoice.xml` ausführt, **Then** werden Validierungsergebnisse nach EN16931 angezeigt
3. **Given** ein PDF mit eingebetteter Rechnung, **When** der Nutzer `erechnung extract invoice.pdf` ausführt, **Then** werden die strukturierten Rechnungsdaten extrahiert

---

### User Story 6 - SEPA-Zahlungsdatei Erstellung (Priority: P3)

Die Buchhaltung muss regelmäßig Sammelüberweisungen (SEPA Credit Transfer) erstellen. Die manuelle Erstellung von pain.001-Dateien ist technisch anspruchsvoll.

**Why this priority**: Jedes Unternehmen braucht SEPA. Häufige Aufgabe mit Fehlerpotenzial.

**Independent Test**: Nutzer kann aus einer Zahlungsliste eine valide pain.001-Datei für den Bankimport erstellen.

**Acceptance Scenarios**:

1. **Given** eine CSV mit Zahlungsdaten (IBAN, Betrag, Verwendungszweck), **When** der Nutzer `sepa pain001 payments.csv` ausführt, **Then** wird eine bankkompatible pain.001-XML erstellt
2. **Given** eine Kontoauszugsdatei (camt.053), **When** der Nutzer `sepa camt053 statement.xml` ausführt, **Then** werden die Umsätze strukturiert ausgegeben
3. **Given** eine IBAN, **When** der Nutzer `sepa validate AT12 1234 1234 1234 1234` ausführt, **Then** wird IBAN-Gültigkeit und zugehörige BIC angezeigt

---

### User Story 7 - MCP Server für AI-Integration (Priority: P3)

Ein Entwickler möchte die Austrian Business Infrastructure in AI-Assistenten (Claude, GPT) integrieren, um Steuerberatungsaufgaben zu automatisieren.

**Why this priority**: MCP ist Differentiator für AI-First-Workflows. Ermöglicht neue Use Cases und Integrationen.

**Independent Test**: AI-Assistent kann über MCP-Protokoll auf FinanzOnline-Funktionen zugreifen.

**Acceptance Scenarios**:

1. **Given** ein konfigurierter MCP-Server, **When** ein AI-Client `fo-databox-list` aufruft, **Then** werden Databox-Einträge als strukturierte Daten zurückgegeben
2. **Given** ein MCP-fähiger Client, **When** `fo-uid-validate` mit einer UID aufgerufen wird, **Then** wird das Validierungsergebnis zurückgegeben
3. **Given** mehrere MCP-Server (fo, elda, fb), **When** der Nutzer alle Server startet, **Then** sind alle Tools im AI-Client verfügbar

---

### User Story 8 - Zusammenfassende Meldung (Priority: P3)

Ein Unternehmen mit innergemeinschaftlichen Lieferungen muss quartalsweise die Zusammenfassende Meldung (ZM) einreichen.

**Why this priority**: Gesetzliche Pflicht für EU-Handel, Basis bereits durch UVA-Implementation vorhanden.

**Independent Test**: Nutzer kann ZM-Daten eingeben und an FinanzOnline übermitteln.

**Acceptance Scenarios**:

1. **Given** ZM-Daten (Partner-UIDs, Beträge nach Lieferart), **When** der Nutzer `fo zm submit` ausführt, **Then** wird die ZM übermittelt und bestätigt
2. **Given** Buchhaltungsdaten, **When** der Nutzer `fo zm generate --period Q4-2025` ausführt, **Then** wird eine ZM-Datei aus den Daten generiert

---

### Edge Cases

- Was passiert bei Timeout während FinanzOnline-Übermittlung? → Automatische Retry mit exponential backoff, Status-Check nach Reconnect
- Wie werden ELDA-Meldungen bei Wartungsfenstern behandelt? → Queue für spätere Übermittlung, Benachrichtigung an Nutzer
- Was passiert bei ungültigen Firmenbuchnummern? → Klare Fehlermeldung mit Vorschlägen ähnlicher Nummern
- Wie werden E-Rechnungen mit fehlenden Pflichtfeldern behandelt? → Detaillierte Validierungsfehler mit Feldnamen und Anforderungen
- Was passiert bei SEPA-Dateien mit ungültigen IBANs? → Zeile wird markiert, Report mit allen Fehlern

---

## Requirements

### Functional Requirements

#### FinanzOnline Erweiterungen

- **FR-001**: System MUSS UVA-Daten im XML-Format gemäß BMF-Schema erstellen können
- **FR-002**: System MUSS UVA-Dateien über fileUploadService an FinanzOnline übermitteln können
- **FR-003**: System MUSS UID-Nummern über uidService validieren und Firmendaten abrufen können
- **FR-004**: System MUSS Batch-Validierung von UIDs aus CSV-Dateien unterstützen
- **FR-005**: System MUSS Zusammenfassende Meldungen erstellen und übermitteln können
- **FR-006**: System MUSS Übermittlungsstatus für alle eingereichten Dokumente abfragen können

#### ELDA Suite

- **FR-010**: System MUSS Dienstnehmer-Anmeldungen (AN-Meldung) erstellen und übermitteln können
- **FR-011**: System MUSS Dienstnehmer-Abmeldungen (AB-Meldung) verarbeiten können
- **FR-012**: System MUSS Änderungsmeldungen für bestehende Dienstverhältnisse unterstützen
- **FR-013**: System MUSS Jahreslohnzettel (L16) generieren und übermitteln können
- **FR-014**: System MUSS Beitragsnachweisungen erstellen können
- **FR-015**: System MUSS Meldungsstatus bei ELDA abfragen können
- **FR-016**: System MUSS ELDA-Credentials sicher und getrennt von FinanzOnline speichern

#### Firmenbuch Suite

- **FR-020**: System MUSS Firmenbuchsuche nach Name, FN-Nummer und weiteren Kriterien unterstützen
- **FR-021**: System MUSS strukturierte Firmenbuchauszüge abrufen und anzeigen können
- **FR-022**: System MUSS Änderungsüberwachung (Monitoring) für definierte Unternehmen bieten
- **FR-023**: System MUSS Insolvenzabfragen über die Ediktsdatei unterstützen
- **FR-024**: System MUSS GISA-Abfragen (Gewerbeinformationssystem) ermöglichen

#### E-Rechnung Suite

- **FR-030**: System MUSS XRechnung/ZUGFeRD-konforme Rechnungen erstellen können
- **FR-031**: System MUSS Rechnungen gegen EN16931-Standard validieren
- **FR-032**: System MUSS Rechnungsdaten aus PDF/XML extrahieren können
- **FR-033**: System MUSS Format-Konvertierung zwischen ZUGFeRD und XRechnung unterstützen
- **FR-034**: System MUSS Rechnungen an Peppol-Netzwerk/USP senden können

#### SEPA Toolkit

- **FR-040**: System MUSS pain.001-Dateien (Credit Transfer) generieren können
- **FR-041**: System MUSS pain.008-Dateien (Direct Debit) generieren können
- **FR-042**: System MUSS camt.053-Kontoauszüge parsen und strukturieren können
- **FR-043**: System MUSS camt.054-Einzelumsätze verarbeiten können
- **FR-044**: System MUSS IBAN/BIC-Validierung mit Bankzuordnung bieten

#### MCP Server Integration

- **FR-050**: System MUSS MCP-Server für FinanzOnline-Funktionen bereitstellen
- **FR-051**: System MUSS MCP-Server für ELDA-Funktionen bereitstellen
- **FR-052**: System MUSS MCP-Server für Firmenbuch-Funktionen bereitstellen
- **FR-053**: MCP-Server MÜSSEN strukturierte JSON-Responses liefern
- **FR-054**: MCP-Server MÜSSEN sichere Credential-Handhabung gewährleisten

#### Cross-Cutting Concerns

- **FR-060**: Alle Module MÜSSEN einheitliche CLI-Patterns verwenden (flags, output-formate)
- **FR-061**: Alle Module MÜSSEN JSON-Output für Scripting/Integration unterstützen
- **FR-062**: Alle Module MÜSSEN detailliertes Error-Handling mit actionable Messages bieten
- **FR-063**: System MUSS Cross-Platform funktionieren (Windows, Linux, macOS)
- **FR-064**: Alle sensiblen Daten MÜSSEN verschlüsselt gespeichert werden
- **FR-065**: System DARF KEINE Credentials oder sensible Daten loggen

---

### Key Entities

- **UVA (Umsatzsteuervoranmeldung)**: Steuererklärung mit Zeitraum, Kennzahlen (KZ), Beträgen, Übermittlungsstatus
- **UID-Validierung**: UID-Nummer, Validierungsergebnis, Firmenname, Adresse, Gültigkeitsdatum
- **ELDA-Meldung**: Meldungsart, Dienstnehmerdaten (SV-Nummer, Name, Geburtsdatum), Dienstgeberdaten, Beschäftigungsdaten, Status
- **Firmenbucheintrag**: FN-Nummer, Firma, Rechtsform, Sitz, Geschäftsführer, Kapital, Gesellschafter, Prokuristen
- **E-Rechnung**: Rechnungsnummer, Rechnungsdatum, Leistungszeitraum, Positionen, Steuersätze, Summen, Käufer/Verkäufer, Leitweg-ID
- **SEPA-Zahlung**: Zahlungsart, Auftraggeber-IBAN, Empfänger-IBAN, Betrag, Währung, Verwendungszweck, Ausführungsdatum

---

## Success Criteria

### Measurable Outcomes

- **SC-001**: Steuerberater kann UVA für 50 Mandanten in unter 30 Minuten einreichen (vs. 5+ Stunden manuell)
- **SC-002**: UID-Batch-Validierung von 1000 Nummern erfolgt in unter 5 Minuten
- **SC-003**: ELDA-Anmeldung eines Dienstnehmers dauert unter 2 Minuten (vs. 15+ Minuten Portal)
- **SC-004**: Firmenbuch-Monitoring für 100 Unternehmen liefert Änderungen in unter 1 Minute
- **SC-005**: E-Rechnung-Erstellung aus strukturierten Daten dauert unter 10 Sekunden
- **SC-006**: SEPA-Datei mit 500 Zahlungen wird in unter 30 Sekunden generiert
- **SC-007**: Alle CLI-Tools funktionieren identisch auf Windows, Linux und macOS
- **SC-008**: 95% aller Nutzer können ohne Dokumentation eine erfolgreiche Erstoperation durchführen
- **SC-009**: Fehlerrate bei automatisierter Übermittlung liegt unter 0.1% (vs. manuelle Eingabefehler)
- **SC-010**: MCP-Integration ermöglicht AI-Assistenten, Standardaufgaben vollständig selbstständig auszuführen

---

## Assumptions

1. FinanzOnline WebService-Zugang (TID + BenID + PIN) ist für alle UVA/UID/ZM-Funktionen vorhanden
2. ELDA-Zugang erfordert separate Credentials (Dienstgeberkonto)
3. Firmenbuch-Abfragen erfolgen über öffentliche Justiz-Schnittstellen
4. E-Rechnung-Versand an USP/Peppol erfordert entsprechende Registrierung
5. SEPA-Dateien werden lokal generiert und vom Nutzer in Banking-Software importiert
6. MCP-Server laufen lokal und erfordern keine Cloud-Infrastruktur
7. Alle Module teilen den bestehenden Credential-Store mit separaten Schlüsseln pro Service

---

## Dependencies

- Spec 001 (FinanzOnline CLI Basis) MUSS abgeschlossen sein
- FinanzOnline fileUploadService WSDL für UVA/ZM-Upload
- FinanzOnline uidService WSDL für UID-Validierung
- ELDA-Schnittstellen-Dokumentation
- Firmenbuch/Justiz-API-Dokumentation
- EN16931 E-Rechnung Standards
- SEPA ISO 20022 XML-Schemas
- MCP SDK-Dokumentation

---

## Out of Scope

- Einkommensteuer (EST) und Körperschaftsteuer (KEST) Erklärungen (komplexe Formulare, spätere Phase)
- Austrian Business Cloud (SaaS-Version mit Team-Features)
- Mobile Apps
- White-Label/Reseller-Funktionen
- Buchhaltungs-/ERP-Funktionen (nur Schnittstellen)
