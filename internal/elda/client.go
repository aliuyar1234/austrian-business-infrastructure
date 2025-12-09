package elda

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const (
	// ELDANS is the namespace for ELDA services
	ELDANS = "https://www.elda.at/elda"

	// ELDAEndpoint is the production ELDA service endpoint
	ELDAEndpoint = "https://elda.sozvers.at/elda-webservice/"

	// ELDATestEndpoint is the test ELDA service endpoint
	ELDATestEndpoint = "https://elda-test.sozvers.at/elda-webservice/"

	// Default timeout for ELDA requests
	DefaultTimeout = 60 * time.Second

	// Maximum retries for transient errors
	DefaultMaxRetries = 3
)

var (
	ErrELDAConnection    = errors.New("ELDA connection failed")
	ErrELDAAuthentication = errors.New("ELDA authentication failed")
	ErrELDAValidation    = errors.New("ELDA validation failed")
)

// ClientConfig holds configuration for the ELDA client
type ClientConfig struct {
	Endpoint    string
	TestMode    bool
	Timeout     time.Duration
	MaxRetries  int
	Certificate *Certificate
	Logger      *slog.Logger
}

// Client handles ELDA API communication
type Client struct {
	endpoint    string
	httpClient  *http.Client
	timeout     time.Duration
	maxRetries  int
	certificate *Certificate
	logger      *slog.Logger
}

// NewClient creates a new ELDA client
func NewClient(testMode bool) *Client {
	return NewClientWithConfig(ClientConfig{
		TestMode:   testMode,
		Timeout:    DefaultTimeout,
		MaxRetries: DefaultMaxRetries,
	})
}

// NewClientWithConfig creates a new ELDA client with configuration
func NewClientWithConfig(cfg ClientConfig) *Client {
	endpoint := ELDAEndpoint
	if cfg.TestMode {
		endpoint = ELDATestEndpoint
	}
	if cfg.Endpoint != "" {
		endpoint = cfg.Endpoint
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = DefaultMaxRetries
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Create HTTP client with TLS config if certificate provided
	httpClient := &http.Client{
		Timeout: timeout,
	}

	if cfg.Certificate != nil && cfg.Certificate.TLSCert != nil {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{*cfg.Certificate.TLSCert},
				MinVersion:   tls.VersionTLS12,
			},
		}
		httpClient.Transport = transport
	}

	return &Client{
		endpoint:    endpoint,
		httpClient:  httpClient,
		timeout:     timeout,
		maxRetries:  maxRetries,
		certificate: cfg.Certificate,
		logger:      logger,
	}
}

// SetCertificate updates the client's certificate
func (c *Client) SetCertificate(cert *Certificate) {
	c.certificate = cert
	if cert != nil && cert.TLSCert != nil {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{*cert.TLSCert},
				MinVersion:   tls.VersionTLS12,
			},
		}
		c.httpClient.Transport = transport
	}
}

// GetEndpoint returns the current endpoint
func (c *Client) GetEndpoint() string {
	return c.endpoint
}

// soapEnvelope wraps a request in a SOAP envelope
func soapEnvelope(body []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">`)
	buf.WriteString(`<soap:Body>`)
	buf.Write(body)
	buf.WriteString(`</soap:Body>`)
	buf.WriteString(`</soap:Envelope>`)
	return buf.Bytes()
}

// call makes a SOAP call to ELDA
func (c *Client) call(action string, request interface{}, response interface{}) error {
	return c.callWithContext(context.Background(), action, request, response)
}

// callWithContext makes a SOAP call to ELDA with context
func (c *Client) callWithContext(ctx context.Context, action string, request interface{}, response interface{}) error {
	// Marshal request body
	body, err := xml.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Wrap in SOAP envelope
	soapBody := soapEnvelope(body)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(soapBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", action)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrELDAConnection, err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: HTTP %d", ErrELDAConnection, resp.StatusCode)
	}

	// Parse response (extract from SOAP envelope)
	return parseSOAPResponse(respBody, response)
}

// callWithRetry makes a SOAP call with retry logic for transient errors
func (c *Client) callWithRetry(ctx context.Context, action string, request interface{}, response interface{}) error {
	backoff := []time.Duration{1 * time.Second, 5 * time.Second, 30 * time.Second}

	var lastErr error
	for i := 0; i <= c.maxRetries; i++ {
		err := c.callWithContext(ctx, action, request, response)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !IsRetryable(err) && !errors.Is(err, ErrELDAConnection) {
			return err
		}

		// Check context
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Don't wait after the last attempt
		if i < c.maxRetries && i < len(backoff) {
			c.logger.Warn("ELDA request failed, retrying",
				"attempt", i+1,
				"wait", backoff[i],
				"error", err)

			select {
			case <-time.After(backoff[i]):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("ELDA request failed after %d retries: %w", c.maxRetries, lastErr)
}

// TestConnection tests the connection to ELDA
func (c *Client) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	start := time.Now()

	// Simple ping request
	type pingRequest struct {
		XMLName xml.Name `xml:"Ping"`
		XMLNS   string   `xml:"xmlns,attr"`
	}
	type pingResponse struct {
		XMLName    xml.Name  `xml:"PingResponse"`
		ServerTime time.Time `xml:"ServerTime"`
	}

	req := pingRequest{XMLNS: ELDANS}
	var resp pingResponse

	err := c.callWithContext(ctx, "Ping", &req, &resp)
	latency := time.Since(start)

	result := &ConnectionTestResult{
		LatencyMs: int(latency.Milliseconds()),
	}

	if err != nil {
		result.Connected = false
		result.Error = err.Error()
		return result, err
	}

	result.Connected = true
	result.ServerTime = resp.ServerTime

	return result, nil
}

// ConnectionTestResult contains the result of a connection test
type ConnectionTestResult struct {
	Connected  bool      `json:"connected"`
	LatencyMs  int       `json:"latency_ms"`
	ServerTime time.Time `json:"server_time,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// parseSOAPResponse extracts the body from a SOAP response
func parseSOAPResponse(data []byte, v interface{}) error {
	// Simple SOAP body extraction
	type soapBody struct {
		Content []byte `xml:",innerxml"`
	}
	type soapEnvelope struct {
		Body soapBody `xml:"Body"`
	}

	var env soapEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("failed to parse SOAP envelope: %w", err)
	}

	if err := xml.Unmarshal(env.Body.Content, v); err != nil {
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	return nil
}

// SubmitAnmeldung submits an employee registration
func (c *Client) SubmitAnmeldung(creds *ELDACredentials, anmeldung *ELDAAnmeldung) (*ELDAResponse, error) {
	// Validate SV-Nummer
	if err := ValidateSVNummer(anmeldung.SVNummer); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrELDAValidation, err)
	}

	// Build XML document
	doc := ELDAAnmeldungDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "AN",
		},
		SVNummer:       anmeldung.SVNummer,
		Vorname:        anmeldung.Vorname,
		Nachname:       anmeldung.Nachname,
		Geburtsdatum:   anmeldung.GeburtsdatumString(),
		Geschlecht:     anmeldung.Geschlecht,
		Eintrittsdatum: anmeldung.EintrittsdatumString(),
		Beschaeftigung: anmeldung.Beschaeftigung,
		Arbeitszeit:    anmeldung.Arbeitszeit,
		Entgelt:        anmeldung.Entgelt,
	}

	var resp ELDAResponse
	err := c.call("SubmitAnmeldung", &doc, &resp)
	if err != nil {
		return nil, err
	}

	if resp.RC != 0 {
		return &resp, fmt.Errorf("ELDA error (code %d): %s", resp.RC, resp.Msg)
	}

	// Update anmeldung status
	anmeldung.Status = ELDAStatusSubmitted
	anmeldung.Reference = resp.Reference

	return &resp, nil
}

// SubmitAbmeldung submits an employee deregistration
func (c *Client) SubmitAbmeldung(creds *ELDACredentials, abmeldung *ELDAAbmeldung) (*ELDAResponse, error) {
	// Validate SV-Nummer
	if err := ValidateSVNummer(abmeldung.SVNummer); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrELDAValidation, err)
	}

	// Build XML document
	doc := ELDAAbmeldungDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "AB",
		},
		SVNummer:       abmeldung.SVNummer,
		Austrittsdatum: abmeldung.AustrittsdatumString(),
		Grund:          abmeldung.Grund,
		Abfertigung:    abmeldung.Abfertigung,
		Urlaubsersatz:  abmeldung.Urlaubsersatz,
	}

	var resp ELDAResponse
	err := c.call("SubmitAbmeldung", &doc, &resp)
	if err != nil {
		return nil, err
	}

	if resp.RC != 0 {
		return &resp, fmt.Errorf("ELDA error (code %d): %s", resp.RC, resp.Msg)
	}

	// Update abmeldung status
	abmeldung.Status = ELDAStatusSubmitted
	abmeldung.Reference = resp.Reference

	return &resp, nil
}

// QueryStatus queries the status of a submission
func (c *Client) QueryStatus(creds *ELDACredentials, reference string) (*ELDAResponse, error) {
	type statusRequest struct {
		XMLName       xml.Name `xml:"StatusAbfrage"`
		XMLNS         string   `xml:"xmlns,attr"`
		DienstgeberNr string   `xml:"DienstgeberNr"`
		Referenz      string   `xml:"Referenz"`
	}

	req := statusRequest{
		XMLNS:         ELDANS,
		DienstgeberNr: creds.DienstgeberNr,
		Referenz:      reference,
	}

	var resp ELDAResponse
	err := c.call("StatusAbfrage", &req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GenerateAnmeldungXML generates XML from an Anmeldung
func GenerateAnmeldungXML(creds *ELDACredentials, anmeldung *ELDAAnmeldung) ([]byte, error) {
	if err := ValidateSVNummer(anmeldung.SVNummer); err != nil {
		return nil, fmt.Errorf("invalid SV-Nummer: %w", err)
	}

	doc := ELDAAnmeldungDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "AN",
		},
		SVNummer:       anmeldung.SVNummer,
		Vorname:        anmeldung.Vorname,
		Nachname:       anmeldung.Nachname,
		Geburtsdatum:   anmeldung.GeburtsdatumString(),
		Geschlecht:     anmeldung.Geschlecht,
		Eintrittsdatum: anmeldung.EintrittsdatumString(),
		Beschaeftigung: anmeldung.Beschaeftigung,
		Arbeitszeit:    anmeldung.Arbeitszeit,
		Entgelt:        anmeldung.Entgelt,
	}

	output, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	result := []byte(xml.Header)
	result = append(result, output...)
	return result, nil
}

// SubmitExtendedMeldung submits an extended meldung (An-/Ab-/Änderungsmeldung)
func (c *Client) SubmitExtendedMeldung(ctx context.Context, creds *ELDACredentials, meldung *ELDAMeldung) (*MeldungResponse, error) {
	// Build XML based on meldung type
	var xmlDoc interface{}
	var action string

	switch meldung.Type {
	case MeldungTypeAnmeldung:
		xmlDoc = buildExtendedAnmeldungXML(creds, meldung)
		action = "SubmitAnmeldung"
	case MeldungTypeAbmeldung:
		xmlDoc = buildExtendedAbmeldungXML(creds, meldung)
		action = "SubmitAbmeldung"
	case MeldungTypeAenderung:
		xmlDoc = buildAenderungXML(creds, meldung)
		action = "SubmitAenderung"
	case MeldungTypeKorrektur:
		xmlDoc = buildKorrekturXML(creds, meldung)
		action = "SubmitKorrektur"
	default:
		return nil, fmt.Errorf("unsupported meldung type: %s", meldung.Type)
	}

	var resp MeldungResponse
	err := c.callWithContext(ctx, action, xmlDoc, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// buildExtendedAnmeldungXML creates extended Anmeldung XML
func buildExtendedAnmeldungXML(creds *ELDACredentials, m *ELDAMeldung) *ExtendedAnmeldungDocument {
	doc := &ExtendedAnmeldungDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "AN",
		},
		SVNummer: m.SVNummer,
		Vorname:  m.Vorname,
		Nachname: m.Nachname,
	}

	if m.Geburtsdatum != nil {
		doc.Geburtsdatum = m.Geburtsdatum.Format("2006-01-02")
	}
	doc.Geschlecht = m.Geschlecht

	if m.Eintrittsdatum != nil {
		doc.Eintrittsdatum = m.Eintrittsdatum.Format("2006-01-02")
	}

	if m.Beschaeftigung != nil {
		doc.Beschaeftigung = &XMLBeschaeftigung{
			Art:               m.Beschaeftigung.Art,
			Taetigkeit:        m.Beschaeftigung.Taetigkeit,
			KollektivCode:     m.Beschaeftigung.KollektivCode,
			Einstufung:        m.Beschaeftigung.Einstufung,
			Verwendungsgruppe: m.Beschaeftigung.Verwendungsgruppe,
			Dienstort:         m.Beschaeftigung.Dienstort,
			Beitragsgruppe:    m.Beschaeftigung.Beitragsgruppe,
		}
		if m.Beschaeftigung.Befristet {
			doc.Beschaeftigung.Befristet = "J"
			doc.Beschaeftigung.BefristetBis = m.Beschaeftigung.BefristetBis
		}
	}

	if m.Arbeitszeit != nil {
		doc.Arbeitszeit = &XMLArbeitszeit{
			WochenStunden:   fmt.Sprintf("%.2f", m.Arbeitszeit.WochenStunden),
			TageProWoche:    m.Arbeitszeit.TageProWoche,
			ArbeitszeitCode: m.Arbeitszeit.ArbeitszeitCode,
		}
		if m.Arbeitszeit.Schichtarbeit {
			doc.Arbeitszeit.Schichtarbeit = "J"
		}
		if m.Arbeitszeit.KollektivStunden > 0 {
			doc.Arbeitszeit.KollektivStunden = fmt.Sprintf("%.2f", m.Arbeitszeit.KollektivStunden)
		}
	}

	if m.Entgelt != nil {
		doc.Entgelt = &XMLEntgelt{
			BruttoMonatlich: fmt.Sprintf("%.2f", float64(m.Entgelt.BruttoMonatlich)/100),
			EntgeltArt:      m.Entgelt.EntgeltArt,
		}
		if m.Entgelt.NettoMonatlich > 0 {
			doc.Entgelt.NettoMonatlich = fmt.Sprintf("%.2f", float64(m.Entgelt.NettoMonatlich)/100)
		}
		if m.Entgelt.Sonderzahlungen > 0 {
			doc.Entgelt.Sonderzahlungen = fmt.Sprintf("%.2f", float64(m.Entgelt.Sonderzahlungen)/100)
		}
	}

	if m.Adresse != nil {
		doc.Adresse = &XMLAdresse{
			Strasse:    m.Adresse.Strasse,
			Hausnummer: m.Adresse.Hausnummer,
			Stiege:     m.Adresse.Stiege,
			Tuer:       m.Adresse.Tuer,
			PLZ:        m.Adresse.PLZ,
			Ort:        m.Adresse.Ort,
			Land:       m.Adresse.Land,
		}
	}

	if m.Bankverbindung != nil {
		doc.Bankverbindung = &XMLBankverbindung{
			IBAN:         m.Bankverbindung.IBAN,
			BIC:          m.Bankverbindung.BIC,
			Kontoinhaber: m.Bankverbindung.Kontoinhaber,
		}
	}

	return doc
}

// buildExtendedAbmeldungXML creates extended Abmeldung XML
func buildExtendedAbmeldungXML(creds *ELDACredentials, m *ELDAMeldung) *ExtendedAbmeldungDocument {
	doc := &ExtendedAbmeldungDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "AB",
		},
		SVNummer: m.SVNummer,
		Grund:    m.AustrittGrund,
	}

	if m.Austrittsdatum != nil {
		doc.Austrittsdatum = m.Austrittsdatum.Format("2006-01-02")
	}

	doc.Abfertigung = m.Abfertigung
	doc.Urlaubsersatz = m.Urlaubsersatz
	doc.URLTage = m.URLTage

	return doc
}

// AenderungDocument represents an Änderungsmeldung
type AenderungDocument struct {
	XMLName        xml.Name      `xml:"Aenderung"`
	XMLNS          string        `xml:"xmlns,attr"`
	Kopf           ELDAKopf      `xml:"Kopf"`
	SVNummer       string        `xml:"SVNummer"`
	AenderungArt   string        `xml:"AenderungArt"`
	AenderungDatum string        `xml:"AenderungDatum"`
	OriginalRef    string        `xml:"OriginalReferenz,omitempty"`
	Beschaeftigung *XMLBeschaeftigung `xml:"Beschaeftigung,omitempty"`
	Arbeitszeit    *XMLArbeitszeit    `xml:"Arbeitszeit,omitempty"`
	Entgelt        *XMLEntgelt        `xml:"Entgelt,omitempty"`
	Adresse        *XMLAdresse        `xml:"Adresse,omitempty"`
	Bankverbindung *XMLBankverbindung `xml:"Bankverbindung,omitempty"`
}

// buildAenderungXML creates Änderungsmeldung XML
func buildAenderungXML(creds *ELDACredentials, m *ELDAMeldung) *AenderungDocument {
	doc := &AenderungDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "AE",
		},
		SVNummer:     m.SVNummer,
		AenderungArt: m.AenderungArt,
	}

	if m.AenderungDatum != nil {
		doc.AenderungDatum = m.AenderungDatum.Format("2006-01-02")
	}

	// Copy changed data based on AenderungArt
	if m.Beschaeftigung != nil {
		doc.Beschaeftigung = &XMLBeschaeftigung{
			Art:               m.Beschaeftigung.Art,
			Taetigkeit:        m.Beschaeftigung.Taetigkeit,
			KollektivCode:     m.Beschaeftigung.KollektivCode,
			Einstufung:        m.Beschaeftigung.Einstufung,
			Verwendungsgruppe: m.Beschaeftigung.Verwendungsgruppe,
			Dienstort:         m.Beschaeftigung.Dienstort,
			Beitragsgruppe:    m.Beschaeftigung.Beitragsgruppe,
		}
	}

	if m.Arbeitszeit != nil {
		doc.Arbeitszeit = &XMLArbeitszeit{
			WochenStunden:   fmt.Sprintf("%.2f", m.Arbeitszeit.WochenStunden),
			TageProWoche:    m.Arbeitszeit.TageProWoche,
			ArbeitszeitCode: m.Arbeitszeit.ArbeitszeitCode,
		}
	}

	if m.Entgelt != nil {
		doc.Entgelt = &XMLEntgelt{
			BruttoMonatlich: fmt.Sprintf("%.2f", float64(m.Entgelt.BruttoMonatlich)/100),
			EntgeltArt:      m.Entgelt.EntgeltArt,
		}
	}

	if m.Adresse != nil {
		doc.Adresse = &XMLAdresse{
			Strasse:    m.Adresse.Strasse,
			Hausnummer: m.Adresse.Hausnummer,
			PLZ:        m.Adresse.PLZ,
			Ort:        m.Adresse.Ort,
		}
	}

	if m.Bankverbindung != nil {
		doc.Bankverbindung = &XMLBankverbindung{
			IBAN:         m.Bankverbindung.IBAN,
			BIC:          m.Bankverbindung.BIC,
			Kontoinhaber: m.Bankverbindung.Kontoinhaber,
		}
	}

	return doc
}

// KorrekturDocument represents a Korrekturmeldung
type KorrekturDocument struct {
	XMLName     xml.Name `xml:"Korrektur"`
	XMLNS       string   `xml:"xmlns,attr"`
	Kopf        ELDAKopf `xml:"Kopf"`
	OriginalRef string   `xml:"OriginalReferenz"`
	SVNummer    string   `xml:"SVNummer"`
	// Corrected fields follow the original meldung structure
	Beschaeftigung *XMLBeschaeftigung `xml:"Beschaeftigung,omitempty"`
	Arbeitszeit    *XMLArbeitszeit    `xml:"Arbeitszeit,omitempty"`
	Entgelt        *XMLEntgelt        `xml:"Entgelt,omitempty"`
}

// buildKorrekturXML creates Korrekturmeldung XML
func buildKorrekturXML(creds *ELDACredentials, m *ELDAMeldung) *KorrekturDocument {
	doc := &KorrekturDocument{
		XMLNS: ELDANS,
		Kopf: ELDAKopf{
			DienstgeberNr: creds.DienstgeberNr,
			Datum:         time.Now().Format("2006-01-02"),
			MeldungsArt:   "KO",
		},
		SVNummer: m.SVNummer,
	}

	if m.OriginalMeldungID != nil {
		doc.OriginalRef = m.OriginalMeldungID.String()
	}

	// Copy corrected data
	if m.Beschaeftigung != nil {
		doc.Beschaeftigung = &XMLBeschaeftigung{
			Art:            m.Beschaeftigung.Art,
			Beitragsgruppe: m.Beschaeftigung.Beitragsgruppe,
		}
	}

	if m.Arbeitszeit != nil {
		doc.Arbeitszeit = &XMLArbeitszeit{
			WochenStunden: fmt.Sprintf("%.2f", m.Arbeitszeit.WochenStunden),
		}
	}

	if m.Entgelt != nil {
		doc.Entgelt = &XMLEntgelt{
			BruttoMonatlich: fmt.Sprintf("%.2f", float64(m.Entgelt.BruttoMonatlich)/100),
		}
	}

	return doc
}

// GenerateExtendedMeldungXML generates XML for an extended meldung
func GenerateExtendedMeldungXML(creds *ELDACredentials, meldung *ELDAMeldung) ([]byte, error) {
	var xmlDoc interface{}

	switch meldung.Type {
	case MeldungTypeAnmeldung:
		xmlDoc = buildExtendedAnmeldungXML(creds, meldung)
	case MeldungTypeAbmeldung:
		xmlDoc = buildExtendedAbmeldungXML(creds, meldung)
	case MeldungTypeAenderung:
		xmlDoc = buildAenderungXML(creds, meldung)
	case MeldungTypeKorrektur:
		xmlDoc = buildKorrekturXML(creds, meldung)
	default:
		return nil, fmt.Errorf("unsupported meldung type: %s", meldung.Type)
	}

	output, err := xml.MarshalIndent(xmlDoc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal XML: %w", err)
	}

	result := []byte(xml.Header)
	result = append(result, output...)
	return result, nil
}
