package service_test

import (
	"testing"
	"envious-cli/internal/model"
	"envious-cli/internal/service"
)

func TestVersionService_GetVersion(t *testing.T) {
	svc := service.NewVersionService("1.0.0", "abc1234", "2024-01-15")
	info := svc.GetVersion()
	if info.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", info.Version)
	}
	if info.Commit != "abc1234" {
		t.Errorf("Commit = %q, want abc1234", info.Commit)
	}
	if info.BuildDate != "2024-01-15" {
		t.Errorf("BuildDate = %q, want 2024-01-15", info.BuildDate)
	}
}

func TestVersionService_GetVersion_ReturnsModel(t *testing.T) {
	svc := service.NewVersionService("x", "y", "z")
	info := svc.GetVersion()
	_ = model.VersionInfo(info)
	if info.Version != "x" {
		t.Fail()
	}
}
