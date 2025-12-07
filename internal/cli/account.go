package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/austrian-business-infrastructure/fo/internal/config"
	"github.com/austrian-business-infrastructure/fo/internal/store"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Manage FinanzOnline accounts",
	Long:  `Add, list, and remove FinanzOnline WebService accounts.`,
}

var accountAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new account",
	Long: `Add a new account with interactive prompts.
Supported types:
  - finanzonline: FinanzOnline WebService account (default)
  - elda: ELDA social insurance account
  - firmenbuch: Firmenbuch API access`,
	Args: cobra.ExactArgs(1),
	RunE: runAccountAdd,
}

var accountTypeFlag string

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	Long:  `List all stored FinanzOnline accounts (without sensitive data).`,
	RunE:  runAccountList,
}

var accountRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an account",
	Long:  `Remove a stored FinanzOnline account by name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runAccountRemove,
}

func init() {
	rootCmd.AddCommand(accountCmd)
	accountCmd.AddCommand(accountAddCmd)
	accountCmd.AddCommand(accountListCmd)
	accountCmd.AddCommand(accountRemoveCmd)

	// Add --type flag for account add
	accountAddCmd.Flags().StringVarP(&accountTypeFlag, "type", "t", "finanzonline",
		"Account type: finanzonline, elda, or firmenbuch")
}

func runAccountAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	LogVerbose("Adding account: %s (type: %s)", name, accountTypeFlag)

	// Validate account type
	accType := store.AccountType(accountTypeFlag)
	if !store.IsValidAccountType(accType) {
		return fmt.Errorf("invalid account type: %s (valid: finanzonline, elda, firmenbuch)", accountTypeFlag)
	}

	// Get config directory
	cfgDir, err := config.GetConfigDir(GetConfigDir())
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Create extended account based on type
	extAccount := &store.ExtendedAccount{
		Name: name,
		Type: accType,
	}

	switch accType {
	case store.AccountTypeFinanzOnline:
		if err := promptFinanzOnlineCredentials(reader, extAccount); err != nil {
			return err
		}
	case store.AccountTypeELDA:
		if err := promptELDACredentials(reader, extAccount); err != nil {
			return err
		}
	case store.AccountTypeFirmenbuch:
		if err := promptFirmenbuchCredentials(reader, extAccount); err != nil {
			return err
		}
	}

	// Validate account
	if err := extAccount.Validate(); err != nil {
		return fmt.Errorf("invalid account: %w", err)
	}

	// Prompt for master password
	masterPassword, err := promptPassword("Master password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Load or create credential store
	credPath := config.GetCredentialPath(cfgDir)
	var credStore *store.CredentialStore

	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		LogVerbose("Creating new credential store")
		credStore = store.NewCredentialStore()
	} else {
		LogVerbose("Loading existing credential store")
		credStore, err = store.Load(credPath, masterPassword)
		if err != nil {
			return fmt.Errorf("failed to load credentials: %w", err)
		}
	}

	// For backward compatibility, convert to legacy Account for FinanzOnline
	account := extAccount.ToAccount()
	if err := credStore.AddAccount(*account); err != nil {
		return fmt.Errorf("failed to add account: %w", err)
	}

	// Save
	if err := credStore.Save(credPath, masterPassword); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"status":  "success",
			"action":  "add",
			"account": name,
			"type":    accountTypeFlag,
		})
	}
	fmt.Printf("Account \"%s\" (%s) added.\n", name, accType)
	return nil
}

func promptFinanzOnlineCredentials(reader *bufio.Reader, acc *store.ExtendedAccount) error {
	fmt.Fprint(os.Stderr, "TID (12 digits): ")
	tid, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read TID: %w", err)
	}
	acc.TID = strings.TrimSpace(tid)

	fmt.Fprint(os.Stderr, "BenID (WebService user): ")
	benid, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read BenID: %w", err)
	}
	acc.BenID = strings.TrimSpace(benid)

	pin, err := promptPassword("PIN: ")
	if err != nil {
		return fmt.Errorf("failed to read PIN: %w", err)
	}
	acc.PIN = pin

	return nil
}

func promptELDACredentials(reader *bufio.Reader, acc *store.ExtendedAccount) error {
	fmt.Fprint(os.Stderr, "Dienstgebernummer (8 digits): ")
	dgNr, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read Dienstgebernummer: %w", err)
	}
	acc.DienstgeberNr = strings.TrimSpace(dgNr)

	fmt.Fprint(os.Stderr, "ELDA Benutzer: ")
	benutzer, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read ELDA Benutzer: %w", err)
	}
	acc.ELDABenutzer = strings.TrimSpace(benutzer)

	pin, err := promptPassword("ELDA PIN: ")
	if err != nil {
		return fmt.Errorf("failed to read ELDA PIN: %w", err)
	}
	acc.ELDAPIN = pin

	return nil
}

func promptFirmenbuchCredentials(reader *bufio.Reader, acc *store.ExtendedAccount) error {
	apiKey, err := promptPassword("API Key: ")
	if err != nil {
		return fmt.Errorf("failed to read API Key: %w", err)
	}
	acc.APIKey = apiKey

	return nil
}

func runAccountList(cmd *cobra.Command, args []string) error {
	LogVerbose("Listing accounts")

	// Get config directory
	cfgDir, err := config.GetConfigDir(GetConfigDir())
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	credPath := config.GetCredentialPath(cfgDir)

	// Check if credential file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		if IsJSONOutput() {
			return outputJSON(map[string]interface{}{
				"accounts": []string{},
			})
		}
		fmt.Println("No accounts stored.")
		return nil
	}

	// Prompt for master password
	masterPassword, err := promptPassword("Master password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Load credential store
	credStore, err := store.Load(credPath, masterPassword)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	accounts := credStore.ListAccounts()

	if IsJSONOutput() {
		// For JSON output, include more details
		type accountInfo struct {
			Name  string `json:"name"`
			TID   string `json:"tid"`
			BenID string `json:"benid"`
		}
		infos := make([]accountInfo, 0, len(accounts))
		for _, name := range accounts {
			acc, _ := credStore.GetAccount(name)
			if acc != nil {
				infos = append(infos, accountInfo{
					Name:  acc.Name,
					TID:   acc.TID,
					BenID: acc.BenID,
				})
			}
		}
		return outputJSON(map[string]interface{}{
			"accounts": infos,
		})
	}

	// Table output
	if len(accounts) == 0 {
		fmt.Println("No accounts stored.")
		return nil
	}

	fmt.Println("NAME                          TID            BENID")
	fmt.Println("-----------------------------------------------------------")
	for _, name := range accounts {
		acc, _ := credStore.GetAccount(name)
		if acc != nil {
			fmt.Printf("%-30s%-15s%s\n", acc.Name, acc.TID, acc.BenID)
		}
	}
	return nil
}

func runAccountRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	LogVerbose("Removing account: %s", name)

	// Get config directory
	cfgDir, err := config.GetConfigDir(GetConfigDir())
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	credPath := config.GetCredentialPath(cfgDir)

	// Check if credential file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		return fmt.Errorf("no accounts stored")
	}

	// Prompt for master password
	masterPassword, err := promptPassword("Master password: ")
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	// Load credential store
	credStore, err := store.Load(credPath, masterPassword)
	if err != nil {
		return fmt.Errorf("failed to load credentials: %w", err)
	}

	// Remove account
	if err := credStore.RemoveAccount(name); err != nil {
		return fmt.Errorf("failed to remove account: %w", err)
	}

	// Save
	if err := credStore.Save(credPath, masterPassword); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"status":  "success",
			"action":  "remove",
			"account": name,
		})
	}
	fmt.Printf("Account \"%s\" removed.\n", name)
	return nil
}
