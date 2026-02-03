package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestGetCompany(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/company" {
			t.Errorf("expected /company, got %s", r.URL.Path)
		}

		company := Company{
			BaseURI:              "https://example.harvestapp.com",
			FullDomain:           "example.harvestapp.com",
			Name:                 "Test Company",
			IsActive:             true,
			WeekStartDay:         "Monday",
			WantsTimestampTimers: false,
			TimeFormat:           "hours_minutes",
			DateFormat:           "%Y-%m-%d",
			PlanType:             "simple-v4",
			Clock:                "12h",
			WeeklyCapacity:       126000,
			ExpenseFeature:       true,
			InvoiceFeature:       true,
			EstimateFeature:      true,
			ApprovalFeature:      true,
			TeamFeature:          true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(company)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	company, err := client.GetCompany(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if company.Name != "Test Company" {
		t.Errorf("expected 'Test Company', got '%s'", company.Name)
	}
	if company.WeekStartDay != "Monday" {
		t.Errorf("expected 'Monday', got '%s'", company.WeekStartDay)
	}
	if company.WantsTimestampTimers {
		t.Error("expected WantsTimestampTimers=false")
	}
	if company.Clock != "12h" {
		t.Errorf("expected '12h', got '%s'", company.Clock)
	}
	if company.WeeklyCapacity != 126000 {
		t.Errorf("expected 126000, got %d", company.WeeklyCapacity)
	}
}

func TestUpdateCompany(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/company" {
			t.Errorf("expected /company, got %s", r.URL.Path)
		}

		var input CompanyUpdateInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		if input.WantsTimestampTimers == nil || !*input.WantsTimestampTimers {
			t.Error("expected WantsTimestampTimers=true")
		}
		if input.WeeklyCapacity == nil || *input.WeeklyCapacity != 144000 {
			t.Errorf("expected WeeklyCapacity=144000, got %v", input.WeeklyCapacity)
		}

		company := Company{
			Name:                 "Test Company",
			WantsTimestampTimers: true,
			WeeklyCapacity:       144000,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(company)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	wantsTimestamp := true
	capacity := 144000
	company, err := client.UpdateCompany(context.Background(), &CompanyUpdateInput{
		WantsTimestampTimers: &wantsTimestamp,
		WeeklyCapacity:       &capacity,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !company.WantsTimestampTimers {
		t.Error("expected WantsTimestampTimers=true")
	}
	if company.WeeklyCapacity != 144000 {
		t.Errorf("expected 144000, got %d", company.WeeklyCapacity)
	}
}

func TestCompanyFeatureFlags(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		company := Company{
			Name:            "Feature Test",
			ExpenseFeature:  false,
			InvoiceFeature:  true,
			EstimateFeature: false,
			ApprovalFeature: true,
			TeamFeature:     false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(company)
	}))
	defer ts.Close()

	client := NewClientWithBaseURL(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"}),
		12345,
		"test@example.com",
		ts.URL,
	)

	company, err := client.GetCompany(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if company.ExpenseFeature {
		t.Error("expected ExpenseFeature=false")
	}
	if !company.InvoiceFeature {
		t.Error("expected InvoiceFeature=true")
	}
	if company.EstimateFeature {
		t.Error("expected EstimateFeature=false")
	}
	if !company.ApprovalFeature {
		t.Error("expected ApprovalFeature=true")
	}
	if company.TeamFeature {
		t.Error("expected TeamFeature=false")
	}
}
