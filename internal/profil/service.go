package profil

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// Service provides profile business logic
type Service struct {
	repo *Repository
}

// NewService creates a new profile service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateInput contains input for creating a profile
type CreateInput struct {
	TenantID           uuid.UUID
	AccountID          *uuid.UUID
	Name               string
	LegalForm          *string
	FoundedYear        *int
	State              *string
	District           *string
	EmployeesCount     *int
	AnnualRevenue      *int
	BalanceTotal       *int
	Industry           *string
	OnaceCodes         []string
	IsStartup          bool
	ProjectDescription *string
	InvestmentAmount   *int
	ProjectTopics      []string
	CreatedBy          *uuid.UUID
}

// UpdateInput contains input for updating a profile
type UpdateInput struct {
	Name               *string
	LegalForm          *string
	FoundedYear        *int
	State              *string
	District           *string
	EmployeesCount     *int
	AnnualRevenue      *int
	BalanceTotal       *int
	Industry           *string
	OnaceCodes         []string
	IsStartup          *bool
	ProjectDescription *string
	InvestmentAmount   *int
	ProjectTopics      []string
}

// Create creates a new profile with validation
func (s *Service) Create(ctx context.Context, input *CreateInput) (*foerderung.Unternehmensprofil, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("profile name is required")
	}

	profile := &foerderung.Unternehmensprofil{
		TenantID:           input.TenantID,
		AccountID:          input.AccountID,
		Name:               input.Name,
		LegalForm:          input.LegalForm,
		FoundedYear:        input.FoundedYear,
		State:              input.State,
		District:           input.District,
		EmployeesCount:     input.EmployeesCount,
		AnnualRevenue:      input.AnnualRevenue,
		BalanceTotal:       input.BalanceTotal,
		Industry:           input.Industry,
		OnaceCodes:         input.OnaceCodes,
		IsStartup:          input.IsStartup,
		ProjectDescription: input.ProjectDescription,
		InvestmentAmount:   input.InvestmentAmount,
		ProjectTopics:      input.ProjectTopics,
		CreatedBy:          input.CreatedBy,
		Status:             foerderung.ProfileStatusDraft,
	}

	// Calculate derived fields
	s.calculateDerivedFields(profile)

	if err := s.repo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// GetByID retrieves a profile by ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*foerderung.Unternehmensprofil, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByIDAndTenant retrieves a profile ensuring tenant access
func (s *Service) GetByIDAndTenant(ctx context.Context, id, tenantID uuid.UUID) (*foerderung.Unternehmensprofil, error) {
	return s.repo.GetByIDAndTenant(ctx, id, tenantID)
}

// ListByTenant lists all profiles for a tenant
func (s *Service) ListByTenant(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*foerderung.Unternehmensprofil, int, error) {
	return s.repo.ListByTenant(ctx, tenantID, limit, offset)
}

// Update updates a profile
func (s *Service) Update(ctx context.Context, id, tenantID uuid.UUID, input *UpdateInput) (*foerderung.Unternehmensprofil, error) {
	profile, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	if input.Name != nil {
		profile.Name = *input.Name
	}
	if input.LegalForm != nil {
		profile.LegalForm = input.LegalForm
	}
	if input.FoundedYear != nil {
		profile.FoundedYear = input.FoundedYear
	}
	if input.State != nil {
		profile.State = input.State
	}
	if input.District != nil {
		profile.District = input.District
	}
	if input.EmployeesCount != nil {
		profile.EmployeesCount = input.EmployeesCount
	}
	if input.AnnualRevenue != nil {
		profile.AnnualRevenue = input.AnnualRevenue
	}
	if input.BalanceTotal != nil {
		profile.BalanceTotal = input.BalanceTotal
	}
	if input.Industry != nil {
		profile.Industry = input.Industry
	}
	if input.OnaceCodes != nil {
		profile.OnaceCodes = input.OnaceCodes
	}
	if input.IsStartup != nil {
		profile.IsStartup = *input.IsStartup
	}
	if input.ProjectDescription != nil {
		profile.ProjectDescription = input.ProjectDescription
	}
	if input.InvestmentAmount != nil {
		profile.InvestmentAmount = input.InvestmentAmount
	}
	if input.ProjectTopics != nil {
		profile.ProjectTopics = input.ProjectTopics
	}

	// Recalculate derived fields
	s.calculateDerivedFields(profile)

	// Check if profile is complete
	if s.isProfileComplete(profile) {
		profile.Status = foerderung.ProfileStatusComplete
	}

	if err := s.repo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return profile, nil
}

// Delete deletes a profile
func (s *Service) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	// Verify ownership
	_, err := s.repo.GetByIDAndTenant(ctx, id, tenantID)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// calculateDerivedFields calculates is_kmu and company_age_category
func (s *Service) calculateDerivedFields(p *foerderung.Unternehmensprofil) {
	// Calculate is_kmu based on EU definition
	isKMU := s.calculateIsKMU(p.EmployeesCount, p.AnnualRevenue, p.BalanceTotal)
	p.IsKMU = &isKMU

	// Calculate company age category
	if p.FoundedYear != nil {
		currentYear := time.Now().Year()
		age := currentYear - *p.FoundedYear
		var category string
		if age <= 5 {
			category = "gruendung"
		} else {
			category = "etabliert"
		}
		p.CompanyAgeCategory = &category
	}
}

// calculateIsKMU calculates if company qualifies as KMU (EU definition)
// KMU: <250 employees AND (<€50M revenue OR <€43M balance)
func (s *Service) calculateIsKMU(employees, revenue, balance *int) bool {
	// If no data, assume KMU (benefit of doubt)
	if employees == nil && revenue == nil && balance == nil {
		return true
	}

	// Check employee count
	if employees != nil && *employees >= 250 {
		return false
	}

	// Check financial criteria (OR condition)
	revenueOK := revenue == nil || *revenue < 50000000
	balanceOK := balance == nil || *balance < 43000000

	// If both financials are known and exceed limits, not KMU
	if revenue != nil && balance != nil {
		if *revenue >= 50000000 && *balance >= 43000000 {
			return false
		}
	}

	return revenueOK || balanceOK
}

// isProfileComplete checks if profile has required fields for search
func (s *Service) isProfileComplete(p *foerderung.Unternehmensprofil) bool {
	if p.Name == "" {
		return false
	}
	if p.State == nil || *p.State == "" {
		return false
	}
	// At least one of: employees, revenue, or explicitly set startup flag
	hasSize := p.EmployeesCount != nil || p.AnnualRevenue != nil || p.IsStartup
	if !hasSize {
		return false
	}
	return true
}

// ValidateForSearch validates if profile is ready for search
func (s *Service) ValidateForSearch(p *foerderung.Unternehmensprofil) error {
	if p.Name == "" {
		return fmt.Errorf("Unternehmensname ist erforderlich")
	}
	if p.State == nil || *p.State == "" {
		return fmt.Errorf("Bundesland ist erforderlich")
	}
	return nil
}

// MarkSearched updates the last search timestamp
func (s *Service) MarkSearched(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateLastSearchAt(ctx, id)
}
