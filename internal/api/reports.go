package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ReportListOptions contains common options for report requests.
type ReportListOptions struct {
	From    string // Required for most reports (YYYY-MM-DD)
	To      string // Required for most reports (YYYY-MM-DD)
	Page    int
	PerPage int
}

// QueryParams converts options to URL query parameters.
func (o ReportListOptions) QueryParams() string {
	v := url.Values{}
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

// TimeReportsResponse is the paginated response for time reports.
type TimeReportsResponse struct {
	Results      []TimeReportResult `json:"results"`
	PerPage      int                `json:"per_page"`
	TotalPages   int                `json:"total_pages"`
	TotalEntries int                `json:"total_entries"`
	NextPage     *int               `json:"next_page"`
	PreviousPage *int               `json:"previous_page"`
	Page         int                `json:"page"`
	Links        PaginationLinks    `json:"links"`
}

// TimeReportResult represents a single row in a time report.
type TimeReportResult struct {
	ClientID       int64   `json:"client_id,omitempty"`
	ClientName     string  `json:"client_name,omitempty"`
	ProjectID      int64   `json:"project_id,omitempty"`
	ProjectName    string  `json:"project_name,omitempty"`
	TaskID         int64   `json:"task_id,omitempty"`
	TaskName       string  `json:"task_name,omitempty"`
	UserID         int64   `json:"user_id,omitempty"`
	UserName       string  `json:"user_name,omitempty"`
	TotalHours     float64 `json:"total_hours"`
	BillableHours  float64 `json:"billable_hours"`
	Currency       string  `json:"currency,omitempty"`
	BillableAmount float64 `json:"billable_amount"`
	WeeklyCapacity int     `json:"weekly_capacity,omitempty"`
	AvatarURL      string  `json:"avatar_url,omitempty"`
	IsContractor   bool    `json:"is_contractor,omitempty"`
}

// ExpenseReportsResponse is the paginated response for expense reports.
type ExpenseReportsResponse struct {
	Results      []ExpenseReportResult `json:"results"`
	PerPage      int                   `json:"per_page"`
	TotalPages   int                   `json:"total_pages"`
	TotalEntries int                   `json:"total_entries"`
	NextPage     *int                  `json:"next_page"`
	PreviousPage *int                  `json:"previous_page"`
	Page         int                   `json:"page"`
	Links        PaginationLinks       `json:"links"`
}

// ExpenseReportResult represents a single row in an expense report.
type ExpenseReportResult struct {
	ClientID            int64   `json:"client_id,omitempty"`
	ClientName          string  `json:"client_name,omitempty"`
	ProjectID           int64   `json:"project_id,omitempty"`
	ProjectName         string  `json:"project_name,omitempty"`
	ExpenseCategoryID   int64   `json:"expense_category_id,omitempty"`
	ExpenseCategoryName string  `json:"expense_category_name,omitempty"`
	UserID              int64   `json:"user_id,omitempty"`
	UserName            string  `json:"user_name,omitempty"`
	TotalAmount         float64 `json:"total_amount"`
	BillableAmount      float64 `json:"billable_amount"`
	Currency            string  `json:"currency,omitempty"`
	IsContractor        bool    `json:"is_contractor,omitempty"`
}

// UninvoicedReportResponse is the paginated response for uninvoiced reports.
type UninvoicedReportResponse struct {
	Results      []UninvoicedReportResult `json:"results"`
	PerPage      int                      `json:"per_page"`
	TotalPages   int                      `json:"total_pages"`
	TotalEntries int                      `json:"total_entries"`
	NextPage     *int                     `json:"next_page"`
	PreviousPage *int                     `json:"previous_page"`
	Page         int                      `json:"page"`
	Links        PaginationLinks          `json:"links"`
}

// UninvoicedReportResult represents a single row in an uninvoiced report.
type UninvoicedReportResult struct {
	ClientID           int64   `json:"client_id"`
	ClientName         string  `json:"client_name"`
	ProjectID          int64   `json:"project_id"`
	ProjectName        string  `json:"project_name"`
	Currency           string  `json:"currency"`
	TotalHours         float64 `json:"total_hours"`
	UninvoicedHours    float64 `json:"uninvoiced_hours"`
	UninvoicedExpenses float64 `json:"uninvoiced_expenses"`
	UninvoicedAmount   float64 `json:"uninvoiced_amount"`
}

// ProjectBudgetReportResponse is the paginated response for project budget reports.
type ProjectBudgetReportResponse struct {
	Results      []ProjectBudgetReportResult `json:"results"`
	PerPage      int                         `json:"per_page"`
	TotalPages   int                         `json:"total_pages"`
	TotalEntries int                         `json:"total_entries"`
	NextPage     *int                        `json:"next_page"`
	PreviousPage *int                        `json:"previous_page"`
	Page         int                         `json:"page"`
	Links        PaginationLinks             `json:"links"`
}

// ProjectBudgetReportResult represents a single row in a project budget report.
type ProjectBudgetReportResult struct {
	ProjectID       int64    `json:"project_id"`
	ProjectName     string   `json:"project_name"`
	ClientID        int64    `json:"client_id"`
	ClientName      string   `json:"client_name"`
	BudgetIsMonthly bool     `json:"budget_is_monthly"`
	BudgetBy        string   `json:"budget_by"`
	IsActive        bool     `json:"is_active"`
	Budget          *float64 `json:"budget"`
	BudgetSpent     float64  `json:"budget_spent"`
	BudgetRemaining float64  `json:"budget_remaining"`
}

// GetReportsLimiterStatus returns current reports rate limit status.
// Returns (remaining requests, is near limit).
func (c *Client) GetReportsLimiterStatus() (int, bool) {
	if c.reportsLimiter == nil {
		return 100, false
	}
	remaining := c.reportsLimiter.Remaining()
	return remaining, remaining < 20
}

// ListTimeReportsByClients returns time report grouped by clients.
func (c *Client) ListTimeReportsByClients(ctx context.Context, opts ReportListOptions) (*TimeReportsResponse, error) {
	path := "/reports/time/clients" + opts.QueryParams()
	var resp TimeReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListTimeReportsByProjects returns time report grouped by projects.
func (c *Client) ListTimeReportsByProjects(ctx context.Context, opts ReportListOptions) (*TimeReportsResponse, error) {
	path := "/reports/time/projects" + opts.QueryParams()
	var resp TimeReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListTimeReportsByTasks returns time report grouped by tasks.
func (c *Client) ListTimeReportsByTasks(ctx context.Context, opts ReportListOptions) (*TimeReportsResponse, error) {
	path := "/reports/time/tasks" + opts.QueryParams()
	var resp TimeReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListTimeReportsByTeam returns time report grouped by team members.
func (c *Client) ListTimeReportsByTeam(ctx context.Context, opts ReportListOptions) (*TimeReportsResponse, error) {
	path := "/reports/time/team" + opts.QueryParams()
	var resp TimeReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListExpenseReportsByClients returns expense report grouped by clients.
func (c *Client) ListExpenseReportsByClients(ctx context.Context, opts ReportListOptions) (*ExpenseReportsResponse, error) {
	path := "/reports/expenses/clients" + opts.QueryParams()
	var resp ExpenseReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListExpenseReportsByProjects returns expense report grouped by projects.
func (c *Client) ListExpenseReportsByProjects(ctx context.Context, opts ReportListOptions) (*ExpenseReportsResponse, error) {
	path := "/reports/expenses/projects" + opts.QueryParams()
	var resp ExpenseReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListExpenseReportsByCategories returns expense report grouped by categories.
func (c *Client) ListExpenseReportsByCategories(ctx context.Context, opts ReportListOptions) (*ExpenseReportsResponse, error) {
	path := "/reports/expenses/categories" + opts.QueryParams()
	var resp ExpenseReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListExpenseReportsByTeam returns expense report grouped by team members.
func (c *Client) ListExpenseReportsByTeam(ctx context.Context, opts ReportListOptions) (*ExpenseReportsResponse, error) {
	path := "/reports/expenses/team" + opts.QueryParams()
	var resp ExpenseReportsResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListUninvoicedReport returns uninvoiced amounts by project.
func (c *Client) ListUninvoicedReport(ctx context.Context, opts ReportListOptions) (*UninvoicedReportResponse, error) {
	path := "/reports/uninvoiced" + opts.QueryParams()
	var resp UninvoicedReportResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ProjectBudgetReportOptions contains options for project budget reports.
type ProjectBudgetReportOptions struct {
	Page     int
	PerPage  int
	IsActive *bool
}

// QueryParams converts options to URL query parameters.
func (o ProjectBudgetReportOptions) QueryParams() string {
	v := url.Values{}
	if o.Page > 0 {
		v.Set("page", strconv.Itoa(o.Page))
	}
	if o.PerPage > 0 {
		v.Set("per_page", strconv.Itoa(o.PerPage))
	}
	if o.IsActive != nil {
		v.Set("is_active", strconv.FormatBool(*o.IsActive))
	}
	if len(v) == 0 {
		return ""
	}
	return "?" + v.Encode()
}

// ListProjectBudgetReport returns project budget status.
func (c *Client) ListProjectBudgetReport(ctx context.Context, opts ProjectBudgetReportOptions) (*ProjectBudgetReportResponse, error) {
	path := "/reports/project_budget" + opts.QueryParams()
	var resp ProjectBudgetReportResponse
	if err := c.GetReports(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAllTimeReportsByClients fetches all time report results by clients.
func (c *Client) ListAllTimeReportsByClients(ctx context.Context, opts ReportListOptions) ([]TimeReportResult, error) {
	return c.listAllTimeReports(ctx, opts, c.ListTimeReportsByClients)
}

// ListAllTimeReportsByProjects fetches all time report results by projects.
func (c *Client) ListAllTimeReportsByProjects(ctx context.Context, opts ReportListOptions) ([]TimeReportResult, error) {
	return c.listAllTimeReports(ctx, opts, c.ListTimeReportsByProjects)
}

// ListAllTimeReportsByTasks fetches all time report results by tasks.
func (c *Client) ListAllTimeReportsByTasks(ctx context.Context, opts ReportListOptions) ([]TimeReportResult, error) {
	return c.listAllTimeReports(ctx, opts, c.ListTimeReportsByTasks)
}

// ListAllTimeReportsByTeam fetches all time report results by team.
func (c *Client) ListAllTimeReportsByTeam(ctx context.Context, opts ReportListOptions) ([]TimeReportResult, error) {
	return c.listAllTimeReports(ctx, opts, c.ListTimeReportsByTeam)
}

type timeReportFetcher func(context.Context, ReportListOptions) (*TimeReportsResponse, error)

func (c *Client) listAllTimeReports(ctx context.Context, opts ReportListOptions, fetch timeReportFetcher) ([]TimeReportResult, error) {
	var all []TimeReportResult
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := fetch(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// ListAllExpenseReportsByClients fetches all expense report results by clients.
func (c *Client) ListAllExpenseReportsByClients(ctx context.Context, opts ReportListOptions) ([]ExpenseReportResult, error) {
	return c.listAllExpenseReports(ctx, opts, c.ListExpenseReportsByClients)
}

// ListAllExpenseReportsByProjects fetches all expense report results by projects.
func (c *Client) ListAllExpenseReportsByProjects(ctx context.Context, opts ReportListOptions) ([]ExpenseReportResult, error) {
	return c.listAllExpenseReports(ctx, opts, c.ListExpenseReportsByProjects)
}

// ListAllExpenseReportsByCategories fetches all expense report results by categories.
func (c *Client) ListAllExpenseReportsByCategories(ctx context.Context, opts ReportListOptions) ([]ExpenseReportResult, error) {
	return c.listAllExpenseReports(ctx, opts, c.ListExpenseReportsByCategories)
}

// ListAllExpenseReportsByTeam fetches all expense report results by team.
func (c *Client) ListAllExpenseReportsByTeam(ctx context.Context, opts ReportListOptions) ([]ExpenseReportResult, error) {
	return c.listAllExpenseReports(ctx, opts, c.ListExpenseReportsByTeam)
}

type expenseReportFetcher func(context.Context, ReportListOptions) (*ExpenseReportsResponse, error)

func (c *Client) listAllExpenseReports(ctx context.Context, opts ReportListOptions, fetch expenseReportFetcher) ([]ExpenseReportResult, error) {
	var all []ExpenseReportResult
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := fetch(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// ListAllUninvoicedReport fetches all uninvoiced report results.
func (c *Client) ListAllUninvoicedReport(ctx context.Context, opts ReportListOptions) ([]UninvoicedReportResult, error) {
	var all []UninvoicedReportResult
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListUninvoicedReport(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// ListAllProjectBudgetReport fetches all project budget report results.
func (c *Client) ListAllProjectBudgetReport(ctx context.Context, opts ProjectBudgetReportOptions) ([]ProjectBudgetReportResult, error) {
	var all []ProjectBudgetReportResult
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListProjectBudgetReport(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Results...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// WarnIfNearReportsLimit prints a warning if approaching reports rate limit.
func (c *Client) WarnIfNearReportsLimit() string {
	remaining, nearLimit := c.GetReportsLimiterStatus()
	if nearLimit {
		return fmt.Sprintf("Warning: Reports API rate limit approaching (%d/100 requests remaining in 15min window)", remaining)
	}
	return ""
}
