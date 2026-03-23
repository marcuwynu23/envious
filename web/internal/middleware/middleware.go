package middleware

import (
	"context"
	"log/slog"
	"time"

	"envious-web/internal/auth"
	"envious-web/internal/storage"

	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"
)

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
			if !auth.Verify(context.Background(), s, key) {
				return echo.NewHTTPError(401, "invalid API key")
			}
			return next(c)
		}
	}
}

