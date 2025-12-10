package imports

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles import HTTP requests
type Handler struct {
	repo      *Repository
	parser    *Parser
	jobRunner *JobRunner
}

// NewHandler creates a new import handler
func NewHandler(repo *Repository, parser *Parser, jobRunner *JobRunner) *Handler {
	return &Handler{
		repo:      repo,
		parser:    parser,
		jobRunner: jobRunner,
	}
}

// RegisterRoutes registers import routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth func(http.Handler) http.Handler) {
	router.Handle("POST /api/v1/accounts/import", requireAuth(http.HandlerFunc(h.Upload)))
	router.Handle("POST /api/v1/accounts/import/preview", requireAuth(http.HandlerFunc(h.Preview)))
	router.Handle("GET /api/v1/accounts/import/{id}", requireAuth(http.HandlerFunc(h.GetStatus)))
	router.Handle("GET /api/v1/accounts/import", requireAuth(http.HandlerFunc(h.List)))
}

// Upload handles POST /api/v1/accounts/import
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	userID := api.GetUserID(r.Context())

	if tenantID == "" || userID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)
	userUUID, _ := uuid.Parse(userID)

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		api.BadRequest(w, "invalid multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		api.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	// Parse CSV
	result, err := h.parser.Parse(file)
	if err != nil {
		switch err {
		case ErrEmptyFile:
			api.BadRequest(w, "CSV file is empty")
		case ErrMissingHeaders:
			api.BadRequest(w, "CSV missing required headers: name, type, tid, ben_id, pin")
		case ErrTooManyRows:
			api.BadRequest(w, "CSV exceeds maximum 500 rows")
		default:
			api.BadRequest(w, "failed to parse CSV: "+err.Error())
		}
		return
	}

	if result.ValidCount == 0 {
		api.BadRequest(w, "no valid rows in CSV")
		return
	}

	// Create import job
	totalRows := result.TotalRows
	job := &ImportJob{
		TenantID:  tenantUUID,
		UserID:    userUUID,
		TotalRows: &totalRows,
	}

	job, err = h.repo.Create(r.Context(), job)
	if err != nil {
		api.InternalError(w)
		return
	}

	// Run import in background
	go func() {
		h.jobRunner.Run(r.Context(), job, result.Rows)
	}()

	api.JSONResponse(w, http.StatusAccepted, map[string]interface{}{
		"id":          job.ID,
		"status":      "pending",
		"total_rows":  totalRows,
		"valid_rows":  result.ValidCount,
		"error_rows":  result.ErrorCount,
	})
}

// Preview handles POST /api/v1/accounts/import/preview
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	// Read body
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB max
	if err != nil {
		api.BadRequest(w, "failed to read request body")
		return
	}

	// Check content type
	contentType := r.Header.Get("Content-Type")

	var csvData io.Reader
	if contentType == "application/json" {
		// JSON body with CSV content
		var req struct {
			CSV string `json:"csv"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			api.BadRequest(w, "invalid JSON body")
			return
		}
		csvData = stringReader(req.CSV)
	} else {
		// Raw CSV body or multipart
		if err := r.ParseMultipartForm(10 << 20); err == nil {
			file, _, err := r.FormFile("file")
			if err != nil {
				api.BadRequest(w, "file is required")
				return
			}
			defer file.Close()
			csvData = file
		} else {
			csvData = bytesReader(body)
		}
	}

	// Parse CSV
	result, err := h.parser.Parse(csvData)
	if err != nil {
		switch err {
		case ErrEmptyFile:
			api.BadRequest(w, "CSV file is empty")
		case ErrMissingHeaders:
			api.BadRequest(w, "CSV missing required headers")
		case ErrTooManyRows:
			api.BadRequest(w, "CSV exceeds maximum 500 rows")
		default:
			api.BadRequest(w, "failed to parse CSV: "+err.Error())
		}
		return
	}

	// Return preview (first 20 rows only)
	previewRows := result.Rows
	if len(previewRows) > 20 {
		previewRows = previewRows[:20]
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"rows":        previewRows,
		"total_rows":  result.TotalRows,
		"valid_count": result.ValidCount,
		"error_count": result.ErrorCount,
	})
}

// GetStatus handles GET /api/v1/accounts/import/{id}
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	jobID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid job ID")
		return
	}

	job, err := h.repo.GetByID(r.Context(), jobID, tenantUUID)
	if err != nil {
		if err == ErrImportJobNotFound {
			api.NotFound(w, "import job not found")
		} else {
			api.InternalError(w)
		}
		return
	}

	api.JSONResponse(w, http.StatusOK, job)
}

// List handles GET /api/v1/accounts/import
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := api.GetTenantID(r.Context())
	if tenantID == "" {
		api.Unauthorized(w, "authentication required")
		return
	}

	tenantUUID, _ := uuid.Parse(tenantID)

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	jobs, total, err := h.repo.List(r.Context(), tenantUUID, limit, offset)
	if err != nil {
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Helper types for io.Reader
type stringReader string

func (s stringReader) Read(p []byte) (n int, err error) {
	return copy(p, s), io.EOF
}

type bytesReader []byte

func (b bytesReader) Read(p []byte) (n int, err error) {
	return copy(p, b), io.EOF
}
