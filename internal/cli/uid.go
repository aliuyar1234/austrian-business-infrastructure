package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"austrian-business-infrastructure/internal/fonws"
	"github.com/spf13/cobra"
)

var (
	// UID command flags
	uidOutputFile string
	uidSource     string
)

var uidCmd = &cobra.Command{
	Use:   "uid",
	Short: "UID (VAT ID) validation operations",
	Long: `Validate EU VAT identification numbers (UID/USt-IdNr).

Commands:
  check  - Validate a single UID number
  batch  - Validate multiple UIDs from a CSV file`,
}

var uidCheckCmd = &cobra.Command{
	Use:   "check <uid>",
	Short: "Validate a single UID number",
	Long: `Validate a VAT identification number and display company information.

Examples:
  fo uid check ATU12345678
  fo uid check DE123456789 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runUIDCheck,
}

var uidBatchCmd = &cobra.Command{
	Use:   "batch <file.csv>",
	Short: "Validate multiple UIDs from a CSV file",
	Long: `Validate multiple VAT identification numbers from a CSV file.

The CSV file must have a 'uid' column. Additional columns are ignored.

Examples:
  fo uid batch uids.csv
  fo uid batch uids.csv --output results.csv`,
	Args: cobra.ExactArgs(1),
	RunE: runUIDBatch,
}

func init() {
	// Check flags
	uidCheckCmd.Flags().StringVar(&uidSource, "source", "finanzonline", "Validation source (finanzonline, vies)")

	// Batch flags
	uidBatchCmd.Flags().StringVarP(&uidOutputFile, "output", "o", "", "Output CSV file for results")
	uidBatchCmd.Flags().StringVar(&uidSource, "source", "finanzonline", "Validation source (finanzonline, vies)")

	uidCmd.AddCommand(uidCheckCmd)
	uidCmd.AddCommand(uidBatchCmd)

	rootCmd.AddCommand(uidCmd)
}

func runUIDCheck(cmd *cobra.Command, args []string) error {
	uid := args[0]

	// First validate format locally
	formatResult := fonws.ValidateUIDFormat(uid)
	if !formatResult.Valid {
		if IsJSONOutput() {
			return outputJSON(map[string]interface{}{
				"uid":   uid,
				"valid": false,
				"error": formatResult.Error,
			})
		}
		fmt.Fprintf(os.Stderr, "❌ Invalid UID format: %s\n", formatResult.Error)
		return fmt.Errorf("invalid UID format")
	}

	// For now, just show format validation result
	// Full implementation would connect to FinanzOnline or VIES
	result := &fonws.UIDValidationResult{
		UID:         uid,
		CountryCode: formatResult.CountryCode,
	}

	// Note: Full implementation would:
	// 1. Load account credentials
	// 2. Create session (login)
	// 3. Call UIDService.Validate()
	// 4. Handle response and logout

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"uid":          result.UID,
			"country_code": result.CountryCode,
			"format_valid": true,
			"message":      "Format is valid. Full validation requires FinanzOnline session.",
		})
	}

	fmt.Printf("✅ UID format is valid\n")
	fmt.Printf("   UID: %s\n", result.UID)
	fmt.Printf("   Country: %s\n", result.CountryCode)
	fmt.Println("\n⚠️  Full validation requires FinanzOnline session.")
	fmt.Println("   Use `fo session login <account>` first.")

	return nil
}

func runUIDBatch(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Read CSV file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse UIDs from CSV
	uids, err := fonws.ParseUIDCSV(data)
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(uids) == 0 {
		fmt.Println("No UIDs found in CSV file")
		return nil
	}

	fmt.Printf("Found %d UIDs in CSV\n\n", len(uids))

	// Validate formats locally
	var results []*fonws.UIDValidationResult
	for _, uid := range uids {
		formatResult := fonws.ValidateUIDFormat(uid)
		results = append(results, &fonws.UIDValidationResult{
			UID:          uid,
			Valid:        formatResult.Valid,
			CountryCode:  formatResult.CountryCode,
			ErrorMessage: formatResult.Error,
		})
	}

	// Output results
	if uidOutputFile != "" {
		csvData, err := fonws.WriteUIDResultsCSV(results)
		if err != nil {
			return fmt.Errorf("failed to generate CSV: %w", err)
		}
		if err := os.WriteFile(uidOutputFile, csvData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Results written to %s\n", uidOutputFile)
	}

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"total":   len(results),
			"results": results,
		})
	}

	// Print summary table
	printUIDResults(results)

	return nil
}

func printUIDResults(results []*fonws.UIDValidationResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "UID\tFormat Valid\tCountry\tCompany\tError")
	fmt.Fprintln(w, "───\t────────────\t───────\t───────\t─────")

	validCount := 0
	for _, r := range results {
		status := "❌"
		if r.Valid {
			status = "✅"
			validCount++
		}

		errMsg := r.ErrorMessage
		if len(errMsg) > 30 {
			errMsg = errMsg[:27] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			r.UID,
			status,
			r.CountryCode,
			r.CompanyName,
			errMsg,
		)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Summary: %d/%d valid format\n", validCount, len(results))
	fmt.Fprintln(w, "\n⚠️  Full validation requires FinanzOnline session.")
}
