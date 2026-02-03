package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ExpenseCategoriesResponse is the paginated response for expense categories.
type ExpenseCategoriesResponse struct {
	ExpenseCategories []ExpenseCategory `json:"expense_categories"`
	PerPage           int               `json:"per_page"`
	TotalPages        int               `json:"total_pages"`
	TotalEntries      int               `json:"total_entries"`
	NextPage          *int              `json:"next_page"`
	PreviousPage      *int              `json:"previous_page"`
	Page              int               `json:"page"`
	Links             PaginationLinks   `json:"links"`
}

// ExpenseCategoryListOptions filters expense category list requests.
type ExpenseCategoryListOptions struct {
	IsActive     *bool
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o ExpenseCategoryListOptions) QueryParams() string {
	v := url.Values{}
	if o.IsActive != nil {
		v.Set("is_active", strconv.FormatBool(*o.IsActive))
	}
	if o.UpdatedSince != "" {
		v.Set("updated_since", o.UpdatedSince)
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

// ListExpenseCategories returns a paginated list of expense categories.
func (c *Client) ListExpenseCategories(ctx context.Context, opts ExpenseCategoryListOptions) (*ExpenseCategoriesResponse, error) {
	path := "/expense_categories" + opts.QueryParams()
	var resp ExpenseCategoriesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetExpenseCategory retrieves a single expense category by ID.
func (c *Client) GetExpenseCategory(ctx context.Context, id int64) (*ExpenseCategory, error) {
	path := fmt.Sprintf("/expense_categories/%d", id)
	var category ExpenseCategory
	if err := c.Get(ctx, path, &category); err != nil {
		return nil, err
	}
	return &category, nil
}

// ListAllExpenseCategories fetches all expense categories across all pages.
func (c *Client) ListAllExpenseCategories(ctx context.Context, opts ExpenseCategoryListOptions) ([]ExpenseCategory, error) {
	var all []ExpenseCategory
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListExpenseCategories(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.ExpenseCategories...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
