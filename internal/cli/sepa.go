package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/sepa"
	"github.com/spf13/cobra"
)

var sepaCmd = &cobra.Command{
	Use:   "sepa",
	Short: "SEPA payment file generation and parsing",
	Long:  `Commands for generating and parsing SEPA payment files (pain.001, pain.008, camt.053).`,
}

var sepaPain001Cmd = &cobra.Command{
	Use:   "pain001 <file.csv>",
	Short: "Generate pain.001 credit transfer file from CSV",
	Long: `Generate a SEPA pain.001 credit transfer XML file from a CSV input.

Expected CSV columns:
  - creditor_name: Name of the creditor
  - creditor_iban: IBAN of the creditor
  - amount: Amount in EUR (e.g., 1500.00)
  - currency: Currency code (optional, default: EUR)
  - reference: Payment reference (optional)
  - creditor_bic: BIC of the creditor (optional)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]
		outputFile, _ := cmd.Flags().GetString("output")
		messageID, _ := cmd.Flags().GetString("message-id")
		debtorName, _ := cmd.Flags().GetString("debtor-name")
		debtorIBAN, _ := cmd.Flags().GetString("debtor-iban")
		debtorBIC, _ := cmd.Flags().GetString("debtor-bic")

		// Read CSV
		csvData, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %v", err)
		}

		// Parse CSV
		txns, err := sepa.ParseCreditTransferCSV(csvData)
		if err != nil {
			return fmt.Errorf("failed to parse CSV: %v", err)
		}

		if len(txns) == 0 {
			return fmt.Errorf("no transactions found in CSV")
		}

		// Build credit transfer
		ct := sepa.NewCreditTransfer(messageID, debtorName)
		ct.Debtor = sepa.SEPAParty{Name: debtorName}
		ct.DebtorAccount = sepa.SEPAAccount{
			IBAN: debtorIBAN,
			BIC:  debtorBIC,
		}
		ct.Transactions = txns
		ct.CalculateTotals()

		// Generate XML
		xmlData, err := sepa.GeneratePain001(ct)
		if err != nil {
			return fmt.Errorf("failed to generate pain.001: %v", err)
		}

		// Output
		if outputFile != "" {
			if err := os.WriteFile(outputFile, xmlData, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %v", err)
			}
			cmd.Printf("pain.001 written to %s\n", outputFile)
			cmd.Printf("  Transactions: %d\n", ct.NumberOfTxs)
			cmd.Printf("  Total: %.2f EUR\n", ct.ControlSumEUR())
		} else {
			cmd.Println(string(xmlData))
		}

		return nil
	},
}

var sepaCamt053Cmd = &cobra.Command{
	Use:   "camt053 <file.xml>",
	Short: "Parse camt.053 bank statement file",
	Long:  `Parse and display contents of a SEPA camt.053 bank statement XML file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		// Read XML
		xmlData, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %v", err)
		}

		// Parse
		stmt, err := sepa.ParseCamt053(xmlData)
		if err != nil {
			return fmt.Errorf("failed to parse camt.053: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(stmt)
		}

		// Display
		cmd.Printf("Statement ID: %s\n", stmt.ID)
		cmd.Printf("Account: %s\n", stmt.Account.IBAN)
		cmd.Printf("Currency: %s\n", stmt.Account.Currency)
		cmd.Printf("Opening Balance: %.2f\n", float64(stmt.OpeningBalance)/100)
		cmd.Printf("Closing Balance: %.2f\n", float64(stmt.ClosingBalance)/100)
		cmd.Printf("\nEntries (%d):\n", len(stmt.Entries))

		for i, entry := range stmt.Entries {
			sign := "+"
			if entry.IsDebit() {
				sign = "-"
			}
			cmd.Printf("\n%d. %s %s%.2f %s\n", i+1, entry.BookingDate.Format("2006-01-02"), sign, entry.AmountEUR(), entry.Currency)
			if entry.RemittanceInfo != "" {
				cmd.Printf("   Reference: %s\n", entry.RemittanceInfo)
			}
			if entry.CounterpartyName != "" {
				cmd.Printf("   Counterparty: %s\n", entry.CounterpartyName)
			}
			if entry.CounterpartyIBAN != "" {
				cmd.Printf("   IBAN: %s\n", entry.CounterpartyIBAN)
			}
		}

		return nil
	},
}

var sepaValidateCmd = &cobra.Command{
	Use:   "validate <IBAN>",
	Short: "Validate an IBAN",
	Long:  `Validate an IBAN and display details including country, check digits, and BIC lookup (for Austrian IBANs).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		iban := args[0]

		result := sepa.ValidateIBANWithDetails(iban)

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(result)
		}

		if !result.Valid {
			return fmt.Errorf("invalid IBAN: %s", result.ErrorMessage)
		}

		cmd.Printf("IBAN: %s\n", sepa.FormatIBAN(result.IBAN))
		cmd.Printf("Valid: Yes\n")
		cmd.Printf("Country: %s\n", result.CountryCode)
		cmd.Printf("Bank Code: %s\n", result.BankCode)

		if result.BIC != "" {
			cmd.Printf("BIC: %s\n", result.BIC)
		}
		if result.BankName != "" {
			cmd.Printf("Bank: %s\n", result.BankName)
		}

		return nil
	},
}

var sepaPain008Cmd = &cobra.Command{
	Use:   "pain008",
	Short: "Generate pain.008 direct debit file",
	Long:  `Generate a SEPA pain.008 direct debit XML file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// This is a placeholder - in practice, direct debits would be created from
		// a database or CSV of mandates, not interactively
		return fmt.Errorf("pain.008 generation requires mandate data - use programmatic API")
	},
}

var sepaBicCmd = &cobra.Command{
	Use:   "bic <bank-code>",
	Short: "Look up BIC for Austrian bank code",
	Long:  `Look up the BIC (Bank Identifier Code) for an Austrian bank code.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		bankCode := args[0]

		bic, name := sepa.LookupAustrianBank(bankCode)
		if bic == "" {
			return fmt.Errorf("bank code %s not found", bankCode)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]string{
				"bank_code": bankCode,
				"bic":       bic,
				"name":      name,
			})
		}

		cmd.Printf("Bank Code: %s\n", bankCode)
		cmd.Printf("BIC: %s\n", bic)
		cmd.Printf("Name: %s\n", name)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sepaCmd)
	sepaCmd.AddCommand(sepaPain001Cmd)
	sepaCmd.AddCommand(sepaCamt053Cmd)
	sepaCmd.AddCommand(sepaValidateCmd)
	sepaCmd.AddCommand(sepaPain008Cmd)
	sepaCmd.AddCommand(sepaBicCmd)

	// pain001 flags
	sepaPain001Cmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	sepaPain001Cmd.Flags().String("message-id", fmt.Sprintf("CT-%s", time.Now().Format("20060102-150405")), "Message ID")
	sepaPain001Cmd.Flags().String("debtor-name", "", "Debtor name (required)")
	sepaPain001Cmd.Flags().String("debtor-iban", "", "Debtor IBAN (required)")
	sepaPain001Cmd.Flags().String("debtor-bic", "", "Debtor BIC")
	sepaPain001Cmd.MarkFlagRequired("debtor-name")
	sepaPain001Cmd.MarkFlagRequired("debtor-iban")

	// camt053 flags
	sepaCamt053Cmd.Flags().Bool("json", false, "Output in JSON format")

	// validate flags
	sepaValidateCmd.Flags().Bool("json", false, "Output in JSON format")

	// bic flags
	sepaBicCmd.Flags().Bool("json", false, "Output in JSON format")
}
