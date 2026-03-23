package version

import (
	"envious-cli/internal/service"
	"envious-cli/internal/view"
	"github.com/spf13/cobra"
)

// GetDeps returns VersionProvider and VersionRenderer (called at runtime so tests can inject).
type GetDeps func() (service.VersionProvider, *view.VersionRenderer)

// NewCommand returns the version subcommand. getDeps is called at run time so deps are always current.
func NewCommand(getDeps GetDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  `Print the application version, git commit, and build date.`,
		RunE:  runVersion(getDeps),
	}
	return cmd
}

func runVersion(getDeps GetDeps) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		vp, v := getDeps()
		info := vp.GetVersion()
		v.Render(c.OutOrStdout(), info)
		return nil
	}
}
