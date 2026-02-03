package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestListClients(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/clients" {
			t.Errorf("expected /clients, got %s", r.URL.Path)
		}

		resp := ClientsResponse{
			Clients: []HarvestClient{
				{
					ID:       1,
					Name:     "ABC Corp",
					IsActive: true,
					Address:  "123 Main St",
					Currency: "USD",
				},
				{
					ID:       2,
					Name:     "XYZ Inc",
					IsActive: true,
					Address:  "456 Oak Ave",
					Currency: "EUR",
				},
			},
			PerPage:      100,
			TotalPages:   1,
			TotalEntries: 2,
			Page:         1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	resp, err := client.ListClients(context.Background(), ClientListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(resp.Clients))
	}
	if resp.Clients[0].Name != "ABC Corp" {
		t.Errorf("expected 'ABC Corp', got '%s'", resp.Clients[0].Name)
	}
	if resp.Clients[1].Currency != "EUR" {
		t.Errorf("expected 'EUR', got '%s'", resp.Clients[1].Currency)
	}
}

func TestListClientsWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("is_active") != "true" {
			t.Errorf("expected is_active=true, got %s", q.Get("is_active"))
		}

		resp := ClientsResponse{Clients: []HarvestClient{}}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	isActive := true
	_, err := client.ListClients(context.Background(), ClientListOptions{
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clients/123" {
			t.Errorf("expected /clients/123, got %s", r.URL.Path)
		}

		c := HarvestClient{
			ID:           123,
			Name:         "Test Client",
			IsActive:     true,
			Address:      "789 Pine St\nSuite 100",
			StatementKey: "abc123def456",
			Currency:     "GBP",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	c, err := client.GetClient(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != 123 {
		t.Errorf("expected ID 123, got %d", c.ID)
	}
	if c.Currency != "GBP" {
		t.Errorf("expected 'GBP', got '%s'", c.Currency)
	}
}

func TestCreateClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var input ClientInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if input.Name != "New Client" {
			t.Errorf("expected 'New Client', got '%s'", input.Name)
		}

		c := HarvestClient{
			ID:       999,
			Name:     input.Name,
			IsActive: true,
		}
		if input.Currency != nil {
			c.Currency = *input.Currency
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	currency := "CAD"
	c, err := client.CreateClient(context.Background(), &ClientInput{
		Name:     "New Client",
		Currency: &currency,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ID != 999 {
		t.Errorf("expected ID 999, got %d", c.ID)
	}
	if c.Currency != "CAD" {
		t.Errorf("expected 'CAD', got '%s'", c.Currency)
	}
}

func TestUpdateClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/clients/123" {
			t.Errorf("expected /clients/123, got %s", r.URL.Path)
		}

		c := HarvestClient{ID: 123, Name: "Updated Client", IsActive: false}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(c)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	isActive := false
	c, err := client.UpdateClient(context.Background(), 123, &ClientInput{
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.IsActive {
		t.Error("expected IsActive=false")
	}
}

func TestDeleteClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/clients/123" {
			t.Errorf("expected /clients/123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	err := client.DeleteClient(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientListOptions_QueryParams(t *testing.T) {
	isActive := false
	opts := ClientListOptions{
		IsActive:     &isActive,
		UpdatedSince: "2024-01-01T00:00:00Z",
		Page:         3,
		PerPage:      25,
	}

	result := opts.QueryParams()
	if result == "" {
		t.Error("expected non-empty query params")
	}
}
