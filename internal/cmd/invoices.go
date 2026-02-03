package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/dateparse"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// InvoicesCmd groups invoice subcommands.
type InvoicesCmd struct {
	List       InvoicesListCmd       `cmd:"" help:"List invoices"`
	Show       InvoicesShowCmd       `cmd:"" help:"Show an invoice"`
	Add        InvoicesAddCmd        `cmd:"" help:"Create an invoice"`
	Edit       InvoicesEditCmd       `cmd:"" help:"Update an invoice"`
	Remove     InvoicesRemoveCmd     `cmd:"" help:"Delete an invoice"`
	Send       InvoicesSendCmd       `cmd:"" help:"Send invoice via email"`
	MarkSent   InvoicesMarkSentCmd   `cmd:"" name:"mark-sent" help:"Mark invoice as sent"`
	MarkClosed InvoicesMarkClosedCmd `cmd:"" name:"mark-closed" help:"Mark invoice as closed"`
	MarkDraft  InvoicesMarkDraftCmd  `cmd:"" name:"mark-draft" help:"Mark invoice as draft"`
	Payments   InvoicePaymentsCmd    `cmd:"" help:"Manage invoice payments"`
}

// InvoicesListCmd lists invoices with filters.
type InvoicesListCmd struct {
	HarvestClient string `help:"Filter by client ID or name" name:"harvest-client" short:"c"`
	Project       string `help:"Filter by project ID or name" short:"p"`
	State         string `help:"Filter by state: draft, open, paid, closed" default:"" enum:",draft,open,paid,closed"`
	UpdatedSince  string `help:"Filter by updated since date"`
	From          string `help:"Filter by issue date from" short:"f"`
	To            string `help:"Filter by issue date to" short:"t"`
}

func (c *InvoicesListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.InvoiceListOptions{
		State: c.State,
	}

	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		opts.ClientID = clientID
	}

	if c.Project != "" {
		projectID, err := resolveProjectID(ctx, client, c.Project)
		if err != nil {
			return err
		}
		opts.ProjectID = projectID
	}

	if c.UpdatedSince != "" {
		t, err := dateparse.Parse(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("invalid updated_since date: %w", err)
		}
		opts.UpdatedSince = t.Format("2006-01-02T15:04:05Z")
	}

	if c.From != "" {
		t, err := dateparse.Parse(c.From)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
		opts.From = dateparse.FormatDate(t)
	}

	if c.To != "" {
		t, err := dateparse.Parse(c.To)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
		opts.To = dateparse.FormatDate(t)
	}

	invoices, err := client.ListAllInvoices(ctx, opts)
	if err != nil {
		return fmt.Errorf("list invoices: %w", err)
	}

	return outputInvoices(os.Stdout, invoices, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// InvoicesShowCmd shows a single invoice.
type InvoicesShowCmd struct {
	ID int64 `arg:"" help:"Invoice ID"`
}

func (c *InvoicesShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	invoice, err := client.GetInvoice(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	return outputInvoice(os.Stdout, invoice, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// InvoicesAddCmd creates a new invoice.
type InvoicesAddCmd struct {
	HarvestClient string  `help:"Client ID or name (required)" name:"harvest-client" short:"c" required:""`
	Number        string  `help:"Invoice number"`
	Subject       string  `help:"Invoice subject"`
	Notes         string  `help:"Invoice notes"`
	IssueDate     string  `help:"Issue date (default: today)"`
	DueDate       string  `help:"Due date"`
	PaymentTerm   string  `help:"Payment term: upon receipt, net 15, net 30, net 45, net 60, custom" default:"" enum:",upon receipt,net 15,net 30,net 45,net 60,custom"`
	Currency      string  `help:"Currency code (e.g., USD, EUR)"`
	Tax           float64 `help:"Tax percentage"`
	Tax2          float64 `help:"Tax2 percentage"`
	Discount      float64 `help:"Discount percentage"`
	PurchaseOrder string  `help:"Purchase order number"`
}

func (c *InvoicesAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	clientID, err := resolveClientID(ctx, client, c.HarvestClient)
	if err != nil {
		return err
	}

	input := &api.InvoiceInput{
		ClientID: clientID,
	}

	if c.Number != "" {
		input.Number = &c.Number
	}
	if c.Subject != "" {
		input.Subject = &c.Subject
	}
	if c.Notes != "" {
		input.Notes = &c.Notes
	}
	if c.IssueDate != "" {
		t, err := dateparse.Parse(c.IssueDate)
		if err != nil {
			return fmt.Errorf("invalid issue_date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.IssueDate = &d
	}
	if c.DueDate != "" {
		t, err := dateparse.Parse(c.DueDate)
		if err != nil {
			return fmt.Errorf("invalid due_date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.DueDate = &d
	}
	if c.PaymentTerm != "" {
		input.PaymentTerm = &c.PaymentTerm
	}
	if c.Currency != "" {
		input.Currency = &c.Currency
	}
	if c.Tax > 0 {
		input.Tax = &c.Tax
	}
	if c.Tax2 > 0 {
		input.Tax2 = &c.Tax2
	}
	if c.Discount > 0 {
		input.Discount = &c.Discount
	}
	if c.PurchaseOrder != "" {
		input.PurchaseOrder = &c.PurchaseOrder
	}

	invoice, err := client.CreateInvoice(ctx, input)
	if err != nil {
		return fmt.Errorf("create invoice: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, invoice)
	}

	fmt.Fprintf(os.Stdout, "Created invoice #%d: %s (%.2f %s)\n",
		invoice.ID, invoice.Number, invoice.Amount, invoice.Currency)
	return nil
}

// InvoicesEditCmd updates an existing invoice.
type InvoicesEditCmd struct {
	ID            int64   `arg:"" help:"Invoice ID"`
	Number        string  `help:"Invoice number"`
	Subject       string  `help:"Invoice subject"`
	Notes         string  `help:"Invoice notes"`
	IssueDate     string  `help:"Issue date"`
	DueDate       string  `help:"Due date"`
	PaymentTerm   string  `help:"Payment term"`
	Currency      string  `help:"Currency code"`
	Tax           float64 `help:"Tax percentage"`
	Tax2          float64 `help:"Tax2 percentage"`
	Discount      float64 `help:"Discount percentage"`
	PurchaseOrder string  `help:"Purchase order number"`
}

func (c *InvoicesEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.InvoiceInput{}
	hasChanges := false

	if c.Number != "" {
		input.Number = &c.Number
		hasChanges = true
	}
	if c.Subject != "" {
		input.Subject = &c.Subject
		hasChanges = true
	}
	if c.Notes != "" {
		input.Notes = &c.Notes
		hasChanges = true
	}
	if c.IssueDate != "" {
		t, err := dateparse.Parse(c.IssueDate)
		if err != nil {
			return fmt.Errorf("invalid issue_date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.IssueDate = &d
		hasChanges = true
	}
	if c.DueDate != "" {
		t, err := dateparse.Parse(c.DueDate)
		if err != nil {
			return fmt.Errorf("invalid due_date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.DueDate = &d
		hasChanges = true
	}
	if c.PaymentTerm != "" {
		input.PaymentTerm = &c.PaymentTerm
		hasChanges = true
	}
	if c.Currency != "" {
		input.Currency = &c.Currency
		hasChanges = true
	}
	if c.Tax > 0 {
		input.Tax = &c.Tax
		hasChanges = true
	}
	if c.Tax2 > 0 {
		input.Tax2 = &c.Tax2
		hasChanges = true
	}
	if c.Discount > 0 {
		input.Discount = &c.Discount
		hasChanges = true
	}
	if c.PurchaseOrder != "" {
		input.PurchaseOrder = &c.PurchaseOrder
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	invoice, err := client.UpdateInvoice(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, invoice)
	}

	fmt.Fprintf(os.Stdout, "Updated invoice #%d: %s\n", invoice.ID, invoice.Number)
	return nil
}

// InvoicesRemoveCmd deletes an invoice.
type InvoicesRemoveCmd struct {
	ID    int64 `arg:"" help:"Invoice ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *InvoicesRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	invoice, err := client.GetInvoice(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete invoice #%d (%s - %.2f %s)?",
			invoice.ID, invoice.Number, invoice.Amount, invoice.Currency)
		confirmed, err := ui.ConfirmPrompt(msg)
		if err != nil {
			if err == ui.ErrCanceled {
				fmt.Fprintln(os.Stderr, "Canceled")
				return nil
			}
			return err
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, "Aborted")
			return nil
		}
	}

	if err := client.DeleteInvoice(ctx, c.ID); err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted invoice #%d\n", c.ID)
	return nil
}

// InvoicesSendCmd sends an invoice via email.
type InvoicesSendCmd struct {
	ID         int64    `arg:"" help:"Invoice ID"`
	Recipients []string `help:"Recipient emails (comma-separated or multiple flags)" short:"r"`
	Subject    string   `help:"Email subject"`
	Body       string   `help:"Email body"`
	AttachPDF  bool     `help:"Attach PDF to email" default:"true"`
	SendCopy   bool     `help:"Send a copy to yourself"`
}

func (c *InvoicesSendCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.InvoiceMessageInput{
		EventType: "send",
	}

	// Parse recipients
	var recipients []api.InvoiceMessageRecipient
	for _, r := range c.Recipients {
		// Handle comma-separated values
		for _, email := range strings.Split(r, ",") {
			email = strings.TrimSpace(email)
			if email != "" {
				recipients = append(recipients, api.InvoiceMessageRecipient{
					Email: email,
				})
			}
		}
	}
	if len(recipients) > 0 {
		input.Recipients = recipients
	}

	if c.Subject != "" {
		input.Subject = c.Subject
	}
	if c.Body != "" {
		input.Body = c.Body
	}
	input.AttachPDF = &c.AttachPDF
	input.SendMeACopy = &c.SendCopy

	msg, err := client.CreateInvoiceMessage(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("send invoice: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	fmt.Fprintf(os.Stdout, "Invoice sent (message #%d)\n", msg.ID)
	return nil
}

// InvoicesMarkSentCmd marks an invoice as sent.
type InvoicesMarkSentCmd struct {
	ID int64 `arg:"" help:"Invoice ID"`
}

func (c *InvoicesMarkSentCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.InvoiceMessageInput{
		EventType: "send",
	}

	_, err = client.CreateInvoiceMessage(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("mark invoice sent: %w", err)
	}

	invoice, err := client.GetInvoice(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, invoice)
	}

	fmt.Fprintf(os.Stdout, "Marked invoice #%d as sent (state: %s)\n", invoice.ID, invoice.State)
	return nil
}

// InvoicesMarkClosedCmd marks an invoice as closed.
type InvoicesMarkClosedCmd struct {
	ID int64 `arg:"" help:"Invoice ID"`
}

func (c *InvoicesMarkClosedCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.InvoiceMessageInput{
		EventType: "close",
	}

	_, err = client.CreateInvoiceMessage(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("mark invoice closed: %w", err)
	}

	invoice, err := client.GetInvoice(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, invoice)
	}

	fmt.Fprintf(os.Stdout, "Marked invoice #%d as closed (state: %s)\n", invoice.ID, invoice.State)
	return nil
}

// InvoicesMarkDraftCmd marks an invoice as draft (re-opens it).
type InvoicesMarkDraftCmd struct {
	ID int64 `arg:"" help:"Invoice ID"`
}

func (c *InvoicesMarkDraftCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.InvoiceMessageInput{
		EventType: "re-open",
	}

	_, err = client.CreateInvoiceMessage(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("mark invoice draft: %w", err)
	}

	invoice, err := client.GetInvoice(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get invoice: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, invoice)
	}

	fmt.Fprintf(os.Stdout, "Marked invoice #%d as draft (state: %s)\n", invoice.ID, invoice.State)
	return nil
}

// InvoicePaymentsCmd manages invoice payments.
type InvoicePaymentsCmd struct {
	List   InvoicePaymentsListCmd   `cmd:"" help:"List payments for an invoice"`
	Add    InvoicePaymentsAddCmd    `cmd:"" help:"Add a payment to an invoice"`
	Remove InvoicePaymentsRemoveCmd `cmd:"" help:"Remove a payment from an invoice"`
}

// InvoicePaymentsListCmd lists payments for an invoice.
type InvoicePaymentsListCmd struct {
	InvoiceID int64 `arg:"" help:"Invoice ID"`
}

func (c *InvoicePaymentsListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	payments, err := client.ListAllInvoicePayments(ctx, c.InvoiceID, api.InvoicePaymentListOptions{})
	if err != nil {
		return fmt.Errorf("list payments: %w", err)
	}

	return outputInvoicePayments(os.Stdout, payments, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// InvoicePaymentsAddCmd adds a payment to an invoice.
type InvoicePaymentsAddCmd struct {
	InvoiceID int64   `arg:"" help:"Invoice ID"`
	Amount    float64 `help:"Payment amount" required:""`
	PaidDate  string  `help:"Payment date (default: today)"`
	Notes     string  `help:"Payment notes"`
}

func (c *InvoicePaymentsAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.InvoicePaymentInput{
		Amount: c.Amount,
		Notes:  c.Notes,
	}

	if c.PaidDate != "" {
		t, err := dateparse.Parse(c.PaidDate)
		if err != nil {
			return fmt.Errorf("invalid paid_date: %w", err)
		}
		input.PaidDate = dateparse.FormatDate(t)
	}

	payment, err := client.CreateInvoicePayment(ctx, c.InvoiceID, input)
	if err != nil {
		return fmt.Errorf("create payment: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, payment)
	}

	fmt.Fprintf(os.Stdout, "Created payment #%d: %.2f on %s\n",
		payment.ID, payment.Amount, payment.PaidDate)
	return nil
}

// InvoicePaymentsRemoveCmd removes a payment from an invoice.
type InvoicePaymentsRemoveCmd struct {
	InvoiceID int64 `arg:"" help:"Invoice ID"`
	PaymentID int64 `arg:"" help:"Payment ID"`
	Force     bool  `help:"Skip confirmation" short:"f"`
}

func (c *InvoicePaymentsRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete payment #%d from invoice #%d?", c.PaymentID, c.InvoiceID)
		confirmed, err := ui.ConfirmPrompt(msg)
		if err != nil {
			if err == ui.ErrCanceled {
				fmt.Fprintln(os.Stderr, "Canceled")
				return nil
			}
			return err
		}
		if !confirmed {
			fmt.Fprintln(os.Stderr, "Aborted")
			return nil
		}
	}

	if err := client.DeleteInvoicePayment(ctx, c.InvoiceID, c.PaymentID); err != nil {
		return fmt.Errorf("delete payment: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted payment #%d\n", c.PaymentID)
	return nil
}

// outputInvoices writes invoices in the specified format.
func outputInvoices(w io.Writer, invoices []api.Invoice, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, invoices)
	case output.ModePlain:
		headers := []string{"ID", "Number", "Client", "Amount", "Due", "State", "IssueDate"}
		rows := make([][]string, len(invoices))
		for i, inv := range invoices {
			rows[i] = []string{
				strconv.FormatInt(inv.ID, 10),
				inv.Number,
				inv.Client.Name,
				fmt.Sprintf("%.2f", inv.Amount),
				fmt.Sprintf("%.2f", inv.DueAmount),
				inv.State,
				inv.IssueDate,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Number", "Client", "Amount", "Due", "State", "Issue Date")
		for _, inv := range invoices {
			t.AddRow(
				strconv.FormatInt(inv.ID, 10),
				inv.Number,
				inv.Client.Name,
				fmt.Sprintf("%.2f %s", inv.Amount, inv.Currency),
				fmt.Sprintf("%.2f", inv.DueAmount),
				inv.State,
				inv.IssueDate,
			)
		}
		return t.Render()
	}
}

// outputInvoice writes a single invoice in the specified format.
func outputInvoice(w io.Writer, inv *api.Invoice, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, inv)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%s\t%.2f\t%.2f\t%s\t%s\n",
			inv.ID, inv.Number, inv.Client.Name, inv.Amount, inv.DueAmount, inv.State, inv.IssueDate)
		return nil
	default:
		fmt.Fprintf(w, "ID:          %d\n", inv.ID)
		fmt.Fprintf(w, "Number:      %s\n", inv.Number)
		fmt.Fprintf(w, "Client:      %s\n", inv.Client.Name)
		fmt.Fprintf(w, "Amount:      %.2f %s\n", inv.Amount, inv.Currency)
		fmt.Fprintf(w, "Due Amount:  %.2f %s\n", inv.DueAmount, inv.Currency)
		fmt.Fprintf(w, "State:       %s\n", inv.State)
		fmt.Fprintf(w, "Issue Date:  %s\n", inv.IssueDate)
		fmt.Fprintf(w, "Due Date:    %s\n", inv.DueDate)
		if inv.Subject != "" {
			fmt.Fprintf(w, "Subject:     %s\n", inv.Subject)
		}
		if inv.Notes != "" {
			fmt.Fprintf(w, "Notes:       %s\n", inv.Notes)
		}
		if inv.PurchaseOrder != "" {
			fmt.Fprintf(w, "PO:          %s\n", inv.PurchaseOrder)
		}
		if len(inv.LineItems) > 0 {
			fmt.Fprintf(w, "\nLine Items:\n")
			for _, li := range inv.LineItems {
				project := ""
				if li.Project != nil {
					project = fmt.Sprintf(" [%s]", li.Project.Name)
				}
				fmt.Fprintf(w, "  - %s%s: %.2f x %.2f = %.2f\n",
					li.Description, project, li.Quantity, li.UnitPrice, li.Amount)
			}
		}
		return nil
	}
}

// outputInvoicePayments writes invoice payments in the specified format.
func outputInvoicePayments(w io.Writer, payments []api.InvoicePayment, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, payments)
	case output.ModePlain:
		headers := []string{"ID", "Amount", "PaidDate", "Notes"}
		rows := make([][]string, len(payments))
		for i, p := range payments {
			rows[i] = []string{
				strconv.FormatInt(p.ID, 10),
				fmt.Sprintf("%.2f", p.Amount),
				p.PaidDate,
				p.Notes,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Amount", "Paid Date", "Notes")
		for _, p := range payments {
			notes := p.Notes
			if len(notes) > 30 {
				notes = notes[:27] + "..."
			}
			t.AddRow(
				strconv.FormatInt(p.ID, 10),
				fmt.Sprintf("%.2f", p.Amount),
				p.PaidDate,
				notes,
			)
		}
		return t.Render()
	}
}
