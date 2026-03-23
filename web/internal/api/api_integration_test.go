package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"envious-web/internal/api"
	"envious-web/internal/auth"
	"envious-web/internal/config"
	"envious-web/internal/storage"
)

func newTestServer(t *testing.T) (*api.Server, string) {
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "api.db")}
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	key, err := auth.InitAdminKey(context.Background(), s)
	if err != nil || key == "" {
		t.Fatalf("init auth: %v", err)
	}
	srv := api.New(s, []byte("secret"))
	return srv, key
}

func TestAPIEnvCRUD(t *testing.T) {
	server, key := newTestServer(t)

	// Create env
	body, _ := json.Marshal(map[string]string{"name": "dev"})
	req := httptest.NewRequest(http.MethodPost, "/api/envs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", key)
	rec := httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}
