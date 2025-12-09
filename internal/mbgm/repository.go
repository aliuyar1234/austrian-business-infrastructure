package mbgm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/austrian-business-infrastructure/fo/internal/elda"
)

var (
	ErrMBGMNotFound     = errors.New("mBGM not found")
	ErrPositionNotFound = errors.New("mBGM position not found")
	ErrDuplicate        = errors.New("mBGM already exists for this period")
)

// Repository handles mBGM database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new mBGM repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new mBGM record
func (r *Repository) Create(ctx context.Context, m *elda.MBGM) error {
	query := `
		INSERT INTO mbgm (
			id, elda_account_id, year, month, status,
			total_dienstnehmer, total_beitragsgrundlage,
			is_correction, corrects_id, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11, $12
		)
	`

	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		m.ID, m.ELDAAccountID, m.Year, m.Month, m.Status,
		m.TotalDienstnehmer, m.TotalBeitragsgrundlage,
		m.IsCorrection, m.CorrectsID, m.CreatedBy, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		if isDuplicateError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("create mBGM: %w", err)
	}

	return nil
}

// GetByID retrieves an mBGM by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*elda.MBGM, error) {
	query := `
		SELECT
			id, elda_account_id, year, month, status, protokollnummer,
			total_dienstnehmer, total_beitragsgrundlage,
			submitted_at, response_received_at, request_xml, response_xml,
			error_message, error_code,
			is_correction, corrects_id, created_by, created_at, updated_at
		FROM mbgm
		WHERE id = $1
	`

	m := &elda.MBGM{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.ELDAAccountID, &m.Year, &m.Month, &m.Status, &m.Protokollnummer,
		&m.TotalDienstnehmer, &m.TotalBeitragsgrundlage,
		&m.SubmittedAt, &m.ResponseReceivedAt, &m.RequestXML, &m.ResponseXML,
		&m.ErrorMessage, &m.ErrorCode,
		&m.IsCorrection, &m.CorrectsID, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMBGMNotFound
		}
		return nil, fmt.Errorf("get mBGM: %w", err)
	}

	return m, nil
}

// GetByIDWithPositions retrieves an mBGM with all its positions
func (r *Repository) GetByIDWithPositions(ctx context.Context, id uuid.UUID) (*elda.MBGM, error) {
	m, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	positions, err := r.GetPositions(ctx, id)
	if err != nil {
		return nil, err
	}

	m.Positionen = positions
	return m, nil
}

// Update updates an mBGM record
func (r *Repository) Update(ctx context.Context, m *elda.MBGM) error {
	query := `
		UPDATE mbgm SET
			status = $2,
			protokollnummer = $3,
			total_dienstnehmer = $4,
			total_beitragsgrundlage = $5,
			submitted_at = $6,
			response_received_at = $7,
			request_xml = $8,
			response_xml = $9,
			error_message = $10,
			error_code = $11,
			updated_at = $12
		WHERE id = $1
	`

	m.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, query,
		m.ID, m.Status, m.Protokollnummer,
		m.TotalDienstnehmer, m.TotalBeitragsgrundlage,
		m.SubmittedAt, m.ResponseReceivedAt, m.RequestXML, m.ResponseXML,
		m.ErrorMessage, m.ErrorCode, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update mBGM: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrMBGMNotFound
	}

	return nil
}

// Delete deletes an mBGM and its positions
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	// Positions are deleted via CASCADE
	query := `DELETE FROM mbgm WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete mBGM: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrMBGMNotFound
	}

	return nil
}

// ListByAccount retrieves all mBGM for an ELDA account
func (r *Repository) ListByAccount(ctx context.Context, eldaAccountID uuid.UUID, filter *ListFilter) ([]*elda.MBGM, error) {
	query := `
		SELECT
			id, elda_account_id, year, month, status, protokollnummer,
			total_dienstnehmer, total_beitragsgrundlage,
			submitted_at, response_received_at,
			error_message, error_code,
			is_correction, corrects_id, created_by, created_at, updated_at
		FROM mbgm
		WHERE elda_account_id = $1
	`
	args := []interface{}{eldaAccountID}
	argIndex := 2

	if filter != nil {
		if filter.Year != nil {
			query += fmt.Sprintf(" AND year = $%d", argIndex)
			args = append(args, *filter.Year)
			argIndex++
		}
		if filter.Month != nil {
			query += fmt.Sprintf(" AND month = $%d", argIndex)
			args = append(args, *filter.Month)
			argIndex++
		}
		if filter.Status != "" {
			query += fmt.Sprintf(" AND status = $%d", argIndex)
			args = append(args, filter.Status)
			argIndex++
		}
	}

	query += " ORDER BY year DESC, month DESC"

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mBGM: %w", err)
	}
	defer rows.Close()

	var results []*elda.MBGM
	for rows.Next() {
		m := &elda.MBGM{}
		err := rows.Scan(
			&m.ID, &m.ELDAAccountID, &m.Year, &m.Month, &m.Status, &m.Protokollnummer,
			&m.TotalDienstnehmer, &m.TotalBeitragsgrundlage,
			&m.SubmittedAt, &m.ResponseReceivedAt,
			&m.ErrorMessage, &m.ErrorCode,
			&m.IsCorrection, &m.CorrectsID, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan mBGM: %w", err)
		}
		results = append(results, m)
	}

	return results, nil
}

// ListFilter contains filter options for listing mBGM
type ListFilter struct {
	Year   *int
	Month  *int
	Status elda.MBGMStatus
	Limit  int
	Offset int
}

// CreatePosition creates a new mBGM position
func (r *Repository) CreatePosition(ctx context.Context, p *elda.MBGMPosition) error {
	validationJSON, _ := json.Marshal(p.ValidationErrors)

	query := `
		INSERT INTO mbgm_positionen (
			id, mbgm_id, sv_nummer, familienname, vorname, geburtsdatum,
			beitragsgruppe, beitragsgrundlage, sonderzahlung,
			von_datum, bis_datum, wochenstunden,
			is_valid, validation_errors, position_index, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16
		)
	`

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = time.Now()

	_, err := r.db.Exec(ctx, query,
		p.ID, p.MBGMID, p.SVNummer, p.Familienname, p.Vorname, p.Geburtsdatum,
		p.Beitragsgruppe, p.Beitragsgrundlage, p.Sonderzahlung,
		p.VonDatum, p.BisDatum, p.Wochenstunden,
		p.IsValid, validationJSON, p.PositionIndex, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create mBGM position: %w", err)
	}

	return nil
}

// CreatePositions creates multiple positions in a transaction
func (r *Repository) CreatePositions(ctx context.Context, positions []*elda.MBGMPosition) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, p := range positions {
		validationJSON, _ := json.Marshal(p.ValidationErrors)

		if p.ID == uuid.Nil {
			p.ID = uuid.New()
		}
		p.CreatedAt = time.Now()

		_, err := tx.Exec(ctx, `
			INSERT INTO mbgm_positionen (
				id, mbgm_id, sv_nummer, familienname, vorname, geburtsdatum,
				beitragsgruppe, beitragsgrundlage, sonderzahlung,
				von_datum, bis_datum, wochenstunden,
				is_valid, validation_errors, position_index, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9,
				$10, $11, $12,
				$13, $14, $15, $16
			)
		`,
			p.ID, p.MBGMID, p.SVNummer, p.Familienname, p.Vorname, p.Geburtsdatum,
			p.Beitragsgruppe, p.Beitragsgrundlage, p.Sonderzahlung,
			p.VonDatum, p.BisDatum, p.Wochenstunden,
			p.IsValid, validationJSON, p.PositionIndex, p.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("create position %d: %w", p.PositionIndex, err)
		}
	}

	return tx.Commit(ctx)
}

// GetPositions retrieves all positions for an mBGM
func (r *Repository) GetPositions(ctx context.Context, mbgmID uuid.UUID) ([]*elda.MBGMPosition, error) {
	query := `
		SELECT
			id, mbgm_id, sv_nummer, familienname, vorname, geburtsdatum,
			beitragsgruppe, beitragsgrundlage, sonderzahlung,
			von_datum, bis_datum, wochenstunden,
			is_valid, validation_errors, position_index, created_at
		FROM mbgm_positionen
		WHERE mbgm_id = $1
		ORDER BY position_index
	`

	rows, err := r.db.Query(ctx, query, mbgmID)
	if err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	defer rows.Close()

	var positions []*elda.MBGMPosition
	for rows.Next() {
		p := &elda.MBGMPosition{}
		var validationJSON []byte

		err := rows.Scan(
			&p.ID, &p.MBGMID, &p.SVNummer, &p.Familienname, &p.Vorname, &p.Geburtsdatum,
			&p.Beitragsgruppe, &p.Beitragsgrundlage, &p.Sonderzahlung,
			&p.VonDatum, &p.BisDatum, &p.Wochenstunden,
			&p.IsValid, &validationJSON, &p.PositionIndex, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}

		if len(validationJSON) > 0 {
			json.Unmarshal(validationJSON, &p.ValidationErrors)
		}

		positions = append(positions, p)
	}

	return positions, nil
}

// UpdatePosition updates a position
func (r *Repository) UpdatePosition(ctx context.Context, p *elda.MBGMPosition) error {
	validationJSON, _ := json.Marshal(p.ValidationErrors)

	query := `
		UPDATE mbgm_positionen SET
			sv_nummer = $2,
			familienname = $3,
			vorname = $4,
			geburtsdatum = $5,
			beitragsgruppe = $6,
			beitragsgrundlage = $7,
			sonderzahlung = $8,
			von_datum = $9,
			bis_datum = $10,
			wochenstunden = $11,
			is_valid = $12,
			validation_errors = $13
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query,
		p.ID, p.SVNummer, p.Familienname, p.Vorname, p.Geburtsdatum,
		p.Beitragsgruppe, p.Beitragsgrundlage, p.Sonderzahlung,
		p.VonDatum, p.BisDatum, p.Wochenstunden,
		p.IsValid, validationJSON,
	)
	if err != nil {
		return fmt.Errorf("update position: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrPositionNotFound
	}

	return nil
}

// DeletePositions deletes all positions for an mBGM
func (r *Repository) DeletePositions(ctx context.Context, mbgmID uuid.UUID) error {
	query := `DELETE FROM mbgm_positionen WHERE mbgm_id = $1`
	_, err := r.db.Exec(ctx, query, mbgmID)
	if err != nil {
		return fmt.Errorf("delete positions: %w", err)
	}
	return nil
}

// GetBeitragsgruppen retrieves all active Beitragsgruppen
func (r *Repository) GetBeitragsgruppen(ctx context.Context) ([]*elda.Beitragsgruppe, error) {
	query := `
		SELECT code, bezeichnung, beschreibung, valid_from, valid_until, is_active
		FROM beitragsgruppen
		WHERE is_active = true
		ORDER BY code
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get beitragsgruppen: %w", err)
	}
	defer rows.Close()

	var results []*elda.Beitragsgruppe
	for rows.Next() {
		bg := &elda.Beitragsgruppe{}
		err := rows.Scan(&bg.Code, &bg.Bezeichnung, &bg.Beschreibung, &bg.ValidFrom, &bg.ValidUntil, &bg.IsActive)
		if err != nil {
			return nil, fmt.Errorf("scan beitragsgruppe: %w", err)
		}
		results = append(results, bg)
	}

	return results, nil
}

// IsBeitragsgruppValid checks if a Beitragsgruppe code is valid
func (r *Repository) IsBeitragsgruppValid(ctx context.Context, code string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM beitragsgruppen
			WHERE code = $1 AND is_active = true
		)
	`
	var exists bool
	err := r.db.QueryRow(ctx, query, code).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check beitragsgruppe: %w", err)
	}
	return exists, nil
}

// Helper function to check for duplicate key error
func isDuplicateError(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) == false &&
		(contains(err.Error(), "duplicate key") || contains(err.Error(), "UNIQUE constraint")))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
