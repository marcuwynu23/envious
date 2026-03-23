package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"envious-cli/cmd"
)

func TestEnvHelpListsSubcommands(t *testing.T) {
	root := cmd.RootCmd()
	out := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"env", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("env --help: %v", err)
	}

	s := out.String()
	if !strings.Contains(s, "list") || !strings.Contains(s, "create") || !strings.Contains(s, "delete") {
		t.Errorf("env help should list subcommands: %q", s)
	}
}

