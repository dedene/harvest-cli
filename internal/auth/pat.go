package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

const (
	// PATClient is the client name used for PAT tokens in the keyring.
	PATClient = "pat"

	// PATEnvToken is the environment variable for the PAT.
	PATEnvToken = "HARVESTCLI_TOKEN"

	// PATEnvAccountID is the environment variable for the account ID.
	PATEnvAccountID = "HARVESTCLI_ACCOUNT_ID"
)

// usersEndpoint is the Harvest users/me API endpoint (var for testing).
var usersEndpoint = "https://api.harvestapp.com/v2/users/me"

// PATTokenSource is a token source that uses a Personal Access Token.
// PATs don't expire and don't need refresh.
type PATTokenSource struct {
	token string
}

// NewPATTokenSource creates a TokenSource from a Personal Access Token.
func NewPATTokenSource(token string) *PATTokenSource {
	return &PATTokenSource{token: token}
}

// Token returns an oauth2.Token containing the PAT.
// PATs don't expire, so we set a far-future expiry.
func (p *PATTokenSource) Token() (*oauth2.Token, error) {
	if p.token == "" {
		return nil, ErrNotAuthenticated
	}

	return &oauth2.Token{
		AccessToken: p.token,
		TokenType:   "Bearer",
		// PATs don't expire; set far-future expiry to satisfy oauth2.Token
		Expiry: time.Now().Add(10 * 365 * 24 * time.Hour),
	}, nil
}

// StorePAT stores a PAT in the keyring for the given email/account.
func StorePAT(store Store, email string, accountID int64, token string) error {
	if email == "" {
		return errMissingEmail
	}
	if accountID == 0 {
		return errMissingAccountID
	}
	if token == "" {
		return ErrNotAuthenticated
	}

	// Use a special Token struct that satisfies the store interface.
	// We store the PAT as the "RefreshToken" since that's what gets persisted.
	tok := Token{
		Email:        email,
		AccountID:    accountID,
		RefreshToken: token, // PAT stored as refresh token
		CreatedAt:    time.Now().UTC(),
	}

	return store.SetToken(PATClient, email, accountID, tok)
}

// GetPAT retrieves a PAT from the keyring for the given email.
func GetPAT(store Store, email string) (token string, accountID int64, err error) {
	tok, err := store.GetToken(PATClient, email)
	if err != nil {
		return "", 0, err
	}

	return tok.RefreshToken, tok.AccountID, nil
}

// DeletePAT removes a PAT from the keyring for the given email.
func DeletePAT(store Store, email string) error {
	return store.DeleteToken(PATClient, email)
}

// userMeResponse is the response from /users/me endpoint.
type userMeResponse struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

// ValidatePAT validates a PAT by calling /users/me endpoint.
// Returns the user's email if valid.
func ValidatePAT(ctx context.Context, token string, accountID int64) (email string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersEndpoint, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Harvest-Account-Id", strconv.FormatInt(accountID, 10))
	req.Header.Set("User-Agent", "harvest/pat-validation")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", fmt.Errorf("invalid token or account ID")
	}

	if resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("token lacks required permissions")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var user userMeResponse
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if user.Email == "" {
		return "", fmt.Errorf("empty email in response")
	}

	return user.Email, nil
}

// GetPATFromEnv checks for HARVESTCLI_TOKEN and HARVESTCLI_ACCOUNT_ID environment variables.
// Returns token, accountID, ok.
func GetPATFromEnv() (token string, accountID int64, ok bool) {
	token = os.Getenv(PATEnvToken)
	if token == "" {
		return "", 0, false
	}

	accountIDStr := os.Getenv(PATEnvAccountID)
	if accountIDStr == "" {
		return "", 0, false
	}

	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil || accountID <= 0 {
		return "", 0, false
	}

	return token, accountID, true
}
