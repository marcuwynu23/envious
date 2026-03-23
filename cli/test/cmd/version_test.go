package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"envious-cli/cmd"
)

func TestVersionCommand(t *testing.T) {
	origVersion, origCommit, origDate := cmd.Version, cmd.Commit, cmd.BuildDate
	defer func() {
		cmd.Version, cmd.Commit, cmd.BuildDate = origVersion, origCommit, origDate
	}()

	cmd.Version = "1.0.0"
	cmd.Commit = "abc1234"
	cmd.BuildDate = "2024-01-15"
	cmd.ResetDepsForTest()

	root := cmd.RootCmd()
	out := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"version"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("version execute: %v", err)
	}

	s := out.String()
	if s == "" {
		t.Fatal("version produced no output")
	}
	if !strings.Contains(s, "1.0.0") {
		t.Errorf("output missing version: %q", s)
	}
	if !strings.Contains(s, "abc1234") {
		t.Errorf("output missing commit: %q", s)
	}
}
