package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// ClientsResponse is the paginated response for clients.
type ClientsResponse struct {
	Clients      []HarvestClient `json:"clients"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// ClientListOptions filters client list requests.
type ClientListOptions struct {
	IsActive     *bool
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o ClientListOptions) QueryParams() string {
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

// ListClients returns a paginated list of clients.
func (c *Client) ListClients(ctx context.Context, opts ClientListOptions) (*ClientsResponse, error) {
	path := "/clients" + opts.QueryParams()
	var resp ClientsResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetClient retrieves a single client by ID.
func (c *Client) GetClient(ctx context.Context, id int64) (*HarvestClient, error) {
	path := fmt.Sprintf("/clients/%d", id)
	var hc HarvestClient
	if err := c.Get(ctx, path, &hc); err != nil {
		return nil, err
	}
	return &hc, nil
}

// CreateClient creates a new client.
func (c *Client) CreateClient(ctx context.Context, input *ClientInput) (*HarvestClient, error) {
	var hc HarvestClient
	if err := c.Post(ctx, "/clients", input, &hc); err != nil {
		return nil, err
	}
	return &hc, nil
}

// UpdateClient updates an existing client.
func (c *Client) UpdateClient(ctx context.Context, id int64, input *ClientInput) (*HarvestClient, error) {
	path := fmt.Sprintf("/clients/%d", id)
	var hc HarvestClient
	if err := c.Patch(ctx, path, input, &hc); err != nil {
		return nil, err
	}
	return &hc, nil
}

// DeleteClient deletes a client.
func (c *Client) DeleteClient(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/clients/%d", id)
	return c.Delete(ctx, path)
}

// ListAllClients fetches all clients across all pages.
func (c *Client) ListAllClients(ctx context.Context, opts ClientListOptions) ([]HarvestClient, error) {
	var all []HarvestClient
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListClients(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Clients...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
