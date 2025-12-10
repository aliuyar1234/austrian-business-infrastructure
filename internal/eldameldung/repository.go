package eldameldung

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/elda"
)

var (
	ErrMeldungNotFound = errors.New("ELDA meldung not found")
)

// Repository handles ELDA meldung database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new ELDA meldung repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// Create creates a new ELDA meldung record
func (r *Repository) Create(ctx context.Context, m *elda.ELDAMeldung) error {
	beschaeftigungJSON, _ := json.Marshal(m.Beschaeftigung)
	arbeitszeitJSON, _ := json.Marshal(m.Arbeitszeit)
	entgeltJSON, _ := json.Marshal(m.Entgelt)
	adresseJSON, _ := json.Marshal(m.Adresse)
	bankJSON, _ := json.Marshal(m.Bankverbindung)

	query := `
		INSERT INTO elda_meldungen (
			id, elda_account_id, type, status,
			sv_nummer, vorname, nachname, geburtsdatum, geschlecht,
			eintrittsdatum, austrittsdatum, austritt_grund,
			beschaeftigung, arbeitszeit, entgelt, adresse, bankverbindung,
			abfertigung, urlaubsersatz, url_tage,
			aenderung_art, aenderung_datum, original_meldung_id,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12,
			$13, $14, $15, $16, $17,
			$18, $19, $20,
			$21, $22, $23,
			$24, $25, $26
		)
	`

	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	_, err := r.db.Exec(ctx, query,
		m.ID, m.ELDAAccountID, m.Type, m.Status,
		m.SVNummer, m.Vorname, m.Nachname, m.Geburtsdatum, m.Geschlecht,
		m.Eintrittsdatum, m.Austrittsdatum, m.AustrittGrund,
		beschaeftigungJSON, arbeitszeitJSON, entgeltJSON, adresseJSON, bankJSON,
		m.Abfertigung, m.Urlaubsersatz, m.URLTage,
		m.AenderungArt, m.AenderungDatum, m.OriginalMeldungID,
		m.CreatedBy, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create meldung: %w", err)
	}

	return nil
}

// GetByID retrieves a meldung by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*elda.ELDAMeldung, error) {
	query := `
		SELECT
			id, elda_account_id, type, status,
			sv_nummer, vorname, nachname, geburtsdatum, geschlecht,
			eintrittsdatum, austrittsdatum, austritt_grund,
			beschaeftigung, arbeitszeit, entgelt, adresse, bankverbindung,
			abfertigung, urlaubsersatz, url_tage,
			aenderung_art, aenderung_datum, original_meldung_id,
			protokollnummer, submitted_at, request_xml, response_xml,
			error_code, error_message,
			created_by, created_at, updated_at
		FROM elda_meldungen
		WHERE id = $1
	`

	m := &elda.ELDAMeldung{}
	var beschaeftigungJSON, arbeitszeitJSON, entgeltJSON, adresseJSON, bankJSON []byte

	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.ELDAAccountID, &m.Type, &m.Status,
		&m.SVNummer, &m.Vorname, &m.Nachname, &m.Geburtsdatum, &m.Geschlecht,
		&m.Eintrittsdatum, &m.Austrittsdatum, &m.AustrittGrund,
		&beschaeftigungJSON, &arbeitszeitJSON, &entgeltJSON, &adresseJSON, &bankJSON,
		&m.Abfertigung, &m.Urlaubsersatz, &m.URLTage,
		&m.AenderungArt, &m.AenderungDatum, &m.OriginalMeldungID,
		&m.Protokollnummer, &m.SubmittedAt, &m.RequestXML, &m.ResponseXML,
		&m.ErrorCode, &m.ErrorMessage,
		&m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMeldungNotFound
		}
		return nil, fmt.Errorf("get meldung: %w", err)
	}

	// Unmarshal JSON fields
	if len(beschaeftigungJSON) > 0 {
		m.Beschaeftigung = &elda.ExtendedBeschaeftigung{}
		json.Unmarshal(beschaeftigungJSON, m.Beschaeftigung)
	}
	if len(arbeitszeitJSON) > 0 {
		m.Arbeitszeit = &elda.ExtendedArbeitszeit{}
		json.Unmarshal(arbeitszeitJSON, m.Arbeitszeit)
	}
	if len(entgeltJSON) > 0 {
		m.Entgelt = &elda.ExtendedEntgelt{}
		json.Unmarshal(entgeltJSON, m.Entgelt)
	}
	if len(adresseJSON) > 0 {
		m.Adresse = &elda.DienstnehmerAdresse{}
		json.Unmarshal(adresseJSON, m.Adresse)
	}
	if len(bankJSON) > 0 {
		m.Bankverbindung = &elda.Bankverbindung{}
		json.Unmarshal(bankJSON, m.Bankverbindung)
	}

	return m, nil
}

// Update updates a meldung record
func (r *Repository) Update(ctx context.Context, m *elda.ELDAMeldung) error {
	beschaeftigungJSON, _ := json.Marshal(m.Beschaeftigung)
	arbeitszeitJSON, _ := json.Marshal(m.Arbeitszeit)
	entgeltJSON, _ := json.Marshal(m.Entgelt)
	adresseJSON, _ := json.Marshal(m.Adresse)
	bankJSON, _ := json.Marshal(m.Bankverbindung)

	query := `
		UPDATE elda_meldungen SET
			status = $2,
			beschaeftigung = $3,
			arbeitszeit = $4,
			entgelt = $5,
			adresse = $6,
			bankverbindung = $7,
			protokollnummer = $8,
			submitted_at = $9,
			request_xml = $10,
			response_xml = $11,
			error_code = $12,
			error_message = $13,
			updated_at = $14
		WHERE id = $1
	`

	m.UpdatedAt = time.Now()

	result, err := r.db.Exec(ctx, query,
		m.ID, m.Status,
		beschaeftigungJSON, arbeitszeitJSON, entgeltJSON, adresseJSON, bankJSON,
		m.Protokollnummer, m.SubmittedAt, m.RequestXML, m.ResponseXML,
		m.ErrorCode, m.ErrorMessage,
		m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update meldung: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrMeldungNotFound
	}

	return nil
}

// ListFilter contains filter options for listing meldungen
type ListFilter struct {
	ELDAAccountID *uuid.UUID
	Type          *elda.MeldungType
	Status        *elda.MeldungStatus
	SVNummer      string
	StartDate     *time.Time
	EndDate       *time.Time
	Limit         int
	Offset        int
}

// List retrieves meldungen with filters
func (r *Repository) List(ctx context.Context, filter ListFilter) ([]*elda.ELDAMeldung, error) {
	query := `
		SELECT
			id, elda_account_id, type, status,
			sv_nummer, vorname, nachname, geburtsdatum, geschlecht,
			eintrittsdatum, austrittsdatum, austritt_grund,
			beschaeftigung, arbeitszeit, entgelt, adresse, bankverbindung,
			abfertigung, urlaubsersatz, url_tage,
			aenderung_art, aenderung_datum, original_meldung_id,
			protokollnummer, submitted_at,
			error_code, error_message,
			created_by, created_at, updated_at
		FROM elda_meldungen
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
		return nil, fmt.Errorf("list meldungen: %w", err)
	}
	defer rows.Close()

	var results []*elda.ELDAMeldung
	for rows.Next() {
		m := &elda.ELDAMeldung{}
		var beschaeftigungJSON, arbeitszeitJSON, entgeltJSON, adresseJSON, bankJSON []byte

		err := rows.Scan(
			&m.ID, &m.ELDAAccountID, &m.Type, &m.Status,
			&m.SVNummer, &m.Vorname, &m.Nachname, &m.Geburtsdatum, &m.Geschlecht,
			&m.Eintrittsdatum, &m.Austrittsdatum, &m.AustrittGrund,
			&beschaeftigungJSON, &arbeitszeitJSON, &entgeltJSON, &adresseJSON, &bankJSON,
			&m.Abfertigung, &m.Urlaubsersatz, &m.URLTage,
			&m.AenderungArt, &m.AenderungDatum, &m.OriginalMeldungID,
			&m.Protokollnummer, &m.SubmittedAt,
			&m.ErrorCode, &m.ErrorMessage,
			&m.CreatedBy, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan meldung: %w", err)
		}

		// Unmarshal JSON fields
		if len(beschaeftigungJSON) > 0 {
			m.Beschaeftigung = &elda.ExtendedBeschaeftigung{}
			json.Unmarshal(beschaeftigungJSON, m.Beschaeftigung)
		}
		if len(arbeitszeitJSON) > 0 {
			m.Arbeitszeit = &elda.ExtendedArbeitszeit{}
			json.Unmarshal(arbeitszeitJSON, m.Arbeitszeit)
		}
		if len(entgeltJSON) > 0 {
			m.Entgelt = &elda.ExtendedEntgelt{}
			json.Unmarshal(entgeltJSON, m.Entgelt)
		}
		if len(adresseJSON) > 0 {
			m.Adresse = &elda.DienstnehmerAdresse{}
			json.Unmarshal(adresseJSON, m.Adresse)
		}
		if len(bankJSON) > 0 {
			m.Bankverbindung = &elda.Bankverbindung{}
			json.Unmarshal(bankJSON, m.Bankverbindung)
		}

		results = append(results, m)
	}

	return results, nil
}

// Count returns the count of meldungen matching the filter
func (r *Repository) Count(ctx context.Context, filter ListFilter) (int, error) {
	query := `SELECT COUNT(*) FROM elda_meldungen WHERE 1=1`
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

	var count int
	err := r.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count meldungen: %w", err)
	}

	return count, nil
}

// Delete deletes a meldung
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM elda_meldungen WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete meldung: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrMeldungNotFound
	}

	return nil
}

// GetByProtokollnummer retrieves a meldung by protokollnummer
func (r *Repository) GetByProtokollnummer(ctx context.Context, protokollnummer string) (*elda.ELDAMeldung, error) {
	query := `
		SELECT id FROM elda_meldungen WHERE protokollnummer = $1
	`

	var id uuid.UUID
	err := r.db.QueryRow(ctx, query, protokollnummer).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMeldungNotFound
		}
		return nil, fmt.Errorf("get by protokollnummer: %w", err)
	}

	return r.GetByID(ctx, id)
}

// GetHistoryBySVNummer returns all meldungen for an SV-Nummer
func (r *Repository) GetHistoryBySVNummer(ctx context.Context, accountID uuid.UUID, svNummer string) ([]*elda.ELDAMeldung, error) {
	filter := ListFilter{
		ELDAAccountID: &accountID,
		SVNummer:      svNummer,
		Limit:         100,
	}
	return r.List(ctx, filter)
}
