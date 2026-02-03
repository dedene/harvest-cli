package api

import (
	"context"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

const (
	// DefaultMaxRetries429 is max retries for rate limit errors.
	DefaultMaxRetries429 = 10
	// DefaultMaxRetries5xx is max retries for server errors.
	DefaultMaxRetries5xx = 3
	// DefaultBaseDelay is initial backoff delay.
	DefaultBaseDelay = 1 * time.Second
	// ServerErrorRetryDelay is delay between 5xx retries.
	ServerErrorRetryDelay = 2 * time.Second
)

// RetryTransport wraps an http.RoundTripper with retry logic.
type RetryTransport struct {
	Base           http.RoundTripper
	MaxRetries429  int
	MaxRetries5xx  int
	BaseDelay      time.Duration
	CircuitBreaker *CircuitBreaker
	RateLimiter    *RateLimiter
}

// NewRetryTransport creates a transport with sensible defaults.
func NewRetryTransport(base http.RoundTripper) *RetryTransport {
	if base == nil {
		base = http.DefaultTransport
	}

	return &RetryTransport{
		Base:           base,
		MaxRetries429:  DefaultMaxRetries429,
		MaxRetries5xx:  DefaultMaxRetries5xx,
		BaseDelay:      DefaultBaseDelay,
		CircuitBreaker: NewCircuitBreaker(),
	}
}

// RoundTrip executes the request with retry logic.
func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Check circuit breaker
	if t.CircuitBreaker != nil && t.CircuitBreaker.IsOpen() {
		return nil, &CircuitBreakerError{}
	}

	// Ensure body can be replayed for retries
	if err := ensureReplayableBody(req); err != nil {
		return nil, err
	}

	var resp *http.Response
	var err error
	retries429 := 0
	retries5xx := 0

	for {
		// Reset body for retry
		if req.GetBody != nil {
			if req.Body != nil {
				_ = req.Body.Close()
			}
			body, getErr := req.GetBody()
			if getErr != nil {
				return nil, fmt.Errorf("reset request body: %w", getErr)
			}
			req.Body = body
		}

		// Proactive rate limiting
		if t.RateLimiter != nil {
			if err := t.RateLimiter.Wait(req.Context()); err != nil {
				return nil, err
			}
		}

		resp, err = t.Base.RoundTrip(req)
		if err != nil {
			return nil, fmt.Errorf("round trip: %w", err)
		}

		// Update rate limiter from response
		if t.RateLimiter != nil {
			t.RateLimiter.UpdateFromHeaders(resp.Header)
		}

		// Success
		if resp.StatusCode < 400 {
			if t.CircuitBreaker != nil {
				t.CircuitBreaker.RecordSuccess()
			}
			return resp, nil
		}

		// Rate limited (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			if retries429 >= t.MaxRetries429 {
				return resp, nil
			}

			delay := t.calculateBackoff(retries429, resp)
			drainAndClose(resp.Body)

			if err := t.sleep(req.Context(), delay); err != nil {
				return nil, err
			}

			retries429++
			continue
		}

		// Server error (5xx)
		if resp.StatusCode >= 500 {
			if t.CircuitBreaker != nil {
				t.CircuitBreaker.RecordFailure()
			}

			if retries5xx >= t.MaxRetries5xx {
				return resp, nil
			}

			delay := t.calculateExponentialBackoff(retries5xx)
			drainAndClose(resp.Body)

			if err := t.sleep(req.Context(), delay); err != nil {
				return nil, err
			}

			retries5xx++
			continue
		}

		// Other errors (4xx except 429) - no retry
		return resp, nil
	}
}

// calculateBackoff determines wait time for 429 responses.
// Uses Retry-After header if present, otherwise exponential backoff.
func (t *RetryTransport) calculateBackoff(attempt int, resp *http.Response) time.Duration {
	if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
		// Try parsing as seconds
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			if seconds < 0 {
				return 0
			}
			return time.Duration(seconds) * time.Second
		}
		// Try parsing as HTTP date
		if parsed, err := http.ParseTime(retryAfter); err == nil {
			d := time.Until(parsed)
			if d < 0 {
				return 0
			}
			return d
		}
	}

	return t.calculateExponentialBackoff(attempt)
}

// calculateExponentialBackoff returns backoff with jitter.
func (t *RetryTransport) calculateExponentialBackoff(attempt int) time.Duration {
	if t.BaseDelay <= 0 {
		return 0
	}

	// Exponential: baseDelay * 2^attempt
	baseDelay := t.BaseDelay * time.Duration(1<<attempt)
	if baseDelay <= 0 {
		return 0
	}

	// Add jitter: 0-50% of base delay
	jitterRange := baseDelay / 2
	if jitterRange <= 0 {
		return baseDelay
	}

	jitter := time.Duration(rand.Int64N(int64(jitterRange))) //nolint:gosec // non-crypto jitter
	return baseDelay + jitter
}

func (t *RetryTransport) sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("sleep interrupted: %w", ctx.Err())
	}
}

// ensureReplayableBody buffers the request body for retries.
func ensureReplayableBody(req *http.Request) error {
	if req == nil || req.Body == nil || req.GetBody != nil {
		return nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	_ = req.Body.Close()

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(newBytesReader(bodyBytes)), nil
	}
	req.Body = io.NopCloser(newBytesReader(bodyBytes))

	return nil
}

// bytesReader is a simple bytes reader for body replay.
type bytesReader struct {
	data []byte
	pos  int
}

func newBytesReader(data []byte) *bytesReader {
	return &bytesReader{data: data}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// drainAndClose drains and closes a response body.
func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(body, 1<<20))
	_ = body.Close()
}
