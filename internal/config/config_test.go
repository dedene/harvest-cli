package config

import (
	"os"
	"testing"
)

func TestReadWriteConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Read non-existent config should return empty
	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}
	if cfg.DefaultAccount != "" {
		t.Errorf("expected empty default account, got %q", cfg.DefaultAccount)
	}

	// Write config
	cfg.DefaultAccount = "test@example.com"
	cfg.AccountAliases = map[string]string{"work": "work@example.com"}
	cfg.WeekStart = "monday"
	cfg.Color = "auto"

	if err := WriteConfig(cfg); err != nil {
		t.Fatalf("WriteConfig() error: %v", err)
	}

	// Verify file exists and has correct permissions
	info, err := os.Stat(ConfigPath())
	if err != nil {
		t.Fatalf("stat config file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file perms = %o, want 0600", info.Mode().Perm())
	}

	// Read back
	cfg2, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}
	if cfg2.DefaultAccount != "test@example.com" {
		t.Errorf("DefaultAccount = %q, want test@example.com", cfg2.DefaultAccount)
	}
	if cfg2.AccountAliases["work"] != "work@example.com" {
		t.Errorf("AccountAliases[work] = %q, want work@example.com", cfg2.AccountAliases["work"])
	}
	if cfg2.WeekStart != "monday" {
		t.Errorf("WeekStart = %q, want monday", cfg2.WeekStart)
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if ConfigExists() {
		t.Error("ConfigExists() = true before creating config")
	}

	if err := WriteConfig(&File{}); err != nil {
		t.Fatalf("WriteConfig() error: %v", err)
	}

	if !ConfigExists() {
		t.Error("ConfigExists() = false after creating config")
	}
}

func TestConfigJSON5(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	EnsureDir()

	// Write JSON5 with comments (simulating manual edit)
	json5Content := `{
  // This is a comment
  "default_account": "user@example.com",
  "week_start": "monday", // trailing comma ok in JSON5
}`
	if err := os.WriteFile(ConfigPath(), []byte(json5Content), 0600); err != nil {
		t.Fatalf("writing json5: %v", err)
	}

	cfg, err := ReadConfig()
	if err != nil {
		t.Fatalf("ReadConfig() error: %v", err)
	}
	if cfg.DefaultAccount != "user@example.com" {
		t.Errorf("DefaultAccount = %q, want user@example.com", cfg.DefaultAccount)
	}
}

func TestInitMaps(t *testing.T) {
	cfg := &File{}
	cfg.initMaps()

	if cfg.AccountAliases == nil {
		t.Error("AccountAliases should be initialized")
	}
	if cfg.AccountClients == nil {
		t.Error("AccountClients should be initialized")
	}
	if cfg.ClientDomains == nil {
		t.Error("ClientDomains should be initialized")
	}
}
