package invoice

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/api"
	"github.com/google/uuid"
)

// Handler handles invoice HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new invoice handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers invoice routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: create, delete invoices
	router.Handle("POST /api/v1/invoices", requireAuth(requireAdmin(http.HandlerFunc(h.Create))))
	router.Handle("DELETE /api/v1/invoices/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.Delete))))

	// Member access: read and generate operations
	router.Handle("GET /api/v1/invoices", requireAuth(http.HandlerFunc(h.List)))
	router.Handle("GET /api/v1/invoices/{id}", requireAuth(http.HandlerFunc(h.Get)))
	router.Handle("POST /api/v1/invoices/{id}/validate", requireAuth(http.HandlerFunc(h.Validate)))
	router.Handle("POST /api/v1/invoices/{id}/generate", requireAuth(http.HandlerFunc(h.Generate)))
	router.Handle("GET /api/v1/invoices/{id}/xml", requireAuth(http.HandlerFunc(h.GetXML)))
}

// Create handles POST /api/v1/invoices
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
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

	var input CreateInvoiceInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if input.InvoiceNumber == "" {
		api.BadRequest(w, "invoice_number is required")
		return
	}
	if input.SellerName == "" {
		api.BadRequest(w, "seller_name is required")
		return
	}
	if input.BuyerName == "" {
		api.BadRequest(w, "buyer_name is required")
		return
	}

	inv, err := h.service.Create(r.Context(), tenantID, userID, &input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toResponse(inv, nil))
}

// List handles GET /api/v1/invoices
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

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}

	if buyerIDStr := r.URL.Query().Get("buyer_id"); buyerIDStr != "" {
		if buyerID, err := uuid.Parse(buyerIDStr); err == nil {
			filter.BuyerID = &buyerID
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

	if search := r.URL.Query().Get("search"); search != "" {
		filter.Search = &search
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

	invoices, total, err := h.service.List(r.Context(), filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]*InvoiceResponse, 0, len(invoices))
	for _, inv := range invoices {
		items = append(items, h.toResponse(inv, nil))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Get handles GET /api/v1/invoices/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid invoice ID")
		return
	}

	inv, items, err := h.service.GetWithItems(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(inv, items))
}

// Delete handles DELETE /api/v1/invoices/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid invoice ID")
		return
	}

	if err := h.service.Delete(r.Context(), id, tenantID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Validate handles POST /api/v1/invoices/{id}/validate
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid invoice ID")
		return
	}

	inv, err := h.service.Validate(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toResponse(inv, nil))
}

// GenerateRequest represents the generate XML request
type GenerateRequest struct {
	Format string `json:"format"` // "xrechnung" or "zugferd"
}

// Generate handles POST /api/v1/invoices/{id}/generate
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid invoice ID")
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if req.Format == "" {
		req.Format = FormatXRechnung
	}
	if req.Format != FormatXRechnung && req.Format != FormatZUGFeRD {
		api.BadRequest(w, "format must be 'xrechnung' or 'zugferd'")
		return
	}

	xmlContent, err := h.service.GenerateXML(r.Context(), id, tenantID, req.Format)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=invoice.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(xmlContent)
}

// GetXML handles GET /api/v1/invoices/{id}/xml
func (h *Handler) GetXML(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid invoice ID")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = FormatXRechnung
	}

	xmlContent, err := h.service.GetXML(r.Context(), id, tenantID, format)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if xmlContent == nil {
		api.NotFound(w, "XML not generated yet")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=invoice.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(xmlContent)
}

// Helper methods

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := api.GetTenantID(r.Context())
	if tenantIDStr == "" {
		return uuid.Nil, ErrInvoiceNotFound
	}
	return uuid.Parse(tenantIDStr)
}

func (h *Handler) getUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr := api.GetUserID(r.Context())
	if userIDStr == "" {
		return uuid.Nil, ErrInvoiceNotFound
	}
	return uuid.Parse(userIDStr)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrInvoiceNotFound:
		api.NotFound(w, "invoice not found")
	case ErrDuplicateNumber:
		api.Conflict(w, "invoice number already exists")
	case ErrInvoiceNotDraft:
		api.BadRequest(w, "invoice is not in draft status")
	case ErrNoItems:
		api.BadRequest(w, "invoice must have at least one item")
	case ErrValidationFailed:
		api.BadRequest(w, "validation failed")
	default:
		api.InternalError(w)
	}
}

func (h *Handler) toResponse(inv *Invoice, items []*InvoiceItem) *InvoiceResponse {
	resp := &InvoiceResponse{
		ID:                 inv.ID,
		InvoiceNumber:      inv.InvoiceNumber,
		InvoiceType:        inv.InvoiceType,
		IssueDate:          inv.IssueDate.Format("2006-01-02"),
		Currency:           inv.Currency,
		SellerName:         inv.SellerName,
		SellerVAT:          inv.SellerVAT,
		BuyerName:          inv.BuyerName,
		BuyerVAT:           inv.BuyerVAT,
		BuyerReference:     inv.BuyerReference,
		TaxExclusiveAmount: float64(inv.TaxExclusiveAmount) / 100,
		TaxAmount:          float64(inv.TaxAmount) / 100,
		TaxInclusiveAmount: float64(inv.TaxInclusiveAmount) / 100,
		PayableAmount:      float64(inv.PayableAmount) / 100,
		Status:             inv.Status,
		ValidationStatus:   inv.ValidationStatus,
		ValidationErrors:   inv.ValidationErrors,
		HasXRechnung:       len(inv.XRechnungXML) > 0,
		HasZUGFeRD:         len(inv.ZUGFeRDXML) > 0,
		HasPDF:             len(inv.PDFContent) > 0,
		CreatedAt:          inv.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          inv.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if inv.DueDate != nil {
		d := inv.DueDate.Format("2006-01-02")
		resp.DueDate = &d
	}

	if items != nil {
		resp.Items = make([]ItemResponse, 0, len(items))
		for _, item := range items {
			resp.Items = append(resp.Items, ItemResponse{
				ID:          item.ID,
				LineNumber:  item.LineNumber,
				Description: item.Description,
				Quantity:    item.Quantity,
				UnitCode:    item.UnitCode,
				UnitPrice:   float64(item.UnitPrice) / 100,
				LineTotal:   float64(item.LineTotal) / 100,
				TaxCategory: item.TaxCategory,
				TaxPercent:  item.TaxPercent,
			})
		}
	}

	return resp
}
