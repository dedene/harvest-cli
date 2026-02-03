package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// ExpensesResponse is the paginated response for expenses.
type ExpensesResponse struct {
	Expenses     []Expense       `json:"expenses"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// ExpenseListOptions filters expense list requests.
type ExpenseListOptions struct {
	UserID         int64
	ClientID       int64
	ProjectID      int64
	IsBilled       *bool
	ApprovalStatus string // "unsubmitted", "submitted", "approved"
	UpdatedSince   string
	From           string
	To             string
	Page           int
	PerPage        int
}

// QueryParams converts options to URL query parameters.
func (o ExpenseListOptions) QueryParams() string {
	v := url.Values{}
	if o.UserID > 0 {
		v.Set("user_id", strconv.FormatInt(o.UserID, 10))
	}
	if o.ClientID > 0 {
		v.Set("client_id", strconv.FormatInt(o.ClientID, 10))
	}
	if o.ProjectID > 0 {
		v.Set("project_id", strconv.FormatInt(o.ProjectID, 10))
	}
	if o.IsBilled != nil {
		v.Set("is_billed", strconv.FormatBool(*o.IsBilled))
	}
	if o.ApprovalStatus != "" {
		v.Set("approval_status", o.ApprovalStatus)
	}
	if o.UpdatedSince != "" {
		v.Set("updated_since", o.UpdatedSince)
	}
	if o.From != "" {
		v.Set("from", o.From)
	}
	if o.To != "" {
		v.Set("to", o.To)
	}
	if o.Page > 0 {
		v.Set("page", strconv.Itoa(o.Page))
	}
	if o.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(o.PerPage))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// ListExpenses returns a paginated list of expenses.
func (c *Client) ListExpenses(ctx context.Context, opts ExpenseListOptions) (*ExpensesResponse, error) {
	path := "/expenses" + opts.QueryParams()
	var resp ExpensesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetExpense retrieves a single expense by ID.
func (c *Client) GetExpense(ctx context.Context, id int64) (*Expense, error) {
	path := fmt.Sprintf("/expenses/%d", id)
	var expense Expense
	if err := c.Get(ctx, path, &expense); err != nil {
		return nil, err
	}
	return &expense, nil
}

// CreateExpense creates a new expense.
func (c *Client) CreateExpense(ctx context.Context, input *ExpenseInput) (*Expense, error) {
	var expense Expense
	if err := c.Post(ctx, "/expenses", input, &expense); err != nil {
		return nil, err
	}
	return &expense, nil
}

// UpdateExpense updates an existing expense.
func (c *Client) UpdateExpense(ctx context.Context, id int64, input *ExpenseInput) (*Expense, error) {
	path := fmt.Sprintf("/expenses/%d", id)
	var expense Expense
	if err := c.Patch(ctx, path, input, &expense); err != nil {
		return nil, err
	}
	return &expense, nil
}

// DeleteExpense deletes an expense.
func (c *Client) DeleteExpense(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/expenses/%d", id)
	return c.Delete(ctx, path)
}

// ListAllExpenses fetches all expenses across all pages.
func (c *Client) ListAllExpenses(ctx context.Context, opts ExpenseListOptions) ([]Expense, error) {
	var all []Expense
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListExpenses(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Expenses...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// UploadExpenseReceipt uploads a receipt file to an expense using multipart/form-data.
func (c *Client) UploadExpenseReceipt(ctx context.Context, expenseID int64, receiptPath string) (*Expense, error) {
	// Open the file
	file, err := os.Open(receiptPath)
	if err != nil {
		return nil, fmt.Errorf("open receipt file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the receipt file
	part, err := writer.CreateFormFile("receipt", filepath.Base(receiptPath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close writer: %w", err)
	}

	// Build request
	reqURL := c.baseURL + fmt.Sprintf("/expenses/%d", expenseID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Get token
	tok, err := c.tokenSource.Token()
	if err != nil {
		return nil, &AuthError{Err: err}
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	req.Header.Set("Harvest-Account-Id", strconv.FormatInt(c.accountID, 10))
	req.Header.Set("User-Agent", fmt.Sprintf("harvest/%s (%s)", c.version, c.contactEmail))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	// Handle errors
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
			Details:    string(bodyBytes),
		}
	}

	// Decode response
	var expense Expense
	if err := json.NewDecoder(resp.Body).Decode(&expense); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &expense, nil
}
