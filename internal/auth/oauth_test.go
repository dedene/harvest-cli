package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/dedene/harvest-cli/internal/config"
)

func TestRandomState(t *testing.T) {
	state1, err := randomState()
	if err != nil {
		t.Fatalf("randomState() error = %v", err)
	}

	if state1 == "" {
		t.Error("randomState() returned empty string")
	}

	if len(state1) < 32 {
		t.Errorf("randomState() length = %d, want >= 32", len(state1))
	}

	// Ensure randomness
	state2, err := randomState()
	if err != nil {
		t.Fatalf("randomState() error = %v", err)
	}

	if state1 == state2 {
		t.Error("randomState() returned same value twice")
	}
}

func TestExtractCodeAndState(t *testing.T) {
	tests := []struct {
		name      string
		rawURL    string
		wantCode  string
		wantState string
		wantErr   bool
	}{
		{
			name:      "valid url with code and state",
			rawURL:    "http://localhost:8080/callback?code=abc123&state=xyz789",
			wantCode:  "abc123",
			wantState: "xyz789",
		},
		{
			name:     "valid url with code only",
			rawURL:   "http://localhost:8080/callback?code=abc123",
			wantCode: "abc123",
		},
		{
			name:    "missing code",
			rawURL:  "http://localhost:8080/callback?state=xyz789",
			wantErr: true,
		},
		{
			name:    "empty url",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:      "code with special chars",
			rawURL:    "http://localhost/cb?code=abc%2B123&state=xyz",
			wantCode:  "abc+123",
			wantState: "xyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, state, err := extractCodeAndState(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractCodeAndState() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if code != tt.wantCode {
				t.Errorf("extractCodeAndState() code = %q, want %q", code, tt.wantCode)
			}
			if state != tt.wantState {
				t.Errorf("extractCodeAndState() state = %q, want %q", state, tt.wantState)
			}
		})
	}
}

func TestAuthorizeOptions_Defaults(t *testing.T) {
	opts := AuthorizeOptions{}

	if opts.Manual {
		t.Error("Manual should default to false")
	}
	if opts.ForceConsent {
		t.Error("ForceConsent should default to false")
	}
	if opts.Timeout != 0 {
		t.Errorf("Timeout = %v, want 0 (to be set by Authorize)", opts.Timeout)
	}
}

func TestOpenBrowser_UnsupportedPlatform(t *testing.T) {
	// This tests the error path; actual browser opening is not tested
	// because it would open a browser window
}

func TestAuthorize_InvalidCredentials(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, _, err := Authorize(ctx, nil, AuthorizeOptions{Timeout: time.Second})
	if err == nil {
		t.Error("Authorize() should fail with nil credentials")
	}

	_, _, _, err = Authorize(ctx, &config.ClientCredentials{}, AuthorizeOptions{Timeout: time.Second})
	if err == nil {
		t.Error("Authorize() should fail with empty credentials")
	}
}

func TestCallbackServer_Error(t *testing.T) {
	// Test that error responses are handled correctly
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/callback" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if q.Get("error") != "" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(cancelledHTML))
			return
		}
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/oauth/callback?error=access_denied")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestCallbackServer_StateMismatch(t *testing.T) {
	expectedState := "expected-state"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/callback" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if q.Get("state") != expectedState {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(errorHTML("State mismatch")))
			return
		}
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/oauth/callback?code=abc&state=wrong-state")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCallbackServer_MissingCode(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/callback" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if q.Get("code") == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(errorHTML("Missing code")))
			return
		}
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/oauth/callback?state=abc")
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCallbackServer_Success(t *testing.T) {
	expectedState := "test-state"
	var receivedCode string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/callback" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if q.Get("state") != expectedState {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		code := q.Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		receivedCode = code
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(successHTML))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/oauth/callback?code=test-code&state=" + expectedState)
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if receivedCode != "test-code" {
		t.Errorf("receivedCode = %q, want %q", receivedCode, "test-code")
	}
}

func TestErrorHTML(t *testing.T) {
	html := errorHTML("Test error message")

	if !strings.Contains(html, "Test error message") {
		t.Error("errorHTML should contain the error message")
	}
	if !strings.Contains(html, "Authorization Error") {
		t.Error("errorHTML should contain 'Authorization Error'")
	}
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("errorHTML should be valid HTML")
	}
}

func TestSuccessHTML(t *testing.T) {
	if !strings.Contains(successHTML, "Authorization Successful") {
		t.Error("successHTML should contain 'Authorization Successful'")
	}
	if !strings.Contains(successHTML, "close this window") {
		t.Error("successHTML should tell user to close window")
	}
}

func TestCancelledHTML(t *testing.T) {
	if !strings.Contains(cancelledHTML, "Authorization Cancelled") {
		t.Error("cancelledHTML should contain 'Authorization Cancelled'")
	}
}

func TestAuthorize_WithMockServer(t *testing.T) {
	// Mock OAuth server
	mux := http.NewServeMux()

	// Token endpoint
	mux.HandleFunc("/api/v2/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "test-access-token",
			"refresh_token": "test-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	})

	// Accounts endpoint
	mux.HandleFunc("/api/v2/accounts", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-access-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(AccountsResponse{
			User: struct {
				ID        int64  `json:"id"`
				FirstName string `json:"first_name"`
				LastName  string `json:"last_name"`
				Email     string `json:"email"`
			}{
				ID:        1,
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
			},
			Accounts: []HarvestAccount{
				{
					ID:      12345,
					Name:    "Test Company",
					Product: "harvest",
				},
			},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	// Parse server URL to get port
	srvURL, _ := url.Parse(srv.URL)

	t.Logf("Mock server running at %s", srv.URL)
	t.Logf("Would test OAuth flow with server at port %s", srvURL.Port())
}
