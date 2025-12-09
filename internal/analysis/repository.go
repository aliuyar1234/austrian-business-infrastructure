package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository errors
var (
	ErrAnalysisNotFound   = errors.New("analysis not found")
	ErrDeadlineNotFound   = errors.New("deadline not found")
	ErrAmountNotFound     = errors.New("amount not found")
	ErrActionItemNotFound = errors.New("action item not found")
	ErrTemplateNotFound   = errors.New("template not found")
)

// Analysis represents a document analysis record
type Analysis struct {
	ID                      uuid.UUID              `json:"id"`
	DocumentID              uuid.UUID              `json:"document_id"`
	TenantID                uuid.UUID              `json:"tenant_id"`
	Status                  string                 `json:"status"`
	DocumentType            string                 `json:"document_type,omitempty"`
	DocumentSubtype         string                 `json:"document_subtype,omitempty"`
	ClassificationConfidence float64               `json:"classification_confidence,omitempty"`
	IsScanned               bool                   `json:"is_scanned"`
	OCRProvider             string                 `json:"ocr_provider,omitempty"`
	OCRConfidence           float64                `json:"ocr_confidence,omitempty"`
	Summary                 string                 `json:"summary,omitempty"`
	KeyPoints               []string               `json:"key_points,omitempty"`
	ExtractedText           string                 `json:"extracted_text,omitempty"`
	TextLength              int                    `json:"text_length"`
	PageCount               int                    `json:"page_count"`
	Language                string                 `json:"language,omitempty"`
	AIModel                 string                 `json:"ai_model,omitempty"`
	PromptVersion           string                 `json:"prompt_version,omitempty"`
	TokensUsed              int                    `json:"tokens_used"`
	ProcessingTimeMs        int                    `json:"processing_time_ms"`
	EstimatedCost           float64                `json:"estimated_cost"`
	ErrorMessage            string                 `json:"error_message,omitempty"`
	ErrorCode               string                 `json:"error_code,omitempty"`
	RetryCount              int                    `json:"retry_count"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
	CompletedAt             *time.Time             `json:"completed_at,omitempty"`
}

// Deadline represents an extracted deadline
type Deadline struct {
	ID              uuid.UUID  `json:"id"`
	AnalysisID      uuid.UUID  `json:"analysis_id"`
	DocumentID      uuid.UUID  `json:"document_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	DeadlineType    string     `json:"deadline_type"`
	Date            time.Time  `json:"date"`
	Description     string     `json:"description"`
	SourceText      string     `json:"source_text,omitempty"`
	Confidence      float64    `json:"confidence"`
	IsHard          bool       `json:"is_hard"`
	IsAcknowledged  bool       `json:"is_acknowledged"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
	ManuallySet     bool       `json:"manually_set"`
	CorrectedByUser bool       `json:"corrected_by_user"`
	Notes           string     `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Amount represents an extracted monetary amount
type Amount struct {
	ID              uuid.UUID  `json:"id"`
	AnalysisID      uuid.UUID  `json:"analysis_id"`
	DocumentID      uuid.UUID  `json:"document_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	AmountType      string     `json:"amount_type"`
	Amount          float64    `json:"amount"`
	Currency        string     `json:"currency"`
	Description     string     `json:"description"`
	SourceText      string     `json:"source_text,omitempty"`
	Confidence      float64    `json:"confidence"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	CorrectedByUser bool       `json:"corrected_by_user"`
	Notes           string     `json:"notes,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Priority represents action item priority
type Priority string

// ActionStatus represents action item status
type ActionStatus string

// ActionItem represents a generated action item
type ActionItem struct {
	ID          uuid.UUID    `json:"id"`
	AnalysisID  uuid.UUID    `json:"analysis_id"`
	DocumentID  uuid.UUID    `json:"document_id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Priority    Priority     `json:"priority"`
	Category    string       `json:"category,omitempty"`
	Status      ActionStatus `json:"status"`
	DueDate     *time.Time   `json:"due_date,omitempty"`
	AssignedTo  *string      `json:"assigned_to,omitempty"`
	SourceText  string       `json:"source_text,omitempty"`
	Confidence  float64      `json:"confidence"`
	Notes       string       `json:"notes,omitempty"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// Suggestion represents a response suggestion
type Suggestion struct {
	ID             uuid.UUID `json:"id"`
	AnalysisID     uuid.UUID `json:"analysis_id"`
	DocumentID     uuid.UUID `json:"document_id"`
	TenantID       uuid.UUID `json:"tenant_id"`
	SuggestionType string    `json:"suggestion_type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Reasoning      string    `json:"reasoning,omitempty"`
	Confidence     float64   `json:"confidence"`
	IsUsed         bool      `json:"is_used"`
	UsedAt         *time.Time `json:"used_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Repository handles analysis database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new analysis repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateAnalysis inserts a new analysis record
func (r *Repository) CreateAnalysis(ctx context.Context, a *Analysis) error {
	keyPointsJSON, _ := json.Marshal(a.KeyPoints)
	metadataJSON, _ := json.Marshal(a.Metadata)

	query := `
		INSERT INTO document_analyses (
			document_id, tenant_id, status, document_type, document_subtype,
			classification_confidence, is_scanned, ocr_provider, ocr_confidence,
			summary, key_points, extracted_text, text_length, page_count,
			language, ai_model, prompt_version, tokens_used, processing_time_ms,
			estimated_cost, error_message, error_code, retry_count, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		a.DocumentID, a.TenantID, a.Status, a.DocumentType, a.DocumentSubtype,
		a.ClassificationConfidence, a.IsScanned, a.OCRProvider, a.OCRConfidence,
		a.Summary, keyPointsJSON, a.ExtractedText, a.TextLength, a.PageCount,
		a.Language, a.AIModel, a.PromptVersion, a.TokensUsed, a.ProcessingTimeMs,
		a.EstimatedCost, a.ErrorMessage, a.ErrorCode, a.RetryCount, metadataJSON,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// GetAnalysisByID retrieves an analysis by ID
func (r *Repository) GetAnalysisByID(ctx context.Context, id uuid.UUID) (*Analysis, error) {
	query := `
		SELECT id, document_id, tenant_id, status, document_type, document_subtype,
			classification_confidence, is_scanned, ocr_provider, ocr_confidence,
			summary, key_points, extracted_text, text_length, page_count,
			language, ai_model, prompt_version, tokens_used, processing_time_ms,
			estimated_cost, error_message, error_code, retry_count, metadata,
			created_at, updated_at, completed_at
		FROM document_analyses
		WHERE id = $1
	`

	a := &Analysis{}
	var keyPointsJSON, metadataJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.DocumentID, &a.TenantID, &a.Status, &a.DocumentType, &a.DocumentSubtype,
		&a.ClassificationConfidence, &a.IsScanned, &a.OCRProvider, &a.OCRConfidence,
		&a.Summary, &keyPointsJSON, &a.ExtractedText, &a.TextLength, &a.PageCount,
		&a.Language, &a.AIModel, &a.PromptVersion, &a.TokensUsed, &a.ProcessingTimeMs,
		&a.EstimatedCost, &a.ErrorMessage, &a.ErrorCode, &a.RetryCount, &metadataJSON,
		&a.CreatedAt, &a.UpdatedAt, &a.CompletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnalysisNotFound
		}
		return nil, fmt.Errorf("get analysis: %w", err)
	}

	json.Unmarshal(keyPointsJSON, &a.KeyPoints)
	json.Unmarshal(metadataJSON, &a.Metadata)

	return a, nil
}

// GetAnalysisByDocumentID retrieves the latest analysis for a document
func (r *Repository) GetAnalysisByDocumentID(ctx context.Context, documentID uuid.UUID) (*Analysis, error) {
	query := `
		SELECT id, document_id, tenant_id, status, document_type, document_subtype,
			classification_confidence, is_scanned, ocr_provider, ocr_confidence,
			summary, key_points, extracted_text, text_length, page_count,
			language, ai_model, prompt_version, tokens_used, processing_time_ms,
			estimated_cost, error_message, error_code, retry_count, metadata,
			created_at, updated_at, completed_at
		FROM document_analyses
		WHERE document_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	a := &Analysis{}
	var keyPointsJSON, metadataJSON []byte

	err := r.db.QueryRow(ctx, query, documentID).Scan(
		&a.ID, &a.DocumentID, &a.TenantID, &a.Status, &a.DocumentType, &a.DocumentSubtype,
		&a.ClassificationConfidence, &a.IsScanned, &a.OCRProvider, &a.OCRConfidence,
		&a.Summary, &keyPointsJSON, &a.ExtractedText, &a.TextLength, &a.PageCount,
		&a.Language, &a.AIModel, &a.PromptVersion, &a.TokensUsed, &a.ProcessingTimeMs,
		&a.EstimatedCost, &a.ErrorMessage, &a.ErrorCode, &a.RetryCount, &metadataJSON,
		&a.CreatedAt, &a.UpdatedAt, &a.CompletedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAnalysisNotFound
		}
		return nil, fmt.Errorf("get analysis by document: %w", err)
	}

	json.Unmarshal(keyPointsJSON, &a.KeyPoints)
	json.Unmarshal(metadataJSON, &a.Metadata)

	return a, nil
}

// UpdateAnalysis updates an analysis record
func (r *Repository) UpdateAnalysis(ctx context.Context, a *Analysis) error {
	keyPointsJSON, _ := json.Marshal(a.KeyPoints)
	metadataJSON, _ := json.Marshal(a.Metadata)

	query := `
		UPDATE document_analyses SET
			status = $2, document_type = $3, document_subtype = $4,
			classification_confidence = $5, is_scanned = $6, ocr_provider = $7,
			ocr_confidence = $8, summary = $9, key_points = $10,
			extracted_text = $11, text_length = $12, page_count = $13,
			language = $14, ai_model = $15, prompt_version = $16,
			tokens_used = $17, processing_time_ms = $18, estimated_cost = $19,
			error_message = $20, error_code = $21, retry_count = $22,
			metadata = $23, updated_at = NOW(), completed_at = $24
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		a.ID, a.Status, a.DocumentType, a.DocumentSubtype,
		a.ClassificationConfidence, a.IsScanned, a.OCRProvider,
		a.OCRConfidence, a.Summary, keyPointsJSON,
		a.ExtractedText, a.TextLength, a.PageCount,
		a.Language, a.AIModel, a.PromptVersion,
		a.TokensUsed, a.ProcessingTimeMs, a.EstimatedCost,
		a.ErrorMessage, a.ErrorCode, a.RetryCount,
		metadataJSON, a.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("update analysis: %w", err)
	}

	return nil
}

// ListAnalyses returns analyses for a tenant
func (r *Repository) ListAnalyses(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Analysis, int, error) {
	countQuery := `SELECT COUNT(*) FROM document_analyses WHERE tenant_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count analyses: %w", err)
	}

	query := `
		SELECT id, document_id, tenant_id, status, document_type, document_subtype,
			classification_confidence, is_scanned, ocr_provider, ocr_confidence,
			summary, key_points, extracted_text, text_length, page_count,
			language, ai_model, prompt_version, tokens_used, processing_time_ms,
			estimated_cost, error_message, error_code, retry_count, metadata,
			created_at, updated_at, completed_at
		FROM document_analyses
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list analyses: %w", err)
	}
	defer rows.Close()

	var analyses []*Analysis
	for rows.Next() {
		a := &Analysis{}
		var keyPointsJSON, metadataJSON []byte

		err := rows.Scan(
			&a.ID, &a.DocumentID, &a.TenantID, &a.Status, &a.DocumentType, &a.DocumentSubtype,
			&a.ClassificationConfidence, &a.IsScanned, &a.OCRProvider, &a.OCRConfidence,
			&a.Summary, &keyPointsJSON, &a.ExtractedText, &a.TextLength, &a.PageCount,
			&a.Language, &a.AIModel, &a.PromptVersion, &a.TokensUsed, &a.ProcessingTimeMs,
			&a.EstimatedCost, &a.ErrorMessage, &a.ErrorCode, &a.RetryCount, &metadataJSON,
			&a.CreatedAt, &a.UpdatedAt, &a.CompletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan analysis: %w", err)
		}

		json.Unmarshal(keyPointsJSON, &a.KeyPoints)
		json.Unmarshal(metadataJSON, &a.Metadata)
		analyses = append(analyses, a)
	}

	return analyses, total, nil
}

// CreateDeadline inserts a new deadline
func (r *Repository) CreateDeadline(ctx context.Context, d *Deadline) error {
	query := `
		INSERT INTO extracted_deadlines (
			analysis_id, document_id, tenant_id, deadline_type, deadline_date,
			description, source_text, confidence, is_hard
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		d.AnalysisID, d.DocumentID, d.TenantID, d.DeadlineType, d.Date,
		d.Description, d.SourceText, d.Confidence, d.IsHard,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

// GetDeadlinesByDocument returns all deadlines for a document
func (r *Repository) GetDeadlinesByDocument(ctx context.Context, documentID uuid.UUID) ([]*Deadline, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, deadline_type, deadline_date,
			description, source_text, confidence, is_hard, acknowledged, created_at, updated_at
		FROM extracted_deadlines
		WHERE document_id = $1
		ORDER BY deadline_date ASC
	`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("get deadlines: %w", err)
	}
	defer rows.Close()

	var deadlines []*Deadline
	for rows.Next() {
		d := &Deadline{}
		err := rows.Scan(
			&d.ID, &d.AnalysisID, &d.DocumentID, &d.TenantID, &d.DeadlineType, &d.Date,
			&d.Description, &d.SourceText, &d.Confidence, &d.IsHard, &d.IsAcknowledged,
			&d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan deadline: %w", err)
		}
		deadlines = append(deadlines, d)
	}

	return deadlines, nil
}

// GetUpcomingDeadlines returns upcoming deadlines for a tenant
func (r *Repository) GetUpcomingDeadlines(ctx context.Context, tenantID uuid.UUID, days int) ([]*Deadline, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, deadline_type, deadline_date,
			description, source_text, confidence, is_hard, acknowledged, created_at, updated_at
		FROM extracted_deadlines
		WHERE tenant_id = $1
			AND deadline_date >= CURRENT_DATE
			AND deadline_date <= CURRENT_DATE + $2 * INTERVAL '1 day'
			AND acknowledged = FALSE
		ORDER BY deadline_date ASC
	`

	rows, err := r.db.Query(ctx, query, tenantID, days)
	if err != nil {
		return nil, fmt.Errorf("get upcoming deadlines: %w", err)
	}
	defer rows.Close()

	var deadlines []*Deadline
	for rows.Next() {
		d := &Deadline{}
		err := rows.Scan(
			&d.ID, &d.AnalysisID, &d.DocumentID, &d.TenantID, &d.DeadlineType, &d.Date,
			&d.Description, &d.SourceText, &d.Confidence, &d.IsHard, &d.IsAcknowledged,
			&d.CreatedAt, &d.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan deadline: %w", err)
		}
		deadlines = append(deadlines, d)
	}

	return deadlines, nil
}

// AcknowledgeDeadline marks a deadline as acknowledged
func (r *Repository) AcknowledgeDeadline(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE extracted_deadlines SET acknowledged = TRUE, updated_at = NOW() WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("acknowledge deadline: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrDeadlineNotFound
	}
	return nil
}

// CreateAmount inserts a new amount
func (r *Repository) CreateAmount(ctx context.Context, a *Amount) error {
	query := `
		INSERT INTO extracted_amounts (
			analysis_id, document_id, tenant_id, amount_type, amount, currency,
			description, source_text, confidence, due_date
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	return r.db.QueryRow(ctx, query,
		a.AnalysisID, a.DocumentID, a.TenantID, a.AmountType, a.Amount, a.Currency,
		a.Description, a.SourceText, a.Confidence, a.DueDate,
	).Scan(&a.ID, &a.CreatedAt)
}

// GetAmountsByDocument returns all amounts for a document
func (r *Repository) GetAmountsByDocument(ctx context.Context, documentID uuid.UUID) ([]*Amount, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, amount_type, amount, currency,
			description, source_text, confidence, due_date,
			COALESCE(corrected_by_user, false), COALESCE(notes, ''),
			created_at, COALESCE(updated_at, created_at)
		FROM extracted_amounts
		WHERE document_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("get amounts: %w", err)
	}
	defer rows.Close()

	var amounts []*Amount
	for rows.Next() {
		a := &Amount{}
		err := rows.Scan(
			&a.ID, &a.AnalysisID, &a.DocumentID, &a.TenantID, &a.AmountType, &a.Amount, &a.Currency,
			&a.Description, &a.SourceText, &a.Confidence, &a.DueDate,
			&a.CorrectedByUser, &a.Notes, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan amount: %w", err)
		}
		amounts = append(amounts, a)
	}

	return amounts, nil
}

// GetAmountByID returns a single amount by ID
func (r *Repository) GetAmountByID(ctx context.Context, id uuid.UUID) (*Amount, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, amount_type, amount, currency,
			description, source_text, confidence, due_date,
			COALESCE(corrected_by_user, false), COALESCE(notes, ''),
			created_at, COALESCE(updated_at, created_at)
		FROM extracted_amounts
		WHERE id = $1
	`

	a := &Amount{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.AnalysisID, &a.DocumentID, &a.TenantID, &a.AmountType, &a.Amount, &a.Currency,
		&a.Description, &a.SourceText, &a.Confidence, &a.DueDate,
		&a.CorrectedByUser, &a.Notes, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAmountNotFound
		}
		return nil, fmt.Errorf("get amount: %w", err)
	}

	return a, nil
}

// UpdateAmount updates an amount
func (r *Repository) UpdateAmount(ctx context.Context, a *Amount) error {
	query := `
		UPDATE extracted_amounts SET
			amount_type = $2, amount = $3, currency = $4, description = $5,
			due_date = $6, corrected_by_user = $7, notes = $8, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		a.ID, a.AmountType, a.Amount, a.Currency, a.Description,
		a.DueDate, a.CorrectedByUser, a.Notes,
	)
	if err != nil {
		return fmt.Errorf("update amount: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAmountNotFound
	}

	return nil
}

// DeleteAmount deletes an amount
func (r *Repository) DeleteAmount(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM extracted_amounts WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete amount: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrAmountNotFound
	}

	return nil
}

// CreateActionItem inserts a new action item
func (r *Repository) CreateActionItem(ctx context.Context, a *ActionItem) error {
	query := `
		INSERT INTO action_items (
			analysis_id, document_id, tenant_id, title, description, priority,
			category, status, due_date, assigned_to, source_text, confidence
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		a.AnalysisID, a.DocumentID, a.TenantID, a.Title, a.Description, a.Priority,
		a.Category, a.Status, a.DueDate, a.AssignedTo, a.SourceText, a.Confidence,
	).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
}

// GetActionItemsByDocument returns all action items for a document
func (r *Repository) GetActionItemsByDocument(ctx context.Context, documentID uuid.UUID) ([]*ActionItem, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, title, description, priority,
			category, status, due_date, assigned_to, source_text, confidence,
			completed_at, created_at, updated_at
		FROM action_items
		WHERE document_id = $1
		ORDER BY priority ASC, created_at DESC
	`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("get action items: %w", err)
	}
	defer rows.Close()

	var items []*ActionItem
	for rows.Next() {
		a := &ActionItem{}
		err := rows.Scan(
			&a.ID, &a.AnalysisID, &a.DocumentID, &a.TenantID, &a.Title, &a.Description, &a.Priority,
			&a.Category, &a.Status, &a.DueDate, &a.AssignedTo, &a.SourceText, &a.Confidence,
			&a.CompletedAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan action item: %w", err)
		}
		items = append(items, a)
	}

	return items, nil
}

// GetPendingActionItems returns pending action items for a tenant
func (r *Repository) GetPendingActionItems(ctx context.Context, tenantID uuid.UUID) ([]*ActionItem, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, title, description, priority,
			category, status, due_date, assigned_to, source_text, confidence,
			completed_at, created_at, updated_at
		FROM action_items
		WHERE tenant_id = $1 AND status = 'pending'
		ORDER BY priority ASC, due_date ASC NULLS LAST
	`

	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("get pending action items: %w", err)
	}
	defer rows.Close()

	var items []*ActionItem
	for rows.Next() {
		a := &ActionItem{}
		err := rows.Scan(
			&a.ID, &a.AnalysisID, &a.DocumentID, &a.TenantID, &a.Title, &a.Description, &a.Priority,
			&a.Category, &a.Status, &a.DueDate, &a.AssignedTo, &a.SourceText, &a.Confidence,
			&a.CompletedAt, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan action item: %w", err)
		}
		items = append(items, a)
	}

	return items, nil
}

// UpdateActionItemStatus updates an action item's status
func (r *Repository) UpdateActionItemStatus(ctx context.Context, id uuid.UUID, status ActionStatus) error {
	var completedAt *time.Time
	if status == "completed" {
		now := time.Now()
		completedAt = &now
	}

	query := `UPDATE action_items SET status = $2, completed_at = $3, updated_at = NOW() WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id, status, completedAt)
	if err != nil {
		return fmt.Errorf("update action item status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

// CreateSuggestion inserts a new suggestion
func (r *Repository) CreateSuggestion(ctx context.Context, s *Suggestion) error {
	query := `
		INSERT INTO response_suggestions (
			analysis_id, document_id, tenant_id, suggestion_type, title,
			content, reasoning, confidence
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`

	return r.db.QueryRow(ctx, query,
		s.AnalysisID, s.DocumentID, s.TenantID, s.SuggestionType, s.Title,
		s.Content, s.Reasoning, s.Confidence,
	).Scan(&s.ID, &s.CreatedAt)
}

// GetSuggestionsByDocument returns all suggestions for a document
func (r *Repository) GetSuggestionsByDocument(ctx context.Context, documentID uuid.UUID) ([]*Suggestion, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, suggestion_type, title,
			content, reasoning, confidence, is_used, used_at, created_at
		FROM response_suggestions
		WHERE document_id = $1
		ORDER BY confidence DESC
	`

	rows, err := r.db.Query(ctx, query, documentID)
	if err != nil {
		return nil, fmt.Errorf("get suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []*Suggestion
	for rows.Next() {
		s := &Suggestion{}
		err := rows.Scan(
			&s.ID, &s.AnalysisID, &s.DocumentID, &s.TenantID, &s.SuggestionType, &s.Title,
			&s.Content, &s.Reasoning, &s.Confidence, &s.IsUsed, &s.UsedAt, &s.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan suggestion: %w", err)
		}
		suggestions = append(suggestions, s)
	}

	return suggestions, nil
}

// MarkSuggestionUsed marks a suggestion as used
func (r *Repository) MarkSuggestionUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE response_suggestions SET is_used = TRUE, used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// GetAnalysisStats returns analysis statistics for a tenant
func (r *Repository) GetAnalysisStats(ctx context.Context, tenantID uuid.UUID) (*AnalysisStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COALESCE(SUM(tokens_used), 0) as total_tokens,
			COALESCE(SUM(estimated_cost), 0) as total_cost,
			COALESCE(AVG(processing_time_ms) FILTER (WHERE status = 'completed'), 0) as avg_processing_time
		FROM document_analyses
		WHERE tenant_id = $1
	`

	stats := &AnalysisStats{}
	err := r.db.QueryRow(ctx, query, tenantID).Scan(
		&stats.TotalAnalyses, &stats.CompletedAnalyses, &stats.PendingAnalyses,
		&stats.FailedAnalyses, &stats.TotalTokensUsed, &stats.TotalCost,
		&stats.AvgProcessingTimeMs,
	)
	if err != nil {
		return nil, fmt.Errorf("get analysis stats: %w", err)
	}

	return stats, nil
}

// AnalysisStats holds aggregate statistics
type AnalysisStats struct {
	TotalAnalyses       int     `json:"total_analyses"`
	CompletedAnalyses   int     `json:"completed_analyses"`
	PendingAnalyses     int     `json:"pending_analyses"`
	FailedAnalyses      int     `json:"failed_analyses"`
	TotalTokensUsed     int     `json:"total_tokens_used"`
	TotalCost           float64 `json:"total_cost"`
	AvgProcessingTimeMs float64 `json:"avg_processing_time_ms"`
}

// LogAIUsage logs AI API usage
func (r *Repository) LogAIUsage(ctx context.Context, log *AIUsageLog) error {
	query := `
		INSERT INTO ai_usage_logs (
			tenant_id, analysis_id, operation, model, prompt_type,
			input_tokens, output_tokens, total_tokens, estimated_cost,
			latency_ms, success, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	return r.db.QueryRow(ctx, query,
		log.TenantID, log.AnalysisID, log.Operation, log.Model, log.PromptType,
		log.InputTokens, log.OutputTokens, log.TotalTokens, log.EstimatedCost,
		log.LatencyMs, log.Success, log.ErrorMessage,
	).Scan(&log.ID, &log.CreatedAt)
}

// AIUsageLog represents an AI API usage log entry
type AIUsageLog struct {
	ID            uuid.UUID  `json:"id"`
	TenantID      uuid.UUID  `json:"tenant_id"`
	AnalysisID    *uuid.UUID `json:"analysis_id,omitempty"`
	Operation     string     `json:"operation"`
	Model         string     `json:"model"`
	PromptType    string     `json:"prompt_type,omitempty"`
	InputTokens   int        `json:"input_tokens"`
	OutputTokens  int        `json:"output_tokens"`
	TotalTokens   int        `json:"total_tokens"`
	EstimatedCost float64    `json:"estimated_cost"`
	LatencyMs     int        `json:"latency_ms"`
	Success       bool       `json:"success"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Analysis status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Deadline type constants
const (
	DeadlineTypeResponse   = "response"
	DeadlineTypePayment    = "payment"
	DeadlineTypeSubmission = "submission"
	DeadlineTypeAppeal     = "appeal"
	DeadlineTypeOther      = "other"
)

// Action item priority constants
const (
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

// Action item status constants
const (
	ActionStatusPending   ActionStatus = "pending"
	ActionStatusCompleted ActionStatus = "completed"
	ActionStatusCancelled ActionStatus = "cancelled"
)

// ResponseTemplate represents a response template
type ResponseTemplate struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	Name        string     `json:"name"`
	Category    string     `json:"category"`
	Content     string     `json:"content"`
	Description string     `json:"description,omitempty"`
	Variables   []string   `json:"variables,omitempty"`
	IsActive    bool       `json:"is_active"`
	UsageCount  int        `json:"usage_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// GetDeadlineByID retrieves a deadline by ID
func (r *Repository) GetDeadlineByID(ctx context.Context, id uuid.UUID) (*Deadline, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, deadline_type, deadline_date,
			description, source_text, confidence, is_hard, is_acknowledged,
			acknowledged_at, manually_set, corrected_by_user, notes, created_at, updated_at
		FROM extracted_deadlines
		WHERE id = $1
	`

	d := &Deadline{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.AnalysisID, &d.DocumentID, &d.TenantID, &d.DeadlineType, &d.Date,
		&d.Description, &d.SourceText, &d.Confidence, &d.IsHard, &d.IsAcknowledged,
		&d.AcknowledgedAt, &d.ManuallySet, &d.CorrectedByUser, &d.Notes,
		&d.CreatedAt, &d.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeadlineNotFound
		}
		return nil, fmt.Errorf("get deadline: %w", err)
	}

	return d, nil
}

// UpdateDeadline updates a deadline
func (r *Repository) UpdateDeadline(ctx context.Context, d *Deadline) error {
	query := `
		UPDATE extracted_deadlines SET
			deadline_date = $2, description = $3, is_acknowledged = $4,
			acknowledged_at = $5, manually_set = $6, corrected_by_user = $7,
			notes = $8, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		d.ID, d.Date, d.Description, d.IsAcknowledged,
		d.AcknowledgedAt, d.ManuallySet, d.CorrectedByUser, d.Notes,
	)
	if err != nil {
		return fmt.Errorf("update deadline: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrDeadlineNotFound
	}
	return nil
}

// GetActionItemByID retrieves an action item by ID
func (r *Repository) GetActionItemByID(ctx context.Context, id uuid.UUID) (*ActionItem, error) {
	query := `
		SELECT id, analysis_id, document_id, tenant_id, title, description, priority,
			category, status, due_date, assigned_to, source_text, confidence,
			notes, completed_at, created_at, updated_at
		FROM action_items
		WHERE id = $1
	`

	a := &ActionItem{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.AnalysisID, &a.DocumentID, &a.TenantID, &a.Title, &a.Description, &a.Priority,
		&a.Category, &a.Status, &a.DueDate, &a.AssignedTo, &a.SourceText, &a.Confidence,
		&a.Notes, &a.CompletedAt, &a.CreatedAt, &a.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrActionItemNotFound
		}
		return nil, fmt.Errorf("get action item: %w", err)
	}

	return a, nil
}

// UpdateActionItem updates an action item
func (r *Repository) UpdateActionItem(ctx context.Context, a *ActionItem) error {
	query := `
		UPDATE action_items SET
			title = $2, description = $3, priority = $4, status = $5,
			due_date = $6, assigned_to = $7, notes = $8, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		a.ID, a.Title, a.Description, a.Priority, a.Status,
		a.DueDate, a.AssignedTo, a.Notes,
	)
	if err != nil {
		return fmt.Errorf("update action item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

// DeleteActionItem deletes an action item
func (r *Repository) DeleteActionItem(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM action_items WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete action item: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrActionItemNotFound
	}
	return nil
}

// CreateResponseTemplate creates a new response template
func (r *Repository) CreateResponseTemplate(ctx context.Context, t *ResponseTemplate) error {
	variablesJSON, _ := json.Marshal(t.Variables)

	query := `
		INSERT INTO response_templates (
			tenant_id, name, category, content, description, variables, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRow(ctx, query,
		t.TenantID, t.Name, t.Category, t.Content, t.Description, variablesJSON, t.IsActive,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// GetResponseTemplateByID retrieves a response template by ID
func (r *Repository) GetResponseTemplateByID(ctx context.Context, id uuid.UUID) (*ResponseTemplate, error) {
	query := `
		SELECT id, tenant_id, name, category, content, description, variables,
			is_active, usage_count, created_at, updated_at
		FROM response_templates
		WHERE id = $1
	`

	t := &ResponseTemplate{}
	var variablesJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.TenantID, &t.Name, &t.Category, &t.Content, &t.Description,
		&variablesJSON, &t.IsActive, &t.UsageCount, &t.CreatedAt, &t.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("get response template: %w", err)
	}

	json.Unmarshal(variablesJSON, &t.Variables)
	return t, nil
}

// ListResponseTemplates lists response templates for a tenant
func (r *Repository) ListResponseTemplates(ctx context.Context, tenantID uuid.UUID, category string) ([]*ResponseTemplate, error) {
	query := `
		SELECT id, tenant_id, name, category, content, description, variables,
			is_active, usage_count, created_at, updated_at
		FROM response_templates
		WHERE tenant_id = $1 AND is_active = TRUE
	`
	args := []interface{}{tenantID}

	if category != "" {
		query += " AND category = $2"
		args = append(args, category)
	}

	query += " ORDER BY usage_count DESC, name ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list response templates: %w", err)
	}
	defer rows.Close()

	var templates []*ResponseTemplate
	for rows.Next() {
		t := &ResponseTemplate{}
		var variablesJSON []byte

		err := rows.Scan(
			&t.ID, &t.TenantID, &t.Name, &t.Category, &t.Content, &t.Description,
			&variablesJSON, &t.IsActive, &t.UsageCount, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan response template: %w", err)
		}

		json.Unmarshal(variablesJSON, &t.Variables)
		templates = append(templates, t)
	}

	return templates, nil
}

// UpdateResponseTemplate updates a response template
func (r *Repository) UpdateResponseTemplate(ctx context.Context, t *ResponseTemplate) error {
	variablesJSON, _ := json.Marshal(t.Variables)

	query := `
		UPDATE response_templates SET
			name = $2, category = $3, content = $4, description = $5,
			variables = $6, is_active = $7, updated_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		t.ID, t.Name, t.Category, t.Content, t.Description, variablesJSON, t.IsActive,
	)
	if err != nil {
		return fmt.Errorf("update response template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// DeleteResponseTemplate deletes a response template
func (r *Repository) DeleteResponseTemplate(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM response_templates WHERE id = $1`
	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete response template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}
