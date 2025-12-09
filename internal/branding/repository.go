package branding

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrBrandingNotFound = errors.New("branding not found")
)

// TenantBranding represents branding configuration for a tenant
type TenantBranding struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`

	// Branding
	CompanyName     string     `json:"company_name"`
	LogoURL         *string    `json:"logo_url,omitempty"`
	FaviconURL      *string    `json:"favicon_url,omitempty"`

	// Colors
	PrimaryColor    string     `json:"primary_color"`
	SecondaryColor  *string    `json:"secondary_color,omitempty"`
	AccentColor     *string    `json:"accent_color,omitempty"`

	// Custom CSS
	CustomCSS       *string    `json:"custom_css,omitempty"`

	// Contact
	SupportEmail    *string    `json:"support_email,omitempty"`
	SupportPhone    *string    `json:"support_phone,omitempty"`

	// Portal settings
	WelcomeMessage  *string    `json:"welcome_message,omitempty"`
	FooterText      *string    `json:"footer_text,omitempty"`

	// Custom domain
	CustomDomain    *string    `json:"custom_domain,omitempty"`

	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// Repository provides branding data access
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new branding repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new branding configuration
func (r *Repository) Create(ctx context.Context, branding *TenantBranding) error {
	if branding.ID == uuid.Nil {
		branding.ID = uuid.New()
	}

	query := `
		INSERT INTO tenant_branding (
			id, tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, custom_css,
			support_email, support_phone, welcome_message, footer_text, custom_domain
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		branding.ID,
		branding.TenantID,
		branding.CompanyName,
		branding.LogoURL,
		branding.FaviconURL,
		branding.PrimaryColor,
		branding.SecondaryColor,
		branding.AccentColor,
		branding.CustomCSS,
		branding.SupportEmail,
		branding.SupportPhone,
		branding.WelcomeMessage,
		branding.FooterText,
		branding.CustomDomain,
	).Scan(&branding.CreatedAt, &branding.UpdatedAt)

	return err
}

// GetByID retrieves branding by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*TenantBranding, error) {
	query := `
		SELECT id, tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, custom_css,
			support_email, support_phone, welcome_message, footer_text, custom_domain,
			created_at, updated_at
		FROM tenant_branding
		WHERE id = $1
	`

	branding := &TenantBranding{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&branding.ID, &branding.TenantID, &branding.CompanyName,
		&branding.LogoURL, &branding.FaviconURL,
		&branding.PrimaryColor, &branding.SecondaryColor, &branding.AccentColor,
		&branding.CustomCSS, &branding.SupportEmail, &branding.SupportPhone,
		&branding.WelcomeMessage, &branding.FooterText, &branding.CustomDomain,
		&branding.CreatedAt, &branding.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBrandingNotFound
		}
		return nil, err
	}

	return branding, nil
}

// GetByTenantID retrieves branding by tenant ID
func (r *Repository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*TenantBranding, error) {
	query := `
		SELECT id, tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, custom_css,
			support_email, support_phone, welcome_message, footer_text, custom_domain,
			created_at, updated_at
		FROM tenant_branding
		WHERE tenant_id = $1
	`

	branding := &TenantBranding{}
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(
		&branding.ID, &branding.TenantID, &branding.CompanyName,
		&branding.LogoURL, &branding.FaviconURL,
		&branding.PrimaryColor, &branding.SecondaryColor, &branding.AccentColor,
		&branding.CustomCSS, &branding.SupportEmail, &branding.SupportPhone,
		&branding.WelcomeMessage, &branding.FooterText, &branding.CustomDomain,
		&branding.CreatedAt, &branding.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBrandingNotFound
		}
		return nil, err
	}

	return branding, nil
}

// GetByCustomDomain retrieves branding by custom domain
func (r *Repository) GetByCustomDomain(ctx context.Context, domain string) (*TenantBranding, error) {
	query := `
		SELECT id, tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, custom_css,
			support_email, support_phone, welcome_message, footer_text, custom_domain,
			created_at, updated_at
		FROM tenant_branding
		WHERE custom_domain = $1
	`

	branding := &TenantBranding{}
	err := r.pool.QueryRow(ctx, query, domain).Scan(
		&branding.ID, &branding.TenantID, &branding.CompanyName,
		&branding.LogoURL, &branding.FaviconURL,
		&branding.PrimaryColor, &branding.SecondaryColor, &branding.AccentColor,
		&branding.CustomCSS, &branding.SupportEmail, &branding.SupportPhone,
		&branding.WelcomeMessage, &branding.FooterText, &branding.CustomDomain,
		&branding.CreatedAt, &branding.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBrandingNotFound
		}
		return nil, err
	}

	return branding, nil
}

// Update updates branding configuration
func (r *Repository) Update(ctx context.Context, branding *TenantBranding) error {
	query := `
		UPDATE tenant_branding
		SET company_name = $2, logo_url = $3, favicon_url = $4,
			primary_color = $5, secondary_color = $6, accent_color = $7,
			custom_css = $8, support_email = $9, support_phone = $10,
			welcome_message = $11, footer_text = $12, custom_domain = $13,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		branding.ID,
		branding.CompanyName,
		branding.LogoURL,
		branding.FaviconURL,
		branding.PrimaryColor,
		branding.SecondaryColor,
		branding.AccentColor,
		branding.CustomCSS,
		branding.SupportEmail,
		branding.SupportPhone,
		branding.WelcomeMessage,
		branding.FooterText,
		branding.CustomDomain,
	).Scan(&branding.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBrandingNotFound
		}
		return err
	}

	return nil
}

// Upsert creates or updates branding for a tenant
func (r *Repository) Upsert(ctx context.Context, branding *TenantBranding) error {
	if branding.ID == uuid.Nil {
		branding.ID = uuid.New()
	}

	query := `
		INSERT INTO tenant_branding (
			id, tenant_id, company_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, custom_css,
			support_email, support_phone, welcome_message, footer_text, custom_domain
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id) DO UPDATE SET
			company_name = EXCLUDED.company_name,
			logo_url = EXCLUDED.logo_url,
			favicon_url = EXCLUDED.favicon_url,
			primary_color = EXCLUDED.primary_color,
			secondary_color = EXCLUDED.secondary_color,
			accent_color = EXCLUDED.accent_color,
			custom_css = EXCLUDED.custom_css,
			support_email = EXCLUDED.support_email,
			support_phone = EXCLUDED.support_phone,
			welcome_message = EXCLUDED.welcome_message,
			footer_text = EXCLUDED.footer_text,
			custom_domain = EXCLUDED.custom_domain,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		branding.ID,
		branding.TenantID,
		branding.CompanyName,
		branding.LogoURL,
		branding.FaviconURL,
		branding.PrimaryColor,
		branding.SecondaryColor,
		branding.AccentColor,
		branding.CustomCSS,
		branding.SupportEmail,
		branding.SupportPhone,
		branding.WelcomeMessage,
		branding.FooterText,
		branding.CustomDomain,
	).Scan(&branding.ID, &branding.CreatedAt, &branding.UpdatedAt)

	return err
}

// Delete deletes branding configuration
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM tenant_branding WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrBrandingNotFound
	}

	return nil
}
