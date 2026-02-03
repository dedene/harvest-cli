package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/titanous/json5"
)

// DefaultClientName is the name of the default OAuth client.
const DefaultClientName = "default"

// NormalizeClientNameOrDefault normalizes a client name, defaulting to "default" if empty.
func NormalizeClientNameOrDefault(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSpace(raw))
	if name == "" {
		return DefaultClientName, nil
	}

	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("invalid client name: %q", raw)
	}

	return name, nil
}

// ClientCredentials holds OAuth client credentials.
type ClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
}

// ClientCredentialsPath returns the path for a client's credentials file.
func ClientCredentialsPath(client string) string {
	if client == "" {
		client = DefaultClientName
	}
	return filepath.Join(ClientsDir(), client+".json5")
}

// ReadClientCredentials reads credentials for the specified client.
func ReadClientCredentials(client string) (*ClientCredentials, error) {
	if client == "" {
		client = DefaultClientName
	}

	path := ClientCredentialsPath(client)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("client %q not found: %w", client, err)
		}
		return nil, fmt.Errorf("reading client credentials: %w", err)
	}

	var creds ClientCredentials
	if err := json5.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing client credentials: %w", err)
	}

	if creds.ClientID == "" {
		return nil, fmt.Errorf("client %q has no client_id", client)
	}
	if creds.ClientSecret == "" {
		return nil, fmt.Errorf("client %q has no client_secret", client)
	}

	return &creds, nil
}

// WriteClientCredentials writes credentials for the specified client.
func WriteClientCredentials(client string, creds *ClientCredentials) error {
	if client == "" {
		client = DefaultClientName
	}
	if creds == nil {
		return fmt.Errorf("credentials cannot be nil")
	}
	if creds.ClientID == "" {
		return fmt.Errorf("client_id is required")
	}
	if creds.ClientSecret == "" {
		return fmt.Errorf("client_secret is required")
	}

	if err := EnsureClientsDir(); err != nil {
		return fmt.Errorf("creating clients dir: %w", err)
	}

	path := ClientCredentialsPath(client)
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	return atomicWrite(path, data, 0600)
}

// ListClients returns a list of configured client names.
func ListClients() ([]string, error) {
	dir := ClientsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading clients dir: %w", err)
	}

	var clients []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json5") {
			clients = append(clients, strings.TrimSuffix(name, ".json5"))
		}
	}
	return clients, nil
}

// ClientCredentialsExist returns true if credentials exist for the client.
func ClientCredentialsExist(client string) bool {
	if client == "" {
		client = DefaultClientName
	}
	path := ClientCredentialsPath(client)
	_, err := os.Stat(path)
	return err == nil
}

// DeleteClientCredentials removes credentials for the specified client.
func DeleteClientCredentials(client string) error {
	if client == "" {
		return fmt.Errorf("client name required")
	}
	path := ClientCredentialsPath(client)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("client %q not found", client)
		}
		return fmt.Errorf("deleting client credentials: %w", err)
	}
	return nil
}
