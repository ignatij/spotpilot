package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ignatij/spotpilot/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	// Point to a non-existent path — missing default config is acceptable.
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timeouts.Login == 0 {
		t.Error("expected non-zero default login timeout")
	}
}

func TestLoad_UnknownField(t *testing.T) {
	f := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(f, []byte("unknown_field: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(f)
	if err == nil {
		t.Error("expected error for unknown YAML field")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("SPOTPILOT_TIMEOUT_LOGIN", "1m")
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Timeouts.Login.Minutes() != 1 {
		t.Errorf("expected 1m login timeout, got %v", cfg.Timeouts.Login)
	}
}
