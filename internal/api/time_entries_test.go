package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestListTimeEntries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/time_entries" {
			t.Errorf("expected /time_entries, got %s", r.URL.Path)
		}

		resp := TimeEntriesResponse{
			TimeEntries: []TimeEntry{
				{
					ID:        123,
					SpentDate: "2024-01-15",
					Hours:     2.5,
					Notes:     "Test entry",
					IsRunning: false,
					User:      UserRef{ID: 1, Name: "Test User"},
					Project:   ProjectRef{ID: 10, Name: "Test Project"},
					Task:      TaskRef{ID: 100, Name: "Development"},
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

	resp, err := client.ListTimeEntries(context.Background(), TimeEntryListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.TimeEntries) != 1 {
		t.Fatalf("expected 1 time entry, got %d", len(resp.TimeEntries))
	}
	if resp.TimeEntries[0].ID != 123 {
		t.Errorf("expected ID 123, got %d", resp.TimeEntries[0].ID)
	}
	if resp.TimeEntries[0].Hours != 2.5 {
		t.Errorf("expected 2.5 hours, got %f", resp.TimeEntries[0].Hours)
	}
}

func TestListTimeEntriesWithFilters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("from") != "2024-01-01" {
			t.Errorf("expected from=2024-01-01, got %s", q.Get("from"))
		}
		if q.Get("to") != "2024-01-31" {
			t.Errorf("expected to=2024-01-31, got %s", q.Get("to"))
		}
		if q.Get("project_id") != "42" {
			t.Errorf("expected project_id=42, got %s", q.Get("project_id"))
		}
		if q.Get("is_running") != "true" {
			t.Errorf("expected is_running=true, got %s", q.Get("is_running"))
		}

		resp := TimeEntriesResponse{TimeEntries: []TimeEntry{}}
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

	isRunning := true
	_, err := client.ListTimeEntries(context.Background(), TimeEntryListOptions{
		From:      "2024-01-01",
		To:        "2024-01-31",
		ProjectID: 42,
		IsRunning: &isRunning,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTimeEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/time_entries/123" {
			t.Errorf("expected /time_entries/123, got %s", r.URL.Path)
		}

		entry := TimeEntry{
			ID:        123,
			SpentDate: "2024-01-15",
			Hours:     3.0,
			Notes:     "Single entry",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	entry, err := client.GetTimeEntry(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != 123 {
		t.Errorf("expected ID 123, got %d", entry.ID)
	}
}

func TestCreateTimeEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/time_entries" {
			t.Errorf("expected /time_entries, got %s", r.URL.Path)
		}

		var input TimeEntryInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if input.ProjectID != 42 {
			t.Errorf("expected project_id 42, got %d", input.ProjectID)
		}
		if input.TaskID != 100 {
			t.Errorf("expected task_id 100, got %d", input.TaskID)
		}

		entry := TimeEntry{
			ID:        999,
			SpentDate: input.SpentDate,
			Hours:     *input.Hours,
			Project:   ProjectRef{ID: input.ProjectID, Name: "Test"},
			Task:      TaskRef{ID: input.TaskID, Name: "Dev"},
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	hours := 2.0
	entry, err := client.CreateTimeEntry(context.Background(), &TimeEntryInput{
		ProjectID: 42,
		TaskID:    100,
		SpentDate: "2024-01-15",
		Hours:     &hours,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != 999 {
		t.Errorf("expected ID 999, got %d", entry.ID)
	}
}

func TestUpdateTimeEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/time_entries/123" {
			t.Errorf("expected /time_entries/123, got %s", r.URL.Path)
		}

		entry := TimeEntry{ID: 123, Notes: "Updated notes"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	notes := "Updated notes"
	entry, err := client.UpdateTimeEntry(context.Background(), 123, &TimeEntryInput{
		Notes: &notes,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Notes != "Updated notes" {
		t.Errorf("expected 'Updated notes', got '%s'", entry.Notes)
	}
}

func TestDeleteTimeEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/time_entries/123" {
			t.Errorf("expected /time_entries/123, got %s", r.URL.Path)
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

	err := client.DeleteTimeEntry(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStopTimeEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/time_entries/123/stop" {
			t.Errorf("expected /time_entries/123/stop, got %s", r.URL.Path)
		}

		entry := TimeEntry{
			ID:        123,
			IsRunning: false,
			Hours:     1.5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	entry, err := client.StopTimeEntry(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.IsRunning {
		t.Error("expected IsRunning=false")
	}
}

func TestRestartTimeEntry(t *testing.T) {
	now := time.Now()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/time_entries/123/restart" {
			t.Errorf("expected /time_entries/123/restart, got %s", r.URL.Path)
		}

		entry := TimeEntry{
			ID:             123,
			IsRunning:      true,
			TimerStartedAt: &now,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entry)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	entry, err := client.RestartTimeEntry(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !entry.IsRunning {
		t.Error("expected IsRunning=true")
	}
}

func TestGetRunningTimeEntry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("is_running") != "true" {
			t.Errorf("expected is_running=true, got %s", q.Get("is_running"))
		}

		resp := TimeEntriesResponse{
			TimeEntries: []TimeEntry{
				{ID: 456, IsRunning: true, Hours: 0.5},
			},
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

	entry, err := client.GetRunningTimeEntry(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected a running entry")
	}
	if entry.ID != 456 {
		t.Errorf("expected ID 456, got %d", entry.ID)
	}
}

func TestGetRunningTimeEntry_None(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TimeEntriesResponse{TimeEntries: []TimeEntry{}}
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

	entry, err := client.GetRunningTimeEntry(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry != nil {
		t.Error("expected nil entry when nothing running")
	}
}

func TestTimeEntryListOptions_QueryParams(t *testing.T) {
	tests := []struct {
		name     string
		opts     TimeEntryListOptions
		contains []string
	}{
		{
			name: "empty",
			opts: TimeEntryListOptions{},
		},
		{
			name: "from and to",
			opts: TimeEntryListOptions{From: "2024-01-01", To: "2024-01-31"},
			contains: []string{"from=2024-01-01", "to=2024-01-31"},
		},
		{
			name: "user_id",
			opts: TimeEntryListOptions{UserID: 123},
			contains: []string{"user_id=123"},
		},
		{
			name: "approval_status",
			opts: TimeEntryListOptions{ApprovalStatus: "submitted"},
			contains: []string{"approval_status=submitted"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.opts.QueryParams()
			for _, c := range tt.contains {
				if len(result) > 0 && !containsString(result, c) {
					t.Errorf("expected query params to contain %q, got %q", c, result)
				}
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i < len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
