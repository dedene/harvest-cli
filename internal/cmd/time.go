package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/dateparse"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// TimeCmd groups time entry subcommands.
type TimeCmd struct {
	List   TimeListCmd   `cmd:"" help:"List time entries"`
	Show   TimeShowCmd   `cmd:"" help:"Show a time entry"`
	Add    TimeAddCmd    `cmd:"" help:"Create a time entry"`
	Edit   TimeEditCmd   `cmd:"" help:"Update a time entry"`
	Remove TimeRemoveCmd `cmd:"" help:"Delete a time entry"`
	Log    TimeLogCmd    `cmd:"" help:"Quick time entry (wizard if no args)"`
}

// TimeListCmd lists time entries with filters.
type TimeListCmd struct {
	From           string `help:"Start date (YYYY-MM-DD or 'today')" short:"f"`
	To             string `help:"End date" short:"t"`
	User           string `help:"Filter by user ID or 'me'"`
	Project        string `help:"Filter by project ID or name"`
	HarvestClient  string `help:"Filter by client ID or name" name:"harvest-client" short:"c"`
	Task           string `help:"Filter by task ID"`
	Billed         bool   `help:"Only billed entries"`
	Unbilled       bool   `help:"Only unbilled entries"`
	Running        bool   `help:"Only running timers"`
	ApprovalStatus string `help:"Filter by approval status" enum:",unsubmitted,submitted,approved" default:""`
}

func (c *TimeListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.TimeEntryListOptions{
		ApprovalStatus: c.ApprovalStatus,
	}

	// Parse date filters
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

	// Parse user filter
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

	// Parse project filter
	if c.Project != "" {
		projectID, err := resolveProjectID(ctx, client, c.Project)
		if err != nil {
			return err
		}
		opts.ProjectID = projectID
	}

	// Parse client filter
	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		opts.ClientID = clientID
	}

	// Parse task filter
	if c.Task != "" {
		id, err := strconv.ParseInt(c.Task, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", c.Task)
		}
		opts.TaskID = id
	}

	// Handle billed/unbilled filters
	if c.Billed {
		t := true
		opts.IsBilled = &t
	} else if c.Unbilled {
		f := false
		opts.IsBilled = &f
	}

	// Handle running filter
	if c.Running {
		t := true
		opts.IsRunning = &t
	}

	entries, err := client.ListAllTimeEntries(ctx, opts)
	if err != nil {
		return fmt.Errorf("list time entries: %w", err)
	}

	return outputTimeEntries(os.Stdout, entries, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// TimeShowCmd shows a single time entry.
type TimeShowCmd struct {
	ID int64 `arg:"" help:"Time entry ID"`
}

func (c *TimeShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	entry, err := client.GetTimeEntry(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get time entry: %w", err)
	}

	return outputTimeEntry(os.Stdout, entry, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// TimeAddCmd creates a new time entry.
type TimeAddCmd struct {
	Project       string  `help:"Project ID or name" short:"p"`
	Task          string  `help:"Task ID or name"`
	Date          string  `help:"Date (default: today)" short:"d"`
	Hours         float64 `help:"Hours (duration mode)" short:"h"`
	Start         string  `help:"Start time (timestamp mode)"`
	End           string  `help:"End time (timestamp mode)"`
	Notes         string  `help:"Notes" short:"n"`
	Duration      bool    `help:"Use duration mode (hours)"`
	Timestamp     bool    `help:"Use timestamp mode (start/end)"`
	ExtRefID      string  `help:"External reference ID (e.g., JIRA-123)" name:"external-ref-id"`
	ExtRefGroupID string  `help:"External reference group ID" name:"external-ref-group-id"`
	ExtRefURL     string  `help:"External reference URL" name:"external-ref-url"`
	ExtRefService string  `help:"External reference service name (e.g., jira, asana)" name:"external-ref-service"`
}

func (c *TimeAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// If project/task not specified, run wizard
	if c.Project == "" || c.Task == "" {
		return c.runWizard(ctx, client, cli)
	}

	projectID, err := resolveProjectID(ctx, client, c.Project)
	if err != nil {
		return err
	}

	taskID, err := resolveTaskID(ctx, client, projectID, c.Task)
	if err != nil {
		return err
	}

	input := &api.TimeEntryInput{
		ProjectID: projectID,
		TaskID:    taskID,
	}

	// Parse date
	if c.Date != "" {
		t, err := dateparse.Parse(c.Date)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
		input.SpentDate = dateparse.FormatDate(t)
	} else {
		input.SpentDate = dateparse.FormatDate(time.Now())
	}

	// Validate hours
	if c.Hours < 0 {
		return fmt.Errorf("hours cannot be negative")
	}
	if c.Hours > 24 {
		return fmt.Errorf("hours cannot exceed 24")
	}

	// Handle duration vs timestamp mode
	if c.Timestamp || (c.Start != "" || c.End != "") {
		if c.Start != "" {
			input.StartedTime = &c.Start
		}
		if c.End != "" {
			input.EndedTime = &c.End
		}
	} else if c.Hours > 0 {
		input.Hours = &c.Hours
	} else {
		// Default to 0 hours (timer will be started)
		h := 0.0
		input.Hours = &h
	}

	if c.Notes != "" {
		input.Notes = &c.Notes
	}

	// Set external reference if any fields provided
	if c.ExtRefID != "" || c.ExtRefGroupID != "" || c.ExtRefURL != "" || c.ExtRefService != "" {
		input.ExternalReference = &api.ExternalReference{
			ID:        c.ExtRefID,
			GroupID:   c.ExtRefGroupID,
			Permalink: c.ExtRefURL,
			Service:   c.ExtRefService,
		}
	}

	entry, err := client.CreateTimeEntry(ctx, input)
	if err != nil {
		return fmt.Errorf("create time entry: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, entry)
	}

	fmt.Fprintf(os.Stdout, "Created time entry #%d: %s - %s (%.2fh)\n",
		entry.ID, entry.Project.Name, entry.Task.Name, entry.Hours)
	return nil
}

func (c *TimeAddCmd) runWizard(ctx context.Context, client *api.Client, cli *CLI) error {
	projects, err := fetchProjectsForWizard(ctx, client)
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		return fmt.Errorf("no projects available")
	}

	tasksFn := func(projectID int64) ([]ui.TaskItem, error) {
		return fetchTasksForProject(ctx, client, projectID)
	}

	wizard := ui.NewTimeEntryWizard(projects, tasksFn)
	data, err := wizard.Run()
	if err != nil {
		if err == ui.ErrCanceled {
			fmt.Fprintln(os.Stderr, "Canceled")
			return nil
		}
		return err
	}

	entryData, err := ui.ParseTimeEntryData(data)
	if err != nil {
		return fmt.Errorf("parse wizard data: %w", err)
	}

	input := &api.TimeEntryInput{
		ProjectID: entryData.ProjectID,
		TaskID:    entryData.TaskID,
		SpentDate: entryData.SpentDate,
		Hours:     &entryData.Hours,
	}
	if entryData.Notes != "" {
		input.Notes = &entryData.Notes
	}

	entry, err := client.CreateTimeEntry(ctx, input)
	if err != nil {
		return fmt.Errorf("create time entry: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, entry)
	}

	fmt.Fprintf(os.Stdout, "Created time entry #%d: %s - %s (%.2fh)\n",
		entry.ID, entry.Project.Name, entry.Task.Name, entry.Hours)
	return nil
}

// TimeEditCmd updates an existing time entry.
type TimeEditCmd struct {
	ID            int64   `arg:"" help:"Time entry ID"`
	Project       string  `help:"Project ID or name"`
	Task          string  `help:"Task ID or name"`
	Date          string  `help:"Date"`
	Hours         float64 `help:"Hours"`
	Start         string  `help:"Start time"`
	End           string  `help:"End time"`
	Notes         string  `help:"Notes"`
	ExtRefID      string  `help:"External reference ID (e.g., JIRA-123)" name:"external-ref-id"`
	ExtRefGroupID string  `help:"External reference group ID" name:"external-ref-group-id"`
	ExtRefURL     string  `help:"External reference URL" name:"external-ref-url"`
	ExtRefService string  `help:"External reference service name (e.g., jira, asana)" name:"external-ref-service"`
}

func (c *TimeEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.TimeEntryInput{}
	hasChanges := false

	if c.Project != "" {
		projectID, err := resolveProjectID(ctx, client, c.Project)
		if err != nil {
			return err
		}
		input.ProjectID = projectID
		hasChanges = true
	}

	if c.Task != "" {
		// Need project ID to resolve task
		projectID := input.ProjectID
		if projectID == 0 {
			// Get current entry to find project
			entry, err := client.GetTimeEntry(ctx, c.ID)
			if err != nil {
				return fmt.Errorf("get time entry: %w", err)
			}
			projectID = entry.Project.ID
		}
		taskID, err := resolveTaskID(ctx, client, projectID, c.Task)
		if err != nil {
			return err
		}
		input.TaskID = taskID
		hasChanges = true
	}

	if c.Date != "" {
		t, err := dateparse.Parse(c.Date)
		if err != nil {
			return fmt.Errorf("invalid date: %w", err)
		}
		input.SpentDate = dateparse.FormatDate(t)
		hasChanges = true
	}

	if c.Hours != 0 {
		if c.Hours < 0 {
			return fmt.Errorf("hours cannot be negative")
		}
		if c.Hours > 24 {
			return fmt.Errorf("hours cannot exceed 24")
		}
		input.Hours = &c.Hours
		hasChanges = true
	}

	if c.Start != "" {
		input.StartedTime = &c.Start
		hasChanges = true
	}

	if c.End != "" {
		input.EndedTime = &c.End
		hasChanges = true
	}

	if c.Notes != "" {
		input.Notes = &c.Notes
		hasChanges = true
	}

	// Set external reference if any fields provided
	if c.ExtRefID != "" || c.ExtRefGroupID != "" || c.ExtRefURL != "" || c.ExtRefService != "" {
		input.ExternalReference = &api.ExternalReference{
			ID:        c.ExtRefID,
			GroupID:   c.ExtRefGroupID,
			Permalink: c.ExtRefURL,
			Service:   c.ExtRefService,
		}
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	entry, err := client.UpdateTimeEntry(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update time entry: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, entry)
	}

	fmt.Fprintf(os.Stdout, "Updated time entry #%d: %s - %s (%.2fh)\n",
		entry.ID, entry.Project.Name, entry.Task.Name, entry.Hours)
	return nil
}

// TimeRemoveCmd deletes a time entry.
type TimeRemoveCmd struct {
	ID    int64 `arg:"" help:"Time entry ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *TimeRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get entry details for confirmation
	entry, err := client.GetTimeEntry(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get time entry: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete time entry #%d (%s - %s, %.2fh on %s)?",
			entry.ID, entry.Project.Name, entry.Task.Name, entry.Hours, entry.SpentDate)
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

	if err := client.DeleteTimeEntry(ctx, c.ID); err != nil {
		return fmt.Errorf("delete time entry: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted time entry #%d\n", c.ID)
	return nil
}

// TimeLogCmd provides quick time entry with wizard fallback.
type TimeLogCmd struct {
	Notes   string  `arg:"" optional:"" help:"Entry notes"`
	Project string  `help:"Project ID or name" short:"p"`
	Task    string  `help:"Task ID or name"`
	Hours   float64 `help:"Hours" short:"h"`
	Date    string  `help:"Date (default: today)" short:"d"`
}

func (c *TimeLogCmd) Run(cli *CLI) error {
	// Delegate to TimeAddCmd with wizard behavior
	add := &TimeAddCmd{
		Project: c.Project,
		Task:    c.Task,
		Date:    c.Date,
		Hours:   c.Hours,
		Notes:   c.Notes,
	}
	return add.Run(cli)
}
