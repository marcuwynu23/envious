package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"envious-cli/cmd"
)

func TestLoginSavesConfig(t *testing.T) {
	// redirect HOME to temp
	tmp := t.TempDir()
	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmp)

	root := cmd.RootCmd()
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"login", "--api-key=abc", "--api-base=http://localhost:8081"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".envious", "config")); err != nil {
		t.Fatalf("config not saved: %v", err)
	}
}
