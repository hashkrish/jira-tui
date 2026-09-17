// Package config resolves Jira connection settings from env vars, the OS
// keyring, and a TOML config file, in that priority order.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "jira-tui"
	keyringUser    = "api-token"
)

// Config holds everything needed to authenticate against a Jira instance.
type Config struct {
	BaseURL  string `toml:"base_url"`
	Email    string `toml:"email"`
	APIToken string `toml:"-"`         // never persisted to disk; lives in keyring or env only
	AuthMode string `toml:"auth_mode"` // "basic" (Cloud, email+token) or "bearer" (Data Center PAT)
}

func defaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "jira-tui", "config.toml"), nil
}

// Load resolves configuration in priority order: env vars > keyring-backed
// token with file-stored non-secrets > config file alone.
func Load() (*Config, error) {
	cfg := &Config{AuthMode: "basic"}

	path, err := defaultPath()
	if err == nil {
		if _, statErr := os.Stat(path); statErr == nil {
			if _, decodeErr := toml.DecodeFile(path, cfg); decodeErr != nil {
				return nil, fmt.Errorf("parsing config file %s: %w", path, decodeErr)
			}
		}
	}

	if v := os.Getenv("JIRA_TUI_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("JIRA_TUI_EMAIL"); v != "" {
		cfg.Email = v
	}
	if v := os.Getenv("JIRA_TUI_AUTH_MODE"); v != "" {
		cfg.AuthMode = v
	}

	if v := os.Getenv("JIRA_TUI_API_TOKEN"); v != "" {
		cfg.APIToken = v
	} else if token, kerr := keyring.Get(keyringService, keyringUser); kerr == nil {
		cfg.APIToken = token
	}

	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	return cfg, nil
}

// Save persists non-secret fields to the config file and the API token to
// the OS keyring. If the keyring is unavailable, it falls back to writing
// the token into the config file (best-effort, with a warning returned).
func (c *Config) Save() error {
	path, err := defaultPath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(c); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	if err := keyring.Set(keyringService, keyringUser, c.APIToken); err != nil {
		return fmt.Errorf("keyring unavailable, token not persisted (set JIRA_TUI_API_TOKEN instead): %w", err)
	}
	return nil
}

// Validate checks that the minimum fields needed to make API calls are present.
func (c *Config) Validate() error {
	var missing []string
	if c.BaseURL == "" {
		missing = append(missing, "base_url")
	}
	if c.Email == "" && c.AuthMode == "basic" {
		missing = append(missing, "email")
	}
	if c.APIToken == "" {
		missing = append(missing, "api_token")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	if !strings.HasPrefix(c.BaseURL, "https://") && !strings.HasPrefix(c.BaseURL, "http://") {
		return errors.New("base_url must start with http:// or https://")
	}
	return nil
}
