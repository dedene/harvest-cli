package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/99designs/keyring"
)

// mockKeyring implements keyring.Keyring for testing.
type mockKeyring struct {
	items map[string]keyring.Item
	err   error // error to return on operations
}

func newMockKeyring() *mockKeyring {
	return &mockKeyring{
		items: make(map[string]keyring.Item),
	}
}

func (m *mockKeyring) Get(key string) (keyring.Item, error) {
	if m.err != nil {
		return keyring.Item{}, m.err
	}
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *mockKeyring) GetMetadata(key string) (keyring.Metadata, error) {
	return keyring.Metadata{}, nil
}

func (m *mockKeyring) Set(item keyring.Item) error {
	if m.err != nil {
		return m.err
	}
	m.items[item.Key] = item
	return nil
}

func (m *mockKeyring) Remove(key string) error {
	if m.err != nil {
		return m.err
	}
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *mockKeyring) Keys() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestKeyringStore_SetGetToken(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	tok := Token{
		RefreshToken: "refresh-123",
		Scopes:       []string{"harvest:read", "harvest:write"},
		CreatedAt:    time.Now().UTC(),
	}

	// Set token
	err := store.SetToken("default", "test@example.com", 12345, tok)
	if err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Get token back
	got, err := store.GetToken("default", "test@example.com")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}
	if got.Client != "default" {
		t.Errorf("Client = %q, want %q", got.Client, "default")
	}
	if got.AccountID != 12345 {
		t.Errorf("AccountID = %d, want %d", got.AccountID, 12345)
	}
	if got.RefreshToken != "refresh-123" {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, "refresh-123")
	}
	if len(got.Scopes) != 2 {
		t.Errorf("Scopes len = %d, want 2", len(got.Scopes))
	}
}

func TestKeyringStore_SetToken_Validation(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	tests := []struct {
		name      string
		client    string
		email     string
		accountID int64
		tok       Token
		wantErr   error
	}{
		{
			name:      "missing email",
			client:    "default",
			email:     "",
			accountID: 123,
			tok:       Token{RefreshToken: "refresh"},
			wantErr:   errMissingEmail,
		},
		{
			name:      "missing refresh token",
			client:    "default",
			email:     "test@example.com",
			accountID: 123,
			tok:       Token{},
			wantErr:   errMissingRefreshToken,
		},
		{
			name:      "missing account ID",
			client:    "default",
			email:     "test@example.com",
			accountID: 0,
			tok:       Token{RefreshToken: "refresh"},
			wantErr:   errMissingAccountID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := store.SetToken(tt.client, tt.email, tt.accountID, tt.tok)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("SetToken() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestKeyringStore_DeleteToken(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	tok := Token{RefreshToken: "refresh-123"}
	_ = store.SetToken("default", "test@example.com", 123, tok)

	// Delete
	err := store.DeleteToken("default", "test@example.com")
	if err != nil {
		t.Fatalf("DeleteToken() error = %v", err)
	}

	// Verify gone
	_, err = store.GetToken("default", "test@example.com")
	if err == nil {
		t.Error("GetToken() after delete should fail")
	}
}

func TestKeyringStore_DeleteToken_NotFound(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	// Deleting non-existent token should not error
	err := store.DeleteToken("default", "nonexistent@example.com")
	if err != nil {
		t.Errorf("DeleteToken() error = %v, want nil", err)
	}
}

func TestKeyringStore_ListTokens(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	// Add multiple tokens
	_ = store.SetToken("default", "user1@example.com", 111, Token{RefreshToken: "r1"})
	_ = store.SetToken("default", "user2@example.com", 222, Token{RefreshToken: "r2"})
	_ = store.SetToken("custom", "user3@example.com", 333, Token{RefreshToken: "r3"})

	tokens, err := store.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens() error = %v", err)
	}

	if len(tokens) != 3 {
		t.Errorf("ListTokens() returned %d tokens, want 3", len(tokens))
	}

	// Verify each token's data
	found := make(map[string]bool)
	for _, tok := range tokens {
		key := tok.Client + ":" + tok.Email
		found[key] = true
	}

	expected := []string{"default:user1@example.com", "default:user2@example.com", "custom:user3@example.com"}
	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("ListTokens() missing %s", exp)
		}
	}
}

func TestKeyringStore_EmailNormalization(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	tok := Token{RefreshToken: "refresh-123"}

	// Set with mixed case
	err := store.SetToken("default", "Test@Example.COM", 123, tok)
	if err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Get with different casing
	got, err := store.GetToken("default", "test@example.com")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want lowercase", got.Email)
	}
}

func TestKeyringStore_ClientNormalization(t *testing.T) {
	ring := newMockKeyring()
	store := &KeyringStore{ring: ring}

	tok := Token{RefreshToken: "refresh-123"}

	// Empty client should default to "default"
	err := store.SetToken("", "test@example.com", 123, tok)
	if err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Should be retrievable with explicit "default"
	got, err := store.GetToken("default", "test@example.com")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if got.Client != "default" {
		t.Errorf("Client = %q, want %q", got.Client, "default")
	}
}

func TestParseTokenKey(t *testing.T) {
	tests := []struct {
		key        string
		wantClient string
		wantEmail  string
		wantOK     bool
	}{
		{"token:default:user@example.com", "default", "user@example.com", true},
		{"token:custom:user@example.com", "custom", "user@example.com", true},
		{"token:user@example.com", "default", "user@example.com", true}, // legacy format
		{"nottoken:foo:bar", "", "", false},
		{"token:", "", "", false},
		{"token::", "", "", false},
		{"token:client:", "", "", false},
		{"other-key", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			client, email, ok := ParseTokenKey(tt.key)
			if ok != tt.wantOK {
				t.Errorf("ParseTokenKey(%q) ok = %v, want %v", tt.key, ok, tt.wantOK)
			}
			if client != tt.wantClient {
				t.Errorf("ParseTokenKey(%q) client = %q, want %q", tt.key, client, tt.wantClient)
			}
			if email != tt.wantEmail {
				t.Errorf("ParseTokenKey(%q) email = %q, want %q", tt.key, email, tt.wantEmail)
			}
		})
	}
}

func TestTokenKey(t *testing.T) {
	got := tokenKey("custom", "user@example.com")
	want := "token:custom:user@example.com"
	if got != want {
		t.Errorf("tokenKey() = %q, want %q", got, want)
	}
}

func TestAllowedBackends(t *testing.T) {
	tests := []struct {
		backend string
		wantLen int
		wantErr bool
	}{
		{"", 0, false},
		{"auto", 0, false},
		{"keychain", 1, false},
		{"file", 1, false},
		{"secret-service", 1, false},
		{"wincred", 1, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.backend, func(t *testing.T) {
			backends, err := allowedBackends(tt.backend)
			if (err != nil) != tt.wantErr {
				t.Errorf("allowedBackends(%q) error = %v, wantErr %v", tt.backend, err, tt.wantErr)
			}
			if len(backends) != tt.wantLen {
				t.Errorf("allowedBackends(%q) len = %d, want %d", tt.backend, len(backends), tt.wantLen)
			}
		})
	}
}

func TestShouldForceFileBackend(t *testing.T) {
	tests := []struct {
		goos     string
		backend  string
		dbusAddr string
		want     bool
	}{
		{"linux", "", "", true},
		{"linux", "auto", "", true},
		{"linux", "", "/run/user/1000/bus", false},
		{"linux", "file", "", false},
		{"darwin", "", "", false},
		{"windows", "", "", false},
	}

	for _, tt := range tests {
		got := shouldForceFileBackend(tt.goos, tt.backend, tt.dbusAddr)
		if got != tt.want {
			t.Errorf("shouldForceFileBackend(%q, %q, %q) = %v, want %v",
				tt.goos, tt.backend, tt.dbusAddr, got, tt.want)
		}
	}
}

func TestShouldUseTimeout(t *testing.T) {
	tests := []struct {
		goos     string
		backend  string
		dbusAddr string
		want     bool
	}{
		{"linux", "", "/run/user/1000/bus", true},
		{"linux", "auto", "/run/user/1000/bus", true},
		{"linux", "", "", false},
		{"linux", "file", "/run/user/1000/bus", false},
		{"darwin", "", "/run/user/1000/bus", false},
	}

	for _, tt := range tests {
		got := shouldUseTimeout(tt.goos, tt.backend, tt.dbusAddr)
		if got != tt.want {
			t.Errorf("shouldUseTimeout(%q, %q, %q) = %v, want %v",
				tt.goos, tt.backend, tt.dbusAddr, got, tt.want)
		}
	}
}

func TestIsKeychainLockedError(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"keychain is locked", true},
		{"The user name or passphrase you entered is not correct", true},
		{"connection refused", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsKeychainLockedError(tt.msg)
		if got != tt.want {
			t.Errorf("IsKeychainLockedError(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

func TestUsesFileBackend(t *testing.T) {
	tests := []struct {
		name     string
		backends []keyring.BackendType
		want     bool
	}{
		{"nil backends", nil, false},
		{"empty backends", []keyring.BackendType{}, false},
		{"file only", []keyring.BackendType{keyring.FileBackend}, true},
		{"keychain only", []keyring.BackendType{keyring.KeychainBackend}, false},
		{"mixed with file", []keyring.BackendType{keyring.KeychainBackend, keyring.FileBackend}, true},
		{"mixed without file", []keyring.BackendType{keyring.KeychainBackend, keyring.SecretServiceBackend}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := usesFileBackend(tt.backends)
			if got != tt.want {
				t.Errorf("usesFileBackend(%v) = %v, want %v", tt.backends, got, tt.want)
			}
		})
	}
}
