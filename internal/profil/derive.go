package profil

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"austrian-business-infrastructure/internal/foerderung"
)

// AccountData represents data from linked account systems
type AccountData struct {
	// From Firmenbuch (009)
	CompanyName      string
	LegalForm        string
	RegisterNumber   string
	RegistrationDate *time.Time
	Address          *AddressData

	// From UVA/Finanz (009)
	AnnualRevenue *int
	BalanceTotal  *int

	// From ELDA (013)
	EmployeesCount *int
}

// AddressData represents address information
type AddressData struct {
	Street     string
	PostalCode string
	City       string
	State      string // Bundesland
	Country    string
}

// AccountDataProvider interface for fetching account data
type AccountDataProvider interface {
	GetAccountData(ctx context.Context, accountID uuid.UUID) (*AccountData, error)
}

// DeriveService handles profile derivation from account data
type DeriveService struct {
	repo         *Repository
	dataProvider AccountDataProvider
}

// NewDeriveService creates a new derive service
func NewDeriveService(repo *Repository, dataProvider AccountDataProvider) *DeriveService {
	return &DeriveService{
		repo:         repo,
		dataProvider: dataProvider,
	}
}

// DeriveFromAccount creates or updates a profile from account data
func (s *DeriveService) DeriveFromAccount(ctx context.Context, tenantID, accountID uuid.UUID, userID *uuid.UUID) (*foerderung.Unternehmensprofil, error) {
	// Fetch account data from connected systems
	accountData, err := s.dataProvider.GetAccountData(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account data: %w", err)
	}

	// Check if profile already exists for this account
	existing, _ := s.repo.ListByAccount(ctx, accountID)
	var profile *foerderung.Unternehmensprofil

	if len(existing) > 0 {
		// Update existing profile
		profile = existing[0]
		s.applyAccountData(profile, accountData)
		profile.DerivedFromAccount = true

		if err := s.repo.Update(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to update profile: %w", err)
		}
	} else {
		// Create new profile
		profile = &foerderung.Unternehmensprofil{
			TenantID:           tenantID,
			AccountID:          &accountID,
			DerivedFromAccount: true,
			Status:             foerderung.ProfileStatusDraft,
			CreatedBy:          userID,
		}
		s.applyAccountData(profile, accountData)

		if err := s.repo.Create(ctx, profile); err != nil {
			return nil, fmt.Errorf("failed to create profile: %w", err)
		}
	}

	return profile, nil
}

// applyAccountData applies account data to a profile
func (s *DeriveService) applyAccountData(profile *foerderung.Unternehmensprofil, data *AccountData) {
	// Company name from Firmenbuch
	if data.CompanyName != "" {
		profile.Name = data.CompanyName
	}

	// Legal form from Firmenbuch
	if data.LegalForm != "" {
		profile.LegalForm = &data.LegalForm
	}

	// Founded year from registration date
	if data.RegistrationDate != nil {
		year := data.RegistrationDate.Year()
		profile.FoundedYear = &year
	}

	// State from address
	if data.Address != nil && data.Address.State != "" {
		profile.State = &data.Address.State
	}

	// District from address
	if data.Address != nil && data.Address.City != "" {
		profile.District = &data.Address.City
	}

	// Revenue from UVA
	if data.AnnualRevenue != nil {
		profile.AnnualRevenue = data.AnnualRevenue
	}

	// Balance from UVA
	if data.BalanceTotal != nil {
		profile.BalanceTotal = data.BalanceTotal
	}

	// Employees from ELDA
	if data.EmployeesCount != nil {
		profile.EmployeesCount = data.EmployeesCount
	}

	// Calculate derived fields
	s.calculateDerivedFields(profile)
}

// calculateDerivedFields calculates is_kmu and company_age_category
func (s *DeriveService) calculateDerivedFields(p *foerderung.Unternehmensprofil) {
	// Calculate is_kmu
	isKMU := calculateIsKMU(p.EmployeesCount, p.AnnualRevenue, p.BalanceTotal)
	p.IsKMU = &isKMU

	// Calculate company age category
	if p.FoundedYear != nil {
		currentYear := time.Now().Year()
		age := currentYear - *p.FoundedYear
		var category string
		if age <= 5 {
			category = "gruendung"
			p.IsStartup = true
		} else {
			category = "etabliert"
		}
		p.CompanyAgeCategory = &category
	}

	// Update status if complete
	if isProfileComplete(p) {
		p.Status = foerderung.ProfileStatusComplete
	}
}

// calculateIsKMU calculates if company qualifies as KMU
func calculateIsKMU(employees, revenue, balance *int) bool {
	if employees == nil && revenue == nil && balance == nil {
		return true
	}

	if employees != nil && *employees >= 250 {
		return false
	}

	if revenue != nil && balance != nil {
		if *revenue >= 50000000 && *balance >= 43000000 {
			return false
		}
	}

	return true
}

// isProfileComplete checks if profile has minimum required fields
func isProfileComplete(p *foerderung.Unternehmensprofil) bool {
	if p.Name == "" {
		return false
	}
	if p.State == nil || *p.State == "" {
		return false
	}
	hasSize := p.EmployeesCount != nil || p.AnnualRevenue != nil || p.IsStartup
	return hasSize
}

// MockAccountDataProvider provides mock data for testing
type MockAccountDataProvider struct{}

// GetAccountData returns mock account data
func (m *MockAccountDataProvider) GetAccountData(ctx context.Context, accountID uuid.UUID) (*AccountData, error) {
	// Return mock data for testing
	revenue := 5000000
	employees := 25
	registrationDate := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)

	return &AccountData{
		CompanyName:      "Test GmbH",
		LegalForm:        "GmbH",
		RegisterNumber:   "FN 123456a",
		RegistrationDate: &registrationDate,
		AnnualRevenue:    &revenue,
		EmployeesCount:   &employees,
		Address: &AddressData{
			Street:     "Teststraße 1",
			PostalCode: "1010",
			City:       "Wien",
			State:      "Wien",
			Country:    "AT",
		},
	}, nil
}
