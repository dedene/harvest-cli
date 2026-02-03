package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	expected := filepath.Join(tmpDir, AppName)
	if dir != expected {
		t.Errorf("Dir() = %q, want %q", dir, expected)
	}
}

func TestDirFallback(t *testing.T) {
	// Test fallback when XDG_CONFIG_HOME is not set
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", AppName)
	if dir != expected {
		t.Errorf("Dir() = %q, want %q", dir, expected)
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	err := EnsureDir()
	if err != nil {
		t.Fatalf("EnsureDir() error: %v", err)
	}

	dir, _ := Dir()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat config dir: %v", err)
	}
	if !info.IsDir() {
		t.Error("config dir is not a directory")
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("config dir perms = %o, want 0700", info.Mode().Perm())
	}
}

func TestConfigPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path := ConfigPath()
	if !strings.HasSuffix(path, "config.json5") {
		t.Errorf("ConfigPath() = %q, want suffix config.json5", path)
	}
}

func TestClientsDirPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path := ClientsDir()
	if !strings.HasSuffix(path, "clients") {
		t.Errorf("ClientsDir() = %q, want suffix clients", path)
	}
}

func TestStateDirPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path := StateDir()
	if !strings.HasSuffix(path, "state") {
		t.Errorf("StateDir() = %q, want suffix state", path)
	}
}

func TestKeyringDirPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	path := KeyringDir()
	if !strings.HasSuffix(path, "keyring") {
		t.Errorf("KeyringDir() = %q, want suffix keyring", path)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"~", home},
		{"~/foo", filepath.Join(home, "foo")},
		{"~/foo/bar", filepath.Join(home, "foo", "bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~notuser", "~notuser"}, // Not a tilde expansion
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExpandPath(tt.input)
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnsureSubDirs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Test EnsureClientsDir
	if err := EnsureClientsDir(); err != nil {
		t.Fatalf("EnsureClientsDir() error: %v", err)
	}
	info, err := os.Stat(ClientsDir())
	if err != nil {
		t.Fatalf("stat clients dir: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("clients dir perms = %o, want 0700", info.Mode().Perm())
	}

	// Test EnsureStateDir
	if err := EnsureStateDir(); err != nil {
		t.Fatalf("EnsureStateDir() error: %v", err)
	}
	info, err = os.Stat(StateDir())
	if err != nil {
		t.Fatalf("stat state dir: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("state dir perms = %o, want 0700", info.Mode().Perm())
	}

	// Test EnsureKeyringDir
	if err := EnsureKeyringDir(); err != nil {
		t.Fatalf("EnsureKeyringDir() error: %v", err)
	}
	info, err = os.Stat(KeyringDir())
	if err != nil {
		t.Fatalf("stat keyring dir: %v", err)
	}
	if info.Mode().Perm() != 0700 {
		t.Errorf("keyring dir perms = %o, want 0700", info.Mode().Perm())
	}
}
