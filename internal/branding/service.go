package branding

import (
	"context"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultBranding provides default values when tenant has no branding configured
var DefaultBranding = &TenantBranding{
	CompanyName:  "Client Portal",
	PrimaryColor: "#3B82F6", // Blue
}

// UpdateRequest contains data for updating branding
type UpdateRequest struct {
	CompanyName    *string `json:"company_name,omitempty"`
	LogoURL        *string `json:"logo_url,omitempty"`
	FaviconURL     *string `json:"favicon_url,omitempty"`
	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
	AccentColor    *string `json:"accent_color,omitempty"`
	CustomCSS      *string `json:"custom_css,omitempty"`
	SupportEmail   *string `json:"support_email,omitempty"`
	SupportPhone   *string `json:"support_phone,omitempty"`
	WelcomeMessage *string `json:"welcome_message,omitempty"`
	FooterText     *string `json:"footer_text,omitempty"`
	CustomDomain   *string `json:"custom_domain,omitempty"`
}

// Service provides branding business logic
type Service struct {
	repo  *Repository
	pool  *pgxpool.Pool
	cache map[uuid.UUID]*TenantBranding
	mu    sync.RWMutex
}

// NewService creates a new branding service
func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		repo:  NewRepository(pool),
		pool:  pool,
		cache: make(map[uuid.UUID]*TenantBranding),
	}
}

// Repository returns the underlying repository
func (s *Service) Repository() *Repository {
	return s.repo
}

// GetForTenant retrieves branding for a tenant (with caching)
func (s *Service) GetForTenant(ctx context.Context, tenantID uuid.UUID) (*TenantBranding, error) {
	// Check cache first
	s.mu.RLock()
	if branding, ok := s.cache[tenantID]; ok {
		s.mu.RUnlock()
		return branding, nil
	}
	s.mu.RUnlock()

	// Fetch from database
	branding, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		if err == ErrBrandingNotFound {
			// Return default branding
			defaultCopy := *DefaultBranding
			defaultCopy.TenantID = tenantID
			return &defaultCopy, nil
		}
		return nil, err
	}

	// Cache the result
	s.mu.Lock()
	s.cache[tenantID] = branding
	s.mu.Unlock()

	return branding, nil
}

// GetByDomain retrieves branding by custom domain
func (s *Service) GetByDomain(ctx context.Context, domain string) (*TenantBranding, error) {
	// Normalize domain
	domain = strings.ToLower(strings.TrimSpace(domain))

	return s.repo.GetByCustomDomain(ctx, domain)
}

// Update updates branding configuration
func (s *Service) Update(ctx context.Context, tenantID uuid.UUID, req *UpdateRequest) (*TenantBranding, error) {
	// Get existing branding or create new
	branding, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil && err != ErrBrandingNotFound {
		return nil, err
	}

	if branding == nil {
		branding = &TenantBranding{
			TenantID:     tenantID,
			CompanyName:  DefaultBranding.CompanyName,
			PrimaryColor: DefaultBranding.PrimaryColor,
		}
	}

	// Apply updates
	if req.CompanyName != nil {
		branding.CompanyName = *req.CompanyName
	}
	if req.LogoURL != nil {
		branding.LogoURL = req.LogoURL
	}
	if req.FaviconURL != nil {
		branding.FaviconURL = req.FaviconURL
	}
	if req.PrimaryColor != nil {
		branding.PrimaryColor = *req.PrimaryColor
	}
	if req.SecondaryColor != nil {
		branding.SecondaryColor = req.SecondaryColor
	}
	if req.AccentColor != nil {
		branding.AccentColor = req.AccentColor
	}
	if req.CustomCSS != nil {
		branding.CustomCSS = req.CustomCSS
	}
	if req.SupportEmail != nil {
		branding.SupportEmail = req.SupportEmail
	}
	if req.SupportPhone != nil {
		branding.SupportPhone = req.SupportPhone
	}
	if req.WelcomeMessage != nil {
		branding.WelcomeMessage = req.WelcomeMessage
	}
	if req.FooterText != nil {
		branding.FooterText = req.FooterText
	}
	if req.CustomDomain != nil {
		domain := strings.ToLower(strings.TrimSpace(*req.CustomDomain))
		if domain == "" {
			branding.CustomDomain = nil
		} else {
			branding.CustomDomain = &domain
		}
	}

	// Upsert to database
	if err := s.repo.Upsert(ctx, branding); err != nil {
		return nil, err
	}

	// Invalidate cache
	s.mu.Lock()
	delete(s.cache, tenantID)
	s.mu.Unlock()

	return branding, nil
}

// InvalidateCache removes a tenant from the cache
func (s *Service) InvalidateCache(tenantID uuid.UUID) {
	s.mu.Lock()
	delete(s.cache, tenantID)
	s.mu.Unlock()
}

// ResolveTenant determines the tenant from a request domain
func (s *Service) ResolveTenant(ctx context.Context, host string) (uuid.UUID, *TenantBranding, error) {
	// Strip port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Try to find by custom domain
	branding, err := s.GetByDomain(ctx, host)
	if err != nil {
		if err == ErrBrandingNotFound {
			return uuid.Nil, nil, nil
		}
		return uuid.Nil, nil, err
	}

	return branding.TenantID, branding, nil
}

// GenerateCSS generates CSS variables from branding
func (s *Service) GenerateCSS(branding *TenantBranding) string {
	var css strings.Builder

	css.WriteString(":root {\n")
	css.WriteString("  --primary-color: " + branding.PrimaryColor + ";\n")

	if branding.SecondaryColor != nil {
		css.WriteString("  --secondary-color: " + *branding.SecondaryColor + ";\n")
	}
	if branding.AccentColor != nil {
		css.WriteString("  --accent-color: " + *branding.AccentColor + ";\n")
	}

	css.WriteString("}\n")

	if branding.CustomCSS != nil && *branding.CustomCSS != "" {
		css.WriteString("\n")
		css.WriteString(*branding.CustomCSS)
	}

	return css.String()
}
