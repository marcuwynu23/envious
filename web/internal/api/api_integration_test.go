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
	t.Cleanup(func() { _ = s.Close() })
	return srv, key
}

func TestAPIVersion(t *testing.T) {
	server, _ := newTestServer(t)

	// Unset version defaults to "dev" and needs no auth.
	req := httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec := httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var unset map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&unset); err != nil || unset["version"] != "dev" {
		t.Fatalf("expected default version dev, got %v (err %v)", unset, err)
	}

	// Stamped version is served as-is.
	server.Version = "v1.0.0"
	req = httptest.NewRequest(http.MethodGet, "/api/version", nil)
	rec = httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil || got["version"] != "v1.0.0" {
		t.Fatalf("expected version v1.0.0, got %v (err %v)", got, err)
	}
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

func TestAPIInvalidIDAndNotFound(t *testing.T) {
	server, key := newTestServer(t)

	cases := []struct {
		name       string
		method     string
		target     string
		wantStatus int
	}{
		{"get app trailing garbage", http.MethodGet, "/api/apps/12abc", 400},
		{"get app zero", http.MethodGet, "/api/apps/0", 400},
		{"get app negative", http.MethodGet, "/api/apps/-1", 400},
		{"get missing app", http.MethodGet, "/api/apps/9999", 404},
		{"delete missing app", http.MethodDelete, "/api/apps/9999", 404},
		{"delete missing env", http.MethodDelete, "/api/envs/9999", 404},
		{"delete missing var", http.MethodDelete, "/api/vars/9999", 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.target, nil)
			req.Header.Set("X-API-Key", key)
			rec := httptest.NewRecorder()
			server.E.ServeHTTP(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("%s %s = %d, want %d: %s", tc.method, tc.target, rec.Code, tc.wantStatus, rec.Body.String())
			}
		})
	}

	// Missing API key → 401.
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	rec := httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("missing key = %d, want 401", rec.Code)
	}
}
