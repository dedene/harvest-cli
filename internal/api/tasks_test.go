package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestListTasks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/tasks" {
			t.Errorf("expected /tasks, got %s", r.URL.Path)
		}

		resp := TasksResponse{
			Tasks: []Task{
				{
					ID:                1,
					Name:              "Development",
					BillableByDefault: true,
					DefaultHourlyRate: 100.0,
					IsDefault:         true,
					IsActive:          true,
				},
				{
					ID:                2,
					Name:              "Research",
					BillableByDefault: false,
					DefaultHourlyRate: 0,
					IsDefault:         false,
					IsActive:          true,
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

	resp, err := client.ListTasks(context.Background(), TaskListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.Tasks))
	}
	if resp.Tasks[0].Name != "Development" {
		t.Errorf("expected 'Development', got '%s'", resp.Tasks[0].Name)
	}
}

func TestListTasksWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("is_active") != "true" {
			t.Errorf("expected is_active=true, got %s", q.Get("is_active"))
		}

		resp := TasksResponse{Tasks: []Task{}}
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
	_, err := client.ListTasks(context.Background(), TaskListOptions{
		IsActive: &isActive,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTask(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tasks/123" {
			t.Errorf("expected /tasks/123, got %s", r.URL.Path)
		}

		task := Task{
			ID:                123,
			Name:              "Programming",
			BillableByDefault: true,
			DefaultHourlyRate: 150.0,
			IsDefault:         true,
			IsActive:          true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	task, err := client.GetTask(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != 123 {
		t.Errorf("expected ID 123, got %d", task.ID)
	}
	if task.DefaultHourlyRate != 150.0 {
		t.Errorf("expected rate 150.0, got %f", task.DefaultHourlyRate)
	}
}

func TestCreateTask(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var input TaskInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if input.Name != "New Task" {
			t.Errorf("expected 'New Task', got '%s'", input.Name)
		}

		task := Task{
			ID:       999,
			Name:     input.Name,
			IsActive: true,
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	task, err := client.CreateTask(context.Background(), &TaskInput{
		Name: "New Task",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID != 999 {
		t.Errorf("expected ID 999, got %d", task.ID)
	}
}

func TestUpdateTask(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/tasks/123" {
			t.Errorf("expected /tasks/123, got %s", r.URL.Path)
		}

		task := Task{ID: 123, Name: "Updated Task", IsDefault: true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	isDefault := true
	task, err := client.UpdateTask(context.Background(), 123, &TaskInput{
		IsDefault: &isDefault,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !task.IsDefault {
		t.Error("expected IsDefault=true")
	}
}

func TestDeleteTask(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/tasks/123" {
			t.Errorf("expected /tasks/123, got %s", r.URL.Path)
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

	err := client.DeleteTask(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
