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

// EstimatesCmd groups estimate subcommands.
type EstimatesCmd struct {
	List         EstimatesListCmd         `cmd:"" help:"List all estimates"`
	Show         EstimatesShowCmd         `cmd:"" help:"Show an estimate"`
	Add          EstimatesAddCmd          `cmd:"" help:"Create an estimate"`
	Edit         EstimatesEditCmd         `cmd:"" help:"Update an estimate"`
	Remove       EstimatesRemoveCmd       `cmd:"" help:"Delete an estimate"`
	Send         EstimatesSendCmd         `cmd:"" help:"Send estimate via email"`
	MarkSent     EstimatesMarkSentCmd     `cmd:"" name:"mark-sent" help:"Mark estimate as sent"`
	MarkAccepted EstimatesMarkAcceptedCmd `cmd:"" name:"mark-accepted" help:"Mark estimate as accepted"`
	MarkDeclined EstimatesMarkDeclinedCmd `cmd:"" name:"mark-declined" help:"Mark estimate as declined"`
	MarkDraft    EstimatesMarkDraftCmd    `cmd:"" name:"mark-draft" help:"Convert estimate back to draft"`
}

// EstimatesListCmd lists estimates with filters.
type EstimatesListCmd struct {
	HarvestClient string `help:"Filter by client ID or name" name:"harvest-client" short:"c"`
	State         string `help:"Filter by state: draft, sent, accepted, declined" enum:",draft,sent,accepted,declined" default:""`
	UpdatedSince  string `help:"Filter by updated since date"`
	From          string `help:"Filter by issue date on or after" short:"f"`
	To            string `help:"Filter by issue date on or before" short:"t"`
}

func (c *EstimatesListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.EstimateListOptions{
		State: c.State,
	}

	// Parse client filter
	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		opts.ClientID = clientID
	}

	// Parse updated_since filter
	if c.UpdatedSince != "" {
		t, err := dateparse.Parse(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("invalid updated_since date: %w", err)
		}
		opts.UpdatedSince = t.Format("2006-01-02T15:04:05Z")
	}

	// Parse from filter
	if c.From != "" {
		t, err := dateparse.Parse(c.From)
		if err != nil {
			return fmt.Errorf("invalid from date: %w", err)
		}
		opts.From = dateparse.FormatDate(t)
	}

	// Parse to filter
	if c.To != "" {
		t, err := dateparse.Parse(c.To)
		if err != nil {
			return fmt.Errorf("invalid to date: %w", err)
		}
		opts.To = dateparse.FormatDate(t)
	}

	estimates, err := client.ListAllEstimates(ctx, opts)
	if err != nil {
		return fmt.Errorf("list estimates: %w", err)
	}

	return outputEstimates(os.Stdout, estimates, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// EstimatesShowCmd shows a single estimate.
type EstimatesShowCmd struct {
	ID int64 `arg:"" help:"Estimate ID"`
}

func (c *EstimatesShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	estimate, err := client.GetEstimate(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get estimate: %w", err)
	}

	return outputEstimate(os.Stdout, estimate, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// EstimatesAddCmd creates a new estimate.
type EstimatesAddCmd struct {
	HarvestClient string  `help:"Client ID or name (required)" name:"harvest-client" short:"c" required:""`
	Subject       string  `help:"Estimate subject" short:"s"`
	Number        string  `help:"Estimate number (auto-generated if not set)"`
	PurchaseOrder string  `help:"Purchase order number"`
	IssueDate     string  `help:"Issue date (default: today)" short:"d"`
	Currency      string  `help:"Currency code"`
	Tax           float64 `help:"Tax percentage"`
	Tax2          float64 `help:"Second tax percentage"`
	Discount      float64 `help:"Discount percentage"`
	Notes         string  `help:"Additional notes" short:"n"`
}

func (c *EstimatesAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	clientID, err := resolveClientID(ctx, client, c.HarvestClient)
	if err != nil {
		return err
	}

	input := &api.EstimateInput{
		ClientID: clientID,
	}

	if c.Subject != "" {
		input.Subject = &c.Subject
	}
	if c.Number != "" {
		input.Number = &c.Number
	}
	if c.PurchaseOrder != "" {
		input.PurchaseOrder = &c.PurchaseOrder
	}
	if c.IssueDate != "" {
		t, err := dateparse.Parse(c.IssueDate)
		if err != nil {
			return fmt.Errorf("invalid issue_date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.IssueDate = &d
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
	if c.Notes != "" {
		input.Notes = &c.Notes
	}

	estimate, err := client.CreateEstimate(ctx, input)
	if err != nil {
		return fmt.Errorf("create estimate: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, estimate)
	}

	fmt.Fprintf(os.Stdout, "Created estimate #%d: %s (%.2f %s)\n",
		estimate.ID, estimate.Subject, estimate.Amount, estimate.Currency)
	return nil
}

// EstimatesEditCmd updates an existing estimate.
type EstimatesEditCmd struct {
	ID            int64   `arg:"" help:"Estimate ID"`
	HarvestClient string  `help:"Client ID or name" name:"harvest-client" short:"c"`
	Subject       string  `help:"Estimate subject" short:"s"`
	Number        string  `help:"Estimate number"`
	PurchaseOrder string  `help:"Purchase order number"`
	IssueDate     string  `help:"Issue date" short:"d"`
	Currency      string  `help:"Currency code"`
	Tax           float64 `help:"Tax percentage"`
	Tax2          float64 `help:"Second tax percentage"`
	Discount      float64 `help:"Discount percentage"`
	Notes         string  `help:"Additional notes" short:"n"`
}

func (c *EstimatesEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.EstimateInput{}
	hasChanges := false

	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		input.ClientID = clientID
		hasChanges = true
	}
	if c.Subject != "" {
		input.Subject = &c.Subject
		hasChanges = true
	}
	if c.Number != "" {
		input.Number = &c.Number
		hasChanges = true
	}
	if c.PurchaseOrder != "" {
		input.PurchaseOrder = &c.PurchaseOrder
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
	if c.Notes != "" {
		input.Notes = &c.Notes
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	estimate, err := client.UpdateEstimate(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update estimate: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, estimate)
	}

	fmt.Fprintf(os.Stdout, "Updated estimate #%d: %s\n", estimate.ID, estimate.Subject)
	return nil
}

// EstimatesRemoveCmd deletes an estimate.
type EstimatesRemoveCmd struct {
	ID    int64 `arg:"" help:"Estimate ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *EstimatesRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get estimate details for confirmation
	estimate, err := client.GetEstimate(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get estimate: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete estimate #%d (%s - %.2f %s)?",
			estimate.ID, estimate.Subject, estimate.Amount, estimate.Currency)
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

	if err := client.DeleteEstimate(ctx, c.ID); err != nil {
		return fmt.Errorf("delete estimate: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted estimate #%d\n", c.ID)
	return nil
}

// EstimatesSendCmd sends an estimate via email.
type EstimatesSendCmd struct {
	ID          int64    `arg:"" help:"Estimate ID"`
	Recipients  []string `help:"Recipient emails (comma-separated or multiple flags)" short:"r" required:""`
	Subject     string   `help:"Email subject" short:"s"`
	Body        string   `help:"Email body" short:"b"`
	SendMeACopy bool     `help:"Send a copy to yourself"`
}

func (c *EstimatesSendCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Parse recipients
	var recipients []api.EstimateMessageRecipient
	for _, r := range c.Recipients {
		// Support comma-separated emails
		for _, email := range strings.Split(r, ",") {
			email = strings.TrimSpace(email)
			if email != "" {
				recipients = append(recipients, api.EstimateMessageRecipient{Email: email})
			}
		}
	}

	if len(recipients) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}

	input := &api.EstimateMessageInput{
		Recipients: recipients,
	}
	if c.Subject != "" {
		input.Subject = &c.Subject
	}
	if c.Body != "" {
		input.Body = &c.Body
	}
	if c.SendMeACopy {
		input.SendMeACopy = &c.SendMeACopy
	}

	msg, err := client.CreateEstimateMessage(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("send estimate: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	fmt.Fprintf(os.Stdout, "Sent estimate #%d to %d recipient(s)\n", c.ID, len(msg.Recipients))
	return nil
}

// EstimatesMarkSentCmd marks an estimate as sent.
type EstimatesMarkSentCmd struct {
	ID int64 `arg:"" help:"Estimate ID"`
}

func (c *EstimatesMarkSentCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	msg, err := client.MarkEstimateSent(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("mark estimate as sent: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	fmt.Fprintf(os.Stdout, "Marked estimate #%d as sent\n", c.ID)
	return nil
}

// EstimatesMarkAcceptedCmd marks an estimate as accepted.
type EstimatesMarkAcceptedCmd struct {
	ID int64 `arg:"" help:"Estimate ID"`
}

func (c *EstimatesMarkAcceptedCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	msg, err := client.MarkEstimateAccepted(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("mark estimate as accepted: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	fmt.Fprintf(os.Stdout, "Marked estimate #%d as accepted\n", c.ID)
	return nil
}

// EstimatesMarkDeclinedCmd marks an estimate as declined.
type EstimatesMarkDeclinedCmd struct {
	ID int64 `arg:"" help:"Estimate ID"`
}

func (c *EstimatesMarkDeclinedCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	msg, err := client.MarkEstimateDeclined(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("mark estimate as declined: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	fmt.Fprintf(os.Stdout, "Marked estimate #%d as declined\n", c.ID)
	return nil
}

// EstimatesMarkDraftCmd converts an estimate back to draft.
type EstimatesMarkDraftCmd struct {
	ID int64 `arg:"" help:"Estimate ID"`
}

func (c *EstimatesMarkDraftCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	msg, err := client.MarkEstimateDraft(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("mark estimate as draft: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, msg)
	}

	fmt.Fprintf(os.Stdout, "Converted estimate #%d back to draft\n", c.ID)
	return nil
}

// outputEstimates writes estimates in the specified format.
func outputEstimates(w io.Writer, estimates []api.Estimate, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, estimates)
	case output.ModePlain:
		headers := []string{"ID", "Number", "Client", "Subject", "Amount", "State", "Issue Date"}
		rows := make([][]string, len(estimates))
		for i, e := range estimates {
			rows[i] = []string{
				strconv.FormatInt(e.ID, 10),
				e.Number,
				e.Client.Name,
				e.Subject,
				fmt.Sprintf("%.2f %s", e.Amount, e.Currency),
				e.State,
				e.IssueDate,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Number", "Client", "Subject", "Amount", "State", "Issue Date")
		for _, e := range estimates {
			t.AddRow(
				strconv.FormatInt(e.ID, 10),
				e.Number,
				e.Client.Name,
				e.Subject,
				fmt.Sprintf("%.2f %s", e.Amount, e.Currency),
				e.State,
				e.IssueDate,
			)
		}
		return t.Render()
	}
}

// outputEstimate writes a single estimate in the specified format.
func outputEstimate(w io.Writer, estimate *api.Estimate, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, estimate)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%.2f\t%s\t%s\n",
			estimate.ID, estimate.Number, estimate.Client.Name, estimate.Subject,
			estimate.Amount, estimate.State, estimate.IssueDate)
		return nil
	default:
		fmt.Fprintf(w, "ID:          %d\n", estimate.ID)
		fmt.Fprintf(w, "Number:      %s\n", estimate.Number)
		fmt.Fprintf(w, "Client:      %s\n", estimate.Client.Name)
		fmt.Fprintf(w, "Subject:     %s\n", estimate.Subject)
		fmt.Fprintf(w, "Amount:      %.2f %s\n", estimate.Amount, estimate.Currency)
		fmt.Fprintf(w, "State:       %s\n", estimate.State)
		if estimate.IssueDate != "" {
			fmt.Fprintf(w, "Issue Date:  %s\n", estimate.IssueDate)
		}
		if estimate.PurchaseOrder != "" {
			fmt.Fprintf(w, "PO Number:   %s\n", estimate.PurchaseOrder)
		}
		if estimate.Tax != nil {
			fmt.Fprintf(w, "Tax:         %.2f%% (%.2f)\n", *estimate.Tax, estimate.TaxAmount)
		}
		if estimate.Tax2 != nil {
			fmt.Fprintf(w, "Tax2:        %.2f%% (%.2f)\n", *estimate.Tax2, estimate.Tax2Amount)
		}
		if estimate.Discount != nil {
			fmt.Fprintf(w, "Discount:    %.2f%% (%.2f)\n", *estimate.Discount, estimate.DiscountAmount)
		}
		if estimate.Notes != "" {
			fmt.Fprintf(w, "Notes:       %s\n", estimate.Notes)
		}
		fmt.Fprintf(w, "Creator:     %s\n", estimate.Creator.Name)

		// Output line items
		if len(estimate.LineItems) > 0 {
			fmt.Fprintf(w, "\nLine Items:\n")
			for _, item := range estimate.LineItems {
				fmt.Fprintf(w, "  - %s: %s (%.0f x %.2f = %.2f)\n",
					item.Kind, item.Description, item.Quantity, item.UnitPrice, item.Amount)
			}
		}

		// Timestamps
		if estimate.SentAt != nil {
			fmt.Fprintf(w, "\nSent:      %s\n", estimate.SentAt.Format("2006-01-02 15:04"))
		}
		if estimate.AcceptedAt != nil {
			fmt.Fprintf(w, "Accepted:  %s\n", estimate.AcceptedAt.Format("2006-01-02 15:04"))
		}
		if estimate.DeclinedAt != nil {
			fmt.Fprintf(w, "Declined:  %s\n", estimate.DeclinedAt.Format("2006-01-02 15:04"))
		}

		return nil
	}
}
