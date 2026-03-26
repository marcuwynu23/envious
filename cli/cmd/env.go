package cmd

import (
	"fmt"

	"envious-cli/internal/client"
	"envious-cli/internal/config"
	"envious-cli/internal/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newEnvCmd())
}

func newEnvCmd() *cobra.Command {
	envCmd := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env", "envs", "environments"},
		Short:   "Manage environments",
	}
	envCmd.AddCommand(envListCmd(), envCreateCmd(), envDeleteCmd())
	return envCmd
}

func envListCmd() *cobra.Command {
	var appID int64
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List environments",
		Example: `  envious env list
  envious env list --app-id 2`,
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
			apps, err := c.ListApps()
			if err != nil {
				return err
			}
			appNameByID := map[string]string{}
			for _, a := range apps {
				appNameByID[fmt.Sprint(a["id"])] = fmt.Sprint(a["name"])
			}

			t := view.Table{Headers: []string{"ID", "APP_ID", "APPLICATION", "NAME"}}
			for _, e := range envs {
				appIDStr := fmt.Sprint(e["app_id"])
				t.Rows = append(t.Rows, []string{
					fmt.Sprint(e["id"]),
					appIDStr,
					appNameByID[appIDStr],
					fmt.Sprint(e["name"]),
				})
			}
			t.Render(cmd.OutOrStdout())
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
