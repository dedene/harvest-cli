package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Invoice represents a Harvest invoice.
type Invoice struct {
	ID                 int64              `json:"id"`
	ClientKey          string             `json:"client_key"`
	Number             string             `json:"number"`
	PurchaseOrder      string             `json:"purchase_order"`
	Amount             float64            `json:"amount"`
	DueAmount          float64            `json:"due_amount"`
	Tax                *float64           `json:"tax"`
	TaxAmount          float64            `json:"tax_amount"`
	Tax2               *float64           `json:"tax2"`
	Tax2Amount         float64            `json:"tax2_amount"`
	Discount           *float64           `json:"discount"`
	DiscountAmount     float64            `json:"discount_amount"`
	Subject            string             `json:"subject"`
	Notes              string             `json:"notes"`
	Currency           string             `json:"currency"`
	State              string             `json:"state"`
	PeriodStart        *string            `json:"period_start"`
	PeriodEnd          *string            `json:"period_end"`
	IssueDate          string             `json:"issue_date"`
	DueDate            string             `json:"due_date"`
	PaymentTerm        string             `json:"payment_term"`
	SentAt             *time.Time         `json:"sent_at"`
	PaidAt             *time.Time         `json:"paid_at"`
	PaidDate           *string            `json:"paid_date"`
	ClosedAt           *time.Time         `json:"closed_at"`
	RecurringInvoiceID *int64             `json:"recurring_invoice_id"`
	Client             ClientRef          `json:"client"`
	Estimate           *EstimateRef       `json:"estimate"`
	Retainer           *RetainerRef       `json:"retainer"`
	Creator            *UserRef           `json:"creator"`
	LineItems          []InvoiceLineItem  `json:"line_items"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// EstimateRef is a reference to an estimate.
type EstimateRef struct {
	ID int64 `json:"id"`
}

// RetainerRef is a reference to a retainer.
type RetainerRef struct {
	ID int64 `json:"id"`
}

// InvoiceLineItem represents a line item on an invoice.
type InvoiceLineItem struct {
	ID          int64       `json:"id"`
	Kind        string      `json:"kind"`
	Description string      `json:"description"`
	Quantity    float64     `json:"quantity"`
	UnitPrice   float64     `json:"unit_price"`
	Amount      float64     `json:"amount"`
	Taxed       bool        `json:"taxed"`
	Taxed2      bool        `json:"taxed2"`
	Project     *ProjectRef `json:"project"`
}

// InvoicesResponse is the paginated response for invoices.
type InvoicesResponse struct {
	Invoices     []Invoice       `json:"invoices"`
	PerPage      int             `json:"per_page"`
	TotalPages   int             `json:"total_pages"`
	TotalEntries int             `json:"total_entries"`
	NextPage     *int            `json:"next_page"`
	PreviousPage *int            `json:"previous_page"`
	Page         int             `json:"page"`
	Links        PaginationLinks `json:"links"`
}

// InvoiceListOptions filters invoice list requests.
type InvoiceListOptions struct {
	ClientID     int64
	ProjectID    int64
	UpdatedSince string
	From         string
	To           string
	State        string // draft, open, paid, closed
	Page         int
	PerPage      int
}

// QueryParams converts options to URL query parameters.
func (o InvoiceListOptions) QueryParams() string {
	v := url.Values{}
	if o.ClientID > 0 {
		v.Set("client_id", strconv.FormatInt(o.ClientID, 10))
	}
	if o.ProjectID > 0 {
		v.Set("project_id", strconv.FormatInt(o.ProjectID, 10))
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
	if o.State != "" {
		v.Set("state", o.State)
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

// InvoiceInput is used to create or update an invoice.
type InvoiceInput struct {
	ClientID       int64                   `json:"client_id,omitempty"`
	RetainerID     *int64                  `json:"retainer_id,omitempty"`
	EstimateID     *int64                  `json:"estimate_id,omitempty"`
	Number         *string                 `json:"number,omitempty"`
	PurchaseOrder  *string                 `json:"purchase_order,omitempty"`
	Tax            *float64                `json:"tax,omitempty"`
	Tax2           *float64                `json:"tax2,omitempty"`
	Discount       *float64                `json:"discount,omitempty"`
	Subject        *string                 `json:"subject,omitempty"`
	Notes          *string                 `json:"notes,omitempty"`
	Currency       *string                 `json:"currency,omitempty"`
	IssueDate      *string                 `json:"issue_date,omitempty"`
	DueDate        *string                 `json:"due_date,omitempty"`
	PaymentTerm    *string                 `json:"payment_term,omitempty"`
	LineItems      []InvoiceLineItemInput  `json:"line_items,omitempty"`
	LineItemsImport *InvoiceLineItemsImport `json:"line_items_import,omitempty"`
}

// InvoiceLineItemInput is used to create or update an invoice line item.
type InvoiceLineItemInput struct {
	ID          *int64   `json:"id,omitempty"`
	Kind        string   `json:"kind,omitempty"`
	Description *string  `json:"description,omitempty"`
	Quantity    *float64 `json:"quantity,omitempty"`
	UnitPrice   *float64 `json:"unit_price,omitempty"`
	Taxed       *bool    `json:"taxed,omitempty"`
	Taxed2      *bool    `json:"taxed2,omitempty"`
	ProjectID   *int64   `json:"project_id,omitempty"`
	Destroy     *bool    `json:"_destroy,omitempty"`
}

// InvoiceLineItemsImport is used to import time/expenses to an invoice.
type InvoiceLineItemsImport struct {
	ProjectIDs []int64 `json:"project_ids,omitempty"`
	Time       *InvoiceTimeImport `json:"time,omitempty"`
	Expenses   *InvoiceExpensesImport `json:"expenses,omitempty"`
}

// InvoiceTimeImport specifies how to import time entries.
type InvoiceTimeImport struct {
	SummaryType string `json:"summary_type,omitempty"` // task, project, people, detailed
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
}

// InvoiceExpensesImport specifies how to import expenses.
type InvoiceExpensesImport struct {
	SummaryType string `json:"summary_type,omitempty"` // category, project, people, detailed
	From        string `json:"from,omitempty"`
	To          string `json:"to,omitempty"`
	AttachReceipts bool `json:"attach_receipts,omitempty"`
}

// ListInvoices returns a paginated list of invoices.
func (c *Client) ListInvoices(ctx context.Context, opts InvoiceListOptions) (*InvoicesResponse, error) {
	path := "/invoices" + opts.QueryParams()
	var resp InvoicesResponse
	if err := c.Get(ctx, path, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetInvoice retrieves a single invoice by ID.
func (c *Client) GetInvoice(ctx context.Context, id int64) (*Invoice, error) {
	path := fmt.Sprintf("/invoices/%d", id)
	var invoice Invoice
	if err := c.Get(ctx, path, &invoice); err != nil {
		return nil, err
	}
	return &invoice, nil
}

// CreateInvoice creates a new invoice.
func (c *Client) CreateInvoice(ctx context.Context, input *InvoiceInput) (*Invoice, error) {
	var invoice Invoice
	if err := c.Post(ctx, "/invoices", input, &invoice); err != nil {
		return nil, err
	}
	return &invoice, nil
}

// UpdateInvoice updates an existing invoice.
func (c *Client) UpdateInvoice(ctx context.Context, id int64, input *InvoiceInput) (*Invoice, error) {
	path := fmt.Sprintf("/invoices/%d", id)
	var invoice Invoice
	if err := c.Patch(ctx, path, input, &invoice); err != nil {
		return nil, err
	}
	return &invoice, nil
}

// DeleteInvoice deletes an invoice.
func (c *Client) DeleteInvoice(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/invoices/%d", id)
	return c.Delete(ctx, path)
}

// MarkInvoiceSent marks an open invoice as sent.
func (c *Client) MarkInvoiceSent(ctx context.Context, id int64, eventType string) (*Invoice, error) {
	path := fmt.Sprintf("/invoices/%d/messages", id)
	body := map[string]string{"event_type": eventType}
	var msg InvoiceMessage
	if err := c.Post(ctx, path, body, &msg); err != nil {
		return nil, err
	}
	return c.GetInvoice(ctx, id)
}

// MarkInvoiceDraft marks an invoice as draft (re-open).
func (c *Client) MarkInvoiceDraft(ctx context.Context, id int64) (*Invoice, error) {
	return c.MarkInvoiceSent(ctx, id, "re-open")
}

// MarkInvoiceClosed marks an open invoice as closed.
func (c *Client) MarkInvoiceClosed(ctx context.Context, id int64) (*Invoice, error) {
	return c.MarkInvoiceSent(ctx, id, "close")
}

// MarkInvoiceOpen reopens a closed invoice.
func (c *Client) MarkInvoiceOpen(ctx context.Context, id int64) (*Invoice, error) {
	return c.MarkInvoiceSent(ctx, id, "re-open")
}

// ListAllInvoices fetches all invoices across all pages.
func (c *Client) ListAllInvoices(ctx context.Context, opts InvoiceListOptions) ([]Invoice, error) {
	var all []Invoice
	opts.Page = 1
	if opts.PerPage == 0 {
		opts.PerPage = 100
	}
	for {
		resp, err := c.ListInvoices(ctx, opts)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Invoices...)
		if resp.NextPage == nil {
			break
		}
		opts.Page = *resp.NextPage
	}
	return all, nil
}
