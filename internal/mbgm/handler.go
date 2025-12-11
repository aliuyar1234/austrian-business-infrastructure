package mbgm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/api"
	"austrian-business-infrastructure/internal/elda"
)

// Handler handles HTTP requests for mBGM operations
type Handler struct {
	service *Service
}

// NewHandler creates a new mBGM HTTP handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers mBGM routes with the router
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/mbgm", func(r chi.Router) {
		r.Post("/", h.Create)                        // T020: POST /api/v1/mbgm
		r.Get("/", h.List)                           // T021: GET /api/v1/mbgm
		r.Post("/import", h.ImportCSV)               // T029: POST /api/v1/mbgm/import
		r.Get("/import/formats", h.GetImportFormats) // Get supported formats
		r.Get("/import/template", h.GetImportTemplate) // Get CSV template
		r.Get("/{id}", h.GetByID)                    // T022: GET /api/v1/mbgm/{id}
		r.Get("/{id}/preview", h.PreviewXML)         // T023: GET /api/v1/mbgm/{id}/preview
		r.Post("/{id}/send", h.Submit)               // T024: POST /api/v1/mbgm/{id}/send
		r.Post("/{id}/validate", h.Validate)         // Validate mBGM
		r.Post("/{id}/correction", h.CreateCorrection) // T026: Create correction
		r.Delete("/{id}", h.Delete)                  // Delete draft mBGM
		r.Get("/{id}/summary", h.GetSummary)         // Get summary
	})

	// Reference data endpoints
	r.Get("/beitragsgruppen", h.GetBeitragsgruppen)
	r.Get("/mbgm-deadline", h.GetDeadlineInfo)
}

// Create handles POST /api/v1/mbgm
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req elda.MBGMCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	mbgm, err := h.service.Create(r.Context(), &req)
	if err != nil {
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			api.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      "Validation failed",
				"validation": validationErr.Result,
			})
			return
		}
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to create mBGM", err)
		return
	}

	api.RespondJSON(w, http.StatusCreated, mbgm)
}

// List handles GET /api/v1/mbgm
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	accountIDStr := r.URL.Query().Get("account_id")
	if accountIDStr == "" {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "account_id is required", nil)
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid account_id", err)
		return
	}

	filter := ServiceListFilter{}

	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		var year int
		if _, err := parseIntParam(yearStr, &year); err == nil {
			filter.Year = &year
		}
	}

	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		var month int
		if _, err := parseIntParam(monthStr, &month); err == nil {
			filter.Month = &month
		}
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status := elda.MBGMStatus(statusStr)
		filter.Status = &status
	}

	mbgms, err := h.service.List(r.Context(), accountID, filter)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to list mBGMs", err)
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"mbgms": mbgms,
		"count": len(mbgms),
	})
}

// GetByID handles GET /api/v1/mbgm/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	mbgm, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusNotFound, "mBGM not found", err)
		return
	}

	api.RespondJSON(w, http.StatusOK, mbgm)
}

// PreviewXML handles GET /api/v1/mbgm/{id}/preview
func (h *Handler) PreviewXML(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	dienstgeberNr := r.URL.Query().Get("dienstgeber_nr")
	if dienstgeberNr == "" {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "dienstgeber_nr is required", nil)
		return
	}

	preview, err := h.service.PreviewXML(r.Context(), id, dienstgeberNr)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to generate preview", err)
		return
	}

	// Check if client wants raw XML
	if r.URL.Query().Get("format") == "xml" {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(preview.XML))
		return
	}

	api.RespondJSON(w, http.StatusOK, preview)
}

// Validate handles POST /api/v1/mbgm/{id}/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	result, err := h.service.Validate(r.Context(), id)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Validation failed", err)
		return
	}

	api.RespondJSON(w, http.StatusOK, result)
}

// Submit handles POST /api/v1/mbgm/{id}/send
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	var req struct {
		DienstgeberNr string `json:"dienstgeber_nr"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.DienstgeberNr == "" {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "dienstgeber_nr is required", nil)
		return
	}

	result, err := h.service.Submit(r.Context(), id, req.DienstgeberNr)
	if err != nil {
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			api.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      "Validation failed",
				"validation": validationErr.Result,
			})
			return
		}
		api.RespondJSON(w, http.StatusOK, result) // Return result even on ELDA error
		return
	}

	api.RespondJSON(w, http.StatusOK, result)
}

// CreateCorrection handles POST /api/v1/mbgm/{id}/correction
func (h *Handler) CreateCorrection(w http.ResponseWriter, r *http.Request) {
	originalID, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	var req elda.MBGMCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	mbgm, err := h.service.CreateCorrection(r.Context(), originalID, &req)
	if err != nil {
		var validationErr *ValidationError
		if errors.As(err, &validationErr) {
			api.RespondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"error":      "Validation failed",
				"validation": validationErr.Result,
			})
			return
		}
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to create correction", err)
		return
	}

	api.RespondJSON(w, http.StatusCreated, mbgm)
}

// Delete handles DELETE /api/v1/mbgm/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetSummary handles GET /api/v1/mbgm/{id}/summary
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUIDParam(chi.URLParam(r, "id"))
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid mBGM ID", err)
		return
	}

	dienstgeberNr := r.URL.Query().Get("dienstgeber_nr")
	if dienstgeberNr == "" {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "dienstgeber_nr is required", nil)
		return
	}

	summary, err := h.service.GetSummary(r.Context(), id, dienstgeberNr)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to get summary", err)
		return
	}

	api.RespondJSON(w, http.StatusOK, summary)
}

// GetBeitragsgruppen handles GET /api/v1/beitragsgruppen
func (h *Handler) GetBeitragsgruppen(w http.ResponseWriter, r *http.Request) {
	groups, err := h.service.GetBeitragsgruppen(r.Context())
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to get Beitragsgruppen", err)
		return
	}

	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"beitragsgruppen": groups,
		"count":           len(groups),
	})
}

// ImportCSV handles POST /api/v1/mbgm/import
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Failed to parse form", err)
		return
	}

	// Get file
	file, _, err := r.FormFile("file")
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "No file provided", err)
		return
	}
	defer file.Close()

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusInternalServerError, "Failed to read file", err)
		return
	}

	// Get parameters
	accountIDStr := r.FormValue("account_id")
	if accountIDStr == "" {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "account_id is required", nil)
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid account_id", err)
		return
	}

	var year, month int
	if yearStr := r.FormValue("year"); yearStr != "" {
		if _, err := parseIntParam(yearStr, &year); err != nil {
			api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid year", err)
			return
		}
	}
	if monthStr := r.FormValue("month"); monthStr != "" {
		if _, err := parseIntParam(monthStr, &month); err != nil {
			api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid month", err)
			return
		}
	}

	format := CSVFormat(r.FormValue("format"))
	if format == "" {
		format = FormatGeneric
	}

	// Import
	importer := NewImporter(nil)
	result, err := importer.Import(data, format, accountID, year, month)
	if err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Import failed", err)
		return
	}

	api.RespondJSON(w, http.StatusOK, result)
}

// GetImportFormats handles GET /api/v1/mbgm/import/formats
func (h *Handler) GetImportFormats(w http.ResponseWriter, r *http.Request) {
	formats := GetFormatSpecs()
	api.RespondJSON(w, http.StatusOK, map[string]interface{}{
		"formats": formats,
	})
}

// GetImportTemplate handles GET /api/v1/mbgm/import/template
func (h *Handler) GetImportTemplate(w http.ResponseWriter, r *http.Request) {
	format := CSVFormat(r.URL.Query().Get("format"))
	if format == "" {
		format = FormatGeneric
	}

	template := GetCSVTemplate(format)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"mbgm_template_%s.csv\"", format))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(template))
}

// GetDeadlineInfo handles GET /api/v1/mbgm-deadline
func (h *Handler) GetDeadlineInfo(w http.ResponseWriter, r *http.Request) {
	var year, month int

	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	if yearStr == "" || monthStr == "" {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "year and month are required", nil)
		return
	}

	if _, err := parseIntParam(yearStr, &year); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid year", err)
		return
	}

	if _, err := parseIntParam(monthStr, &month); err != nil {
		api.RespondErrorWithDetails(w, http.StatusBadRequest, "Invalid month", err)
		return
	}

	info := h.service.GetDeadlineInfo(year, month)
	api.RespondJSON(w, http.StatusOK, info)
}

// Helper functions

func parseUUIDParam(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func parseIntParam(s string, v *int) (bool, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return false, err
	}
	*v = n
	return true, nil
}

