package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port          int
	DBPath        string
	EncryptionKey []byte
	LogLevel      string
}

func Load() *Config {
	cfg := &Config{
		Port:     getInt("PORT", 8080),
		DBPath:   getenvDefault("DATABASE_PATH", defaultDBPath()),
		LogLevel: getenvDefault("LOG_LEVEL", "info"),
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

