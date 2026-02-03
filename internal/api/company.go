package api

import "context"

// GetCompany retrieves the company for the currently authenticated user.
func (c *Client) GetCompany(ctx context.Context) (*Company, error) {
	var company Company
	if err := c.Get(ctx, "/company", &company); err != nil {
		return nil, err
	}
	return &company, nil
}

// CompanyUpdateInput is used to update company settings.
type CompanyUpdateInput struct {
	WantsTimestampTimers *bool `json:"wants_timestamp_timers,omitempty"`
	WeeklyCapacity       *int  `json:"weekly_capacity,omitempty"`
}

// UpdateCompany updates the company settings.
func (c *Client) UpdateCompany(ctx context.Context, input *CompanyUpdateInput) (*Company, error) {
	var company Company
	if err := c.Patch(ctx, "/company", input, &company); err != nil {
		return nil, err
	}
	return &company, nil
}
