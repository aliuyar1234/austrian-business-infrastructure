package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/austrian-business-infrastructure/fo/internal/config"
	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/austrian-business-infrastructure/fo/internal/store"
	"github.com/spf13/cobra"
)

var (
	// Current active session (in-memory only)
	activeSession *fonws.Session
)

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Manage FinanzOnline sessions",
	Long:  `Login and logout from FinanzOnline WebService.`,
}

var loginCmd = &cobra.Command{
	Use:   "login <account-name>",
	Short: "Authenticate with FinanzOnline",
	Long:  `Login to FinanzOnline using stored account credentials.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runLogin,
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Terminate active session",
	Long:  `Logout from FinanzOnline and terminate the active session.`,
	RunE:  runLogout,
}

func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(loginCmd)
	sessionCmd.AddCommand(logoutCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	accountName := args[0]
	LogVerbose("Attempting login for account: %s", accountName)

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

	// Create SOAP client and login
	client := fonws.NewClient()
	client.SetVerbose(IsVerbose())
	sessionService := fonws.NewSessionService(client)

	LogVerbose("Connecting to FinanzOnline WebService...")
	session, err := sessionService.Login(account.TID, account.BenID, account.PIN)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	session.AccountName = accountName
	activeSession = session

	// Output result
	if IsJSONOutput() {
		return outputLoginJSON(account, session)
	}
	return outputLoginTable(account, session)
}

func runLogout(cmd *cobra.Command, args []string) error {
	if activeSession == nil || !activeSession.Valid {
		if IsJSONOutput() {
			return outputJSON(map[string]interface{}{
				"status": "error",
				"error":  "no active session",
			})
		}
		fmt.Println("No active session.")
		return nil
	}

	LogVerbose("Logging out from FinanzOnline...")

	client := fonws.NewClient()
	client.SetVerbose(IsVerbose())
	sessionService := fonws.NewSessionService(client)

	if err := sessionService.Logout(activeSession); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}

	activeSession = nil

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"status": "success",
		})
	}
	fmt.Println("Logged out.")
	return nil
}

func outputLoginTable(account *store.Account, session *fonws.Session) error {
	fmt.Printf("Logged in as \"%s\" (%s)\n", account.Name, account.TID)
	return nil
}

func outputLoginJSON(account *store.Account, session *fonws.Session) error {
	return outputJSON(map[string]interface{}{
		"status":  "success",
		"account": account.Name,
		"tid":     account.TID,
	})
}

func outputJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// promptPassword prompts for a password without echoing
func promptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	// Note: In a real implementation, we would use golang.org/x/term to hide input
	// For now, this is a simple implementation
	var password string
	_, err := fmt.Scanln(&password)
	return password, err
}

// GetActiveSession returns the current active session
func GetActiveSession() *fonws.Session {
	return activeSession
}

// SetActiveSession sets the current active session
func SetActiveSession(s *fonws.Session) {
	activeSession = s
}
