package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// UsersResponse is the paginated response for users.
type UsersResponse struct {
	Users        []User          `json:"users"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// UserListOptions filters user list requests.
type UserListOptions struct {
	IsActive     *bool
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o UserListOptions) QueryParams() string {
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

// GetMe retrieves the currently authenticated user.
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	var user User
	if err := c.Get(ctx, "/users/me", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers returns a paginated list of users.
func (c *Client) ListUsers(ctx context.Context, opts UserListOptions) (*UsersResponse, error) {
	path := "/users" + opts.QueryParams()
	var resp UsersResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetUser retrieves a single user by ID.
func (c *Client) GetUser(ctx context.Context, id int64) (*User, error) {
	path := fmt.Sprintf("/users/%d", id)
	var user User
	if err := c.Get(ctx, path, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user.
func (c *Client) CreateUser(ctx context.Context, input *UserInput) (*User, error) {
	var user User
	if err := c.Post(ctx, "/users", input, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, id int64, input *UserInput) (*User, error) {
	path := fmt.Sprintf("/users/%d", id)
	var user User
	if err := c.Patch(ctx, path, input, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// DeleteUser deletes a user.
func (c *Client) DeleteUser(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/users/%d", id)
	return c.Delete(ctx, path)
}

// ListAllUsers fetches all users across all pages.
func (c *Client) ListAllUsers(ctx context.Context, opts UserListOptions) ([]User, error) {
	var all []User
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListUsers(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Users...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// MyProjectAssignmentsResponse is the paginated response for user's project assignments.
type MyProjectAssignmentsResponse struct {
	ProjectAssignments []ProjectAssignment `json:"project_assignments"`
	PerPage            int                 `json:"per_page"`
	TotalPages         int                 `json:"total_pages"`
	TotalEntries       int                 `json:"total_entries"`
	NextPage           *int                `json:"next_page"`
	PreviousPage       *int                `json:"previous_page"`
	Page               int                 `json:"page"`
	Links              PaginationLinks     `json:"links"`
}

// MyProjectAssignmentsOptions filters my project assignments requests.
type MyProjectAssignmentsOptions struct {
	Page    int
	PerPage int
}

// QueryParams converts options to URL query parameters.
func (o MyProjectAssignmentsOptions) QueryParams() string {
	v := url.Values{}
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

// ListMyProjectAssignments returns project assignments for the current user.
func (c *Client) ListMyProjectAssignments(ctx context.Context, opts MyProjectAssignmentsOptions) (*MyProjectAssignmentsResponse, error) {
	path := "/users/me/project_assignments" + opts.QueryParams()
	var resp MyProjectAssignmentsResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListAllMyProjectAssignments fetches all project assignments for the current user.
func (c *Client) ListAllMyProjectAssignments(ctx context.Context) ([]ProjectAssignment, error) {
	var all []ProjectAssignment
	opts := MyProjectAssignmentsOptions{Page: 1, PerPage: 100}
	for {
		resp, err := c.ListMyProjectAssignments(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.ProjectAssignments...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
