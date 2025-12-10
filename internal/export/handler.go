package export

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// SearchRepository interface for search data access
type SearchRepository interface {
	GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.FoerderungsSuche, error)
}

// Handler handles export HTTP requests
type Handler struct {
	searchRepo SearchRepository
}

// NewHandler creates a new export handler
func NewHandler(searchRepo SearchRepository) *Handler {
	return &Handler{searchRepo: searchRepo}
}

// RegisterRoutes registers export routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/foerderungssuche/{id}/export", h.Export)
	r.Get("/foerderungssuche/{id}/export/pdf", h.ExportPDF)
	r.Get("/foerderungssuche/{id}/export/markdown", h.ExportMarkdown)
}

// Export handles GET /api/v1/foerderungssuche/{id}/export
// Returns format based on Accept header or ?format= query param
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		accept := r.Header.Get("Accept")
		switch accept {
		case "application/pdf":
			format = "pdf"
		case "text/markdown":
			format = "markdown"
		default:
			format = "markdown" // Default to markdown
		}
	}

	switch format {
	case "pdf":
		h.ExportPDF(w, r)
	case "markdown", "md":
		h.ExportMarkdown(w, r)
	default:
		writeError(w, http.StatusBadRequest, "Unsupported format: "+format)
	}
}

// ExportPDF handles GET /api/v1/foerderungssuche/{id}/export/pdf
func (h *Handler) ExportPDF(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid search ID")
		return
	}

	search, err := h.searchRepo.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Search not found")
		return
	}

	matches, _ := search.GetMatchesSlice()

	pdfBytes, err := GeneratePDF(search, matches)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to generate PDF")
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"foerderungssuche-%s.pdf\"", search.ID.String()[:8]))
	w.Write(pdfBytes)
}

// ExportMarkdown handles GET /api/v1/foerderungssuche/{id}/export/markdown
func (h *Handler) ExportMarkdown(w http.ResponseWriter, r *http.Request) {
	tenantID, err := getTenantIDFromContext(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid search ID")
		return
	}

	search, err := h.searchRepo.GetByIDAndTenant(r.Context(), id, tenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Search not found")
		return
	}

	matches, _ := search.GetMatchesSlice()

	markdown := GenerateMarkdown(search, matches)

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"foerderungssuche-%s.md\"", search.ID.String()[:8]))
	w.Write([]byte(markdown))
}

// Context helper functions

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
)

func getTenantIDFromContext(r *http.Request) (uuid.UUID, error) {
	v := r.Context().Value(tenantIDKey)
	if v == nil {
		if h := r.Header.Get("X-Tenant-ID"); h != "" {
			return uuid.Parse(h)
		}
		return uuid.Nil, nil
	}
	if id, ok := v.(uuid.UUID); ok {
		return id, nil
	}
	return uuid.Nil, nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
