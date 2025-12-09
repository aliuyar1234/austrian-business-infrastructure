package branding

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/austrian-business-infrastructure/fo/internal/tenant"
)

// Handler handles branding-related HTTP requests
type Handler struct {
	service *Service
}

// NewHandler creates a new branding handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// StaffRoutes returns routes for staff managing branding
func (h *Handler) StaffRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.Get)
	r.Put("/", h.Update)
	r.Get("/css", h.GetCSS)
	r.Get("/preview", h.Preview)

	return r
}

// PublicRoutes returns public routes for portal
func (h *Handler) PublicRoutes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.GetPublic)
	r.Get("/css", h.GetPublicCSS)

	return r
}

// Get returns branding for the current tenant
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	branding, err := h.service.GetForTenant(ctx, tenantID)
	if err != nil {
		http.Error(w, "failed to get branding", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(branding)
}

// Update updates branding for the current tenant
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	branding, err := h.service.Update(ctx, tenantID, &req)
	if err != nil {
		http.Error(w, "failed to update branding", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(branding)
}

// GetCSS returns the generated CSS for the tenant
func (h *Handler) GetCSS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	branding, err := h.service.GetForTenant(ctx, tenantID)
	if err != nil {
		http.Error(w, "failed to get branding", http.StatusInternalServerError)
		return
	}

	css := h.service.GenerateCSS(branding)

	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(css))
}

// Preview returns branding preview with provided values
func (h *Handler) Preview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID := tenant.GetTenantID(ctx)
	if tenantID == uuid.Nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Get current branding as base
	branding, err := h.service.GetForTenant(ctx, tenantID)
	if err != nil {
		http.Error(w, "failed to get branding", http.StatusInternalServerError)
		return
	}

	// Apply query params for preview
	if primaryColor := r.URL.Query().Get("primary_color"); primaryColor != "" {
		branding.PrimaryColor = primaryColor
	}
	if secondaryColor := r.URL.Query().Get("secondary_color"); secondaryColor != "" {
		branding.SecondaryColor = &secondaryColor
	}
	if accentColor := r.URL.Query().Get("accent_color"); accentColor != "" {
		branding.AccentColor = &accentColor
	}

	css := h.service.GenerateCSS(branding)

	w.Header().Set("Content-Type", "text/css")
	w.Write([]byte(css))
}

// GetPublic returns public branding information
func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Try to resolve tenant from host
	host := r.Host
	tenantID, branding, err := h.service.ResolveTenant(ctx, host)
	if err != nil {
		http.Error(w, "failed to resolve tenant", http.StatusInternalServerError)
		return
	}

	// If not found by domain, try tenant ID from query or header
	if tenantID == uuid.Nil {
		tenantIDStr := r.URL.Query().Get("tenant_id")
		if tenantIDStr == "" {
			tenantIDStr = r.Header.Get("X-Tenant-ID")
		}

		if tenantIDStr != "" {
			tenantID, err = uuid.Parse(tenantIDStr)
			if err != nil {
				http.Error(w, "invalid tenant ID", http.StatusBadRequest)
				return
			}

			branding, err = h.service.GetForTenant(ctx, tenantID)
			if err != nil {
				http.Error(w, "failed to get branding", http.StatusInternalServerError)
				return
			}
		}
	}

	if branding == nil {
		http.Error(w, "branding not found", http.StatusNotFound)
		return
	}

	// Return only public fields
	publicBranding := map[string]interface{}{
		"company_name":    branding.CompanyName,
		"logo_url":        branding.LogoURL,
		"favicon_url":     branding.FaviconURL,
		"primary_color":   branding.PrimaryColor,
		"secondary_color": branding.SecondaryColor,
		"accent_color":    branding.AccentColor,
		"support_email":   branding.SupportEmail,
		"support_phone":   branding.SupportPhone,
		"welcome_message": branding.WelcomeMessage,
		"footer_text":     branding.FooterText,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(publicBranding)
}

// GetPublicCSS returns the generated CSS for public portal
func (h *Handler) GetPublicCSS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Try to resolve tenant from host
	host := r.Host
	tenantID, branding, err := h.service.ResolveTenant(ctx, host)
	if err != nil {
		http.Error(w, "failed to resolve tenant", http.StatusInternalServerError)
		return
	}

	// If not found by domain, try tenant ID from query
	if tenantID == uuid.Nil {
		tenantIDStr := r.URL.Query().Get("tenant_id")
		if tenantIDStr != "" {
			tenantID, err = uuid.Parse(tenantIDStr)
			if err != nil {
				http.Error(w, "invalid tenant ID", http.StatusBadRequest)
				return
			}

			branding, err = h.service.GetForTenant(ctx, tenantID)
			if err != nil {
				http.Error(w, "failed to get branding", http.StatusInternalServerError)
				return
			}
		}
	}

	if branding == nil {
		// Return default CSS
		branding = DefaultBranding
	}

	css := h.service.GenerateCSS(branding)

	w.Header().Set("Content-Type", "text/css")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(css))
}
