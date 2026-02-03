package config

import (
	"os"
	"testing"
)

func TestReadWriteClientCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := &ClientCredentials{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:8080/callback",
	}

	// Write credentials
	if err := WriteClientCredentials("testclient", creds); err != nil {
		t.Fatalf("WriteClientCredentials() error: %v", err)
	}

	// Verify file permissions
	path := ClientCredentialsPath("testclient")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("credentials file perms = %o, want 0600", info.Mode().Perm())
	}

	// Read back
	creds2, err := ReadClientCredentials("testclient")
	if err != nil {
		t.Fatalf("ReadClientCredentials() error: %v", err)
	}
	if creds2.ClientID != creds.ClientID {
		t.Errorf("ClientID = %q, want %q", creds2.ClientID, creds.ClientID)
	}
	if creds2.ClientSecret != creds.ClientSecret {
		t.Errorf("ClientSecret = %q, want %q", creds2.ClientSecret, creds.ClientSecret)
	}
	if creds2.RedirectURI != creds.RedirectURI {
		t.Errorf("RedirectURI = %q, want %q", creds2.RedirectURI, creds.RedirectURI)
	}
}

func TestDefaultClientName(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := &ClientCredentials{
		ClientID:     "default-id",
		ClientSecret: "default-secret",
	}

	// Empty string should use default
	if err := WriteClientCredentials("", creds); err != nil {
		t.Fatalf("WriteClientCredentials() error: %v", err)
	}

	creds2, err := ReadClientCredentials("")
	if err != nil {
		t.Fatalf("ReadClientCredentials() error: %v", err)
	}
	if creds2.ClientID != "default-id" {
		t.Errorf("ClientID = %q, want default-id", creds2.ClientID)
	}
}

func TestClientCredentialsExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if ClientCredentialsExist("nonexistent") {
		t.Error("ClientCredentialsExist() = true for nonexistent client")
	}

	creds := &ClientCredentials{ClientID: "id", ClientSecret: "secret"}
	WriteClientCredentials("exists", creds)

	if !ClientCredentialsExist("exists") {
		t.Error("ClientCredentialsExist() = false for existing client")
	}
}

func TestListClients(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Empty list initially
	clients, err := ListClients()
	if err != nil {
		t.Fatalf("ListClients() error: %v", err)
	}
	if len(clients) != 0 {
		t.Errorf("ListClients() = %v, want empty", clients)
	}

	// Add some clients
	creds := &ClientCredentials{ClientID: "id", ClientSecret: "secret"}
	WriteClientCredentials("client1", creds)
	WriteClientCredentials("client2", creds)

	clients, err = ListClients()
	if err != nil {
		t.Fatalf("ListClients() error: %v", err)
	}
	if len(clients) != 2 {
		t.Errorf("ListClients() returned %d clients, want 2", len(clients))
	}
}

func TestWriteCredentialsValidation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Nil creds
	if err := WriteClientCredentials("test", nil); err == nil {
		t.Error("WriteClientCredentials(nil) should error")
	}

	// Empty client ID
	if err := WriteClientCredentials("test", &ClientCredentials{ClientSecret: "s"}); err == nil {
		t.Error("WriteClientCredentials with empty client_id should error")
	}

	// Empty client secret
	if err := WriteClientCredentials("test", &ClientCredentials{ClientID: "id"}); err == nil {
		t.Error("WriteClientCredentials with empty client_secret should error")
	}
}

func TestReadNonexistentClient(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	_, err := ReadClientCredentials("nonexistent")
	if err == nil {
		t.Error("ReadClientCredentials(nonexistent) should error")
	}
}

func TestDeleteClientCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := &ClientCredentials{ClientID: "id", ClientSecret: "secret"}
	WriteClientCredentials("todelete", creds)

	if !ClientCredentialsExist("todelete") {
		t.Fatal("client should exist before delete")
	}

	if err := DeleteClientCredentials("todelete"); err != nil {
		t.Fatalf("DeleteClientCredentials() error: %v", err)
	}

	if ClientCredentialsExist("todelete") {
		t.Error("client should not exist after delete")
	}

	// Delete nonexistent should error
	if err := DeleteClientCredentials("nonexistent"); err == nil {
		t.Error("DeleteClientCredentials(nonexistent) should error")
	}

	// Empty name should error
	if err := DeleteClientCredentials(""); err == nil {
		t.Error("DeleteClientCredentials('') should error")
	}
}
