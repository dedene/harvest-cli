package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter tracks API rate limits from response headers.
// Supports both reactive (general API) and proactive (reports) modes.
type RateLimiter struct {
	mu        sync.Mutex
	limit     int           // requests per window
	window    time.Duration // window duration
	remaining int
	resetAt   time.Time
	proactive bool // if true, proactively wait before requests
}

// NewRateLimiter creates a rate limiter.
// proactive=true for reports API (must track and wait).
func NewRateLimiter(limit int, window time.Duration, proactive bool) *RateLimiter {
	return &RateLimiter{
		limit:     limit,
		window:    window,
		remaining: limit,
		proactive: proactive,
	}
}

// NewGeneralRateLimiter creates a rate limiter for general API calls.
// Harvest: 100 requests / 15 seconds (reactive).
func NewGeneralRateLimiter() *RateLimiter {
	return NewRateLimiter(100, 15*time.Second, false)
}

// NewReportsRateLimiter creates a rate limiter for reports API.
// Harvest: 100 requests / 15 minutes (proactive).
func NewReportsRateLimiter() *RateLimiter {
	return NewRateLimiter(100, 15*time.Minute, true)
}

// Wait blocks if proactive limiting is needed.
// For proactive limiters, spreads requests across the window.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if !rl.proactive {
		return nil
	}

	rl.mu.Lock()
	remaining := rl.remaining
	resetAt := rl.resetAt
	limit := rl.limit
	rl.mu.Unlock()

	// No rate info yet or window expired
	if resetAt.IsZero() || time.Now().After(resetAt) {
		return nil
	}

	// If exhausted, wait for reset
	if remaining <= 0 {
		return sleepUntil(ctx, resetAt)
	}

	// Spread remaining requests across remaining time
	timeLeft := time.Until(resetAt)
	if timeLeft <= 0 || remaining >= limit {
		return nil
	}

	// Calculate interval to spread requests
	interval := timeLeft / time.Duration(remaining)
	if interval <= 0 {
		return nil
	}

	return sleep(ctx, interval)
}

// UpdateFromHeaders updates rate limit state from Harvest API headers.
// Headers: X-RateLimit-Limit, X-RateLimit-Remaining, Retry-After.
func (rl *RateLimiter) UpdateFromHeaders(h http.Header) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v := h.Get("X-RateLimit-Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.limit = n
		}
	}

	if v := h.Get("X-RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			rl.remaining = n
		}
	}

	// Calculate reset time from remaining and window
	if rl.remaining < rl.limit && rl.resetAt.Before(time.Now()) {
		rl.resetAt = time.Now().Add(rl.window)
	}
}

// Remaining returns current remaining requests.
func (rl *RateLimiter) Remaining() int {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.remaining
}

func sleepUntil(ctx context.Context, t time.Time) error {
	d := time.Until(t)
	if d <= 0 {
		return nil
	}
	return sleep(ctx, d)
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("rate limit wait interrupted: %w", ctx.Err())
	}
}
