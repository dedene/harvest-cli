// Package auth provides keyring-based credential storage and OAuth token management.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"golang.org/x/term"

	"github.com/dedene/harvest-cli/internal/config"
)

// Store defines the interface for token storage.
type Store interface {
	Keys() ([]string, error)
	SetToken(client, email string, accountID int64, tok Token) error
	GetToken(client, email string) (Token, error)
	DeleteToken(client, email string) error
	ListTokens() ([]Token, error)
}

// KeyringStore implements Store using the system keyring.
type KeyringStore struct {
	ring keyring.Keyring
}

// Token represents stored authentication information.
// RefreshToken is stored separately in the keyring and never serialized to JSON.
type Token struct {
	Client       string    `json:"client,omitempty"`
	Email        string    `json:"email"`
	AccountID    int64     `json:"account_id"`
	Scopes       []string  `json:"scopes,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	RefreshToken string    `json:"-"` // NEVER serialize to JSON/logs
}

const (
	keyringPasswordEnv = "HARVESTCLI_KEYRING_PASSWORD" //nolint:gosec // env var name
	keyringBackendEnv  = "HARVESTCLI_KEYRING_BACKEND"  //nolint:gosec // env var name
)

var (
	errMissingEmail        = errors.New("missing email")
	errMissingRefreshToken = errors.New("missing refresh token")
	errMissingAccountID    = errors.New("missing account ID")
	errNoTTY               = errors.New("no TTY available for keyring password prompt")
	errInvalidBackend      = errors.New("invalid keyring backend")
	errKeyringTimeout      = errors.New("keyring connection timed out")

	// Allow overriding for tests
	openKeyringFunc = openKeyring
	keyringOpenFunc = keyring.Open
)

const keyringOpenTimeout = 5 * time.Second

func openKeyring() (keyring.Keyring, error) {
	backend := normalizeBackend(os.Getenv(keyringBackendEnv))

	backends, err := allowedBackends(backend)
	if err != nil {
		return nil, err
	}

	dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")

	// Platform-specific backend defaults
	switch {
	case runtime.GOOS == "darwin" && (backend == "" || backend == "auto"):
		backends = []keyring.BackendType{keyring.KeychainBackend}
	case shouldForceFileBackend(runtime.GOOS, backend, dbusAddr):
		backends = []keyring.BackendType{keyring.FileBackend}
	}

	keyringDir := config.KeyringDir()
	if keyringDir == "" {
		return nil, errors.New("could not determine keyring directory")
	}

	cfg := keyring.Config{
		ServiceName:              config.AppName,
		KeychainTrustApplication: false,
		AllowedBackends:          backends,
		FileDir:                  keyringDir,
		FilePasswordFunc:         fileKeyringPasswordFunc(),
	}

	// Only create keyring dir when file backend is explicitly selected
	if usesFileBackend(backends) {
		if err := os.MkdirAll(keyringDir, 0700); err != nil {
			return nil, fmt.Errorf("ensure keyring dir: %w", err)
		}
	}

	if shouldUseTimeout(runtime.GOOS, backend, dbusAddr) {
		return openKeyringWithTimeout(cfg, keyringOpenTimeout)
	}

	ring, err := keyringOpenFunc(cfg)
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return ring, nil
}

func normalizeBackend(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func allowedBackends(backend string) ([]keyring.BackendType, error) {
	switch backend {
	case "", "auto":
		return nil, nil
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend}, nil
	case "file":
		return []keyring.BackendType{keyring.FileBackend}, nil
	case "secret-service":
		return []keyring.BackendType{keyring.SecretServiceBackend}, nil
	case "wincred":
		return []keyring.BackendType{keyring.WinCredBackend}, nil
	default:
		return nil, fmt.Errorf("%w: %q", errInvalidBackend, backend)
	}
}

func shouldForceFileBackend(goos, backend, dbusAddr string) bool {
	return goos == "linux" && (backend == "" || backend == "auto") && dbusAddr == ""
}

func shouldUseTimeout(goos, backend, dbusAddr string) bool {
	return goos == "linux" && (backend == "" || backend == "auto") && dbusAddr != ""
}

func usesFileBackend(backends []keyring.BackendType) bool {
	for _, b := range backends {
		if b == keyring.FileBackend {
			return true
		}
	}
	return false
}

func fileKeyringPasswordFunc() keyring.PromptFunc {
	password := os.Getenv(keyringPasswordEnv)
	if password != "" {
		return keyring.FixedStringPrompt(password)
	}

	if term.IsTerminal(int(os.Stdin.Fd())) {
		return keyring.TerminalPrompt
	}

	return func(_ string) (string, error) {
		return "", fmt.Errorf("%w; set %s", errNoTTY, keyringPasswordEnv)
	}
}

type keyringResult struct {
	ring keyring.Keyring
	err  error
}

func openKeyringWithTimeout(cfg keyring.Config, timeout time.Duration) (keyring.Keyring, error) {
	ch := make(chan keyringResult, 1)

	go func() {
		ring, err := keyringOpenFunc(cfg)
		ch <- keyringResult{ring, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("open keyring: %w", res.err)
		}
		return res.ring, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%w after %v; set %s=file and %s=<password>",
			errKeyringTimeout, timeout, keyringBackendEnv, keyringPasswordEnv)
	}
}

// OpenDefault opens the keyring store with auto-detected backend.
func OpenDefault() (Store, error) {
	ring, err := openKeyringFunc()
	if err != nil {
		return nil, err
	}

	return &KeyringStore{ring: ring}, nil
}

// OpenWithBackend opens the keyring store with a specific backend.
func OpenWithBackend(backend string) (Store, error) {
	orig := os.Getenv(keyringBackendEnv)
	os.Setenv(keyringBackendEnv, backend)
	defer os.Setenv(keyringBackendEnv, orig)

	return OpenDefault()
}

// storedToken is the JSON-serialized format for keyring storage.
// RefreshToken is included here because it goes into the keyring.
type storedToken struct {
	RefreshToken string    `json:"refresh_token"`
	AccountID    int64     `json:"account_id"`
	Scopes       []string  `json:"scopes,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
}

// Keys returns all keys in the keyring.
func (s *KeyringStore) Keys() ([]string, error) {
	keys, err := s.ring.Keys()
	if err != nil {
		return nil, fmt.Errorf("list keyring keys: %w", err)
	}

	return keys, nil
}

// SetToken stores a token in the keyring.
func (s *KeyringStore) SetToken(client, email string, accountID int64, tok Token) error {
	email = normalize(email)
	if email == "" {
		return errMissingEmail
	}

	if tok.RefreshToken == "" {
		return errMissingRefreshToken
	}

	if accountID == 0 {
		return errMissingAccountID
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(client)
	if err != nil {
		return fmt.Errorf("normalize client: %w", err)
	}

	if tok.CreatedAt.IsZero() {
		tok.CreatedAt = time.Now().UTC()
	}

	payload, err := json.Marshal(storedToken{
		RefreshToken: tok.RefreshToken,
		AccountID:    accountID,
		Scopes:       tok.Scopes,
		CreatedAt:    tok.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("encode token: %w", err)
	}

	if err := s.ring.Set(keyring.Item{
		Key:  tokenKey(normalizedClient, email),
		Data: payload,
	}); err != nil {
		return wrapKeychainError(fmt.Errorf("store token: %w", err))
	}

	return nil
}

// GetToken retrieves a token from the keyring.
func (s *KeyringStore) GetToken(client, email string) (Token, error) {
	email = normalize(email)
	if email == "" {
		return Token{}, errMissingEmail
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(client)
	if err != nil {
		return Token{}, fmt.Errorf("normalize client: %w", err)
	}

	item, err := s.ring.Get(tokenKey(normalizedClient, email))
	if err != nil {
		return Token{}, fmt.Errorf("read token: %w", err)
	}

	var st storedToken
	if err := json.Unmarshal(item.Data, &st); err != nil {
		return Token{}, fmt.Errorf("decode token: %w", err)
	}

	return Token{
		Client:       normalizedClient,
		Email:        email,
		AccountID:    st.AccountID,
		Scopes:       st.Scopes,
		CreatedAt:    st.CreatedAt,
		RefreshToken: st.RefreshToken,
	}, nil
}

// DeleteToken removes a token from the keyring.
func (s *KeyringStore) DeleteToken(client, email string) error {
	email = normalize(email)
	if email == "" {
		return errMissingEmail
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(client)
	if err != nil {
		return fmt.Errorf("normalize client: %w", err)
	}

	if err := s.ring.Remove(tokenKey(normalizedClient, email)); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
		return fmt.Errorf("delete token: %w", err)
	}

	return nil
}

// ListTokens returns all tokens in the keyring.
func (s *KeyringStore) ListTokens() ([]Token, error) {
	keys, err := s.Keys()
	if err != nil {
		return nil, fmt.Errorf("list tokens: %w", err)
	}

	out := make([]Token, 0)
	seen := make(map[string]struct{})

	for _, k := range keys {
		client, email, ok := ParseTokenKey(k)
		if !ok {
			continue
		}

		key := client + "\n" + email
		if _, exists := seen[key]; exists {
			continue
		}

		tok, err := s.GetToken(client, email)
		if err != nil {
			return nil, fmt.Errorf("read token for %s: %w", email, err)
		}

		seen[key] = struct{}{}
		out = append(out, tok)
	}

	return out, nil
}

// ParseTokenKey parses a keyring key into client and email.
func ParseTokenKey(k string) (client, email string, ok bool) {
	const prefix = "token:"
	if !strings.HasPrefix(k, prefix) {
		return "", "", false
	}

	rest := strings.TrimPrefix(k, prefix)
	if strings.TrimSpace(rest) == "" {
		return "", "", false
	}

	if !strings.Contains(rest, ":") {
		return config.DefaultClientName, rest, true
	}

	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func tokenKey(client, email string) string {
	return fmt.Sprintf("token:%s:%s", client, email)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func wrapKeychainError(err error) error {
	if err == nil {
		return nil
	}

	if IsKeychainLockedError(err.Error()) {
		return fmt.Errorf("%w\n\nYour macOS keychain is locked. Run:\n  security unlock-keychain ~/Library/Keychains/login.keychain-db", err)
	}

	return err
}

// IsKeychainLockedError returns true if the error indicates the keychain is locked.
func IsKeychainLockedError(msg string) bool {
	return strings.Contains(msg, "keychain is locked") ||
		strings.Contains(msg, "The user name or passphrase you entered is not correct")
}
