package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// EstimateMessagesResponse is the paginated response for estimate messages.
type EstimateMessagesResponse struct {
	EstimateMessages []EstimateMessage `json:"estimate_messages"`
	PerPage          int               `json:"per_page"`
	TotalPages       int               `json:"total_pages"`
	TotalEntries     int               `json:"total_entries"`
	NextPage         *int              `json:"next_page"`
	PreviousPage     *int              `json:"previous_page"`
	Page             int               `json:"page"`
	Links            PaginationLinks   `json:"links"`
}

// EstimateMessageListOptions filters estimate message list requests.
type EstimateMessageListOptions struct {
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o EstimateMessageListOptions) QueryParams() string {
	v := url.Values{}
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

// ListEstimateMessages returns a paginated list of messages for an estimate.
func (c *Client) ListEstimateMessages(ctx context.Context, estimateID int64, opts EstimateMessageListOptions) (*EstimateMessagesResponse, error) {
	path := fmt.Sprintf("/estimates/%d/messages%s", estimateID, opts.QueryParams())
	var resp EstimateMessagesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateEstimateMessage creates a new estimate message (sends estimate via email).
func (c *Client) CreateEstimateMessage(ctx context.Context, estimateID int64, input *EstimateMessageInput) (*EstimateMessage, error) {
	path := fmt.Sprintf("/estimates/%d/messages", estimateID)
	var msg EstimateMessage
	if err := c.Post(ctx, path, input, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DeleteEstimateMessage deletes an estimate message.
func (c *Client) DeleteEstimateMessage(ctx context.Context, estimateID, messageID int64) error {
	path := fmt.Sprintf("/estimates/%d/messages/%d", estimateID, messageID)
	return c.Delete(ctx, path)
}

// ListAllEstimateMessages fetches all messages for an estimate across all pages.
func (c *Client) ListAllEstimateMessages(ctx context.Context, estimateID int64, opts EstimateMessageListOptions) ([]EstimateMessage, error) {
	var all []EstimateMessage
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListEstimateMessages(ctx, estimateID, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.EstimateMessages...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
