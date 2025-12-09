package firmenbuch

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCompanyNotFound   = errors.New("company not found")
	ErrWatchlistNotFound = errors.New("watchlist entry not found")
)

// Repository handles firmenbuch database operations
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new firmenbuch repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// CreateCompany creates or updates a company record
func (r *Repository) CreateCompany(ctx context.Context, company *Company) (*Company, error) {
	company.ID = uuid.New()
	company.CreatedAt = time.Now()
	company.UpdatedAt = company.CreatedAt

	query := `
		INSERT INTO firmenbuch_cache (
			id, tenant_id, fn, name, rechtsform, sitz, adresse,
			stammkapital, waehrung, status, gruendungsdatum, uid,
			gegenstand, extract_data, last_fetched_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (tenant_id, fn) DO UPDATE SET
			name = EXCLUDED.name,
			rechtsform = EXCLUDED.rechtsform,
			sitz = EXCLUDED.sitz,
			adresse = EXCLUDED.adresse,
			stammkapital = EXCLUDED.stammkapital,
			waehrung = EXCLUDED.waehrung,
			status = EXCLUDED.status,
			gruendungsdatum = EXCLUDED.gruendungsdatum,
			uid = EXCLUDED.uid,
			gegenstand = EXCLUDED.gegenstand,
			extract_data = EXCLUDED.extract_data,
			last_fetched_at = EXCLUDED.last_fetched_at,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		company.ID, company.TenantID, company.FN, company.Name, company.Rechtsform, company.Sitz, company.Adresse,
		company.Stammkapital, company.Waehrung, company.Status, company.Gruendungsdatum, company.UID,
		company.Gegenstand, company.ExtractData, company.LastFetchedAt, company.CreatedAt, company.UpdatedAt,
	).Scan(&company.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to create company: %w", err)
	}

	return company, nil
}

// GetCompanyByFN retrieves a company by FN
func (r *Repository) GetCompanyByFN(ctx context.Context, tenantID uuid.UUID, fn string) (*Company, error) {
	query := `
		SELECT id, tenant_id, fn, name, rechtsform, sitz, adresse,
			stammkapital, waehrung, status, gruendungsdatum, uid,
			gegenstand, extract_data, last_fetched_at, created_at, updated_at
		FROM firmenbuch_cache
		WHERE tenant_id = $1 AND fn = $2`

	var company Company
	var adresse, extractData sql.NullString
	var stammkapital sql.NullInt64
	var waehrung, uid, gegenstand sql.NullString
	var gruendungsdatum, lastFetchedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, tenantID, fn).Scan(
		&company.ID, &company.TenantID, &company.FN, &company.Name, &company.Rechtsform, &company.Sitz, &adresse,
		&stammkapital, &waehrung, &company.Status, &gruendungsdatum, &uid,
		&gegenstand, &extractData, &lastFetchedAt, &company.CreatedAt, &company.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCompanyNotFound
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	if adresse.Valid {
		company.Adresse = []byte(adresse.String)
	}
	if extractData.Valid {
		company.ExtractData = []byte(extractData.String)
	}
	if stammkapital.Valid {
		company.Stammkapital = &stammkapital.Int64
	}
	if waehrung.Valid {
		company.Waehrung = &waehrung.String
	}
	if uid.Valid {
		company.UID = &uid.String
	}
	if gegenstand.Valid {
		company.Gegenstand = &gegenstand.String
	}
	if gruendungsdatum.Valid {
		company.Gruendungsdatum = &gruendungsdatum.Time
	}
	if lastFetchedAt.Valid {
		company.LastFetchedAt = &lastFetchedAt.Time
	}

	return &company, nil
}

// GetCompanyByID retrieves a company by ID
func (r *Repository) GetCompanyByID(ctx context.Context, id, tenantID uuid.UUID) (*Company, error) {
	query := `
		SELECT id, tenant_id, fn, name, rechtsform, sitz, adresse,
			stammkapital, waehrung, status, gruendungsdatum, uid,
			gegenstand, extract_data, last_fetched_at, created_at, updated_at
		FROM firmenbuch_cache
		WHERE id = $1 AND tenant_id = $2`

	var company Company
	var adresse, extractData sql.NullString
	var stammkapital sql.NullInt64
	var waehrung, uid, gegenstand sql.NullString
	var gruendungsdatum, lastFetchedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, id, tenantID).Scan(
		&company.ID, &company.TenantID, &company.FN, &company.Name, &company.Rechtsform, &company.Sitz, &adresse,
		&stammkapital, &waehrung, &company.Status, &gruendungsdatum, &uid,
		&gegenstand, &extractData, &lastFetchedAt, &company.CreatedAt, &company.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCompanyNotFound
		}
		return nil, fmt.Errorf("failed to get company: %w", err)
	}

	if adresse.Valid {
		company.Adresse = []byte(adresse.String)
	}
	if extractData.Valid {
		company.ExtractData = []byte(extractData.String)
	}
	if stammkapital.Valid {
		company.Stammkapital = &stammkapital.Int64
	}
	if waehrung.Valid {
		company.Waehrung = &waehrung.String
	}
	if uid.Valid {
		company.UID = &uid.String
	}
	if gegenstand.Valid {
		company.Gegenstand = &gegenstand.String
	}
	if gruendungsdatum.Valid {
		company.Gruendungsdatum = &gruendungsdatum.Time
	}
	if lastFetchedAt.Valid {
		company.LastFetchedAt = &lastFetchedAt.Time
	}

	return &company, nil
}

// ListCompanies lists cached companies
func (r *Repository) ListCompanies(ctx context.Context, filter ListFilter) ([]*Company, int, error) {
	baseQuery := ` FROM firmenbuch_cache WHERE tenant_id = $1`
	args := []interface{}{filter.TenantID}
	argIdx := 2

	if filter.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.Search != nil {
		baseQuery += fmt.Sprintf(" AND (name ILIKE $%d OR fn ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+*filter.Search+"%")
		argIdx++
	}

	// Count total
	var total int
	countQuery := "SELECT COUNT(*)" + baseQuery
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count companies: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, fn, name, rechtsform, sitz, status, last_fetched_at, created_at, updated_at
		` + baseQuery + `
		ORDER BY name ASC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list companies: %w", err)
	}
	defer rows.Close()

	var companies []*Company
	for rows.Next() {
		var company Company
		var lastFetchedAt sql.NullTime

		err := rows.Scan(
			&company.ID, &company.TenantID, &company.FN, &company.Name, &company.Rechtsform, &company.Sitz,
			&company.Status, &lastFetchedAt, &company.CreatedAt, &company.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan company: %w", err)
		}

		if lastFetchedAt.Valid {
			company.LastFetchedAt = &lastFetchedAt.Time
		}

		companies = append(companies, &company)
	}

	return companies, total, nil
}

// AddToWatchlist adds a company to the watchlist
func (r *Repository) AddToWatchlist(ctx context.Context, entry *WatchlistEntry) (*WatchlistEntry, error) {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()
	entry.UpdatedAt = entry.CreatedAt

	query := `
		INSERT INTO firmenbuch_watchlist (
			id, tenant_id, company_id, fn, name, last_status, last_checked, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tenant_id, fn) DO UPDATE SET
			company_id = EXCLUDED.company_id,
			name = EXCLUDED.name,
			last_status = EXCLUDED.last_status,
			last_checked = EXCLUDED.last_checked,
			notes = EXCLUDED.notes,
			updated_at = EXCLUDED.updated_at
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		entry.ID, entry.TenantID, entry.CompanyID, entry.FN, entry.Name, entry.LastStatus,
		entry.LastChecked, entry.Notes, entry.CreatedAt, entry.UpdatedAt,
	).Scan(&entry.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to add to watchlist: %w", err)
	}

	return entry, nil
}

// GetWatchlistEntry retrieves a watchlist entry by FN
func (r *Repository) GetWatchlistEntry(ctx context.Context, tenantID uuid.UUID, fn string) (*WatchlistEntry, error) {
	query := `
		SELECT id, tenant_id, company_id, fn, name, last_status, last_checked, notes, created_at, updated_at
		FROM firmenbuch_watchlist
		WHERE tenant_id = $1 AND fn = $2`

	var entry WatchlistEntry
	var lastChecked sql.NullTime
	var notes sql.NullString

	err := r.db.QueryRow(ctx, query, tenantID, fn).Scan(
		&entry.ID, &entry.TenantID, &entry.CompanyID, &entry.FN, &entry.Name, &entry.LastStatus,
		&lastChecked, &notes, &entry.CreatedAt, &entry.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWatchlistNotFound
		}
		return nil, fmt.Errorf("failed to get watchlist entry: %w", err)
	}

	if lastChecked.Valid {
		entry.LastChecked = &lastChecked.Time
	}
	if notes.Valid {
		entry.Notes = &notes.String
	}

	return &entry, nil
}

// ListWatchlist lists all watchlist entries for a tenant
func (r *Repository) ListWatchlist(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*WatchlistEntry, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM firmenbuch_watchlist WHERE tenant_id = $1`
	err := r.db.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count watchlist: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, tenant_id, company_id, fn, name, last_status, last_checked, notes, created_at, updated_at
		FROM firmenbuch_watchlist
		WHERE tenant_id = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, selectQuery, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list watchlist: %w", err)
	}
	defer rows.Close()

	var entries []*WatchlistEntry
	for rows.Next() {
		var entry WatchlistEntry
		var lastChecked sql.NullTime
		var notes sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.CompanyID, &entry.FN, &entry.Name, &entry.LastStatus,
			&lastChecked, &notes, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan watchlist entry: %w", err)
		}

		if lastChecked.Valid {
			entry.LastChecked = &lastChecked.Time
		}
		if notes.Valid {
			entry.Notes = &notes.String
		}

		entries = append(entries, &entry)
	}

	return entries, total, nil
}

// UpdateWatchlistEntry updates a watchlist entry
func (r *Repository) UpdateWatchlistEntry(ctx context.Context, entry *WatchlistEntry) error {
	entry.UpdatedAt = time.Now()

	query := `
		UPDATE firmenbuch_watchlist SET
			last_status = $1,
			last_checked = $2,
			notes = $3,
			updated_at = $4
		WHERE id = $5 AND tenant_id = $6`

	result, err := r.db.Exec(ctx, query,
		entry.LastStatus, entry.LastChecked, entry.Notes, entry.UpdatedAt, entry.ID, entry.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to update watchlist entry: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}

	return nil
}

// RemoveFromWatchlist removes a company from the watchlist
func (r *Repository) RemoveFromWatchlist(ctx context.Context, tenantID uuid.UUID, fn string) error {
	query := `DELETE FROM firmenbuch_watchlist WHERE tenant_id = $1 AND fn = $2`
	result, err := r.db.Exec(ctx, query, tenantID, fn)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}
	return nil
}

// AddHistoryEntry adds a history entry for a company
func (r *Repository) AddHistoryEntry(ctx context.Context, entry *HistoryEntry) (*HistoryEntry, error) {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()

	query := `
		INSERT INTO firmenbuch_history (
			id, company_id, change_type, old_value, new_value, detected_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	err := r.db.QueryRow(ctx, query,
		entry.ID, entry.CompanyID, entry.ChangeType, entry.OldValue, entry.NewValue, entry.DetectedAt, entry.CreatedAt,
	).Scan(&entry.ID)

	if err != nil {
		return nil, fmt.Errorf("failed to add history entry: %w", err)
	}

	return entry, nil
}

// GetCompanyHistory retrieves history entries for a company
func (r *Repository) GetCompanyHistory(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]*HistoryEntry, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM firmenbuch_history WHERE company_id = $1`
	err := r.db.QueryRow(ctx, countQuery, companyID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count history: %w", err)
	}

	// Get paginated results
	selectQuery := `
		SELECT id, company_id, change_type, old_value, new_value, detected_at, created_at
		FROM firmenbuch_history
		WHERE company_id = $1
		ORDER BY detected_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, selectQuery, companyID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get history: %w", err)
	}
	defer rows.Close()

	var entries []*HistoryEntry
	for rows.Next() {
		var entry HistoryEntry
		var oldValue, newValue sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.CompanyID, &entry.ChangeType, &oldValue, &newValue, &entry.DetectedAt, &entry.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan history entry: %w", err)
		}

		if oldValue.Valid {
			entry.OldValue = []byte(oldValue.String)
		}
		if newValue.Valid {
			entry.NewValue = []byte(newValue.String)
		}

		entries = append(entries, &entry)
	}

	return entries, total, nil
}

// GetWatchlistEntriesForCheck returns watchlist entries that need checking
func (r *Repository) GetWatchlistEntriesForCheck(ctx context.Context, olderThan time.Time, limit int) ([]*WatchlistEntry, error) {
	query := `
		SELECT id, tenant_id, company_id, fn, name, last_status, last_checked, notes, created_at, updated_at
		FROM firmenbuch_watchlist
		WHERE last_checked IS NULL OR last_checked < $1
		ORDER BY last_checked ASC NULLS FIRST
		LIMIT $2`

	rows, err := r.db.Query(ctx, query, olderThan, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get watchlist entries: %w", err)
	}
	defer rows.Close()

	var entries []*WatchlistEntry
	for rows.Next() {
		var entry WatchlistEntry
		var lastChecked sql.NullTime
		var notes sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.CompanyID, &entry.FN, &entry.Name, &entry.LastStatus,
			&lastChecked, &notes, &entry.CreatedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan watchlist entry: %w", err)
		}

		if lastChecked.Valid {
			entry.LastChecked = &lastChecked.Time
		}
		if notes.Valid {
			entry.Notes = &notes.String
		}

		entries = append(entries, &entry)
	}

	return entries, nil
}
