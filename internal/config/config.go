// Package config loads, merges, and validates the typed configuration for spotpilot.
// Precedence: flags > env vars > config file > defaults.
package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Timeouts holds bounded-wait durations for various operations.
type Timeouts struct {
	Login         time.Duration `yaml:"login"`
	DesktopDevice time.Duration `yaml:"desktop_device"`
	WebDevice     time.Duration `yaml:"web_device"`
}

// Config is the resolved, typed configuration for the process.
type Config struct {
	Timeouts Timeouts `yaml:"timeouts"`

	// Verbose and Debug are populated from flags; not stored in config file.
	Verbose bool `yaml:"-"`
	Debug   bool `yaml:"-"`
	Plain   bool `yaml:"-"`
}

// defaults returns a Config with conservative default values.
func defaults() Config {
	return Config{
		Timeouts: Timeouts{
			Login:         5 * time.Minute,
			DesktopDevice: 10 * time.Second,
			WebDevice:     15 * time.Second,
		},
	}
}

// fileConfig is a strict-decode intermediate type used to reject unknown keys.
type fileConfig struct {
	Timeouts struct {
		Login         string `yaml:"login"`
		DesktopDevice string `yaml:"desktop_device"`
		WebDevice     string `yaml:"web_device"`
	} `yaml:"timeouts"`
}

// Load reads config from the given file path (empty = default path) and
// applies environment variable overrides. Flags are applied by the caller
// after Load returns.
func Load(configPath string) (Config, error) {
	cfg := defaults()

	path, err := resolvePath(configPath)
	if err != nil {
		return cfg, fmt.Errorf("resolving config path: %w", err)
	}

	if err := loadFile(path, &cfg); err != nil {
		return cfg, err
	}

	applyEnv(&cfg)
	return cfg, nil
}

func resolvePath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	dir, err := defaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// defaultDir returns the XDG-style config directory for spotpilot.
func defaultDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("finding home directory: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "spotpilot"), nil
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // missing default config file is acceptable
	}
	if err != nil {
		return fmt.Errorf("reading config file %s: %w", path, err)
	}

	var fc fileConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true) // reject unknown fields
	if err := dec.Decode(&fc); err != nil {
		return fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if fc.Timeouts.Login != "" {
		d, err := time.ParseDuration(fc.Timeouts.Login)
		if err != nil {
			return fmt.Errorf("invalid timeouts.login: %w", err)
		}
		cfg.Timeouts.Login = d
	}
	if fc.Timeouts.DesktopDevice != "" {
		d, err := time.ParseDuration(fc.Timeouts.DesktopDevice)
		if err != nil {
			return fmt.Errorf("invalid timeouts.desktop_device: %w", err)
		}
		cfg.Timeouts.DesktopDevice = d
	}
	if fc.Timeouts.WebDevice != "" {
		d, err := time.ParseDuration(fc.Timeouts.WebDevice)
		if err != nil {
			return fmt.Errorf("invalid timeouts.web_device: %w", err)
		}
		cfg.Timeouts.WebDevice = d
	}
	return nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("SPOTPILOT_TIMEOUT_LOGIN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeouts.Login = d
		}
	}
	if v := os.Getenv("SPOTPILOT_TIMEOUT_DESKTOP_DEVICE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeouts.DesktopDevice = d
		}
	}
	if v := os.Getenv("SPOTPILOT_TIMEOUT_WEB_DEVICE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Timeouts.WebDevice = d
		}
	}
}
