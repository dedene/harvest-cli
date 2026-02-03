package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/titanous/json5"
)

// File represents the main configuration file structure.
type File struct {
	DefaultAccount  string            `json:"default_account,omitempty"`
	AccountAliases  map[string]string `json:"account_aliases,omitempty"`
	AccountClients  map[string]string `json:"account_clients,omitempty"`
	ClientDomains   map[string]string `json:"client_domains,omitempty"`
	DefaultTimezone string            `json:"default_timezone,omitempty"`
	WeekStart       string            `json:"week_start,omitempty"`
	Color           string            `json:"color,omitempty"`
	KeyringBackend  string            `json:"keyring_backend,omitempty"`
	ContactEmail    string            `json:"contact_email,omitempty"`
}

// ReadConfig reads and parses the config file.
// Returns an empty config if the file doesn't exist.
func ReadConfig() (*File, error) {
	path := ConfigPath()
	if path == "" {
		return nil, fmt.Errorf("could not determine config path")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{}, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg File
	if err := json5.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// WriteConfig writes the config to disk atomically.
func WriteConfig(cfg *File) error {
	if err := EnsureDir(); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path := ConfigPath()
	if path == "" {
		return fmt.Errorf("could not determine config path")
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return atomicWrite(path, data, 0600)
}

// ConfigExists returns true if the config file exists.
func ConfigExists() bool {
	path := ConfigPath()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// atomicWrite writes data to a temp file then renames it to path.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)

	// Generate random suffix for temp file
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	tmpPath := filepath.Join(dir, ".tmp-"+hex.EncodeToString(randBytes))

	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) // Clean up on failure
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// initMaps ensures all maps in the config are initialized.
func (f *File) initMaps() {
	if f.AccountAliases == nil {
		f.AccountAliases = make(map[string]string)
	}
	if f.AccountClients == nil {
		f.AccountClients = make(map[string]string)
	}
	if f.ClientDomains == nil {
		f.ClientDomains = make(map[string]string)
	}
}
