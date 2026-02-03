package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestFetchAccounts_Success(t *testing.T) {
	expected := AccountsResponse{
		User: struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
		}{
			ID:        123,
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		},
		Accounts: []HarvestAccount{
			{
				ID:      456,
				Name:    "Acme Corp",
				Product: "harvest",
			},
			{
				ID:      789,
				Name:    "Forecast Co",
				Product: "forecast",
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	// Override accounts endpoint for test
	origEndpoint := accountsEndpoint
	accountsEndpoint = srv.URL
	defer func() {
		accountsEndpoint = origEndpoint
	}()

	tok := &oauth2.Token{AccessToken: "test-token"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := FetchAccounts(ctx, tok)
	if err != nil {
		t.Fatalf("FetchAccounts error: %v", err)
	}

	if result.User.Email != expected.User.Email {
		t.Errorf("email = %q, want %q", result.User.Email, expected.User.Email)
	}

	if len(result.Accounts) != len(expected.Accounts) {
		t.Errorf("accounts count = %d, want %d", len(result.Accounts), len(expected.Accounts))
	}
}

func TestFetchAccounts_InvalidToken(t *testing.T) {
	ctx := context.Background()

	_, err := FetchAccounts(ctx, nil)
	if err == nil {
		t.Error("FetchAccounts(nil) should return error")
	}

	_, err = FetchAccounts(ctx, &oauth2.Token{})
	if err == nil {
		t.Error("FetchAccounts(empty token) should return error")
	}
}

func TestFetchAccounts_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	// Override accounts endpoint for test
	origEndpoint := accountsEndpoint
	accountsEndpoint = srv.URL
	defer func() {
		accountsEndpoint = origEndpoint
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tok := &oauth2.Token{AccessToken: "test-token"}
	_, err := FetchAccounts(ctx, tok)
	if err == nil {
		t.Error("FetchAccounts() should fail on server error")
	}
}

func TestSelectAccount_SingleAccount(t *testing.T) {
	accounts := []HarvestAccount{
		{
			ID:   123,
			Name: "Test Account",
			Product: "harvest",
		},
	}

	id, err := SelectAccount(accounts)
	if err != nil {
		t.Fatalf("SelectAccount() error = %v", err)
	}

	if id != 123 {
		t.Errorf("id = %d, want %d", id, 123)
	}
}

func TestSelectAccount_NoAccounts(t *testing.T) {
	_, err := SelectAccount(nil)
	if err == nil {
		t.Error("SelectAccount(nil) should return error")
	}

	_, err = SelectAccount([]HarvestAccount{})
	if err == nil {
		t.Error("SelectAccount([]) should return error")
	}
}

func TestFilterHarvestAccounts(t *testing.T) {
	tests := []struct {
		name     string
		accounts []HarvestAccount
		wantLen  int
		wantIDs  []int64
	}{
		{
			name: "mixed accounts",
			accounts: []HarvestAccount{
				{ID: 1, Name: "Harvest 1", Product: "harvest"},
				{ID: 2, Name: "Forecast 1", Product: "forecast"},
				{ID: 3, Name: "Harvest 2", Product: "Harvest"}, // uppercase
			},
			wantLen: 2,
			wantIDs: []int64{1, 3},
		},
		{
			name: "only forecast",
			accounts: []HarvestAccount{
				{ID: 1, Name: "Forecast 1", Product: "forecast"},
			},
			wantLen: 1, // returns all if no harvest accounts
			wantIDs: []int64{1},
		},
		{
			name: "only harvest",
			accounts: []HarvestAccount{
				{ID: 1, Name: "Harvest 1", Product: "harvest"},
			},
			wantLen: 1,
			wantIDs: []int64{1},
		},
		{
			name:     "empty",
			accounts: []HarvestAccount{},
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterHarvestAccounts(tt.accounts)

			if len(result) != tt.wantLen {
				t.Errorf("len(result) = %d, want %d", len(result), tt.wantLen)
			}

			if tt.wantIDs != nil {
				for i, id := range tt.wantIDs {
					if i >= len(result) {
						t.Errorf("missing result[%d]", i)
						continue
					}
					if result[i].ID != id {
						t.Errorf("result[%d].ID = %d, want %d", i, result[i].ID, id)
					}
				}
			}
		})
	}
}

func TestHarvestAccount_JSONMarshaling(t *testing.T) {
	acc := HarvestAccount{
		ID:      12345,
		Name:    "Test Company",
		Product: "harvest",
	}

	data, err := json.Marshal(acc)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded HarvestAccount
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.ID != acc.ID {
		t.Errorf("ID = %d, want %d", decoded.ID, acc.ID)
	}
	if decoded.Name != acc.Name {
		t.Errorf("Name = %q, want %q", decoded.Name, acc.Name)
	}
	if decoded.Product != acc.Product {
		t.Errorf("Product = %q, want %q", decoded.Product, acc.Product)
	}
}

func TestAccountsResponse_JSONMarshaling(t *testing.T) {
	resp := AccountsResponse{
		User: struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
		}{
			ID:        1,
			FirstName: "John",
			LastName:  "Doe",
			Email:     "john@example.com",
		},
		Accounts: []HarvestAccount{
			{ID: 100, Name: "Account 1"},
			{ID: 200, Name: "Account 2"},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded AccountsResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded.User.Email != resp.User.Email {
		t.Errorf("User.Email = %q, want %q", decoded.User.Email, resp.User.Email)
	}
	if len(decoded.Accounts) != len(resp.Accounts) {
		t.Errorf("len(Accounts) = %d, want %d", len(decoded.Accounts), len(resp.Accounts))
	}
}
