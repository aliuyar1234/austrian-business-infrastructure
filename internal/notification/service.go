package notification

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"time"

	"austrian-business-infrastructure/internal/document"
	"austrian-business-infrastructure/internal/email"
	"github.com/google/uuid"
)

// Service handles notification business logic
type Service struct {
	repo       *Repository
	docRepo    *document.Repository
	emailSvc   email.Service
	logger     *slog.Logger
	appURL     string
	templates  *Templates
}

// Templates holds email templates
type Templates struct {
	NewDocument *template.Template
	Digest      *template.Template
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Logger *slog.Logger
	AppURL string
}

// NewService creates a new notification service
func NewService(repo *Repository, docRepo *document.Repository, emailSvc email.Service, cfg *ServiceConfig) *Service {
	logger := slog.Default()
	appURL := "http://localhost:3000"
	if cfg != nil {
		if cfg.Logger != nil {
			logger = cfg.Logger
		}
		if cfg.AppURL != "" {
			appURL = cfg.AppURL
		}
	}

	return &Service{
		repo:      repo,
		docRepo:   docRepo,
		emailSvc:  emailSvc,
		logger:    logger,
		appURL:    appURL,
		templates: loadTemplates(),
	}
}

// loadTemplates loads email templates
func loadTemplates() *Templates {
	newDocTmpl := template.Must(template.New("new_document").Parse(newDocumentTemplate))
	digestTmpl := template.Must(template.New("digest").Parse(digestTemplate))

	return &Templates{
		NewDocument: newDocTmpl,
		Digest:      digestTmpl,
	}
}

// GetPreferences retrieves notification preferences for a user
func (s *Service) GetPreferences(ctx context.Context, userID, tenantID uuid.UUID) (*NotificationPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID, tenantID)
	if err == ErrPreferencesNotFound {
		// Return default preferences
		return &NotificationPreferences{
			UserID:       userID,
			TenantID:     tenantID,
			EmailEnabled: false,
			EmailMode:    ModeOff,
			DigestTime:   "08:00",
		}, nil
	}
	return prefs, err
}

// UpdatePreferences updates notification preferences
func (s *Service) UpdatePreferences(ctx context.Context, prefs *NotificationPreferences) error {
	return s.repo.UpsertPreferences(ctx, prefs)
}

// NotifyNewDocument queues notification for a new document
func (s *Service) NotifyNewDocument(ctx context.Context, tenantID uuid.UUID, doc *document.Document, userEmail string) error {
	// Get user preferences - we need user ID from the context or document
	// For now, we'll queue with document info and let the processor handle user lookup

	// Check if document type should trigger notification
	// Queue immediate or digest based on preferences
	item := &NotificationQueueItem{
		TenantID:   tenantID,
		DocumentID: doc.ID,
		Type:       "new_document",
		Status:     "pending",
	}

	return s.repo.QueueNotification(ctx, item)
}

// NotifyUsersAboutDocument notifies all users in a tenant about a new document
func (s *Service) NotifyUsersAboutDocument(ctx context.Context, tenantID uuid.UUID, doc *document.Document) error {
	// This would typically look up all users in the tenant with notification preferences
	// and queue notifications for each
	s.logger.Info("queuing notifications for new document",
		"tenant_id", tenantID,
		"document_id", doc.ID,
		"type", doc.Type)

	return nil
}

// ShouldNotify checks if a user should be notified about a document
func (s *Service) ShouldNotify(prefs *NotificationPreferences, doc *document.Document) bool {
	if !prefs.EmailEnabled {
		return false
	}

	// Check document type filter
	if len(prefs.DocumentTypes) > 0 {
		found := false
		for _, t := range prefs.DocumentTypes {
			if t == doc.Type {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check account filter
	if len(prefs.AccountIDs) > 0 {
		found := false
		for _, id := range prefs.AccountIDs {
			if id == doc.AccountID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// ProcessPendingNotifications processes pending notifications in the queue
func (s *Service) ProcessPendingNotifications(ctx context.Context, batchSize int) error {
	items, err := s.repo.GetPendingNotifications(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("get pending: %w", err)
	}

	for _, item := range items {
		if err := s.sendNotification(ctx, item); err != nil {
			s.logger.Error("failed to send notification",
				"id", item.ID,
				"error", err)
			s.repo.MarkNotificationFailed(ctx, item.ID, err.Error())
		} else {
			s.repo.MarkNotificationSent(ctx, item.ID)
		}
	}

	return nil
}

// sendNotification sends a single notification
func (s *Service) sendNotification(ctx context.Context, item *NotificationQueueItem) error {
	// Get document details with tenant isolation
	doc, err := s.docRepo.GetByID(ctx, item.TenantID, item.DocumentID)
	if err != nil {
		return fmt.Errorf("get document: %w", err)
	}

	// Build email content
	data := NewDocumentEmailData{
		DocumentTitle: doc.Title,
		DocumentType:  doc.Type,
		Sender:        doc.Sender,
		ReceivedAt:    doc.ReceivedAt.Format("02.01.2006 15:04"),
		DocumentURL:   fmt.Sprintf("%s/documents/%s", s.appURL, doc.ID),
		Priority:      document.TypePriority(doc.Type),
	}

	var buf bytes.Buffer
	if err := s.templates.NewDocument.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	// TODO: Get user email from user service
	// For now, this is a placeholder
	// return s.emailSvc.SendNewDocument(ctx, userEmail, doc.Title, buf.String())

	return nil
}

// SendDigest sends a digest email to a user
func (s *Service) SendDigest(ctx context.Context, userID, tenantID uuid.UUID, userEmail string) error {
	// Get documents from last 24 hours
	since := time.Now().Add(-24 * time.Hour)
	items, err := s.repo.GetDigestItems(ctx, userID, tenantID, since)
	if err != nil {
		return fmt.Errorf("get digest items: %w", err)
	}

	if len(items) == 0 {
		return nil // No new documents, skip digest
	}

	// Get document details for each item with tenant isolation
	var docs []DigestDocumentData
	for _, item := range items {
		doc, err := s.docRepo.GetByID(ctx, tenantID, item.DocumentID)
		if err != nil {
			continue // Skip documents that can't be retrieved
		}

		docs = append(docs, DigestDocumentData{
			Title:      doc.Title,
			Type:       doc.Type,
			Sender:     doc.Sender,
			ReceivedAt: doc.ReceivedAt.Format("02.01.2006 15:04"),
			URL:        fmt.Sprintf("%s/documents/%s", s.appURL, doc.ID),
		})
	}

	if len(docs) == 0 {
		return nil
	}

	data := DigestEmailData{
		DocumentCount: len(docs),
		Documents:     docs,
		DashboardURL:  fmt.Sprintf("%s/documents", s.appURL),
	}

	var buf bytes.Buffer
	if err := s.templates.Digest.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute digest template: %w", err)
	}

	// TODO: Send email
	// return s.emailSvc.SendDigest(ctx, userEmail, buf.String())

	return nil
}

// Email template data structures

// NewDocumentEmailData holds data for new document email template
type NewDocumentEmailData struct {
	DocumentTitle string
	DocumentType  string
	Sender        string
	ReceivedAt    string
	DocumentURL   string
	Priority      int
}

// DigestEmailData holds data for digest email template
type DigestEmailData struct {
	DocumentCount int
	Documents     []DigestDocumentData
	DashboardURL  string
}

// DigestDocumentData holds document data for digest
type DigestDocumentData struct {
	Title      string
	Type       string
	Sender     string
	ReceivedAt string
	URL        string
}

// Email templates

const newDocumentTemplate = `
Neues Dokument eingegangen

Ein neues Dokument wurde in Ihrer Databox empfangen:

Titel: {{.DocumentTitle}}
Typ: {{.DocumentType}}
Absender: {{.Sender}}
Empfangen: {{.ReceivedAt}}

Dokument ansehen: {{.DocumentURL}}

--
Austrian Business Platform
`

const digestTemplate = `
Tägliche Dokumenten-Zusammenfassung

Sie haben {{.DocumentCount}} neue Dokumente erhalten:

{{range .Documents}}
- {{.Title}} ({{.Type}})
  Von: {{.Sender}}
  Empfangen: {{.ReceivedAt}}
  {{.URL}}

{{end}}

Alle Dokumente ansehen: {{.DashboardURL}}

--
Austrian Business Platform
`
