package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/oauth2"

	"github.com/dedene/harvest-cli/internal/config"
)

// ErrNotAuthenticated indicates no valid token is available.
var ErrNotAuthenticated = errors.New("not authenticated")

// HarvestOAuthEndpoint is the Harvest OAuth2 endpoint.
var HarvestOAuthEndpoint = oauth2.Endpoint{
	AuthURL:  "https://id.getharvest.com/oauth2/authorize",
	TokenURL: "https://id.getharvest.com/api/v2/oauth2/token",
}

// TokenSource provides OAuth2 tokens with lazy refresh on 401.
// Access tokens are kept in memory only; refresh tokens are stored in keyring.
type TokenSource struct {
	mu           sync.Mutex
	store        Store
	client       string
	email        string
	oauth2Config *oauth2.Config

	accessToken  string
	accessExpiry time.Time
}

// NewTokenSource creates a new TokenSource for the given client and email.
func NewTokenSource(store Store, client, email string, cfg *oauth2.Config) *TokenSource {
	return &TokenSource{
		store:        store,
		client:       client,
		email:        email,
		oauth2Config: cfg,
	}
}

// Token returns a valid OAuth2 token, refreshing if necessary.
// Implements oauth2.TokenSource interface.
func (ts *TokenSource) Token() (*oauth2.Token, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	// Return cached access token if still valid (with 30s buffer)
	if ts.accessToken != "" && time.Now().Add(30*time.Second).Before(ts.accessExpiry) {
		return &oauth2.Token{
			AccessToken: ts.accessToken,
			Expiry:      ts.accessExpiry,
		}, nil
	}

	// Refresh the token
	if err := ts.refresh(); err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: ts.accessToken,
		Expiry:      ts.accessExpiry,
	}, nil
}

// Invalidate marks the current access token as invalid.
// Forces a refresh on the next Token() call.
// Call this on 401 responses.
func (ts *TokenSource) Invalidate() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.accessToken = ""
	ts.accessExpiry = time.Time{}
}

func (ts *TokenSource) refresh() error {
	// Get refresh token from keyring
	tok, err := ts.store.GetToken(ts.client, ts.email)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrNotAuthenticated, err)
	}

	if tok.RefreshToken == "" {
		return ErrNotAuthenticated
	}

	// Build OAuth config if not provided
	cfg := ts.oauth2Config
	if cfg == nil {
		creds, err := config.ReadClientCredentials(ts.client)
		if err != nil {
			return fmt.Errorf("read credentials: %w", err)
		}

		cfg = &oauth2.Config{
			ClientID:     creds.ClientID,
			ClientSecret: creds.ClientSecret,
			Endpoint:     HarvestOAuthEndpoint,
		}
	}

	// Use refresh token to get new access token
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	newTok, err := cfg.TokenSource(ctx, &oauth2.Token{
		RefreshToken: tok.RefreshToken,
	}).Token()
	if err != nil {
		return fmt.Errorf("refresh token: %w", err)
	}

	ts.accessToken = newTok.AccessToken
	ts.accessExpiry = newTok.Expiry

	// If we got a new refresh token, store it
	if newTok.RefreshToken != "" && newTok.RefreshToken != tok.RefreshToken {
		tok.RefreshToken = newTok.RefreshToken
		if storeErr := ts.store.SetToken(ts.client, ts.email, tok.AccountID, tok); storeErr != nil {
			// Log but don't fail - we still have a working access token
			fmt.Printf("Warning: failed to store new refresh token: %v\n", storeErr)
		}
	}

	return nil
}

// GetAuthenticatedEmail returns the email for any authenticated account,
// optionally filtered by client name.
func GetAuthenticatedEmail(client string) (string, error) {
	store, err := OpenDefault()
	if err != nil {
		return "", err
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return "", fmt.Errorf("list tokens: %w", err)
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(client)
	if err != nil {
		return "", fmt.Errorf("normalize client: %w", err)
	}

	for _, tok := range tokens {
		if tok.Client == normalizedClient {
			return tok.Email, nil
		}
	}

	return "", ErrNotAuthenticated
}
