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
}

func Load() *Config {
	cfg := &Config{
		Port:      getInt("PORT", 8080),
		DBPath:    getenvDefault("DATABASE_PATH", defaultDBPath()),
		LogLevel:  getenvDefault("LOG_LEVEL", "info"),
		LogFormat: strings.ToLower(getenvDefault("LOG_FORMAT", "json")),
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

