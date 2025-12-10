package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"austrian-business-infrastructure/internal/config"
	"austrian-business-infrastructure/internal/elda"
	"austrian-business-infrastructure/internal/store"
	"github.com/spf13/cobra"
)

var eldaCmd = &cobra.Command{
	Use:   "elda",
	Short: "ELDA employee registration management",
	Long:  `Commands for managing employee registrations and deregistrations via ELDA (Elektronischer Datenaustausch mit den österreichischen Sozialversicherungsträgern).`,
}

var eldaValidateCmd = &cobra.Command{
	Use:   "validate [sv-nummer]",
	Short: "Validate an Austrian social security number (SV-Nummer)",
	Long: `Validate an Austrian SV-Nummer (Sozialversicherungsnummer).
The SV-Nummer is a 10-digit number with an embedded birth date and check digit.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		svNummer := args[0]

		err := elda.ValidateSVNummer(svNummer)
		if err != nil {
			if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
				return outputJSON(map[string]interface{}{
					"valid":     false,
					"sv_nummer": svNummer,
					"error":     err.Error(),
				})
			}
			return fmt.Errorf("invalid SV-Nummer: %v", err)
		}

		// Extract birth date
		birthDate, _ := elda.ExtractBirthDateFromSVNummer(svNummer)

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]interface{}{
				"valid":      true,
				"sv_nummer":  svNummer,
				"formatted":  elda.FormatSVNummer(svNummer),
				"birth_date": birthDate.Format("2006-01-02"),
			})
		}

		cmd.Printf("SV-Nummer: %s\n", elda.FormatSVNummer(svNummer))
		cmd.Printf("Status: Valid\n")
		cmd.Printf("Birth Date: %s\n", birthDate.Format("02.01.2006"))
		return nil
	},
}

// EmployeeJSONInput represents the JSON format for employee input file
type EmployeeJSONInput struct {
	SVNummer     string  `json:"sv_nummer"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	DateOfBirth  string  `json:"date_of_birth"`
	Gender       string  `json:"gender,omitempty"`
	EmployerVSNR string  `json:"employer_vsnr"`
	StartDate    string  `json:"start_date"`
	InsuranceType string `json:"insurance_type,omitempty"`
	WeeklyHours  float64 `json:"weekly_hours,omitempty"`
	MonthlyGross int64   `json:"monthly_gross,omitempty"`
	JobTitle     string  `json:"job_title,omitempty"`
}

var eldaAnmeldenCmd = &cobra.Command{
	Use:   "anmelden",
	Short: "Register an employee (Anmeldung)",
	Long: `Submit an employee registration (Anmeldung) to ELDA.

You can provide employee data either via flags or JSON file:

  fo elda anmelden --employee-file employee.json --account "My Company"

Or via individual flags:

  fo elda anmelden --sv-nummer 1234150189 --vorname Max --nachname Mustermann ...`,
	RunE: runEldaAnmelden,
}

func runEldaAnmelden(cmd *cobra.Command, args []string) error {
	employeeFile, _ := cmd.Flags().GetString("employee-file")
	accountName, _ := cmd.Flags().GetString("account")
	testMode, _ := cmd.Flags().GetBool("test")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	var anmeldung *elda.ELDAAnmeldung

	if employeeFile != "" {
		// Load from JSON file
		var err error
		anmeldung, err = loadAnmeldungFromFile(employeeFile)
		if err != nil {
			return err
		}
	} else {
		// Use flags
		var err error
		anmeldung, err = buildAnmeldungFromFlags(cmd)
		if err != nil {
			return err
		}
	}

	// For dry-run, just generate and display XML
	if dryRun {
		creds := &elda.ELDACredentials{
			DienstgeberNr: "TEST12345",
		}
		xmlData, err := elda.GenerateAnmeldungXML(creds, anmeldung)
		if err != nil {
			return fmt.Errorf("failed to generate XML: %v", err)
		}
		fmt.Println(string(xmlData))
		return nil
	}

	// Get credentials from store
	creds, err := getELDACredentials(accountName)
	if err != nil {
		return err
	}

	// Create client
	client := elda.NewClient(testMode)

	resp, err := client.SubmitAnmeldung(creds, anmeldung)
	if err != nil {
		return fmt.Errorf("failed to submit Anmeldung: %v", err)
	}

	if IsJSONOutput() {
		return outputJSON(map[string]interface{}{
			"status":    "submitted",
			"reference": resp.Reference,
			"message":   resp.Msg,
		})
	}

	fmt.Printf("Anmeldung submitted successfully\n")
	fmt.Printf("Reference: %s\n", resp.Reference)
	return nil
}

func loadAnmeldungFromFile(filePath string) (*elda.ELDAAnmeldung, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read employee file: %w", err)
	}

	var input EmployeeJSONInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("failed to parse employee JSON: %w", err)
	}

	// Validate SV-Nummer
	if err := elda.ValidateSVNummer(input.SVNummer); err != nil {
		return nil, fmt.Errorf("invalid SV-Nummer: %v", err)
	}

	// Parse dates
	dateOfBirth, err := time.Parse("2006-01-02", input.DateOfBirth)
	if err != nil {
		return nil, fmt.Errorf("invalid date_of_birth format (use YYYY-MM-DD): %v", err)
	}

	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format (use YYYY-MM-DD): %v", err)
	}

	// Validate SV-Nummer matches birth date
	if err := elda.ValidateSVNummerWithBirthDate(input.SVNummer, dateOfBirth); err != nil {
		return nil, fmt.Errorf("SV-Nummer birth date mismatch: %v", err)
	}

	// Set defaults
	gender := input.Gender
	if gender == "" {
		gender = "M"
	}
	weeklyHours := input.WeeklyHours
	if weeklyHours == 0 {
		weeklyHours = 38.5
	}

	anmeldung := elda.NewELDAAnmeldung()
	anmeldung.SVNummer = input.SVNummer
	anmeldung.Vorname = input.FirstName
	anmeldung.Nachname = input.LastName
	anmeldung.Geburtsdatum = dateOfBirth
	anmeldung.Geschlecht = gender
	anmeldung.Eintrittsdatum = startDate
	anmeldung.DienstgeberNr = input.EmployerVSNR
	anmeldung.Beschaeftigung = elda.ELDABeschaeftigung{
		Art:        elda.BeschaeftigungVollzeit,
		Taetigkeit: input.JobTitle,
	}
	anmeldung.Arbeitszeit = elda.ELDAArbeitszeit{
		Stunden: weeklyHours,
		Tage:    5,
	}
	anmeldung.Entgelt = elda.ELDAEntgelt{
		Brutto: input.MonthlyGross,
	}

	return anmeldung, nil
}

func buildAnmeldungFromFlags(cmd *cobra.Command) (*elda.ELDAAnmeldung, error) {
	svNummer, _ := cmd.Flags().GetString("sv-nummer")
	vorname, _ := cmd.Flags().GetString("vorname")
	nachname, _ := cmd.Flags().GetString("nachname")
	geburtsdatumStr, _ := cmd.Flags().GetString("geburtsdatum")
	geschlecht, _ := cmd.Flags().GetString("geschlecht")
	eintrittsdatumStr, _ := cmd.Flags().GetString("eintrittsdatum")
	beschaeftigungArt, _ := cmd.Flags().GetString("beschaeftigung")
	taetigkeit, _ := cmd.Flags().GetString("taetigkeit")
	stunden, _ := cmd.Flags().GetFloat64("stunden")
	tage, _ := cmd.Flags().GetInt("tage")
	brutto, _ := cmd.Flags().GetInt64("brutto")

	// Validate required fields
	if svNummer == "" || vorname == "" || nachname == "" {
		return nil, fmt.Errorf("sv-nummer, vorname, and nachname are required")
	}

	// Validate SV-Nummer
	if err := elda.ValidateSVNummer(svNummer); err != nil {
		return nil, fmt.Errorf("invalid SV-Nummer: %v", err)
	}

	// Parse dates
	geburtsdatum, err := time.Parse("2006-01-02", geburtsdatumStr)
	if err != nil {
		return nil, fmt.Errorf("invalid geburtsdatum format (use YYYY-MM-DD): %v", err)
	}

	eintrittsdatum, err := time.Parse("2006-01-02", eintrittsdatumStr)
	if err != nil {
		return nil, fmt.Errorf("invalid eintrittsdatum format (use YYYY-MM-DD): %v", err)
	}

	// Validate SV-Nummer matches birth date
	if err := elda.ValidateSVNummerWithBirthDate(svNummer, geburtsdatum); err != nil {
		return nil, fmt.Errorf("SV-Nummer birth date mismatch: %v", err)
	}

	anmeldung := elda.NewELDAAnmeldung()
	anmeldung.SVNummer = svNummer
	anmeldung.Vorname = vorname
	anmeldung.Nachname = nachname
	anmeldung.Geburtsdatum = geburtsdatum
	anmeldung.Geschlecht = geschlecht
	anmeldung.Eintrittsdatum = eintrittsdatum
	anmeldung.Beschaeftigung = elda.ELDABeschaeftigung{
		Art:        beschaeftigungArt,
		Taetigkeit: taetigkeit,
	}
	anmeldung.Arbeitszeit = elda.ELDAArbeitszeit{
		Stunden: stunden,
		Tage:    tage,
	}
	anmeldung.Entgelt = elda.ELDAEntgelt{
		Brutto: brutto,
	}

	return anmeldung, nil
}

func getELDACredentials(accountName string) (*elda.ELDACredentials, error) {
	if accountName == "" {
		// Use placeholder credentials for testing
		return &elda.ELDACredentials{
			DienstgeberNr: "12345678",
			BenutzerNr:    "USER001",
			PIN:           "secret",
		}, nil
	}

	// Get config directory
	cfgDir, err := config.GetConfigDir(GetConfigDir())
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Prompt for master password
	masterPassword, err := promptPassword("Master password: ")
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}

	// Load credential store
	credPath := config.GetCredentialPath(cfgDir)
	credStore, err := store.Load(credPath, masterPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	// Find account
	account, err := credStore.GetAccount(accountName)
	if err != nil {
		return nil, fmt.Errorf("account not found: %s", accountName)
	}

	// For now, return placeholder since CredentialStore uses legacy Account struct
	// TODO: Update when ExtendedAccount is integrated into CredentialStore
	return &elda.ELDACredentials{
		DienstgeberNr: account.TID[:8], // Use first 8 digits of TID as DienstgeberNr
		BenutzerNr:    account.BenID,
		PIN:           account.PIN,
	}, nil
}

var eldaAbmeldenCmd = &cobra.Command{
	Use:   "abmelden",
	Short: "Deregister an employee (Abmeldung)",
	Long:  `Submit an employee deregistration (Abmeldung) to ELDA.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svNummer, _ := cmd.Flags().GetString("sv-nummer")
		austrittsdatumStr, _ := cmd.Flags().GetString("austrittsdatum")
		grund, _ := cmd.Flags().GetString("grund")
		abfertigung, _ := cmd.Flags().GetInt64("abfertigung")
		urlaubsersatz, _ := cmd.Flags().GetInt64("urlaubsersatz")
		testMode, _ := cmd.Flags().GetBool("test")

		if svNummer == "" || austrittsdatumStr == "" {
			return fmt.Errorf("sv-nummer and austrittsdatum are required")
		}

		// Validate SV-Nummer
		if err := elda.ValidateSVNummer(svNummer); err != nil {
			return fmt.Errorf("invalid SV-Nummer: %v", err)
		}

		austrittsdatum, err := time.Parse("2006-01-02", austrittsdatumStr)
		if err != nil {
			return fmt.Errorf("invalid austrittsdatum format (use YYYY-MM-DD): %v", err)
		}

		// Build Abmeldung
		abmeldung := elda.NewELDAAbmeldung()
		abmeldung.SVNummer = svNummer
		abmeldung.Austrittsdatum = austrittsdatum
		abmeldung.Grund = elda.ELDAAustrittGrund(grund)
		abmeldung.Abfertigung = abfertigung
		abmeldung.Urlaubsersatz = urlaubsersatz

		// Create client
		client := elda.NewClient(testMode)

		// Get credentials from store (placeholder)
		creds := &elda.ELDACredentials{
			DienstgeberNr: "12345678",
			BenutzerNr:    "USER001",
			PIN:           "secret",
		}

		resp, err := client.SubmitAbmeldung(creds, abmeldung)
		if err != nil {
			return fmt.Errorf("failed to submit Abmeldung: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]interface{}{
				"status":    "submitted",
				"reference": resp.Reference,
				"message":   resp.Msg,
			})
		}

		cmd.Printf("Abmeldung submitted successfully\n")
		cmd.Printf("Reference: %s\n", resp.Reference)
		return nil
	},
}

var eldaStatusCmd = &cobra.Command{
	Use:   "status [reference]",
	Short: "Query status of an ELDA submission",
	Long:  `Query the processing status of a previously submitted ELDA Anmeldung or Abmeldung.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		reference := args[0]
		testMode, _ := cmd.Flags().GetBool("test")

		client := elda.NewClient(testMode)

		// Get credentials from store (placeholder)
		creds := &elda.ELDACredentials{
			DienstgeberNr: "12345678",
			BenutzerNr:    "USER001",
			PIN:           "secret",
		}

		resp, err := client.QueryStatus(creds, reference)
		if err != nil {
			return fmt.Errorf("failed to query status: %v", err)
		}

		if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
			return outputJSON(map[string]interface{}{
				"reference": reference,
				"status":    resp.RC,
				"message":   resp.Msg,
			})
		}

		cmd.Printf("Reference: %s\n", reference)
		cmd.Printf("Status Code: %d\n", resp.RC)
		cmd.Printf("Message: %s\n", resp.Msg)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(eldaCmd)
	eldaCmd.AddCommand(eldaValidateCmd)
	eldaCmd.AddCommand(eldaAnmeldenCmd)
	eldaCmd.AddCommand(eldaAbmeldenCmd)
	eldaCmd.AddCommand(eldaStatusCmd)

	// Common flags
	eldaValidateCmd.Flags().Bool("json", false, "Output in JSON format")
	eldaAnmeldenCmd.Flags().Bool("json", false, "Output in JSON format")
	eldaAbmeldenCmd.Flags().Bool("json", false, "Output in JSON format")
	eldaStatusCmd.Flags().Bool("json", false, "Output in JSON format")

	// Anmelden flags
	eldaAnmeldenCmd.Flags().String("employee-file", "", "JSON file containing employee data")
	eldaAnmeldenCmd.Flags().String("account", "", "Account name to use for credentials")
	eldaAnmeldenCmd.Flags().String("sv-nummer", "", "Social security number (SV-Nummer)")
	eldaAnmeldenCmd.Flags().String("vorname", "", "First name")
	eldaAnmeldenCmd.Flags().String("nachname", "", "Last name")
	eldaAnmeldenCmd.Flags().String("geburtsdatum", "", "Birth date (YYYY-MM-DD)")
	eldaAnmeldenCmd.Flags().String("geschlecht", "M", "Gender (M/W)")
	eldaAnmeldenCmd.Flags().String("eintrittsdatum", "", "Entry date (YYYY-MM-DD)")
	eldaAnmeldenCmd.Flags().String("beschaeftigung", "vollzeit", "Employment type (vollzeit/teilzeit/geringfuegig)")
	eldaAnmeldenCmd.Flags().String("taetigkeit", "", "Job description")
	eldaAnmeldenCmd.Flags().Float64("stunden", 38.5, "Weekly hours")
	eldaAnmeldenCmd.Flags().Int("tage", 5, "Working days per week")
	eldaAnmeldenCmd.Flags().Int64("brutto", 0, "Monthly gross salary in cents")
	eldaAnmeldenCmd.Flags().Bool("test", false, "Use test endpoint")
	eldaAnmeldenCmd.Flags().Bool("dry-run", false, "Generate XML without submitting")

	// Abmelden flags
	eldaAbmeldenCmd.Flags().String("sv-nummer", "", "Social security number (SV-Nummer)")
	eldaAbmeldenCmd.Flags().String("austrittsdatum", "", "Exit date (YYYY-MM-DD)")
	eldaAbmeldenCmd.Flags().String("grund", "K", "Exit reason (K=Kündigung, E=Einvernehmlich, EN=Entlassung, A=Austritt, B=Befristet)")
	eldaAbmeldenCmd.Flags().Int64("abfertigung", 0, "Severance pay in cents")
	eldaAbmeldenCmd.Flags().Int64("urlaubsersatz", 0, "Vacation compensation in cents")
	eldaAbmeldenCmd.Flags().Bool("test", false, "Use test endpoint")

	// Status flags
	eldaStatusCmd.Flags().Bool("test", false, "Use test endpoint")
}
