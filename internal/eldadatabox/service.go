package eldadatabox

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

// StorageService interface for document storage
type StorageService interface {
	Store(ctx context.Context, accountID uuid.UUID, filename string, content []byte, contentType string) (string, error)
	Get(ctx context.Context, path string) ([]byte, error)
}

// Service handles ELDA databox business logic
type Service struct {
	repo     *Repository
	databox  *elda.DataboxService
	storage  StorageService
}

// NewService creates a new ELDA databox service
func NewService(pool *pgxpool.Pool, eldaClient *elda.Client, storage StorageService) *Service {
	return &Service{
		repo:    NewRepository(pool),
		databox: elda.NewDataboxService(eldaClient),
		storage: storage,
	}
}

// Sync synchronizes documents from ELDA databox
func (s *Service) Sync(ctx context.Context, accountID uuid.UUID) (*SyncResult, error) {
	result := &SyncResult{
		SyncedAt: time.Now(),
	}

	// Get last sync time
	lastSync, _ := s.repo.GetLastSyncTime(ctx, accountID)

	// Fetch documents from ELDA
	listReq := &elda.DataboxListRequest{
		Limit:     100,
		StartDate: lastSync,
	}

	eldaResult, err := s.databox.ListDocuments(ctx, listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list ELDA documents: %w", err)
	}

	// Process each document
	for _, eldaDoc := range eldaResult.Documents {
		// Check if document already exists
		existing, err := s.repo.GetByELDADocumentID(ctx, accountID, eldaDoc.ID)
		if err == nil && existing != nil {
			// Update existing
			existing.IsRead = eldaDoc.IsRead
			if err := s.repo.Update(ctx, existing); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("update %s: %s", eldaDoc.ID, err.Error()))
			} else {
				result.UpdatedCount++
			}
			continue
		}

		// Create new document
		doc := &elda.ELDADocument{
			ID:             uuid.New(),
			ELDAAccountID:  accountID,
			ELDADocumentID: eldaDoc.ID,
			Name:           eldaDoc.Name,
			Category:       eldaDoc.Category,
			ContentType:    eldaDoc.ContentType,
			Size:           eldaDoc.Size,
			ReceivedAt:     eldaDoc.ReceivedAt,
			IsRead:         eldaDoc.IsRead,
			Description:    eldaDoc.Description,
		}

		if err := s.repo.Create(ctx, doc); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("create %s: %s", eldaDoc.ID, err.Error()))
		} else {
			result.NewCount++
		}
	}

	result.TotalCount = result.NewCount + result.UpdatedCount

	return result, nil
}

// SyncResult contains the result of a sync operation
type SyncResult struct {
	NewCount     int       `json:"new_count"`
	UpdatedCount int       `json:"updated_count"`
	TotalCount   int       `json:"total_count"`
	SyncedAt     time.Time `json:"synced_at"`
	Errors       []string  `json:"errors,omitempty"`
}

// List retrieves ELDA documents with filters
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*elda.ELDADocument, error) {
	repoFilter := ListFilter{
		ELDAAccountID: filter.ELDAAccountID,
		Category:      filter.Category,
		Unread:        filter.Unread,
		StartDate:     filter.StartDate,
		EndDate:       filter.EndDate,
		Limit:         filter.Limit,
		Offset:        filter.Offset,
	}
	return s.repo.List(ctx, repoFilter)
}

// Get retrieves a document by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*elda.ELDADocument, error) {
	return s.repo.GetByID(ctx, id)
}

// Count returns the count of documents matching the filter
func (s *Service) Count(ctx context.Context, filter ListFilter) (int, error) {
	return s.repo.Count(ctx, filter)
}

// GetUnreadCount returns the count of unread documents
func (s *Service) GetUnreadCount(ctx context.Context, accountID uuid.UUID) (int, error) {
	return s.repo.GetUnreadCount(ctx, accountID)
}

// GetContent downloads and returns the document content
func (s *Service) GetContent(ctx context.Context, id uuid.UUID) ([]byte, string, error) {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, "", err
	}

	// Check if already stored locally
	if doc.StoragePath != "" && s.storage != nil {
		content, err := s.storage.Get(ctx, doc.StoragePath)
		if err == nil {
			return content, doc.ContentType, nil
		}
		// Fall through to download from ELDA if local storage fails
	}

	// Download from ELDA
	content, contentType, err := s.databox.DownloadDocument(ctx, doc.ELDADocumentID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download document: %w", err)
	}

	// Store locally if storage service is available
	if s.storage != nil {
		storagePath, err := s.storage.Store(ctx, doc.ELDAAccountID, doc.Name, content, contentType)
		if err == nil {
			doc.StoragePath = storagePath
			s.repo.SetStoragePath(ctx, id, storagePath)
		}
	}

	return content, contentType, nil
}

// MarkAsRead marks a document as read
func (s *Service) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	doc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Mark as read in ELDA
	if err := s.databox.MarkAsRead(ctx, doc.ELDADocumentID); err != nil {
		return fmt.Errorf("failed to mark as read in ELDA: %w", err)
	}

	// Mark as read locally
	return s.repo.MarkAsRead(ctx, id)
}

// GetSyncStatus returns the sync status for an account
func (s *Service) GetSyncStatus(ctx context.Context, accountID uuid.UUID) (*SyncStatus, error) {
	lastSync, _ := s.repo.GetLastSyncTime(ctx, accountID)
	unreadCount, _ := s.repo.GetUnreadCount(ctx, accountID)

	filter := ListFilter{
		ELDAAccountID: &accountID,
	}
	totalCount, _ := s.repo.Count(ctx, filter)

	return &SyncStatus{
		LastSyncAt:   lastSync,
		UnreadCount:  unreadCount,
		TotalCount:   totalCount,
		AccountID:    accountID,
	}, nil
}

// SyncStatus contains sync status information
type SyncStatus struct {
	AccountID    uuid.UUID  `json:"account_id"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
	UnreadCount  int        `json:"unread_count"`
	TotalCount   int        `json:"total_count"`
}

// Delete deletes a document
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// GetCategories returns available document categories
func (s *Service) GetCategories() []string {
	return []string{
		string(elda.CategoryBescheid),
		string(elda.CategoryMitteilung),
		string(elda.CategoryProtokoll),
		string(elda.CategoryBestaetigung),
		string(elda.CategorySonstige),
	}
}
