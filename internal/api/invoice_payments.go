package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// InvoicePayment represents a payment on an invoice.
type InvoicePayment struct {
	ID              int64              `json:"id"`
	Amount          float64            `json:"amount"`
	PaidAt          time.Time          `json:"paid_at"`
	PaidDate        string             `json:"paid_date"`
	RecordedBy      string             `json:"recorded_by"`
	RecordedByEmail string             `json:"recorded_by_email"`
	Notes           string             `json:"notes"`
	TransactionID   string             `json:"transaction_id"`
	PaymentGateway  *PaymentGatewayRef `json:"payment_gateway"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

// PaymentGatewayRef is a reference to a payment gateway.
type PaymentGatewayRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// InvoicePaymentsResponse is the paginated response for invoice payments.
type InvoicePaymentsResponse struct {
	InvoicePayments []InvoicePayment `json:"invoice_payments"`
	PerPage         int              `json:"per_page"`
	TotalPages      int              `json:"total_pages"`
	TotalEntries    int              `json:"total_entries"`
	NextPage        *int             `json:"next_page"`
	PreviousPage    *int             `json:"previous_page"`
	Page            int              `json:"page"`
	Links           PaginationLinks  `json:"links"`
}

// InvoicePaymentListOptions filters invoice payment list requests.
type InvoicePaymentListOptions struct {
	UpdatedSince string
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o InvoicePaymentListOptions) QueryParams() string {
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

// InvoicePaymentInput is used to create an invoice payment.
type InvoicePaymentInput struct {
	Amount   float64 `json:"amount"`
	PaidAt   string  `json:"paid_at,omitempty"`
	PaidDate string  `json:"paid_date,omitempty"`
	Notes    string  `json:"notes,omitempty"`
}

// ListInvoicePayments returns a paginated list of payments for an invoice.
func (c *Client) ListInvoicePayments(ctx context.Context, invoiceID int64, opts InvoicePaymentListOptions) (*InvoicePaymentsResponse, error) {
	path := fmt.Sprintf("/invoices/%d/payments%s", invoiceID, opts.QueryParams())
	var resp InvoicePaymentsResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateInvoicePayment creates a new payment for an invoice.
func (c *Client) CreateInvoicePayment(ctx context.Context, invoiceID int64, input *InvoicePaymentInput) (*InvoicePayment, error) {
	path := fmt.Sprintf("/invoices/%d/payments", invoiceID)
	var payment InvoicePayment
	if err := c.Post(ctx, path, input, &payment); err != nil {
		return nil, err
	}
	return &payment, nil
}

// DeleteInvoicePayment deletes a payment from an invoice.
func (c *Client) DeleteInvoicePayment(ctx context.Context, invoiceID, paymentID int64) error {
	path := fmt.Sprintf("/invoices/%d/payments/%d", invoiceID, paymentID)
	return c.Delete(ctx, path)
}

// ListAllInvoicePayments fetches all payments for an invoice across all pages.
func (c *Client) ListAllInvoicePayments(ctx context.Context, invoiceID int64, opts InvoicePaymentListOptions) ([]InvoicePayment, error) {
	var all []InvoicePayment
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListInvoicePayments(ctx, invoiceID, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.InvoicePayments...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
