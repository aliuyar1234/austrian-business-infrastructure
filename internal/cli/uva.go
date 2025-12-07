package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/austrian-business-infrastructure/fo/internal/fonws"
	"github.com/spf13/cobra"
)

var (
	// UVA command flags
	uvaFile    string
	uvaYear    int
	uvaMonth   int
	uvaQuarter int
	uvaKZ017   int64
	uvaKZ060   int64
	uvaKZ095   int64
	uvaAll     bool
)

var uvaCmd = &cobra.Command{
	Use:   "uva",
	Short: "VAT advance return (Umsatzsteuervoranmeldung) operations",
	Long: `Manage VAT advance returns (Umsatzsteuervoranmeldung) for FinanzOnline.

Commands:
  validate  - Validate a UVA XML file or values
  submit    - Submit a UVA to FinanzOnline
  status    - Check status of a submitted UVA`,
}

var uvaValidateCmd = &cobra.Command{
	Use:   "validate [file.xml]",
	Short: "Validate a UVA XML file or values",
	Long: `Validate a VAT advance return before submission.

Examples:
  fo uva validate uva.xml
  fo uva validate --year 2025 --month 1 --kz017 80000 --kz060 16000`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUVAValidate,
}

var uvaSubmitCmd = &cobra.Command{
	Use:   "submit <account>",
	Short: "Submit a UVA to FinanzOnline",
	Long: `Submit a VAT advance return to FinanzOnline.

Examples:
  fo uva submit firma-xyz --file uva.xml
  fo uva submit firma-xyz --year 2025 --month 1 --kz017 80000 --kz060 16000
  fo uva submit --all --file uva.xml  # Submit to all accounts`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUVASubmit,
}

var uvaStatusCmd = &cobra.Command{
	Use:   "status <reference>",
	Short: "Check status of a submitted UVA",
	Long: `Check the status of a previously submitted UVA.

Example:
  fo uva status FON-2025-12345678 --account firma-xyz`,
	Args: cobra.ExactArgs(1),
	RunE: runUVAStatus,
}

func init() {
	// Validate flags
	uvaValidateCmd.Flags().StringVarP(&uvaFile, "file", "f", "", "UVA XML file path")
	uvaValidateCmd.Flags().IntVarP(&uvaYear, "year", "y", 0, "Tax year")
	uvaValidateCmd.Flags().IntVarP(&uvaMonth, "month", "m", 0, "Tax month (1-12)")
	uvaValidateCmd.Flags().IntVarP(&uvaQuarter, "quarter", "q", 0, "Tax quarter (1-4)")
	uvaValidateCmd.Flags().Int64Var(&uvaKZ017, "kz017", 0, "KZ017 - 20% VAT taxable amount (cents)")
	uvaValidateCmd.Flags().Int64Var(&uvaKZ060, "kz060", 0, "KZ060 - Input tax (cents)")

	// Submit flags
	uvaSubmitCmd.Flags().StringVarP(&uvaFile, "file", "f", "", "UVA XML file path")
	uvaSubmitCmd.Flags().IntVarP(&uvaYear, "year", "y", 0, "Tax year")
	uvaSubmitCmd.Flags().IntVarP(&uvaMonth, "month", "m", 0, "Tax month (1-12)")
	uvaSubmitCmd.Flags().IntVarP(&uvaQuarter, "quarter", "q", 0, "Tax quarter (1-4)")
	uvaSubmitCmd.Flags().Int64Var(&uvaKZ017, "kz017", 0, "KZ017 - 20% VAT taxable amount (cents)")
	uvaSubmitCmd.Flags().Int64Var(&uvaKZ060, "kz060", 0, "KZ060 - Input tax (cents)")
	uvaSubmitCmd.Flags().Int64Var(&uvaKZ095, "kz095", 0, "KZ095 - Tax liability/credit (cents)")
	uvaSubmitCmd.Flags().BoolVar(&uvaAll, "all", false, "Submit to all accounts")

	uvaCmd.AddCommand(uvaValidateCmd)
	uvaCmd.AddCommand(uvaSubmitCmd)
	uvaCmd.AddCommand(uvaStatusCmd)

	rootCmd.AddCommand(uvaCmd)
}

func runUVAValidate(cmd *cobra.Command, args []string) error {
	var uva *fonws.UVA
	var err error

	// Load from file or build from flags
	if len(args) > 0 {
		uva, err = loadUVAFromFile(args[0])
	} else if uvaFile != "" {
		uva, err = loadUVAFromFile(uvaFile)
	} else if uvaYear > 0 {
		uva, err = buildUVAFromFlags()
	} else {
		return fmt.Errorf("provide a file path or --year with other flags")
	}

	if err != nil {
		return fmt.Errorf("failed to load UVA: %w", err)
	}

	// Validate
	if err := fonws.ValidateUVA(uva); err != nil {
		if IsJSONOutput() {
			return outputJSON(map[string]interface{}{
				"valid":   false,
				"error":   err.Error(),
			})
		}
		fmt.Fprintf(os.Stderr, "❌ Validation failed: %s\n", err)
		return err
	}

	// Success
	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"valid":   true,
			"year":    uva.Year,
			"period":  uva.Period,
			"kz017":   uva.KZ017,
			"kz060":   uva.KZ060,
			"kz095":   uva.KZ095,
		})
	}

	fmt.Println("✅ UVA is valid")
	printUVASummary(uva)
	return nil
}

func runUVASubmit(cmd *cobra.Command, args []string) error {
	// Require account unless --all
	if !uvaAll && len(args) == 0 {
		return fmt.Errorf("provide an account name or use --all")
	}

	var uva *fonws.UVA
	var err error

	// Load UVA data
	if uvaFile != "" {
		uva, err = loadUVAFromFile(uvaFile)
	} else if uvaYear > 0 {
		uva, err = buildUVAFromFlags()
	} else {
		return fmt.Errorf("provide --file or --year with other flags")
	}

	if err != nil {
		return fmt.Errorf("failed to load UVA: %w", err)
	}

	// Validate before submission
	if err := fonws.ValidateUVA(uva); err != nil {
		return fmt.Errorf("UVA validation failed: %w", err)
	}

	// For now, show what would be submitted
	// Full implementation would integrate with session service
	fmt.Println("📋 UVA ready for submission")
	printUVASummary(uva)

	if len(args) > 0 {
		fmt.Printf("\n📤 Would submit to account: %s\n", args[0])
	}

	// TODO: Implement actual submission with:
	// 1. Load account credentials
	// 2. Create session (login)
	// 3. Submit UVA via FileUploadService
	// 4. Handle response and logout

	return nil
}

func runUVAStatus(cmd *cobra.Command, args []string) error {
	reference := args[0]

	// TODO: Implement status check via FinanzOnline API
	// For now, return placeholder

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"reference": reference,
			"status":    "unknown",
			"message":   "Status check not yet implemented",
		})
	}

	fmt.Printf("📋 Reference: %s\n", reference)
	fmt.Println("⚠️  Status check not yet implemented")
	return nil
}

func loadUVAFromFile(path string) (*fonws.UVA, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return fonws.ParseUVAFromXML(data)
}

func buildUVAFromFlags() (*fonws.UVA, error) {
	uva := &fonws.UVA{
		Year:   uvaYear,
		KZ017:  uvaKZ017,
		KZ060:  uvaKZ060,
		KZ095:  uvaKZ095,
		Status: fonws.UVAStatusDraft,
	}

	// Set period
	if uvaMonth > 0 {
		uva.Period = fonws.UVAPeriod{Type: fonws.PeriodTypeMonthly, Value: uvaMonth}
	} else if uvaQuarter > 0 {
		uva.Period = fonws.UVAPeriod{Type: fonws.PeriodTypeQuarterly, Value: uvaQuarter}
	} else {
		return nil, fmt.Errorf("provide --month or --quarter")
	}

	// Calculate KZ095 if not provided
	if uva.KZ095 == 0 {
		uva.KZ095 = uva.CalculateKZ095()
	}

	return uva, nil
}

func printUVASummary(uva *fonws.UVA) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "\nUVA Summary")
	fmt.Fprintln(w, "═══════════════════════════════════")

	fmt.Fprintf(w, "Year:\t%d\n", uva.Year)
	if uva.Period.Type == fonws.PeriodTypeMonthly {
		fmt.Fprintf(w, "Month:\t%d\n", uva.Period.Value)
	} else {
		fmt.Fprintf(w, "Quarter:\tQ%d\n", uva.Period.Value)
	}

	fmt.Fprintln(w, "\nKennzahlen:")
	if uva.KZ000 > 0 {
		fmt.Fprintf(w, "  KZ000 (Gesamtbetrag):\t%.2f EUR\n", float64(uva.KZ000)/100)
	}
	if uva.KZ017 > 0 {
		fmt.Fprintf(w, "  KZ017 (20%% Umsätze):\t%.2f EUR\n", float64(uva.KZ017)/100)
	}
	if uva.KZ018 > 0 {
		fmt.Fprintf(w, "  KZ018 (10%% Umsätze):\t%.2f EUR\n", float64(uva.KZ018)/100)
	}
	if uva.KZ019 > 0 {
		fmt.Fprintf(w, "  KZ019 (13%% Umsätze):\t%.2f EUR\n", float64(uva.KZ019)/100)
	}
	if uva.KZ060 > 0 {
		fmt.Fprintf(w, "  KZ060 (Vorsteuer):\t%.2f EUR\n", float64(uva.KZ060)/100)
	}

	fmt.Fprintf(w, "\nKZ095 (Zahllast):\t%.2f EUR\n", float64(uva.KZ095)/100)
	if uva.KZ095 > 0 {
		fmt.Fprintln(w, "  → Zahlung an Finanzamt")
	} else if uva.KZ095 < 0 {
		fmt.Fprintln(w, "  → Gutschrift vom Finanzamt")
	}
}

// outputJSON is defined in session.go
