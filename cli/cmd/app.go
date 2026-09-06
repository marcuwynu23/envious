package cmd

import (
	"fmt"

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
			c, err := loadClient()
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
			c, err := loadClient()
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
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			c, err := loadClient()
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

