package eldameldung

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/elda"
)

// Service handles ELDA meldung business logic
type Service struct {
	repo      *Repository
	client    *elda.Client
	validator *Validator
}

// NewService creates a new ELDA meldung service
func NewService(pool *pgxpool.Pool, eldaClient *elda.Client) *Service {
	return &Service{
		repo:      NewRepository(pool),
		client:    eldaClient,
		validator: NewValidator(),
	}
}

// Create creates a new ELDA meldung
func (s *Service) Create(ctx context.Context, req *elda.MeldungCreateRequest) (*elda.ELDAMeldung, error) {
	// Validate the request
	validation := s.validator.ValidateCreateRequest(req)
	if !validation.Valid {
		return nil, &ValidationError{
			Message: "Validierungsfehler",
			Errors:  validation.ErrorMessages(),
		}
	}

	// Create the meldung entity
	meldung := &elda.ELDAMeldung{
		ID:            uuid.New(),
		ELDAAccountID: req.ELDAAccountID,
		Type:          req.Type,
		Status:        elda.MeldungStatusDraft,
		SVNummer:      req.SVNummer,
		Vorname:       req.Vorname,
		Nachname:      req.Nachname,
		Geschlecht:    req.Geschlecht,
		Beschaeftigung: req.Beschaeftigung,
		Arbeitszeit:    req.Arbeitszeit,
		Entgelt:        req.Entgelt,
		Adresse:        req.Adresse,
		Bankverbindung: req.Bankverbindung,
		AustrittGrund:  req.AustrittGrund,
		Abfertigung:    req.Abfertigung,
		Urlaubsersatz:  req.Urlaubsersatz,
		URLTage:        req.URLTage,
		AenderungArt:   req.AenderungArt,
		OriginalMeldungID: req.OriginalMeldungID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Parse dates
	if req.Geburtsdatum != "" {
		if t, err := time.Parse("2006-01-02", req.Geburtsdatum); err == nil {
			meldung.Geburtsdatum = &t
		}
	}
	if req.Eintrittsdatum != "" {
		if t, err := time.Parse("2006-01-02", req.Eintrittsdatum); err == nil {
			meldung.Eintrittsdatum = &t
		}
	}
	if req.Austrittsdatum != "" {
		if t, err := time.Parse("2006-01-02", req.Austrittsdatum); err == nil {
			meldung.Austrittsdatum = &t
		}
	}
	if req.AenderungDatum != "" {
		if t, err := time.Parse("2006-01-02", req.AenderungDatum); err == nil {
			meldung.AenderungDatum = &t
		}
	}

	// Save to database
	if err := s.repo.Create(ctx, meldung); err != nil {
		return nil, fmt.Errorf("failed to create meldung: %w", err)
	}

	return meldung, nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
	Errors  []string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Get retrieves a meldung by ID
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*elda.ELDAMeldung, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves meldungen with filters
func (s *Service) List(ctx context.Context, filter ListFilter) ([]*elda.ELDAMeldung, error) {
	return s.repo.List(ctx, filter)
}

// Count returns the count of meldungen matching the filter
func (s *Service) Count(ctx context.Context, filter ListFilter) (int, error) {
	return s.repo.Count(ctx, filter)
}

// Validate validates a meldung
func (s *Service) Validate(ctx context.Context, id uuid.UUID) (*ValidationResult, error) {
	meldung, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	result := s.validator.ValidateMeldung(meldung)

	// Update status if valid
	if result.Valid && meldung.Status == elda.MeldungStatusDraft {
		meldung.Status = elda.MeldungStatusValidated
		meldung.UpdatedAt = time.Now()
		if err := s.repo.Update(ctx, meldung); err != nil {
			return nil, fmt.Errorf("failed to update meldung status: %w", err)
		}
	}

	return result, nil
}

// CredentialsProvider provides ELDA credentials for an account
type CredentialsProvider interface {
	GetCredentials(ctx context.Context, accountID uuid.UUID) (*elda.ELDACredentials, error)
}

// Submit submits a meldung to ELDA
func (s *Service) Submit(ctx context.Context, id uuid.UUID) (*SubmitResult, error) {
	return s.SubmitWithCredentials(ctx, id, nil)
}

// SubmitWithCredentials submits a meldung to ELDA with provided credentials
func (s *Service) SubmitWithCredentials(ctx context.Context, id uuid.UUID, creds *elda.ELDACredentials) (*SubmitResult, error) {
	meldung, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate first
	validation := s.validator.ValidateMeldung(meldung)
	if !validation.Valid {
		return nil, &ValidationError{
			Message: "Meldung ist nicht valide",
			Errors:  validation.ErrorMessages(),
		}
	}

	// Build XML document for preview/storage
	xmlDoc, err := s.buildXML(meldung)
	if err != nil {
		return nil, fmt.Errorf("failed to build XML: %w", err)
	}

	// Store request XML
	meldung.RequestXML = string(xmlDoc)
	meldung.UpdatedAt = time.Now()

	// If no credentials provided, use a default
	if creds == nil {
		creds = &elda.ELDACredentials{
			DienstgeberNr: "", // Must be set by caller or provider
		}
	}

	// Submit to ELDA using the extended submission
	resp, err := s.client.SubmitExtendedMeldung(ctx, creds, meldung)

	result := &SubmitResult{
		SubmittedAt: time.Now(),
	}

	if resp != nil {
		result.Success = resp.Erfolg
		if len(resp.Warnungen) > 0 {
			result.Warnings = resp.Warnungen
		}
	}

	if err != nil || (resp != nil && !resp.Erfolg) {
		meldung.Status = elda.MeldungStatusRejected
		if resp != nil {
			meldung.ErrorCode = resp.ErrorCode
			meldung.ErrorMessage = resp.ErrorMessage
		}
		if err != nil {
			meldung.ErrorMessage = err.Error()
		}
		result.ErrorCode = meldung.ErrorCode
		result.ErrorMessage = meldung.ErrorMessage
		result.Success = false
	} else if resp != nil {
		meldung.Status = elda.MeldungStatusSubmitted
		meldung.Protokollnummer = resp.Protokollnummer
		now := time.Now()
		meldung.SubmittedAt = &now
		result.Protokollnummer = resp.Protokollnummer
		result.Success = true
	}

	// Update database
	if updateErr := s.repo.Update(ctx, meldung); updateErr != nil {
		return result, fmt.Errorf("submitted but failed to update record: %w", updateErr)
	}

	if !result.Success {
		return result, fmt.Errorf("ELDA rejected meldung: %s - %s", result.ErrorCode, result.ErrorMessage)
	}

	return result, nil
}

// SubmitResult contains the result of a meldung submission
type SubmitResult struct {
	Success         bool      `json:"success"`
	Protokollnummer string    `json:"protokollnummer,omitempty"`
	ErrorCode       string    `json:"error_code,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	Warnings        []string  `json:"warnings,omitempty"`
	SubmittedAt     time.Time `json:"submitted_at"`
}

// Preview generates an XML preview without submitting
func (s *Service) Preview(ctx context.Context, id uuid.UUID) (*PreviewResult, error) {
	meldung, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	xmlDoc, err := s.buildXML(meldung)
	if err != nil {
		return nil, err
	}

	return &PreviewResult{
		XML:       string(xmlDoc),
		Type:      meldung.Type,
		SVNummer:  meldung.SVNummer,
		Name:      fmt.Sprintf("%s %s", meldung.Vorname, meldung.Nachname),
		GeneratedAt: time.Now(),
	}, nil
}

// PreviewResult contains the XML preview
type PreviewResult struct {
	XML         string          `json:"xml"`
	Type        elda.MeldungType `json:"type"`
	SVNummer    string          `json:"sv_nummer"`
	Name        string          `json:"name"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// Delete deletes a meldung (only if draft)
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	meldung, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if meldung.Status != elda.MeldungStatusDraft {
		return fmt.Errorf("kann nur Entwürfe löschen, aktueller Status: %s", meldung.Status)
	}

	return s.repo.Delete(ctx, id)
}

// GetHistory returns the meldung history for an SV-Nummer
func (s *Service) GetHistory(ctx context.Context, accountID uuid.UUID, svNummer string) ([]*elda.ELDAMeldung, error) {
	return s.repo.GetHistoryBySVNummer(ctx, accountID, svNummer)
}

// Retry retries a rejected meldung
func (s *Service) Retry(ctx context.Context, id uuid.UUID) (*SubmitResult, error) {
	meldung, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if meldung.Status != elda.MeldungStatusRejected {
		return nil, fmt.Errorf("kann nur abgelehnte Meldungen erneut senden")
	}

	// Reset status and clear errors
	meldung.Status = elda.MeldungStatusValidated
	meldung.ErrorCode = ""
	meldung.ErrorMessage = ""
	meldung.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, meldung); err != nil {
		return nil, err
	}

	return s.Submit(ctx, id)
}

// ChangeDetectionResult contains detected changes for Änderungsmeldung
type ChangeDetectionResult struct {
	Changes      []DetectedChange `json:"changes"`
	HasChanges   bool             `json:"has_changes"`
	AenderungArt []string         `json:"aenderung_art"`
}

// DetectedChange represents a single detected change
type DetectedChange struct {
	Field     string      `json:"field"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	ChangeArt string      `json:"change_art"`
}

// DetectChanges compares current data with the last accepted meldung for an SV-Nummer
func (s *Service) DetectChanges(ctx context.Context, accountID uuid.UUID, svNummer string, current *ChangeComparisonData) (*ChangeDetectionResult, error) {
	// Get the last accepted meldung for this SV-Nummer
	history, err := s.repo.GetHistoryBySVNummer(ctx, accountID, svNummer)
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %w", err)
	}

	// Find the last accepted meldung (Anmeldung or Änderung)
	var lastMeldung *elda.ELDAMeldung
	for _, m := range history {
		if m.Status == elda.MeldungStatusAccepted || m.Status == elda.MeldungStatusSubmitted {
			if m.Type == elda.MeldungTypeAnmeldung || m.Type == elda.MeldungTypeAenderung {
				lastMeldung = m
				break // History is ordered by date desc
			}
		}
	}

	if lastMeldung == nil {
		return nil, fmt.Errorf("keine akzeptierte Anmeldung für SV-Nummer %s gefunden", svNummer)
	}

	result := &ChangeDetectionResult{
		Changes:      make([]DetectedChange, 0),
		AenderungArt: make([]string, 0),
	}

	// Compare Entgelt
	if current.BruttoMonatlich > 0 && lastMeldung.Entgelt != nil {
		if current.BruttoMonatlich != lastMeldung.Entgelt.BruttoMonatlich {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "brutto_monatlich",
				OldValue:  lastMeldung.Entgelt.BruttoMonatlich,
				NewValue:  current.BruttoMonatlich,
				ChangeArt: "ENTGELT",
			})
			if !containsString(result.AenderungArt, "ENTGELT") {
				result.AenderungArt = append(result.AenderungArt, "ENTGELT")
			}
		}
	}

	// Compare Arbeitszeit
	if current.WochenStunden > 0 && lastMeldung.Arbeitszeit != nil {
		if current.WochenStunden != lastMeldung.Arbeitszeit.WochenStunden {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "wochen_stunden",
				OldValue:  lastMeldung.Arbeitszeit.WochenStunden,
				NewValue:  current.WochenStunden,
				ChangeArt: "ARBEITSZEIT",
			})
			if !containsString(result.AenderungArt, "ARBEITSZEIT") {
				result.AenderungArt = append(result.AenderungArt, "ARBEITSZEIT")
			}
		}
	}

	// Compare Beschaeftigung
	if current.Taetigkeit != "" && lastMeldung.Beschaeftigung != nil {
		if current.Taetigkeit != lastMeldung.Beschaeftigung.Taetigkeit {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "taetigkeit",
				OldValue:  lastMeldung.Beschaeftigung.Taetigkeit,
				NewValue:  current.Taetigkeit,
				ChangeArt: "TAETIGKEIT",
			})
			if !containsString(result.AenderungArt, "TAETIGKEIT") {
				result.AenderungArt = append(result.AenderungArt, "TAETIGKEIT")
			}
		}
		if current.Einstufung != "" && current.Einstufung != lastMeldung.Beschaeftigung.Einstufung {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "einstufung",
				OldValue:  lastMeldung.Beschaeftigung.Einstufung,
				NewValue:  current.Einstufung,
				ChangeArt: "EINSTUFUNG",
			})
			if !containsString(result.AenderungArt, "EINSTUFUNG") {
				result.AenderungArt = append(result.AenderungArt, "EINSTUFUNG")
			}
		}
		if current.Dienstort != "" && current.Dienstort != lastMeldung.Beschaeftigung.Dienstort {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "dienstort",
				OldValue:  lastMeldung.Beschaeftigung.Dienstort,
				NewValue:  current.Dienstort,
				ChangeArt: "DIENSTORT",
			})
			if !containsString(result.AenderungArt, "DIENSTORT") {
				result.AenderungArt = append(result.AenderungArt, "DIENSTORT")
			}
		}
		if current.KollektivCode != "" && current.KollektivCode != lastMeldung.Beschaeftigung.KollektivCode {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "kollektiv_code",
				OldValue:  lastMeldung.Beschaeftigung.KollektivCode,
				NewValue:  current.KollektivCode,
				ChangeArt: "KOLLEKTIV",
			})
			if !containsString(result.AenderungArt, "KOLLEKTIV") {
				result.AenderungArt = append(result.AenderungArt, "KOLLEKTIV")
			}
		}
		if current.Beitragsgruppe != "" && current.Beitragsgruppe != lastMeldung.Beschaeftigung.Beitragsgruppe {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "beitragsgruppe",
				OldValue:  lastMeldung.Beschaeftigung.Beitragsgruppe,
				NewValue:  current.Beitragsgruppe,
				ChangeArt: "BEITRAGSGRUPPE",
			})
			if !containsString(result.AenderungArt, "BEITRAGSGRUPPE") {
				result.AenderungArt = append(result.AenderungArt, "BEITRAGSGRUPPE")
			}
		}
	}

	// Compare Adresse
	if current.Adresse != nil && lastMeldung.Adresse != nil {
		if current.Adresse.Strasse != lastMeldung.Adresse.Strasse ||
			current.Adresse.PLZ != lastMeldung.Adresse.PLZ ||
			current.Adresse.Ort != lastMeldung.Adresse.Ort {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "adresse",
				OldValue:  lastMeldung.Adresse,
				NewValue:  current.Adresse,
				ChangeArt: "ADRESSE",
			})
			if !containsString(result.AenderungArt, "ADRESSE") {
				result.AenderungArt = append(result.AenderungArt, "ADRESSE")
			}
		}
	}

	// Compare Bankverbindung
	if current.Bankverbindung != nil && lastMeldung.Bankverbindung != nil {
		if current.Bankverbindung.IBAN != lastMeldung.Bankverbindung.IBAN {
			result.Changes = append(result.Changes, DetectedChange{
				Field:     "iban",
				OldValue:  lastMeldung.Bankverbindung.IBAN,
				NewValue:  current.Bankverbindung.IBAN,
				ChangeArt: "BANK",
			})
			if !containsString(result.AenderungArt, "BANK") {
				result.AenderungArt = append(result.AenderungArt, "BANK")
			}
		}
	}

	result.HasChanges = len(result.Changes) > 0
	return result, nil
}

// ChangeComparisonData contains the current data to compare
type ChangeComparisonData struct {
	BruttoMonatlich int64                     `json:"brutto_monatlich,omitempty"`
	WochenStunden   float64                   `json:"wochen_stunden,omitempty"`
	Taetigkeit      string                    `json:"taetigkeit,omitempty"`
	Einstufung      string                    `json:"einstufung,omitempty"`
	Dienstort       string                    `json:"dienstort,omitempty"`
	KollektivCode   string                    `json:"kollektiv_code,omitempty"`
	Beitragsgruppe  string                    `json:"beitragsgruppe,omitempty"`
	Adresse         *elda.DienstnehmerAdresse `json:"adresse,omitempty"`
	Bankverbindung  *elda.Bankverbindung      `json:"bankverbindung,omitempty"`
}

// CreateAenderungFromDetection creates an Änderungsmeldung from detected changes
func (s *Service) CreateAenderungFromDetection(ctx context.Context, accountID uuid.UUID, svNummer string, current *ChangeComparisonData, aenderungDatum time.Time) (*elda.ELDAMeldung, error) {
	// First detect what changed
	detection, err := s.DetectChanges(ctx, accountID, svNummer, current)
	if err != nil {
		return nil, err
	}

	if !detection.HasChanges {
		return nil, fmt.Errorf("keine Änderungen erkannt für SV-Nummer %s", svNummer)
	}

	// Get the last accepted meldung for employee data
	history, _ := s.repo.GetHistoryBySVNummer(ctx, accountID, svNummer)
	var lastMeldung *elda.ELDAMeldung
	for _, m := range history {
		if m.Status == elda.MeldungStatusAccepted || m.Status == elda.MeldungStatusSubmitted {
			lastMeldung = m
			break
		}
	}

	if lastMeldung == nil {
		return nil, fmt.Errorf("keine Stammdaten für SV-Nummer %s gefunden", svNummer)
	}

	// Create the Änderungsmeldung
	aenderung := &elda.ELDAMeldung{
		ID:             uuid.New(),
		ELDAAccountID:  accountID,
		Type:           elda.MeldungTypeAenderung,
		Status:         elda.MeldungStatusDraft,
		SVNummer:       svNummer,
		Vorname:        lastMeldung.Vorname,
		Nachname:       lastMeldung.Nachname,
		Geburtsdatum:   lastMeldung.Geburtsdatum,
		Geschlecht:     lastMeldung.Geschlecht,
		AenderungDatum: &aenderungDatum,
		OriginalMeldungID: &lastMeldung.ID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Combine change types for AenderungArt
	if len(detection.AenderungArt) == 1 {
		aenderung.AenderungArt = detection.AenderungArt[0]
	} else {
		aenderung.AenderungArt = "MEHRFACH"
	}

	// Copy changed fields
	if containsString(detection.AenderungArt, "ENTGELT") {
		aenderung.Entgelt = &elda.ExtendedEntgelt{
			BruttoMonatlich: current.BruttoMonatlich,
		}
	}
	if containsString(detection.AenderungArt, "ARBEITSZEIT") {
		aenderung.Arbeitszeit = &elda.ExtendedArbeitszeit{
			WochenStunden: current.WochenStunden,
		}
	}
	if containsString(detection.AenderungArt, "TAETIGKEIT") ||
		containsString(detection.AenderungArt, "EINSTUFUNG") ||
		containsString(detection.AenderungArt, "DIENSTORT") ||
		containsString(detection.AenderungArt, "KOLLEKTIV") ||
		containsString(detection.AenderungArt, "BEITRAGSGRUPPE") {
		aenderung.Beschaeftigung = &elda.ExtendedBeschaeftigung{
			Taetigkeit:     current.Taetigkeit,
			Einstufung:     current.Einstufung,
			Dienstort:      current.Dienstort,
			KollektivCode:  current.KollektivCode,
			Beitragsgruppe: current.Beitragsgruppe,
		}
	}
	if containsString(detection.AenderungArt, "ADRESSE") {
		aenderung.Adresse = current.Adresse
	}
	if containsString(detection.AenderungArt, "BANK") {
		aenderung.Bankverbindung = current.Bankverbindung
	}

	// Save to database
	if err := s.repo.Create(ctx, aenderung); err != nil {
		return nil, fmt.Errorf("failed to create Änderungsmeldung: %w", err)
	}

	return aenderung, nil
}

// Helper function
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper methods

func (s *Service) buildXML(m *elda.ELDAMeldung) ([]byte, error) {
	var doc interface{}

	switch m.Type {
	case elda.MeldungTypeAnmeldung:
		doc = s.buildAnmeldungXML(m)
	case elda.MeldungTypeAbmeldung:
		doc = s.buildAbmeldungXML(m)
	case elda.MeldungTypeAenderung:
		doc = s.buildAenderungXML(m)
	default:
		return nil, fmt.Errorf("unsupported meldung type: %s", m.Type)
	}

	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}

	result := []byte(xml.Header)
	result = append(result, xmlData...)
	return result, nil
}

func (s *Service) buildAnmeldungXML(m *elda.ELDAMeldung) *elda.ExtendedAnmeldungDocument {
	doc := &elda.ExtendedAnmeldungDocument{
		XMLNS:    elda.ELDANS,
		SVNummer: m.SVNummer,
		Vorname:  m.Vorname,
		Nachname: m.Nachname,
	}

	doc.Kopf = elda.ELDAKopf{
		MeldungsArt: string(elda.MeldungTypeAnmeldung),
		Datum:       time.Now().Format("2006-01-02"),
	}

	if m.Geburtsdatum != nil {
		doc.Geburtsdatum = m.Geburtsdatum.Format("2006-01-02")
	}
	if m.Geschlecht != "" {
		doc.Geschlecht = m.Geschlecht
	}
	if m.Eintrittsdatum != nil {
		doc.Eintrittsdatum = m.Eintrittsdatum.Format("2006-01-02")
	}

	if m.Beschaeftigung != nil {
		doc.Beschaeftigung = &elda.XMLBeschaeftigung{
			Art:              m.Beschaeftigung.Art,
			Taetigkeit:       m.Beschaeftigung.Taetigkeit,
			KollektivCode:    m.Beschaeftigung.KollektivCode,
			Einstufung:       m.Beschaeftigung.Einstufung,
			Verwendungsgruppe: m.Beschaeftigung.Verwendungsgruppe,
			Dienstort:        m.Beschaeftigung.Dienstort,
			Beitragsgruppe:   m.Beschaeftigung.Beitragsgruppe,
		}
		if m.Beschaeftigung.Befristet {
			doc.Beschaeftigung.Befristet = "J"
			doc.Beschaeftigung.BefristetBis = m.Beschaeftigung.BefristetBis
		}
	}

	if m.Arbeitszeit != nil {
		doc.Arbeitszeit = &elda.XMLArbeitszeit{
			WochenStunden:   formatAmount(m.Arbeitszeit.WochenStunden),
			TageProWoche:    m.Arbeitszeit.TageProWoche,
			ArbeitszeitCode: m.Arbeitszeit.ArbeitszeitCode,
		}
		if m.Arbeitszeit.Schichtarbeit {
			doc.Arbeitszeit.Schichtarbeit = "J"
		}
		if m.Arbeitszeit.KollektivStunden > 0 {
			doc.Arbeitszeit.KollektivStunden = formatAmount(m.Arbeitszeit.KollektivStunden)
		}
	}

	if m.Entgelt != nil {
		doc.Entgelt = &elda.XMLEntgelt{
			BruttoMonatlich: formatCents(m.Entgelt.BruttoMonatlich),
			EntgeltArt:      m.Entgelt.EntgeltArt,
		}
		if m.Entgelt.NettoMonatlich > 0 {
			doc.Entgelt.NettoMonatlich = formatCents(m.Entgelt.NettoMonatlich)
		}
		if m.Entgelt.Sonderzahlungen > 0 {
			doc.Entgelt.Sonderzahlungen = formatCents(m.Entgelt.Sonderzahlungen)
		}
		if m.Entgelt.Zulagen > 0 {
			doc.Entgelt.Zulagen = formatCents(m.Entgelt.Zulagen)
		}
		if m.Entgelt.Sachbezuege > 0 {
			doc.Entgelt.Sachbezuege = formatCents(m.Entgelt.Sachbezuege)
		}
	}

	if m.Adresse != nil {
		doc.Adresse = &elda.XMLAdresse{
			Strasse:    m.Adresse.Strasse,
			Hausnummer: m.Adresse.Hausnummer,
			Stiege:     m.Adresse.Stiege,
			Tuer:       m.Adresse.Tuer,
			PLZ:        m.Adresse.PLZ,
			Ort:        m.Adresse.Ort,
			Land:       m.Adresse.Land,
		}
	}

	if m.Bankverbindung != nil {
		doc.Bankverbindung = &elda.XMLBankverbindung{
			IBAN:         m.Bankverbindung.IBAN,
			BIC:          m.Bankverbindung.BIC,
			Kontoinhaber: m.Bankverbindung.Kontoinhaber,
		}
	}

	return doc
}

func (s *Service) buildAbmeldungXML(m *elda.ELDAMeldung) *elda.ExtendedAbmeldungDocument {
	doc := &elda.ExtendedAbmeldungDocument{
		XMLNS:         elda.ELDANS,
		SVNummer:      m.SVNummer,
		Grund:         m.AustrittGrund,
		Abfertigung:   m.Abfertigung,
		Urlaubsersatz: m.Urlaubsersatz,
		URLTage:       m.URLTage,
	}

	doc.Kopf = elda.ELDAKopf{
		MeldungsArt: string(elda.MeldungTypeAbmeldung),
		Datum:       time.Now().Format("2006-01-02"),
	}

	if m.Austrittsdatum != nil {
		doc.Austrittsdatum = m.Austrittsdatum.Format("2006-01-02")
	}

	return doc
}

func (s *Service) buildAenderungXML(m *elda.ELDAMeldung) *AenderungDocument {
	doc := &AenderungDocument{
		XMLNS:        elda.ELDANS,
		SVNummer:     m.SVNummer,
		AenderungArt: m.AenderungArt,
	}

	doc.Kopf = elda.ELDAKopf{
		MeldungsArt: string(elda.MeldungTypeAenderung),
		Datum:       time.Now().Format("2006-01-02"),
	}

	if m.AenderungDatum != nil {
		doc.AenderungDatum = m.AenderungDatum.Format("2006-01-02")
	}

	// Include changed fields based on AenderungArt
	if m.Beschaeftigung != nil {
		doc.Beschaeftigung = &elda.XMLBeschaeftigung{
			Art:              m.Beschaeftigung.Art,
			Taetigkeit:       m.Beschaeftigung.Taetigkeit,
			KollektivCode:    m.Beschaeftigung.KollektivCode,
			Einstufung:       m.Beschaeftigung.Einstufung,
			Beitragsgruppe:   m.Beschaeftigung.Beitragsgruppe,
		}
	}

	if m.Arbeitszeit != nil {
		doc.Arbeitszeit = &elda.XMLArbeitszeit{
			WochenStunden:   formatAmount(m.Arbeitszeit.WochenStunden),
			TageProWoche:    m.Arbeitszeit.TageProWoche,
			ArbeitszeitCode: m.Arbeitszeit.ArbeitszeitCode,
		}
	}

	if m.Entgelt != nil {
		doc.Entgelt = &elda.XMLEntgelt{
			BruttoMonatlich: formatCents(m.Entgelt.BruttoMonatlich),
		}
	}

	if m.Adresse != nil {
		doc.Adresse = &elda.XMLAdresse{
			Strasse:    m.Adresse.Strasse,
			Hausnummer: m.Adresse.Hausnummer,
			PLZ:        m.Adresse.PLZ,
			Ort:        m.Adresse.Ort,
		}
	}

	return doc
}

// AenderungDocument is the XML document for Änderungsmeldung
type AenderungDocument struct {
	XMLName        xml.Name             `xml:"Aenderung"`
	XMLNS          string               `xml:"xmlns,attr"`
	Kopf           elda.ELDAKopf        `xml:"Kopf"`
	SVNummer       string               `xml:"SVNummer"`
	AenderungArt   string               `xml:"AenderungArt"`
	AenderungDatum string               `xml:"AenderungDatum,omitempty"`
	Beschaeftigung *elda.XMLBeschaeftigung `xml:"Beschaeftigung,omitempty"`
	Arbeitszeit    *elda.XMLArbeitszeit    `xml:"Arbeitszeit,omitempty"`
	Entgelt        *elda.XMLEntgelt        `xml:"Entgelt,omitempty"`
	Adresse        *elda.XMLAdresse        `xml:"Adresse,omitempty"`
}

// Helper functions
func formatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

func formatCents(cents int64) string {
	euros := float64(cents) / 100
	return strconv.FormatFloat(euros, 'f', 2, 64)
}

// Validator validates ELDA meldungen
type Validator struct{}

// NewValidator creates a new meldung validator
func NewValidator() *Validator {
	return &Validator{}
}

// ValidationResult contains the validation result
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ErrorMessages returns all error messages
func (r *ValidationResult) ErrorMessages() []string {
	return r.Errors
}

// ValidateCreateRequest validates a create request
func (v *Validator) ValidateCreateRequest(req *elda.MeldungCreateRequest) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if req.ELDAAccountID == uuid.Nil {
		result.Errors = append(result.Errors, "elda_account_id: ELDA-Konto ID erforderlich")
	}

	if req.Type == "" {
		result.Errors = append(result.Errors, "type: Meldungsart erforderlich")
	}

	if req.SVNummer == "" {
		result.Errors = append(result.Errors, "sv_nummer: SV-Nummer erforderlich")
	} else if err := elda.ValidateSVNummer(req.SVNummer); err != nil {
		result.Errors = append(result.Errors, "sv_nummer: "+err.Error())
	}

	if req.Vorname == "" {
		result.Errors = append(result.Errors, "vorname: Vorname erforderlich")
	}

	if req.Nachname == "" {
		result.Errors = append(result.Errors, "nachname: Nachname erforderlich")
	}

	// Type-specific validation
	switch req.Type {
	case elda.MeldungTypeAnmeldung:
		if req.Eintrittsdatum == "" {
			result.Errors = append(result.Errors, "eintrittsdatum: Eintrittsdatum erforderlich für Anmeldung")
		}
	case elda.MeldungTypeAbmeldung:
		if req.Austrittsdatum == "" {
			result.Errors = append(result.Errors, "austrittsdatum: Austrittsdatum erforderlich für Abmeldung")
		}
		if req.AustrittGrund == "" {
			result.Errors = append(result.Errors, "austritt_grund: Austrittgrund erforderlich für Abmeldung")
		}
	case elda.MeldungTypeAenderung:
		if req.AenderungArt == "" {
			result.Errors = append(result.Errors, "aenderung_art: Änderungsart erforderlich")
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// ValidateMeldung validates a meldung entity
func (v *Validator) ValidateMeldung(m *elda.ELDAMeldung) *ValidationResult {
	result := &ValidationResult{Valid: true}

	if err := elda.ValidateSVNummer(m.SVNummer); err != nil {
		result.Errors = append(result.Errors, "sv_nummer: "+err.Error())
	}

	if m.Vorname == "" {
		result.Errors = append(result.Errors, "vorname: Vorname erforderlich")
	}

	if m.Nachname == "" {
		result.Errors = append(result.Errors, "nachname: Nachname erforderlich")
	}

	// Type-specific validation
	switch m.Type {
	case elda.MeldungTypeAnmeldung:
		if m.Eintrittsdatum == nil {
			result.Errors = append(result.Errors, "eintrittsdatum: Eintrittsdatum erforderlich")
		}
	case elda.MeldungTypeAbmeldung:
		if m.Austrittsdatum == nil {
			result.Errors = append(result.Errors, "austrittsdatum: Austrittsdatum erforderlich")
		}
		if m.AustrittGrund == "" {
			result.Errors = append(result.Errors, "austritt_grund: Austrittgrund erforderlich")
		}
	case elda.MeldungTypeAenderung:
		if m.AenderungArt == "" {
			result.Errors = append(result.Errors, "aenderung_art: Änderungsart erforderlich")
		}
	}

	result.Valid = len(result.Errors) == 0
	return result
}
