package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ProjectsResponse is the paginated response for projects.
type ProjectsResponse struct {
	Projects     []Project       `json:"projects"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// ProjectListOptions filters project list requests.
type ProjectListOptions struct {
	IsActive     *bool
	ClientID     int64
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o ProjectListOptions) QueryParams() string {
	v := url.Values{}
	if o.IsActive != nil {
		v.Set("is_active", strconv.FormatBool(*o.IsActive))
	}
	if o.ClientID > 0 {
		v.Set("client_id", strconv.FormatInt(o.ClientID, 10))
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

// ListProjects returns a paginated list of projects.
func (c *Client) ListProjects(ctx context.Context, opts ProjectListOptions) (*ProjectsResponse, error) {
	path := "/projects" + opts.QueryParams()
	var resp ProjectsResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProject retrieves a single project by ID.
func (c *Client) GetProject(ctx context.Context, id int64) (*Project, error) {
	path := fmt.Sprintf("/projects/%d", id)
	var project Project
	if err := c.Get(ctx, path, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// CreateProject creates a new project.
func (c *Client) CreateProject(ctx context.Context, input *ProjectInput) (*Project, error) {
	var project Project
	if err := c.Post(ctx, "/projects", input, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// UpdateProject updates an existing project.
func (c *Client) UpdateProject(ctx context.Context, id int64, input *ProjectInput) (*Project, error) {
	path := fmt.Sprintf("/projects/%d", id)
	var project Project
	if err := c.Patch(ctx, path, input, &project); err != nil {
		return nil, err
	}
	return &project, nil
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/projects/%d", id)
	return c.Delete(ctx, path)
}

// ListAllProjects fetches all projects across all pages.
func (c *Client) ListAllProjects(ctx context.Context, opts ProjectListOptions) ([]Project, error) {
	var all []Project
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListProjects(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Projects...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
