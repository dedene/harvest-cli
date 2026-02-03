package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// EstimatesResponse is the paginated response for estimates.
type EstimatesResponse struct {
	Estimates    []Estimate      `json:"estimates"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// EstimateListOptions filters estimate list requests.
type EstimateListOptions struct {
	ClientID     int64
	State        string
	UpdatedSince string
	From         string
	To           string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o EstimateListOptions) QueryParams() string {
	v := url.Values{}
	if o.ClientID > 0 {
		v.Set("client_id", strconv.FormatInt(o.ClientID, 10))
	}
	if o.State != "" {
		v.Set("state", o.State)
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

// ListEstimates returns a paginated list of estimates.
func (c *Client) ListEstimates(ctx context.Context, opts EstimateListOptions) (*EstimatesResponse, error) {
	path := "/estimates" + opts.QueryParams()
	var resp EstimatesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetEstimate retrieves a single estimate by ID.
func (c *Client) GetEstimate(ctx context.Context, id int64) (*Estimate, error) {
	path := fmt.Sprintf("/estimates/%d", id)
	var estimate Estimate
	if err := c.Get(ctx, path, &estimate); err != nil {
		return nil, err
	}
	return &estimate, nil
}

// CreateEstimate creates a new estimate.
func (c *Client) CreateEstimate(ctx context.Context, input *EstimateInput) (*Estimate, error) {
	var estimate Estimate
	if err := c.Post(ctx, "/estimates", input, &estimate); err != nil {
		return nil, err
	}
	return &estimate, nil
}

// UpdateEstimate updates an existing estimate.
func (c *Client) UpdateEstimate(ctx context.Context, id int64, input *EstimateInput) (*Estimate, error) {
	path := fmt.Sprintf("/estimates/%d", id)
	var estimate Estimate
	if err := c.Patch(ctx, path, input, &estimate); err != nil {
		return nil, err
	}
	return &estimate, nil
}

// DeleteEstimate deletes an estimate.
func (c *Client) DeleteEstimate(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/estimates/%d", id)
	return c.Delete(ctx, path)
}

// ListAllEstimates fetches all estimates across all pages.
func (c *Client) ListAllEstimates(ctx context.Context, opts EstimateListOptions) ([]Estimate, error) {
	var all []Estimate
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListEstimates(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Estimates...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// MarkEstimateSent marks a draft estimate as sent.
func (c *Client) MarkEstimateSent(ctx context.Context, estimateID int64) (*EstimateMessage, error) {
	eventType := "send"
	input := &EstimateMessageInput{
		EventType: &eventType,
	}
	return c.CreateEstimateMessage(ctx, estimateID, input)
}

// MarkEstimateAccepted marks an open estimate as accepted.
func (c *Client) MarkEstimateAccepted(ctx context.Context, estimateID int64) (*EstimateMessage, error) {
	eventType := "accept"
	input := &EstimateMessageInput{
		EventType: &eventType,
	}
	return c.CreateEstimateMessage(ctx, estimateID, input)
}

// MarkEstimateDeclined marks an open estimate as declined.
func (c *Client) MarkEstimateDeclined(ctx context.Context, estimateID int64) (*EstimateMessage, error) {
	eventType := "decline"
	input := &EstimateMessageInput{
		EventType: &eventType,
	}
	return c.CreateEstimateMessage(ctx, estimateID, input)
}

// MarkEstimateDraft re-opens a closed estimate (converts back to draft).
func (c *Client) MarkEstimateDraft(ctx context.Context, estimateID int64) (*EstimateMessage, error) {
	eventType := "re-open"
	input := &EstimateMessageInput{
		EventType: &eventType,
	}
	return c.CreateEstimateMessage(ctx, estimateID, input)
}
