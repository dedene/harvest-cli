package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/dedene/harvest-cli/internal/config"
)

// AuthorizeOptions configures the OAuth authorization flow.
type AuthorizeOptions struct {
	Manual       bool          // Manual copy/paste flow instead of browser
	ForceConsent bool          // Force consent screen
	Timeout      time.Duration // Timeout for callback server
	Client       string        // OAuth client name
}

var (
	errAuthorization       = errors.New("authorization error")
	errMissingCode         = errors.New("missing code")
	errNoCodeInURL         = errors.New("no code found in URL")
	errNoRefreshToken      = errors.New("no refresh token received; try with --force-consent")
	errStateMismatch       = errors.New("state mismatch")
	errUnsupportedPlatform = errors.New("unsupported platform")

	// Allow overriding for tests
	openBrowserFn = openBrowser
	randomStateFn = randomState
)

// Authorize performs OAuth authorization and returns the email, account ID, and token.
func Authorize(ctx context.Context, creds *config.ClientCredentials, opts AuthorizeOptions) (email string, accountID int64, tok *oauth2.Token, err error) {
	if creds == nil {
		return "", 0, nil, errors.New("credentials cannot be nil")
	}
	if creds.ClientID == "" || creds.ClientSecret == "" {
		return "", 0, nil, errors.New("credentials missing client_id or client_secret")
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 2 * time.Minute
	}

	state, err := randomStateFn()
	if err != nil {
		return "", 0, nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	// Start callback server on fixed port
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:8484")
	if err != nil {
		return "", 0, nil, fmt.Errorf("listen for callback: %w", err)
	}
	defer func() { _ = ln.Close() }()

	redirectURI := "http://localhost:8484/oauth/callback"

	cfg := oauth2.Config{
		ClientID:     creds.ClientID,
		ClientSecret: creds.ClientSecret,
		Endpoint:     HarvestOAuthEndpoint,
		RedirectURL:  redirectURI,
	}

	authOpts := []oauth2.AuthCodeOption{oauth2.AccessTypeOffline}
	if opts.ForceConsent {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("prompt", "consent"))
	}

	if opts.Manual {
		// Close the listener since we're doing manual flow
		_ = ln.Close()
		return authorizeManual(ctx, cfg, state, authOpts)
	}

	return authorizeWithServer(ctx, cfg, state, authOpts, ln)
}

func authorizeManual(ctx context.Context, cfg oauth2.Config, state string, authOpts []oauth2.AuthCodeOption) (string, int64, *oauth2.Token, error) {
	authURL := cfg.AuthCodeURL(state, authOpts...)

	fmt.Fprintln(os.Stderr, "Visit this URL to authorize:")
	fmt.Fprintln(os.Stderr, authURL)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "After authorizing, you'll be redirected to a URL.")
	fmt.Fprintln(os.Stderr, "Copy the URL from your browser and paste it here.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Paste redirect URL: ")

	var line string
	if _, err := fmt.Scanln(&line); err != nil {
		return "", 0, nil, fmt.Errorf("read redirect url: %w", err)
	}

	line = strings.TrimSpace(line)

	code, gotState, err := extractCodeAndState(line)
	if err != nil {
		return "", 0, nil, err
	}

	if gotState != "" && gotState != state {
		return "", 0, nil, errStateMismatch
	}

	tok, err := cfg.Exchange(ctx, code)
	if err != nil {
		return "", 0, nil, fmt.Errorf("exchange code: %w", err)
	}

	if tok.RefreshToken == "" {
		return "", 0, nil, errNoRefreshToken
	}

	// Fetch user info and accounts
	resp, err := FetchAccounts(ctx, tok)
	if err != nil {
		return "", 0, nil, fmt.Errorf("fetch accounts: %w", err)
	}

	if len(resp.Accounts) == 0 {
		return "", 0, nil, errors.New("no Harvest accounts found")
	}

	// Select account if multiple
	accountID, err := SelectAccount(resp.Accounts)
	if err != nil {
		return "", 0, nil, fmt.Errorf("select account: %w", err)
	}

	return resp.User.Email, accountID, tok, nil
}

func authorizeWithServer(ctx context.Context, cfg oauth2.Config, state string, authOpts []oauth2.AuthCodeOption, ln net.Listener) (string, int64, *oauth2.Token, error) {
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          log.New(io.Discard, "", 0),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/callback" {
				http.NotFound(w, r)
				return
			}

			q := r.URL.Query()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")

			if q.Get("error") != "" {
				select {
				case errCh <- fmt.Errorf("%w: %s", errAuthorization, q.Get("error")):
				default:
				}

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(cancelledHTML))
				return
			}

			if q.Get("state") != state {
				select {
				case errCh <- errStateMismatch:
				default:
				}

				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(errorHTML("State mismatch - please try again.")))
				return
			}

			code := q.Get("code")
			if code == "" {
				select {
				case errCh <- errMissingCode:
				default:
				}

				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(errorHTML("Missing authorization code.")))
				return
			}

			select {
			case codeCh <- code:
			default:
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(successHTML))
		}),
	}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	authURL := cfg.AuthCodeURL(state, authOpts...)

	fmt.Fprintln(os.Stderr, "Opening browser for authorization...")
	fmt.Fprintln(os.Stderr, "If the browser doesn't open, visit:")
	fmt.Fprintln(os.Stderr, authURL)
	_ = openBrowserFn(authURL)

	select {
	case code := <-codeCh:
		fmt.Fprintln(os.Stderr, "Authorization received. Finishing...")

		tok, err := cfg.Exchange(ctx, code)
		if err != nil {
			_ = srv.Close()
			return "", 0, nil, fmt.Errorf("exchange code: %w", err)
		}

		if tok.RefreshToken == "" {
			_ = srv.Close()
			return "", 0, nil, errNoRefreshToken
		}

		// Fetch user info and accounts
		resp, err := FetchAccounts(ctx, tok)
		if err != nil {
			_ = srv.Close()
			return "", 0, nil, fmt.Errorf("fetch accounts: %w", err)
		}

		if len(resp.Accounts) == 0 {
			_ = srv.Close()
			return "", 0, nil, errors.New("no Harvest accounts found")
		}

		// Select account if multiple
		accountID, err := SelectAccount(resp.Accounts)
		if err != nil {
			_ = srv.Close()
			return "", 0, nil, fmt.Errorf("select account: %w", err)
		}

		shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)

		return resp.User.Email, accountID, tok, nil

	case err := <-errCh:
		_ = srv.Close()
		return "", 0, nil, err

	case <-ctx.Done():
		_ = srv.Close()
		return "", 0, nil, fmt.Errorf("authorization canceled: %w", ctx.Err())
	}
}

func randomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate state: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func extractCodeAndState(rawURL string) (code, state string, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("parse redirect url: %w", err)
	}

	code = parsed.Query().Get("code")
	if code == "" {
		return "", "", errNoCodeInURL
	}

	return code, parsed.Query().Get("state"), nil
}

func openBrowser(targetURL string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", targetURL) //nolint:gosec // fire-and-forget browser open
	case "linux":
		cmd = exec.Command("xdg-open", targetURL) //nolint:gosec // fire-and-forget browser open
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", targetURL) //nolint:gosec // fire-and-forget browser open
	default:
		return fmt.Errorf("%w: %s", errUnsupportedPlatform, runtime.GOOS)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start browser: %w", err)
	}

	return nil
}

const successHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Authorization Successful</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
           display: flex; justify-content: center; align-items: center;
           min-height: 100vh; margin: 0; background: #f5f5f5; }
    .container { text-align: center; padding: 40px; background: white;
                 border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
    h1 { color: #22c55e; margin-bottom: 16px; }
    p { color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>&#10004; Authorization Successful</h1>
    <p>You can close this window and return to the terminal.</p>
  </div>
</body>
</html>`

const cancelledHTML = `<!DOCTYPE html>
<html>
<head>
  <title>Authorization Cancelled</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
           display: flex; justify-content: center; align-items: center;
           min-height: 100vh; margin: 0; background: #f5f5f5; }
    .container { text-align: center; padding: 40px; background: white;
                 border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
    h1 { color: #f59e0b; margin-bottom: 16px; }
    p { color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Authorization Cancelled</h1>
    <p>You can close this window.</p>
  </div>
</body>
</html>`

func errorHTML(msg string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>Authorization Error</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
           display: flex; justify-content: center; align-items: center;
           min-height: 100vh; margin: 0; background: #f5f5f5; }
    .container { text-align: center; padding: 40px; background: white;
                 border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
    h1 { color: #ef4444; margin-bottom: 16px; }
    p { color: #666; }
  </style>
</head>
<body>
  <div class="container">
    <h1>Authorization Error</h1>
    <p>%s</p>
  </div>
</body>
</html>`, html.EscapeString(msg))
}
