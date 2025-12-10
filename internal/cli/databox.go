package cli

import (
	"context"
	"fmt"
	"time"

	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/fonws"
	"austrian-business-infrastructure/internal/store"
	"github.com/spf13/cobra"
)

var (
	databoxFromDate string
	databoxToDate   string
	databoxAll      bool
	downloadOutput  string
)

var databoxCmd = &cobra.Command{
	Use:   "databox",
	Short: "Access FinanzOnline Databox",
	Long:  `List and download documents from the FinanzOnline Databox.`,
}

var databoxListCmd = &cobra.Command{
	Use:   "list <account>",
	Short: "List databox documents",
	Long:  `List documents in the FinanzOnline Databox for an account.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDataboxList,
}

var databoxDownloadCmd = &cobra.Command{
	Use:   "download <account> <applkey>",
	Short: "Download a document",
	Long:  `Download a specific document from the FinanzOnline Databox.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runDataboxDownload,
}

func init() {
	rootCmd.AddCommand(databoxCmd)
	databoxCmd.AddCommand(databoxListCmd)
	databoxCmd.AddCommand(databoxDownloadCmd)

	// Flags for list command
	databoxListCmd.Flags().StringVar(&databoxFromDate, "from", "", "Filter documents from date (YYYY-MM-DD)")
	databoxListCmd.Flags().StringVar(&databoxToDate, "to", "", "Filter documents until date (YYYY-MM-DD)")
	databoxListCmd.Flags().BoolVar(&databoxAll, "all", false, "List documents from all accounts")

	// Flags for download command
	databoxDownloadCmd.Flags().StringVarP(&downloadOutput, "output", "o", ".", "Output directory for downloaded file")
}

func runDataboxList(cmd *cobra.Command, args []string) error {
	if databoxAll {
		return runDataboxListAll()
	}

	if len(args) < 1 {
		return fmt.Errorf("account name required (or use --all flag)")
	}

	accountName := args[0]
	LogVerbose("Listing databox for account: %s", accountName)

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

	// Find account
	account, err := credStore.GetAccount(accountName)
	if err != nil {
		return fmt.Errorf("account not found: %s", accountName)
	}

	// Login
	client := fonws.NewClient()
	client.SetVerbose(IsVerbose())
	sessionService := fonws.NewSessionService(client)

	LogVerbose("Logging in...")
	session, err := sessionService.Login(account.TID, account.BenID, account.PIN)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer sessionService.Logout(session)

	// Get databox info
	databoxService := fonws.NewDataboxService(client)
	entries, err := databoxService.GetInfo(session, databoxFromDate, databoxToDate)
	if err != nil {
		return fmt.Errorf("failed to get databox info: %w", err)
	}

	return outputDataboxList(accountName, entries)
}

func runDataboxListAll() error {
	LogVerbose("Listing databox for all accounts")

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

	accountNames := credStore.ListAccounts()
	if len(accountNames) == 0 {
		if IsJSONOutput() {
			return outputJSON(map[string]interface{}{
				"accounts": []interface{}{},
			})
		}
		fmt.Println("No accounts stored.")
		return nil
	}

	// Use dashboard service for parallel processing
	dashboard := NewDashboardService(credStore, databoxFromDate, databoxToDate)
	results, err := dashboard.CheckAllAccounts(context.Background())
	if err != nil {
		return fmt.Errorf("failed to check accounts: %w", err)
	}

	if IsJSONOutput() {
		return OutputDashboardJSON(results)
	}

	OutputDashboardTable(results)
	return nil
}

func runDataboxDownload(cmd *cobra.Command, args []string) error {
	accountName := args[0]
	applkey := args[1]
	LogVerbose("Downloading document %s for account: %s", applkey, accountName)

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

	// Find account
	account, err := credStore.GetAccount(accountName)
	if err != nil {
		return fmt.Errorf("account not found: %s", accountName)
	}

	// Login
	client := fonws.NewClient()
	client.SetVerbose(IsVerbose())
	sessionService := fonws.NewSessionService(client)

	LogVerbose("Logging in...")
	session, err := sessionService.Login(account.TID, account.BenID, account.PIN)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	defer sessionService.Logout(session)

	// Download document
	databoxService := fonws.NewDataboxService(client)
	outputPath, err := databoxService.Download(session, applkey, downloadOutput)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"status": "success",
			"file":   outputPath,
		})
	}
	fmt.Printf("Downloaded: %s\n", outputPath)
	return nil
}

func outputDataboxList(accountName string, entries []fonws.DataboxEntry) error {
	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"account": accountName,
			"entries": entries,
			"count":   len(entries),
		})
	}

	if len(entries) == 0 {
		fmt.Println("No documents found.")
		return nil
	}

	// Table output
	fmt.Printf("Databox for %s (%d documents)\n\n", accountName, len(entries))
	fmt.Println("DATE                  TYPE                  DESCRIPTION                     APPLKEY         ACTION")
	fmt.Println("------------------------------------------------------------------------------------------------------")

	for _, e := range entries {
		actionFlag := "  "
		if e.ActionRequired() {
			actionFlag = "⚠️"
		}

		// Parse and format date
		date := e.TsZust
		if t, err := time.Parse("2006-01-02T15:04:05", e.TsZust); err == nil {
			date = t.Format("2006-01-02 15:04")
		}

		// Truncate description if needed
		desc := e.Filebez
		if len(desc) > 30 {
			desc = desc[:27] + "..."
		}

		fmt.Printf("%-22s%-22s%-32s%-16s%s\n", date, e.TypeName(), desc, e.Applkey, actionFlag)
	}

	return nil
}
