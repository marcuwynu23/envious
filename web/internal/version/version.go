package version

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// Build-time version info (set via ldflags, e.g. make build VERSION=v1.0.0).
// Defaults keep `go run ./cmd/server` working without flags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Info describes the running server build.
type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// Current returns the build info stamped at compile time.
func Current() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}
}

// Describe returns `git describe --tags --always --dirty` for the current
// directory, or "" when git is missing, this is not a repo, or it times out.
// Because of --always it resolves to the tag when available, else the short
// commit hash — so it is never empty inside a checkout.
func Describe() string {
	return describeGit("")
}

func describeGit(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "describe", "--tags", "--always", "--dirty")
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Resolve returns the stamped version, falling back to a live git describe
// so unstamped dev builds still report the current tag.
func Resolve() string {
	if Version != "" && Version != "dev" {
		return Version
	}
	if d := Describe(); d != "" {
		return d
	}
	return "dev"
}
