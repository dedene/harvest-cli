package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/dateparse"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// ApprovalsCmd groups approval workflow subcommands.
type ApprovalsCmd struct {
	List     ApprovalsListCmd     `cmd:"" help:"List time entries pending approval"`
	Submit   ApprovalsSubmitCmd   `cmd:"" help:"Submit time entries for approval"`
	Approve  ApprovalsApproveCmd  `cmd:"" help:"Approve submitted time entries (manager)"`
	Reject   ApprovalsRejectCmd   `cmd:"" help:"Reject submitted time entries (manager)"`
	Unsubmit ApprovalsUnsubmitCmd `cmd:"" help:"Unsubmit entries back to draft"`
}

// ApprovalsListCmd lists time entries pending approval.
type ApprovalsListCmd struct {
	Status string `help:"Filter by status" enum:"submitted,unsubmitted,approved" default:"submitted"`
	User   string `help:"Filter by user ID or 'me'"`
	Week   bool   `help:"Show current week only"`
}

func (c *ApprovalsListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.TimeEntryListOptions{
		ApprovalStatus: c.Status,
	}

	// Handle week filter
	if c.Week {
		from, to := currentWeekRange()
		opts.From = from
		opts.To = to
	}

	// Handle user filter
	if c.User != "" {
		if c.User == "me" {
			me, err := client.GetMe(ctx)
			if err != nil {
				return fmt.Errorf("get current user: %w", err)
			}
			opts.UserID = me.ID
		} else {
			id, err := strconv.ParseInt(c.User, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid user ID: %s", c.User)
			}
			opts.UserID = id
		}
	}

	entries, err := client.ListAllTimeEntries(ctx, opts)
	if err != nil {
		return fmt.Errorf("list time entries: %w", err)
	}

	return outputApprovalsEntries(os.Stdout, entries, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ApprovalsSubmitCmd submits time entries for approval.
type ApprovalsSubmitCmd struct {
	IDs   []int64 `arg:"" optional:"" help:"Time entry IDs to submit"`
	Week  bool    `help:"Submit all unsubmitted entries for current week" short:"w"`
	Force bool    `help:"Skip confirmation" short:"f"`
}

func (c *ApprovalsSubmitCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	ids := c.IDs

	// If --week, fetch unsubmitted entries for current week
	if c.Week {
		from, to := currentWeekRange()
		me, err := client.GetMe(ctx)
		if err != nil {
			return fmt.Errorf("get current user: %w", err)
		}

		entries, err := client.ListAllTimeEntries(ctx, api.TimeEntryListOptions{
			From:           from,
			To:             to,
			UserID:         me.ID,
			ApprovalStatus: "unsubmitted",
		})
		if err != nil {
			return fmt.Errorf("list time entries: %w", err)
		}

		if len(entries) == 0 {
			fmt.Fprintln(os.Stdout, "No unsubmitted entries for current week")
			return nil
		}

		ids = make([]int64, len(entries))
		for i, e := range entries {
			ids[i] = e.ID
		}

		// Show what will be submitted
		fmt.Fprintf(os.Stderr, "Entries to submit (%d):\n", len(entries))
		var totalHours float64
		for _, e := range entries {
			fmt.Fprintf(os.Stderr, "  #%d: %s - %s - %.2fh (%s)\n",
				e.ID, e.Project.Name, e.Task.Name, e.Hours, e.SpentDate)
			totalHours += e.Hours
		}
		fmt.Fprintf(os.Stderr, "Total: %.2fh\n\n", totalHours)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no time entry IDs specified; use --week or provide IDs")
	}

	// Confirm
	if !c.Force {
		msg := fmt.Sprintf("Submit %d time entries for approval?", len(ids))
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

	if err := client.SubmitTimeEntriesForApproval(ctx, ids); err != nil {
		return fmt.Errorf("submit for approval: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, map[string]any{
			"submitted": len(ids),
			"ids":       ids,
		})
	}

	fmt.Fprintf(os.Stdout, "Submitted %d entries for approval\n", len(ids))
	return nil
}

// ApprovalsApproveCmd approves submitted time entries.
type ApprovalsApproveCmd struct {
	IDs   []int64 `arg:"" optional:"" help:"Time entry IDs to approve"`
	Week  bool    `help:"Approve all submitted entries for current week" short:"w"`
	User  string  `help:"Filter by user ID when using --week"`
	Force bool    `help:"Skip confirmation" short:"f"`
}

func (c *ApprovalsApproveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	ids := c.IDs

	// If --week, fetch submitted entries for current week
	if c.Week {
		from, to := currentWeekRange()
		opts := api.TimeEntryListOptions{
			From:           from,
			To:             to,
			ApprovalStatus: "submitted",
		}

		if c.User != "" {
			id, err := strconv.ParseInt(c.User, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid user ID: %s", c.User)
			}
			opts.UserID = id
		}

		entries, err := client.ListAllTimeEntries(ctx, opts)
		if err != nil {
			return fmt.Errorf("list time entries: %w", err)
		}

		if len(entries) == 0 {
			fmt.Fprintln(os.Stdout, "No submitted entries to approve")
			return nil
		}

		ids = make([]int64, len(entries))
		for i, e := range entries {
			ids[i] = e.ID
		}

		// Show what will be approved
		fmt.Fprintf(os.Stderr, "Entries to approve (%d):\n", len(entries))
		var totalHours float64
		for _, e := range entries {
			fmt.Fprintf(os.Stderr, "  #%d: %s - %s - %.2fh (%s) [%s]\n",
				e.ID, e.User.Name, e.Project.Name, e.Hours, e.SpentDate, e.Task.Name)
			totalHours += e.Hours
		}
		fmt.Fprintf(os.Stderr, "Total: %.2fh\n\n", totalHours)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no time entry IDs specified; use --week or provide IDs")
	}

	// Confirm
	if !c.Force {
		msg := fmt.Sprintf("Approve %d time entries?", len(ids))
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

	if err := client.ApproveTimeEntries(ctx, ids); err != nil {
		return fmt.Errorf("approve entries: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, map[string]any{
			"approved": len(ids),
			"ids":      ids,
		})
	}

	fmt.Fprintf(os.Stdout, "Approved %d entries\n", len(ids))
	return nil
}

// ApprovalsRejectCmd rejects submitted time entries.
type ApprovalsRejectCmd struct {
	IDs   []int64 `arg:"" help:"Time entry IDs to reject"`
	Force bool    `help:"Skip confirmation" short:"f"`
}

func (c *ApprovalsRejectCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	if len(c.IDs) == 0 {
		return fmt.Errorf("no time entry IDs specified")
	}

	// Confirm
	if !c.Force {
		msg := fmt.Sprintf("Reject %d time entries?", len(c.IDs))
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

	if err := client.RejectTimeEntries(ctx, c.IDs); err != nil {
		return fmt.Errorf("reject entries: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, map[string]any{
			"rejected": len(c.IDs),
			"ids":      c.IDs,
		})
	}

	fmt.Fprintf(os.Stdout, "Rejected %d entries\n", len(c.IDs))
	return nil
}

// ApprovalsUnsubmitCmd returns submitted entries back to draft.
type ApprovalsUnsubmitCmd struct {
	IDs   []int64 `arg:"" optional:"" help:"Time entry IDs to unsubmit"`
	Week  bool    `help:"Unsubmit all submitted entries for current week" short:"w"`
	Force bool    `help:"Skip confirmation" short:"f"`
}

func (c *ApprovalsUnsubmitCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	ids := c.IDs

	// If --week, fetch submitted entries for current week
	if c.Week {
		from, to := currentWeekRange()
		me, err := client.GetMe(ctx)
		if err != nil {
			return fmt.Errorf("get current user: %w", err)
		}

		entries, err := client.ListAllTimeEntries(ctx, api.TimeEntryListOptions{
			From:           from,
			To:             to,
			UserID:         me.ID,
			ApprovalStatus: "submitted",
		})
		if err != nil {
			return fmt.Errorf("list time entries: %w", err)
		}

		if len(entries) == 0 {
			fmt.Fprintln(os.Stdout, "No submitted entries for current week")
			return nil
		}

		ids = make([]int64, len(entries))
		for i, e := range entries {
			ids[i] = e.ID
		}

		// Show what will be unsubmitted
		fmt.Fprintf(os.Stderr, "Entries to unsubmit (%d):\n", len(entries))
		var totalHours float64
		for _, e := range entries {
			fmt.Fprintf(os.Stderr, "  #%d: %s - %s - %.2fh (%s)\n",
				e.ID, e.Project.Name, e.Task.Name, e.Hours, e.SpentDate)
			totalHours += e.Hours
		}
		fmt.Fprintf(os.Stderr, "Total: %.2fh\n\n", totalHours)
	}

	if len(ids) == 0 {
		return fmt.Errorf("no time entry IDs specified; use --week or provide IDs")
	}

	// Confirm
	if !c.Force {
		msg := fmt.Sprintf("Unsubmit %d time entries?", len(ids))
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

	if err := client.UnsubmitTimeEntries(ctx, ids); err != nil {
		return fmt.Errorf("unsubmit entries: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, map[string]any{
			"unsubmitted": len(ids),
			"ids":         ids,
		})
	}

	fmt.Fprintf(os.Stdout, "Unsubmitted %d entries\n", len(ids))
	return nil
}

// currentWeekRange returns the start and end dates for the current week (Monday-Sunday).
func currentWeekRange() (from, to string) {
	now := time.Now()
	weekday := now.Weekday()

	// Calculate days since Monday (Go: Sunday=0, Monday=1, ..., Saturday=6)
	daysSinceMonday := int(weekday) - 1
	if daysSinceMonday < 0 {
		daysSinceMonday = 6 // Sunday
	}

	monday := now.AddDate(0, 0, -daysSinceMonday)
	sunday := monday.AddDate(0, 0, 6)

	return dateparse.FormatDate(monday), dateparse.FormatDate(sunday)
}

// outputApprovalsEntries writes time entries with approval status.
func outputApprovalsEntries(w io.Writer, entries []api.TimeEntry, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, entries)
	case output.ModePlain:
		headers := []string{"ID", "Date", "User", "Project", "Task", "Hours", "Status"}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			rows[i] = []string{
				strconv.FormatInt(e.ID, 10),
				e.SpentDate,
				e.User.Name,
				e.Project.Name,
				e.Task.Name,
				fmt.Sprintf("%.2f", e.Hours),
				e.ApprovalStatus,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Date", "User", "Project", "Task", "Hours", "Status")
		for _, e := range entries {
			t.AddRow(
				strconv.FormatInt(e.ID, 10),
				e.SpentDate,
				e.User.Name,
				e.Project.Name,
				e.Task.Name,
				fmt.Sprintf("%.2f", e.Hours),
				e.ApprovalStatus,
			)
		}
		return t.Render()
	}
}
