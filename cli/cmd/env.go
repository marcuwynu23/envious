package cmd

import (
	"fmt"

	"envious-cli/internal/client"
	"envious-cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newEnvCmd())
}

func newEnvCmd() *cobra.Command {
	envCmd := &cobra.Command{
		Use:   "env",
		Short: "Manage environments",
	}
	envCmd.AddCommand(envListCmd(), envCreateCmd(), envDeleteCmd())
	return envCmd
}

func envListCmd() *cobra.Command {
	var appID int64
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			envs, err := c.ListEnvs(appID)
			if err != nil {
				return err
			}
			for _, e := range envs {
				fmt.Fprintf(cmd.OutOrStdout(), "%v\tapp=%v\t%s\n", e["id"], e["app_id"], e["name"])
			}
			return nil
		},
	}
	cmd.Flags().Int64Var(&appID, "app-id", 0, "application ID (0 = all)")
	return cmd
}

func envCreateCmd() *cobra.Command {
	var appID int64
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			_, err = c.CreateEnv(appID, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
	cmd.Flags().Int64Var(&appID, "app-id", 0, "application ID (0 = default)")
	return cmd
}

func envDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var id int64
			if _, err := fmt.Sscan(args[0], &id); err != nil {
				return err
			}
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			if err := c.DeleteEnv(id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}
