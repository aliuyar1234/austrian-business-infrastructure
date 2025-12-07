package elda

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// ELDANS is the namespace for ELDA services
	ELDANS = "https://www.elda.at/elda"

	// ELDAEndpoint is the production ELDA service endpoint
	ELDAEndpoint = "https://www.elda.at/elda/ws/meldung"

	// ELDATestEndpoint is the test ELDA service endpoint
	ELDATestEndpoint = "https://test.elda.at/elda/ws/meldung"
)

var (
	ErrELDAConnection    = errors.New("ELDA connection failed")
	ErrELDAAuthentication = errors.New("ELDA authentication failed")
	ErrELDAValidation    = errors.New("ELDA validation failed")
)

// Client handles ELDA API communication
type Client struct {
	endpoint   string
	httpClient *http.Client
	timeout    time.Duration
}

// NewClient creates a new ELDA client
func NewClient(testMode bool) *Client {
	endpoint := ELDAEndpoint
	if testMode {
		endpoint = ELDATestEndpoint
	}

	return &Client{
		endpoint: endpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		timeout: 30 * time.Second,
	}
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
	// Marshal request body
	body, err := xml.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Wrap in SOAP envelope
	soapBody := soapEnvelope(body)

	// Create HTTP request
	req, err := http.NewRequest("POST", c.endpoint, bytes.NewReader(soapBody))
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
