package version

import (
	"strings"
	"testing"
)

func TestDescribeOutsideRepo(t *testing.T) {
	if got := describeGit(t.TempDir()); got != "" {
		t.Fatalf("describeGit(temp dir) = %q, want empty", got)
	}
}

func TestResolveNeverEmpty(t *testing.T) {
	if got := Resolve(); strings.TrimSpace(got) == "" {
		t.Fatalf("Resolve() is empty")
	}
}
