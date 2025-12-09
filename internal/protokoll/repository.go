package protokoll

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProtokollNotFound = errors.New("protokoll not found")
)

// ProtokollType represents the type of ELDA communication
type ProtokollType string

const (
	ProtokollTypeAnmeldung    ProtokollType = "ANMELDUNG"
	ProtokollTypeAbmeldung    ProtokollType = "ABMELDUNG"
	ProtokollTypeAenderung    ProtokollType = "AENDERUNG"
	ProtokollTypeMBGM         ProtokollType = "MBGM"
	ProtokollTypeL16          ProtokollType = "L16"
	ProtokollTypeDataboxSync  ProtokollType = "DATABOX_SYNC"
	ProtokollTypeStatusQuery  ProtokollType = "STATUS_QUERY"
)

// ProtokollStatus represents the result status
type ProtokollStatus string

const (
	ProtokollStatusSuccess  ProtokollStatus = "success"
	ProtokollStatusError    ProtokollStatus = "error"
	ProtokollStatusPending  ProtokollStatus = "pending"
)

// Protokoll represents an ELDA communication log entry
type Protokoll struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	ELDAAccountID   uuid.UUID       `json:"elda_account_id" db:"elda_account_id"`
	Type            ProtokollType   `json:"type" db:"type"`
	Status          ProtokollStatus `json:"status" db:"status"`
	Protokollnummer string          `json:"protokollnummer,omitempty" db:"protokollnummer"`
	RelatedID       *uuid.UUID      `json:"related_id,omitempty" db:"related_id"` // mBGM, L16, meldung ID
	SVNummer        string          `json:"sv_nummer,omitempty" db:"sv_nummer"`
	Description     string          `json:"description" db:"description"`
	RequestXML      string          `json:"-" db:"request_xml"`
	ResponseXML     string          `json:"-" db:"response_xml"`
	ErrorCode       string          `json:"error_code,omitempty" db:"error_code"`
	ErrorMessage    string          `json:"error_message,omitempty" db:"error_message"`
	DurationMS      int64           `json:"duration_ms" db:"duration_ms"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	CreatedBy       *uuid.UUID      `json:"created_by,omitempty" db:"created_by"`
}

// Repository handles ELDA protokoll database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new protokoll repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new protokoll entry
func (r *Repository) Create(ctx context.Context, p *Protokoll) error {
	query := `
		INSERT INTO elda_protokolle (
			id, elda_account_id, type, status, protokollnummer,
			related_id, sv_nummer, description,
			request_xml, response_xml, error_code, error_message,
			duration_ms, created_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15
		)
	`

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}

	_, err := r.db.Exec(ctx, query,
		p.ID, p.ELDAAccountID, p.Type, p.Status, p.Protokollnummer,
		p.RelatedID, p.SVNummer, p.Description,
		p.RequestXML, p.ResponseXML, p.ErrorCode, p.ErrorMessage,
		p.DurationMS, p.CreatedAt, p.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("create protokoll: %w", err)
	}

	return nil
}

// GetByID retrieves a protokoll by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Protokoll, error) {
	query := `
		SELECT
			id, elda_account_id, type, status, protokollnummer,
			related_id, sv_nummer, description,
			request_xml, response_xml, error_code, error_message,
			duration_ms, created_at, created_by
		FROM elda_protokolle
		WHERE id = $1
	`

	p := &Protokoll{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID, &p.ELDAAccountID, &p.Type, &p.Status, &p.Protokollnummer,
		&p.RelatedID, &p.SVNummer, &p.Description,
		&p.RequestXML, &p.ResponseXML, &p.ErrorCode, &p.ErrorMessage,
		&p.DurationMS, &p.CreatedAt, &p.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProtokollNotFound
		}
		return nil, fmt.Errorf("get protokoll: %w", err)
	}

	return p, nil
}

// GetByProtokollnummer retrieves by ELDA Protokollnummer
func (r *Repository) GetByProtokollnummer(ctx context.Context, nummer string) (*Protokoll, error) {
	query := `
		SELECT
			id, elda_account_id, type, status, protokollnummer,
			related_id, sv_nummer, description,
			request_xml, response_xml, error_code, error_message,
			duration_ms, created_at, created_by
		FROM elda_protokolle
		WHERE protokollnummer = $1
	`

	p := &Protokoll{}
	err := r.db.QueryRow(ctx, query, nummer).Scan(
		&p.ID, &p.ELDAAccountID, &p.Type, &p.Status, &p.Protokollnummer,
		&p.RelatedID, &p.SVNummer, &p.Description,
		&p.RequestXML, &p.ResponseXML, &p.ErrorCode, &p.ErrorMessage,
		&p.DurationMS, &p.CreatedAt, &p.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProtokollNotFound
		}
		return nil, fmt.Errorf("get by protokollnummer: %w", err)
	}

	return p, nil
}

// ListFilter contains filter options for listing protokolle
type ListFilter struct {
	ELDAAccountID *uuid.UUID
	Type          *ProtokollType
	Status        *ProtokollStatus
	SVNummer      string
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int
	Offset        int
}

// List retrieves protokolle with filters
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*Protokoll, error) {
	query := `
		SELECT
			id, elda_account_id, type, status, protokollnummer,
			related_id, sv_nummer, description,
			error_code, error_message, duration_ms, created_at, created_by
		FROM elda_protokolle
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *filter.Type)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.SVNummer != "" {
		query += fmt.Sprintf(" AND sv_nummer = $%d", argIndex)
		args = append(args, filter.SVNummer)
		argIndex++
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIndex)
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIndex)
		args = append(args, *filter.EndDate)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list protokolle: %w", err)
	}
	defer rows.Close()

	var results []*Protokoll
	for rows.Next() {
		p := &Protokoll{}
		err := rows.Scan(
			&p.ID, &p.ELDAAccountID, &p.Type, &p.Status, &p.Protokollnummer,
			&p.RelatedID, &p.SVNummer, &p.Description,
			&p.ErrorCode, &p.ErrorMessage, &p.DurationMS, &p.CreatedAt, &p.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("scan protokoll: %w", err)
		}
		results = append(results, p)
	}

	return results, nil
}

// Count returns the count of protokolle matching the filter
func (r *Repository) Count(ctx context.Context, filter ListFilter) (int, error) {
	query := `SELECT COUNT(*) FROM elda_protokolle WHERE 1=1`
	args := []interface{}{}
	argIndex := 1

	if filter.ELDAAccountID != nil {
		query += fmt.Sprintf(" AND elda_account_id = $%d", argIndex)
		args = append(args, *filter.ELDAAccountID)
		argIndex++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, *filter.Type)
		argIndex++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, *filter.Status)
		argIndex++
	}

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count protokolle: %w", err)
	}

	return count, nil
}

// GetHistoryBySVNummer returns all protokolle for an SV-Nummer
func (r *Repository) GetHistoryBySVNummer(ctx context.Context, accountID uuid.UUID, svNummer string) ([]*Protokoll, error) {
	filter := ListFilter{
		ELDAAccountID: &accountID,
		SVNummer:      svNummer,
		Limit:         100,
	}
	return r.List(ctx, filter)
}

// GetStatistics returns submission statistics for an account
func (r *Repository) GetStatistics(ctx context.Context, accountID uuid.UUID, days int) (*ProtokollStatistics, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE status = 'success') as success,
			COUNT(*) FILTER (WHERE status = 'error') as errors,
			COUNT(*) FILTER (WHERE type = 'MBGM') as mbgm_count,
			COUNT(*) FILTER (WHERE type = 'L16') as l16_count,
			COUNT(*) FILTER (WHERE type = 'ANMELDUNG') as anmeldung_count,
			COUNT(*) FILTER (WHERE type = 'ABMELDUNG') as abmeldung_count,
			AVG(duration_ms) as avg_duration
		FROM elda_protokolle
		WHERE elda_account_id = $1 AND created_at >= $2
	`

	stats := &ProtokollStatistics{Days: days}
	err := r.db.QueryRow(ctx, query, accountID, startDate).Scan(
		&stats.Total,
		&stats.Success,
		&stats.Errors,
		&stats.MBGMCount,
		&stats.L16Count,
		&stats.AnmeldungCount,
		&stats.AbmeldungCount,
		&stats.AvgDurationMS,
	)
	if err != nil {
		return nil, fmt.Errorf("get statistics: %w", err)
	}

	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Success) / float64(stats.Total) * 100
	}

	return stats, nil
}

// ProtokollStatistics contains submission statistics
type ProtokollStatistics struct {
	Days           int     `json:"days"`
	Total          int     `json:"total"`
	Success        int     `json:"success"`
	Errors         int     `json:"errors"`
	SuccessRate    float64 `json:"success_rate"`
	MBGMCount      int     `json:"mbgm_count"`
	L16Count       int     `json:"l16_count"`
	AnmeldungCount int     `json:"anmeldung_count"`
	AbmeldungCount int     `json:"abmeldung_count"`
	AvgDurationMS  *int64  `json:"avg_duration_ms,omitempty"`
}
