package payment

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"austrian-business-infrastructure/internal/api"
	"github.com/google/uuid"
)

// Handler handles payment HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new payment handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers payment routes
func (h *Handler) RegisterRoutes(router *api.Router, requireAuth, requireAdmin func(http.Handler) http.Handler) {
	// Admin-only: create, delete, import (financial operations)
	router.Handle("POST /api/v1/payments/batches", requireAuth(requireAdmin(http.HandlerFunc(h.CreateBatch))))
	router.Handle("DELETE /api/v1/payments/batches/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.DeleteBatch))))
	router.Handle("POST /api/v1/payments/batches/import", requireAuth(requireAdmin(http.HandlerFunc(h.ImportCSV))))
	router.Handle("POST /api/v1/payments/statements", requireAuth(requireAdmin(http.HandlerFunc(h.ImportStatement))))
	router.Handle("DELETE /api/v1/payments/statements/{id}", requireAuth(requireAdmin(http.HandlerFunc(h.DeleteStatement))))
	router.Handle("POST /api/v1/payments/transactions/{id}/match", requireAuth(requireAdmin(http.HandlerFunc(h.MatchTransaction))))

	// Member access: read-only, validation, and generate operations
	router.Handle("GET /api/v1/payments/batches", requireAuth(http.HandlerFunc(h.ListBatches)))
	router.Handle("GET /api/v1/payments/batches/{id}", requireAuth(http.HandlerFunc(h.GetBatch)))
	router.Handle("POST /api/v1/payments/batches/{id}/validate", requireAuth(http.HandlerFunc(h.ValidateBatch)))
	router.Handle("POST /api/v1/payments/batches/{id}/generate", requireAuth(http.HandlerFunc(h.GenerateXML)))
	router.Handle("GET /api/v1/payments/batches/{id}/xml", requireAuth(http.HandlerFunc(h.GetXML)))
	router.Handle("GET /api/v1/payments/statements", requireAuth(http.HandlerFunc(h.ListStatements)))
	router.Handle("GET /api/v1/payments/statements/{id}", requireAuth(http.HandlerFunc(h.GetStatement)))
}

// CreateBatch handles POST /api/v1/payments/batches
func (h *Handler) CreateBatch(w http.ResponseWriter, r *http.Request) {
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

	var input CreateBatchInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if input.Name == "" {
		api.BadRequest(w, "name is required")
		return
	}
	if input.DebtorName == "" {
		api.BadRequest(w, "debtor_name is required")
		return
	}
	if input.DebtorIBAN == "" {
		api.BadRequest(w, "debtor_iban is required")
		return
	}

	batch, err := h.service.CreateBatch(r.Context(), tenantID, userID, &input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toBatchResponse(batch, nil))
}

// ListBatches handles GET /api/v1/payments/batches
func (h *Handler) ListBatches(w http.ResponseWriter, r *http.Request) {
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

	if typeParam := r.URL.Query().Get("type"); typeParam != "" {
		filter.Type = &typeParam
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
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

	batches, total, err := h.service.ListBatches(r.Context(), filter)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]*BatchResponse, 0, len(batches))
	for _, batch := range batches {
		items = append(items, h.toBatchResponse(batch, nil))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetBatch handles GET /api/v1/payments/batches/{id}
func (h *Handler) GetBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid batch ID")
		return
	}

	batch, items, err := h.service.GetBatchWithItems(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toBatchResponse(batch, items))
}

// DeleteBatch handles DELETE /api/v1/payments/batches/{id}
func (h *Handler) DeleteBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid batch ID")
		return
	}

	if err := h.service.DeleteBatch(r.Context(), id, tenantID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ValidateBatch handles POST /api/v1/payments/batches/{id}/validate
func (h *Handler) ValidateBatch(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid batch ID")
		return
	}

	batch, err := h.service.ValidateBatch(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toBatchResponse(batch, nil))
}

// GenerateXML handles POST /api/v1/payments/batches/{id}/generate
func (h *Handler) GenerateXML(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid batch ID")
		return
	}

	xmlContent, err := h.service.GenerateXML(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=payment.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(xmlContent)
}

// GetXML handles GET /api/v1/payments/batches/{id}/xml
func (h *Handler) GetXML(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid batch ID")
		return
	}

	xmlContent, err := h.service.GetBatchXML(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	if xmlContent == nil {
		api.NotFound(w, "XML not generated yet")
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", "attachment; filename=payment.xml")
	w.WriteHeader(http.StatusOK)
	w.Write(xmlContent)
}

// ImportCSV handles POST /api/v1/payments/batches/import
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

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		api.BadRequest(w, "failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		api.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	csvData, err := io.ReadAll(file)
	if err != nil {
		api.BadRequest(w, "failed to read file")
		return
	}

	name := r.FormValue("name")
	if name == "" {
		name = "CSV Import"
	}
	debtorName := r.FormValue("debtor_name")
	debtorIBAN := r.FormValue("debtor_iban")

	if debtorName == "" || debtorIBAN == "" {
		api.BadRequest(w, "debtor_name and debtor_iban are required")
		return
	}

	batch, err := h.service.ImportCSVBatch(r.Context(), tenantID, userID, name, debtorName, debtorIBAN, csvData)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toBatchResponse(batch, nil))
}

// ImportStatement handles POST /api/v1/payments/statements
func (h *Handler) ImportStatement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		api.BadRequest(w, "failed to parse multipart form")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		api.BadRequest(w, "file is required")
		return
	}
	defer file.Close()

	xmlData, err := io.ReadAll(file)
	if err != nil {
		api.BadRequest(w, "failed to read file")
		return
	}

	stmt, err := h.service.ImportBankStatement(r.Context(), tenantID, xmlData)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusCreated, h.toStatementResponse(stmt, nil))
}

// ListStatements handles GET /api/v1/payments/statements
func (h *Handler) ListStatements(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	limit := 50
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	statements, total, err := h.service.ListStatements(r.Context(), tenantID, limit, offset)
	if err != nil {
		api.InternalError(w)
		return
	}

	items := make([]*StatementResponse, 0, len(statements))
	for _, stmt := range statements {
		items = append(items, h.toStatementResponse(stmt, nil))
	}

	api.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetStatement handles GET /api/v1/payments/statements/{id}
func (h *Handler) GetStatement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid statement ID")
		return
	}

	stmt, txns, err := h.service.GetStatementWithTransactions(r.Context(), id, tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	api.JSONResponse(w, http.StatusOK, h.toStatementResponse(stmt, txns))
}

// DeleteStatement handles DELETE /api/v1/payments/statements/{id}
func (h *Handler) DeleteStatement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.getTenantID(r)
	if err != nil {
		api.Unauthorized(w, "tenant not found in context")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid statement ID")
		return
	}

	if err := h.service.DeleteStatement(r.Context(), id, tenantID); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// MatchTransaction handles POST /api/v1/payments/transactions/{id}/match
func (h *Handler) MatchTransaction(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		api.BadRequest(w, "invalid transaction ID")
		return
	}

	var input struct {
		PaymentID *uuid.UUID `json:"payment_id,omitempty"`
		InvoiceID *uuid.UUID `json:"invoice_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		api.BadRequest(w, "invalid request body")
		return
	}

	if err := h.service.MatchTransaction(r.Context(), id, input.PaymentID, input.InvoiceID); err != nil {
		api.InternalError(w)
		return
	}

	api.JSONResponse(w, http.StatusOK, map[string]string{"status": "matched"})
}

// Helper methods

func (h *Handler) getTenantID(r *http.Request) (uuid.UUID, error) {
	tenantIDStr := api.GetTenantID(r.Context())
	if tenantIDStr == "" {
		return uuid.Nil, ErrBatchNotFound
	}
	return uuid.Parse(tenantIDStr)
}

func (h *Handler) getUserID(r *http.Request) (uuid.UUID, error) {
	userIDStr := api.GetUserID(r.Context())
	if userIDStr == "" {
		return uuid.Nil, ErrBatchNotFound
	}
	return uuid.Parse(userIDStr)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch err {
	case ErrBatchNotFound:
		api.NotFound(w, "batch not found")
	case ErrStatementNotFound:
		api.NotFound(w, "statement not found")
	case ErrBatchNotDraft:
		api.BadRequest(w, "batch is not in draft status")
	case ErrNoItems:
		api.BadRequest(w, "batch must have at least one item")
	case ErrInvalidBatchType:
		api.BadRequest(w, "invalid batch type, must be 'pain.001' or 'pain.008'")
	default:
		api.InternalError(w)
	}
}

func (h *Handler) toBatchResponse(batch *Batch, items []*Item) *BatchResponse {
	resp := &BatchResponse{
		ID:               batch.ID,
		Name:             batch.Name,
		Type:             batch.Type,
		DebtorName:       batch.DebtorName,
		DebtorIBAN:       batch.DebtorIBAN,
		ItemCount:        batch.ItemCount,
		TotalAmount:      float64(batch.TotalAmount) / 100,
		Status:           batch.Status,
		ValidationErrors: batch.ValidationErrors,
		HasXML:           batch.GeneratedAt != nil,
		CreatedAt:        batch.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if batch.ExecutionDate != nil {
		d := batch.ExecutionDate.Format("2006-01-02")
		resp.ExecutionDate = &d
	}
	if batch.GeneratedAt != nil {
		d := batch.GeneratedAt.Format("2006-01-02T15:04:05Z")
		resp.GeneratedAt = &d
	}
	if batch.SentAt != nil {
		d := batch.SentAt.Format("2006-01-02T15:04:05Z")
		resp.SentAt = &d
	}

	if items != nil {
		resp.Items = make([]ItemResponse, 0, len(items))
		for _, item := range items {
			itemResp := ItemResponse{
				ID:           item.ID,
				EndToEndID:   item.EndToEndID,
				Amount:       float64(item.Amount) / 100,
				Currency:     item.Currency,
				CreditorName: item.CreditorName,
				CreditorIBAN: item.CreditorIBAN,
				Status:       item.Status,
			}
			if item.RemittanceInfo != nil {
				itemResp.RemittanceInfo = item.RemittanceInfo
			}
			if item.ErrorMessage != nil {
				itemResp.ErrorMessage = item.ErrorMessage
			}
			resp.Items = append(resp.Items, itemResp)
		}
	}

	return resp
}

// StatementResponse is the API response format for bank statements
type StatementResponse struct {
	ID             uuid.UUID             `json:"id"`
	IBAN           string                `json:"iban"`
	StatementID    string                `json:"statement_id"`
	StatementDate  string                `json:"statement_date"`
	OpeningBalance float64               `json:"opening_balance"`
	ClosingBalance float64               `json:"closing_balance"`
	EntryCount     int                   `json:"entry_count"`
	ImportedAt     string                `json:"imported_at"`
	Transactions   []TransactionResponse `json:"transactions,omitempty"`
}

// TransactionResponse is the API response format for transactions
type TransactionResponse struct {
	ID               uuid.UUID  `json:"id"`
	Amount           float64    `json:"amount"`
	Currency         string     `json:"currency"`
	CreditDebit      string     `json:"credit_debit"`
	BookingDate      string     `json:"booking_date"`
	ValueDate        *string    `json:"value_date,omitempty"`
	Reference        *string    `json:"reference,omitempty"`
	EndToEndID       *string    `json:"end_to_end_id,omitempty"`
	RemittanceInfo   *string    `json:"remittance_info,omitempty"`
	CounterpartyName *string    `json:"counterparty_name,omitempty"`
	CounterpartyIBAN *string    `json:"counterparty_iban,omitempty"`
	MatchedPaymentID *uuid.UUID `json:"matched_payment_id,omitempty"`
	MatchedInvoiceID *uuid.UUID `json:"matched_invoice_id,omitempty"`
}

func (h *Handler) toStatementResponse(stmt *BankStatement, txns []*Transaction) *StatementResponse {
	resp := &StatementResponse{
		ID:             stmt.ID,
		IBAN:           stmt.IBAN,
		StatementID:    stmt.StatementID,
		StatementDate:  stmt.StatementDate.Format("2006-01-02"),
		OpeningBalance: float64(stmt.OpeningBalance) / 100,
		ClosingBalance: float64(stmt.ClosingBalance) / 100,
		EntryCount:     stmt.EntryCount,
		ImportedAt:     stmt.ImportedAt.Format("2006-01-02T15:04:05Z"),
	}

	if txns != nil {
		resp.Transactions = make([]TransactionResponse, 0, len(txns))
		for _, txn := range txns {
			txnResp := TransactionResponse{
				ID:               txn.ID,
				Amount:           float64(txn.Amount) / 100,
				Currency:         txn.Currency,
				CreditDebit:      txn.CreditDebit,
				BookingDate:      txn.BookingDate.Format("2006-01-02"),
				Reference:        txn.Reference,
				EndToEndID:       txn.EndToEndID,
				RemittanceInfo:   txn.RemittanceInfo,
				CounterpartyName: txn.CounterpartyName,
				CounterpartyIBAN: txn.CounterpartyIBAN,
				MatchedPaymentID: txn.MatchedPaymentID,
				MatchedInvoiceID: txn.MatchedInvoiceID,
			}
			if txn.ValueDate != nil {
				d := txn.ValueDate.Format("2006-01-02")
				txnResp.ValueDate = &d
			}
			resp.Transactions = append(resp.Transactions, txnResp)
		}
	}

	return resp
}
