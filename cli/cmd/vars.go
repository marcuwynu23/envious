package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"envious-cli/internal/client"
	"envious-cli/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newVarCmd())
}

func newVarCmd() *cobra.Command {
	varCmd := &cobra.Command{
		Use:   "var",
		Short: "Manage variables",
	}
	varCmd.AddCommand(varListCmd(), varSetCmd(), varDeleteCmd(), varExportCmd(), varImportCmd())
	return varCmd
}

func varListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <env_id>",
		Short: "List variables in an environment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var envID int64
			if _, err := fmt.Sscan(args[0], &envID); err != nil {
				return err
			}
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			vars, err := c.ListVars(envID)
			if err != nil {
				return err
			}
			for _, v := range vars {
				fmt.Fprintf(cmd.OutOrStdout(), "%v\t%s\t%s\tv%v\n", v["id"], v["key"], v["value"], v["version"])
			}
			return nil
		},
	}
}

func varSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <env_id> <key> <value>",
		Short: "Set a variable (creates or updates with versioning)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			var envID int64
			if _, err := fmt.Sscan(args[0], &envID); err != nil {
				return err
			}
			key := args[1]
			val := args[2]
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			if _, err := c.SetVar(envID, key, val); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}

func varDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <var_id>",
		Short: "Delete a variable by ID",
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
			if err := c.DeleteVarByID(id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}

func varExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export <env_id>",
		Short: "Export variables as .env format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var envID int64
			if _, err := fmt.Sscan(args[0], &envID); err != nil {
				return err
			}
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			vars, err := c.ListVars(envID)
			if err != nil {
				return err
			}
			for _, v := range vars {
				fmt.Fprintf(cmd.OutOrStdout(), "%s=%s\n", v["key"], v["value"])
			}
			return nil
		},
	}
}

func varImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <env_id> <file>",
		Short: "Import variables from .env file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var envID int64
			if _, err := fmt.Sscan(args[0], &envID); err != nil {
				return err
			}
			f, err := os.Open(args[1])
			if err != nil {
				return err
			}
			defer f.Close()
			cfg, _ := config.Load()
			c, err := client.New(cfg.APIBase, cfg.APIKey)
			if err != nil {
				return err
			}
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				line := sc.Text()
				if len(line) == 0 || line[0] == '#' {
					continue
				}
				var key, val string
				if i := strings.IndexRune(line, '='); i > 0 {
					key = line[:i]
					val = line[i+1:]
				} else {
					continue
				}
				if _, err := c.SetVar(envID, key, val); err != nil {
					return fmt.Errorf("set %s: %w", key, err)
				}
			}
			return sc.Err()
		},
	}
}
