package eldameldung

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/elda"
)

// Handler handles HTTP requests for ELDA meldung operations
type Handler struct {
	service  *Service
	refData  *ReferenceDataService
}

// NewHandler creates a new ELDA meldung handler
func NewHandler(service *Service, refData *ReferenceDataService) *Handler {
	return &Handler{service: service, refData: refData}
}

// Routes returns the router for ELDA meldung endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// CRUD
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)

	// Validation and preview
	r.Post("/{id}/validate", h.Validate)
	r.Get("/{id}/preview", h.Preview)

	// Submission
	r.Post("/{id}/send", h.Submit)
	r.Post("/{id}/retry", h.Retry)

	// History
	r.Get("/history/{sv_nummer}", h.GetHistory)

	// Änderungsmeldungen
	r.Post("/detect-changes", h.DetectChanges)
	r.Post("/aenderung-from-detection", h.CreateAenderungFromDetection)

	// Reference data endpoints
	r.Get("/kollektivvertraege/search", h.SearchKollektivvertraege)
	r.Get("/arbeitszeit-codes", h.GetArbeitszeitCodes)
	r.Get("/beitragsgruppen/search", h.SearchBeitragsgruppen)
	r.Get("/austritt-gruende", h.GetAustrittGruende)
	r.Get("/aenderung-arten", h.GetAenderungArten)

	return r
}

// Create handles POST /api/v1/elda-meldungen
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req elda.MeldungCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	meldung, err := h.service.Create(r.Context(), &req)
	if err != nil {
		if ve, ok := err.(*ValidationError); ok {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":  ve.Message,
				"errors": ve.Errors,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, meldung)
}

// Get handles GET /api/v1/elda-meldungen/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	meldung, err := h.service.Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Meldung nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, meldung)
}

// List handles GET /api/v1/elda-meldungen
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if v := r.URL.Query().Get("elda_account_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ELDAAccountID = &id
		}
	}

	if v := r.URL.Query().Get("type"); v != "" {
		t := elda.MeldungType(v)
		filter.Type = &t
	}

	if v := r.URL.Query().Get("status"); v != "" {
		s := elda.MeldungStatus(v)
		filter.Status = &s
	}

	if v := r.URL.Query().Get("sv_nummer"); v != "" {
		filter.SVNummer = v
	}

	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartDate = &t
		}
	}

	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndDate = &t
		}
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 && limit <= 500 {
			filter.Limit = limit
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	meldungen, err := h.service.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	count, _ := h.service.Count(r.Context(), filter)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   meldungen,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Delete handles DELETE /api/v1/elda-meldungen/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Meldung gelöscht"})
}

// Validate handles POST /api/v1/elda-meldungen/{id}/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	result, err := h.service.Validate(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Preview handles GET /api/v1/elda-meldungen/{id}/preview
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	preview, err := h.service.Preview(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	// Check if XML format requested
	if r.URL.Query().Get("format") == "xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(preview.XML))
		return
	}

	respondJSON(w, http.StatusOK, preview)
}

// Submit handles POST /api/v1/elda-meldungen/{id}/send
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	result, err := h.service.Submit(r.Context(), id)
	if err != nil {
		if ve, ok := err.(*ValidationError); ok {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":  ve.Message,
				"errors": ve.Errors,
			})
			return
		}
		// Return result even on error (contains ELDA error details)
		if result != nil {
			respondJSON(w, http.StatusBadGateway, map[string]interface{}{
				"success":       false,
				"error":         err.Error(),
				"error_code":    result.ErrorCode,
				"error_message": result.ErrorMessage,
			})
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// Retry handles POST /api/v1/elda-meldungen/{id}/retry
func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	result, err := h.service.Retry(r.Context(), id)
	if err != nil {
		if result != nil {
			respondJSON(w, http.StatusBadGateway, map[string]interface{}{
				"success":       false,
				"error":         err.Error(),
				"error_code":    result.ErrorCode,
				"error_message": result.ErrorMessage,
			})
			return
		}
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// GetHistory handles GET /api/v1/elda-meldungen/history/{sv_nummer}
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	svNummer := chi.URLParam(r, "sv_nummer")
	if svNummer == "" {
		respondError(w, http.StatusBadRequest, "SV-Nummer erforderlich")
		return
	}

	accountIDStr := r.URL.Query().Get("elda_account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige elda_account_id")
		return
	}

	history, err := h.service.GetHistory(r.Context(), accountID, svNummer)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sv_nummer": svNummer,
		"history":   history,
		"count":     len(history),
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// SearchKollektivvertraege handles GET /api/v1/elda-meldungen/kollektivvertraege/search
func (h *Handler) SearchKollektivvertraege(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	results, err := h.refData.SearchKollektivvertraege(r.Context(), query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

// GetArbeitszeitCodes handles GET /api/v1/elda-meldungen/arbeitszeit-codes
func (h *Handler) GetArbeitszeitCodes(w http.ResponseWriter, r *http.Request) {
	codes := h.refData.GetArbeitszeitCodes(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"codes": codes,
	})
}

// SearchBeitragsgruppen handles GET /api/v1/elda-meldungen/beitragsgruppen/search
func (h *Handler) SearchBeitragsgruppen(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	results, err := h.refData.SearchBeitragsgruppen(r.Context(), query, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"query":   query,
	})
}

// GetAustrittGruende handles GET /api/v1/elda-meldungen/austritt-gruende
func (h *Handler) GetAustrittGruende(w http.ResponseWriter, r *http.Request) {
	gruende := h.refData.GetAustrittGruende(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"gruende": gruende,
	})
}

// GetAenderungArten handles GET /api/v1/elda-meldungen/aenderung-arten
func (h *Handler) GetAenderungArten(w http.ResponseWriter, r *http.Request) {
	arten := h.refData.GetAenderungArten(r.Context())
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"arten": arten,
	})
}

// DetectChangesRequest is the request for change detection
type DetectChangesRequest struct {
	ELDAAccountID   uuid.UUID              `json:"elda_account_id"`
	SVNummer        string                 `json:"sv_nummer"`
	ComparisonData  ChangeComparisonData   `json:"comparison_data"`
}

// DetectChanges handles POST /api/v1/elda-meldungen/detect-changes
func (h *Handler) DetectChanges(w http.ResponseWriter, r *http.Request) {
	var req DetectChangesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	if req.ELDAAccountID == uuid.Nil {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}

	if req.SVNummer == "" {
		respondError(w, http.StatusBadRequest, "sv_nummer erforderlich")
		return
	}

	result, err := h.service.DetectChanges(r.Context(), req.ELDAAccountID, req.SVNummer, &req.ComparisonData)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// CreateAenderungFromDetectionRequest is the request for creating Änderungsmeldung from detected changes
type CreateAenderungFromDetectionRequest struct {
	ELDAAccountID   uuid.UUID              `json:"elda_account_id"`
	SVNummer        string                 `json:"sv_nummer"`
	AenderungDatum  string                 `json:"aenderung_datum"` // YYYY-MM-DD
	ComparisonData  ChangeComparisonData   `json:"comparison_data"`
}

// CreateAenderungFromDetection handles POST /api/v1/elda-meldungen/aenderung-from-detection
func (h *Handler) CreateAenderungFromDetection(w http.ResponseWriter, r *http.Request) {
	var req CreateAenderungFromDetectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	if req.ELDAAccountID == uuid.Nil {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}

	if req.SVNummer == "" {
		respondError(w, http.StatusBadRequest, "sv_nummer erforderlich")
		return
	}

	aenderungDatum := time.Now()
	if req.AenderungDatum != "" {
		if t, err := time.Parse("2006-01-02", req.AenderungDatum); err == nil {
			aenderungDatum = t
		} else {
			respondError(w, http.StatusBadRequest, "Ungültiges Datum (Format: YYYY-MM-DD)")
			return
		}
	}

	meldung, err := h.service.CreateAenderungFromDetection(r.Context(), req.ELDAAccountID, req.SVNummer, &req.ComparisonData, aenderungDatum)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, meldung)
}

// RegisterRoutes registers ELDA meldung routes with the router
func RegisterRoutes(r chi.Router, service *Service, refData *ReferenceDataService) {
	handler := NewHandler(service, refData)
	r.Mount("/elda-meldungen", handler.Routes())
}
