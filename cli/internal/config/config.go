package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	APIBase string `json:"api_base"`
	APIKey  string `json:"api_key"`
	Output  string `json:"output"` // json|table
}

func Default() *Config {
	return &Config{
		APIBase: "http://127.0.0.1:8080",
		Output:  "table",
	}
}

func Dir() (string, error) {
	// Prefer HOME for testability; fallback to OS user home.
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".envious"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".envious"), nil
}

func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config"), nil
}

func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var cfg Config
	if err := json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

