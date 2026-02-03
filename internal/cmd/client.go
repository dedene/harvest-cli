package cmd

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/auth"
	"github.com/dedene/harvest-cli/internal/config"
)

// NewClientFromFlags creates an API client from CLI flags.
func NewClientFromFlags(ctx context.Context, flags *RootFlags) (*api.Client, error) {
	ts, accountID, err := GetTokenSource(ctx, flags)
	if err != nil {
		return nil, err
	}

	// Get contact email from config for User-Agent
	cfg, _ := config.ReadConfig()
	contactEmail := ""
	if cfg != nil {
		contactEmail = cfg.ContactEmail
	}
	if contactEmail == "" {
		contactEmail = "harvest@example.com"
	}

	client := api.NewClient(ts, accountID, contactEmail)
	client.SetVersion(VersionString())

	return client, nil
}

// GetTokenSource returns an oauth2.TokenSource and account ID for API calls.
// Priority: env PAT > --account-id flag > keyring OAuth token
func GetTokenSource(ctx context.Context, flags *RootFlags) (oauth2.TokenSource, int64, error) {
	// 1. Check for PAT in environment
	if token, accountID, ok := auth.GetPATFromEnv(); ok {
		// Override account ID from flag if provided
		if flags != nil && flags.AccountID > 0 {
			accountID = flags.AccountID
		}
		return auth.NewPATTokenSource(token), accountID, nil
	}

	// 2. Resolve account email
	var email string
	var err error
	if flags != nil && flags.Account != "" {
		email, err = config.ResolveAccount(flags.Account)
		if err != nil {
			return nil, 0, err
		}
	} else {
		// Try to get default or only account
		email, err = resolveDefaultAccount()
		if err != nil {
			return nil, 0, err
		}
	}

	// 3. Determine client name
	clientName := ""
	if flags != nil {
		clientName = flags.Client
	}
	clientName, err = config.ResolveClientForAccount(email, clientName)
	if err != nil {
		return nil, 0, err
	}

	// 4. Open keyring and get token
	store, err := auth.OpenDefault()
	if err != nil {
		return nil, 0, fmt.Errorf("open keyring: %w", err)
	}

	// Check for PAT stored in keyring
	if pat, patAccountID, err := auth.GetPAT(store, email); err == nil && pat != "" {
		accountID := patAccountID
		if flags != nil && flags.AccountID > 0 {
			accountID = flags.AccountID
		}
		return auth.NewPATTokenSource(pat), accountID, nil
	}

	// Get OAuth token
	tok, err := store.GetToken(clientName, email)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", auth.ErrNotAuthenticated, err)
	}

	accountID := tok.AccountID
	if flags != nil && flags.AccountID > 0 {
		accountID = flags.AccountID
	}

	// Create OAuth token source
	ts := auth.NewTokenSource(store, clientName, email, nil)

	return ts, accountID, nil
}

// resolveDefaultAccount finds the account to use when none specified.
func resolveDefaultAccount() (string, error) {
	// Check config for default
	cfg, err := config.ReadConfig()
	if err == nil && cfg.DefaultAccount != "" {
		return config.ResolveAccount(cfg.DefaultAccount)
	}

	// Check for single authenticated account
	store, err := auth.OpenDefault()
	if err != nil {
		return "", fmt.Errorf("open keyring: %w", err)
	}

	tokens, err := store.ListTokens()
	if err != nil {
		return "", fmt.Errorf("list tokens: %w", err)
	}

	if len(tokens) == 0 {
		return "", fmt.Errorf("not authenticated; run 'harvest auth login'")
	}

	if len(tokens) == 1 {
		return tokens[0].Email, nil
	}

	return "", fmt.Errorf("multiple accounts found; specify --account or set default_account in config")
}
