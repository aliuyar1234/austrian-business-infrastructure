package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/austrian-business-infrastructure/fo/internal/fb"
	"github.com/spf13/cobra"
)

var fbCmd = &cobra.Command{
	Use:   "fb",
	Short: "Firmenbuch (company register) queries",
	Long:  `Commands for querying the Austrian company register (Firmenbuch).`,
}

var fbSearchCmd = &cobra.Command{
	Use:   "search <name>",
	Short: "Search for companies by name",
	Long:  `Search the Austrian Firmenbuch for companies matching the given name.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ort, _ := cmd.Flags().GetString("ort")
		maxHits, _ := cmd.Flags().GetInt("max")
		testMode, _ := cmd.Flags().GetBool("test")
		apiKey, _ := cmd.Flags().GetString("api-key")

		if apiKey == "" {
			return fmt.Errorf("API key required (use --api-key or FB_API_KEY environment variable)")
		}

		client := fb.NewClient(apiKey, testMode)

		req := &fb.FBSearchRequest{
			Name:    name,
			Ort:     ort,
			MaxHits: maxHits,
		}

		resp, err := client.Search(req)
		if err != nil {
			return fmt.Errorf("search failed: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(resp)
		}

		if len(resp.Results) == 0 {
			cmd.Println("No companies found")
			return nil
		}

		cmd.Printf("Found %d companies:\n\n", resp.TotalCount)
		for _, r := range resp.Results {
			cmd.Printf("FN: %s\n", r.FN)
			cmd.Printf("  Firma: %s\n", r.Firma)
			cmd.Printf("  Rechtsform: %s\n", r.Rechtsform)
			cmd.Printf("  Sitz: %s\n", r.Sitz)
			cmd.Printf("  Status: %s\n\n", r.Status)
		}

		return nil
	},
}

var fbExtractCmd = &cobra.Command{
	Use:   "extract <FN>",
	Short: "Get company extract by Firmenbuch number",
	Long: `Retrieve the full company extract from the Austrian Firmenbuch.
The Firmenbuch number (FN) must be in the format FN123456a.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fn := args[0]
		testMode, _ := cmd.Flags().GetBool("test")
		apiKey, _ := cmd.Flags().GetString("api-key")

		// Validate FN format
		if err := fb.ValidateFN(fn); err != nil {
			return fmt.Errorf("invalid Firmenbuch number: %v", err)
		}

		if apiKey == "" {
			return fmt.Errorf("API key required (use --api-key or FB_API_KEY environment variable)")
		}

		client := fb.NewClient(apiKey, testMode)

		extract, err := client.Extract(fn)
		if err != nil {
			return fmt.Errorf("extract failed: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(extract)
		}

		// Display extract
		cmd.Printf("=== Firmenbuchauszug %s ===\n\n", extract.FN)
		cmd.Printf("Firma: %s\n", extract.Firma)
		cmd.Printf("Rechtsform: %s\n", extract.Rechtsform)
		cmd.Printf("Sitz: %s\n", extract.Sitz)
		cmd.Printf("Adresse: %s, %s %s\n", extract.Adresse.Strasse, extract.Adresse.PLZ, extract.Adresse.Ort)
		cmd.Printf("Status: %s\n", extract.Status)
		if extract.UID != "" {
			cmd.Printf("UID: %s\n", extract.UID)
		}
		if extract.Stammkapital > 0 {
			cmd.Printf("Stammkapital: %.2f %s\n", extract.StammkapitalEUR(), extract.Waehrung)
		}
		if extract.Gegenstand != "" {
			cmd.Printf("Unternehmensgegenstand: %s\n", extract.Gegenstand)
		}

		if len(extract.Geschaeftsfuehrer) > 0 {
			cmd.Printf("\nGeschäftsführer:\n")
			for _, gf := range extract.Geschaeftsfuehrer {
				cmd.Printf("  - %s (%s, seit %s)\n", gf.FullName(), gf.VertretungsArt, gf.SeitString())
			}
		}

		if len(extract.Gesellschafter) > 0 {
			cmd.Printf("\nGesellschafter:\n")
			for _, g := range extract.Gesellschafter {
				cmd.Printf("  - %s: %.2f%% (%.2f EUR)\n", g.Name, g.AnteilProzent(), g.StammeinlageEUR())
			}
		}

		return nil
	},
}

var fbValidateCmd = &cobra.Command{
	Use:   "validate <FN>",
	Short: "Validate a Firmenbuch number format",
	Long:  `Validate that a Firmenbuch number (FN) is in the correct format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fn := args[0]

		err := fb.ValidateFN(fn)
		if err != nil {
			if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
				return outputJSON(map[string]interface{}{
					"valid": false,
					"fn":    fn,
					"error": err.Error(),
				})
			}
			return fmt.Errorf("invalid Firmenbuch number: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]interface{}{
				"valid": true,
				"fn":    fn,
			})
		}

		cmd.Printf("Firmenbuch number %s is valid\n", fn)
		return nil
	},
}

// Watchlist commands
var fbWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Manage company watchlist",
	Long:  `Commands for managing a watchlist of companies to monitor for changes.`,
}

var fbWatchAddCmd = &cobra.Command{
	Use:   "add <FN> [note]",
	Short: "Add a company to the watchlist",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fn := args[0]
		note := ""
		if len(args) > 1 {
			note = args[1]
		}

		if err := fb.ValidateFN(fn); err != nil {
			return fmt.Errorf("invalid Firmenbuch number: %v", err)
		}

		wl, err := loadWatchlist()
		if err != nil {
			wl = fb.NewWatchlist()
		}

		entry := fb.FBWatchlistEntry{
			FN:      fn,
			AddedAt: time.Now(),
			Notes:   note,
		}

		wl.Add(entry)

		if err := saveWatchlist(wl); err != nil {
			return fmt.Errorf("failed to save watchlist: %v", err)
		}

		cmd.Printf("Added %s to watchlist\n", fn)
		return nil
	},
}

var fbWatchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all companies in the watchlist",
	RunE: func(cmd *cobra.Command, args []string) error {
		wl, err := loadWatchlist()
		if err != nil {
			cmd.Println("Watchlist is empty")
			return nil
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(wl)
		}

		if len(wl.Entries) == 0 {
			cmd.Println("Watchlist is empty")
			return nil
		}

		cmd.Printf("Watchlist (%d companies):\n\n", len(wl.Entries))
		for _, e := range wl.Entries {
			cmd.Printf("FN: %s\n", e.FN)
			if e.Firma != "" {
				cmd.Printf("  Firma: %s\n", e.Firma)
			}
			cmd.Printf("  Added: %s\n", e.AddedAt.Format("2006-01-02"))
			if !e.LastCheck.IsZero() {
				cmd.Printf("  Last Check: %s\n", e.LastCheck.Format("2006-01-02"))
			}
			if e.Notes != "" {
				cmd.Printf("  Notes: %s\n", e.Notes)
			}
			cmd.Println()
		}

		return nil
	},
}

var fbWatchRemoveCmd = &cobra.Command{
	Use:   "remove <FN>",
	Short: "Remove a company from the watchlist",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fn := args[0]

		wl, err := loadWatchlist()
		if err != nil {
			return fmt.Errorf("no watchlist found")
		}

		if !wl.Remove(fn) {
			return fmt.Errorf("%s not found in watchlist", fn)
		}

		if err := saveWatchlist(wl); err != nil {
			return fmt.Errorf("failed to save watchlist: %v", err)
		}

		cmd.Printf("Removed %s from watchlist\n", fn)
		return nil
	},
}

var fbWatchCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check all companies in the watchlist for changes",
	RunE: func(cmd *cobra.Command, args []string) error {
		testMode, _ := cmd.Flags().GetBool("test")
		apiKey, _ := cmd.Flags().GetString("api-key")

		if apiKey == "" {
			return fmt.Errorf("API key required")
		}

		wl, err := loadWatchlist()
		if err != nil || len(wl.Entries) == 0 {
			cmd.Println("Watchlist is empty")
			return nil
		}

		client := fb.NewClient(apiKey, testMode)
		changes := make([]map[string]interface{}, 0)

		for i := range wl.Entries {
			entry := &wl.Entries[i]
			extract, err := client.Extract(entry.FN)
			if err != nil {
				cmd.Printf("Error checking %s: %v\n", entry.FN, err)
				continue
			}

			// Check for status change
			if entry.LastStatus != "" && entry.LastStatus != extract.Status {
				change := map[string]interface{}{
					"fn":         entry.FN,
					"firma":      extract.Firma,
					"change":     "status",
					"old_status": entry.LastStatus,
					"new_status": extract.Status,
				}
				changes = append(changes, change)
				cmd.Printf("CHANGE: %s (%s) status changed from %s to %s\n",
					entry.FN, extract.Firma, entry.LastStatus, extract.Status)
			}

			// Update entry
			entry.Firma = extract.Firma
			entry.LastCheck = time.Now()
			entry.LastStatus = extract.Status
		}

		if err := saveWatchlist(wl); err != nil {
			return fmt.Errorf("failed to save watchlist: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]interface{}{
				"checked": len(wl.Entries),
				"changes": changes,
			})
		}

		if len(changes) == 0 {
			cmd.Printf("Checked %d companies, no changes detected\n", len(wl.Entries))
		}

		return nil
	},
}

func watchlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".fo", "fb-watchlist.json")
}

func loadWatchlist() (*fb.FBWatchlist, error) {
	data, err := os.ReadFile(watchlistPath())
	if err != nil {
		return nil, err
	}

	var wl fb.FBWatchlist
	if err := json.Unmarshal(data, &wl); err != nil {
		return nil, err
	}

	return &wl, nil
}

func saveWatchlist(wl *fb.FBWatchlist) error {
	data, err := json.MarshalIndent(wl, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(watchlistPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(watchlistPath(), data, 0600)
}

func init() {
	rootCmd.AddCommand(fbCmd)
	fbCmd.AddCommand(fbSearchCmd)
	fbCmd.AddCommand(fbExtractCmd)
	fbCmd.AddCommand(fbValidateCmd)
	fbCmd.AddCommand(fbWatchCmd)

	fbWatchCmd.AddCommand(fbWatchAddCmd)
	fbWatchCmd.AddCommand(fbWatchListCmd)
	fbWatchCmd.AddCommand(fbWatchRemoveCmd)
	fbWatchCmd.AddCommand(fbWatchCheckCmd)

	// Common flags
	fbSearchCmd.Flags().Bool("json", false, "Output in JSON format")
	fbExtractCmd.Flags().Bool("json", false, "Output in JSON format")
	fbValidateCmd.Flags().Bool("json", false, "Output in JSON format")
	fbWatchListCmd.Flags().Bool("json", false, "Output in JSON format")
	fbWatchCheckCmd.Flags().Bool("json", false, "Output in JSON format")

	// Search flags
	fbSearchCmd.Flags().String("ort", "", "Filter by city/location")
	fbSearchCmd.Flags().Int("max", 20, "Maximum number of results")
	fbSearchCmd.Flags().Bool("test", false, "Use test endpoint")
	fbSearchCmd.Flags().String("api-key", os.Getenv("FB_API_KEY"), "API key for Firmenbuch")

	// Extract flags
	fbExtractCmd.Flags().Bool("test", false, "Use test endpoint")
	fbExtractCmd.Flags().String("api-key", os.Getenv("FB_API_KEY"), "API key for Firmenbuch")

	// Watch check flags
	fbWatchCheckCmd.Flags().Bool("test", false, "Use test endpoint")
	fbWatchCheckCmd.Flags().String("api-key", os.Getenv("FB_API_KEY"), "API key for Firmenbuch")
}
