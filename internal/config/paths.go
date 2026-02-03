// Package config provides configuration management for harvest.
package config

import (
	"os"
	"path/filepath"
	"strings"
)

// AppName is the application name used for config directories.
const AppName = "harvest"

// Dir returns the XDG config directory for harvest.
// Falls back to ~/.config/harvest/ if XDG_CONFIG_HOME is not set.
func Dir() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, AppName), nil
}

// EnsureDir creates the config directory with 0700 permissions if it doesn't exist.
func EnsureDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

// ConfigPath returns the path to the main config file.
func ConfigPath() string {
	dir, err := Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "config.json5")
}

// ClientsDir returns the path to the clients subdirectory.
func ClientsDir() string {
	dir, err := Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "clients")
}

// StateDir returns the path to the state subdirectory.
func StateDir() string {
	dir, err := Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "state")
}

// KeyringDir returns the path to the keyring fallback directory.
func KeyringDir() string {
	dir, err := Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "keyring")
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// EnsureClientsDir creates the clients directory with 0700 permissions.
func EnsureClientsDir() error {
	return os.MkdirAll(ClientsDir(), 0700)
}

// EnsureStateDir creates the state directory with 0700 permissions.
func EnsureStateDir() error {
	return os.MkdirAll(StateDir(), 0700)
}

// EnsureKeyringDir creates the keyring directory with 0700 permissions.
func EnsureKeyringDir() error {
	return os.MkdirAll(KeyringDir(), 0700)
}
