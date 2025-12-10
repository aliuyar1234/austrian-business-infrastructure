package firmenbuch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"austrian-business-infrastructure/internal/fb"
	"github.com/google/uuid"
)

var (
	ErrInvalidFN        = errors.New("invalid Firmenbuch number format")
	ErrSearchEmpty      = errors.New("at least one search parameter required")
	ErrAlreadyOnWatch   = errors.New("company already on watchlist")
)

// CacheDuration is how long to use cached data before refreshing
const CacheDuration = 24 * time.Hour

// Service handles firmenbuch business logic
type Service struct {
	repo   *Repository
	client *fb.Client
}

// NewService creates a new firmenbuch service
func NewService(repo *Repository, client *fb.Client) *Service {
	return &Service{
		repo:   repo,
		client: client,
	}
}

// Search searches for companies by name, FN, or location
func (s *Service) Search(ctx context.Context, tenantID uuid.UUID, input *SearchInput) (*SearchResponse, error) {
	if input.Name == "" && input.FN == "" && input.Ort == "" {
		return nil, ErrSearchEmpty
	}

	// If searching by FN, validate format
	if input.FN != "" {
		if err := fb.ValidateFN(input.FN); err != nil {
			return nil, ErrInvalidFN
		}
	}

	// Build search request
	req := &fb.FBSearchRequest{
		Name:    input.Name,
		FN:      input.FN,
		Ort:     input.Ort,
		MaxHits: input.MaxHits,
	}
	if req.MaxHits <= 0 {
		req.MaxHits = 20
	}

	// Call Firmenbuch API
	fbResp, err := s.client.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results
	results := make([]SearchResult, 0, len(fbResp.Results))
	for _, r := range fbResp.Results {
		results = append(results, SearchResult{
			FN:         r.FN,
			Name:       r.Firma,
			Rechtsform: string(r.Rechtsform),
			Sitz:       r.Sitz,
			Status:     string(r.Status),
		})
	}

	return &SearchResponse{
		Results:    results,
		TotalCount: fbResp.TotalCount,
		Cached:     false,
	}, nil
}

// GetExtract retrieves a company extract by FN
func (s *Service) GetExtract(ctx context.Context, tenantID uuid.UUID, fn string, forceRefresh bool) (*ExtractResponse, error) {
	// Validate FN format
	if err := fb.ValidateFN(fn); err != nil {
		return nil, ErrInvalidFN
	}

	// Check cache first
	if !forceRefresh {
		cached, err := s.repo.GetCompanyByFN(ctx, tenantID, fn)
		if err == nil && cached.LastFetchedAt != nil {
			if time.Since(*cached.LastFetchedAt) < CacheDuration {
				return s.companyToExtractResponse(cached, true), nil
			}
		}
	}

	// Fetch from Firmenbuch API
	extract, err := s.client.Extract(fn)
	if err != nil {
		return nil, fmt.Errorf("extract failed: %w", err)
	}

	// Build company record
	company := s.extractToCompany(tenantID, extract)

	// Store/update in cache
	savedCompany, err := s.repo.CreateCompany(ctx, company)
	if err != nil {
		// Log but don't fail - we still have the data
		fmt.Printf("failed to cache company: %v\n", err)
		return s.extractToResponse(extract, false), nil
	}

	return s.companyToExtractResponse(savedCompany, false), nil
}

// extractToCompany converts a Firmenbuch extract to a company record
func (s *Service) extractToCompany(tenantID uuid.UUID, extract *fb.FBExtract) *Company {
	now := time.Now()

	// Serialize address
	addrJSON, _ := json.Marshal(map[string]string{
		"strasse": extract.Adresse.Strasse,
		"plz":     extract.Adresse.PLZ,
		"ort":     extract.Adresse.Ort,
		"land":    extract.Adresse.Land,
	})

	// Serialize full extract data
	extractJSON, _ := json.Marshal(extract)

	company := &Company{
		TenantID:      tenantID,
		FN:            extract.FN,
		Name:          extract.Firma,
		Rechtsform:    string(extract.Rechtsform),
		Sitz:          extract.Sitz,
		Adresse:       addrJSON,
		Status:        string(extract.Status),
		ExtractData:   extractJSON,
		LastFetchedAt: &now,
	}

	if extract.Stammkapital > 0 {
		company.Stammkapital = &extract.Stammkapital
	}
	if extract.Waehrung != "" {
		company.Waehrung = &extract.Waehrung
	}
	if !extract.Gruendungsdatum.IsZero() {
		company.Gruendungsdatum = &extract.Gruendungsdatum
	}
	if extract.UID != "" {
		company.UID = &extract.UID
	}
	if extract.Gegenstand != "" {
		company.Gegenstand = &extract.Gegenstand
	}

	return company
}

// extractToResponse converts a Firmenbuch extract to API response
func (s *Service) extractToResponse(extract *fb.FBExtract, cached bool) *ExtractResponse {
	resp := &ExtractResponse{
		FN:         extract.FN,
		Name:       extract.Firma,
		Rechtsform: string(extract.Rechtsform),
		Sitz:       extract.Sitz,
		Status:     string(extract.Status),
		Cached:     cached,
	}

	resp.Adresse = &AddressResponse{
		Strasse: extract.Adresse.Strasse,
		PLZ:     extract.Adresse.PLZ,
		Ort:     extract.Adresse.Ort,
		Land:    extract.Adresse.Land,
	}

	if extract.Stammkapital > 0 {
		stammkapital := extract.StammkapitalEUR()
		resp.Stammkapital = &stammkapital
	}
	if extract.Waehrung != "" {
		resp.Waehrung = &extract.Waehrung
	}
	if !extract.Gruendungsdatum.IsZero() {
		d := extract.GruendungsdatumString()
		resp.Gruendungsdatum = &d
	}
	if !extract.LetzteAenderung.IsZero() {
		d := extract.LetzteAenderungString()
		resp.LetzteAenderung = &d
	}
	if extract.UID != "" {
		resp.UID = &extract.UID
	}
	if extract.Gegenstand != "" {
		resp.Gegenstand = &extract.Gegenstand
	}

	// Convert Geschäftsführer
	for _, gf := range extract.Geschaeftsfuehrer {
		person := PersonResponse{
			Vorname:        gf.Vorname,
			Nachname:       gf.Nachname,
			Funktion:       string(gf.Funktion),
			VertretungsArt: string(gf.VertretungsArt),
		}
		if !gf.Seit.IsZero() {
			d := gf.SeitString()
			person.Seit = &d
		}
		if gf.Bis != nil && !gf.Bis.IsZero() {
			d := gf.Bis.Format("2006-01-02")
			person.Bis = &d
		}
		resp.Geschaeftsfuehrer = append(resp.Geschaeftsfuehrer, person)
	}

	// Convert Gesellschafter
	for _, gs := range extract.Gesellschafter {
		shareholder := ShareholderResponse{
			Name:      gs.Name,
			AnteilPct: gs.AnteilProzent(),
		}
		if gs.FN != "" {
			shareholder.FN = &gs.FN
		}
		if gs.Stammeinlage > 0 {
			einlage := gs.StammeinlageEUR()
			shareholder.Stammeinlage = &einlage
		}
		if !gs.Seit.IsZero() {
			d := gs.Seit.Format("2006-01-02")
			shareholder.Seit = &d
		}
		resp.Gesellschafter = append(resp.Gesellschafter, shareholder)
	}

	return resp
}

// companyToExtractResponse converts a cached company to API response
func (s *Service) companyToExtractResponse(company *Company, cached bool) *ExtractResponse {
	resp := &ExtractResponse{
		FN:         company.FN,
		Name:       company.Name,
		Rechtsform: company.Rechtsform,
		Sitz:       company.Sitz,
		Status:     company.Status,
		Cached:     cached,
	}

	// Parse address
	if len(company.Adresse) > 0 {
		var addr AddressResponse
		if json.Unmarshal(company.Adresse, &addr) == nil {
			resp.Adresse = &addr
		}
	}

	if company.Stammkapital != nil {
		stammkapital := float64(*company.Stammkapital) / 100
		resp.Stammkapital = &stammkapital
	}
	if company.Waehrung != nil {
		resp.Waehrung = company.Waehrung
	}
	if company.Gruendungsdatum != nil {
		d := company.Gruendungsdatum.Format("2006-01-02")
		resp.Gruendungsdatum = &d
	}
	if company.UID != nil {
		resp.UID = company.UID
	}
	if company.Gegenstand != nil {
		resp.Gegenstand = company.Gegenstand
	}
	if company.LastFetchedAt != nil {
		d := company.LastFetchedAt.Format("2006-01-02T15:04:05Z")
		resp.LastFetchedAt = &d
	}

	// Parse full extract data for persons/shareholders
	if len(company.ExtractData) > 0 {
		var extract fb.FBExtract
		if json.Unmarshal(company.ExtractData, &extract) == nil {
			for _, gf := range extract.Geschaeftsfuehrer {
				person := PersonResponse{
					Vorname:        gf.Vorname,
					Nachname:       gf.Nachname,
					Funktion:       string(gf.Funktion),
					VertretungsArt: string(gf.VertretungsArt),
				}
				if !gf.Seit.IsZero() {
					d := gf.SeitString()
					person.Seit = &d
				}
				resp.Geschaeftsfuehrer = append(resp.Geschaeftsfuehrer, person)
			}

			for _, gs := range extract.Gesellschafter {
				shareholder := ShareholderResponse{
					Name:      gs.Name,
					AnteilPct: gs.AnteilProzent(),
				}
				if gs.FN != "" {
					shareholder.FN = &gs.FN
				}
				if gs.Stammeinlage > 0 {
					einlage := gs.StammeinlageEUR()
					shareholder.Stammeinlage = &einlage
				}
				resp.Gesellschafter = append(resp.Gesellschafter, shareholder)
			}
		}
	}

	return resp
}

// ValidateFN validates a Firmenbuch number format
func (s *Service) ValidateFN(fn string) error {
	return fb.ValidateFN(fn)
}

// AddToWatchlist adds a company to the watchlist
func (s *Service) AddToWatchlist(ctx context.Context, tenantID uuid.UUID, input *WatchlistInput) (*WatchlistEntry, error) {
	// Validate FN format
	if err := fb.ValidateFN(input.FN); err != nil {
		return nil, ErrInvalidFN
	}

	// Get or create company record
	company, err := s.repo.GetCompanyByFN(ctx, tenantID, input.FN)
	if err != nil {
		if errors.Is(err, ErrCompanyNotFound) {
			// Fetch from Firmenbuch first
			extract, fetchErr := s.client.Extract(input.FN)
			if fetchErr != nil {
				return nil, fmt.Errorf("failed to fetch company: %w", fetchErr)
			}

			companyRec := s.extractToCompany(tenantID, extract)
			company, err = s.repo.CreateCompany(ctx, companyRec)
			if err != nil {
				return nil, fmt.Errorf("failed to create company: %w", err)
			}
		} else {
			return nil, err
		}
	}

	// Create watchlist entry
	entry := &WatchlistEntry{
		TenantID:   tenantID,
		CompanyID:  company.ID,
		FN:         company.FN,
		Name:       company.Name,
		LastStatus: company.Status,
		Notes:      input.Notes,
	}

	return s.repo.AddToWatchlist(ctx, entry)
}

// ListWatchlist lists all companies on the watchlist
func (s *Service) ListWatchlist(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*WatchlistEntry, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListWatchlist(ctx, tenantID, limit, offset)
}

// RemoveFromWatchlist removes a company from the watchlist
func (s *Service) RemoveFromWatchlist(ctx context.Context, tenantID uuid.UUID, fn string) error {
	if err := fb.ValidateFN(fn); err != nil {
		return ErrInvalidFN
	}
	return s.repo.RemoveFromWatchlist(ctx, tenantID, fn)
}

// GetCompanyHistory retrieves change history for a company
func (s *Service) GetCompanyHistory(ctx context.Context, tenantID uuid.UUID, fn string, limit, offset int) ([]*HistoryEntry, int, error) {
	if err := fb.ValidateFN(fn); err != nil {
		return nil, 0, ErrInvalidFN
	}

	company, err := s.repo.GetCompanyByFN(ctx, tenantID, fn)
	if err != nil {
		return nil, 0, err
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	return s.repo.GetCompanyHistory(ctx, company.ID, limit, offset)
}

// CheckWatchlist checks all watchlist entries for changes
func (s *Service) CheckWatchlist(ctx context.Context, olderThan time.Duration, batchSize int) (int, error) {
	cutoff := time.Now().Add(-olderThan)
	entries, err := s.repo.GetWatchlistEntriesForCheck(ctx, cutoff, batchSize)
	if err != nil {
		return 0, err
	}

	checked := 0
	for _, entry := range entries {
		if err := s.checkWatchlistEntry(ctx, entry); err != nil {
			// Log but continue
			fmt.Printf("failed to check %s: %v\n", entry.FN, err)
			continue
		}
		checked++
	}

	return checked, nil
}

// checkWatchlistEntry checks a single watchlist entry for changes
func (s *Service) checkWatchlistEntry(ctx context.Context, entry *WatchlistEntry) error {
	// Fetch current data
	extract, err := s.client.Extract(entry.FN)
	if err != nil {
		return err
	}

	// Get previous company data
	oldCompany, err := s.repo.GetCompanyByID(ctx, entry.CompanyID, entry.TenantID)
	if err != nil {
		return err
	}

	// Check for status change
	newStatus := string(extract.Status)
	if oldCompany.Status != newStatus {
		// Record change
		historyEntry := &HistoryEntry{
			CompanyID:  entry.CompanyID,
			ChangeType: "status_change",
			DetectedAt: time.Now(),
		}
		historyEntry.OldValue, _ = json.Marshal(map[string]string{"status": oldCompany.Status})
		historyEntry.NewValue, _ = json.Marshal(map[string]string{"status": newStatus})

		if _, err := s.repo.AddHistoryEntry(ctx, historyEntry); err != nil {
			return err
		}
	}

	// Update company cache
	updatedCompany := s.extractToCompany(entry.TenantID, extract)
	updatedCompany.ID = oldCompany.ID
	if _, err := s.repo.CreateCompany(ctx, updatedCompany); err != nil {
		return err
	}

	// Update watchlist entry
	now := time.Now()
	entry.LastStatus = newStatus
	entry.LastChecked = &now
	return s.repo.UpdateWatchlistEntry(ctx, entry)
}

// ListCachedCompanies lists cached companies
func (s *Service) ListCachedCompanies(ctx context.Context, filter ListFilter) ([]*Company, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	return s.repo.ListCompanies(ctx, filter)
}
