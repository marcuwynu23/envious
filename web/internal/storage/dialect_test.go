package storage_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"envious-web/internal/config"
	"envious-web/internal/storage"
)

// openDialect returns a fresh store for the named backend. Postgres runs only
// when TEST_POSTGRES_URL is set (e.g. a local container); otherwise the case
// is skipped so unit runs stay hermetic.
func openDialect(t *testing.T, dialect string) (*storage.Storage, bool) {
	t.Helper()
	if dialect == storage.DialectPostgres {
		base := os.Getenv("TEST_POSTGRES_URL")
		if base == "" {
			t.Skip("TEST_POSTGRES_URL unset: skipping postgres case")
		}
		s, err := openIsolatedPostgres(t, base)
		if err != nil {
			t.Fatalf("open postgres: %v", err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s, true
	}
	dir := t.TempDir()
	s, err := storage.Open(&config.Config{
		DBPath: filepath.Join(dir, "test.db"),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s, true
}

func pgTestURL() string {
	// Read at call time so the suite stays hermetic by default.
	return os.Getenv("TEST_POSTGRES_URL")
}

// openIsolatedPostgres creates a throwaway database per test so cases never
// share state (and never touch the database named in TEST_POSTGRES_URL).
func openIsolatedPostgres(t *testing.T, base string) (*storage.Storage, error) {
	t.Helper()
	name := fmt.Sprintf("envious_test_%d_%d", time.Now().UnixNano(), os.Getpid())
	name = strings.ReplaceAll(strings.ToLower(name), "-", "_")
	admin, err := sql.Open("pgx", base)
	if err != nil {
		return nil, err
	}
	defer admin.Close()
	if _, err := admin.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(`DROP DATABASE IF EXISTS "` + name + `"`)
	})
	head, query := base, ""
	if i := strings.Index(base, "?"); i >= 0 {
		head, query = base[:i], base[i:]
	}
	slash := strings.LastIndex(head, "/")
	if slash < 0 {
		return nil, fmt.Errorf("bad TEST_POSTGRES_URL %q", base)
	}
	dbURL := head[:slash+1] + name + query
	return storage.Open(&config.Config{
		Driver:      storage.DialectPostgres,
		DatabaseURL: dbURL,
	})
}

func dialects() []string { return []string{storage.DialectSQLite, storage.DialectPostgres} }

// TestDialectFullStack runs the core CRUD + audit contract on every backend.
func TestDialectFullStack(t *testing.T) {
	for _, d := range dialects() {
		t.Run(d, func(t *testing.T) {
			s, _ := openDialect(t, d)
			if s == nil {
				return
			}
			ctx := context.Background()

			appID, err := s.CreateApp(ctx, "dialect-app")
			if err != nil {
				t.Fatalf("create app: %v", err)
			}
			if _, err := s.CreateApp(ctx, "dialect-app"); err != storage.ErrDuplicateKey {
				t.Fatalf("duplicate app = %v, want ErrDuplicateKey", err)
			}
			envID, err := s.CreateEnv(ctx, appID, "prod")
			if err != nil {
				t.Fatalf("create env: %v", err)
			}
			v, err := s.SetVar(ctx, envID, "FOO", "bar")
			if err != nil {
				t.Fatalf("set var: %v", err)
			}
			if v.Version != 1 || v.Value != "bar" {
				t.Fatalf("unexpected var: %+v", v)
			}
			v2, err := s.SetVar(ctx, envID, "FOO", "baz")
			if err != nil {
				t.Fatalf("update var: %v", err)
			}
			if v2.Version != 2 {
				t.Fatalf("expected version 2, got %+v", v2)
			}
			if err := s.LogActivity(ctx, "admin", "var.set", "var", v.ID, "env-key", "127.0.0.1", "r1"); err != nil {
				t.Fatalf("audit: %v", err)
			}
			acts, err := s.ListActivity(ctx, "var.set", 10)
			if err != nil || len(acts) != 1 {
				t.Fatalf("audit list: %v entries %d", err, len(acts))
			}
			if err := s.DeleteEnv(ctx, envID); err != nil {
				t.Fatalf("delete env: %v", err)
			}
			if err := s.DeleteEnv(ctx, envID); err != storage.ErrNotFound {
				t.Fatalf("second delete = %v, want ErrNotFound", err)
			}
		})
	}
}

// TestConcurrentSetVar hammers the same key from N goroutines: versions must
// come out strictly 1..N with no duplicates or lost updates.
func TestConcurrentSetVar(t *testing.T) {
	for _, d := range dialects() {
		t.Run(d, func(t *testing.T) {
			s, _ := openDialect(t, d)
			if s == nil {
				return
			}
			ctx := context.Background()
			appID, err := s.CreateApp(ctx, "race-app")
			if err != nil {
				t.Fatalf("create app: %v", err)
			}
			envID, err := s.CreateEnv(ctx, appID, "race")
			if err != nil {
				t.Fatalf("create env: %v", err)
			}
			const n = 16
			var wg sync.WaitGroup
			errs := make([]error, n)
			for i := 0; i < n; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					_, errs[i] = s.SetVar(ctx, envID, "RACE", fmt.Sprintf("v%d", i))
				}(i)
			}
			wg.Wait()
			for i, err := range errs {
				if err != nil {
					t.Fatalf("writer %d: %v", i, err)
				}
			}
			got, err := s.GetVar(ctx, envID, "RACE")
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got.Version != n {
				t.Fatalf("final version = %d, want %d (lost update?)", got.Version, n)
			}
		})
	}
}
