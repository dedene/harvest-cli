// Package errfmt provides user-friendly error formatting with actionable suggestions.
package errfmt

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/dedene/harvest-cli/internal/api"
)

// FormatError formats an error for user-friendly display with actionable suggestions.
func FormatError(err error) string {
	if err == nil {
		return ""
	}

	var sb strings.Builder

	// Get the base message
	msg := err.Error()

	// Check for specific error types and add suggestions
	switch {
	case IsAuthError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Authentication failed. Try 'harvest auth login'")

	case IsNotFoundError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Resource not found. Verify the ID exists.")

	case IsRateLimitError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Rate limit exceeded. Wait before retrying.")

	case isNetworkError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Network error. Check your connection.")

	case isConfigError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Config error. Try 'harvest config set'")

	case isPermissionError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Permission denied. Check account permissions.")

	case isCircuitBreakerError(err):
		sb.WriteString(msg)
		sb.WriteString("\n\nSuggestion: Service unavailable. Wait a moment and retry.")

	default:
		sb.WriteString(msg)
	}

	return sb.String()
}

// IsAuthError returns true if the error is authentication-related.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	// Check for api.AuthError
	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return true
	}

	// Check for API error with 401 status
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized
	}

	// Check for sentinel error
	if errors.Is(err, api.ErrNotAuthenticated) {
		return true
	}

	// Check error message for auth-related keywords
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not authenticated") ||
		strings.Contains(msg, "authentication error") ||
		strings.Contains(msg, "invalid token") ||
		strings.Contains(msg, "token expired")
}

// IsNotFoundError returns true if the error indicates a resource was not found.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for api.NotFoundError
	var notFoundErr *api.NotFoundError
	if errors.As(err, &notFoundErr) {
		return true
	}

	// Check for API error with 404 status
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusNotFound
	}

	// Check for sentinel error
	if errors.Is(err, api.ErrNotFound) {
		return true
	}

	return false
}

// IsRateLimitError returns true if the error indicates rate limiting.
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	// Check for api.RateLimitError
	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	// Check for API error with 429 status
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusTooManyRequests
	}

	// Check for sentinel error
	if errors.Is(err, api.ErrRateLimited) {
		return true
	}

	return false
}

// isNetworkError checks if the error is network-related.
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error (includes timeouts, connection refused, etc.)
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check for net.OpError
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	// Check error message for network-related keywords
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network unreachable") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "dial tcp")
}

// isConfigError checks if the error is configuration-related.
func isConfigError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "config") ||
		strings.Contains(msg, "configuration") ||
		strings.Contains(msg, "could not determine config path")
}

// isPermissionError checks if the error is permission-related (403).
func isPermissionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for API error with 403 status
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusForbidden
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "permission denied") ||
		strings.Contains(msg, "forbidden") ||
		strings.Contains(msg, "access denied")
}

// isCircuitBreakerError checks if the error is from the circuit breaker.
func isCircuitBreakerError(err error) bool {
	if err == nil {
		return false
	}

	var cbErr *api.CircuitBreakerError
	return errors.As(err, &cbErr)
}

// ErrorContext builds context from an error chain for debugging.
func ErrorContext(err error) []string {
	if err == nil {
		return nil
	}

	var contexts []string
	for err != nil {
		contexts = append(contexts, err.Error())
		err = errors.Unwrap(err)
	}
	return contexts
}

// WrapWithSuggestion wraps an error with a custom suggestion.
func WrapWithSuggestion(err error, suggestion string) error {
	if err == nil {
		return nil
	}
	return &suggestedError{err: err, suggestion: suggestion}
}

type suggestedError struct {
	err        error
	suggestion string
}

func (e *suggestedError) Error() string {
	return fmt.Sprintf("%v\n\nSuggestion: %s", e.err, e.suggestion)
}

func (e *suggestedError) Unwrap() error {
	return e.err
}
