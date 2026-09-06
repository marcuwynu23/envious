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

func APIKeyAuth(s *storage.Storage) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := c.Request().Header.Get("X-API-Key")
			if key == "" {
				return echo.NewHTTPError(401, "missing API key")
			}
			if !auth.Verify(c.Request().Context(), s, key) {
				return echo.NewHTTPError(401, "invalid API key")
			}
			return next(c)
		}
	}
}

