package foerderung

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FoerderungType represents the type of funding
type FoerderungType string

const (
	TypeZuschuss  FoerderungType = "zuschuss"
	TypeKredit    FoerderungType = "kredit"
	TypeGarantie  FoerderungType = "garantie"
	TypeBeratung  FoerderungType = "beratung"
	TypeKombination FoerderungType = "kombination"
)

// FoerderungStatus represents the status of a funding program
type FoerderungStatus string

const (
	StatusActive   FoerderungStatus = "active"
	StatusUpcoming FoerderungStatus = "upcoming"
	StatusPaused   FoerderungStatus = "paused"
	StatusClosed   FoerderungStatus = "closed"
)

// TargetSize represents the target company size
type TargetSize string

const (
	TargetSizeKMU             TargetSize = "kmu"
	TargetSizeStartup         TargetSize = "startup"
	TargetSizeGrossunternehmen TargetSize = "grossunternehmen"
	TargetSizeAlle            TargetSize = "alle"
)

// CompanySize represents granular company size (TypeScript-style)
// Based on EU KMU definition:
// - EPU: Ein-Personen-Unternehmen (1 employee)
// - kleinst: Kleinstunternehmen (< 10 employees, < €2M revenue)
// - klein: Kleinunternehmen (< 50 employees, < €10M revenue)
// - mittel: Mittlere Unternehmen (< 250 employees, < €50M revenue)
// - gross: Großunternehmen (>= 250 employees or >= €50M revenue)
type CompanySize string

const (
	SizeEPU     CompanySize = "epu"
	SizeKleinst CompanySize = "kleinst"
	SizeKlein   CompanySize = "klein"
	SizeMittel  CompanySize = "mittel"
	SizeGross   CompanySize = "gross"
)

// AllKMUSizes contains all sizes that qualify as KMU
var AllKMUSizes = []CompanySize{SizeEPU, SizeKleinst, SizeKlein, SizeMittel}

// DeadlineType represents how the deadline works
type DeadlineType string

const (
	DeadlineFixed          DeadlineType = "fixed"
	DeadlineRolling        DeadlineType = "rolling"
	DeadlineBudgetExhausted DeadlineType = "budget_exhausted"
)

// Provider constants for common funding providers
const (
	ProviderAWS = "AWS"
	ProviderFFG = "FFG"
	ProviderWKO = "WKO"
	ProviderAMS = "AMS"
	ProviderOeKB = "OeKB"
	ProviderEU  = "EU"
)

// Bundesland constants
var Bundeslaender = []string{
	"Wien", "Niederösterreich", "Oberösterreich", "Salzburg",
	"Tirol", "Vorarlberg", "Kärnten", "Steiermark", "Burgenland",
}

// Topic constants for common funding topics
var Topics = []string{
	"innovation", "digitalisierung", "export", "umwelt", "energie",
	"forschung", "entwicklung", "wachstum", "investition", "arbeitsplaetze",
	"gruendung", "nachfolge", "internationalisierung", "nachhaltigkeit",
}

// Foerderung represents a funding program
type Foerderung struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	ShortName   *string          `json:"short_name,omitempty"`
	Description *string          `json:"description,omitempty"`
	Provider    string           `json:"provider"`

	// Type
	Type           FoerderungType `json:"type"`
	FundingRateMin *float64       `json:"funding_rate_min,omitempty"`
	FundingRateMax *float64       `json:"funding_rate_max,omitempty"`
	MaxAmount      *int           `json:"max_amount,omitempty"`
	MinAmount      *int           `json:"min_amount,omitempty"`

	// Target Group
	TargetSize       *TargetSize   `json:"target_size,omitempty"`       // Legacy single size
	TargetSizes      []CompanySize `json:"target_sizes,omitempty"`      // TypeScript-style array of sizes
	TargetAge        *string       `json:"target_age,omitempty"`
	TargetAgeMin     *int          `json:"target_age_min,omitempty"`    // MinAlterJahre from TypeScript
	TargetAgeMax     *int          `json:"target_age_max,omitempty"`    // MaxAlterJahre from TypeScript
	TargetLegalForms []string      `json:"target_legal_forms,omitempty"`
	TargetIndustries []string      `json:"target_industries,omitempty"`
	ExcludedIndustries []string    `json:"excluded_industries,omitempty"` // BranchenAusschluss from TypeScript
	TargetStates     []string      `json:"target_states,omitempty"`

	// Topics & Categories
	Topics     []string `json:"topics"`
	Categories []string `json:"categories,omitempty"`

	// Requirements
	Requirements        *string         `json:"requirements,omitempty"`
	EligibilityCriteria json.RawMessage `json:"eligibility_criteria,omitempty"`

	// Deadlines
	ApplicationDeadline *time.Time   `json:"application_deadline,omitempty"`
	DeadlineType        *DeadlineType `json:"deadline_type,omitempty"`
	CallStart           *time.Time   `json:"call_start,omitempty"`
	CallEnd             *time.Time   `json:"call_end,omitempty"`

	// Links
	URL            *string `json:"url,omitempty"`
	ApplicationURL *string `json:"application_url,omitempty"`
	GuidelineURL   *string `json:"guideline_url,omitempty"`

	// Combinations
	CombinableWith    []uuid.UUID `json:"combinable_with,omitempty"`
	NotCombinableWith []uuid.UUID `json:"not_combinable_with,omitempty"`

	// Status
	Status        FoerderungStatus `json:"status"`
	IsHighlighted bool             `json:"is_highlighted"`

	// Metadata
	Source        *string    `json:"source,omitempty"`
	SourceID      *string    `json:"source_id,omitempty"`
	LastUpdatedAt *time.Time `json:"last_updated_at,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Computed fields (from view)
	DeadlineStatus    *string `json:"deadline_status,omitempty"`
	DaysUntilDeadline *int    `json:"days_until_deadline,omitempty"`
}

// Unternehmensprofil represents a company profile for funding search
type Unternehmensprofil struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	AccountID *uuid.UUID `json:"account_id,omitempty"`

	// Company Info
	Name        string  `json:"name"`
	LegalForm   *string `json:"legal_form,omitempty"`
	FoundedYear *int    `json:"founded_year,omitempty"`
	State       *string `json:"state,omitempty"`
	District    *string `json:"district,omitempty"`

	// Size
	EmployeesCount *int `json:"employees_count,omitempty"`
	AnnualRevenue  *int `json:"annual_revenue,omitempty"`
	BalanceTotal   *int `json:"balance_total,omitempty"`

	// Classification
	Industry   *string  `json:"industry,omitempty"`
	OnaceCodes []string `json:"onace_codes,omitempty"`
	IsStartup  bool     `json:"is_startup"`

	// Project
	ProjectDescription *string  `json:"project_description,omitempty"`
	InvestmentAmount   *int     `json:"investment_amount,omitempty"`
	ProjectTopics      []string `json:"project_topics,omitempty"`

	// Derived Info
	IsKMU              *bool   `json:"is_kmu,omitempty"`
	CompanyAgeCategory *string `json:"company_age_category,omitempty"`

	// Status
	Status             string     `json:"status"`
	DerivedFromAccount bool       `json:"derived_from_account"`
	LastSearchAt       *time.Time `json:"last_search_at,omitempty"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ProfileStatus constants
const (
	ProfileStatusDraft    = "draft"
	ProfileStatusComplete = "complete"
)

// FoerderungsSuche represents a search session
type FoerderungsSuche struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ProfileID uuid.UUID `json:"profile_id"`

	// Results
	TotalFoerderungen int             `json:"total_foerderungen"`
	TotalMatches      int             `json:"total_matches"`
	Matches           json.RawMessage `json:"matches"` // []FoerderungsMatch

	// Status
	Status   string  `json:"status"`
	Phase    *string `json:"phase,omitempty"`
	Progress int     `json:"progress"`

	// Timing
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`

	// Usage
	LLMTokensUsed int `json:"llm_tokens_used"`
	LLMCostCents  int `json:"llm_cost_cents"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Search status constants
const (
	SearchStatusPending      = "pending"
	SearchStatusRuleFiltering = "rule_filtering"
	SearchStatusLLMAnalysis  = "llm_analysis"
	SearchStatusCompleted    = "completed"
	SearchStatusFailed       = "failed"
)

// FoerderungsMatch represents a match result
type FoerderungsMatch struct {
	FoerderungID   uuid.UUID `json:"foerderung_id"`
	FoerderungName string    `json:"foerderung_name"`
	Provider       string    `json:"provider"`

	RuleScore  float64 `json:"rule_score"`
	LLMScore   float64 `json:"llm_score"`
	TotalScore float64 `json:"total_score"`

	LLMResult *LLMEligibilityResult `json:"llm_result,omitempty"`
}

// LLMEligibilityResult represents the LLM analysis result
type LLMEligibilityResult struct {
	Eligible        bool     `json:"eligible"`
	Confidence      string   `json:"confidence"` // high, medium, low
	MatchedCriteria []string `json:"matched_criteria"`
	ImplicitMatches []string `json:"implicit_matches,omitempty"`
	Concerns        []string `json:"concerns,omitempty"`
	EstimatedAmount *int     `json:"estimated_amount,omitempty"`
	CombinationHint *string  `json:"combination_hint,omitempty"`
	NextSteps       []string `json:"next_steps,omitempty"`
	InsiderTip      *string  `json:"insider_tip,omitempty"`
}

// ProfilMonitor represents monitoring settings for a profile
type ProfilMonitor struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	ProfileID uuid.UUID `json:"profile_id"`

	// Settings
	IsActive           bool   `json:"is_active"`
	MinScoreThreshold  int    `json:"min_score_threshold"`
	NotificationEmail  bool   `json:"notification_email"`
	NotificationPortal bool   `json:"notification_portal"`
	DigestMode         string `json:"digest_mode"` // immediate, daily, weekly

	// Tracking
	LastCheckAt        *time.Time `json:"last_check_at,omitempty"`
	LastNotificationAt *time.Time `json:"last_notification_at,omitempty"`
	MatchesFound       int        `json:"matches_found"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MonitorNotification represents a notification for a matched funding
type MonitorNotification struct {
	ID           uuid.UUID `json:"id"`
	MonitorID    uuid.UUID `json:"monitor_id"`
	FoerderungID uuid.UUID `json:"foerderung_id"`

	// Match Info
	Score        int     `json:"score"`
	MatchSummary *string `json:"match_summary,omitempty"`

	// Delivery
	EmailSent    bool       `json:"email_sent"`
	EmailSentAt  *time.Time `json:"email_sent_at,omitempty"`
	PortalNotified bool     `json:"portal_notified"`

	// User Action
	ViewedAt  *time.Time `json:"viewed_at,omitempty"`
	Dismissed bool       `json:"dismissed"`

	CreatedAt time.Time `json:"created_at"`
}

// FoerderungsAntrag represents a funding application
type FoerderungsAntrag struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	ProfileID    uuid.UUID `json:"profile_id"`
	FoerderungID uuid.UUID `json:"foerderung_id"`

	// Application
	Status            string     `json:"status"`
	InternalReference *string    `json:"internal_reference,omitempty"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`

	// Amounts
	RequestedAmount *int `json:"requested_amount,omitempty"`
	ApprovedAmount  *int `json:"approved_amount,omitempty"`

	// Decision
	DecisionDate  *time.Time `json:"decision_date,omitempty"`
	DecisionNotes *string    `json:"decision_notes,omitempty"`

	// Attachments and Timeline
	Attachments []Attachment     `json:"attachments,omitempty"`
	Timeline    []TimelineEntry  `json:"timeline,omitempty"`
	Notes       *string          `json:"notes,omitempty"`

	CreatedBy *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Attachment represents a document attached to an application
type Attachment struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	URL        string    `json:"url"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// TimelineEntry represents a status change in the application timeline
type TimelineEntry struct {
	Date        time.Time  `json:"date"`
	Status      string     `json:"status"`
	Description string     `json:"description"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
}

// Application status constants
const (
	AntragStatusPlanned   = "planned"
	AntragStatusDrafting  = "drafting"
	AntragStatusSubmitted = "submitted"
	AntragStatusInReview  = "in_review"
	AntragStatusApproved  = "approved"
	AntragStatusRejected  = "rejected"
	AntragStatusWithdrawn = "withdrawn"
)

// FoerderungsImport represents an import history record
type FoerderungsImport struct {
	ID       uuid.UUID `json:"id"`
	Source   string    `json:"source"`
	Filename *string   `json:"filename,omitempty"`

	// Results
	TotalRecords int             `json:"total_records"`
	Imported     int             `json:"imported"`
	Updated      int             `json:"updated"`
	Failed       int             `json:"failed"`
	Errors       json.RawMessage `json:"errors,omitempty"`

	// Status
	Status string `json:"status"`

	ImportedBy *uuid.UUID `json:"imported_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Import status constants
const (
	ImportStatusPending    = "pending"
	ImportStatusProcessing = "processing"
	ImportStatusCompleted  = "completed"
	ImportStatusFailed     = "failed"
)

// GetMatchesSlice parses the matches JSONB to []FoerderungsMatch
func (s *FoerderungsSuche) GetMatchesSlice() ([]FoerderungsMatch, error) {
	if s.Matches == nil {
		return []FoerderungsMatch{}, nil
	}
	var matches []FoerderungsMatch
	if err := json.Unmarshal(s.Matches, &matches); err != nil {
		return nil, err
	}
	return matches, nil
}

// SetMatchesSlice sets the matches JSONB from []FoerderungsMatch
func (s *FoerderungsSuche) SetMatchesSlice(matches []FoerderungsMatch) error {
	data, err := json.Marshal(matches)
	if err != nil {
		return err
	}
	s.Matches = data
	return nil
}
