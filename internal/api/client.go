package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	// BaseURL is the Harvest API base URL.
	BaseURL = "https://api.harvestapp.com/v2"
	// ContentType for JSON requests.
	ContentType = "application/json"
)

// Client is the Harvest API client.
type Client struct {
	baseURL        string
	httpClient     *http.Client
	tokenSource    oauth2.TokenSource
	accountID      int64
	rateLimiter    *RateLimiter
	reportsLimiter *RateLimiter
	contactEmail   string
	version        string
}

// NewClient creates a new Harvest API client.
func NewClient(ts oauth2.TokenSource, accountID int64, contactEmail string) *Client {
	transport := NewRetryTransport(http.DefaultTransport)
	rateLimiter := NewGeneralRateLimiter()
	transport.RateLimiter = rateLimiter

	return &Client{
		baseURL:        BaseURL,
		tokenSource:    ts,
		accountID:      accountID,
		contactEmail:   contactEmail,
		version:        "0.1.0",
		httpClient:     &http.Client{Transport: transport},
		rateLimiter:    rateLimiter,
		reportsLimiter: NewReportsRateLimiter(),
	}
}

// NewClientWithBaseURL creates a client with a custom base URL (for testing).
func NewClientWithBaseURL(ts oauth2.TokenSource, accountID int64, contactEmail, baseURL string) *Client {
	client := NewClient(ts, accountID, contactEmail)
	if strings.TrimSpace(baseURL) != "" {
		client.baseURL = strings.TrimRight(baseURL, "/")
	}
	return client
}

// SetVersion sets the version string for User-Agent.
func (c *Client) SetVersion(version string) {
	c.version = version
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result, false)
}

// GetReports performs a GET request with reports rate limiting.
func (c *Client) GetReports(ctx context.Context, path string, result any) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, result, true)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.doRequest(ctx, http.MethodPost, path, body, result, false)
}

// Patch performs a PATCH request.
func (c *Client) Patch(ctx context.Context, path string, body, result any) error {
	return c.doRequest(ctx, http.MethodPatch, path, body, result, false)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) error {
	return c.doRequest(ctx, http.MethodDelete, path, nil, nil, false)
}

// doRequest executes an HTTP request with auth and error handling.
func (c *Client) doRequest(ctx context.Context, method, path string, body, result any, isReports bool) error {
	reqURL := c.baseURL + path

	// Proactive rate limiting for reports
	if isReports && c.reportsLimiter != nil {
		if err := c.reportsLimiter.Wait(ctx); err != nil {
			return err
		}
	}

	// Marshal body if present
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Get token
	tok, err := c.tokenSource.Token()
	if err != nil {
		return &AuthError{Err: err}
	}

	// Set required headers
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Harvest-Account-Id", strconv.FormatInt(c.accountID, 10))
	req.Header.Set("User-Agent", fmt.Sprintf("harvest/%s (%s)", c.version, c.contactEmail))
	req.Header.Set("Accept", ContentType)
	if body != nil {
		req.Header.Set("Content-Type", ContentType)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Update reports rate limiter
	if isReports && c.reportsLimiter != nil {
		c.reportsLimiter.UpdateFromHeaders(resp.Header)
	}

	// Handle errors
	if resp.StatusCode == http.StatusUnauthorized {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "unauthorized",
			Details:    "token may be expired or invalid; try logging in again",
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "forbidden",
			Details:    "insufficient permissions for this operation",
		}
	}

	if resp.StatusCode == http.StatusNotFound {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "not found",
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := 0
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			retryAfter, _ = strconv.Atoi(ra)
		}
		return &RateLimitError{RetryAfter: time.Duration(retryAfter) * time.Second}
	}

	if resp.StatusCode == http.StatusUnprocessableEntity {
		// Parse validation errors
		var errResp struct {
			Message string            `json:"message"`
			Errors  map[string]string `json:"errors"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil && len(errResp.Errors) > 0 {
			return &ValidationError{Fields: errResp.Errors}
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    "validation error",
			Details:    errResp.Message,
		}
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Details:    string(bodyBytes),
		}
	}

	// Decode response
	if result != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
