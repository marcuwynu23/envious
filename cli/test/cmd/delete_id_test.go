package cmd_test

import (
	"bytes"
	"testing"

	"envious-cli/cmd"
)

// Regression: strict ID parsing must reject "12abc", "0", negatives
// instead of silently operating on the wrong ID.
func TestDeleteStrictIDParsing(t *testing.T) {
	cases := [][]string{
		{"app", "delete", "12abc"},
		{"app", "delete", "0"},
		{"app", "delete", "-1"},
		{"env", "delete", "12abc"},
		{"variable", "delete", "0"},
	}
	for _, args := range cases {
		root := cmd.RootCmd()
		root.SetOut(&bytes.Buffer{})
		root.SetErr(&bytes.Buffer{})
		root.SetArgs(args)
		if err := root.Execute(); err == nil {
			t.Errorf("%v: expected error, got nil", args)
		}
	}
}
