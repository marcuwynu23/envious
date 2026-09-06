package storage_test

import (
	"context"
	"path/filepath"
	"testing"

	"envious-web/internal/config"
	"envious-web/internal/storage"
)

func tempCfg(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		DBPath:        filepath.Join(dir, "test.db"),
		EncryptionKey: []byte("test-key-32bytes-length-for-aes-256-xyz"),
		Port:          0,
	}
}

func TestEnvAndVarCRUD(t *testing.T) {
	cfg := tempCfg(t)
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	appID, err := s.CreateApp(ctx, "myapp")
	if err != nil {
		t.Fatalf("create app: %v", err)
	}
	envID, err := s.CreateEnv(ctx, appID, "dev")
	if err != nil {
		t.Fatalf("create env: %v", err)
	}
	if envID == 0 {
		t.Fatalf("expected env id > 0")
	}

	if _, err := s.SetVar(ctx, envID, "FOO", "bar"); err != nil {
		t.Fatalf("set var: %v", err)
	}
	v, err := s.GetVar(ctx, envID, "FOO")
	if err != nil {
		t.Fatalf("get var: %v", err)
	}
	if v.Value != "bar" || v.Version != 1 {
		t.Fatalf("unexpected var: %+v", v)
	}
	if _, err := s.SetVar(ctx, envID, "FOO", "baz"); err != nil {
		t.Fatalf("update var: %v", err)
	}
	v2, _ := s.GetVar(ctx, envID, "FOO")
	if v2.Value != "baz" || v2.Version != 2 {
		t.Fatalf("unexpected var: %+v", v2)
	}
	if err := s.DeleteVar(ctx, envID, "FOO"); err != nil {
		t.Fatalf("delete var: %v", err)
	}
}

func TestDeleteNotFound(t *testing.T) {
	cfg := tempCfg(t)
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	if err := s.DeleteApp(ctx, 9999); err != storage.ErrNotFound {
		t.Fatalf("DeleteApp missing = %v, want ErrNotFound", err)
	}
	if err := s.DeleteEnv(ctx, 9999); err != storage.ErrNotFound {
		t.Fatalf("DeleteEnv missing = %v, want ErrNotFound", err)
	}
}

func TestActivityLog(t *testing.T) {
	cfg := tempCfg(t)
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	// Empty trail reads as empty, not an error.
	acts, err := s.ListActivity(ctx, "", 0)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}
	if len(acts) != 0 {
		t.Fatalf("expected no entries, got %d", len(acts))
	}

	if err := s.LogActivity(ctx, "admin", "app.create", "app", 2, "name=myapp", "127.0.0.1", "req-1"); err != nil {
		t.Fatalf("log 1: %v", err)
	}
	if err := s.LogActivity(ctx, "admin", "var.set", "var", 7, "env=3 key=FOO", "127.0.0.1", "req-2"); err != nil {
		t.Fatalf("log 2: %v", err)
	}

	acts, err = s.ListActivity(ctx, "", 0)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(acts) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(acts))
	}
	if acts[0].Action != "var.set" {
		t.Fatalf("newest first: got %q", acts[0].Action)
	}
	if acts[1].Actor != "admin" || acts[1].ResourceID != 2 || acts[1].Detail != "name=myapp" {
		t.Fatalf("unexpected entry: %+v", acts[1])
	}

	filtered, err := s.ListActivity(ctx, "app.create", 10)
	if err != nil || len(filtered) != 1 {
		t.Fatalf("filter app.create: %v entries %d", err, len(filtered))
	}
	limited, err := s.ListActivity(ctx, "", 1)
	if err != nil || len(limited) != 1 {
		t.Fatalf("limit 1: %v entries %d", err, len(limited))
	}
}
