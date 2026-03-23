package auth_test

import (
	"context"
	"path/filepath"
	"testing"

	"envious-web/internal/auth"
	"envious-web/internal/config"
	"envious-web/internal/storage"
)

func TestInitAndVerify(t *testing.T) {
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "auth.db")}
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	ctx := context.Background()
	key, err := auth.InitAdminKey(ctx, s)
	if err != nil || key == "" {
		t.Fatalf("init: key=%q err=%v", key, err)
	}
	if !auth.Verify(ctx, s, key) {
		t.Fatal("verify failed")
	}
}
