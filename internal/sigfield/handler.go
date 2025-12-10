package sigfield

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// Handler provides HTTP handlers for signature field operations
type Handler struct {
	repo     *Repository
	embedder *Embedder
}

// NewHandler creates a new signature field handler
func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:     repo,
		embedder: NewEmbedder(),
	}
}

// CreateFieldRequest is the request body for creating a signature field
type CreateFieldRequest struct {
	DocumentID uuid.UUID  `json:"document_id"`
	SignerID   *uuid.UUID `json:"signer_id,omitempty"`
	Page       int        `json:"page"`
	X          float64    `json:"x"`
	Y          float64    `json:"y"`
	Width      float64    `json:"width"`
	Height     float64    `json:"height"`
	FieldName  string     `json:"field_name"`
	Required   bool       `json:"required"`
	ShowName   bool       `json:"show_name"`
	ShowDate   bool       `json:"show_date"`
	ShowReason bool       `json:"show_reason"`
	CustomText string     `json:"custom_text,omitempty"`
	FontSize   float64    `json:"font_size"`
}

// UpdateFieldRequest is the request body for updating a signature field
type UpdateFieldRequest struct {
	SignerID   *uuid.UUID `json:"signer_id,omitempty"`
	Page       int        `json:"page"`
	X          float64    `json:"x"`
	Y          float64    `json:"y"`
	Width      float64    `json:"width"`
	Height     float64    `json:"height"`
	FieldName  string     `json:"field_name"`
	Required   bool       `json:"required"`
	ShowName   bool       `json:"show_name"`
	ShowDate   bool       `json:"show_date"`
	ShowReason bool       `json:"show_reason"`
	CustomText string     `json:"custom_text,omitempty"`
	FontSize   float64    `json:"font_size"`
}

// FieldResponse is the response for a signature field
type FieldResponse struct {
	ID         string  `json:"id"`
	DocumentID string  `json:"document_id"`
	SignerID   *string `json:"signer_id,omitempty"`
	Page       int     `json:"page"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Width      float64 `json:"width"`
	Height     float64 `json:"height"`
	FieldName  string  `json:"field_name"`
	Required   bool    `json:"required"`
	ShowName   bool    `json:"show_name"`
	ShowDate   bool    `json:"show_date"`
	ShowReason bool    `json:"show_reason"`
	CustomText string  `json:"custom_text,omitempty"`
	FontSize   float64 `json:"font_size"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

// CreateField handles POST /api/v1/documents/{id}/sigfields
func (h *Handler) CreateField(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := getContextTenantID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	docID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	var req CreateFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Page < 1 {
		writeError(w, http.StatusBadRequest, "page must be at least 1")
		return
	}
	if req.Width <= 0 || req.Height <= 0 {
		writeError(w, http.StatusBadRequest, "width and height must be positive")
		return
	}

	field := &SignatureField{
		DocumentID: docID,
		TenantID:   tenantID,
		SignerID:   req.SignerID,
		Page:       req.Page,
		X:          req.X,
		Y:          req.Y,
		Width:      req.Width,
		Height:     req.Height,
		FieldName:  req.FieldName,
		Required:   req.Required,
		ShowName:   req.ShowName,
		ShowDate:   req.ShowDate,
		ShowReason: req.ShowReason,
		CustomText: req.CustomText,
		FontSize:   req.FontSize,
	}

	if field.FontSize == 0 {
		field.FontSize = 10
	}

	if err := h.repo.CreateField(r.Context(), field); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toFieldResponse(field))
}

// ListFields handles GET /api/v1/documents/{id}/sigfields
func (h *Handler) ListFields(w http.ResponseWriter, r *http.Request) {
	docID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	fields, err := h.repo.ListFieldsByDocument(r.Context(), docID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	responses := make([]*FieldResponse, len(fields))
	for i, f := range fields {
		responses[i] = toFieldResponse(f)
	}

	writeJSON(w, http.StatusOK, responses)
}

// GetField handles GET /api/v1/sigfields/{id}
func (h *Handler) GetField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field id")
		return
	}

	field, err := h.repo.GetFieldByID(r.Context(), fieldID)
	if err != nil {
		writeError(w, http.StatusNotFound, "field not found")
		return
	}

	writeJSON(w, http.StatusOK, toFieldResponse(field))
}

// UpdateField handles PUT /api/v1/sigfields/{id}
func (h *Handler) UpdateField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field id")
		return
	}

	var req UpdateFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get existing field
	field, err := h.repo.GetFieldByID(r.Context(), fieldID)
	if err != nil {
		writeError(w, http.StatusNotFound, "field not found")
		return
	}

	// Update fields
	field.SignerID = req.SignerID
	field.Page = req.Page
	field.X = req.X
	field.Y = req.Y
	field.Width = req.Width
	field.Height = req.Height
	field.FieldName = req.FieldName
	field.Required = req.Required
	field.ShowName = req.ShowName
	field.ShowDate = req.ShowDate
	field.ShowReason = req.ShowReason
	field.CustomText = req.CustomText
	field.FontSize = req.FontSize

	if err := h.repo.UpdateField(r.Context(), field); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toFieldResponse(field))
}

// DeleteField handles DELETE /api/v1/sigfields/{id}
func (h *Handler) DeleteField(w http.ResponseWriter, r *http.Request) {
	fieldID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid field id")
		return
	}

	if err := h.repo.DeleteField(r.Context(), fieldID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// toFieldResponse converts a SignatureField to a FieldResponse
func toFieldResponse(f *SignatureField) *FieldResponse {
	resp := &FieldResponse{
		ID:         f.ID.String(),
		DocumentID: f.DocumentID.String(),
		Page:       f.Page,
		X:          f.X,
		Y:          f.Y,
		Width:      f.Width,
		Height:     f.Height,
		FieldName:  f.FieldName,
		Required:   f.Required,
		ShowName:   f.ShowName,
		ShowDate:   f.ShowDate,
		ShowReason: f.ShowReason,
		CustomText: f.CustomText,
		FontSize:   f.FontSize,
		CreatedAt:  f.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  f.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if f.SignerID != nil {
		signerIDStr := f.SignerID.String()
		resp.SignerID = &signerIDStr
	}

	return resp
}

// Helper functions

func getContextTenantID(r *http.Request) (uuid.UUID, bool) {
	tenantIDValue := r.Context().Value("tenant_id")
	if tenantIDValue == nil {
		return uuid.Nil, false
	}
	tenantID, ok := tenantIDValue.(uuid.UUID)
	return tenantID, ok
}

func getPathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
