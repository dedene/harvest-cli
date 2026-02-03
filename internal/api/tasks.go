package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// TasksResponse is the paginated response for tasks.
type TasksResponse struct {
	Tasks        []Task          `json:"tasks"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// TaskListOptions filters task list requests.
type TaskListOptions struct {
	IsActive     *bool
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o TaskListOptions) QueryParams() string {
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

// ListTasks returns a paginated list of tasks.
func (c *Client) ListTasks(ctx context.Context, opts TaskListOptions) (*TasksResponse, error) {
	path := "/tasks" + opts.QueryParams()
	var resp TasksResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTask retrieves a single task by ID.
func (c *Client) GetTask(ctx context.Context, id int64) (*Task, error) {
	path := fmt.Sprintf("/tasks/%d", id)
	var task Task
	if err := c.Get(ctx, path, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, input *TaskInput) (*Task, error) {
	var task Task
	if err := c.Post(ctx, "/tasks", input, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateTask updates an existing task.
func (c *Client) UpdateTask(ctx context.Context, id int64, input *TaskInput) (*Task, error) {
	path := fmt.Sprintf("/tasks/%d", id)
	var task Task
	if err := c.Patch(ctx, path, input, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/tasks/%d", id)
	return c.Delete(ctx, path)
}

// ListAllTasks fetches all tasks across all pages.
func (c *Client) ListAllTasks(ctx context.Context, opts TaskListOptions) ([]Task, error) {
	var all []Task
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListTasks(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Tasks...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
