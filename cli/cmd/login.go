package cmd

import (
	"fmt"

	"envious-cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newLoginCmd())
}

func newLoginCmd() *cobra.Command {
	var apiKey string
	var apiBase string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store API key and server URL for future commands",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				cfg = config.Default()
			}
			if apiBase != "" {
				cfg.APIBase = apiBase
			}
			if apiKey != "" {
				cfg.APIKey = apiKey
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "login saved")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (from server)")
	cmd.Flags().StringVar(&apiBase, "api-base", "", "API base URL (e.g. http://127.0.0.1:8080)")
	return cmd
}
