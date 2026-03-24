package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"time"
	"envious-web/internal/middleware"
	"envious-web/internal/storage"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type Server struct {
	E      *echo.Echo
	Store  *storage.Storage
	secret []byte
}

func New(store *storage.Storage, secret []byte) *Server {
	e := echo.New()
	s := &Server{E: e, Store: store, secret: secret}
	// Middlewares
	e.Use(middleware.Logging())
	e.Use(middleware.Recovery())
	// Templates
	e.Renderer = &TemplateRegistry{templates: template.Must(template.ParseFS(templatesFS, "templates/*.html"))}
	// Routes
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	e := s.E
	e.POST("/login", s.handleLogin)
	e.POST("/logout", s.handleLogout)

	// Admin dashboard
	e.GET("/", s.requireSession(s.handleAdminApps))
	e.POST("/apps", s.requireSession(s.handleAdminCreateApp))
	e.POST("/apps/:id/delete", s.requireSession(s.handleAdminDeleteApp))
	e.GET("/apps/:id", s.requireSession(s.handleAdminApp))
	e.POST("/apps/:id/envs", s.requireSession(s.handleAdminCreateEnv))
	e.POST("/apps/:id/envs/:envID/delete", s.requireSession(s.handleAdminDeleteEnv))
	e.GET("/apps/:appID/envs/:envID", s.requireSession(s.handleAdminEnv))
	e.POST("/apps/:appID/envs/:envID/vars", s.requireSession(s.handleAdminCreateVar))
	e.POST("/apps/:appID/envs/:envID/vars/:key/delete", s.requireSession(s.handleAdminDeleteVar))
	e.POST("/apps/:appID/envs/:envID/vars/:key/update", s.requireSession(s.handleAdminUpdateVar))

	// API (header X-API-Key required)
	api := e.Group("/api")
	api.Use(middleware.APIKeyAuth(s.Store))
	api.GET("/apps", s.handleListApps)
	api.POST("/apps", s.handleCreateApp)
	api.GET("/apps/:id", s.handleGetApp)
	api.DELETE("/apps/:id", s.handleDeleteApp)
	api.GET("/apps/:id/envs", s.handleListEnvsByApp)
	api.POST("/apps/:id/envs", s.handleCreateEnvInApp)
	api.GET("/envs", s.handleListEnvs)
	api.POST("/envs", s.handleCreateEnv)
	api.GET("/envs/:id", s.handleGetEnv)
	api.PUT("/envs/:id", s.handleUpdateEnv) // reserved for rename
	api.DELETE("/envs/:id", s.handleDeleteEnv)
	api.GET("/envs/:id/vars", s.handleListVars)
	api.POST("/envs/:id/vars", s.handleSetVar)
	api.PUT("/vars/:id", s.handleUpdateVarByID)
	api.DELETE("/vars/:id", s.handleDeleteVarByID)
}

// Template renderer
type TemplateRegistry struct {
	templates *template.Template
}

func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// Session helpers
func (s *Server) requireSession(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !s.isAuthed(c) {
			return c.Render(http.StatusOK, "login.html", map[string]any{"Title": "Envious - Login", "Error": ""})
		}
		return next(c)
	}
}

func (s *Server) isAuthed(c echo.Context) bool {
	cookie, err := c.Cookie("envious_auth")
	if err != nil || cookie == nil {
		return false
	}
	return s.verifySig(cookie.Value)
}

func (s *Server) sign(v string) string {
	m := hmac.New(sha256.New, s.secret)
	m.Write([]byte(v))
	return hex.EncodeToString(m.Sum(nil))
}

func (s *Server) verifySig(sig string) bool {
	expected := s.sign("ok")
	if len(sig) != len(expected) {
		return false
	}
	var res byte
	for i := 0; i < len(sig); i++ {
		res |= sig[i] ^ expected[i]
	}
	return res == 0
}

// Handlers
func (s *Server) handleLogin(c echo.Context) error {
	var body struct {
		APIKey string `json:"api_key" form:"api_key"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid body"})
	}
	hash, err := s.Store.GetAPIKeyHash(context.Background())
	ok := (err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.APIKey)) == nil)
	if !ok {
		if c.Request().Header.Get("Content-Type") == "application/json" {
			return c.JSON(401, map[string]string{"error": "invalid api key"})
		}
		return c.Render(200, "login.html", map[string]any{"Error": "Invalid API key"})
	}
	cookie := &http.Cookie{
		Name:     "envious_auth",
		Value:    s.sign("ok"),
		Path:     "/",
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(24 * time.Hour),
	}
	c.SetCookie(cookie)
	if c.Request().Header.Get("Content-Type") == "application/json" {
		return c.JSON(200, map[string]string{"status": "ok"})
	}
	return c.Redirect(302, "/")
}

func (s *Server) handleLogout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:     "envious_auth",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return c.Redirect(302, "/")
}

func (s *Server) handleListApps(c echo.Context) error {
	apps, err := s.Store.ListApps(c.Request().Context())
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, apps)
}

func (s *Server) handleCreateApp(c echo.Context) error {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}
	id, err := s.Store.CreateApp(c.Request().Context(), body.Name)
	if err != nil {
		code := 500
		if err == storage.ErrDuplicateKey {
			code = 409
		}
		return c.JSON(code, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, map[string]any{"id": id, "name": body.Name})
}

func (s *Server) handleGetApp(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	app, err := s.Store.GetApp(c.Request().Context(), id)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.JSON(404, map[string]string{"error": "not found"})
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, app)
}

func (s *Server) handleDeleteApp(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	if err := s.Store.DeleteApp(c.Request().Context(), id); err != nil {
		return c.JSON(400, map[string]string{"error": err.Error()})
	}
	return c.NoContent(204)
}

func (s *Server) handleListEnvsByApp(c echo.Context) error {
	appID, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	envs, err := s.Store.ListEnvs(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, envs)
}

func (s *Server) handleCreateEnvInApp(c echo.Context) error {
	appID, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}
	id, err := s.Store.CreateEnv(c.Request().Context(), appID, body.Name)
	if err != nil {
		code := 500
		if err == storage.ErrDuplicateKey {
			code = 409
		}
		return c.JSON(code, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, map[string]any{"id": id, "app_id": appID, "name": body.Name})
}

func (s *Server) handleListEnvs(c echo.Context) error {
	var appID int64
	if v := c.QueryParam("app_id"); v != "" {
		if parsed, err := parseInt64(v); err == nil {
			appID = parsed
		}
	}
	envs, err := s.Store.ListEnvs(c.Request().Context(), appID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, envs)
}

func (s *Server) handleCreateEnv(c echo.Context) error {
	var body struct {
		AppID int64  `json:"app_id"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Name == "" {
		return c.JSON(400, map[string]string{"error": "name required"})
	}
	id, err := s.Store.CreateEnv(c.Request().Context(), body.AppID, body.Name)
	if err != nil {
		code := 500
		if err == storage.ErrDuplicateKey {
			code = 409
		}
		return c.JSON(code, map[string]string{"error": err.Error()})
	}
	return c.JSON(201, map[string]any{"id": id, "app_id": body.AppID, "name": body.Name})
}

func (s *Server) handleGetEnv(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	en, err := s.Store.GetEnv(c.Request().Context(), id)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.JSON(404, map[string]string{"error": "not found"})
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, en)
}

func (s *Server) handleUpdateEnv(c echo.Context) error {
	return c.NoContent(501)
}

func (s *Server) handleDeleteEnv(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	if err := s.Store.DeleteEnv(c.Request().Context(), id); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.NoContent(204)
}

func (s *Server) handleListVars(c echo.Context) error {
	envID, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	vars, err := s.Store.ListVars(c.Request().Context(), envID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, vars)
}

func (s *Server) handleSetVar(c echo.Context) error {
	envID, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil || body.Key == "" {
		return c.JSON(400, map[string]string{"error": "key and value required"})
	}
	v, err := s.Store.SetVar(c.Request().Context(), envID, body.Key, body.Value)
	if err != nil {
		code := 500
		if err == storage.ErrDuplicateKey {
			code = 409
		}
		return c.JSON(code, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, v)
}

func (s *Server) handleUpdateVarByID(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
		return c.JSON(400, map[string]string{"error": "value required"})
	}
	v, err := s.Store.UpdateVar(c.Request().Context(), id, body.Value)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.JSON(404, map[string]string{"error": "not found"})
		}
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.JSON(200, v)
}

func (s *Server) handleDeleteVarByID(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return c.JSON(400, map[string]string{"error": "invalid id"})
	}
	envID, key, err := s.Store.GetVarMetaByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(404, map[string]string{"error": "not found"})
	}
	if err := s.Store.DeleteVar(c.Request().Context(), envID, key); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	return c.NoContent(204)
}

// Admin handlers
func (s *Server) handleAdminApps(c echo.Context) error {
	apps, _ := s.Store.ListApps(c.Request().Context())
	return c.Render(200, "apps.html", map[string]any{"Apps": apps})
}

func (s *Server) handleAdminCreateApp(c echo.Context) error {
	name := c.FormValue("name")
	if name == "" {
		return c.Redirect(302, "/")
	}
	_, _ = s.Store.CreateApp(c.Request().Context(), name)
	return c.Redirect(302, "/")
}

func (s *Server) handleAdminDeleteApp(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err == nil {
		_ = s.Store.DeleteApp(c.Request().Context(), id)
	}
	return c.Redirect(302, "/")
}

func (s *Server) handleAdminApp(c echo.Context) error {
	appID, err := parseIDParam(c, "id")
	if err != nil {
		return c.Render(400, "error.html", map[string]any{"Error": "invalid id"})
	}
	app, err := s.Store.GetApp(c.Request().Context(), appID)
	if err != nil {
		return c.Render(404, "error.html", map[string]any{"Error": "not found"})
	}
	envs, _ := s.Store.ListEnvs(c.Request().Context(), appID)
	return c.Render(200, "app_envs.html", map[string]any{
		"App":  app,
		"Envs": envs,
		"Error": "",
	})
}

func (s *Server) handleAdminEnv(c echo.Context) error {
	envID, err := parseIDParam(c, "envID")
	if err != nil {
		return c.Render(400, "error.html", map[string]any{"Error": "invalid id"})
	}
	en, err := s.Store.GetEnv(c.Request().Context(), envID)
	if err != nil {
		return c.Render(404, "error.html", map[string]any{"Error": "not found"})
	}
	vars, _ := s.Store.ListVars(c.Request().Context(), envID)
	app, _ := s.Store.GetApp(c.Request().Context(), en.AppID)
	return c.Render(200, "vars.html", map[string]any{
		"Env":  en,
		"Vars": vars,
		"App":  app,
	})
}

func (s *Server) handleAdminCreateEnv(c echo.Context) error {
	appID, err := parseIDParam(c, "id")
	if err != nil {
		return c.Redirect(302, "/")
	}
	name := c.FormValue("name")
	name = strings.TrimSpace(name)
	if name == "" {
		app, _ := s.Store.GetApp(c.Request().Context(), appID)
		envs, _ := s.Store.ListEnvs(c.Request().Context(), appID)
		return c.Render(200, "app_envs.html", map[string]any{
			"App":   app,
			"Envs":  envs,
			"Error": "Environment name is required",
		})
	}
	if _, err := s.Store.CreateEnv(c.Request().Context(), appID, name); err != nil {
		app, _ := s.Store.GetApp(c.Request().Context(), appID)
		envs, _ := s.Store.ListEnvs(c.Request().Context(), appID)
		msg := err.Error()
		if err == storage.ErrDuplicateKey {
			msg = "Environment already exists in this application"
		}
		return c.Render(200, "app_envs.html", map[string]any{
			"App":   app,
			"Envs":  envs,
			"Error": msg,
		})
	}
	return c.Redirect(302, "/apps/"+c.Param("id"))
}

func (s *Server) handleAdminDeleteEnv(c echo.Context) error {
	id, err := parseIDParam(c, "envID")
	if err == nil {
		_ = s.Store.DeleteEnv(c.Request().Context(), id)
	}
	return c.Redirect(302, "/apps/"+c.Param("id"))
}

func (s *Server) handleAdminCreateVar(c echo.Context) error {
	envID, err := parseIDParam(c, "envID")
	if err != nil {
		return c.Redirect(302, "/")
	}
	key := c.FormValue("key")
	val := c.FormValue("value")
	if key != "" {
		_, _ = s.Store.SetVar(c.Request().Context(), envID, key, val)
	}
	return c.Redirect(302, "/apps/"+c.Param("appID")+"/envs/"+c.Param("envID"))
}

func (s *Server) handleAdminDeleteVar(c echo.Context) error {
	envID, err := parseIDParam(c, "envID")
	if err == nil {
		key := c.Param("key")
		if key != "" {
			_ = s.Store.DeleteVar(c.Request().Context(), envID, key)
		}
	}
	return c.Redirect(302, "/apps/"+c.Param("appID")+"/envs/"+c.Param("envID"))
}

func (s *Server) handleAdminUpdateVar(c echo.Context) error {
	envID, err := parseIDParam(c, "envID")
	if err != nil {
		return c.Redirect(302, "/")
	}
	key := c.Param("key")
	val := c.FormValue("value")
	if key != "" {
		_, _ = s.Store.SetVar(c.Request().Context(), envID, key, val)
	}
	return c.Redirect(302, "/apps/"+c.Param("appID")+"/envs/"+c.Param("envID"))
}

func parseIDParam(c echo.Context, name string) (int64, error) {
	return parseInt64(c.Param(name))
}

func parseInt64(s string) (int64, error) {
	var x int64
	_, err := fmt.Sscan(s, &x)
	return x, err
}

