package signature

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/atrust"
)

// BatchService provides batch signature operations
type BatchService struct {
	repo      *Repository
	service   *Service
	atrust    atrust.Signer
	documents DocumentStore
	maxConcurrent int
}

// NewBatchService creates a new batch signature service
func NewBatchService(repo *Repository, service *Service, atrustClient atrust.Signer, docStore DocumentStore) *BatchService {
	return &BatchService{
		repo:          repo,
		service:       service,
		atrust:        atrustClient,
		documents:     docStore,
		maxConcurrent: 10, // Max concurrent signing operations
	}
}

// CreateBatchInput contains input for creating a batch
type CreateBatchInput struct {
	TenantID    uuid.UUID
	Name        string
	DocumentIDs []uuid.UUID
	SignerID    uuid.UUID // The user who will sign all documents
}

// CreateBatch creates a new batch signing operation
func (s *BatchService) CreateBatch(ctx context.Context, input *CreateBatchInput) (*Batch, error) {
	if len(input.DocumentIDs) == 0 {
		return nil, fmt.Errorf("no documents provided")
	}

	if len(input.DocumentIDs) > 100 {
		return nil, fmt.Errorf("batch size exceeds maximum of 100 documents")
	}

	batch := &Batch{
		ID:             uuid.New(),
		TenantID:       input.TenantID,
		TotalDocuments: len(input.DocumentIDs),
		Status:         BatchStatusPending,
		SignerUserID:   &input.SignerID,
	}
	if input.Name != "" {
		batch.Name = &input.Name
	}

	// Insert batch
	if err := s.createBatchRecord(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	// Create batch items
	items := make([]*BatchItem, len(input.DocumentIDs))
	for i, docID := range input.DocumentIDs {
		item := &BatchItem{
			ID:         uuid.New(),
			BatchID:    batch.ID,
			DocumentID: docID,
			Status:     BatchItemStatusPending,
		}
		if err := s.createBatchItemRecord(ctx, item); err != nil {
			return nil, fmt.Errorf("failed to create batch item: %w", err)
		}
		items[i] = item
	}
	batch.Items = items

	// Audit event
	s.service.createAuditEvent(ctx, input.TenantID, nil, nil, &batch.ID, nil, AuditEventBatchStarted,
		map[string]interface{}{
			"document_count": len(input.DocumentIDs),
		}, "user", input.SignerID.String(), "", "")

	return batch, nil
}

// GetBatch retrieves a batch by ID
func (s *BatchService) GetBatch(ctx context.Context, batchID uuid.UUID) (*Batch, error) {
	return s.getBatchWithItems(ctx, batchID)
}

// StartBatchSigning initiates the ID Austria auth flow for batch signing
func (s *BatchService) StartBatchSigning(ctx context.Context, batchID uuid.UUID) (string, error) {
	batch, err := s.getBatchWithItems(ctx, batchID)
	if err != nil {
		return "", err
	}

	if batch.Status != BatchStatusPending {
		return "", fmt.Errorf("batch is not in pending state")
	}

	// Update batch status
	if err := s.updateBatchStatus(ctx, batchID, BatchStatusSigning); err != nil {
		return "", err
	}

	// Create ID Austria auth request
	authReq, err := s.service.idaustria.CreateAuthorizationRequest(
		fmt.Sprintf("%s/batch/%s/complete", s.service.config.SigningCallbackURL, batchID.String()),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}

	// Store session with batch context
	if err := s.repo.SaveIDAustriaSession(ctx, authReq.State, authReq.Nonce,
		authReq.CodeVerifier, authReq.RedirectAfter, nil, &batchID); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	// Get auth URL
	authURL, err := s.service.idaustria.AuthorizationURL(ctx, authReq)
	if err != nil {
		return "", fmt.Errorf("failed to generate auth URL: %w", err)
	}

	return authURL, nil
}

// CompleteBatchSigningInput contains input for completing batch signing
type CompleteBatchSigningInput struct {
	BatchID   uuid.UUID
	State     string
	Code      string
	Error     string
	ErrorDesc string
	IP        string
	UserAgent string
}

// CompleteBatchSigning completes the batch signing after ID Austria callback
func (s *BatchService) CompleteBatchSigning(ctx context.Context, input *CompleteBatchSigningInput) (*Batch, error) {
	// Get session
	nonce, codeVerifier, _, batchID, redirectAfter, err := s.repo.GetIDAustriaSessionByState(ctx, input.State)
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	// Validate callback
	if input.Error != "" {
		s.updateBatchStatus(ctx, *batchID, BatchStatusCancelled)
		return nil, fmt.Errorf("ID Austria error: %s - %s", input.Error, input.ErrorDesc)
	}

	if batchID == nil || *batchID != input.BatchID {
		return nil, fmt.Errorf("session mismatch")
	}

	// Exchange code for tokens
	token, err := s.service.idaustria.ExchangeCode(ctx, input.Code, codeVerifier)
	if err != nil {
		s.updateBatchStatus(ctx, *batchID, BatchStatusCancelled)
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	// Get user info
	userInfo, err := s.service.idaustria.GetUserInfo(ctx, token.AccessToken)
	if err != nil {
		s.updateBatchStatus(ctx, *batchID, BatchStatusCancelled)
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	// Update session as authenticated
	if err := s.repo.UpdateIDAustriaSessionAuthenticated(ctx, input.State, userInfo.Subject, userInfo.Name, ""); err != nil {
		// Non-fatal
	}

	_ = nonce
	_ = redirectAfter

	// Get batch items
	batch, err := s.getBatchWithItems(ctx, *batchID)
	if err != nil {
		return nil, err
	}

	// Process documents in parallel
	results := s.processDocumentsParallel(ctx, batch, userInfo.Subject)

	// Update batch counts (triggers are set up, but let's be explicit)
	signed := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			signed++
		} else {
			failed++
		}
	}

	// Determine final status
	var finalStatus BatchStatus
	if failed == 0 {
		finalStatus = BatchStatusCompleted
	} else if signed == 0 {
		finalStatus = BatchStatusPartialFailure
	} else {
		finalStatus = BatchStatusPartialFailure
	}

	s.updateBatchCompleted(ctx, *batchID, finalStatus, signed, failed)

	// Record usage
	if signed > 0 {
		usage := &Usage{
			TenantID:       batch.TenantID,
			BatchID:        batchID,
			SignatureCount: signed,
			UsageDate:      time.Now(),
		}
		if s.service.config.SignatureCostCents > 0 {
			cost := s.service.config.SignatureCostCents * signed
			usage.CostCents = &cost
		}
		s.repo.RecordUsage(ctx, usage)
	}

	// Audit
	s.service.createAuditEvent(ctx, batch.TenantID, nil, nil, batchID, nil, AuditEventBatchCompleted,
		map[string]interface{}{
			"signed_count": signed,
			"failed_count": failed,
		}, "user", "", input.IP, input.UserAgent)

	// Refresh batch data
	return s.getBatchWithItems(ctx, *batchID)
}

// BatchSignResult represents the result of signing one document
type BatchSignResult struct {
	ItemID    uuid.UUID
	DocumentID uuid.UUID
	Success   bool
	Error     string
}

// processDocumentsParallel signs multiple documents in parallel
func (s *BatchService) processDocumentsParallel(ctx context.Context, batch *Batch, signerCertID string) []BatchSignResult {
	results := make([]BatchSignResult, len(batch.Items))
	var wg sync.WaitGroup

	// Use semaphore to limit concurrency
	sem := make(chan struct{}, s.maxConcurrent)

	for i, item := range batch.Items {
		wg.Add(1)
		go func(index int, item *BatchItem) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result := s.signSingleDocument(ctx, item, signerCertID)
			results[index] = result
		}(i, item)
	}

	wg.Wait()
	return results
}

// signSingleDocument signs a single document in the batch
func (s *BatchService) signSingleDocument(ctx context.Context, item *BatchItem, signerCertID string) BatchSignResult {
	result := BatchSignResult{
		ItemID:     item.ID,
		DocumentID: item.DocumentID,
	}

	// Update item status to signing
	s.updateBatchItemStatus(ctx, item.ID, BatchItemStatusSigning, "")

	// Get document content
	docContent, err := s.documents.GetDocumentContent(ctx, item.DocumentID)
	if err != nil {
		result.Error = fmt.Sprintf("failed to get document: %v", err)
		s.updateBatchItemStatus(ctx, item.ID, BatchItemStatusFailed, result.Error)
		return result
	}

	// Calculate hash
	docHash := atrust.HashDocument(docContent)

	// Sign with A-Trust
	signReq := &atrust.SignRequest{
		DocumentHash:  docHash,
		HashAlgorithm: atrust.HashAlgoSHA256,
		SignerCertID:  signerCertID,
	}

	_, err = s.atrust.Sign(ctx, signReq)
	if err != nil {
		result.Error = fmt.Sprintf("failed to sign: %v", err)
		s.updateBatchItemStatus(ctx, item.ID, BatchItemStatusFailed, result.Error)
		return result
	}

	// TODO: Embed signature into PDF
	// For now, mark as signed successfully

	// Update item as signed
	s.updateBatchItemSigned(ctx, item.ID, nil) // signedDocID would be passed here

	result.Success = true
	return result
}

// RetryFailedItems retries failed items in a batch
func (s *BatchService) RetryFailedItems(ctx context.Context, batchID uuid.UUID) (*Batch, error) {
	batch, err := s.getBatchWithItems(ctx, batchID)
	if err != nil {
		return nil, err
	}

	// Count failed items
	failedCount := 0
	for _, item := range batch.Items {
		if item.Status == BatchItemStatusFailed {
			failedCount++
		}
	}

	if failedCount == 0 {
		return batch, nil
	}

	// Reset failed items to pending
	for _, item := range batch.Items {
		if item.Status == BatchItemStatusFailed {
			s.updateBatchItemStatus(ctx, item.ID, BatchItemStatusPending, "")
		}
	}

	// Reset batch status
	s.updateBatchStatus(ctx, batchID, BatchStatusPending)

	return s.getBatchWithItems(ctx, batchID)
}

// CancelBatch cancels a pending batch
func (s *BatchService) CancelBatch(ctx context.Context, batchID uuid.UUID) error {
	batch, err := s.getBatchWithItems(ctx, batchID)
	if err != nil {
		return err
	}

	if batch.Status != BatchStatusPending {
		return fmt.Errorf("batch is not in pending state")
	}

	return s.updateBatchStatus(ctx, batchID, BatchStatusCancelled)
}

// ListBatches lists batches for a tenant
func (s *BatchService) ListBatches(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Batch, int, error) {
	return s.listBatchesByTenant(ctx, tenantID, limit, offset)
}

// ===== Database Helper Methods =====

func (s *BatchService) createBatchRecord(ctx context.Context, batch *Batch) error {
	query := `
		INSERT INTO signature_batches (id, tenant_id, name, total_documents, signer_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING status, created_at
	`
	return s.repo.pool.QueryRow(ctx, query,
		batch.ID, batch.TenantID, batch.Name, batch.TotalDocuments, batch.SignerUserID,
	).Scan(&batch.Status, &batch.CreatedAt)
}

func (s *BatchService) createBatchItemRecord(ctx context.Context, item *BatchItem) error {
	query := `
		INSERT INTO signature_batch_items (id, batch_id, document_id)
		VALUES ($1, $2, $3)
		RETURNING status, created_at
	`
	return s.repo.pool.QueryRow(ctx, query,
		item.ID, item.BatchID, item.DocumentID,
	).Scan(&item.Status, &item.CreatedAt)
}

func (s *BatchService) getBatchWithItems(ctx context.Context, batchID uuid.UUID) (*Batch, error) {
	// Get batch
	batchQuery := `
		SELECT id, tenant_id, name, total_documents, status, started_at, completed_at,
			signed_count, failed_count, signer_user_id, idaustria_session_id, created_at
		FROM signature_batches WHERE id = $1
	`
	batch := &Batch{}
	err := s.repo.pool.QueryRow(ctx, batchQuery, batchID).Scan(
		&batch.ID, &batch.TenantID, &batch.Name, &batch.TotalDocuments, &batch.Status,
		&batch.StartedAt, &batch.CompletedAt, &batch.SignedCount, &batch.FailedCount,
		&batch.SignerUserID, &batch.IDAustriaSessionID, &batch.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("batch not found: %w", err)
	}

	// Get items
	itemsQuery := `
		SELECT bi.id, bi.batch_id, bi.document_id, bi.status, bi.signed_at,
			bi.error_message, bi.signed_document_id, bi.created_at,
			d.title as document_title
		FROM signature_batch_items bi
		LEFT JOIN documents d ON bi.document_id = d.id
		WHERE bi.batch_id = $1
		ORDER BY bi.created_at ASC
	`
	rows, err := s.repo.pool.Query(ctx, itemsQuery, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*BatchItem
	for rows.Next() {
		item := &BatchItem{}
		err := rows.Scan(
			&item.ID, &item.BatchID, &item.DocumentID, &item.Status, &item.SignedAt,
			&item.ErrorMessage, &item.SignedDocumentID, &item.CreatedAt,
			&item.DocumentTitle,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	batch.Items = items

	return batch, rows.Err()
}

func (s *BatchService) updateBatchStatus(ctx context.Context, batchID uuid.UUID, status BatchStatus) error {
	query := `UPDATE signature_batches SET status = $2, started_at = COALESCE(started_at, NOW()) WHERE id = $1`
	_, err := s.repo.pool.Exec(ctx, query, batchID, status)
	return err
}

func (s *BatchService) updateBatchCompleted(ctx context.Context, batchID uuid.UUID, status BatchStatus, signedCount, failedCount int) error {
	query := `
		UPDATE signature_batches
		SET status = $2, completed_at = NOW(), signed_count = $3, failed_count = $4
		WHERE id = $1
	`
	_, err := s.repo.pool.Exec(ctx, query, batchID, status, signedCount, failedCount)
	return err
}

func (s *BatchService) updateBatchItemStatus(ctx context.Context, itemID uuid.UUID, status BatchItemStatus, errorMsg string) error {
	query := `UPDATE signature_batch_items SET status = $2, error_message = $3 WHERE id = $1`
	_, err := s.repo.pool.Exec(ctx, query, itemID, status, errorMsg)
	return err
}

func (s *BatchService) updateBatchItemSigned(ctx context.Context, itemID uuid.UUID, signedDocID *uuid.UUID) error {
	query := `UPDATE signature_batch_items SET status = 'signed', signed_at = NOW(), signed_document_id = $2 WHERE id = $1`
	_, err := s.repo.pool.Exec(ctx, query, itemID, signedDocID)
	return err
}

func (s *BatchService) listBatchesByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Batch, int, error) {
	countQuery := `SELECT COUNT(*) FROM signature_batches WHERE tenant_id = $1`
	var total int
	if err := s.repo.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, err
	}

	listQuery := `
		SELECT id, tenant_id, name, total_documents, status, started_at, completed_at,
			signed_count, failed_count, signer_user_id, created_at
		FROM signature_batches
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.repo.pool.Query(ctx, listQuery, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var batches []*Batch
	for rows.Next() {
		batch := &Batch{}
		err := rows.Scan(
			&batch.ID, &batch.TenantID, &batch.Name, &batch.TotalDocuments, &batch.Status,
			&batch.StartedAt, &batch.CompletedAt, &batch.SignedCount, &batch.FailedCount,
			&batch.SignerUserID, &batch.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		batches = append(batches, batch)
	}

	return batches, total, rows.Err()
}

// ===== Batch Handler Extensions =====

// BatchHandler provides HTTP handlers for batch signature operations
type BatchHandler struct {
	service *BatchService
}

// NewBatchHandler creates a new batch handler
func NewBatchHandler(service *BatchService) *BatchHandler {
	return &BatchHandler{service: service}
}

// CreateBatchPayload is the request body for creating a batch
type CreateBatchPayload struct {
	Name        string   `json:"name,omitempty"`
	DocumentIDs []string `json:"document_ids"`
}

// BatchResponse is the response for a batch
type BatchResponse struct {
	ID             string              `json:"id"`
	Name           string              `json:"name,omitempty"`
	TotalDocuments int                 `json:"total_documents"`
	Status         string              `json:"status"`
	StartedAt      *time.Time          `json:"started_at,omitempty"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
	SignedCount    int                 `json:"signed_count"`
	FailedCount    int                 `json:"failed_count"`
	Items          []BatchItemResponse `json:"items,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
}

// BatchItemResponse is a batch item in a response
type BatchItemResponse struct {
	ID               string     `json:"id"`
	DocumentID       string     `json:"document_id"`
	DocumentTitle    string     `json:"document_title,omitempty"`
	Status           string     `json:"status"`
	SignedAt         *time.Time `json:"signed_at,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	SignedDocumentID string     `json:"signed_document_id,omitempty"`
}

func toBatchResponse(batch *Batch) *BatchResponse {
	resp := &BatchResponse{
		ID:             batch.ID.String(),
		TotalDocuments: batch.TotalDocuments,
		Status:         string(batch.Status),
		StartedAt:      batch.StartedAt,
		CompletedAt:    batch.CompletedAt,
		SignedCount:    batch.SignedCount,
		FailedCount:    batch.FailedCount,
		CreatedAt:      batch.CreatedAt,
	}
	if batch.Name != nil {
		resp.Name = *batch.Name
	}

	if len(batch.Items) > 0 {
		resp.Items = make([]BatchItemResponse, len(batch.Items))
		for i, item := range batch.Items {
			itemResp := BatchItemResponse{
				ID:            item.ID.String(),
				DocumentID:    item.DocumentID.String(),
				DocumentTitle: item.DocumentTitle,
				Status:        string(item.Status),
				SignedAt:      item.SignedAt,
			}
			if item.ErrorMessage != nil {
				itemResp.ErrorMessage = *item.ErrorMessage
			}
			if item.SignedDocumentID != nil {
				itemResp.SignedDocumentID = item.SignedDocumentID.String()
			}
			resp.Items[i] = itemResp
		}
	}

	return resp
}

// BatchAuditDetails for JSON marshaling
type BatchAuditDetails struct {
	DocumentCount int `json:"document_count,omitempty"`
	SignedCount   int `json:"signed_count,omitempty"`
	FailedCount   int `json:"failed_count,omitempty"`
}

func (d BatchAuditDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DocumentCount int `json:"document_count,omitempty"`
		SignedCount   int `json:"signed_count,omitempty"`
		FailedCount   int `json:"failed_count,omitempty"`
	}{
		DocumentCount: d.DocumentCount,
		SignedCount:   d.SignedCount,
		FailedCount:   d.FailedCount,
	})
}
