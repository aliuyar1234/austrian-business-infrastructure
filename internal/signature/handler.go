package signature

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Handler provides HTTP handlers for signature operations
type Handler struct {
	service *Service
}

// NewHandler creates a new signature handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ===== Request Types =====

// CreateRequestPayload is the request body for creating a signature request
type CreateRequestPayload struct {
	DocumentID   string         `json:"document_id"`
	Name         string         `json:"name,omitempty"`
	Message      string         `json:"message,omitempty"`
	IsSequential bool           `json:"is_sequential"`
	ExpiryDays   int            `json:"expiry_days,omitempty"`
	Signers      []SignerPayload `json:"signers"`
	Fields       []FieldPayload  `json:"fields,omitempty"`
}

// SignerPayload is a signer in a request
type SignerPayload struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	OrderIndex int    `json:"order_index"`
}

// FieldPayload is a signature field in a request
type FieldPayload struct {
	SignerIndex int     `json:"signer_index"`
	Page        int     `json:"page"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	ShowName    bool    `json:"show_name"`
	ShowDate    bool    `json:"show_date"`
	ShowReason  bool    `json:"show_reason"`
	Reason      string  `json:"reason,omitempty"`
}

// CreateFromTemplatePayload is the request body for creating from template
type CreateFromTemplatePayload struct {
	TemplateID string          `json:"template_id"`
	DocumentID string          `json:"document_id"`
	Signers    []SignerPayload `json:"signers"`
}

// ===== Response Types =====

// RequestResponse is the response for a signature request
type RequestResponse struct {
	ID               string            `json:"id"`
	DocumentID       string            `json:"document_id"`
	DocumentTitle    string            `json:"document_title,omitempty"`
	Name             string            `json:"name,omitempty"`
	Message          string            `json:"message,omitempty"`
	ExpiresAt        time.Time         `json:"expires_at"`
	Status           string            `json:"status"`
	CompletedAt      *time.Time        `json:"completed_at,omitempty"`
	IsSequential     bool              `json:"is_sequential"`
	SignedDocumentID string            `json:"signed_document_id,omitempty"`
	Signers          []SignerResponse  `json:"signers,omitempty"`
	Fields           []FieldResponse   `json:"fields,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// SignerResponse is a signer in a response
type SignerResponse struct {
	ID                 string     `json:"id"`
	Email              string     `json:"email"`
	Name               string     `json:"name"`
	OrderIndex         int        `json:"order_index"`
	Status             string     `json:"status"`
	NotifiedAt         *time.Time `json:"notified_at,omitempty"`
	SignedAt           *time.Time `json:"signed_at,omitempty"`
	CertificateSubject string     `json:"certificate_subject,omitempty"`
	CertificateIssuer  string     `json:"certificate_issuer,omitempty"`
	ReminderCount      int        `json:"reminder_count"`
	LastReminderAt     *time.Time `json:"last_reminder_at,omitempty"`
}

// FieldResponse is a field in a response
type FieldResponse struct {
	ID         string  `json:"id"`
	SignerID   string  `json:"signer_id,omitempty"`
	Page       int     `json:"page"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	ShowName   bool    `json:"show_name"`
	ShowDate   bool    `json:"show_date"`
	ShowReason bool    `json:"show_reason"`
	Reason     string  `json:"reason,omitempty"`
}

// ListResponse is a paginated list response
type ListResponse struct {
	Items      []RequestResponse `json:"items"`
	Total      int               `json:"total"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
}

// SigningInfoResponse is the response for signing info
type SigningInfoResponse struct {
	Request *RequestResponse `json:"request"`
	Signer  *SignerResponse  `json:"signer"`
}

// ===== Handlers =====

// CreateRequest handles POST /api/v1/signatures
func (h *Handler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload CreateRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	docID, err := uuid.Parse(payload.DocumentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document_id")
		return
	}

	if len(payload.Signers) == 0 {
		writeError(w, http.StatusBadRequest, "at least one signer is required")
		return
	}

	// Convert payload to input
	signers := make([]SignerInput, len(payload.Signers))
	for i, s := range payload.Signers {
		signers[i] = SignerInput{
			Email:      s.Email,
			Name:       s.Name,
			OrderIndex: s.OrderIndex,
		}
	}

	fields := make([]FieldInput, len(payload.Fields))
	for i, f := range payload.Fields {
		fields[i] = FieldInput{
			SignerIndex: f.SignerIndex,
			Page:        f.Page,
			X:           f.X,
			Y:           f.Y,
			Width:       f.Width,
			Height:      f.Height,
			ShowName:    f.ShowName,
			ShowDate:    f.ShowDate,
			ShowReason:  f.ShowReason,
			Reason:      f.Reason,
		}
	}

	input := &CreateRequestInput{
		TenantID:     tenantID,
		DocumentID:   docID,
		Name:         payload.Name,
		Message:      payload.Message,
		IsSequential: payload.IsSequential,
		ExpiryDays:   payload.ExpiryDays,
		Signers:      signers,
		Fields:       fields,
		CreatedBy:    userID,
	}

	req, err := h.service.CreateRequest(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify signers
	h.service.NotifySigners(r.Context(), req.ID)

	writeJSON(w, http.StatusCreated, toRequestResponse(req))
}

// GetRequest handles GET /api/v1/signatures/{id}
func (h *Handler) GetRequest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request id")
		return
	}

	req, err := h.service.GetRequest(r.Context(), id)
	if err != nil {
		if err == ErrRequestNotFound {
			writeError(w, http.StatusNotFound, "request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toRequestResponse(req))
}

// ListRequests handles GET /api/v1/signatures
func (h *Handler) ListRequests(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse query params
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	var status *RequestStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := RequestStatus(s)
		status = &st
	}

	requests, total, err := h.service.ListRequests(r.Context(), tenantID, status, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	items := make([]RequestResponse, len(requests))
	for i, req := range requests {
		items[i] = *toRequestResponse(req)
	}

	writeJSON(w, http.StatusOK, ListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

// CancelRequest handles DELETE /api/v1/signatures/{id}
func (h *Handler) CancelRequest(w http.ResponseWriter, r *http.Request) {
	_, userID, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request id")
		return
	}

	if err := h.service.CancelRequest(r.Context(), id, userID); err != nil {
		if err == ErrRequestNotFound {
			writeError(w, http.StatusNotFound, "request not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// SendReminder handles POST /api/v1/signatures/{id}/remind
func (h *Handler) SendReminder(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request id")
		return
	}

	// Get signer_id from body or query
	signerIDStr := r.URL.Query().Get("signer_id")
	if signerIDStr == "" {
		var body struct {
			SignerID string `json:"signer_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
			signerIDStr = body.SignerID
		}
	}

	if signerIDStr == "" {
		writeError(w, http.StatusBadRequest, "signer_id is required")
		return
	}

	signerID, err := uuid.Parse(signerIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid signer_id")
		return
	}

	// Verify signer belongs to this request
	req, err := h.service.GetRequest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "request not found")
		return
	}

	found := false
	for _, s := range req.Signers {
		if s.ID == signerID {
			found = true
			break
		}
	}
	if !found {
		writeError(w, http.StatusBadRequest, "signer does not belong to this request")
		return
	}

	if err := h.service.SendReminder(r.Context(), signerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// CreateFromTemplate handles POST /api/v1/signatures/from-template
func (h *Handler) CreateFromTemplate(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var payload CreateFromTemplatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templateID, err := uuid.Parse(payload.TemplateID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid template_id")
		return
	}

	docID, err := uuid.Parse(payload.DocumentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document_id")
		return
	}

	signers := make([]SignerInput, len(payload.Signers))
	for i, s := range payload.Signers {
		signers[i] = SignerInput{
			Email:      s.Email,
			Name:       s.Name,
			OrderIndex: s.OrderIndex,
		}
	}

	req, err := h.service.CreateFromTemplate(r.Context(), templateID, docID, signers, tenantID, userID)
	if err != nil {
		if err == ErrTemplateNotFound {
			writeError(w, http.StatusNotFound, "template not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Notify signers
	h.service.NotifySigners(r.Context(), req.ID)

	writeJSON(w, http.StatusCreated, toRequestResponse(req))
}

// ===== Signing Handlers (public endpoints for signers) =====

// GetSigningInfo handles GET /api/v1/sign/{token}
func (h *Handler) GetSigningInfo(w http.ResponseWriter, r *http.Request) {
	token := getPathParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	req, signer, err := h.service.GetSigningInfo(r.Context(), token)
	if err != nil {
		if err == ErrInvalidToken {
			writeError(w, http.StatusNotFound, "invalid or expired signing link")
			return
		}
		if err == ErrAlreadySigned {
			writeError(w, http.StatusConflict, "document already signed")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SigningInfoResponse{
		Request: toRequestResponse(req),
		Signer:  toSignerResponse(signer),
	})
}

// StartSigning handles GET /api/v1/sign/{token}/auth
func (h *Handler) StartSigning(w http.ResponseWriter, r *http.Request) {
	token := getPathParam(r, "token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "token is required")
		return
	}

	authURL, err := h.service.StartSigning(r.Context(), token)
	if err != nil {
		if err == ErrInvalidToken {
			writeError(w, http.StatusNotFound, "invalid or expired signing link")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Redirect to ID Austria
	http.Redirect(w, r, authURL, http.StatusFound)
}

// SigningCallback handles GET /api/v1/sign/{token}/callback
func (h *Handler) SigningCallback(w http.ResponseWriter, r *http.Request) {
	token := getPathParam(r, "token")
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorCode := r.URL.Query().Get("error")
	errorDesc := r.URL.Query().Get("error_description")

	input := &CompleteSigningInput{
		Token:     token,
		State:     state,
		Code:      code,
		Error:     errorCode,
		ErrorDesc: errorDesc,
		IP:        getClientIP(r),
		UserAgent: r.UserAgent(),
	}

	req, err := h.service.CompleteSigning(r.Context(), input)
	if err != nil {
		// Redirect to error page
		// TODO: Use proper frontend error URL
		http.Redirect(w, r, "/sign/error?message="+err.Error(), http.StatusFound)
		return
	}

	// Redirect to success page
	// TODO: Use proper frontend success URL
	successURL := "/sign/success?request=" + req.ID.String()
	if req.Status == RequestStatusCompleted {
		successURL += "&complete=true"
	}
	http.Redirect(w, r, successURL, http.StatusFound)
}

// ===== Helper Functions =====

func toRequestResponse(req *SignatureRequest) *RequestResponse {
	resp := &RequestResponse{
		ID:           req.ID.String(),
		DocumentID:   req.DocumentID.String(),
		DocumentTitle: req.DocumentTitle,
		ExpiresAt:    req.ExpiresAt,
		Status:       string(req.Status),
		CompletedAt:  req.CompletedAt,
		IsSequential: req.IsSequential,
		CreatedAt:    req.CreatedAt,
		UpdatedAt:    req.UpdatedAt,
	}

	if req.Name != nil {
		resp.Name = *req.Name
	}
	if req.Message != nil {
		resp.Message = *req.Message
	}
	if req.SignedDocumentID != nil {
		resp.SignedDocumentID = req.SignedDocumentID.String()
	}

	if len(req.Signers) > 0 {
		resp.Signers = make([]SignerResponse, len(req.Signers))
		for i, s := range req.Signers {
			resp.Signers[i] = *toSignerResponse(s)
		}
	}

	if len(req.Fields) > 0 {
		resp.Fields = make([]FieldResponse, len(req.Fields))
		for i, f := range req.Fields {
			resp.Fields[i] = *toFieldResponse(f)
		}
	}

	return resp
}

func toSignerResponse(signer *Signer) *SignerResponse {
	resp := &SignerResponse{
		ID:             signer.ID.String(),
		Email:          signer.Email,
		Name:           signer.Name,
		OrderIndex:     signer.OrderIndex,
		Status:         string(signer.Status),
		NotifiedAt:     signer.NotifiedAt,
		SignedAt:       signer.SignedAt,
		ReminderCount:  signer.ReminderCount,
		LastReminderAt: signer.LastReminderAt,
	}

	if signer.CertificateSubject != nil {
		resp.CertificateSubject = *signer.CertificateSubject
	}
	if signer.CertificateIssuer != nil {
		resp.CertificateIssuer = *signer.CertificateIssuer
	}

	return resp
}

func toFieldResponse(field *Field) *FieldResponse {
	resp := &FieldResponse{
		ID:         field.ID.String(),
		Page:       field.Page,
		X:          field.X,
		Y:          field.Y,
		Width:      field.Width,
		Height:     field.Height,
		ShowName:   field.ShowName,
		ShowDate:   field.ShowDate,
		ShowReason: field.ShowReason,
	}

	if field.SignerID != nil {
		resp.SignerID = field.SignerID.String()
	}
	if field.Reason != nil {
		resp.Reason = *field.Reason
	}

	return resp
}

// Helper function stubs - these should be implemented based on your router/middleware
func getContextIDs(r *http.Request) (tenantID uuid.UUID, userID uuid.UUID, ok bool) {
	// Get from context set by auth middleware
	tenantIDValue := r.Context().Value("tenant_id")
	userIDValue := r.Context().Value("user_id")

	if tenantIDValue == nil || userIDValue == nil {
		return uuid.Nil, uuid.Nil, false
	}

	tenantID, ok1 := tenantIDValue.(uuid.UUID)
	userID, ok2 := userIDValue.(uuid.UUID)

	return tenantID, userID, ok1 && ok2
}

func getPathParam(r *http.Request, name string) string {
	// This depends on your router (chi, gorilla mux, etc.)
	// Example for chi: chi.URLParam(r, name)
	// Example for gorilla mux: mux.Vars(r)[name]
	// Placeholder implementation:
	return r.PathValue(name)
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
