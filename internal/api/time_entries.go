package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// TimeEntriesResponse is the paginated response for time entries.
type TimeEntriesResponse struct {
	TimeEntries  []TimeEntry     `json:"time_entries"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// TimeEntryListOptions filters time entry list requests.
type TimeEntryListOptions struct {
	From                string
	To                  string
	UserID              int64
	ProjectID           int64
	ClientID            int64
	TaskID              int64
	ExternalReferenceID string
	IsBilled            *bool
	IsRunning           *bool
	ApprovalStatus      string // "unsubmitted", "submitted", "approved"
	UpdatedSince        string
	Page                int
	PerPage             int
}

// QueryParams converts options to URL query parameters.
func (o TimeEntryListOptions) QueryParams() string {
	v := url.Values{}
	if o.From != "" {
		v.Set("from", o.From)
	}
	if o.To != "" {
		v.Set("to", o.To)
	}
	if o.UserID > 0 {
		v.Set("user_id", strconv.FormatInt(o.UserID, 10))
	}
	if o.ProjectID > 0 {
		v.Set("project_id", strconv.FormatInt(o.ProjectID, 10))
	}
	if o.ClientID > 0 {
		v.Set("client_id", strconv.FormatInt(o.ClientID, 10))
	}
	if o.TaskID > 0 {
		v.Set("task_id", strconv.FormatInt(o.TaskID, 10))
	}
	if o.ExternalReferenceID != "" {
		v.Set("external_reference_id", o.ExternalReferenceID)
	}
	if o.IsBilled != nil {
		v.Set("is_billed", strconv.FormatBool(*o.IsBilled))
	}
	if o.IsRunning != nil {
		v.Set("is_running", strconv.FormatBool(*o.IsRunning))
	}
	if o.ApprovalStatus != "" {
		v.Set("approval_status", o.ApprovalStatus)
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

// ListTimeEntries returns a paginated list of time entries.
func (c *Client) ListTimeEntries(ctx context.Context, opts TimeEntryListOptions) (*TimeEntriesResponse, error) {
	path := "/time_entries" + opts.QueryParams()
	var resp TimeEntriesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTimeEntry retrieves a single time entry by ID.
func (c *Client) GetTimeEntry(ctx context.Context, id int64) (*TimeEntry, error) {
	path := fmt.Sprintf("/time_entries/%d", id)
	var entry TimeEntry
	if err := c.Get(ctx, path, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// CreateTimeEntry creates a new time entry.
func (c *Client) CreateTimeEntry(ctx context.Context, input *TimeEntryInput) (*TimeEntry, error) {
	var entry TimeEntry
	if err := c.Post(ctx, "/time_entries", input, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// UpdateTimeEntry updates an existing time entry.
func (c *Client) UpdateTimeEntry(ctx context.Context, id int64, input *TimeEntryInput) (*TimeEntry, error) {
	path := fmt.Sprintf("/time_entries/%d", id)
	var entry TimeEntry
	if err := c.Patch(ctx, path, input, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// DeleteTimeEntry deletes a time entry.
func (c *Client) DeleteTimeEntry(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/time_entries/%d", id)
	return c.Delete(ctx, path)
}

// StopTimeEntry stops a running time entry.
func (c *Client) StopTimeEntry(ctx context.Context, id int64) (*TimeEntry, error) {
	path := fmt.Sprintf("/time_entries/%d/stop", id)
	var entry TimeEntry
	if err := c.Patch(ctx, path, nil, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// RestartTimeEntry restarts a stopped time entry.
func (c *Client) RestartTimeEntry(ctx context.Context, id int64) (*TimeEntry, error) {
	path := fmt.Sprintf("/time_entries/%d/restart", id)
	var entry TimeEntry
	if err := c.Patch(ctx, path, nil, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// DeleteTimeEntryExternalReference deletes the external reference from a time entry.
func (c *Client) DeleteTimeEntryExternalReference(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/time_entries/%d/external_reference", id)
	return c.Delete(ctx, path)
}

// GetRunningTimeEntry returns the currently running time entry for the user, if any.
func (c *Client) GetRunningTimeEntry(ctx context.Context) (*TimeEntry, error) {
	isRunning := true
	opts := TimeEntryListOptions{
		IsRunning: &isRunning,
		PerPage:   1,
	}
	resp, err := c.ListTimeEntries(ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(resp.TimeEntries) == 0 {
		return nil, nil
	}
	return &resp.TimeEntries[0], nil
}

// ListAllTimeEntries fetches all time entries across all pages.
func (c *Client) ListAllTimeEntries(ctx context.Context, opts TimeEntryListOptions) ([]TimeEntry, error) {
	var all []TimeEntry
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListTimeEntries(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.TimeEntries...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}

// TimeEntryApprovalRequest is the request body for approval actions.
type TimeEntryApprovalRequest struct {
	TimeEntryIDs []int64 `json:"time_entry_ids"`
}

// SubmitTimeEntriesForApproval submits time entries for manager approval.
func (c *Client) SubmitTimeEntriesForApproval(ctx context.Context, ids []int64) error {
	req := TimeEntryApprovalRequest{TimeEntryIDs: ids}
	return c.Post(ctx, "/time_entries/submit_for_approval", req, nil)
}

// ApproveTimeEntries approves submitted time entries (manager action).
func (c *Client) ApproveTimeEntries(ctx context.Context, ids []int64) error {
	req := TimeEntryApprovalRequest{TimeEntryIDs: ids}
	return c.Post(ctx, "/time_entries/approve", req, nil)
}

// RejectTimeEntries rejects submitted time entries (manager action).
func (c *Client) RejectTimeEntries(ctx context.Context, ids []int64) error {
	req := TimeEntryApprovalRequest{TimeEntryIDs: ids}
	return c.Post(ctx, "/time_entries/reject", req, nil)
}

// UnsubmitTimeEntries returns submitted time entries to draft status.
func (c *Client) UnsubmitTimeEntries(ctx context.Context, ids []int64) error {
	req := TimeEntryApprovalRequest{TimeEntryIDs: ids}
	return c.Post(ctx, "/time_entries/unsubmit", req, nil)
}
