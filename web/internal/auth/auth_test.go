package auth_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"envious-web/internal/auth"
	"envious-web/internal/config"
	"envious-web/internal/storage"

	"golang.org/x/crypto/bcrypt"
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

func TestCachedVerifier(t *testing.T) {
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "cache.db")}
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	ctx := context.Background()

	key, err := auth.InitAdminKey(ctx, s)
	if err != nil || key == "" {
		t.Fatalf("init: %v", err)
	}
	cached := auth.NewCachedVerifier(time.Hour)
	if !cached.Verify(ctx, s, key) {
		t.Fatal("cached verify failed")
	}
	if cached.Verify(ctx, s, "wrong") {
		t.Fatal("cached verify accepted wrong key")
	}

	// Rotate the hash: the cache still honors the old key until TTL lapses.
	rotated, _ := bcrypt.GenerateFromPassword([]byte("rotated-key"), bcrypt.MinCost)
	if err := s.SetAPIKeyHash(ctx, string(rotated)); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if !cached.Verify(ctx, s, key) {
		t.Fatal("expected stale cache hit for old key")
	}
	if cached.Verify(ctx, s, "rotated-key") {
		t.Fatal("rotated key must miss while cache is warm")
	}

	// No cache: rotation takes effect immediately.
	fresh := auth.NewCachedVerifier(0)
	if !fresh.Verify(ctx, s, "rotated-key") {
		t.Fatal("uncached verify missed rotated key")
	}
	if fresh.Verify(ctx, s, key) {
		t.Fatal("uncached verify accepted revoked key")
	}
}
