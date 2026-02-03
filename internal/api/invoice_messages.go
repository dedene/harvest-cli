package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// InvoiceMessage represents a message/email sent for an invoice.
type InvoiceMessage struct {
	ID                         int64                     `json:"id"`
	SentBy                     string                    `json:"sent_by"`
	SentByEmail                string                    `json:"sent_by_email"`
	SentFrom                   string                    `json:"sent_from"`
	SentFromEmail              string                    `json:"sent_from_email"`
	Recipients                 []InvoiceMessageRecipient `json:"recipients"`
	Subject                    string                    `json:"subject"`
	Body                       string                    `json:"body"`
	IncludeLinkToClientInvoice bool                      `json:"include_link_to_client_invoice"`
	AttachPDF                  bool                      `json:"attach_pdf"`
	SendMeACopy                bool                      `json:"send_me_a_copy"`
	ThankYou                   bool                      `json:"thank_you"`
	EventType                  string                    `json:"event_type"`
	Reminder                   bool                      `json:"reminder"`
	SendReminderOn             *string                   `json:"send_reminder_on"`
	CreatedAt                  time.Time                 `json:"created_at"`
	UpdatedAt                  time.Time                 `json:"updated_at"`
}

// InvoiceMessageRecipient represents a recipient of an invoice message.
type InvoiceMessageRecipient struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// InvoiceMessagesResponse is the paginated response for invoice messages.
type InvoiceMessagesResponse struct {
	InvoiceMessages []InvoiceMessage `json:"invoice_messages"`
	PerPage         int              `json:"per_page"`
	TotalPages      int              `json:"total_pages"`
	TotalEntries    int              `json:"total_entries"`
	NextPage        *int             `json:"next_page"`
	PreviousPage    *int             `json:"previous_page"`
	Page            int              `json:"page"`
	Links           PaginationLinks  `json:"links"`
}

// InvoiceMessageListOptions filters invoice message list requests.
type InvoiceMessageListOptions struct {
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o InvoiceMessageListOptions) QueryParams() string {
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

// InvoiceMessageInput is used to create an invoice message (send email).
type InvoiceMessageInput struct {
	EventType                  string                    `json:"event_type,omitempty"`
	Recipients                 []InvoiceMessageRecipient `json:"recipients,omitempty"`
	Subject                    string                    `json:"subject,omitempty"`
	Body                       string                    `json:"body,omitempty"`
	IncludeLinkToClientInvoice *bool                     `json:"include_link_to_client_invoice,omitempty"`
	AttachPDF                  *bool                     `json:"attach_pdf,omitempty"`
	SendMeACopy                *bool                     `json:"send_me_a_copy,omitempty"`
	ThankYou                   *bool                     `json:"thank_you,omitempty"`
}

// ListInvoiceMessages returns a paginated list of messages for an invoice.
func (c *Client) ListInvoiceMessages(ctx context.Context, invoiceID int64, opts InvoiceMessageListOptions) (*InvoiceMessagesResponse, error) {
	path := fmt.Sprintf("/invoices/%d/messages%s", invoiceID, opts.QueryParams())
	var resp InvoiceMessagesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateInvoiceMessage sends an invoice via email.
func (c *Client) CreateInvoiceMessage(ctx context.Context, invoiceID int64, input *InvoiceMessageInput) (*InvoiceMessage, error) {
	path := fmt.Sprintf("/invoices/%d/messages", invoiceID)
	var msg InvoiceMessage
	if err := c.Post(ctx, path, input, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// DeleteInvoiceMessage deletes a message from an invoice.
func (c *Client) DeleteInvoiceMessage(ctx context.Context, invoiceID, messageID int64) error {
	path := fmt.Sprintf("/invoices/%d/messages/%d", invoiceID, messageID)
	return c.Delete(ctx, path)
}

// ListAllInvoiceMessages fetches all messages for an invoice across all pages.
func (c *Client) ListAllInvoiceMessages(ctx context.Context, invoiceID int64, opts InvoiceMessageListOptions) ([]InvoiceMessage, error) {
	var all []InvoiceMessage
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListInvoiceMessages(ctx, invoiceID, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.InvoiceMessages...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
