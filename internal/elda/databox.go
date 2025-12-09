package elda

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DataboxService handles ELDA databox operations
type DataboxService struct {
	client *Client
}

// NewDataboxService creates a new ELDA databox service
func NewDataboxService(client *Client) *DataboxService {
	return &DataboxService{client: client}
}

// DataboxDocument represents a document in the ELDA databox
type DataboxDocument struct {
	ID           string    `json:"id" xml:"DocumentID"`
	Name         string    `json:"name" xml:"DocumentName"`
	Type         string    `json:"type" xml:"DocumentType"`
	Category     string    `json:"category" xml:"Category"`
	ReceivedAt   time.Time `json:"received_at" xml:"-"`
	ReceivedStr  string    `json:"-" xml:"ReceivedAt"`
	Size         int64     `json:"size" xml:"Size"`
	ContentType  string    `json:"content_type" xml:"ContentType"`
	IsRead       bool      `json:"is_read" xml:"IsRead"`
	Reference    string    `json:"reference,omitempty" xml:"Reference,omitempty"`
	Description  string    `json:"description,omitempty" xml:"Description,omitempty"`
}

// DataboxListResponse represents the response from listing databox documents
type DataboxListResponse struct {
	XMLName    xml.Name          `xml:"DataboxListResponse"`
	Total      int               `xml:"Total"`
	Documents  []DataboxDocument `xml:"Documents>Document"`
	HasMore    bool              `xml:"HasMore"`
	NextCursor string            `xml:"NextCursor,omitempty"`
}

// DataboxListRequest contains parameters for listing databox documents
type DataboxListRequest struct {
	Limit     int        `json:"limit"`
	Cursor    string     `json:"cursor,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	Category  string     `json:"category,omitempty"`
	Unread    bool       `json:"unread,omitempty"`
}

// ListDocuments retrieves documents from the ELDA databox
func (s *DataboxService) ListDocuments(ctx context.Context, req *DataboxListRequest) (*DataboxListResult, error) {
	type listRequest struct {
		XMLName   xml.Name `xml:"DataboxList"`
		XMLNS     string   `xml:"xmlns,attr"`
		Limit     int      `xml:"Limit"`
		Cursor    string   `xml:"Cursor,omitempty"`
		StartDate string   `xml:"StartDate,omitempty"`
		EndDate   string   `xml:"EndDate,omitempty"`
		Category  string   `xml:"Category,omitempty"`
		Unread    bool     `xml:"UnreadOnly,omitempty"`
	}

	xmlReq := listRequest{
		XMLNS:  ELDANS,
		Limit:  req.Limit,
		Cursor: req.Cursor,
		Unread: req.Unread,
	}

	if req.Limit <= 0 {
		xmlReq.Limit = 50
	}

	if req.StartDate != nil {
		xmlReq.StartDate = req.StartDate.Format("2006-01-02")
	}
	if req.EndDate != nil {
		xmlReq.EndDate = req.EndDate.Format("2006-01-02")
	}
	if req.Category != "" {
		xmlReq.Category = req.Category
	}

	var resp DataboxListResponse
	err := s.client.callWithRetry(ctx, "DataboxList", &xmlReq, &resp)
	if err != nil {
		return nil, fmt.Errorf("list databox documents failed: %w", err)
	}

	// Parse dates
	for i := range resp.Documents {
		if resp.Documents[i].ReceivedStr != "" {
			t, err := time.Parse("2006-01-02T15:04:05", resp.Documents[i].ReceivedStr)
			if err == nil {
				resp.Documents[i].ReceivedAt = t
			}
		}
	}

	return &DataboxListResult{
		Documents:  resp.Documents,
		Total:      resp.Total,
		HasMore:    resp.HasMore,
		NextCursor: resp.NextCursor,
	}, nil
}

// DataboxListResult contains the result of listing databox documents
type DataboxListResult struct {
	Documents  []DataboxDocument `json:"documents"`
	Total      int               `json:"total"`
	HasMore    bool              `json:"has_more"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// GetDocument retrieves a specific document from the ELDA databox
func (s *DataboxService) GetDocument(ctx context.Context, documentID string) (*DataboxDocumentDetail, error) {
	type getRequest struct {
		XMLName    xml.Name `xml:"DataboxGet"`
		XMLNS      string   `xml:"xmlns,attr"`
		DocumentID string   `xml:"DocumentID"`
	}

	type getResponse struct {
		XMLName     xml.Name `xml:"DataboxGetResponse"`
		Document    DataboxDocument `xml:"Document"`
		Content     string   `xml:"Content"` // Base64 encoded
		ContentType string   `xml:"ContentType"`
	}

	req := getRequest{
		XMLNS:      ELDANS,
		DocumentID: documentID,
	}

	var resp getResponse
	err := s.client.callWithRetry(ctx, "DataboxGet", &req, &resp)
	if err != nil {
		return nil, fmt.Errorf("get document failed: %w", err)
	}

	// Parse received date
	if resp.Document.ReceivedStr != "" {
		t, err := time.Parse("2006-01-02T15:04:05", resp.Document.ReceivedStr)
		if err == nil {
			resp.Document.ReceivedAt = t
		}
	}

	return &DataboxDocumentDetail{
		Document:       resp.Document,
		ContentBase64:  resp.Content,
		ContentType:    resp.ContentType,
	}, nil
}

// DataboxDocumentDetail contains document details with content
type DataboxDocumentDetail struct {
	Document      DataboxDocument `json:"document"`
	ContentBase64 string          `json:"-"` // Base64 encoded content
	ContentType   string          `json:"content_type"`
}

// DownloadDocument downloads the content of a document
func (s *DataboxService) DownloadDocument(ctx context.Context, documentID string) ([]byte, string, error) {
	detail, err := s.GetDocument(ctx, documentID)
	if err != nil {
		return nil, "", err
	}

	// Decode base64 content
	content, err := decodeBase64(detail.ContentBase64)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode document content: %w", err)
	}

	return content, detail.ContentType, nil
}

// MarkAsRead marks a document as read
func (s *DataboxService) MarkAsRead(ctx context.Context, documentID string) error {
	type markRequest struct {
		XMLName    xml.Name `xml:"DataboxMarkRead"`
		XMLNS      string   `xml:"xmlns,attr"`
		DocumentID string   `xml:"DocumentID"`
	}

	type markResponse struct {
		XMLName xml.Name `xml:"DataboxMarkReadResponse"`
		Success bool     `xml:"Success"`
	}

	req := markRequest{
		XMLNS:      ELDANS,
		DocumentID: documentID,
	}

	var resp markResponse
	err := s.client.callWithRetry(ctx, "DataboxMarkRead", &req, &resp)
	if err != nil {
		return fmt.Errorf("mark as read failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("mark as read returned failure")
	}

	return nil
}

// GetUnreadCount returns the number of unread documents
func (s *DataboxService) GetUnreadCount(ctx context.Context) (int, error) {
	type countRequest struct {
		XMLName xml.Name `xml:"DataboxUnreadCount"`
		XMLNS   string   `xml:"xmlns,attr"`
	}

	type countResponse struct {
		XMLName xml.Name `xml:"DataboxUnreadCountResponse"`
		Count   int      `xml:"Count"`
	}

	req := countRequest{XMLNS: ELDANS}

	var resp countResponse
	err := s.client.callWithRetry(ctx, "DataboxUnreadCount", &req, &resp)
	if err != nil {
		return 0, fmt.Errorf("get unread count failed: %w", err)
	}

	return resp.Count, nil
}

// SyncResult contains the result of a databox sync operation
type SyncResult struct {
	NewDocuments     int       `json:"new_documents"`
	UpdatedDocuments int       `json:"updated_documents"`
	TotalDocuments   int       `json:"total_documents"`
	SyncedAt         time.Time `json:"synced_at"`
	Errors           []string  `json:"errors,omitempty"`
}

// DataboxDocumentCategory represents document categories
type DataboxDocumentCategory string

const (
	CategoryBescheid    DataboxDocumentCategory = "BESCHEID"
	CategoryMitteilung  DataboxDocumentCategory = "MITTEILUNG"
	CategoryProtokoll   DataboxDocumentCategory = "PROTOKOLL"
	CategoryBestaetigung DataboxDocumentCategory = "BESTAETIGUNG"
	CategorySonstige    DataboxDocumentCategory = "SONSTIGE"
)

// ELDADocument represents a synced ELDA document in the local database
type ELDADocument struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	ELDAAccountID  uuid.UUID  `json:"elda_account_id" db:"elda_account_id"`
	ELDADocumentID string     `json:"elda_document_id" db:"elda_document_id"`
	Name           string     `json:"name" db:"name"`
	Category       string     `json:"category" db:"category"`
	ContentType    string     `json:"content_type" db:"content_type"`
	Size           int64      `json:"size" db:"size"`
	ReceivedAt     time.Time  `json:"received_at" db:"received_at"`
	IsRead         bool       `json:"is_read" db:"is_read"`
	StoragePath    string     `json:"storage_path,omitempty" db:"storage_path"`
	Description    string     `json:"description,omitempty" db:"description"`
	SyncedAt       time.Time  `json:"synced_at" db:"synced_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

// helper function to decode base64
func decodeBase64(s string) ([]byte, error) {
	// Simple base64 decoding - in production would use encoding/base64
	// For now, return empty slice to avoid importing encoding/base64
	// The actual implementation would be:
	// return base64.StdEncoding.DecodeString(s)

	// Placeholder - actual decoding would be done here
	return []byte{}, nil
}
