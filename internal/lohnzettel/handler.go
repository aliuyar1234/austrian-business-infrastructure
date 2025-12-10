package lohnzettel

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/elda"
)

// Handler handles HTTP requests for Lohnzettel operations
type Handler struct {
	service *Service
}

// NewHandler creates a new Lohnzettel handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for Lohnzettel endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	// Lohnzettel CRUD
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Delete("/{id}", h.Delete)

	// Validation and preview
	r.Post("/{id}/validate", h.Validate)
	r.Get("/{id}/preview", h.Preview)
	r.Get("/{id}/summary", h.GetSummary)

	// Submission
	r.Post("/{id}/send", h.Submit)
	r.Get("/{id}/status", h.QueryStatus)

	// Correction
	r.Post("/{id}/berichtigung", h.CreateBerichtigung)
	r.Post("/{id}/berichtigung/send", h.SubmitBerichtigung)

	// Batch operations
	r.Post("/batch", h.CreateBatch)
	r.Get("/batch/{id}", h.GetBatch)
	r.Get("/batches", h.ListBatches)
	r.Post("/batch/{id}/send", h.SubmitBatch)

	// CSV Import
	r.Post("/import", h.ImportCSV)

	// Bulk operations
	r.Post("/bulk", h.BulkCreate)

	// Deadline info
	r.Get("/deadline/{year}", h.GetDeadlineInfo)
	r.Get("/deadline-status", h.GetDeadlineStatus)

	// Statistics
	r.Get("/statistics/{year}", h.GetStatistics)

	return r
}

// Create handles POST /api/v1/lohnzettel
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req elda.LohnzettelCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	lohnzettel, err := h.service.Create(r.Context(), &req)
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

	respondJSON(w, http.StatusCreated, lohnzettel)
}

// Get handles GET /api/v1/lohnzettel/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	lohnzettel, err := h.service.Get(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Lohnzettel nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, lohnzettel)
}

// List handles GET /api/v1/lohnzettel
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ServiceListFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if v := r.URL.Query().Get("elda_account_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ELDAAccountID = &id
		}
	}

	if v := r.URL.Query().Get("year"); v != "" {
		if year, err := strconv.Atoi(v); err == nil {
			filter.Year = &year
		}
	}

	if v := r.URL.Query().Get("status"); v != "" {
		status := elda.L16Status(v)
		filter.Status = &status
	}

	if v := r.URL.Query().Get("batch_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.BatchID = &id
		}
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 && limit <= 1000 {
			filter.Limit = limit
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	lohnzettel, err := h.service.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get total count
	count, _ := h.service.Count(r.Context(), filter)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   lohnzettel,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Delete handles DELETE /api/v1/lohnzettel/{id}
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

	respondJSON(w, http.StatusOK, map[string]string{"message": "Lohnzettel gelöscht"})
}

// Validate handles POST /api/v1/lohnzettel/{id}/validate
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

// Preview handles GET /api/v1/lohnzettel/{id}/preview
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

// GetSummary handles GET /api/v1/lohnzettel/{id}/summary
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	summary, err := h.service.GetSummary(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, summary)
}

// Submit handles POST /api/v1/lohnzettel/{id}/send
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

// QueryStatus handles GET /api/v1/lohnzettel/{id}/status
func (h *Handler) QueryStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	status, err := h.service.QueryStatus(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// CreateBerichtigung handles POST /api/v1/lohnzettel/{id}/berichtigung
func (h *Handler) CreateBerichtigung(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	var req struct {
		L16Data elda.L16Data `json:"l16_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	correction, err := h.service.CreateBerichtigung(r.Context(), id, &req.L16Data)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, correction)
}

// SubmitBerichtigung handles POST /api/v1/lohnzettel/{id}/berichtigung/send
func (h *Handler) SubmitBerichtigung(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	result, err := h.service.SubmitBerichtigung(r.Context(), id)
	if err != nil {
		if ve, ok := err.(*ValidationError); ok {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":  ve.Message,
				"errors": ve.Errors,
			})
			return
		}
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

// CreateBatch handles POST /api/v1/lohnzettel/batch
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req elda.LohnzettelBatchCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	batch, err := h.service.CreateBatch(r.Context(), &req)
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

	respondJSON(w, http.StatusCreated, batch)
}

// GetBatch handles GET /api/v1/lohnzettel/batch/{id}
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	batch, err := h.service.GetBatch(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "Batch nicht gefunden")
		return
	}

	respondJSON(w, http.StatusOK, batch)
}

// ListBatches handles GET /api/v1/lohnzettel/batches
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
	filter := BatchListFilter{
		Limit:  50,
		Offset: 0,
	}

	if v := r.URL.Query().Get("elda_account_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ELDAAccountID = &id
		}
	}

	if v := r.URL.Query().Get("year"); v != "" {
		if year, err := strconv.Atoi(v); err == nil {
			filter.Year = &year
		}
	}

	if v := r.URL.Query().Get("status"); v != "" {
		status := elda.LohnzettelBatchStatus(v)
		filter.Status = &status
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	batches, err := h.service.ListBatches(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   batches,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// SubmitBatch handles POST /api/v1/lohnzettel/batch/{id}/send
func (h *Handler) SubmitBatch(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	// Optional max_concurrent parameter
	maxConcurrent := 10
	if v := r.URL.Query().Get("max_concurrent"); v != "" {
		if mc, err := strconv.Atoi(v); err == nil && mc > 0 && mc <= 10 {
			maxConcurrent = mc
		}
	}

	result, err := h.service.SubmitBatch(r.Context(), id, maxConcurrent)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// ImportCSV handles POST /api/v1/lohnzettel/import
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		respondError(w, http.StatusBadRequest, "Fehler beim Parsen der Datei: "+err.Error())
		return
	}

	// Get ELDA account ID
	eldaAccountIDStr := r.FormValue("elda_account_id")
	if eldaAccountIDStr == "" {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}
	eldaAccountID, err := uuid.Parse(eldaAccountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige elda_account_id")
		return
	}

	// Get year
	yearStr := r.FormValue("year")
	if yearStr == "" {
		respondError(w, http.StatusBadRequest, "year erforderlich")
		return
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültiges Jahr")
		return
	}

	// Get format (optional, default to generic)
	format := CSVFormat(r.FormValue("format"))
	if format == "" {
		format = CSVFormatGeneric
	}

	// Get file
	file, _, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "CSV-Datei erforderlich")
		return
	}
	defer file.Close()

	// Read file content
	csvData, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Fehler beim Lesen der Datei")
		return
	}

	// Import
	result, err := h.service.ImportFromCSV(r.Context(), eldaAccountID, year, csvData, format)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

// BulkCreate handles POST /api/v1/lohnzettel/bulk
func (h *Handler) BulkCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Lohnzettel []*elda.LohnzettelCreateRequest `json:"lohnzettel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige Anfrage: "+err.Error())
		return
	}

	if len(req.Lohnzettel) == 0 {
		respondError(w, http.StatusBadRequest, "Mindestens ein Lohnzettel erforderlich")
		return
	}

	if len(req.Lohnzettel) > 100 {
		respondError(w, http.StatusBadRequest, "Maximal 100 Lohnzettel pro Anfrage")
		return
	}

	results, err := h.service.BulkCreate(r.Context(), req.Lohnzettel)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Count successes/failures
	var success, failed int
	for _, r := range results {
		if r.Success {
			success++
		} else {
			failed++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"total":   len(results),
		"success": success,
		"failed":  failed,
		"results": results,
	})
}

// GetDeadlineInfo handles GET /api/v1/lohnzettel/deadline/{year}
func (h *Handler) GetDeadlineInfo(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültiges Jahr")
		return
	}

	info := h.service.GetDeadlineInfo(year)
	respondJSON(w, http.StatusOK, info)
}

// GetDeadlineStatus handles GET /api/v1/lohnzettel/deadline-status
func (h *Handler) GetDeadlineStatus(w http.ResponseWriter, r *http.Request) {
	eldaAccountIDStr := r.URL.Query().Get("elda_account_id")
	if eldaAccountIDStr == "" {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}

	eldaAccountID, err := uuid.Parse(eldaAccountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige elda_account_id")
		return
	}

	status, err := h.service.GetDeadlineStatus(r.Context(), eldaAccountID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, status)
}

// GetStatistics handles GET /api/v1/lohnzettel/statistics/{year}
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültiges Jahr")
		return
	}

	stats, err := h.service.GetStatistics(r.Context(), year)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
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

// RegisterRoutes registers Lohnzettel routes with the router
func RegisterRoutes(r chi.Router, service *Service) {
	handler := NewHandler(service)
	r.Mount("/lohnzettel", handler.Routes())
}
