package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewPATTokenSource(t *testing.T) {
	ts := NewPATTokenSource("test-token")

	if ts == nil {
		t.Fatal("NewPATTokenSource() returned nil")
	}
	if ts.token != "test-token" {
		t.Errorf("token = %q, want %q", ts.token, "test-token")
	}
}

func TestPATTokenSource_Token(t *testing.T) {
	ts := NewPATTokenSource("my-pat-token")

	tok, err := ts.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if tok.AccessToken != "my-pat-token" {
		t.Errorf("AccessToken = %q, want %q", tok.AccessToken, "my-pat-token")
	}
	if tok.TokenType != "Bearer" {
		t.Errorf("TokenType = %q, want %q", tok.TokenType, "Bearer")
	}
	// PAT should not expire (far-future expiry)
	if tok.Expiry.Before(time.Now().Add(365 * 24 * time.Hour)) {
		t.Error("Expiry should be far in the future for PAT")
	}
}

func TestPATTokenSource_Token_Empty(t *testing.T) {
	ts := NewPATTokenSource("")

	_, err := ts.Token()
	if err == nil {
		t.Error("Token() should fail with empty token")
	}
	if err != ErrNotAuthenticated {
		t.Errorf("error = %v, want ErrNotAuthenticated", err)
	}
}

func TestStorePAT(t *testing.T) {
	store := newMockStore()

	err := StorePAT(store, "user@example.com", 12345, "pat-token-123")
	if err != nil {
		t.Fatalf("StorePAT() error = %v", err)
	}

	// Verify stored correctly
	tok, err := store.GetToken(PATClient, "user@example.com")
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}

	if tok.RefreshToken != "pat-token-123" {
		t.Errorf("token = %q, want %q", tok.RefreshToken, "pat-token-123")
	}
	if tok.AccountID != 12345 {
		t.Errorf("accountID = %d, want %d", tok.AccountID, 12345)
	}
	if tok.Client != PATClient {
		t.Errorf("client = %q, want %q", tok.Client, PATClient)
	}
}

func TestStorePAT_Validation(t *testing.T) {
	store := newMockStore()

	tests := []struct {
		name      string
		email     string
		accountID int64
		token     string
		wantErr   bool
	}{
		{
			name:      "missing email",
			email:     "",
			accountID: 123,
			token:     "token",
			wantErr:   true,
		},
		{
			name:      "missing account ID",
			email:     "user@example.com",
			accountID: 0,
			token:     "token",
			wantErr:   true,
		},
		{
			name:      "missing token",
			email:     "user@example.com",
			accountID: 123,
			token:     "",
			wantErr:   true,
		},
		{
			name:      "valid",
			email:     "user@example.com",
			accountID: 123,
			token:     "valid-token",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StorePAT(store, tt.email, tt.accountID, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("StorePAT() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPAT(t *testing.T) {
	store := newMockStore()

	// Store a PAT
	_ = StorePAT(store, "user@example.com", 12345, "pat-token-123")

	// Retrieve it
	token, accountID, err := GetPAT(store, "user@example.com")
	if err != nil {
		t.Fatalf("GetPAT() error = %v", err)
	}

	if token != "pat-token-123" {
		t.Errorf("token = %q, want %q", token, "pat-token-123")
	}
	if accountID != 12345 {
		t.Errorf("accountID = %d, want %d", accountID, 12345)
	}
}

func TestGetPAT_NotFound(t *testing.T) {
	store := newMockStore()

	_, _, err := GetPAT(store, "nonexistent@example.com")
	if err == nil {
		t.Error("GetPAT() should fail for nonexistent user")
	}
}

func TestDeletePAT(t *testing.T) {
	store := newMockStore()

	// Store and then delete
	_ = StorePAT(store, "user@example.com", 12345, "pat-token-123")

	err := DeletePAT(store, "user@example.com")
	if err != nil {
		t.Fatalf("DeletePAT() error = %v", err)
	}

	// Verify deleted
	_, _, err = GetPAT(store, "user@example.com")
	if err == nil {
		t.Error("GetPAT() should fail after delete")
	}
}

func TestValidatePAT_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Harvest-Account-Id") != "12345" {
			t.Errorf("Harvest-Account-Id = %q", r.Header.Get("Harvest-Account-Id"))
		}

		// Return success
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         123,
			"first_name": "Test",
			"last_name":  "User",
			"email":      "test@example.com",
		})
	}))
	defer server.Close()

	// Override endpoint for test
	origEndpoint := usersEndpoint
	usersEndpoint = server.URL
	defer func() {
		usersEndpoint = origEndpoint
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email, err := ValidatePAT(ctx, "test-token", 12345)
	if err != nil {
		t.Fatalf("ValidatePAT error: %v", err)
	}
	if email != "test@example.com" {
		t.Errorf("email = %q, want %q", email, "test@example.com")
	}
}

func TestValidatePAT_InvalidToken(t *testing.T) {
	// Create test server that returns 401
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	// Override endpoint for test
	origEndpoint := usersEndpoint
	usersEndpoint = server.URL
	defer func() {
		usersEndpoint = origEndpoint
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := ValidatePAT(ctx, "invalid-token", 12345)
	if err == nil {
		t.Error("ValidatePAT() should fail with invalid token")
	}
}

func TestGetPATFromEnv(t *testing.T) {
	// Save original env
	origToken := os.Getenv(PATEnvToken)
	origAccountID := os.Getenv(PATEnvAccountID)
	defer func() {
		os.Setenv(PATEnvToken, origToken)
		os.Setenv(PATEnvAccountID, origAccountID)
	}()

	tests := []struct {
		name          string
		token         string
		accountID     string
		wantToken     string
		wantAccountID int64
		wantOK        bool
	}{
		{
			name:          "both set",
			token:         "my-token",
			accountID:     "12345",
			wantToken:     "my-token",
			wantAccountID: 12345,
			wantOK:        true,
		},
		{
			name:      "missing token",
			token:     "",
			accountID: "12345",
			wantOK:    false,
		},
		{
			name:      "missing account ID",
			token:     "my-token",
			accountID: "",
			wantOK:    false,
		},
		{
			name:      "invalid account ID",
			token:     "my-token",
			accountID: "not-a-number",
			wantOK:    false,
		},
		{
			name:      "zero account ID",
			token:     "my-token",
			accountID: "0",
			wantOK:    false,
		},
		{
			name:      "negative account ID",
			token:     "my-token",
			accountID: "-1",
			wantOK:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(PATEnvToken, tt.token)
			os.Setenv(PATEnvAccountID, tt.accountID)

			token, accountID, ok := GetPATFromEnv()
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok {
				if token != tt.wantToken {
					t.Errorf("token = %q, want %q", token, tt.wantToken)
				}
				if accountID != tt.wantAccountID {
					t.Errorf("accountID = %d, want %d", accountID, tt.wantAccountID)
				}
			}
		})
	}
}

func TestPATClient_Constant(t *testing.T) {
	if PATClient != "pat" {
		t.Errorf("PATClient = %q, want %q", PATClient, "pat")
	}
}

func TestPATEnvVars_Constants(t *testing.T) {
	if PATEnvToken != "HARVESTCLI_TOKEN" {
		t.Errorf("PATEnvToken = %q, want %q", PATEnvToken, "HARVESTCLI_TOKEN")
	}
	if PATEnvAccountID != "HARVESTCLI_ACCOUNT_ID" {
		t.Errorf("PATEnvAccountID = %q, want %q", PATEnvAccountID, "HARVESTCLI_ACCOUNT_ID")
	}
}
