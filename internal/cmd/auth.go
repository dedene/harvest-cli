package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/dedene/harvest-cli/internal/auth"
	"github.com/dedene/harvest-cli/internal/config"
)

// AuthCmd groups authentication subcommands.
type AuthCmd struct {
	Setup  AuthSetupCmd  `cmd:"" help:"Store OAuth client credentials"`
	Login  AuthLoginCmd  `cmd:"" help:"Authenticate with Harvest"`
	Logout AuthLogoutCmd `cmd:"" help:"Remove stored authentication"`
	Status AuthStatusCmd `cmd:"" help:"Show authentication status"`
	List   AuthListCmd   `cmd:"" help:"List authenticated accounts"`
	Switch AuthSwitchCmd `cmd:"" help:"Switch default account"`
}

// AuthSetupCmd stores OAuth credentials.
type AuthSetupCmd struct {
	ClientID     string `arg:"" optional:"" help:"OAuth client ID"`
	ClientSecret string `help:"OAuth client secret (for non-interactive use)" name:"client-secret"`
	ClientName   string `help:"Client name (default: default)" default:"default" name:"client-name"`
}

func (c *AuthSetupCmd) Run() error {
	if c.ClientID == "" {
		fmt.Fprintln(os.Stderr, `Usage: harvest auth setup <client_id> [--client-secret <secret>]

Create a developer app at https://id.getharvest.com/developers
Use http://localhost:8484/oauth/callback as the Redirect URI

Example:
  harvest auth setup abc123
  # You will be prompted for the client secret

For non-interactive use (scripts/agents):
  harvest auth setup abc123 --client-secret secret456`)
		return fmt.Errorf("missing client ID")
	}

	clientSecret := c.ClientSecret
	if clientSecret == "" {
		// Prompt for client secret securely (no echo)
		fmt.Fprint(os.Stderr, "Client Secret: ")
		secretBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return fmt.Errorf("read secret: %w", err)
		}
		clientSecret = strings.TrimSpace(string(secretBytes))
	}

	if clientSecret == "" {
		return fmt.Errorf("client secret cannot be empty")
	}

	creds := &config.ClientCredentials{
		ClientID:     c.ClientID,
		ClientSecret: clientSecret,
	}

	if err := config.WriteClientCredentials(c.ClientName, creds); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	path := config.ClientCredentialsPath(c.ClientName)
	fmt.Fprintf(os.Stdout, "Credentials saved to %s\n", path)
	fmt.Fprintln(os.Stdout, "Run 'harvest auth login' to authenticate.")

	return nil
}

// AuthLoginCmd authenticates with Harvest.
type AuthLoginCmd struct {
	ClientName   string `help:"OAuth client name" default:"default" name:"client-name"`
	ForceConsent bool   `help:"Force consent prompt even if already authorized" name:"force-consent"`
	Manual       bool   `help:"Manual authorization (paste URL instead of callback server)"`
	PAT          bool   `help:"Use Personal Access Token instead of OAuth" name:"pat"`
}

func (c *AuthLoginCmd) Run(cli *CLI) error {
	ctx := context.Background()

	if c.PAT {
		return c.loginWithPAT(ctx)
	}

	return c.loginWithOAuth(ctx)
}

func (c *AuthLoginCmd) loginWithPAT(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprint(os.Stderr, "Personal Access Token: ")
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read token: %w", err)
	}
	token = strings.TrimSpace(token)

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	fmt.Fprint(os.Stderr, "Account ID: ")
	accountIDStr, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read account ID: %w", err)
	}
	accountIDStr = strings.TrimSpace(accountIDStr)

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil || accountID <= 0 {
		return fmt.Errorf("invalid account ID: %q", accountIDStr)
	}

	// Validate PAT by calling /users/me
	fmt.Fprintln(os.Stderr, "Validating token...")
	email, err := auth.ValidatePAT(ctx, token, accountID)
	if err != nil {
		return fmt.Errorf("validate token: %w", err)
	}

	// Store PAT
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	if err := auth.StorePAT(store, email, accountID, token); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Successfully authenticated as %s (account %d)\n", email, accountID)

	// Set as default if no default exists
	cfg, _ := config.ReadConfig()
	if cfg != nil && cfg.DefaultAccount == "" {
		_ = config.SetDefaultAccount(email)
		fmt.Fprintf(os.Stdout, "Set %s as default account\n", email)
	}

	return nil
}

func (c *AuthLoginCmd) loginWithOAuth(ctx context.Context) error {
	// Read client credentials
	creds, err := config.ReadClientCredentials(c.ClientName)
	if err != nil {
		return fmt.Errorf("read credentials: %w\n\nRun 'harvest auth setup <client_id> <client_secret>' first", err)
	}

	opts := auth.AuthorizeOptions{
		Client:       c.ClientName,
		ForceConsent: c.ForceConsent,
		Manual:       c.Manual,
		Timeout:      3 * time.Minute,
	}

	email, accountID, tok, err := auth.Authorize(ctx, creds, opts)
	if err != nil {
		return fmt.Errorf("authorization failed: %w", err)
	}

	// Store token in keyring
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	authTok := auth.Token{
		Email:        email,
		AccountID:    accountID,
		RefreshToken: tok.RefreshToken,
		CreatedAt:    time.Now().UTC(),
	}

	if err := store.SetToken(c.ClientName, email, accountID, authTok); err != nil {
		return fmt.Errorf("store token: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Successfully authenticated as %s (account %d)\n", email, accountID)

	// Set as default if no default exists
	cfg, _ := config.ReadConfig()
	if cfg != nil && cfg.DefaultAccount == "" {
		_ = config.SetDefaultAccount(email)
		fmt.Fprintf(os.Stdout, "Set %s as default account\n", email)
	}

	return nil
}

// AuthLogoutCmd removes stored tokens.
type AuthLogoutCmd struct {
	Email      string `help:"Email/account to log out" name:"email"`
	ClientName string `help:"OAuth client name" default:"default" name:"client-name"`
	All        bool   `help:"Log out all accounts"`
}

func (c *AuthLogoutCmd) Run() error {
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	if c.All {
		return c.logoutAll(store)
	}

	return c.logoutOne(store)
}

func (c *AuthLogoutCmd) logoutAll(store auth.Store) error {
	tokens, err := store.ListTokens()
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	count := 0
	for _, tok := range tokens {
		if err := store.DeleteToken(tok.Client, tok.Email); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove token for %s: %v\n", tok.Email, err)
		} else {
			count++
		}
	}

	fmt.Fprintf(os.Stdout, "Logged out %d account(s)\n", count)

	// Clear default account
	cfg, _ := config.ReadConfig()
	if cfg != nil && cfg.DefaultAccount != "" {
		cfg.DefaultAccount = ""
		_ = config.WriteConfig(cfg)
	}

	return nil
}

func (c *AuthLogoutCmd) logoutOne(store auth.Store) error {
	email := c.Email
	clientName := c.ClientName

	if email == "" {
		// Find the account to logout
		tokens, err := store.ListTokens()
		if err != nil {
			return fmt.Errorf("list tokens: %w", err)
		}

		normalizedClient, err := config.NormalizeClientNameOrDefault(clientName)
		if err != nil {
			return err
		}

		var match auth.Token
		count := 0

		for _, tok := range tokens {
			if tok.Client == normalizedClient {
				match = tok
				count++
			}
		}

		if count == 0 {
			return fmt.Errorf("no authenticated accounts found")
		}

		if count > 1 {
			return fmt.Errorf("multiple accounts found; specify --email or use --all")
		}

		email = match.Email
	}

	// Try both OAuth and PAT deletion
	oauthErr := store.DeleteToken(clientName, email)
	patErr := auth.DeletePAT(store, email)

	// If both failed, something is wrong
	if oauthErr != nil && patErr != nil {
		return fmt.Errorf("no tokens found for %s", email)
	}

	fmt.Fprintf(os.Stdout, "Logged out %s\n", email)

	// Update default if needed
	cfg, _ := config.ReadConfig()
	if cfg != nil && cfg.DefaultAccount == email {
		cfg.DefaultAccount = ""
		_ = config.WriteConfig(cfg)
		fmt.Fprintln(os.Stdout, "Cleared default account")
	}

	return nil
}

// AuthStatusCmd shows authentication status.
type AuthStatusCmd struct {
	ClientName string `help:"OAuth client name" default:"default" name:"client-name"`
}

func (c *AuthStatusCmd) Run() error {
	// Check if credentials exist
	exists := config.ClientCredentialsExist(c.ClientName)

	if !exists {
		fmt.Fprintln(os.Stdout, "Not configured")
		fmt.Fprintln(os.Stdout, "Run 'harvest auth setup <client_id> <client_secret>' to configure OAuth.")
		fmt.Fprintln(os.Stdout, "Or run 'harvest auth login --pat' to use a Personal Access Token.")
		return nil
	}

	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	normalizedClient, err := config.NormalizeClientNameOrDefault(c.ClientName)
	if err != nil {
		return err
	}

	// Filter to matching client
	var matching []auth.Token
	for _, tok := range tokens {
		if tok.Client == normalizedClient || tok.Client == auth.PATClient {
			matching = append(matching, tok)
		}
	}

	if len(matching) == 0 {
		fmt.Fprintln(os.Stdout, "OAuth credentials configured but not authenticated.")
		fmt.Fprintln(os.Stdout, "Run 'harvest auth login' to authenticate.")
		return nil
	}

	// Show default account
	cfg, _ := config.ReadConfig()
	defaultAccount := ""
	if cfg != nil {
		defaultAccount = cfg.DefaultAccount
	}

	fmt.Fprintf(os.Stdout, "Authenticated: %d account(s)\n", len(matching))
	for _, tok := range matching {
		marker := ""
		if tok.Email == defaultAccount {
			marker = " (default)"
		}
		authType := "oauth"
		if tok.Client == auth.PATClient {
			authType = "pat"
		}
		fmt.Fprintf(os.Stdout, "  - %s [%s] account:%d%s (since %s)\n",
			tok.Email, authType, tok.AccountID, marker, tok.CreatedAt.Format("2006-01-02"))
	}

	return nil
}

// AuthListCmd lists all authenticated accounts.
type AuthListCmd struct{}

func (c *AuthListCmd) Run() error {
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	if len(tokens) == 0 {
		fmt.Fprintln(os.Stdout, "No authenticated accounts.")
		return nil
	}

	// Show default account
	cfg, _ := config.ReadConfig()
	defaultAccount := ""
	if cfg != nil {
		defaultAccount = cfg.DefaultAccount
	}

	fmt.Fprintln(os.Stdout, "Authenticated accounts:")
	for _, tok := range tokens {
		marker := ""
		if tok.Email == defaultAccount {
			marker = " (default)"
		}
		authType := "oauth"
		if tok.Client == auth.PATClient {
			authType = "pat"
		}
		fmt.Fprintf(os.Stdout, "  %s [%s] client:%s account:%d%s (since %s)\n",
			tok.Email, authType, tok.Client, tok.AccountID, marker, tok.CreatedAt.Format("2006-01-02"))
	}

	return nil
}

// AuthSwitchCmd switches the default account.
type AuthSwitchCmd struct {
	Account string `arg:"" help:"Account email or alias to set as default"`
}

func (c *AuthSwitchCmd) Run() error {
	// Resolve alias if needed
	email, err := config.ResolveAccount(c.Account)
	if err != nil {
		// Not an alias; use as-is
		email = c.Account
	}

	// Verify account exists
	store, err := auth.OpenDefault()
	if err != nil {
		return fmt.Errorf("open keyring: %w", err)
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return fmt.Errorf("list tokens: %w", err)
	}

	found := false
	for _, tok := range tokens {
		if tok.Email == email {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("account %q not found; run 'harvest auth list' to see available accounts", email)
	}

	if err := config.SetDefaultAccount(email); err != nil {
		return fmt.Errorf("set default account: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Default account set to %s\n", email)

	return nil
}
