package cli

import (
	"fmt"
	"os"

	"github.com/austrian-business-infrastructure/fo/internal/erechnung"
	"github.com/spf13/cobra"
)

var erechnungCmd = &cobra.Command{
	Use:     "erechnung",
	Aliases: []string{"invoice", "einvoice"},
	Short:   "Electronic invoice creation and validation",
	Long:    `Commands for creating and validating EN16931-compliant electronic invoices (XRechnung/ZUGFeRD).`,
}

var erechnungCreateCmd = &cobra.Command{
	Use:   "create <file.json>",
	Short: "Create an electronic invoice from JSON input",
	Long: `Create an electronic invoice in XRechnung (UBL) or ZUGFeRD (CII) format from a JSON file.

Example JSON input:
{
  "id": "INV-2025-001",
  "invoice_type": "380",
  "issue_date": "2025-01-15",
  "currency": "EUR",
  "seller": {
    "name": "Seller GmbH",
    "country": "AT",
    "vat_number": "ATU12345678"
  },
  "buyer": {
    "name": "Buyer AG",
    "country": "DE"
  },
  "lines": [
    {
      "id": "1",
      "description": "Product A",
      "quantity": 5,
      "unit_code": "C62",
      "unit_price": 10000,
      "tax_percent": 20,
      "tax_category": "S"
    }
  ]
}`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]
		format, _ := cmd.Flags().GetString("format")
		outputFile, _ := cmd.Flags().GetString("output")
		validate, _ := cmd.Flags().GetBool("validate")

		// Read input file
		jsonData, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %v", err)
		}

		// Parse JSON
		invoice, err := erechnung.ParseInvoiceJSON(jsonData)
		if err != nil {
			return fmt.Errorf("failed to parse JSON: %v", err)
		}

		// Validate if requested
		if validate {
			result := erechnung.ValidateEN16931(invoice)
			if !result.Valid {
				cmd.Println("Validation errors:")
				for _, e := range result.Errors {
					cmd.Printf("  [%s] %s\n", e.Code, e.Message)
				}
				return fmt.Errorf("invoice validation failed")
			}
		}

		// Calculate totals
		if err := invoice.CalculateTotals(); err != nil {
			return fmt.Errorf("failed to calculate totals: %v", err)
		}

		// Generate XML
		var xmlData []byte
		switch format {
		case "xrechnung", "ubl":
			xmlData, err = erechnung.GenerateXRechnung(invoice)
		case "zugferd", "cii":
			xmlData, err = erechnung.GenerateZUGFeRD(invoice)
		default:
			return fmt.Errorf("unknown format: %s (use xrechnung or zugferd)", format)
		}

		if err != nil {
			return fmt.Errorf("failed to generate XML: %v", err)
		}

		// Output
		if outputFile != "" {
			if err := os.WriteFile(outputFile, xmlData, 0644); err != nil {
				return fmt.Errorf("failed to write output file: %v", err)
			}
			cmd.Printf("Invoice written to %s\n", outputFile)
		} else {
			cmd.Println(string(xmlData))
		}

		return nil
	},
}

var erechnungValidateCmd = &cobra.Command{
	Use:   "validate <file.json>",
	Short: "Validate an invoice against EN16931 rules",
	Long:  `Validate an invoice JSON file against EN16931 business rules.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		// Read input file
		jsonData, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %v", err)
		}

		// Parse JSON
		invoice, err := erechnung.ParseInvoiceJSON(jsonData)
		if err != nil {
			return fmt.Errorf("failed to parse JSON: %v", err)
		}

		// Validate
		result := erechnung.ValidateEN16931(invoice)

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(result)
		}

		if result.Valid {
			cmd.Println("Invoice is valid")
			return nil
		}

		cmd.Println("Validation errors:")
		for _, e := range result.Errors {
			cmd.Printf("  [%s] %s", e.Code, e.Message)
			if e.Field != "" {
				cmd.Printf(" (field: %s)", e.Field)
			}
			cmd.Println()
		}

		if len(result.Warnings) > 0 {
			cmd.Println("\nWarnings:")
			for _, w := range result.Warnings {
				cmd.Printf("  [%s] %s\n", w.Code, w.Message)
			}
		}

		return fmt.Errorf("invoice validation failed with %d errors", len(result.Errors))
	},
}

var erechnungCalcCmd = &cobra.Command{
	Use:   "calc <file.json>",
	Short: "Calculate invoice totals",
	Long:  `Calculate and display invoice totals from a JSON file.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputFile := args[0]

		// Read input file
		jsonData, err := os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("failed to read input file: %v", err)
		}

		// Parse JSON
		invoice, err := erechnung.ParseInvoiceJSON(jsonData)
		if err != nil {
			return fmt.Errorf("failed to parse JSON: %v", err)
		}

		// Calculate totals
		if err := invoice.CalculateTotals(); err != nil {
			return fmt.Errorf("failed to calculate totals: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]interface{}{
				"invoice_id":          invoice.ID,
				"currency":            invoice.Currency,
				"tax_exclusive_amount": erechnung.AmountEUR(invoice.TaxExclusiveAmount),
				"tax_amount":          erechnung.AmountEUR(invoice.TaxAmount),
				"tax_inclusive_amount": erechnung.AmountEUR(invoice.TaxInclusiveAmount),
				"payable_amount":      erechnung.AmountEUR(invoice.PayableAmount),
				"tax_subtotals":       invoice.TaxSubtotals,
			})
		}

		cmd.Printf("Invoice: %s\n", invoice.ID)
		cmd.Printf("Currency: %s\n\n", invoice.Currency)

		cmd.Println("Line totals:")
		for _, line := range invoice.Lines {
			cmd.Printf("  %s: %.2f %s\n", line.ID, erechnung.AmountEUR(line.LineTotal), invoice.Currency)
		}

		cmd.Println("\nTax breakdown:")
		for _, ts := range invoice.TaxSubtotals {
			cmd.Printf("  %s (%.0f%%): %.2f on %.2f = %.2f %s\n",
				ts.TaxCategory,
				ts.TaxPercent,
				ts.TaxPercent,
				erechnung.AmountEUR(ts.TaxableAmount),
				erechnung.AmountEUR(ts.TaxAmount),
				invoice.Currency)
		}

		cmd.Println("\nTotals:")
		cmd.Printf("  Net amount:   %.2f %s\n", erechnung.AmountEUR(invoice.TaxExclusiveAmount), invoice.Currency)
		cmd.Printf("  Tax amount:   %.2f %s\n", erechnung.AmountEUR(invoice.TaxAmount), invoice.Currency)
		cmd.Printf("  Gross amount: %.2f %s\n", erechnung.AmountEUR(invoice.TaxInclusiveAmount), invoice.Currency)
		cmd.Printf("  Amount due:   %.2f %s\n", erechnung.AmountEUR(invoice.PayableAmount), invoice.Currency)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(erechnungCmd)
	erechnungCmd.AddCommand(erechnungCreateCmd)
	erechnungCmd.AddCommand(erechnungValidateCmd)
	erechnungCmd.AddCommand(erechnungCalcCmd)

	// Create flags
	erechnungCreateCmd.Flags().StringP("format", "f", "xrechnung", "Output format (xrechnung/zugferd)")
	erechnungCreateCmd.Flags().StringP("output", "o", "", "Output file (default: stdout)")
	erechnungCreateCmd.Flags().Bool("validate", true, "Validate invoice before generating")

	// Validate flags
	erechnungValidateCmd.Flags().Bool("json", false, "Output in JSON format")

	// Calc flags
	erechnungCalcCmd.Flags().Bool("json", false, "Output in JSON format")
}
