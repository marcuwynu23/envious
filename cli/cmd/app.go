package cmd

import (
	"fmt"

	"envious-cli/internal/client"
	"envious-cli/internal/config"
	"envious-cli/internal/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newAppCmd())
}

func newAppCmd() *cobra.Command {
	appCmd := &cobra.Command{
		Use:     "application",
		Aliases: []string{"app", "apps"},
		Short:   "Manage applications",
	}
	appCmd.AddCommand(appListCmd(), appCreateCmd(), appDeleteCmd())
	return appCmd
}

func appListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List applications",
		Example: `  envious app list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			apps, err := c.ListApps()
			if err != nil {
				return err
			}
			t := view.Table{Headers: []string{"ID", "NAME"}}
			for _, a := range apps {
				t.Rows = append(t.Rows, []string{
					fmt.Sprint(a["id"]),
					fmt.Sprint(a["name"]),
				})
			}
			t.Render(cmd.OutOrStdout())
			return nil
		},
	}
}

func appCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <name>",
		Short: "Create application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			if _, err := c.CreateApp(args[0]); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}

func appDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete application",
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
			if err := c.DeleteApp(id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}

