package fonws

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
)

// GetDataboxInfoRequest represents a SOAP GetDataboxInfo request
type GetDataboxInfoRequest struct {
	XMLName   xml.Name `xml:"GetDataboxInfo"`
	Xmlns     string   `xml:"xmlns,attr"`
	ID        string   `xml:"id"`
	TID       string   `xml:"tid"`
	BenID     string   `xml:"benid"`
	TsZustVon string   `xml:"ts_zust_von,omitempty"`
	TsZustBis string   `xml:"ts_zust_bis,omitempty"`
}

// GetDataboxInfoResponse represents a SOAP GetDataboxInfo response
type GetDataboxInfoResponse struct {
	XMLName xml.Name      `xml:"GetDataboxInfoResponse"`
	RC      int           `xml:"rc"`
	Msg     string        `xml:"msg"`
	Result  DataboxResult `xml:"result"`
}

// DataboxResult contains the list of databox entries
type DataboxResult struct {
	Entries []DataboxEntry `xml:"databox"`
}

// DataboxEntry represents a single document in the databox
type DataboxEntry struct {
	Applkey   string `xml:"applkey"`
	Filebez   string `xml:"filebez"`
	TsZust    string `xml:"ts_zust"`
	Erlession string `xml:"erlession"`
	Veression string `xml:"veression"`
}

// ActionRequired returns true if this document requires user action
func (e DataboxEntry) ActionRequired() bool {
	return e.Erlession == "E" || e.Erlession == "V"
}

// TypeName returns the human-readable document type
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
	XMLName xml.Name        `xml:"GetDataboxResponse"`
	RC      int             `xml:"rc"`
	Msg     string          `xml:"msg"`
	Result  DataboxDownload `xml:"result"`
}

// DataboxDownload contains the downloaded document
type DataboxDownload struct {
	Filename string `xml:"filename"`
	Content  string `xml:"content"` // Base64 encoded
}

// DataboxService handles databox operations
type DataboxService struct {
	client *Client
}

// NewDataboxService creates a new databox service
func NewDataboxService(client *Client) *DataboxService {
	return &DataboxService{client: client}
}

// GetInfo retrieves the list of documents in the databox
func (s *DataboxService) GetInfo(session *Session, fromDate, toDate string) ([]DataboxEntry, error) {
	if session == nil || !session.Valid {
		return nil, ErrNoActiveSession
	}

	req := GetDataboxInfoRequest{
		Xmlns:     DataboxNS,
		ID:        session.Token,
		TID:       session.TID,
		BenID:     session.BenID,
		TsZustVon: fromDate,
		TsZustBis: toDate,
	}

	var resp GetDataboxInfoResponse
	if err := s.client.Call(DataboxServiceURL, req, &resp); err != nil {
		return nil, err
	}

	if err := CheckResponse(resp.RC, resp.Msg); err != nil {
		if IsSessionExpired(err) {
			session.Invalidate()
		}
		return nil, err
	}

	return resp.Result.Entries, nil
}

// Download downloads a document from the databox and saves it to the specified directory
func (s *DataboxService) Download(session *Session, applkey, outputDir string) (string, error) {
	if session == nil || !session.Valid {
		return "", ErrNoActiveSession
	}

	req := GetDataboxRequest{
		Xmlns:   DataboxNS,
		ID:      session.Token,
		TID:     session.TID,
		BenID:   session.BenID,
		Applkey: applkey,
	}

	var resp GetDataboxResponse
	if err := s.client.Call(DataboxServiceURL, req, &resp); err != nil {
		return "", err
	}

	if err := CheckResponse(resp.RC, resp.Msg); err != nil {
		if IsSessionExpired(err) {
			session.Invalidate()
		}
		return "", err
	}

	// Decode base64 content
	content, err := base64.StdEncoding.DecodeString(resp.Result.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode document content: %w", err)
	}

	// Determine filename
	filename := resp.Result.Filename
	if filename == "" {
		filename = applkey + ".pdf"
	}

	// Write to file
	outputPath := filepath.Join(outputDir, filename)
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputPath, nil
}

// DownloadToBytes downloads a document and returns the content as bytes
func (s *DataboxService) DownloadToBytes(session *Session, applkey string) ([]byte, string, error) {
	if session == nil || !session.Valid {
		return nil, "", ErrNoActiveSession
	}

	req := GetDataboxRequest{
		Xmlns:   DataboxNS,
		ID:      session.Token,
		TID:     session.TID,
		BenID:   session.BenID,
		Applkey: applkey,
	}

	var resp GetDataboxResponse
	if err := s.client.Call(DataboxServiceURL, req, &resp); err != nil {
		return nil, "", err
	}

	if err := CheckResponse(resp.RC, resp.Msg); err != nil {
		if IsSessionExpired(err) {
			session.Invalidate()
		}
		return nil, "", err
	}

	content, err := base64.StdEncoding.DecodeString(resp.Result.Content)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode document content: %w", err)
	}

	return content, resp.Result.Filename, nil
}
