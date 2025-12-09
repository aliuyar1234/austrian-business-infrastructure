package sync

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/document"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/google/uuid"
)

// DataboxFetcher fetches documents from FinanzOnline databox
type DataboxFetcher struct {
	client *fonws.Client
}

// NewDataboxFetcher creates a new databox fetcher
func NewDataboxFetcher(client *fonws.Client) *DataboxFetcher {
	return &DataboxFetcher{client: client}
}

// FetchedDocument represents a document fetched from databox
type FetchedDocument struct {
	ExternalID  string
	Type        string
	Title       string
	Sender      string
	ReceivedAt  time.Time
	Content     []byte
	ContentType string
	Metadata    map[string]interface{}
}

// Fetch retrieves all documents from a FinanzOnline databox
func (f *DataboxFetcher) Fetch(ctx context.Context, session *fonws.Session, fromDate, toDate string) ([]*FetchedDocument, error) {
	// Get databox info
	databoxSvc := fonws.NewDataboxService(f.client)
	entries, err := databoxSvc.GetInfo(session, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get databox info: %w", err)
	}

	var documents []*FetchedDocument

	for _, entry := range entries {
		// Download document content
		content, filename, err := databoxSvc.DownloadToBytes(session, entry.Applkey)
		if err != nil {
			// Log error but continue with other documents
			continue
		}

		// Parse received date
		receivedAt := parseDataboxDate(entry.TsZust)

		// Determine document type
		docType := mapDataboxType(entry.Erlession)

		doc := &FetchedDocument{
			ExternalID:  entry.Applkey,
			Type:        docType,
			Title:       entry.Filebez,
			Sender:      "FinanzOnline",
			ReceivedAt:  receivedAt,
			Content:     content,
			ContentType: detectContentType(filename),
			Metadata: map[string]interface{}{
				"applkey":   entry.Applkey,
				"erlession": entry.Erlession,
				"veression": entry.Veression,
				"filename":  filename,
			},
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// FetchSingle retrieves a single document by applkey
func (f *DataboxFetcher) FetchSingle(ctx context.Context, session *fonws.Session, applkey string) (*FetchedDocument, error) {
	databoxSvc := fonws.NewDataboxService(f.client)

	content, filename, err := databoxSvc.DownloadToBytes(session, applkey)
	if err != nil {
		return nil, fmt.Errorf("download document: %w", err)
	}

	return &FetchedDocument{
		ExternalID:  applkey,
		Type:        document.TypeSonstige,
		Title:       filename,
		Sender:      "FinanzOnline",
		ReceivedAt:  time.Now(),
		Content:     content,
		ContentType: detectContentType(filename),
		Metadata: map[string]interface{}{
			"applkey":  applkey,
			"filename": filename,
		},
	}, nil
}

// SyncResult holds the result of a sync operation
type SyncResult struct {
	Found   int
	New     int
	Skipped int
	Errors  []string
}

// Syncer handles databox synchronization
type Syncer struct {
	fetcher       *DataboxFetcher
	docService    *document.Service
	docRepo       *document.Repository
	onNewDocument func(ctx context.Context, tenantID uuid.UUID, doc *document.Document)
}

// NewSyncer creates a new syncer
func NewSyncer(client *fonws.Client, docService *document.Service, docRepo *document.Repository) *Syncer {
	return &Syncer{
		fetcher:    NewDataboxFetcher(client),
		docService: docService,
		docRepo:    docRepo,
	}
}

// SetNewDocumentCallback sets the callback for new documents
func (s *Syncer) SetNewDocumentCallback(fn func(ctx context.Context, tenantID uuid.UUID, doc *document.Document)) {
	s.onNewDocument = fn
}

// SyncAccount synchronizes documents for a single account
func (s *Syncer) SyncAccount(ctx context.Context, session *fonws.Session, accountID uuid.UUID, tenantID string, fromDate, toDate string) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch documents from databox
	fetched, err := s.fetcher.Fetch(ctx, session, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("fetch documents: %w", err)
	}

	result.Found = len(fetched)

	// Process each document
	for _, doc := range fetched {
		// Check if document already exists
		existing, err := s.docRepo.GetByExternalID(ctx, accountID, doc.ExternalID)
		if err == nil && existing != nil {
			result.Skipped++
			continue
		}

		// Create document
		input := &document.CreateDocumentInput{
			AccountID:   accountID,
			ExternalID:  doc.ExternalID,
			Type:        doc.Type,
			Title:       doc.Title,
			Sender:      doc.Sender,
			ReceivedAt:  doc.ReceivedAt,
			Content:     bytes.NewReader(doc.Content),
			ContentType: doc.ContentType,
			Metadata:    doc.Metadata,
		}

		newDoc, err := s.docService.Create(ctx, tenantID, input)
		if err != nil {
			if err == document.ErrDuplicateDocument {
				result.Skipped++
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("create document %s: %v", doc.ExternalID, err))
			}
			continue
		}

		result.New++

		// Trigger new document callback (for analysis, notifications, etc.)
		if s.onNewDocument != nil && newDoc != nil {
			tenantUUID, _ := uuid.Parse(tenantID)
			s.onNewDocument(ctx, tenantUUID, newDoc)
		}
	}

	return result, nil
}

// Helper functions

// parseDataboxDate parses FinanzOnline date format (YYYY-MM-DD HH:MM:SS)
func parseDataboxDate(dateStr string) time.Time {
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"02.01.2006 15:04:05",
		"02.01.2006",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, dateStr)
		if err == nil {
			return t
		}
	}

	return time.Now()
}

// mapDataboxType maps FinanzOnline erlession codes to document types
func mapDataboxType(erlession string) string {
	switch erlession {
	case "B":
		return document.TypeBescheid
	case "E", "V":
		return document.TypeErsuchen
	case "M":
		return document.TypeMitteilung
	case "Z":
		return document.TypeMahnung
	default:
		return document.TypeSonstige
	}
}

// detectContentType determines MIME type from filename
func detectContentType(filename string) string {
	ext := ""
	for i := len(filename) - 1; i >= 0 && filename[i] != '.'; i-- {
		ext = string(filename[i]) + ext
	}

	switch ext {
	case "pdf":
		return "application/pdf"
	case "xml":
		return "application/xml"
	case "html", "htm":
		return "text/html"
	case "txt":
		return "text/plain"
	default:
		return "application/pdf" // Default for FO documents
	}
}
