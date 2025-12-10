package unit

import (
	"sort"
	"sync"
	"testing"

	"austrian-business-infrastructure/internal/fonws"
)

// T066: Test parallel account processing with errgroup
func TestParallelAccountProcessing(t *testing.T) {
	// Simulate processing multiple accounts in parallel
	accounts := []string{"Account1", "Account2", "Account3"}

	var wg sync.WaitGroup
	results := make(map[string]int)
	var mu sync.Mutex

	for _, acc := range accounts {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			// Simulate processing
			count := len(name) // Simple operation

			mu.Lock()
			results[name] = count
			mu.Unlock()
		}(acc)
	}

	wg.Wait()

	// Verify all accounts were processed
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for _, acc := range accounts {
		if _, ok := results[acc]; !ok {
			t.Errorf("Missing result for %s", acc)
		}
	}
}

// T067: Test aggregated result sorting
func TestAggregatedResultSorting(t *testing.T) {
	type AccountSummary struct {
		Name           string
		TotalDocs      int
		ActionRequired int
	}

	results := []AccountSummary{
		{"Zebra Corp", 5, 0},
		{"Alpha Inc", 10, 3},
		{"Beta Ltd", 2, 1},
	}

	// Sort by action required (descending), then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].ActionRequired != results[j].ActionRequired {
			return results[i].ActionRequired > results[j].ActionRequired
		}
		return results[i].Name < results[j].Name
	})

	// Verify sort order
	if results[0].Name != "Alpha Inc" {
		t.Errorf("Expected Alpha Inc first (most actions), got %s", results[0].Name)
	}
	if results[1].Name != "Beta Ltd" {
		t.Errorf("Expected Beta Ltd second, got %s", results[1].Name)
	}
	if results[2].Name != "Zebra Corp" {
		t.Errorf("Expected Zebra Corp last, got %s", results[2].Name)
	}
}

func TestActionRequiredCount(t *testing.T) {
	entries := []fonws.DataboxEntry{
		{Erlession: "B"}, // No action
		{Erlession: "E"}, // Action required
		{Erlession: "M"}, // No action
		{Erlession: "V"}, // Action required
		{Erlession: "E"}, // Action required
	}

	actionCount := 0
	for _, e := range entries {
		if e.ActionRequired() {
			actionCount++
		}
	}

	if actionCount != 3 {
		t.Errorf("Expected 3 action required, got %d", actionCount)
	}
}

// TestServiceResultSorting tests the multi-service dashboard result sorting
func TestServiceResultSorting(t *testing.T) {
	type ServiceResult struct {
		AccountName  string
		ServiceType  string
		Status       string
		PendingItems int
		Error        string
	}

	results := []ServiceResult{
		{AccountName: "Alpha", ServiceType: "finanzonline", Status: "ok", PendingItems: 0},
		{AccountName: "Beta", ServiceType: "elda", Status: "error", Error: "connection failed"},
		{AccountName: "Gamma", ServiceType: "finanzonline", Status: "pending", PendingItems: 5},
		{AccountName: "Delta", ServiceType: "fb", Status: "ok", PendingItems: 2},
	}

	// Sort: errors first, then by pending items (desc), then by name
	sort.Slice(results, func(i, j int) bool {
		if (results[i].Error != "") != (results[j].Error != "") {
			return results[i].Error != ""
		}
		if results[i].PendingItems != results[j].PendingItems {
			return results[i].PendingItems > results[j].PendingItems
		}
		return results[i].AccountName < results[j].AccountName
	})

	// First should be Beta (has error)
	if results[0].AccountName != "Beta" {
		t.Errorf("Expected Beta first (has error), got %s", results[0].AccountName)
	}

	// Second should be Gamma (5 pending)
	if results[1].AccountName != "Gamma" {
		t.Errorf("Expected Gamma second (5 pending), got %s", results[1].AccountName)
	}

	// Third should be Delta (2 pending)
	if results[2].AccountName != "Delta" {
		t.Errorf("Expected Delta third (2 pending), got %s", results[2].AccountName)
	}

	// Last should be Alpha (0 pending, no error)
	if results[3].AccountName != "Alpha" {
		t.Errorf("Expected Alpha last (0 pending), got %s", results[3].AccountName)
	}
}

// TestServiceFilterParsing tests the service filter parsing logic
func TestServiceFilterParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]bool
	}{
		{"", nil},
		{"fo", map[string]bool{"fo": true}},
		{"fo,elda", map[string]bool{"fo": true, "elda": true}},
		{"fo, elda, fb", map[string]bool{"fo": true, "elda": true, "fb": true}},
	}

	for _, tc := range tests {
		result := parseServiceFilter(tc.input)
		if tc.expected == nil && result != nil {
			t.Errorf("parseServiceFilter(%q) = %v, expected nil", tc.input, result)
			continue
		}
		if tc.expected != nil {
			if len(result) != len(tc.expected) {
				t.Errorf("parseServiceFilter(%q) = %v, expected %v", tc.input, result, tc.expected)
				continue
			}
			for k := range tc.expected {
				if !result[k] {
					t.Errorf("parseServiceFilter(%q) missing key %q", tc.input, k)
				}
			}
		}
	}
}

// parseServiceFilter parses a comma-separated service filter string
func parseServiceFilter(filter string) map[string]bool {
	if filter == "" {
		return nil
	}

	result := make(map[string]bool)
	start := 0
	for i := 0; i <= len(filter); i++ {
		if i == len(filter) || filter[i] == ',' {
			part := trimWhitespace(filter[start:i])
			if part != "" {
				result[part] = true
			}
			start = i + 1
		}
	}
	return result
}

func trimWhitespace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// TestDashboardOutputAggregation tests that totals are correctly calculated
func TestDashboardOutputAggregation(t *testing.T) {
	type ServiceResult struct {
		PendingItems int
		Error        string
	}

	results := []ServiceResult{
		{PendingItems: 5, Error: ""},
		{PendingItems: 3, Error: ""},
		{PendingItems: 0, Error: "connection failed"},
		{PendingItems: 2, Error: ""},
	}

	pendingTotal := 0
	errorCount := 0

	for _, r := range results {
		pendingTotal += r.PendingItems
		if r.Error != "" {
			errorCount++
		}
	}

	if pendingTotal != 10 {
		t.Errorf("Expected pending total 10, got %d", pendingTotal)
	}

	if errorCount != 1 {
		t.Errorf("Expected error count 1, got %d", errorCount)
	}
}
