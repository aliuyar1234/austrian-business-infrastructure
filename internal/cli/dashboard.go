package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/austrian-business-infrastructure/fo/internal/config"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/austrian-business-infrastructure/fo/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	dashboardServices string
	dashboardAll      bool
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Unified status across all services",
	Long: `Show consolidated status from all configured accounts and services.
Supports: FinanzOnline (fo), ELDA (elda), Firmenbuch (fb)

Examples:
  fo dashboard --all           # Check all accounts
  fo dashboard --services fo   # Check only FinanzOnline accounts
  fo dashboard --services fo,elda  # Check FO and ELDA accounts`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.Flags().BoolVar(&dashboardAll, "all", false, "Check all accounts")
	dashboardCmd.Flags().StringVar(&dashboardServices, "services", "", "Filter by service types (fo,elda,fb)")
}

func runDashboard(cmd *cobra.Command, args []string) error {
	LogVerbose("Running dashboard")

	// Get config directory
	cfgDir, err := config.GetConfigDir(GetConfigDir())
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Prompt for master password
	masterPassword, err := promptPassword("Master password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Load credential store
	credPath := config.GetCredentialPath(cfgDir)
	credStore, err := store.Load(credPath, masterPassword)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Use multi-service dashboard
	dashboard := NewMultiServiceDashboard(credStore)
	results, err := dashboard.CheckAllServices(context.Background(), dashboardServices)
	if err != nil {
		return fmt.Errorf("failed to check services: %w", err)
	}

	if IsJSONOutput() {
		return OutputMultiServiceJSON(results)
	}

	OutputMultiServiceTable(results)
	return nil
}

// ServiceResult holds the result of checking a service for an account
type ServiceResult struct {
	AccountName  string `json:"account"`
	ServiceType  string `json:"service_type"`
	Identifier   string `json:"identifier"` // TID, DienstgeberNr, etc.
	Status       string `json:"status"`     // ok, error, pending
	PendingItems int    `json:"pending_items"`
	Details      string `json:"details,omitempty"`
	Error        string `json:"error,omitempty"`
}

// AccountResult holds the result of checking a single account's databox
type AccountResult struct {
	AccountName    string               `json:"account"`
	TID            string               `json:"tid"`
	Entries        []fonws.DataboxEntry `json:"entries,omitempty"`
	TotalDocs      int                  `json:"total_docs"`
	ActionRequired int                  `json:"action_required"`
	Error          string               `json:"error,omitempty"`
}

// DashboardService handles batch operations across multiple accounts
type DashboardService struct {
	credStore *store.CredentialStore
	client    *fonws.Client
	fromDate  string
	toDate    string
}

// NewDashboardService creates a new dashboard service
func NewDashboardService(credStore *store.CredentialStore, fromDate, toDate string) *DashboardService {
	client := fonws.NewClient()
	client.SetVerbose(IsVerbose())
	return &DashboardService{
		credStore: credStore,
		client:    client,
		fromDate:  fromDate,
		toDate:    toDate,
	}
}

// CheckAllAccounts checks the databox for all accounts in parallel
func (d *DashboardService) CheckAllAccounts(ctx context.Context) ([]AccountResult, error) {
	accountNames := d.credStore.ListAccounts()
	if len(accountNames) == 0 {
		return []AccountResult{}, nil
	}

	results := make([]AccountResult, len(accountNames))
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	// Print progress indicator
	fmt.Fprintf(LogWriter(), "Checking %d accounts...\n", len(accountNames))

	for i, name := range accountNames {
		idx := i
		accountName := name

		g.Go(func() error {
			result := d.checkAccount(ctx, accountName)

			mu.Lock()
			results[idx] = result
			mu.Unlock()

			return nil // Don't fail the group on individual account errors
		})
	}

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort results: accounts with action required first, then by name
	sort.Slice(results, func(i, j int) bool {
		if results[i].ActionRequired != results[j].ActionRequired {
			return results[i].ActionRequired > results[j].ActionRequired
		}
		return results[i].AccountName < results[j].AccountName
	})

	return results, nil
}

func (d *DashboardService) checkAccount(ctx context.Context, accountName string) AccountResult {
	result := AccountResult{AccountName: accountName}

	account, err := d.credStore.GetAccount(accountName)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	result.TID = account.TID

	// Check context cancellation
	select {
	case <-ctx.Done():
		result.Error = "cancelled"
		return result
	default:
	}

	// Login
	sessionService := fonws.NewSessionService(d.client)
	session, err := sessionService.Login(account.TID, account.BenID, account.PIN)
	if err != nil {
		result.Error = fmt.Sprintf("login failed: %v", err)
		return result
	}
	defer sessionService.Logout(session)

	// Get databox info
	databoxService := fonws.NewDataboxService(d.client)
	entries, err := databoxService.GetInfo(session, d.fromDate, d.toDate)
	if err != nil {
		result.Error = fmt.Sprintf("databox failed: %v", err)
		return result
	}

	result.Entries = entries
	result.TotalDocs = len(entries)
	for _, e := range entries {
		if e.ActionRequired() {
			result.ActionRequired++
		}
	}

	return result
}

// OutputDashboardTable outputs results in table format
func OutputDashboardTable(results []AccountResult) {
	fmt.Println()
	fmt.Println("ACCOUNT                       TID            TOTAL    ACTION REQ.  STATUS")
	fmt.Println("--------------------------------------------------------------------------")

	totalDocs := 0
	totalActions := 0
	errorCount := 0

	for _, r := range results {
		status := "OK"
		if r.Error != "" {
			status = "ERROR"
			errorCount++
		}

		actionStr := fmt.Sprintf("%d", r.ActionRequired)
		if r.ActionRequired > 0 {
			actionStr = fmt.Sprintf("%d ⚠️", r.ActionRequired)
		}

		fmt.Printf("%-30s%-15s%-9d%-13s%s\n",
			truncate(r.AccountName, 29),
			r.TID,
			r.TotalDocs,
			actionStr,
			status,
		)

		totalDocs += r.TotalDocs
		totalActions += r.ActionRequired
	}

	fmt.Println("--------------------------------------------------------------------------")
	fmt.Printf("TOTAL: %d accounts, %d documents, %d actions required",
		len(results), totalDocs, totalActions)
	if errorCount > 0 {
		fmt.Printf(", %d errors", errorCount)
	}
	fmt.Println()
}

// OutputDashboardJSON outputs results in JSON format
func OutputDashboardJSON(results []AccountResult) error {
	totalDocs := 0
	totalActions := 0
	errorCount := 0

	for _, r := range results {
		totalDocs += r.TotalDocs
		totalActions += r.ActionRequired
		if r.Error != "" {
			errorCount++
		}
	}

	return outputJSON(map[string]interface{}{
		"accounts":        results,
		"total_accounts":  len(results),
		"total_documents": totalDocs,
		"total_actions":   totalActions,
		"errors":          errorCount,
	})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// LogWriter returns stderr for logging progress
func LogWriter() *logWriterType {
	return &logWriterType{}
}

type logWriterType struct{}

func (l *logWriterType) Write(p []byte) (n int, err error) {
	return fmt.Fprint(errWriter, string(p))
}

// MultiServiceDashboard handles checking status across multiple service types
type MultiServiceDashboard struct {
	credStore *store.CredentialStore
}

// NewMultiServiceDashboard creates a new multi-service dashboard
func NewMultiServiceDashboard(credStore *store.CredentialStore) *MultiServiceDashboard {
	return &MultiServiceDashboard{
		credStore: credStore,
	}
}

// CheckAllServices checks all configured services in parallel
func (d *MultiServiceDashboard) CheckAllServices(ctx context.Context, serviceFilter string) ([]ServiceResult, error) {
	accountNames := d.credStore.ListAccounts()
	if len(accountNames) == 0 {
		return []ServiceResult{}, nil
	}

	// Parse service filter
	filterSet := parseServiceFilter(serviceFilter)

	var results []ServiceResult
	var mu sync.Mutex

	g, ctx := errgroup.WithContext(ctx)

	fmt.Fprintf(os.Stderr, "Checking %d accounts...\n", len(accountNames))

	for _, name := range accountNames {
		accountName := name

		g.Go(func() error {
			serviceResults := d.checkAccountServices(ctx, accountName, filterSet)

			mu.Lock()
			results = append(results, serviceResults...)
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Sort results: errors first, then by pending items, then by account name
	sort.Slice(results, func(i, j int) bool {
		// Errors first
		if (results[i].Error != "") != (results[j].Error != "") {
			return results[i].Error != ""
		}
		// Then by pending items (descending)
		if results[i].PendingItems != results[j].PendingItems {
			return results[i].PendingItems > results[j].PendingItems
		}
		// Then by account name
		return results[i].AccountName < results[j].AccountName
	})

	return results, nil
}

func (d *MultiServiceDashboard) checkAccountServices(ctx context.Context, accountName string, filterSet map[string]bool) []ServiceResult {
	var results []ServiceResult

	account, err := d.credStore.GetAccount(accountName)
	if err != nil {
		return []ServiceResult{{
			AccountName: accountName,
			ServiceType: "unknown",
			Status:      "error",
			Error:       err.Error(),
		}}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return []ServiceResult{{
			AccountName: accountName,
			ServiceType: "unknown",
			Status:      "cancelled",
			Error:       "cancelled",
		}}
	default:
	}

	// For now, all accounts are treated as FinanzOnline
	// TODO: When ExtendedAccount is used in CredentialStore, check account type
	if len(filterSet) == 0 || filterSet["fo"] || filterSet["finanzonline"] {
		result := d.checkFinanzOnline(ctx, account)
		results = append(results, result)
	}

	return results
}

func (d *MultiServiceDashboard) checkFinanzOnline(ctx context.Context, account *store.Account) ServiceResult {
	result := ServiceResult{
		AccountName: account.Name,
		ServiceType: "finanzonline",
		Identifier:  account.TID,
		Status:      "ok",
	}

	client := fonws.NewClient()
	client.SetVerbose(IsVerbose())
	sessionService := fonws.NewSessionService(client)

	session, err := sessionService.Login(account.TID, account.BenID, account.PIN)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("login failed: %v", err)
		return result
	}
	defer sessionService.Logout(session)

	// Get databox info
	databoxService := fonws.NewDataboxService(client)
	entries, err := databoxService.GetInfo(session, "", "")
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("databox failed: %v", err)
		return result
	}

	result.PendingItems = 0
	for _, e := range entries {
		if e.ActionRequired() {
			result.PendingItems++
		}
	}

	if result.PendingItems > 0 {
		result.Status = "pending"
		result.Details = fmt.Sprintf("%d docs, %d require action", len(entries), result.PendingItems)
	} else {
		result.Details = fmt.Sprintf("%d docs", len(entries))
	}

	return result
}

func parseServiceFilter(filter string) map[string]bool {
	if filter == "" {
		return nil
	}

	result := make(map[string]bool)
	for _, s := range splitAndTrim(filter, ",") {
		result[s] = true
	}
	return result
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range splitString(s, sep) {
		p = trimString(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func trimString(s string) string {
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

// OutputMultiServiceTable outputs multi-service results in table format
func OutputMultiServiceTable(results []ServiceResult) {
	fmt.Println()
	fmt.Println("ACCOUNT                       SERVICE        IDENTIFIER       PENDING  STATUS")
	fmt.Println("-------------------------------------------------------------------------------")

	pendingTotal := 0
	errorCount := 0

	for _, r := range results {
		status := r.Status
		if r.Error != "" {
			status = "ERROR"
			errorCount++
		}

		pendingStr := "-"
		if r.PendingItems > 0 {
			pendingStr = fmt.Sprintf("%d", r.PendingItems)
			pendingTotal += r.PendingItems
		}

		fmt.Printf("%-30s%-15s%-17s%-9s%s\n",
			truncate(r.AccountName, 29),
			r.ServiceType,
			truncate(r.Identifier, 16),
			pendingStr,
			status,
		)
	}

	fmt.Println("-------------------------------------------------------------------------------")
	fmt.Printf("TOTAL: %d services", len(results))
	if pendingTotal > 0 {
		fmt.Printf(", %d pending items", pendingTotal)
	}
	if errorCount > 0 {
		fmt.Printf(", %d errors", errorCount)
	}
	fmt.Println()
}

// OutputMultiServiceJSON outputs multi-service results in JSON format
func OutputMultiServiceJSON(results []ServiceResult) error {
	pendingTotal := 0
	errorCount := 0

	for _, r := range results {
		pendingTotal += r.PendingItems
		if r.Error != "" {
			errorCount++
		}
	}

	return outputJSON(map[string]interface{}{
		"services":       results,
		"total_services": len(results),
		"total_pending":  pendingTotal,
		"errors":         errorCount,
	})
}
