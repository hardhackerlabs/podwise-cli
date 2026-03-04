package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const defaultAPIBaseURL = "https://podwise.ai/api"

// Config holds all application-level settings.
//
// Priority (highest → lowest):
//
//	environment variables > config file > built-in defaults
type Config struct {
	APIKey     string `toml:"api_key"`
	APIBaseURL string `toml:"api_base_url"`
}

// Load returns the effective Config by merging the config file and
// environment variables. Env vars always win over the file.
func Load() (*Config, error) {
	cfg := &Config{
		APIBaseURL: defaultAPIBaseURL,
	}

	if err := loadFile(cfg); err != nil {
		return nil, err
	}

	loadEnv(cfg)

	return cfg, nil
}

// Save writes cfg to the config file, creating the directory if needed.
func Save(cfg *Config) error {
	path, err := filePath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}

// FilePath returns the absolute path of the config file.
func FilePath() (string, error) {
	return filePath()
}

// Validate returns an error when required fields are missing.
func Validate(cfg *Config) error {
	if cfg.APIKey == "" {
		return errors.New("API key is not set — run: podwise config set api_key <your-key>")
	}
	return nil
}

// loadFile reads the TOML config file into cfg.
// Missing file is not an error (first-run case).
func loadFile(cfg *Config) error {
	path, err := filePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil // no config file yet — use defaults
	}

	_, err = toml.DecodeFile(path, cfg)
	return err
}

// loadEnv overrides cfg fields with environment variables when set.
func loadEnv(cfg *Config) {
	if v := os.Getenv("PODWISE_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("PODWISE_API_URL"); v != "" {
		cfg.APIBaseURL = v
	}
}

func filePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "podwise", "config.toml"), nil
}
