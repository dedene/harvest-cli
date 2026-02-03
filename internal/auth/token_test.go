package auth

import (
	"errors"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// mockStore implements Store for testing.
type mockStore struct {
	tokens map[string]Token
	mu     sync.Mutex
	err    error
}

func newMockStore() *mockStore {
	return &mockStore{
		tokens: make(map[string]Token),
	}
}

func (m *mockStore) Keys() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	keys := make([]string, 0, len(m.tokens))
	for k := range m.tokens {
		keys = append(keys, k)
	}
	return keys, nil
}

func (m *mockStore) SetToken(client, email string, accountID int64, tok Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	tok.Client = client
	tok.Email = email
	tok.AccountID = accountID
	key := client + ":" + email
	m.tokens[key] = tok
	return nil
}

func (m *mockStore) GetToken(client, email string) (Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return Token{}, m.err
	}
	key := client + ":" + email
	tok, ok := m.tokens[key]
	if !ok {
		return Token{}, errors.New("token not found")
	}
	return tok, nil
}

func (m *mockStore) DeleteToken(client, email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	key := client + ":" + email
	delete(m.tokens, key)
	return nil
}

func (m *mockStore) ListTokens() ([]Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	tokens := make([]Token, 0, len(m.tokens))
	for _, tok := range m.tokens {
		tokens = append(tokens, tok)
	}
	return tokens, nil
}

func TestNewTokenSource(t *testing.T) {
	store := newMockStore()
	cfg := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint:     HarvestOAuthEndpoint,
	}

	ts := NewTokenSource(store, "default", "test@example.com", cfg)

	if ts == nil {
		t.Fatal("NewTokenSource() returned nil")
	}
	if ts.store != store {
		t.Error("store not set correctly")
	}
	if ts.client != "default" {
		t.Errorf("client = %q, want %q", ts.client, "default")
	}
	if ts.email != "test@example.com" {
		t.Errorf("email = %q, want %q", ts.email, "test@example.com")
	}
	if ts.oauth2Config != cfg {
		t.Error("oauth2Config not set correctly")
	}
}

func TestTokenSource_Token_NoStoredToken(t *testing.T) {
	store := newMockStore()
	ts := NewTokenSource(store, "default", "test@example.com", nil)

	_, err := ts.Token()
	if err == nil {
		t.Error("Token() should fail with no stored token")
	}
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("error = %v, want ErrNotAuthenticated", err)
	}
}

func TestTokenSource_Token_EmptyRefreshToken(t *testing.T) {
	store := newMockStore()
	_ = store.SetToken("default", "test@example.com", 123, Token{
		RefreshToken: "", // empty
	})

	ts := NewTokenSource(store, "default", "test@example.com", nil)

	_, err := ts.Token()
	if err == nil {
		t.Error("Token() should fail with empty refresh token")
	}
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Errorf("error = %v, want ErrNotAuthenticated", err)
	}
}

func TestTokenSource_Invalidate(t *testing.T) {
	store := newMockStore()
	ts := NewTokenSource(store, "default", "test@example.com", nil)

	// Set cached token
	ts.mu.Lock()
	ts.accessToken = "cached-token"
	ts.accessExpiry = time.Now().Add(time.Hour)
	ts.mu.Unlock()

	// Invalidate
	ts.Invalidate()

	// Verify cleared
	ts.mu.Lock()
	if ts.accessToken != "" {
		t.Error("accessToken should be empty after Invalidate()")
	}
	if !ts.accessExpiry.IsZero() {
		t.Error("accessExpiry should be zero after Invalidate()")
	}
	ts.mu.Unlock()
}

func TestTokenSource_CachedToken(t *testing.T) {
	store := newMockStore()
	ts := NewTokenSource(store, "default", "test@example.com", nil)

	// Pre-set a cached token
	futureExpiry := time.Now().Add(time.Hour)
	ts.mu.Lock()
	ts.accessToken = "cached-access-token"
	ts.accessExpiry = futureExpiry
	ts.mu.Unlock()

	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if tok.AccessToken != "cached-access-token" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "cached-access-token")
	}
}

func TestTokenSource_ExpiredCachedToken(t *testing.T) {
	store := newMockStore()
	// Need a refresh token but no credentials for the refresh
	_ = store.SetToken("default", "test@example.com", 123, Token{
		RefreshToken: "refresh-token",
	})

	ts := NewTokenSource(store, "default", "test@example.com", nil)

	// Pre-set an expired token
	ts.mu.Lock()
	ts.accessToken = "expired-token"
	ts.accessExpiry = time.Now().Add(-time.Hour) // expired
	ts.mu.Unlock()

	// Should attempt refresh, but will fail because no credentials
	_, err := ts.Token()
	if err == nil {
		t.Error("Token() should fail without credentials for refresh")
	}
}

func TestTokenSource_ThreadSafe(t *testing.T) {
	store := newMockStore()
	ts := NewTokenSource(store, "default", "test@example.com", nil)

	// Set cached token
	ts.mu.Lock()
	ts.accessToken = "token"
	ts.accessExpiry = time.Now().Add(time.Hour)
	ts.mu.Unlock()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = ts.Token()
		}()
	}

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ts.Invalidate()
		}()
	}

	wg.Wait()
}

func TestHarvestOAuthEndpoint(t *testing.T) {
	if HarvestOAuthEndpoint.AuthURL == "" {
		t.Error("AuthURL is empty")
	}
	if HarvestOAuthEndpoint.TokenURL == "" {
		t.Error("TokenURL is empty")
	}

	// Verify correct Harvest URLs
	expectedAuth := "https://id.getharvest.com/oauth2/authorize"
	expectedToken := "https://id.getharvest.com/api/v2/oauth2/token"

	if HarvestOAuthEndpoint.AuthURL != expectedAuth {
		t.Errorf("AuthURL = %q, want %q", HarvestOAuthEndpoint.AuthURL, expectedAuth)
	}
	if HarvestOAuthEndpoint.TokenURL != expectedToken {
		t.Errorf("TokenURL = %q, want %q", HarvestOAuthEndpoint.TokenURL, expectedToken)
	}
}
