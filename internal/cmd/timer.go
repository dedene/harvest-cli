package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// TimerCmd groups timer subcommands.
type TimerCmd struct {
	Status  TimerStatusCmd  `cmd:"" default:"1" help:"Show running timer"`
	Start   TimerStartCmd   `cmd:"" help:"Start a timer"`
	Stop    TimerStopCmd    `cmd:"" help:"Stop running timer"`
	Restart TimerRestartCmd `cmd:"" help:"Restart a stopped timer"`
	Toggle  TimerToggleCmd  `cmd:"" help:"Toggle timer (stop if running, start last if not)"`
}

// TimerStatusCmd shows the current running timer.
type TimerStatusCmd struct{}

// Run executes the status command.
func (c *TimerStatusCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	entry, err := client.GetRunningTimeEntry(ctx)
	if err != nil {
		return fmt.Errorf("get running timer: %w", err)
	}

	if entry == nil {
		fmt.Fprintln(os.Stdout, "No timer running")
		return nil
	}

	mode := output.ModeFromFlags(cli.JSON, cli.Plain)
	return formatTimerStatus(os.Stdout, entry, mode)
}

// TimerStartCmd starts a new timer.
type TimerStartCmd struct {
	Project string `help:"Project ID or name" short:"p"`
	Task    string `help:"Task ID or name"`
	Notes   string `help:"Notes" short:"n"`
}

// Run executes the start command.
func (c *TimerStartCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Check if timer is already running
	running, err := client.GetRunningTimeEntry(ctx)
	if err != nil {
		return fmt.Errorf("check running timer: %w", err)
	}
	if running != nil {
		return fmt.Errorf("timer already running: %s - %s (use 'timer stop' first)",
			running.Project.Name, running.Task.Name)
	}

	// Resolve project and task
	projectID, taskID, err := c.resolveProjectTask(ctx, client)
	if err != nil {
		return err
	}

	// Create time entry with no hours to start the timer
	input := &api.TimeEntryInput{
		ProjectID: projectID,
		TaskID:    taskID,
		SpentDate: time.Now().Format("2006-01-02"),
	}
	if c.Notes != "" {
		input.Notes = &c.Notes
	}

	entry, err := client.CreateTimeEntry(ctx, input)
	if err != nil {
		return fmt.Errorf("start timer: %w", err)
	}

	mode := output.ModeFromFlags(cli.JSON, cli.Plain)
	if mode == output.ModeJSON {
		return output.WriteJSON(os.Stdout, entry)
	}

	fmt.Fprintf(os.Stdout, "Started timer: %s - %s\n", entry.Project.Name, entry.Task.Name)
	return nil
}

// resolveProjectTask resolves project and task IDs from flags or TUI picker.
func (c *TimerStartCmd) resolveProjectTask(ctx context.Context, client *api.Client) (int64, int64, error) {
	// Get all project assignments for the user
	assignments, err := client.ListAllMyProjectAssignments(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("list project assignments: %w", err)
	}

	if len(assignments) == 0 {
		return 0, 0, fmt.Errorf("no project assignments found")
	}

	// Resolve project
	var selectedAssignment *api.ProjectAssignment
	if c.Project != "" {
		selectedAssignment = findProjectAssignment(assignments, c.Project)
		if selectedAssignment == nil {
			return 0, 0, fmt.Errorf("project not found: %s", c.Project)
		}
	} else {
		// Use TUI picker
		items := make([]ui.ProjectItem, len(assignments))
		for i, a := range assignments {
			items[i] = ui.ProjectItem{
				ProjectID:   a.Project.ID,
				ProjectName: a.Project.Name,
				ClientName:  a.Client.Name,
				Code:        a.Project.Code,
			}
		}

		selected, err := ui.PickProject("Select project", items)
		if err != nil {
			return 0, 0, err
		}
		if selected == nil {
			return 0, 0, ui.ErrCanceled
		}

		for i := range assignments {
			if assignments[i].Project.ID == selected.ProjectID {
				selectedAssignment = &assignments[i]
				break
			}
		}
	}

	if selectedAssignment == nil {
		return 0, 0, fmt.Errorf("project not found")
	}

	// Get active tasks for this project assignment
	var activeTasks []api.ProjectTaskAssignment
	for _, ta := range selectedAssignment.TaskAssignments {
		if ta.IsActive {
			activeTasks = append(activeTasks, ta)
		}
	}

	if len(activeTasks) == 0 {
		return 0, 0, fmt.Errorf("no active tasks for project %s", selectedAssignment.Project.Name)
	}

	// Resolve task
	var taskID int64
	if c.Task != "" {
		taskID = findTaskID(activeTasks, c.Task)
		if taskID == 0 {
			return 0, 0, fmt.Errorf("task not found: %s", c.Task)
		}
	} else {
		// Use TUI picker
		items := make([]ui.TaskItem, len(activeTasks))
		for i, ta := range activeTasks {
			items[i] = ui.TaskItem{
				TaskID:   ta.Task.ID,
				TaskName: ta.Task.Name,
				Billable: ta.Billable,
			}
		}

		selected, err := ui.PickTask("Select task", items)
		if err != nil {
			return 0, 0, err
		}
		if selected == nil {
			return 0, 0, ui.ErrCanceled
		}
		taskID = selected.TaskID
	}

	return selectedAssignment.Project.ID, taskID, nil
}

// findProjectAssignment finds a project by ID or name.
func findProjectAssignment(assignments []api.ProjectAssignment, search string) *api.ProjectAssignment {
	// Try as ID first
	if id, err := strconv.ParseInt(search, 10, 64); err == nil {
		for i := range assignments {
			if assignments[i].Project.ID == id {
				return &assignments[i]
			}
		}
	}

	// Try case-insensitive name match
	searchLower := strings.ToLower(search)
	for i := range assignments {
		if strings.ToLower(assignments[i].Project.Name) == searchLower {
			return &assignments[i]
		}
	}

	// Try partial match
	for i := range assignments {
		if strings.Contains(strings.ToLower(assignments[i].Project.Name), searchLower) {
			return &assignments[i]
		}
	}

	return nil
}

// findTaskID finds a task by ID or name.
func findTaskID(tasks []api.ProjectTaskAssignment, search string) int64 {
	// Try as ID first
	if id, err := strconv.ParseInt(search, 10, 64); err == nil {
		for _, t := range tasks {
			if t.Task.ID == id {
				return t.Task.ID
			}
		}
	}

	// Try case-insensitive name match
	searchLower := strings.ToLower(search)
	for _, t := range tasks {
		if strings.ToLower(t.Task.Name) == searchLower {
			return t.Task.ID
		}
	}

	// Try partial match
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Task.Name), searchLower) {
			return t.Task.ID
		}
	}

	return 0
}

// TimerStopCmd stops the running timer.
type TimerStopCmd struct{}

// Run executes the stop command.
func (c *TimerStopCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	running, err := client.GetRunningTimeEntry(ctx)
	if err != nil {
		return fmt.Errorf("get running timer: %w", err)
	}

	if running == nil {
		fmt.Fprintln(os.Stdout, "No timer running")
		return nil
	}

	stopped, err := client.StopTimeEntry(ctx, running.ID)
	if err != nil {
		return fmt.Errorf("stop timer: %w", err)
	}

	mode := output.ModeFromFlags(cli.JSON, cli.Plain)
	if mode == output.ModeJSON {
		return output.WriteJSON(os.Stdout, stopped)
	}

	fmt.Fprintf(os.Stdout, "Stopped: %s - %s (%.2fh)\n",
		stopped.Project.Name, stopped.Task.Name, stopped.Hours)
	return nil
}

// TimerRestartCmd restarts a stopped time entry.
type TimerRestartCmd struct {
	ID int64 `arg:"" help:"Time entry ID to restart"`
}

// Run executes the restart command.
func (c *TimerRestartCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Check if timer is already running
	running, err := client.GetRunningTimeEntry(ctx)
	if err != nil {
		return fmt.Errorf("check running timer: %w", err)
	}
	if running != nil {
		return fmt.Errorf("timer already running: %s - %s (use 'timer stop' first)",
			running.Project.Name, running.Task.Name)
	}

	entry, err := client.RestartTimeEntry(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("restart timer: %w", err)
	}

	mode := output.ModeFromFlags(cli.JSON, cli.Plain)
	if mode == output.ModeJSON {
		return output.WriteJSON(os.Stdout, entry)
	}

	fmt.Fprintf(os.Stdout, "Restarted: %s - %s\n", entry.Project.Name, entry.Task.Name)
	return nil
}

// TimerToggleCmd toggles the timer (stop if running, start last if not).
type TimerToggleCmd struct {
	Project string `help:"Project for new timer if starting" short:"p"`
	Task    string `help:"Task for new timer"`
}

// Run executes the toggle command.
func (c *TimerToggleCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	running, err := client.GetRunningTimeEntry(ctx)
	if err != nil {
		return fmt.Errorf("get running timer: %w", err)
	}

	// If running, stop it
	if running != nil {
		stopped, err := client.StopTimeEntry(ctx, running.ID)
		if err != nil {
			return fmt.Errorf("stop timer: %w", err)
		}

		mode := output.ModeFromFlags(cli.JSON, cli.Plain)
		if mode == output.ModeJSON {
			return output.WriteJSON(os.Stdout, stopped)
		}

		fmt.Fprintf(os.Stdout, "Stopped: %s - %s (%.2fh)\n",
			stopped.Project.Name, stopped.Task.Name, stopped.Hours)
		return nil
	}

	// Not running - start a new timer
	// If project/task specified, use them
	if c.Project != "" || c.Task != "" {
		startCmd := &TimerStartCmd{
			Project: c.Project,
			Task:    c.Task,
		}
		return startCmd.Run(cli)
	}

	// Try to restart the most recent entry
	lastEntry, err := getLastTimeEntry(ctx, client)
	if err != nil {
		return fmt.Errorf("get last entry: %w", err)
	}

	if lastEntry == nil {
		// No recent entry, start with picker
		startCmd := &TimerStartCmd{}
		return startCmd.Run(cli)
	}

	// Restart the last entry
	entry, err := client.RestartTimeEntry(ctx, lastEntry.ID)
	if err != nil {
		return fmt.Errorf("restart timer: %w", err)
	}

	mode := output.ModeFromFlags(cli.JSON, cli.Plain)
	if mode == output.ModeJSON {
		return output.WriteJSON(os.Stdout, entry)
	}

	fmt.Fprintf(os.Stdout, "Restarted: %s - %s\n", entry.Project.Name, entry.Task.Name)
	return nil
}

// getLastTimeEntry returns the most recent non-running time entry.
func getLastTimeEntry(ctx context.Context, client *api.Client) (*api.TimeEntry, error) {
	// Get today's entries first
	today := time.Now().Format("2006-01-02")
	resp, err := client.ListTimeEntries(ctx, api.TimeEntryListOptions{
		From:    today,
		To:      today,
		PerPage: 10,
	})
	if err != nil {
		return nil, err
	}

	for _, e := range resp.TimeEntries {
		if !e.IsRunning {
			return &e, nil
		}
	}

	return nil, nil
}

// formatTimerStatus formats a running timer for display.
func formatTimerStatus(w io.Writer, entry *api.TimeEntry, mode output.Mode) error {
	if mode == output.ModeJSON {
		return output.WriteJSON(w, entry)
	}

	if mode == output.ModePlain {
		elapsed := calculateElapsed(entry)
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			entry.ID, entry.Project.Name, entry.Task.Name, elapsed, entry.Notes)
		return nil
	}

	// Table/human format
	elapsed := calculateElapsed(entry)
	startTime := formatStartTime(entry)

	fmt.Fprintf(w, "▶ Running: %s - %s\n", entry.Project.Name, entry.Task.Name)
	fmt.Fprintf(w, "  Started: %s (%s elapsed)\n", startTime, elapsed)
	if entry.Notes != "" {
		fmt.Fprintf(w, "  Notes: %s\n", entry.Notes)
	}

	return nil
}

// calculateElapsed calculates elapsed time from timer start.
func calculateElapsed(entry *api.TimeEntry) string {
	if entry.TimerStartedAt == nil {
		return fmt.Sprintf("%.2fh", entry.Hours)
	}

	duration := time.Since(*entry.TimerStartedAt)
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// formatStartTime formats the timer start time.
func formatStartTime(entry *api.TimeEntry) string {
	if entry.TimerStartedAt == nil {
		return entry.StartedTime
	}
	return entry.TimerStartedAt.Local().Format("3:04 PM")
}
