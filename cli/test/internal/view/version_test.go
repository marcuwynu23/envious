package view_test

import (
	"bytes"
	"strings"
	"testing"

	"envious-cli/internal/model"
	"envious-cli/internal/view"
)

func TestVersionRenderer_Render(t *testing.T) {
	r := view.NewVersionRenderer()
	info := model.VersionInfo{Version: "1.0.0", Commit: "abc1234", BuildDate: "2024-01-15"}
	var buf bytes.Buffer
	r.Render(&buf, info)
	s := buf.String()
	if !strings.Contains(s, "version 1.0.0") {
		t.Errorf("output missing version: %q", s)
	}
	if !strings.Contains(s, "commit: abc1234") {
		t.Errorf("output missing commit: %q", s)
	}
	if !strings.Contains(s, "built:  2024-01-15") {
		t.Errorf("output missing build date: %q", s)
	}
}
