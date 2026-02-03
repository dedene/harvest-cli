package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestGetMe(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/me" {
			t.Errorf("expected /users/me, got %s", r.URL.Path)
		}

		user := User{
			ID:              12345,
			FirstName:       "Test",
			LastName:        "User",
			Email:           "test@example.com",
			Timezone:        "Eastern Time (US & Canada)",
			IsActive:        true,
			WeeklyCapacity:  126000,
			Roles:           []string{"Developer"},
			AccessRoles:     []string{"member"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	user, err := client.GetMe(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 12345 {
		t.Errorf("expected ID 12345, got %d", user.ID)
	}
	if user.FullName() != "Test User" {
		t.Errorf("expected 'Test User', got '%s'", user.FullName())
	}
}

func TestListUsers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			t.Errorf("expected /users, got %s", r.URL.Path)
		}

		resp := UsersResponse{
			Users: []User{
				{ID: 1, FirstName: "Alice", LastName: "Smith", IsActive: true},
				{ID: 2, FirstName: "Bob", LastName: "Jones", IsActive: true},
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

	resp, err := client.ListUsers(context.Background(), UserListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(resp.Users))
	}
}

func TestListUsersWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("is_active") != "false" {
			t.Errorf("expected is_active=false, got %s", q.Get("is_active"))
		}

		resp := UsersResponse{Users: []User{}}
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

	isActive := false
	_, err := client.ListUsers(context.Background(), UserListOptions{
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users/999" {
			t.Errorf("expected /users/999, got %s", r.URL.Path)
		}

		user := User{
			ID:          999,
			FirstName:   "Jane",
			LastName:    "Doe",
			Email:       "jane@example.com",
			IsActive:    true,
			AccessRoles: []string{"administrator"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	user, err := client.GetUser(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 999 {
		t.Errorf("expected ID 999, got %d", user.ID)
	}
	if len(user.AccessRoles) != 1 || user.AccessRoles[0] != "administrator" {
		t.Errorf("expected administrator role, got %v", user.AccessRoles)
	}
}

func TestCreateUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var input UserInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if input.Email != "new@example.com" {
			t.Errorf("expected 'new@example.com', got '%s'", input.Email)
		}

		user := User{
			ID:        1000,
			FirstName: input.FirstName,
			LastName:  input.LastName,
			Email:     input.Email,
			IsActive:  true,
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	user, err := client.CreateUser(context.Background(), &UserInput{
		FirstName: "New",
		LastName:  "User",
		Email:     "new@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1000 {
		t.Errorf("expected ID 1000, got %d", user.ID)
	}
}

func TestUpdateUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/users/999" {
			t.Errorf("expected /users/999, got %s", r.URL.Path)
		}

		user := User{ID: 999, FirstName: "Updated", LastName: "Name"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	user, err := client.UpdateUser(context.Background(), 999, &UserInput{
		FirstName: "Updated",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.FirstName != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", user.FirstName)
	}
}

func TestDeleteUser(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/users/999" {
			t.Errorf("expected /users/999, got %s", r.URL.Path)
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

	err := client.DeleteUser(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUserFullName(t *testing.T) {
	user := User{FirstName: "John", LastName: "Smith"}
	if user.FullName() != "John Smith" {
		t.Errorf("expected 'John Smith', got '%s'", user.FullName())
	}
}
