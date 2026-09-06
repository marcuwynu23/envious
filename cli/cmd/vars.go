package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"envious-cli/internal/client"
	"envious-cli/internal/view"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newVarCmd())
}

func newVarCmd() *cobra.Command {
	varCmd := &cobra.Command{
		Use:     "variable",
		Aliases: []string{"var", "vars", "variables"},
		Short:   "Manage variables",
	}
	varCmd.AddCommand(varListCmd(), varSetCmd(), varDeleteCmd(), varExportCmd(), varImportCmd())
	return varCmd
}

func resolveEnvID(
	c *client.Client,
	explicitEnvID int64,
	argEnvID string,
	appID int64,
	envName string,
) (int64, error) {
	if explicitEnvID > 0 {
		return explicitEnvID, nil
	}
	if strings.TrimSpace(argEnvID) != "" {
		envID, err := parseID(argEnvID)
		if err != nil {
			return 0, err
		}
		return envID, nil
	}
	if strings.TrimSpace(envName) == "" {
		return 0, fmt.Errorf("env is required (use <env_id> or --env-id, or --env-name with optional --app-id)")
	}
	envs, err := c.ListEnvs(appID)
	if err != nil {
		return 0, err
	}
	var matchID int64
	for _, e := range envs {
		if fmt.Sprint(e["name"]) == envName {
			if matchID != 0 {
				return 0, fmt.Errorf("environment name %q is ambiguous; use --app-id or --env-id", envName)
			}
			id, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(e["id"])), 10, 64)
			if err != nil || id <= 0 {
				return 0, fmt.Errorf("invalid environment id %v", e["id"])
			}
			matchID = id
		}
	}
	if matchID == 0 {
		return 0, fmt.Errorf("environment %q not found", envName)
	}
	return matchID, nil
}

func resolveAppID(c *client.Client, explicitAppID int64, appName string) (int64, error) {
	if explicitAppID > 0 {
		return explicitAppID, nil
	}
	appName = strings.TrimSpace(appName)
	if appName == "" {
		return 0, nil
	}
	apps, err := c.ListApps()
	if err != nil {
		return 0, err
	}
	for _, a := range apps {
		if fmt.Sprint(a["name"]) == appName {
			id, err := strconv.ParseInt(strings.TrimSpace(fmt.Sprint(a["id"])), 10, 64)
			if err == nil && id > 0 {
				return id, nil
			}
		}
	}
	return 0, fmt.Errorf("application %q not found", appName)
}

func varListCmd() *cobra.Command {
	var showValues bool
	var envIDFlag int64
	var appIDFlag int64
	var appName string
	var envName string
	cmd := &cobra.Command{
		Use:   "list [env_id]",
		Short: "List variables in an environment (values hidden by default)",
		Example: `  envious variable list 1
  envious variable list --env-id 1
  envious variable list --app-id 2 --env-name development
  envious variable list --app-name default --env-name development
  envious variable list 1 --show-values`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var argEnvID string
			if len(args) == 1 {
				argEnvID = args[0]
			}
			c, err := loadClient()
			if err != nil {
				return err
			}
			resolvedAppID, err := resolveAppID(c, appIDFlag, appName)
			if err != nil {
				return err
			}
			envID, err := resolveEnvID(c, envIDFlag, argEnvID, resolvedAppID, envName)
			if err != nil {
				return err
			}
			vars, err := c.ListVars(envID)
			if err != nil {
				return err
			}
			headers := []string{"ID", "KEY", "VERSION"}
			if showValues {
				headers = append(headers, "VALUE")
			}
			t := view.Table{Headers: headers}
			for _, v := range vars {
				row := []string{
					fmt.Sprint(v["id"]),
					fmt.Sprint(v["key"]),
					"v" + fmt.Sprint(v["version"]),
				}
				if showValues {
					row = append(row, fmt.Sprint(v["value"]))
				}
				t.Rows = append(t.Rows, row)
			}
			t.Render(cmd.OutOrStdout())
			return nil
		},
	}
	cmd.Flags().BoolVar(&showValues, "show-values", false, "show variable values in output")
	cmd.Flags().Int64Var(&envIDFlag, "env-id", 0, "environment ID")
	cmd.Flags().Int64Var(&appIDFlag, "app-id", 0, "application ID (used with --env-name; 0 = all)")
	cmd.Flags().StringVar(&appName, "app-name", "", "application name (used with --env-name)")
	cmd.Flags().StringVar(&envName, "env-name", "", "environment name (used to resolve env ID)")
	return cmd
}

func varSetCmd() *cobra.Command {
	var envIDFlag int64
	var appIDFlag int64
	var appName string
	var envName string
	cmd := &cobra.Command{
		Use:   "set [env_id] <key> <value>",
		Short: "Set a variable (creates or updates with versioning)",
		Example: `  envious variable set 1 API_KEY secret
  envious variable set --env-id 1 API_KEY secret
  envious variable set --app-id 2 --env-name development API_KEY secret`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			var argEnvID string
			var key, val string
			if len(args) == 3 {
				argEnvID = args[0]
				key = args[1]
				val = args[2]
			} else {
				key = args[0]
				val = args[1]
			}
			c, err := loadClient()
			if err != nil {
				return err
			}
			resolvedAppID, err := resolveAppID(c, appIDFlag, appName)
			if err != nil {
				return err
			}
			envID, err := resolveEnvID(c, envIDFlag, argEnvID, resolvedAppID, envName)
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
	cmd.Flags().Int64Var(&envIDFlag, "env-id", 0, "environment ID")
	cmd.Flags().Int64Var(&appIDFlag, "app-id", 0, "application ID (used with --env-name; 0 = all)")
	cmd.Flags().StringVar(&appName, "app-name", "", "application name (used with --env-name)")
	cmd.Flags().StringVar(&envName, "env-name", "", "environment name (used to resolve env ID)")
	return cmd
}

func varDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <var_id>",
		Short: "Delete a variable by ID",
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
			if err := c.DeleteVarByID(id); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ok")
			return nil
		},
	}
}

func varExportCmd() *cobra.Command {
	var envIDFlag int64
	var appIDFlag int64
	var appName string
	var envName string
	cmd := &cobra.Command{
		Use:   "export [env_id]",
		Short: "Export variables as .env format",
		Example: `  envious variable export 1
  envious variable export --env-id 1
  envious variable export --app-id 2 --env-name development`,
		Args: cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var argEnvID string
			if len(args) == 1 {
				argEnvID = args[0]
			}
			c, err := loadClient()
			if err != nil {
				return err
			}
			resolvedAppID, err := resolveAppID(c, appIDFlag, appName)
			if err != nil {
				return err
			}
			envID, err := resolveEnvID(c, envIDFlag, argEnvID, resolvedAppID, envName)
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
	cmd.Flags().Int64Var(&envIDFlag, "env-id", 0, "environment ID")
	cmd.Flags().Int64Var(&appIDFlag, "app-id", 0, "application ID (used with --env-name; 0 = all)")
	cmd.Flags().StringVar(&appName, "app-name", "", "application name (used with --env-name)")
	cmd.Flags().StringVar(&envName, "env-name", "", "environment name (used to resolve env ID)")
	return cmd
}

func varImportCmd() *cobra.Command {
	var envIDFlag int64
	var appIDFlag int64
	var appName string
	var envName string
	cmd := &cobra.Command{
		Use:   "import [env_id] <file>",
		Short: "Import variables from .env file",
		Example: `  envious variable import 1 .env
  envious variable import --env-id 1 .env
  envious variable import --app-id 2 --env-name development .env`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var argEnvID string
			filePath := args[0]
			if len(args) == 2 {
				argEnvID = args[0]
				filePath = args[1]
			}
			f, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer f.Close()
			c, err := loadClient()
			if err != nil {
				return err
			}
			resolvedAppID, err := resolveAppID(c, appIDFlag, appName)
			if err != nil {
				return err
			}
			envID, err := resolveEnvID(c, envIDFlag, argEnvID, resolvedAppID, envName)
			if err != nil {
				return err
			}
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 64*1024), 1024*1024)
			for sc.Scan() {
				line := strings.TrimSpace(sc.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				var key, val string
				if i := strings.IndexRune(line, '='); i > 0 {
					key = strings.TrimSpace(line[:i])
					val = line[i+1:]
				} else {
					continue
				}
				if key == "" {
					continue
				}
				if _, err := c.SetVar(envID, key, val); err != nil {
					return fmt.Errorf("set %s: %w", key, err)
				}
			}
			return sc.Err()
		},
	}
	cmd.Flags().Int64Var(&envIDFlag, "env-id", 0, "environment ID")
	cmd.Flags().Int64Var(&appIDFlag, "app-id", 0, "application ID (used with --env-name; 0 = all)")
	cmd.Flags().StringVar(&appName, "app-name", "", "application name (used with --env-name)")
	cmd.Flags().StringVar(&envName, "env-name", "", "environment name (used to resolve env ID)")
	return cmd
}
