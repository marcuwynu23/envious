package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
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
	srv := api.New(s, []byte("secret"), api.Options{})
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

func TestAdminAbout(t *testing.T) {
	server, key := newTestServer(t)
	server.Version = "v1.0.0"

	// Log in through the form to obtain the session cookie.
	form := strings.NewReader("api_key=" + key)
	req := httptest.NewRequest(http.MethodPost, "/login", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 302 {
		t.Fatalf("login = %d, want 302: %s", rec.Code, rec.Body.String())
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatalf("login set no cookies")
	}

	// About page requires the session and shows build info.
	req = httptest.NewRequest(http.MethodGet, "/about", nil)
	for _, ck := range cookies {
		req.AddCookie(ck)
	}
	rec = httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("about = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"About Envious", "v1.0.0", "Git tag"} {
		if !strings.Contains(body, want) {
			t.Fatalf("about page missing %q", want)
		}
	}

	// Without a session the about page redirects to login.
	req = httptest.NewRequest(http.MethodGet, "/about", nil)
	rec = httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Login") {
		t.Fatalf("anonymous about = %d, want login page", rec.Code)
	}
}

func TestAPIActivityAudit(t *testing.T) {
	server, key := newTestServer(t)

	// Mutate through the API.
	body, _ := json.Marshal(map[string]string{"name": "audited"})
	req := httptest.NewRequest(http.MethodPost, "/api/apps", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", key)
	req.Header.Set("X-Request-ID", "test-req-1")
	rec := httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("create app = %d: %s", rec.Code, rec.Body.String())
	}

	// The audit trail records it (actor + metadata, never values).
	req = httptest.NewRequest(http.MethodGet, "/api/activity?action=app.create", nil)
	req.Header.Set("X-API-Key", key)
	rec = httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("activity = %d: %s", rec.Code, rec.Body.String())
	}
	var acts []map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&acts); err != nil || len(acts) != 1 {
		t.Fatalf("expected 1 entry, got %v (err %v)", acts, err)
	}
	if acts[0]["actor"] != "admin" || acts[0]["action"] != "app.create" {
		t.Fatalf("unexpected entry: %v", acts[0])
	}
	if acts[0]["request_id"] != "test-req-1" {
		t.Fatalf("request id not propagated: %v", acts[0])
	}

	// Activity endpoint itself requires auth.
	req = httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	rec = httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("anonymous activity = %d, want 401", rec.Code)
	}
}

func TestHealthReady(t *testing.T) {
	server, _ := newTestServer(t)

	// Probes need no auth.
	for _, target := range []string{"/healthz", "/readyz"} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		server.E.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s = %d, want 200: %s", target, rec.Code, rec.Body.String())
		}
	}
	var ready map[string]string
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	server.E.ServeHTTP(rec, req)
	if err := json.NewDecoder(rec.Body).Decode(&ready); err != nil || ready["dialect"] == "" {
		t.Fatalf("readyz missing dialect: %v (err %v)", ready, err)
	}
}

func TestRateLimit(t *testing.T) {
	cfg := &config.Config{DBPath: filepath.Join(t.TempDir(), "rate.db")}
	s, err := storage.Open(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()
	key, err := auth.InitAdminKey(context.Background(), s)
	if err != nil || key == "" {
		t.Fatalf("init auth: %v", err)
	}
	server := api.New(s, []byte("secret"), api.Options{RateRPS: 1, RateBurst: 1})

	get := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
		req.Header.Set("X-API-Key", key)
		rec := httptest.NewRecorder()
		server.E.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := get(); code != 200 {
		t.Fatalf("first = %d, want 200", code)
	}
	if code := get(); code != 429 {
		t.Fatalf("immediate second = %d, want 429", code)
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
