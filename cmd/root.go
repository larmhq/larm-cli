package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/larmhq/larm-cli/internal/config"
	"github.com/larmhq/larm-cli/internal/output"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:           "larm",
	Short:         "Larm CLI — uptime monitoring from the command line",
	Long:          "Manage monitors, alerts, status pages, incidents, and webhooks via the Larm API.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringP("output", "o", "", "Output format: table or json (auto-detects)")
	rootCmd.PersistentFlags().String("jq", "", "JQ expression to filter JSON output")
	rootCmd.PersistentFlags().String("fields", "", "Comma-separated fields to display")
	rootCmd.PersistentFlags().String("api-key", "", "API key (overrides LARM_API_KEY and config)")
	rootCmd.PersistentFlags().String("api-url", "https://app.larm.dev", "API base URL")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress output on success")
	rootCmd.PersistentFlags().Bool("dry-run", false, "Print request without sending")
	rootCmd.PersistentFlags().BoolP("yes", "y", false, "Skip confirmation prompts")

	rootCmd.Version = fmt.Sprintf("%s (%s) built %s", version, commit, date)

	config.Init()
}

// Execute runs the root command and returns an exit code.
func Execute() int {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, ErrDryRun) {
			return 0
		}
		format, _ := rootCmd.PersistentFlags().GetString("output")
		jsonMode := format == "json" || !output.IsTTY()
		output.PrintError(os.Stderr, err, jsonMode)
		return output.ExitCodeFor(err)
	}
	return 0
}
