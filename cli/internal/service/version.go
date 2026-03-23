package service

import "envious-cli/internal/model"

// VersionProvider returns build-time version information.
type VersionProvider interface {
	GetVersion() model.VersionInfo
}

// VersionService implements VersionProvider using build-time values.
type VersionService struct {
	version   string
	commit    string
	buildDate string
}

// NewVersionService returns a VersionProvider with the given build info.
func NewVersionService(version, commit, buildDate string) *VersionService {
	return &VersionService{
		version:   version,
		commit:    commit,
		buildDate: buildDate,
	}
}

// GetVersion returns the current version info (model only, no I/O).
func (s *VersionService) GetVersion() model.VersionInfo {
	return model.VersionInfo{
		Version:   s.version,
		Commit:    s.commit,
		BuildDate: s.buildDate,
	}
}
