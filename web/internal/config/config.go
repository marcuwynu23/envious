package config

import (
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port          int
	DBPath        string
	EncryptionKey []byte
	LogLevel      string
	// LogFormat is "json" (default, Fluent Bit / log-collector friendly)
	// or "text" (human-readable local development).
	LogFormat string
	// Driver selects the database backend: "sqlite" (default, single file)
	// or "postgres" (server-based, horizontally scalable behind the API).
	Driver string
	// DatabaseURL is the Postgres connection string (e.g.
	// postgres://user:pass@host:5432/envious?sslmode=require).
	// Ignored unless Driver is "postgres".
	DatabaseURL string
	// Pool tuning for database/sql.
	DBMaxOpenConns int
	DBMaxIdleConns int
	// RateLimitRPS caps requests per client IP (0 disables).
	RateLimitRPS float64
	RateLimitBurst int
	// AuthCacheTTLSeconds caches the API-key bcrypt hash in memory so
	// verification doesn't cost ~100ms of CPU on every request.
	// 0 disables the cache (verify against the DB every time).
	AuthCacheTTLSeconds int
}

func Load() *Config {
	cfg := &Config{
		Port:                getInt("PORT", 8080),
		DBPath:              getenvDefault("DATABASE_PATH", defaultDBPath()),
		LogLevel:            getenvDefault("LOG_LEVEL", "info"),
		LogFormat:           strings.ToLower(getenvDefault("LOG_FORMAT", "json")),
		Driver:              strings.ToLower(getenvDefault("DB_DRIVER", "sqlite")),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		DBMaxOpenConns:      getInt("DB_MAX_OPEN_CONNS", 10),
		DBMaxIdleConns:      getInt("DB_MAX_IDLE_CONNS", 10),
		RateLimitRPS:        getFloat("RATE_LIMIT_RPS", 20),
		RateLimitBurst:      getInt("RATE_LIMIT_BURST", 40),
		AuthCacheTTLSeconds: getInt("AUTH_CACHE_TTL_SECONDS", 60),
	}
	// Optional encryption key for values (hex or raw)
	if key := os.Getenv("ENCRYPTION_KEY"); key != "" {
		cfg.EncryptionKey = []byte(key)
	}
	return cfg
}

func defaultDBPath() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("could not get working directory: %v", err)
		return "envious.db"
	}
	return filepath.Join(wd, "envious.db")
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			return n
		}
	}
	return def
}

// Level parses LogLevel (debug|info|warn|error), defaulting to info.
func (c *Config) Level() slog.Level {
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

