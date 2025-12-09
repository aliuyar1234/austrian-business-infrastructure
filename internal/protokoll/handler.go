package protokoll

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler handles HTTP requests for ELDA protokoll operations
type Handler struct {
	repo *Repository
}

// NewHandler creates a new protokoll handler
func NewHandler(db *pgxpool.Pool) *Handler {
	return &Handler{repo: NewRepository(db)}
}

// Routes returns the router for protokoll endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Get("/by-nummer/{nummer}", h.GetByProtokollnummer)
	r.Get("/history/{sv_nummer}", h.GetHistoryBySVNummer)
	r.Get("/statistics", h.GetStatistics)

	return r
}

// List handles GET /api/v1/elda-protokolle
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{
		Limit:  100,
		Offset: 0,
	}

	// Parse query parameters
	if v := r.URL.Query().Get("elda_account_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			filter.ELDAAccountID = &id
		}
	}

	if v := r.URL.Query().Get("type"); v != "" {
		t := ProtokollType(v)
		filter.Type = &t
	}

	if v := r.URL.Query().Get("status"); v != "" {
		s := ProtokollStatus(v)
		filter.Status = &s
	}

	if v := r.URL.Query().Get("sv_nummer"); v != "" {
		filter.SVNummer = v
	}

	if v := r.URL.Query().Get("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.StartDate = &t
		}
	}

	if v := r.URL.Query().Get("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			filter.EndDate = &t
		}
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if limit, err := strconv.Atoi(v); err == nil && limit > 0 && limit <= 500 {
			filter.Limit = limit
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if offset, err := strconv.Atoi(v); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	protokolle, err := h.repo.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	count, _ := h.repo.Count(r.Context(), filter)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   protokolle,
		"total":  count,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// Get handles GET /api/v1/elda-protokolle/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige ID")
		return
	}

	protokoll, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err == ErrProtokollNotFound {
			respondError(w, http.StatusNotFound, "Protokoll nicht gefunden")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Include XML if requested
	includeXML := r.URL.Query().Get("include_xml") == "true"
	if !includeXML {
		protokoll.RequestXML = ""
		protokoll.ResponseXML = ""
	}

	respondJSON(w, http.StatusOK, protokoll)
}

// GetByProtokollnummer handles GET /api/v1/elda-protokolle/by-nummer/{nummer}
func (h *Handler) GetByProtokollnummer(w http.ResponseWriter, r *http.Request) {
	nummer := chi.URLParam(r, "nummer")
	if nummer == "" {
		respondError(w, http.StatusBadRequest, "Protokollnummer erforderlich")
		return
	}

	protokoll, err := h.repo.GetByProtokollnummer(r.Context(), nummer)
	if err != nil {
		if err == ErrProtokollNotFound {
			respondError(w, http.StatusNotFound, "Protokoll nicht gefunden")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Include XML if requested
	includeXML := r.URL.Query().Get("include_xml") == "true"
	if !includeXML {
		protokoll.RequestXML = ""
		protokoll.ResponseXML = ""
	}

	respondJSON(w, http.StatusOK, protokoll)
}

// GetHistoryBySVNummer handles GET /api/v1/elda-protokolle/history/{sv_nummer}
func (h *Handler) GetHistoryBySVNummer(w http.ResponseWriter, r *http.Request) {
	svNummer := chi.URLParam(r, "sv_nummer")
	if svNummer == "" {
		respondError(w, http.StatusBadRequest, "SV-Nummer erforderlich")
		return
	}

	accountIDStr := r.URL.Query().Get("elda_account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige elda_account_id")
		return
	}

	history, err := h.repo.GetHistoryBySVNummer(r.Context(), accountID, svNummer)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sv_nummer": svNummer,
		"history":   history,
		"count":     len(history),
	})
}

// GetStatistics handles GET /api/v1/elda-protokolle/statistics
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	accountIDStr := r.URL.Query().Get("elda_account_id")
	if accountIDStr == "" {
		respondError(w, http.StatusBadRequest, "elda_account_id erforderlich")
		return
	}

	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Ungültige elda_account_id")
		return
	}

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if d, err := strconv.Atoi(v); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	stats, err := h.repo.GetStatistics(r.Context(), accountID, days)
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

// RegisterRoutes registers protokoll routes with the router
func RegisterRoutes(r chi.Router, db *pgxpool.Pool) {
	handler := NewHandler(db)
	r.Mount("/elda-protokolle", handler.Routes())
}
