package firmenbuch

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Search status constants
const (
	StatusPending   = "pending"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Company represents a cached company record
type Company struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	FN              string          `json:"fn"`
	Name            string          `json:"name"`
	Rechtsform      string          `json:"rechtsform"`
	Sitz            string          `json:"sitz"`
	Adresse         json.RawMessage `json:"adresse,omitempty"`
	Stammkapital    *int64          `json:"stammkapital,omitempty"`
	Waehrung        *string         `json:"waehrung,omitempty"`
	Status          string          `json:"status"`
	Gruendungsdatum *time.Time      `json:"gruendungsdatum,omitempty"`
	UID             *string         `json:"uid,omitempty"`
	Gegenstand      *string         `json:"gegenstand,omitempty"`
	ExtractData     json.RawMessage `json:"extract_data,omitempty"`
	LastFetchedAt   *time.Time      `json:"last_fetched_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// WatchlistEntry represents a company on the watchlist
type WatchlistEntry struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	CompanyID   uuid.UUID  `json:"company_id"`
	FN          string     `json:"fn"`
	Name        string     `json:"name"`
	LastStatus  string     `json:"last_status"`
	LastChecked *time.Time `json:"last_checked,omitempty"`
	Notes       *string    `json:"notes,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// HistoryEntry represents a change in company data
type HistoryEntry struct {
	ID         uuid.UUID       `json:"id"`
	CompanyID  uuid.UUID       `json:"company_id"`
	ChangeType string          `json:"change_type"`
	OldValue   json.RawMessage `json:"old_value,omitempty"`
	NewValue   json.RawMessage `json:"new_value,omitempty"`
	DetectedAt time.Time       `json:"detected_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

// SearchInput represents search parameters
type SearchInput struct {
	Name    string `json:"name,omitempty"`
	FN      string `json:"fn,omitempty"`
	Ort     string `json:"ort,omitempty"`
	MaxHits int    `json:"max_hits,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	FN         string `json:"fn"`
	Name       string `json:"name"`
	Rechtsform string `json:"rechtsform"`
	Sitz       string `json:"sitz"`
	Status     string `json:"status"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	Cached     bool           `json:"cached"`
}

// ExtractResponse represents a company extract response
type ExtractResponse struct {
	FN               string                `json:"fn"`
	Name             string                `json:"name"`
	Rechtsform       string                `json:"rechtsform"`
	Sitz             string                `json:"sitz"`
	Adresse          *AddressResponse      `json:"adresse,omitempty"`
	Stammkapital     *float64              `json:"stammkapital,omitempty"`
	Waehrung         *string               `json:"waehrung,omitempty"`
	Status           string                `json:"status"`
	Gruendungsdatum  *string               `json:"gruendungsdatum,omitempty"`
	LetzteAenderung  *string               `json:"letzte_aenderung,omitempty"`
	UID              *string               `json:"uid,omitempty"`
	Gegenstand       *string               `json:"gegenstand,omitempty"`
	Geschaeftsfuehrer []PersonResponse     `json:"geschaeftsfuehrer,omitempty"`
	Gesellschafter   []ShareholderResponse `json:"gesellschafter,omitempty"`
	LastFetchedAt    *string               `json:"last_fetched_at,omitempty"`
	Cached           bool                  `json:"cached"`
}

// AddressResponse represents an address in API responses
type AddressResponse struct {
	Strasse string `json:"strasse,omitempty"`
	PLZ     string `json:"plz,omitempty"`
	Ort     string `json:"ort,omitempty"`
	Land    string `json:"land,omitempty"`
}

// PersonResponse represents a person (e.g., Geschäftsführer) in API responses
type PersonResponse struct {
	Vorname        string  `json:"vorname"`
	Nachname       string  `json:"nachname"`
	Funktion       string  `json:"funktion"`
	VertretungsArt string  `json:"vertretungsart,omitempty"`
	Seit           *string `json:"seit,omitempty"`
	Bis            *string `json:"bis,omitempty"`
}

// ShareholderResponse represents a shareholder in API responses
type ShareholderResponse struct {
	Name         string   `json:"name"`
	FN           *string  `json:"fn,omitempty"`
	AnteilPct    float64  `json:"anteil_pct"`
	Stammeinlage *float64 `json:"stammeinlage,omitempty"`
	Seit         *string  `json:"seit,omitempty"`
}

// WatchlistInput represents input for adding to watchlist
type WatchlistInput struct {
	FN    string  `json:"fn"`
	Notes *string `json:"notes,omitempty"`
}

// WatchlistResponse represents a watchlist entry response
type WatchlistResponse struct {
	ID          uuid.UUID `json:"id"`
	FN          string    `json:"fn"`
	Name        string    `json:"name"`
	LastStatus  string    `json:"last_status"`
	LastChecked *string   `json:"last_checked,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   string    `json:"created_at"`
}

// HistoryResponse represents a history entry response
type HistoryResponse struct {
	ID         uuid.UUID       `json:"id"`
	ChangeType string          `json:"change_type"`
	OldValue   json.RawMessage `json:"old_value,omitempty"`
	NewValue   json.RawMessage `json:"new_value,omitempty"`
	DetectedAt string          `json:"detected_at"`
}

// ListFilter represents filtering options
type ListFilter struct {
	TenantID uuid.UUID
	Status   *string
	Search   *string
	Limit    int
	Offset   int
}
