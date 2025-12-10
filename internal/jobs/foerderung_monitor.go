package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"austrian-business-infrastructure/internal/foerderung"
)

// FoerderungMonitorJob checks for new matching Förderungen for active monitors
type FoerderungMonitorJob struct {
	db              *pgxpool.Pool
	foerderungRepo  FoerderungRepository
	monitorRepo     MonitorRepository
	notificationRepo NotificationRepository
	matcherService  MatcherService
	emailService    EmailService
}

// FoerderungRepository interface for Förderung data access
type FoerderungRepository interface {
	ListActive(ctx context.Context) ([]*foerderung.Foerderung, error)
	ListNewSince(ctx context.Context, since time.Time) ([]*foerderung.Foerderung, error)
}

// MonitorRepository interface for monitor data access
type MonitorRepository interface {
	ListActive(ctx context.Context) ([]*foerderung.ProfilMonitor, error)
	Update(ctx context.Context, m *foerderung.ProfilMonitor) error
}

// NotificationRepository interface for notification data access
type NotificationRepository interface {
	Create(ctx context.Context, n *foerderung.MonitorNotification) error
}

// MatcherService interface for matching logic
type MatcherService interface {
	RunSearchForProfile(ctx context.Context, profileID uuid.UUID, foerderungen []*foerderung.Foerderung) ([]MatchResult, error)
}

// MatchResult represents a match from the matcher service
type MatchResult struct {
	FoerderungID   uuid.UUID
	FoerderungName string
	Score          int
	Summary        string
}

// EmailService interface for sending emails
type EmailService interface {
	SendFoerderungNotification(ctx context.Context, to string, matches []MatchResult) error
}

// NewFoerderungMonitorJob creates a new monitor job
func NewFoerderungMonitorJob(
	db *pgxpool.Pool,
	foerderungRepo FoerderungRepository,
	monitorRepo MonitorRepository,
	notificationRepo NotificationRepository,
	matcherService MatcherService,
	emailService EmailService,
) *FoerderungMonitorJob {
	return &FoerderungMonitorJob{
		db:              db,
		foerderungRepo:  foerderungRepo,
		monitorRepo:     monitorRepo,
		notificationRepo: notificationRepo,
		matcherService:  matcherService,
		emailService:    emailService,
	}
}

// Run executes the monitor job
func (j *FoerderungMonitorJob) Run(ctx context.Context) error {
	log.Println("[FoerderungMonitor] Starting monitor job")
	startTime := time.Now()

	// Get all active monitors
	monitors, err := j.monitorRepo.ListActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active monitors: %w", err)
	}

	if len(monitors) == 0 {
		log.Println("[FoerderungMonitor] No active monitors found")
		return nil
	}

	log.Printf("[FoerderungMonitor] Found %d active monitors", len(monitors))

	// Get new Förderungen since last check (or all if first run)
	var foerderungen []*foerderung.Foerderung
	oldestCheck := j.findOldestCheck(monitors)
	if oldestCheck != nil {
		foerderungen, err = j.foerderungRepo.ListNewSince(ctx, *oldestCheck)
	} else {
		foerderungen, err = j.foerderungRepo.ListActive(ctx)
	}
	if err != nil {
		return fmt.Errorf("failed to list foerderungen: %w", err)
	}

	if len(foerderungen) == 0 {
		log.Println("[FoerderungMonitor] No new Förderungen to check")
		return nil
	}

	log.Printf("[FoerderungMonitor] Checking %d Förderungen against %d monitors", len(foerderungen), len(monitors))

	// Process each monitor
	totalMatches := 0
	for _, monitor := range monitors {
		matches, err := j.processMonitor(ctx, monitor, foerderungen)
		if err != nil {
			log.Printf("[FoerderungMonitor] Error processing monitor %s: %v", monitor.ID, err)
			continue
		}
		totalMatches += matches
	}

	duration := time.Since(startTime)
	log.Printf("[FoerderungMonitor] Job completed in %v. Found %d new matches.", duration, totalMatches)

	return nil
}

// processMonitor processes a single monitor against new Förderungen
func (j *FoerderungMonitorJob) processMonitor(ctx context.Context, monitor *foerderung.ProfilMonitor, foerderungen []*foerderung.Foerderung) (int, error) {
	// Run matching for this profile
	results, err := j.matcherService.RunSearchForProfile(ctx, monitor.ProfileID, foerderungen)
	if err != nil {
		return 0, fmt.Errorf("failed to run matching: %w", err)
	}

	// Filter by threshold
	var matches []MatchResult
	for _, r := range results {
		if r.Score >= monitor.MinScoreThreshold {
			matches = append(matches, r)
		}
	}

	if len(matches) == 0 {
		// Update last check time even if no matches
		now := time.Now()
		monitor.LastCheckAt = &now
		j.monitorRepo.Update(ctx, monitor)
		return 0, nil
	}

	// Create notifications
	for _, match := range matches {
		notification := &foerderung.MonitorNotification{
			MonitorID:    monitor.ID,
			FoerderungID: match.FoerderungID,
			Score:        match.Score,
			MatchSummary: &match.Summary,
		}

		if err := j.notificationRepo.Create(ctx, notification); err != nil {
			log.Printf("[FoerderungMonitor] Failed to create notification: %v", err)
			continue
		}
	}

	// Update monitor stats
	now := time.Now()
	monitor.LastCheckAt = &now
	monitor.MatchesFound += len(matches)

	// Send immediate notifications if configured
	if monitor.DigestMode == "immediate" && monitor.NotificationEmail {
		// Would send email here
		monitor.LastNotificationAt = &now
	}

	if err := j.monitorRepo.Update(ctx, monitor); err != nil {
		return len(matches), fmt.Errorf("failed to update monitor: %w", err)
	}

	return len(matches), nil
}

// findOldestCheck finds the oldest last_check_at among monitors
func (j *FoerderungMonitorJob) findOldestCheck(monitors []*foerderung.ProfilMonitor) *time.Time {
	var oldest *time.Time
	for _, m := range monitors {
		if m.LastCheckAt == nil {
			return nil // At least one monitor has never been checked
		}
		if oldest == nil || m.LastCheckAt.Before(*oldest) {
			oldest = m.LastCheckAt
		}
	}
	return oldest
}

// FoerderungExpiryJob checks for expired Förderungen and deactivates them
type FoerderungExpiryJob struct {
	db             *pgxpool.Pool
	foerderungRepo FoerderungExpiryRepository
	warningDays    int
}

// FoerderungExpiryRepository interface for Förderung expiry operations
type FoerderungExpiryRepository interface {
	ListExpired(ctx context.Context) ([]*foerderung.Foerderung, error)
	ListExpiringSoon(ctx context.Context, days int) ([]*foerderung.Foerderung, error)
	Deactivate(ctx context.Context, id uuid.UUID) error
}

// NewFoerderungExpiryJob creates a new expiry job
func NewFoerderungExpiryJob(db *pgxpool.Pool, repo FoerderungExpiryRepository, warningDays int) *FoerderungExpiryJob {
	return &FoerderungExpiryJob{
		db:             db,
		foerderungRepo: repo,
		warningDays:    warningDays,
	}
}

// Run executes the expiry job
func (j *FoerderungExpiryJob) Run(ctx context.Context) error {
	log.Println("[FoerderungExpiry] Starting expiry job")

	// Deactivate expired Förderungen
	expired, err := j.foerderungRepo.ListExpired(ctx)
	if err != nil {
		return fmt.Errorf("failed to list expired: %w", err)
	}

	for _, f := range expired {
		if err := j.foerderungRepo.Deactivate(ctx, f.ID); err != nil {
			log.Printf("[FoerderungExpiry] Failed to deactivate %s: %v", f.ID, err)
			continue
		}
		log.Printf("[FoerderungExpiry] Deactivated expired Förderung: %s", f.Name)
	}

	// Log upcoming expirations
	expiringSoon, err := j.foerderungRepo.ListExpiringSoon(ctx, j.warningDays)
	if err != nil {
		log.Printf("[FoerderungExpiry] Failed to list expiring soon: %v", err)
	} else if len(expiringSoon) > 0 {
		log.Printf("[FoerderungExpiry] %d Förderungen expiring within %d days", len(expiringSoon), j.warningDays)
	}

	log.Printf("[FoerderungExpiry] Job completed. Deactivated %d expired Förderungen.", len(expired))
	return nil
}
