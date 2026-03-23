package cmd_test

import (
	"bytes"
	"strings"
	"testing"

	"envious-cli/cmd"
)

func TestVarHelpListsSubcommands(t *testing.T) {
	root := cmd.RootCmd()
	out := bytes.NewBuffer(nil)
	errBuf := bytes.NewBuffer(nil)
	root.SetOut(out)
	root.SetErr(errBuf)
	root.SetArgs([]string{"var", "--help"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("var --help: %v", err)
	}

	s := out.String()
	if !strings.Contains(s, "list") || !strings.Contains(s, "set") || !strings.Contains(s, "delete") {
		t.Errorf("var help should list subcommands: %q", s)
	}
}

