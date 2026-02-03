package errfmt

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/dedene/harvest-cli/internal/api"
)

func TestFormatError_NilError(t *testing.T) {
	result := FormatError(nil)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFormatError_AuthError(t *testing.T) {
	err := &api.AuthError{Err: errors.New("token invalid")}
	result := FormatError(err)
	if !strings.Contains(result, "harvest auth login") {
		t.Errorf("expected auth suggestion, got %q", result)
	}
}

func TestFormatError_APIError401(t *testing.T) {
	err := &api.APIError{StatusCode: http.StatusUnauthorized, Message: "Unauthorized"}
	result := FormatError(err)
	if !strings.Contains(result, "harvest auth login") {
		t.Errorf("expected auth suggestion for 401, got %q", result)
	}
}

func TestFormatError_APIError403(t *testing.T) {
	err := &api.APIError{StatusCode: http.StatusForbidden, Message: "Forbidden"}
	result := FormatError(err)
	if !strings.Contains(result, "Permission denied") {
		t.Errorf("expected permission suggestion for 403, got %q", result)
	}
}

func TestFormatError_APIError404(t *testing.T) {
	err := &api.APIError{StatusCode: http.StatusNotFound, Message: "Not Found"}
	result := FormatError(err)
	if !strings.Contains(result, "Verify the ID exists") {
		t.Errorf("expected not found suggestion for 404, got %q", result)
	}
}

func TestFormatError_APIError429(t *testing.T) {
	err := &api.APIError{StatusCode: http.StatusTooManyRequests, Message: "Too Many Requests"}
	result := FormatError(err)
	if !strings.Contains(result, "Rate limit exceeded") {
		t.Errorf("expected rate limit suggestion for 429, got %q", result)
	}
}

func TestFormatError_RateLimitError(t *testing.T) {
	err := &api.RateLimitError{}
	result := FormatError(err)
	if !strings.Contains(result, "Rate limit exceeded") {
		t.Errorf("expected rate limit suggestion, got %q", result)
	}
}

func TestFormatError_NotFoundError(t *testing.T) {
	err := &api.NotFoundError{Resource: "project", ID: "123"}
	result := FormatError(err)
	if !strings.Contains(result, "Verify the ID exists") {
		t.Errorf("expected not found suggestion, got %q", result)
	}
}

func TestFormatError_CircuitBreakerError(t *testing.T) {
	err := &api.CircuitBreakerError{}
	result := FormatError(err)
	if !strings.Contains(result, "Wait a moment") {
		t.Errorf("expected circuit breaker suggestion, got %q", result)
	}
}

func TestFormatError_ConfigError(t *testing.T) {
	err := errors.New("could not determine config path")
	result := FormatError(err)
	if !strings.Contains(result, "harvest config set") {
		t.Errorf("expected config suggestion, got %q", result)
	}
}

func TestFormatError_NetworkError(t *testing.T) {
	err := errors.New("dial tcp: connection refused")
	result := FormatError(err)
	if !strings.Contains(result, "Check your connection") {
		t.Errorf("expected network suggestion, got %q", result)
	}
}

func TestFormatError_GenericError(t *testing.T) {
	err := errors.New("something went wrong")
	result := FormatError(err)
	if result != "something went wrong" {
		t.Errorf("expected plain message, got %q", result)
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"AuthError", &api.AuthError{Err: errors.New("test")}, true},
		{"APIError 401", &api.APIError{StatusCode: 401}, true},
		{"APIError 403", &api.APIError{StatusCode: 403}, false},
		{"ErrNotAuthenticated", api.ErrNotAuthenticated, true},
		{"wrapped auth", fmt.Errorf("wrap: %w", api.ErrNotAuthenticated), true},
		{"generic", errors.New("random error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"NotFoundError", &api.NotFoundError{Resource: "test"}, true},
		{"APIError 404", &api.APIError{StatusCode: 404}, true},
		{"APIError 401", &api.APIError{StatusCode: 401}, false},
		{"ErrNotFound", api.ErrNotFound, true},
		{"wrapped", fmt.Errorf("wrap: %w", api.ErrNotFound), true},
		{"generic", errors.New("random error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"RateLimitError", &api.RateLimitError{}, true},
		{"APIError 429", &api.APIError{StatusCode: 429}, true},
		{"APIError 500", &api.APIError{StatusCode: 500}, false},
		{"ErrRateLimited", api.ErrRateLimited, true},
		{"wrapped", fmt.Errorf("wrap: %w", api.ErrRateLimited), true},
		{"generic", errors.New("random error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimitError(tt.err); got != tt.expected {
				t.Errorf("IsRateLimitError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorContext(t *testing.T) {
	inner := errors.New("inner error")
	middle := fmt.Errorf("middle: %w", inner)
	outer := fmt.Errorf("outer: %w", middle)

	contexts := ErrorContext(outer)
	if len(contexts) != 3 {
		t.Errorf("expected 3 contexts, got %d", len(contexts))
	}
	if contexts[0] != "outer: middle: inner error" {
		t.Errorf("unexpected first context: %s", contexts[0])
	}
}

func TestWrapWithSuggestion(t *testing.T) {
	err := errors.New("base error")
	wrapped := WrapWithSuggestion(err, "Try this fix")

	if !strings.Contains(wrapped.Error(), "base error") {
		t.Error("wrapped error should contain base error")
	}
	if !strings.Contains(wrapped.Error(), "Try this fix") {
		t.Error("wrapped error should contain suggestion")
	}

	unwrapped := errors.Unwrap(wrapped)
	if unwrapped != err {
		t.Error("unwrap should return original error")
	}
}

func TestWrapWithSuggestion_Nil(t *testing.T) {
	result := WrapWithSuggestion(nil, "suggestion")
	if result != nil {
		t.Error("wrapping nil should return nil")
	}
}
