package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Exit codes for CLI.
const (
	ExitSuccess   = 0
	ExitError     = 1
	ExitUsage     = 2
	ExitAuth      = 3
	ExitNotFound  = 4
	ExitRateLimit = 5
)

// Sentinel errors.
var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrRateLimited      = errors.New("rate limit exceeded")
	ErrNotFound         = errors.New("not found")
)

// APIError represents an error response from the Harvest API.
type APIError struct {
	StatusCode int
	Message    string
	Details    string
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Details)
	}
	return e.Message
}

// ExitCode returns the appropriate CLI exit code for this error.
func (e *APIError) ExitCode() int {
	switch e.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return ExitAuth
	case http.StatusNotFound:
		return ExitNotFound
	case http.StatusTooManyRequests:
		return ExitRateLimit
	default:
		return ExitError
	}
}

// RateLimitError indicates the API rate limit was exceeded.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("rate limit exceeded, retry after %v", e.RetryAfter)
	}
	return "rate limit exceeded"
}

// CircuitBreakerError indicates the circuit breaker is open.
type CircuitBreakerError struct{}

func (e *CircuitBreakerError) Error() string {
	return "circuit breaker is open: too many consecutive failures"
}

// AuthError wraps authentication-related errors.
type AuthError struct {
	Err error
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("authentication error: %v", e.Err)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// ValidationError contains field-level validation errors.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	if len(e.Fields) == 0 {
		return "validation error"
	}
	return fmt.Sprintf("validation error: %d field(s) invalid", len(e.Fields))
}

// NotFoundError indicates a resource was not found.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s '%s' not found", e.Resource, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Resource)
}

// ExitCode maps an error to an appropriate CLI exit code.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.ExitCode()
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return ExitAuth
	}

	var notFoundErr *NotFoundError
	if errors.As(err, &notFoundErr) {
		return ExitNotFound
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return ExitRateLimit
	}

	var cbErr *CircuitBreakerError
	if errors.As(err, &cbErr) {
		return ExitError
	}

	return ExitError
}
