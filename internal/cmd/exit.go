package cmd

import (
	"errors"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/auth"
)

// Exit codes follow standard conventions:
// 0 - Success
// 1 - General error
// 2 - Usage/parse error
// 3 - Authentication error
// 4 - API/rate limit error
// 5 - Configuration error

// ExitError wraps an error with an exit code.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ExitCode returns the appropriate exit code for an error.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	// Check for ExitError first
	var ee *ExitError
	if errors.As(err, &ee) && ee != nil {
		if ee.Code < 0 {
			return 1
		}
		return ee.Code
	}

	// Auth errors -> 3
	if errors.Is(err, auth.ErrNotAuthenticated) {
		return 3
	}

	// API errors -> 4
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		return 4
	}

	var rateLimitErr *api.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return 4
	}

	var authErr *api.AuthError
	if errors.As(err, &authErr) {
		return 3
	}

	// Default
	return 1
}
