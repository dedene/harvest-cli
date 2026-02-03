package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/oauth2"
)

// accountsEndpoint is the Harvest accounts API endpoint (var for testing).
var accountsEndpoint = "https://id.getharvest.com/api/v2/accounts"

// HarvestAccount represents a Harvest account from the accounts endpoint.
type HarvestAccount struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Product string `json:"product"`
}

// AccountsResponse is the response from /accounts endpoint.
type AccountsResponse struct {
	User struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	} `json:"user"`
	Accounts []HarvestAccount `json:"accounts"`
}

// FetchAccounts retrieves the user's Harvest accounts using the token.
func FetchAccounts(ctx context.Context, tok *oauth2.Token) (*AccountsResponse, error) {
	if tok == nil || tok.AccessToken == "" {
		return nil, errors.New("invalid token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, accountsEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "harvest/0.1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch accounts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch accounts: status %d", resp.StatusCode)
	}

	var result AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode accounts: %w", err)
	}

	return &result, nil
}

// SelectAccount prompts user to select an account if multiple exist.
// Returns the selected account ID.
func SelectAccount(accounts []HarvestAccount) (int64, error) {
	if len(accounts) == 0 {
		return 0, errors.New("no accounts available")
	}

	// Filter to Harvest accounts (not Forecast)
	harvestAccounts := filterHarvestAccounts(accounts)
	if len(harvestAccounts) == 0 {
		return 0, errors.New("no Harvest accounts available (only Forecast accounts found)")
	}

	// Single account: auto-select
	if len(harvestAccounts) == 1 {
		return harvestAccounts[0].ID, nil
	}

	// Multiple accounts: prompt user
	fmt.Fprintln(os.Stderr, "\nMultiple Harvest accounts found:")
	for i, acc := range harvestAccounts {
		fmt.Fprintf(os.Stderr, "  [%d] %s (ID: %d)\n", i+1, acc.Name, acc.ID)
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "Select account (1-", len(harvestAccounts), "): ")

	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return 0, fmt.Errorf("read selection: %w", err)
	}

	input = strings.TrimSpace(input)
	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(harvestAccounts) {
		return 0, fmt.Errorf("invalid selection: %q", input)
	}

	return harvestAccounts[idx-1].ID, nil
}

// filterHarvestAccounts filters out non-Harvest accounts (e.g., Forecast).
func filterHarvestAccounts(accounts []HarvestAccount) []HarvestAccount {
	var result []HarvestAccount
	for _, acc := range accounts {
		// Harvest accounts have product == "harvest"
		if strings.EqualFold(acc.Product, "harvest") {
			result = append(result, acc)
		}
	}

	// If no explicit harvest accounts, return all (backwards compat)
	if len(result) == 0 {
		return accounts
	}

	return result
}
