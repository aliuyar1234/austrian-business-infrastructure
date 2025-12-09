package eldameldung

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReferenceDataService provides access to ELDA reference data
type ReferenceDataService struct {
	db *pgxpool.Pool
}

// NewReferenceDataService creates a new reference data service
func NewReferenceDataService(db *pgxpool.Pool) *ReferenceDataService {
	return &ReferenceDataService{db: db}
}

// KollektivvertragInfo contains KV information
type KollektivvertragInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Branche     string `json:"branche,omitempty"`
	WKONummer   string `json:"wko_nummer,omitempty"`
	GueltigAb   string `json:"gueltig_ab,omitempty"`
	GueltigBis  string `json:"gueltig_bis,omitempty"`
}

// SearchKollektivvertraege searches for KV codes by name or code
func (s *ReferenceDataService) SearchKollektivvertraege(ctx context.Context, query string, limit int) ([]KollektivvertragInfo, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	sql := `
		SELECT code, name, branche, wko_nummer, gueltig_ab, gueltig_bis
		FROM kollektivvertraege
		WHERE code ILIKE $1 OR name ILIKE $1 OR branche ILIKE $1
		ORDER BY name
		LIMIT $2
	`

	searchPattern := "%" + query + "%"

	rows, err := s.db.Query(ctx, sql, searchPattern, limit)
	if err != nil {
		// Return hardcoded defaults if table doesn't exist
		return s.getDefaultKollektivvertraege(query, limit), nil
	}
	defer rows.Close()

	var results []KollektivvertragInfo
	for rows.Next() {
		var kv KollektivvertragInfo
		var gueltigAb, gueltigBis *string
		if err := rows.Scan(&kv.Code, &kv.Name, &kv.Branche, &kv.WKONummer, &gueltigAb, &gueltigBis); err != nil {
			continue
		}
		if gueltigAb != nil {
			kv.GueltigAb = *gueltigAb
		}
		if gueltigBis != nil {
			kv.GueltigBis = *gueltigBis
		}
		results = append(results, kv)
	}

	if len(results) == 0 {
		return s.getDefaultKollektivvertraege(query, limit), nil
	}

	return results, nil
}

// getDefaultKollektivvertraege returns common KV codes
func (s *ReferenceDataService) getDefaultKollektivvertraege(query string, limit int) []KollektivvertragInfo {
	defaults := []KollektivvertragInfo{
		{Code: "HANDEL", Name: "Handelsangestellte", Branche: "Handel"},
		{Code: "IT", Name: "IT-Kollektivvertrag", Branche: "IT/EDV"},
		{Code: "METALL", Name: "Metallindustrie", Branche: "Industrie"},
		{Code: "ELEKTRO", Name: "Elektroindustrie", Branche: "Industrie"},
		{Code: "GEWERBE", Name: "Gewerbliche Angestellte", Branche: "Gewerbe"},
		{Code: "BANK", Name: "Banken und Sparkassen", Branche: "Finanz"},
		{Code: "VERSICHERUNG", Name: "Versicherungen", Branche: "Finanz"},
		{Code: "GASTRO", Name: "Hotel- und Gastgewerbe", Branche: "Tourismus"},
		{Code: "BAU", Name: "Bauindustrie", Branche: "Bau"},
		{Code: "TRANSPORT", Name: "Transport und Verkehr", Branche: "Transport"},
		{Code: "CHEMIE", Name: "Chemische Industrie", Branche: "Industrie"},
		{Code: "PAPIER", Name: "Papierindustrie", Branche: "Industrie"},
		{Code: "TEXTIL", Name: "Textilindustrie", Branche: "Industrie"},
		{Code: "HOLZ", Name: "Holzindustrie", Branche: "Industrie"},
		{Code: "GEMEINDE", Name: "Gemeindebedienstete", Branche: "Öffentlich"},
	}

	if query == "" {
		if len(defaults) > limit {
			return defaults[:limit]
		}
		return defaults
	}

	query = strings.ToLower(query)
	var filtered []KollektivvertragInfo
	for _, kv := range defaults {
		if strings.Contains(strings.ToLower(kv.Code), query) ||
			strings.Contains(strings.ToLower(kv.Name), query) ||
			strings.Contains(strings.ToLower(kv.Branche), query) {
			filtered = append(filtered, kv)
			if len(filtered) >= limit {
				break
			}
		}
	}

	return filtered
}

// ArbeitszeitCodeInfo contains working time code information
type ArbeitszeitCodeInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GetArbeitszeitCodes returns all working time codes
func (s *ReferenceDataService) GetArbeitszeitCodes(ctx context.Context) []ArbeitszeitCodeInfo {
	// Standard ELDA Arbeitszeit-Codes
	return []ArbeitszeitCodeInfo{
		{Code: "VZ", Name: "Vollzeit", Description: "Normalarbeitszeit gemäß KV"},
		{Code: "TZ", Name: "Teilzeit", Description: "Weniger als Normalarbeitszeit"},
		{Code: "GF", Name: "Geringfügig", Description: "Unter Geringfügigkeitsgrenze"},
		{Code: "SC", Name: "Schichtarbeit", Description: "Wechselschichtdienst"},
		{Code: "NA", Name: "Nachtarbeit", Description: "Überwiegend Nachtarbeit"},
		{Code: "SA", Name: "Samstagsarbeit", Description: "Regelmäßige Samstagsarbeit"},
		{Code: "SO", Name: "Sonntagsarbeit", Description: "Regelmäßige Sonntagsarbeit"},
		{Code: "FE", Name: "Feiertagsarbeit", Description: "Regelmäßige Feiertagsarbeit"},
		{Code: "BE", Name: "Bereitschaft", Description: "Bereitschaftsdienst"},
		{Code: "RU", Name: "Rufbereitschaft", Description: "Rufbereitschaft"},
		{Code: "GL", Name: "Gleitzeit", Description: "Gleitende Arbeitszeit"},
		{Code: "HO", Name: "Home Office", Description: "Telearbeit/Home Office"},
	}
}

// BeitragsgruppenInfo contains contribution group information
type BeitragsgruppenInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	SVSatz      string `json:"sv_satz,omitempty"` // SV contribution rate
}

// SearchBeitragsgruppen searches for contribution groups
func (s *ReferenceDataService) SearchBeitragsgruppen(ctx context.Context, query string, limit int) ([]BeitragsgruppenInfo, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// Try database first
	sql := `
		SELECT code, name, description, sv_satz
		FROM beitragsgruppen
		WHERE code ILIKE $1 OR name ILIKE $1
		ORDER BY code
		LIMIT $2
	`

	searchPattern := "%" + query + "%"

	rows, err := s.db.Query(ctx, sql, searchPattern, limit)
	if err != nil {
		// Return defaults if table doesn't exist
		return s.getDefaultBeitragsgruppen(query, limit), nil
	}
	defer rows.Close()

	var results []BeitragsgruppenInfo
	for rows.Next() {
		var bg BeitragsgruppenInfo
		if err := rows.Scan(&bg.Code, &bg.Name, &bg.Description, &bg.SVSatz); err != nil {
			continue
		}
		results = append(results, bg)
	}

	if len(results) == 0 {
		return s.getDefaultBeitragsgruppen(query, limit), nil
	}

	return results, nil
}

// getDefaultBeitragsgruppen returns standard contribution groups
func (s *ReferenceDataService) getDefaultBeitragsgruppen(query string, limit int) []BeitragsgruppenInfo {
	defaults := []BeitragsgruppenInfo{
		{Code: "A1", Name: "Angestellte Vollversicherung", Description: "Kranken-, Unfall-, Pensions- und Arbeitslosenversicherung", SVSatz: "39.35%"},
		{Code: "A2", Name: "Angestellte ohne AV", Description: "Kranken-, Unfall- und Pensionsversicherung", SVSatz: "33.35%"},
		{Code: "D1", Name: "Arbeiter Vollversicherung", Description: "Kranken-, Unfall-, Pensions- und Arbeitslosenversicherung", SVSatz: "39.35%"},
		{Code: "D2", Name: "Arbeiter ohne AV", Description: "Kranken-, Unfall- und Pensionsversicherung", SVSatz: "33.35%"},
		{Code: "G1", Name: "Geringfügig Beschäftigte", Description: "Nur Unfallversicherung", SVSatz: "1.2%"},
		{Code: "L1", Name: "Lehrlinge 1. Lehrjahr", Description: "Vollversicherung für Lehrlinge", SVSatz: "0%"},
		{Code: "L2", Name: "Lehrlinge 2. Lehrjahr", Description: "Vollversicherung für Lehrlinge", SVSatz: "0%"},
		{Code: "L3", Name: "Lehrlinge 3./4. Lehrjahr", Description: "Vollversicherung für Lehrlinge", SVSatz: "0%"},
		{Code: "P1", Name: "Praktikanten", Description: "Praktikantenversicherung", SVSatz: "25%"},
		{Code: "F1", Name: "Freie Dienstnehmer", Description: "Freie Dienstverträge", SVSatz: "39.35%"},
	}

	if query == "" {
		if len(defaults) > limit {
			return defaults[:limit]
		}
		return defaults
	}

	query = strings.ToLower(query)
	var filtered []BeitragsgruppenInfo
	for _, bg := range defaults {
		if strings.Contains(strings.ToLower(bg.Code), query) ||
			strings.Contains(strings.ToLower(bg.Name), query) {
			filtered = append(filtered, bg)
			if len(filtered) >= limit {
				break
			}
		}
	}

	return filtered
}

// AustrittGrundInfo contains exit reason information
type AustrittGrundInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GetAustrittGruende returns all exit reasons
func (s *ReferenceDataService) GetAustrittGruende(ctx context.Context) []AustrittGrundInfo {
	return []AustrittGrundInfo{
		{Code: "K", Name: "Kündigung durch Arbeitgeber", Description: "Arbeitgeberseitige Kündigung"},
		{Code: "KA", Name: "Kündigung durch Arbeitnehmer", Description: "Arbeitnehmerseitige Kündigung"},
		{Code: "E", Name: "Einvernehmliche Auflösung", Description: "Einvernehmliche Beendigung des Dienstverhältnisses"},
		{Code: "EN", Name: "Entlassung", Description: "Fristlose Entlassung durch Arbeitgeber"},
		{Code: "A", Name: "Vorzeitiger Austritt", Description: "Berechtigter vorzeitiger Austritt"},
		{Code: "B", Name: "Befristung", Description: "Ablauf eines befristeten Dienstverhältnisses"},
		{Code: "P", Name: "Pension", Description: "Pensionsantritt"},
		{Code: "T", Name: "Tod", Description: "Tod des Dienstnehmers"},
		{Code: "PZ", Name: "Probezeit", Description: "Auflösung während Probezeit"},
		{Code: "I", Name: "Insolvenz", Description: "Insolvenz des Arbeitgebers"},
	}
}

// AenderungArtInfo contains change type information
type AenderungArtInfo struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// GetAenderungArten returns all change types for Änderungsmeldungen
func (s *ReferenceDataService) GetAenderungArten(ctx context.Context) []AenderungArtInfo {
	return []AenderungArtInfo{
		{Code: "ENTGELT", Name: "Entgeltänderung", Description: "Änderung des Bruttoentgelts"},
		{Code: "ARBEITSZEIT", Name: "Arbeitszeitänderung", Description: "Änderung der Wochenstunden"},
		{Code: "TAETIGKEIT", Name: "Tätigkeitsänderung", Description: "Änderung der Tätigkeit/Position"},
		{Code: "ADRESSE", Name: "Adressänderung", Description: "Änderung der Wohnadresse"},
		{Code: "EINSTUFUNG", Name: "Einstufungsänderung", Description: "Änderung der Einstufung/Verwendungsgruppe"},
		{Code: "KOLLEKTIV", Name: "KV-Änderung", Description: "Änderung des Kollektivvertrags"},
		{Code: "BEITRAGSGRUPPE", Name: "Beitragsgruppenänderung", Description: "Änderung der Beitragsgruppe"},
		{Code: "BANK", Name: "Bankverbindung", Description: "Änderung der Bankverbindung"},
		{Code: "NAME", Name: "Namensänderung", Description: "Änderung des Namens"},
		{Code: "DIENSTORT", Name: "Dienstortänderung", Description: "Änderung des Dienstorts"},
	}
}
