package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// staticTokenSource returns a fixed token.
type staticTokenSource struct {
	token string
}

func (s *staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token}, nil
}

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("wrong auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Harvest-Account-Id") != "12345" {
			t.Errorf("wrong account id: %s", r.Header.Get("Harvest-Account-Id"))
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing user-agent")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"Test"}`))
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	var result map[string]any
	err := client.Get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if result["name"] != "Test" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestClientPost(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("wrong content-type: %s", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if body["name"] != "New Item" {
			t.Errorf("unexpected body: %v", body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":2,"name":"New Item"}`))
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	body := map[string]any{"name": "New Item"}
	var result map[string]any
	err := client.Post(context.Background(), "/items", body, &result)
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if result["id"] != float64(2) {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestClientPatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"Updated"}`))
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	body := map[string]any{"name": "Updated"}
	var result map[string]any
	err := client.Patch(context.Background(), "/items/1", body, &result)
	if err != nil {
		t.Fatalf("Patch failed: %v", err)
	}
}

func TestClientDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	err := client.Delete(context.Background(), "/items/1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestClient401Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "bad-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	err := client.Get(context.Background(), "/test", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", apiErr.StatusCode)
	}
	if ExitCode(err) != ExitAuth {
		t.Errorf("expected ExitAuth, got %d", ExitCode(err))
	}
}

func TestClient404Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	err := client.Get(context.Background(), "/notfound", nil)
	if err == nil {
		t.Fatal("expected error")
	}

	if ExitCode(err) != ExitNotFound {
		t.Errorf("expected ExitNotFound, got %d", ExitCode(err))
	}
}

func TestClientValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"message":"Invalid","errors":{"name":"is required"}}`))
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	err := client.Post(context.Background(), "/items", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}

	valErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if valErr.Fields["name"] != "is required" {
		t.Errorf("unexpected fields: %v", valErr.Fields)
	}
}

func TestTransportRetry429(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)

	var result map[string]any
	err := client.Get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestTransportRetry5xx(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	ts := &staticTokenSource{token: "test-token"}
	client := NewClientWithBaseURL(ts, 12345, "test@example.com", srv.URL)
	// Reduce retry delay for test
	if rt, ok := client.httpClient.Transport.(*RetryTransport); ok {
		rt.BaseDelay = 1 * time.Millisecond
	}

	var result map[string]any
	err := client.Get(context.Background(), "/test", &result)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker()

	// Should start closed
	if cb.IsOpen() {
		t.Error("circuit should start closed")
	}

	// Record failures until open
	for i := 0; i < CircuitBreakerThreshold-1; i++ {
		if cb.RecordFailure() {
			t.Errorf("circuit opened too early at %d", i)
		}
	}

	// This one should open it
	if !cb.RecordFailure() {
		t.Error("circuit should have opened")
	}

	if !cb.IsOpen() {
		t.Error("circuit should be open")
	}

	// Success should close it
	cb.RecordSuccess()
	if cb.IsOpen() {
		t.Error("circuit should be closed after success")
	}
}

func TestRateLimiterUpdateFromHeaders(t *testing.T) {
	rl := NewGeneralRateLimiter()

	h := http.Header{}
	h.Set("X-RateLimit-Limit", "100")
	h.Set("X-RateLimit-Remaining", "50")

	rl.UpdateFromHeaders(h)

	if rl.Remaining() != 50 {
		t.Errorf("expected remaining=50, got %d", rl.Remaining())
	}
}

func TestRateLimiterProactiveWait(t *testing.T) {
	rl := NewRateLimiter(10, 1*time.Second, true)

	// Set up state with low remaining
	rl.mu.Lock()
	rl.remaining = 2
	rl.resetAt = time.Now().Add(100 * time.Millisecond)
	rl.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := rl.Wait(ctx)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	// Should have waited ~50ms (100ms / 2 remaining)
	if elapsed < 40*time.Millisecond {
		t.Errorf("waited too short: %v", elapsed)
	}
}

func TestExitCodes(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{nil, ExitSuccess},
		{&APIError{StatusCode: 401}, ExitAuth},
		{&APIError{StatusCode: 403}, ExitAuth},
		{&APIError{StatusCode: 404}, ExitNotFound},
		{&APIError{StatusCode: 429}, ExitRateLimit},
		{&APIError{StatusCode: 500}, ExitError},
		{&AuthError{}, ExitAuth},
		{&NotFoundError{Resource: "user"}, ExitNotFound},
		{&RateLimitError{}, ExitRateLimit},
		{&CircuitBreakerError{}, ExitError},
	}

	for _, tt := range tests {
		got := ExitCode(tt.err)
		if got != tt.want {
			t.Errorf("ExitCode(%T) = %d, want %d", tt.err, got, tt.want)
		}
	}
}

// sequenceTokenSource returns different tokens on each call.
type sequenceTokenSource struct {
	mu    sync.Mutex
	count int
}

func (s *sequenceTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	token := "token1"
	if s.count > 1 {
		token = "token2"
	}
	return &oauth2.Token{AccessToken: token}, nil
}
