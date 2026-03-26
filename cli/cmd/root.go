package cmd

import (
	"fmt"
	"os"

	"envious-cli/cmd/version"
	"envious-cli/internal/service"
	"envious-cli/internal/view"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

// Build-time version (set via ldflags).
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
	Author    = "unknown"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "envious",
	Short: "Envious CLI",
	Long: `Manage applications, environments, and variables via the Envious server.

Examples:
  envious application list
  envious environment list --app-id 2
  envious variable list --env-id 1
  envious variable set --env-id 1 API_KEY secret
  envious variable import --env-id 1 .env`,
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Optional: validate global config, load config file, etc.
		return nil
	},
}

// Execute runs the root command and all subcommands.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// RootCmd returns the root command for testing (e.g. from test/ folder).
func RootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true // Disable the default completion command

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	rootCmd.AddCommand(version.NewCommand(func() (service.VersionProvider, *view.VersionRenderer) {
		d := deps()
		return d.VersionProvider, d.VersionView
	}))
}
