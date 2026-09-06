package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"envious-web/internal/auth"
	"envious-web/internal/storage"

	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

// RequestIDKey is the response/request header carrying the correlation ID.
const RequestIDKey = "X-Request-ID"

// RequestID assigns every request a random correlation ID: it reuses an
// inbound X-Request-ID (trusted proxies / upstreams) or generates one,
// echoes it back, and stores it in the context for audit logs.
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := c.Request().Header.Get(RequestIDKey)
			if id == "" {
				var b [8]byte
				if _, err := rand.Read(b[:]); err != nil {
					id = "unknown"
				} else {
					id = hex.EncodeToString(b[:])
				}
			}
			c.Response().Header().Set(RequestIDKey, id)
			c.Set(RequestIDKey, id)
			return next(c)
		}
	}
}

// FromContext returns the request correlation ID, or "" outside requests.
func FromContext(c echo.Context) string {
	if v, ok := c.Get(RequestIDKey).(string); ok {
		return v
	}
	return ""
}

func Logging() echo.MiddlewareFunc {
	logger := slog.Default()
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			lat := time.Since(start)
			logger.Info("request",
				"method", c.Request().Method,
				"path", c.Path(),
				"status", c.Response().Status,
				"latency_ms", lat.Milliseconds(),
				"remote_ip", c.RealIP(),
				"request_id", FromContext(c),
			)
			return err
		}
	}
}

func Recovery() echo.MiddlewareFunc {
	return echoMw.Recover()
}

func APIKeyAuth(s *storage.Storage, v *auth.CachedVerifier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-API-Key")
			if key == "" {
				return echo.NewHTTPError(401, "missing API key")
			}
			if v != nil {
				if !v.Verify(c.Request().Context(), s, key) {
					return echo.NewHTTPError(401, "invalid API key")
				}
				return next(c)
			}
			if !auth.Verify(c.Request().Context(), s, key) {
				return echo.NewHTTPError(401, "invalid API key")
			}
			return next(c)
		}
	}
}

// RateLimit caps requests per client IP. Probes and the public version
// endpoint are exempt. An rps <= 0 disables limiting (pass-through).
func RateLimit(rps float64, burst int) echo.MiddlewareFunc {
	if rps <= 0 || burst <= 0 {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error { return next(c) }
		}
	}
	store := echoMw.NewRateLimiterMemoryStoreWithConfig(
		echoMw.RateLimiterMemoryStoreConfig{
			Rate:      rate.Limit(rps),
			Burst:     burst,
			ExpiresIn: 3 * time.Minute,
		},
	)
	return echoMw.RateLimiterWithConfig(echoMw.RateLimiterConfig{
		Store: store,
		Skipper: func(c echo.Context) bool {
			switch c.Path() {
			case "/healthz", "/readyz", "/api/version":
				return true
			}
			return false
		},
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			return echo.NewHTTPError(429, "rate limit exceeded")
		},
	})
}

