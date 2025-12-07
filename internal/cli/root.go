package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgDir  string
	jsonOut bool
	verbose bool

	// Version info (set via ldflags)
	Version   = "dev"
	BuildDate = "unknown"

	// Output writers
	errWriter io.Writer = os.Stderr
)

var rootCmd = &cobra.Command{
	Use:   "fo",
	Short: "FinanzOnline CLI - Multi-account session management and databox access",
	Long: `fo is a command-line tool for interacting with the Austrian FinanzOnline
WebService API. It enables tax accountants and businesses to manage
multiple accounts and check databoxes for new documents efficiently.

Features:
  - Multi-account credential management with encrypted storage
  - Session login/logout with FinanzOnline WebService
  - Databox document listing and download
  - Batch operations across all accounts`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgDir, "config", "c", "", "config directory (default: platform-specific)")
	rootCmd.PersistentFlags().BoolVarP(&jsonOut, "json", "j", false, "output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for the specified shell.

To load completions:

Bash:
  $ source <(fo completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ fo completion bash > /etc/bash_completion.d/fo
  # macOS:
  $ fo completion bash > $(brew --prefix)/etc/bash_completion.d/fo

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ fo completion zsh > "${fpath[1]}/_fo"
  # You will need to start a new shell for this setup to take effect.

Fish:
  $ fo completion fish | source
  # To load completions for each session, execute once:
  $ fo completion fish > ~/.config/fish/completions/fo.fish

PowerShell:
  PS> fo completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> fo completion powershell > fo.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fo version %s (built %s)\n", Version, BuildDate)
	},
}

// GetConfigDir returns the config directory from flag or default
func GetConfigDir() string {
	return cfgDir
}

// IsJSONOutput returns true if JSON output is enabled
func IsJSONOutput() bool {
	return jsonOut
}

// IsVerbose returns true if verbose logging is enabled
func IsVerbose() bool {
	return verbose
}

// LogVerbose prints a message if verbose mode is enabled
func LogVerbose(format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}
