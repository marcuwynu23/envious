package cmd

import (
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
		VersionProvider: service.NewVersionService(Version, Commit, BuildDate),
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
