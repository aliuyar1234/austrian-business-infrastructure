package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"austrian-business-infrastructure/internal/fonws"
	"github.com/spf13/cobra"
)

var (
	zmPeriod string
	zmYear   int
	zmQuarter int
	zmOutput string
)

var zmCmd = &cobra.Command{
	Use:   "zm",
	Short: "Zusammenfassende Meldung (ZM) operations",
	Long: `Submit and manage recapitulative statements (Zusammenfassende Meldung).

ZM is required for EU intra-community trade reporting.

Commands:
  generate  - Generate ZM from CSV data
  submit    - Submit ZM to FinanzOnline
  validate  - Validate ZM entries`,
}

var zmGenerateCmd = &cobra.Command{
	Use:   "generate <file.csv>",
	Short: "Generate ZM XML from CSV data",
	Long: `Generate ZM XML from a CSV file containing intra-community transactions.

The CSV file must have the following columns:
  partner_uid,country_code,delivery_type,amount

Delivery types:
  L - Lieferungen (goods)
  D - Dreiecksgeschäfte (triangular transactions)
  S - Sonstige Leistungen (services)

Amount should be in cents.

Examples:
  fo zm generate transactions.csv --period Q1-2025
  fo zm generate data.csv --year 2025 --quarter 1 --output zm.xml`,
	Args: cobra.ExactArgs(1),
	RunE: runZMGenerate,
}

var zmSubmitCmd = &cobra.Command{
	Use:   "submit <file.xml>",
	Short: "Submit ZM to FinanzOnline",
	Long: `Submit a ZM XML file to FinanzOnline.

Requires an active FinanzOnline session. Use 'fo session login' first.

Examples:
  fo zm submit zm_q1_2025.xml
  fo zm submit zm.xml --json`,
	Args: cobra.ExactArgs(1),
	RunE: runZMSubmit,
}

var zmValidateCmd = &cobra.Command{
	Use:   "validate <file.csv>",
	Short: "Validate ZM entries from CSV",
	Long: `Validate ZM entries from a CSV file without submitting.

Checks partner UIDs, country codes, and amounts.

Examples:
  fo zm validate transactions.csv
  fo zm validate data.csv --json`,
	Args: cobra.ExactArgs(1),
	RunE: runZMValidate,
}

func init() {
	// Generate flags
	zmGenerateCmd.Flags().StringVarP(&zmPeriod, "period", "p", "", "Period in format Q1-2025")
	zmGenerateCmd.Flags().IntVar(&zmYear, "year", 0, "Year (e.g., 2025)")
	zmGenerateCmd.Flags().IntVar(&zmQuarter, "quarter", 0, "Quarter (1-4)")
	zmGenerateCmd.Flags().StringVarP(&zmOutput, "output", "o", "", "Output XML file")

	zmCmd.AddCommand(zmGenerateCmd)
	zmCmd.AddCommand(zmSubmitCmd)
	zmCmd.AddCommand(zmValidateCmd)

	rootCmd.AddCommand(zmCmd)
}

func parsePeriod(period string) (int, int, error) {
	// Format: Q1-2025 or Q1/2025
	period = strings.ToUpper(strings.TrimSpace(period))
	period = strings.ReplaceAll(period, "/", "-")

	if !strings.HasPrefix(period, "Q") {
		return 0, 0, fmt.Errorf("invalid period format, expected Q1-2025")
	}

	parts := strings.Split(period[1:], "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid period format, expected Q1-2025")
	}

	quarter, err := strconv.Atoi(parts[0])
	if err != nil || quarter < 1 || quarter > 4 {
		return 0, 0, fmt.Errorf("invalid quarter: %s", parts[0])
	}

	year, err := strconv.Atoi(parts[1])
	if err != nil || year < 2000 || year > 2100 {
		return 0, 0, fmt.Errorf("invalid year: %s", parts[1])
	}

	return year, quarter, nil
}

func runZMGenerate(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Parse year and quarter
	var year, quarter int
	var err error

	if zmPeriod != "" {
		year, quarter, err = parsePeriod(zmPeriod)
		if err != nil {
			return err
		}
	} else if zmYear != 0 && zmQuarter != 0 {
		year = zmYear
		quarter = zmQuarter
	} else {
		return fmt.Errorf("must specify --period or both --year and --quarter")
	}

	// Read CSV file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse entries
	entries, err := fonws.ParseZMFromCSV(data)
	if err != nil {
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No entries found in CSV file")
		return nil
	}

	// Create ZM
	zm := fonws.NewZM(year, quarter)
	zm.Entries = entries

	// Generate XML
	xmlData, err := fonws.GenerateZMXML(zm)
	if err != nil {
		return fmt.Errorf("failed to generate XML: %w", err)
	}

	// Write output
	if zmOutput != "" {
		if err := os.WriteFile(zmOutput, xmlData, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("ZM XML written to %s\n", zmOutput)
	} else if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"period":       zm.PeriodString(),
			"entries":      len(entries),
			"total_amount": zm.TotalAmountEUR(),
			"xml":          string(xmlData),
		})
	} else {
		// Print to stdout
		fmt.Println(string(xmlData))
	}

	if !IsJSONOutput() && zmOutput != "" {
		fmt.Printf("\nZM Summary:\n")
		fmt.Printf("  Period: %s\n", zm.PeriodString())
		fmt.Printf("  Entries: %d\n", len(entries))
		fmt.Printf("  Total: %.2f EUR\n", zm.TotalAmountEUR())
	}

	return nil
}

func runZMSubmit(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Read XML file
	_, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// For now, show a message that session is required
	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"status":  "pending",
			"message": "ZM submission requires active FinanzOnline session. Use 'fo session login' first.",
			"file":    filePath,
		})
	}

	fmt.Println("ZM Submission")
	fmt.Println("─────────────")
	fmt.Printf("File: %s\n", filePath)
	fmt.Println()
	fmt.Println("⚠️  ZM submission requires an active FinanzOnline session.")
	fmt.Println("   Use 'fo session login <account>' first.")
	fmt.Println()
	fmt.Println("After logging in, the ZM will be submitted to FinanzOnline")
	fmt.Println("and you will receive a confirmation with a reference number.")

	return nil
}

func runZMValidate(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	// Read CSV file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Parse entries
	entries, err := fonws.ParseZMFromCSV(data)
	if err != nil {
		if IsJSONOutput() {
			return outputJSON(map[string]interface{}{
				"valid":   false,
				"error":   err.Error(),
				"entries": 0,
			})
		}
		return fmt.Errorf("validation failed: %w", err)
	}

	// Validate each entry
	var validCount, invalidCount int
	type entryResult struct {
		PartnerUID   string  `json:"partner_uid"`
		CountryCode  string  `json:"country_code"`
		DeliveryType string  `json:"delivery_type"`
		Amount       float64 `json:"amount"`
		Valid        bool    `json:"valid"`
		Error        string  `json:"error,omitempty"`
	}

	results := make([]entryResult, 0, len(entries))
	for _, entry := range entries {
		result := entryResult{
			PartnerUID:   entry.PartnerUID,
			CountryCode:  entry.CountryCode,
			DeliveryType: string(entry.DeliveryType),
			Amount:       entry.AmountEUR(),
			Valid:        true,
		}
		if err := entry.Validate(); err != nil {
			result.Valid = false
			result.Error = err.Error()
			invalidCount++
		} else {
			validCount++
		}
		results = append(results, result)
	}

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"valid":         invalidCount == 0,
			"total_entries": len(entries),
			"valid_count":   validCount,
			"invalid_count": invalidCount,
			"entries":       results,
		})
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "Partner UID\tCountry\tType\tAmount (EUR)\tStatus\tError")
	fmt.Fprintln(w, "───────────\t───────\t────\t────────────\t──────\t─────")

	for _, r := range results {
		status := "✅"
		if !r.Valid {
			status = "❌"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%s\t%s\n",
			r.PartnerUID, r.CountryCode, r.DeliveryType, r.Amount, status, r.Error)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Summary: %d valid, %d invalid out of %d entries\n",
		validCount, invalidCount, len(entries))

	if invalidCount > 0 {
		return fmt.Errorf("validation failed: %d invalid entries", invalidCount)
	}

	return nil
}

// ZMService provides ZM-related CLI operations
type ZMService struct {
	client *fonws.Client
}

// NewZMService creates a new ZM service
func NewZMService(client *fonws.Client) *ZMService {
	return &ZMService{client: client}
}

// FormatZMEntry formats a ZM entry for display
func FormatZMEntry(entry *fonws.ZMEntry) string {
	deliveryTypeNames := map[fonws.ZMDeliveryType]string{
		fonws.ZMDeliveryTypeGoods:      "Goods",
		fonws.ZMDeliveryTypeTriangular: "Triangular",
		fonws.ZMDeliveryTypeServices:   "Services",
	}
	typeName := deliveryTypeNames[entry.DeliveryType]
	if typeName == "" {
		typeName = string(entry.DeliveryType)
	}
	return fmt.Sprintf("%s (%s): %.2f EUR [%s]",
		entry.PartnerUID, entry.CountryCode, entry.AmountEUR(), typeName)
}

// PrintZMSummary prints a summary of a ZM
func PrintZMSummary(zm *fonws.ZM) {
	fmt.Printf("ZM Summary: %s\n", zm.PeriodString())
	fmt.Printf("─────────────────────\n")
	fmt.Printf("Year:     %d\n", zm.Year)
	fmt.Printf("Quarter:  Q%d\n", zm.Quarter)
	fmt.Printf("Entries:  %d\n", len(zm.Entries))
	fmt.Printf("Total:    %.2f EUR\n", zm.TotalAmountEUR())
	fmt.Printf("Status:   %s\n", zm.Status)
	if zm.Reference != "" {
		fmt.Printf("Reference: %s\n", zm.Reference)
	}
	if zm.SubmittedAt != nil {
		fmt.Printf("Submitted: %s\n", zm.SubmittedAt.Format(time.RFC3339))
	}
}

// OutputZMAsJSON outputs a ZM as JSON
func OutputZMAsJSON(zm *fonws.ZM) error {
	data, err := json.MarshalIndent(zm, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
