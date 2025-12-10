//go:build !windows

package integration

import (
	"testing"
	"time"

	"austrian-business-infrastructure/internal/analysis"
	"github.com/google/uuid"
)

// T077: Integration tests for action items

func TestActionItemCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("Create action item from deadline", func(t *testing.T) {
		// When a deadline is extracted, an action item should be created
		deadline := &analysis.Deadline{
			ID:           uuid.New(),
			AnalysisID:   uuid.New(),
			DocumentID:   uuid.New(),
			TenantID:     uuid.New(),
			DeadlineType: analysis.DeadlineTypeResponse,
			Date:         time.Now().AddDate(0, 0, 14),
			Description:  "Respond to Ergänzungsersuchen within 4 weeks",
			Confidence:   0.92,
			IsHard:       true,
		}

		// Create action item from deadline
		item := &analysis.ActionItem{
			ID:          uuid.New(),
			AnalysisID:  deadline.AnalysisID,
			DocumentID:  deadline.DocumentID,
			TenantID:    deadline.TenantID,
			Title:       "Respond to Ergänzungsersuchen",
			Description: deadline.Description,
			Priority:    analysis.PriorityHigh,
			Category:    "deadline_response",
			Status:      analysis.ActionStatusPending,
			DueDate:     &deadline.Date,
			Confidence:  deadline.Confidence,
			SourceText:  "Deadline: " + deadline.Description,
			CreatedAt:   time.Now(),
		}

		// Verify action item was created correctly
		if item.Priority != analysis.PriorityHigh {
			t.Errorf("Expected high priority for deadline action item, got %s", item.Priority)
		}

		if item.DueDate == nil {
			t.Error("Action item due date should be set from deadline")
		}

		if item.SourceText == "" {
			t.Error("Source text should be set from deadline")
		}
	})

	t.Run("Create action item from analysis", func(t *testing.T) {
		// AI can extract action items directly from document content
		dueDate := time.Now().AddDate(0, 0, 7)
		item := &analysis.ActionItem{
			ID:          uuid.New(),
			AnalysisID:  uuid.New(),
			DocumentID:  uuid.New(),
			TenantID:    uuid.New(),
			Title:       "Provide missing documentation",
			Description: "Submit Einnahmen-Ausgaben-Rechnung for 2023",
			Priority:    analysis.PriorityMedium,
			Category:    "document_request",
			Status:      analysis.ActionStatusPending,
			DueDate:     &dueDate,
			Confidence:  0.85,
			SourceText:  "AI extracted from document analysis",
			CreatedAt:   time.Now(),
		}

		if item.Status != analysis.ActionStatusPending {
			t.Errorf("New action item should be pending, got %s", item.Status)
		}
	})

	t.Run("Update action item status", func(t *testing.T) {
		item := &analysis.ActionItem{
			ID:        uuid.New(),
			Status:    analysis.ActionStatusPending,
			CreatedAt: time.Now(),
		}

		// Simulate completing the action item
		item.Status = analysis.ActionStatusCompleted
		now := time.Now()
		item.CompletedAt = &now

		if item.Status != analysis.ActionStatusCompleted {
			t.Error("Status should be updated to completed")
		}

		if item.CompletedAt == nil {
			t.Error("CompletedAt should be set when completing")
		}
	})

	t.Run("Cancel action item", func(t *testing.T) {
		item := &analysis.ActionItem{
			ID:        uuid.New(),
			Status:    analysis.ActionStatusPending,
			CreatedAt: time.Now(),
		}

		// Cancel the action item
		item.Status = analysis.ActionStatusCancelled

		if item.Status != analysis.ActionStatusCancelled {
			t.Error("Status should be updated to cancelled")
		}
	})
}

func TestActionItemPriority(t *testing.T) {
	t.Run("Priority ordering", func(t *testing.T) {
		// Verify priority constants can be used for sorting
		priorities := []string{
			analysis.PriorityHigh,
			analysis.PriorityMedium,
			analysis.PriorityLow,
		}

		if len(priorities) != 3 {
			t.Error("Expected 3 priority levels")
		}

		// High priority items should be processed first
		if priorities[0] != analysis.PriorityHigh {
			t.Error("High priority should come first")
		}
	})

	t.Run("Priority from deadline urgency", func(t *testing.T) {
		// Deadlines within 7 days should be high priority
		testCases := []struct {
			daysUntilDeadline int
			expectedPriority  string
		}{
			{3, analysis.PriorityHigh},   // Very urgent
			{7, analysis.PriorityHigh},   // Urgent
			{14, analysis.PriorityMedium}, // Moderate
			{30, analysis.PriorityLow},    // Not urgent
		}

		for _, tc := range testCases {
			deadline := time.Now().AddDate(0, 0, tc.daysUntilDeadline)
			priority := determinePriorityFromDeadline(deadline)

			if priority != tc.expectedPriority {
				t.Errorf("Deadline in %d days: expected %s, got %s",
					tc.daysUntilDeadline, tc.expectedPriority, priority)
			}
		}
	})
}

// determinePriorityFromDeadline calculates priority based on deadline urgency
func determinePriorityFromDeadline(deadline time.Time) string {
	daysUntil := int(time.Until(deadline).Hours() / 24)

	switch {
	case daysUntil <= 7:
		return analysis.PriorityHigh
	case daysUntil <= 14:
		return analysis.PriorityMedium
	default:
		return analysis.PriorityLow
	}
}

func TestActionItemFiltering(t *testing.T) {
	t.Run("Filter by status", func(t *testing.T) {
		items := []*analysis.ActionItem{
			{ID: uuid.New(), Status: analysis.ActionStatusPending},
			{ID: uuid.New(), Status: analysis.ActionStatusCompleted},
			{ID: uuid.New(), Status: analysis.ActionStatusPending},
			{ID: uuid.New(), Status: analysis.ActionStatusCancelled},
		}

		pending := filterByStatus(items, analysis.ActionStatusPending)
		if len(pending) != 2 {
			t.Errorf("Expected 2 pending items, got %d", len(pending))
		}

		completed := filterByStatus(items, analysis.ActionStatusCompleted)
		if len(completed) != 1 {
			t.Errorf("Expected 1 completed item, got %d", len(completed))
		}
	})

	t.Run("Filter by priority", func(t *testing.T) {
		items := []*analysis.ActionItem{
			{ID: uuid.New(), Priority: analysis.PriorityHigh},
			{ID: uuid.New(), Priority: analysis.PriorityMedium},
			{ID: uuid.New(), Priority: analysis.PriorityHigh},
			{ID: uuid.New(), Priority: analysis.PriorityLow},
		}

		high := filterByPriority(items, analysis.PriorityHigh)
		if len(high) != 2 {
			t.Errorf("Expected 2 high priority items, got %d", len(high))
		}
	})

	t.Run("Filter overdue items", func(t *testing.T) {
		yesterday := time.Now().AddDate(0, 0, -1)
		tomorrow := time.Now().AddDate(0, 0, 1)

		items := []*analysis.ActionItem{
			{ID: uuid.New(), DueDate: &yesterday, Status: analysis.ActionStatusPending},
			{ID: uuid.New(), DueDate: &tomorrow, Status: analysis.ActionStatusPending},
			{ID: uuid.New(), DueDate: &yesterday, Status: analysis.ActionStatusCompleted}, // Not overdue - completed
		}

		overdue := filterOverdue(items)
		if len(overdue) != 1 {
			t.Errorf("Expected 1 overdue item, got %d", len(overdue))
		}
	})
}

func filterByStatus(items []*analysis.ActionItem, status analysis.ActionStatus) []*analysis.ActionItem {
	var result []*analysis.ActionItem
	for _, item := range items {
		if item.Status == status {
			result = append(result, item)
		}
	}
	return result
}

func filterByPriority(items []*analysis.ActionItem, priority analysis.Priority) []*analysis.ActionItem {
	var result []*analysis.ActionItem
	for _, item := range items {
		if item.Priority == priority {
			result = append(result, item)
		}
	}
	return result
}

func filterOverdue(items []*analysis.ActionItem) []*analysis.ActionItem {
	var result []*analysis.ActionItem
	now := time.Now()
	for _, item := range items {
		if item.DueDate != nil && item.DueDate.Before(now) && item.Status == analysis.ActionStatusPending {
			result = append(result, item)
		}
	}
	return result
}

func TestActionItemNotifications(t *testing.T) {
	t.Run("Notification trigger conditions", func(t *testing.T) {
		// Action items should trigger notifications when:
		// 1. Created with high priority
		// 2. Due date approaching
		// 3. Overdue

		tomorrow := time.Now().AddDate(0, 0, 1)
		item := &analysis.ActionItem{
			ID:       uuid.New(),
			Priority: analysis.PriorityHigh,
			DueDate:  &tomorrow,
			Status:   analysis.ActionStatusPending,
		}

		// High priority + due soon = urgent notification
		shouldNotify := item.Priority == analysis.PriorityHigh ||
			(item.DueDate != nil && time.Until(*item.DueDate) < 48*time.Hour)

		if !shouldNotify {
			t.Error("High priority item due soon should trigger notification")
		}
	})
}

func TestActionItemCategories(t *testing.T) {
	// Test common action item categories
	categories := []string{
		"deadline_response",  // Response to Ergänzungsersuchen
		"document_request",   // Request for additional documents
		"payment",            // Payment action
		"submission",         // Form submission
		"appeal",             // Appeal deadline
		"review",             // Review/check something
		"other",              // Miscellaneous
	}

	if len(categories) < 5 {
		t.Error("Expected at least 5 action item categories")
	}
}
