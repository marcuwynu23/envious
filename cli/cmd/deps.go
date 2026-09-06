package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"envious-cli/internal/client"
	"envious-cli/internal/config"
	"envious-cli/internal/service"
	"envious-cli/internal/view"
)

// Deps holds service and view dependencies for the CLI (injectable for testing).
type Deps struct {
	VersionProvider service.VersionProvider
	VersionView     *view.VersionRenderer
}

// defaultDeps is set in init(); tests may replace it.
var defaultDeps *Deps

func initDeps() {
	if defaultDeps != nil {
		return
	}
	defaultDeps = &Deps{
		VersionProvider: service.NewVersionService(Version, Commit, BuildDate, Author),
		VersionView:     view.NewVersionRenderer(),
	}
}

func deps() *Deps {
	initDeps()
	return defaultDeps
}

// ResetDepsForTest sets defaultDeps to nil so the next deps() call recreates them (e.g. with test Version).
// Only use from tests (e.g. test/cmd).
func ResetDepsForTest() {
	defaultDeps = nil
}

// loadClient loads CLI config and constructs an API client.
// It returns a wrapped error instead of panicking on corrupt config.
func loadClient() (*client.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	c, err := client.New(cfg.APIBase, cfg.APIKey)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// parseID strictly parses a positive integer ID (rejects "12abc", "0", negatives).
func parseID(s string) (int64, error) {
	s = strings.TrimSpace(s)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", s, err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("invalid id %q: must be > 0", s)
	}
	return id, nil
}
