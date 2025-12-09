package uid

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/google/uuid"
)

// Handler handles UID validation HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new UID handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers UID validation routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: batch operations and imports (consume API quota)
	router.Handle("POST /api/v1/uid/validate/batch", requireAuth(requireAdmin(http.HandlerFunc(h.ValidateBatch))))
	router.Handle("POST /api/v1/uid/import", requireAuth(requireAdmin(http.HandlerFunc(h.ImportCSV))))

	// Member access: single validation, format check, read operations
	router.Handle("POST /api/v1/uid/validate", requireAuth(http.HandlerFunc(h.Validate)))
	router.Handle("POST /api/v1/uid/validate/format", requireAuth(http.HandlerFunc(h.ValidateFormat)))
	router.Handle("GET /api/v1/uid/validations", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/uid/validations/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("GET /api/v1/uid/validations/export", requireAuth(http.HandlerFunc(h.Export)))
}

// ValidateRequest represents the validate UID request
type ValidateRequest struct {
	UID       string `json:"uid"`
	Level     int    `json:"level"`
	AccountID string `json:"account_id"`
}

// Validate handles POST /api/v1/uid/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	userID, err := h.getUserID(r)
	if err != nil {
		api.Unauthorized(w, "user not found in context")
		return
	}

	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if req.UID == "" {
		api.BadRequest(w, "uid is required")
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		api.BadRequest(w, "invalid account_id")
		return
	}

	input := &ValidateInput{
		UID:       req.UID,
		Level:     req.Level,
		AccountID: accountID,
	}

	validation, err := h.service.Validate(r.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(validation))
}

// ValidateBatchRequest represents the batch validate request
type ValidateBatchRequest struct {
	UIDs      []string `json:"uids"`
	Level     int      `json:"level"`
	AccountID string   `json:"account_id"`
}

// ValidateBatch handles POST /api/v1/uid/validate/batch
func (h *Handler) ValidateBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	userID, err := h.getUserID(r)
	if err != nil {
		api.Unauthorized(w, "user not found in context")
		return
	}

	var req ValidateBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if len(req.UIDs) == 0 {
		api.BadRequest(w, "at least one uid is required")
		return
	}

	if len(req.UIDs) > 100 {
		api.BadRequest(w, "maximum 100 UIDs per batch")
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		api.BadRequest(w, "invalid account_id")
		return
	}

	input := &ValidateBatchInput{
		UIDs:      req.UIDs,
		Level:     req.Level,
		AccountID: accountID,
	}

	validations, err := h.service.ValidateBatch(r.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	var validCount, invalidCount int
	results := make([]*ValidationResponse, 0, len(validations))
	for _, v := range validations {
		if v.Valid {
			validCount++
		} else {
			invalidCount++
		}
		results = append(results, h.toResponse(v))
	}

	api.JSONResponse(w, http.StatusOK, BatchValidationResponse{
		Total:       len(validations),
		Valid:       validCount,
		Invalid:     invalidCount,
		Results:     results,
		ProcessedAt: time.Now().Format("2006-01-02T15:04:05Z"),
	})
}

// ValidateFormatRequest represents the format validation request
type ValidateFormatRequest struct {
	UIDs []string `json:"uids"`
}

// ValidateFormat handles POST /api/v1/uid/validate/format
func (h *Handler) ValidateFormat(w http.ResponseWriter, r *http.Request) {
	var req ValidateFormatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if len(req.UIDs) == 0 {
		api.BadRequest(w, "at least one uid is required")
		return
	}

	if len(req.UIDs) > 1000 {
		api.BadRequest(w, "maximum 1000 UIDs per request")
		return
	}

	results := make([]*FormatValidationResult, 0, len(req.UIDs))
	for _, uid := range req.UIDs {
		results = append(results, h.service.ValidateFormat(uid))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"total":   len(results),
		"results": results,
	})
}

// List handles GET /api/v1/uid/validations
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	filter := ListFilter{
		TenantID: tenantID,
		Limit:    50,
		Offset:   0,
	}

	if accountIDStr := r.URL.Query().Get("account_id"); accountIDStr != "" {
		if accountID, err := uuid.Parse(accountIDStr); err == nil {
			filter.AccountID = &accountID
		}
	}

	if uid := r.URL.Query().Get("uid"); uid != "" {
		filter.UID = &uid
	}

	if validStr := r.URL.Query().Get("valid"); validStr != "" {
		valid := validStr == "true"
		filter.Valid = &valid
	}

	if countryCode := r.URL.Query().Get("country_code"); countryCode != "" {
		filter.CountryCode = &countryCode
	}

	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			filter.DateFrom = &dateFrom
		}
	}

	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = dateTo.Add(24*time.Hour - time.Nanosecond) // End of day
			filter.DateTo = &dateTo
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit <= 100 {
			filter.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	validations, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]*ValidationResponse, 0, len(validations))
	for _, v := range validations {
		items = append(items, h.toResponse(v))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Get handles GET /api/v1/uid/validations/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid validation ID")
		return
	}

	validation, err := h.service.Get(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(validation))
}

// Export handles GET /api/v1/uid/validations/export
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	filter := ListFilter{
		TenantID: tenantID,
	}

	if accountIDStr := r.URL.Query().Get("account_id"); accountIDStr != "" {
		if accountID, err := uuid.Parse(accountIDStr); err == nil {
			filter.AccountID = &accountID
		}
	}

	if dateFromStr := r.URL.Query().Get("date_from"); dateFromStr != "" {
		if dateFrom, err := time.Parse("2006-01-02", dateFromStr); err == nil {
			filter.DateFrom = &dateFrom
		}
	}

	if dateToStr := r.URL.Query().Get("date_to"); dateToStr != "" {
		if dateTo, err := time.Parse("2006-01-02", dateToStr); err == nil {
			dateTo = dateTo.Add(24*time.Hour - time.Nanosecond)
			filter.DateTo = &dateTo
		}
	}

	csvData, err := h.service.ExportCSV(r.Context(), tenantID, filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=uid_validations.csv")
	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}

// ImportCSV handles POST /api/v1/uid/import
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	userID, err := h.getUserID(r)
	if err != nil {
		api.Unauthorized(w, "user not found in context")
		return
	}

	accountIDStr := r.URL.Query().Get("account_id")
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		api.BadRequest(w, "invalid account_id")
		return
	}

	levelStr := r.URL.Query().Get("level")
	level := 1
	if levelStr != "" {
		if l, err := strconv.Atoi(levelStr); err == nil && (l == 1 || l == 2) {
			level = l
		}
	}

	var csvData []byte

	contentType := r.Header.Get("Content-Type")
	if contentType == "text/csv" || contentType == "application/csv" {
		csvData, err = io.ReadAll(r.Body)
		if err != nil {
			api.BadRequest(w, "failed to read CSV data")
			return
		}
	} else {
		// Multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			api.BadRequest(w, "failed to parse multipart form")
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			api.BadRequest(w, "file is required")
			return
		}
		defer file.Close()

		csvData, err = io.ReadAll(file)
		if err != nil {
			api.BadRequest(w, "failed to read file")
			return
		}
	}

	// Parse UIDs from CSV
	uids, err := fonws.ParseUIDCSV(csvData)
	if err != nil {
		api.BadRequest(w, err.Error())
		return
	}

	if len(uids) == 0 {
		api.BadRequest(w, "no UIDs found in CSV")
		return
	}

	if len(uids) > 100 {
		api.BadRequest(w, "maximum 100 UIDs per import")
		return
	}

	input := &ValidateBatchInput{
		UIDs:      uids,
		Level:     level,
		AccountID: accountID,
	}

	validations, err := h.service.ValidateBatch(r.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	var validCount, invalidCount int
	results := make([]*ValidationResponse, 0, len(validations))
	for _, v := range validations {
		if v.Valid {
			validCount++
		} else {
			invalidCount++
		}
		results = append(results, h.toResponse(v))
	}

	api.JSONResponse(w, http.StatusOK, BatchValidationResponse{
		Total:       len(validations),
		Valid:       validCount,
		Invalid:     invalidCount,
		Results:     results,
		ProcessedAt: time.Now().Format("2006-01-02T15:04:05Z"),
	})
}

// Helper methods

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := api.GetTenantID(r.Context())
	if tenantIDStr == "" {
		return uuid.Nil, ErrAccountNotFound
	}
	return uuid.Parse(tenantIDStr)
}

func (h *Handler) getUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr := api.GetUserID(r.Context())
	if userIDStr == "" {
		return uuid.Nil, ErrAccountNotFound
	}
	return uuid.Parse(userIDStr)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrValidationNotFound:
		api.NotFound(w, "validation not found")
	case ErrAccountNotFound:
		api.NotFound(w, "account not found")
	case ErrInvalidLevel:
		api.BadRequest(w, "level must be 1 or 2")
	case ErrDailyLimit:
		api.JSONError(w, http.StatusTooManyRequests, "daily validation limit exceeded", "DAILY_LIMIT")
	case ErrInvalidUID:
		api.BadRequest(w, "invalid UID format")
	default:
		api.InternalError(w)
	}
}

func (h *Handler) toResponse(v *Validation) *ValidationResponse {
	resp := &ValidationResponse{
		ID:           v.ID,
		UID:          v.UID,
		CountryCode:  v.CountryCode,
		Valid:        v.Valid,
		Level:        v.Level,
		CompanyName:  v.CompanyName,
		Street:       v.Street,
		PostCode:     v.PostCode,
		City:         v.City,
		Country:      v.Country,
		ErrorMessage: v.ErrorMessage,
		Source:       v.Source,
		ValidatedAt:  v.ValidatedAt.Format("2006-01-02T15:04:05Z"),
		CreatedAt:    v.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	return resp
}
