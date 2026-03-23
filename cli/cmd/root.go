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
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "envious",
	Short: "Envious CLI - environment manager",
	Long:  `Envious CLI manages environments and variables via the Envious server.`,
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.app.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(version.NewCommand(func() (service.VersionProvider, *view.VersionRenderer) {
		d := deps()
		return d.VersionProvider, d.VersionView
	}))
}
