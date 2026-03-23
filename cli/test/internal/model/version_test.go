package model_test

import (
	"testing"
	"envious-cli/internal/model"
)

func TestVersionInfo_ZeroValue(t *testing.T) {
	var info model.VersionInfo
	if info.Version != "" {
		t.Errorf("zero value Version should be empty")
	}
}
