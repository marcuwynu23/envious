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
