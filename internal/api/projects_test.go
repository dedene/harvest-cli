package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestListProjects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/projects" {
			t.Errorf("expected /projects, got %s", r.URL.Path)
		}

		resp := ProjectsResponse{
			Projects: []Project{
				{
					ID:         42,
					Name:       "Test Project",
					Code:       "TP",
					IsActive:   true,
					IsBillable: true,
					BillBy:     "Project",
					BudgetBy:   "project",
					Client:     ProjectClientRef{ID: 1, Name: "Test Client", Currency: "USD"},
				},
			},
			PerPage:      100,
			TotalPages:   1,
			TotalEntries: 1,
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

	resp, err := client.ListProjects(context.Background(), ProjectListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(resp.Projects))
	}
	if resp.Projects[0].ID != 42 {
		t.Errorf("expected ID 42, got %d", resp.Projects[0].ID)
	}
	if resp.Projects[0].Name != "Test Project" {
		t.Errorf("expected 'Test Project', got '%s'", resp.Projects[0].Name)
	}
}

func TestListProjectsWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("is_active") != "true" {
			t.Errorf("expected is_active=true, got %s", q.Get("is_active"))
		}
		if q.Get("client_id") != "99" {
			t.Errorf("expected client_id=99, got %s", q.Get("client_id"))
		}

		resp := ProjectsResponse{Projects: []Project{}}
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
	_, err := client.ListProjects(context.Background(), ProjectListOptions{
		IsActive: &isActive,
		ClientID: 99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetProject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/42" {
			t.Errorf("expected /projects/42, got %s", r.URL.Path)
		}

		project := Project{
			ID:         42,
			Name:       "Single Project",
			IsActive:   true,
			IsBillable: true,
			BillBy:     "Project",
			BudgetBy:   "project",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	project, err := client.GetProject(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != 42 {
		t.Errorf("expected ID 42, got %d", project.ID)
	}
}

func TestCreateProject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var input ProjectInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if input.Name != "New Project" {
			t.Errorf("expected 'New Project', got '%s'", input.Name)
		}
		if input.ClientID != 10 {
			t.Errorf("expected client_id 10, got %d", input.ClientID)
		}

		project := Project{
			ID:       100,
			Name:     input.Name,
			IsActive: true,
			Client:   ProjectClientRef{ID: input.ClientID, Name: "Test Client"},
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	project, err := client.CreateProject(context.Background(), &ProjectInput{
		ClientID: 10,
		Name:     "New Project",
		BillBy:   "Project",
		BudgetBy: "project",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != 100 {
		t.Errorf("expected ID 100, got %d", project.ID)
	}
}

func TestUpdateProject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/projects/42" {
			t.Errorf("expected /projects/42, got %s", r.URL.Path)
		}

		project := Project{ID: 42, Name: "Updated Name"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(project)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	project, err := client.UpdateProject(context.Background(), 42, &ProjectInput{
		Name: "Updated Name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.Name != "Updated Name" {
		t.Errorf("expected 'Updated Name', got '%s'", project.Name)
	}
}

func TestDeleteProject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/projects/42" {
			t.Errorf("expected /projects/42, got %s", r.URL.Path)
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

	err := client.DeleteProject(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectListOptions_QueryParams(t *testing.T) {
	isActive := true
	opts := ProjectListOptions{
		IsActive: &isActive,
		ClientID: 55,
		Page:     2,
		PerPage:  50,
	}

	result := opts.QueryParams()
	if result == "" {
		t.Error("expected non-empty query params")
	}
}
