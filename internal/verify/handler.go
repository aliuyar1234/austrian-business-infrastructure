package verify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// Handler provides HTTP handlers for signature verification
type Handler struct {
	service *Service
}

// NewHandler creates a new verification handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// VerificationResponse is the response for a verification
type VerificationResponse struct {
	ID             string          `json:"id,omitempty"`
	IsValid        bool            `json:"is_valid"`
	Status         string          `json:"status"`
	DocumentHash   string          `json:"document_hash"`
	SignatureCount int             `json:"signature_count"`
	Signatures     []SignatureInfo `json:"signatures"`
	Warnings       []string        `json:"warnings,omitempty"`
	Errors         []string        `json:"errors,omitempty"`
	VerifiedAt     string          `json:"verified_at"`
}

// VerifyUpload handles POST /api/v1/verify
// Accepts a multipart form with a PDF file
func (h *Handler) VerifyUpload(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Check file type
	contentType := header.Header.Get("Content-Type")
	if contentType != "application/pdf" && contentType != "" {
		// Try to detect from filename
		filename := header.Filename
		if len(filename) < 4 || filename[len(filename)-4:] != ".pdf" {
			writeError(w, http.StatusBadRequest, "only PDF files are supported")
			return
		}
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	// Verify
	result, err := h.service.VerifyDocument(r.Context(), content, header.Filename, tenantID, &userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toVerificationResponse(result))
}

// VerifyDocument handles GET /api/v1/documents/{id}/verification
func (h *Handler) VerifyDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := getContextIDs(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	docID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	// Get document store from context or dependency injection
	docStore := getDocumentStore(r)
	if docStore == nil {
		writeError(w, http.StatusInternalServerError, "document store not available")
		return
	}

	result, err := h.service.VerifyDocumentByID(r.Context(), docID, tenantID, &userID, docStore)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toVerificationResponse(result))
}

// GetVerification handles GET /api/v1/verifications/{id}
func (h *Handler) GetVerification(w http.ResponseWriter, r *http.Request) {
	verifyID, err := uuid.Parse(getPathParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid verification id")
		return
	}

	verification, err := h.service.GetVerification(r.Context(), verifyID)
	if err != nil {
		writeError(w, http.StatusNotFound, "verification not found")
		return
	}

	// Parse stored signatures
	var signatures []SignatureInfo
	if len(verification.Signatures) > 0 {
		json.Unmarshal(verification.Signatures, &signatures)
	}

	resp := VerificationResponse{
		ID:             verification.ID.String(),
		IsValid:        verification.IsValid,
		Status:         string(verification.VerificationStatus),
		DocumentHash:   verification.DocumentHash,
		SignatureCount: verification.SignatureCount,
		Signatures:     signatures,
		VerifiedAt:     verification.VerifiedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	writeJSON(w, http.StatusOK, resp)
}

func toVerificationResponse(result *VerificationResult) *VerificationResponse {
	return &VerificationResponse{
		IsValid:        result.IsValid,
		Status:         string(result.Status),
		DocumentHash:   result.DocumentHash,
		SignatureCount: result.SignatureCount,
		Signatures:     result.Signatures,
		Warnings:       result.Warnings,
		Errors:         result.Errors,
		VerifiedAt:     result.VerifiedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// Helper function stubs
func getContextIDs(r *http.Request) (tenantID uuid.UUID, userID uuid.UUID, ok bool) {
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
	return r.PathValue(name)
}

func getDocumentStore(r *http.Request) interface {
	GetDocumentContent(ctx context.Context, documentID uuid.UUID) ([]byte, error)
	StoreSignedDocument(ctx context.Context, tenantID, originalDocID uuid.UUID, content []byte, title string) (uuid.UUID, error)
} {
	// Get from context or use dependency injection
	return nil
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
