package version

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
