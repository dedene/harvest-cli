package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/dateparse"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// TasksCmd groups task subcommands.
type TasksCmd struct {
	List   TasksListCmd   `cmd:"" help:"List all tasks"`
	Show   TasksShowCmd   `cmd:"" help:"Show a task"`
	Add    TasksAddCmd    `cmd:"" help:"Create a task"`
	Edit   TasksEditCmd   `cmd:"" help:"Update a task"`
	Remove TasksRemoveCmd `cmd:"" help:"Delete a task"`
}

// TasksListCmd lists tasks with filters.
type TasksListCmd struct {
	Active       *bool  `help:"Filter by active status"`
	UpdatedSince string `help:"Filter by updated since (ISO datetime)"`
}

func (c *TasksListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.TaskListOptions{
		IsActive: c.Active,
	}
	if c.UpdatedSince != "" {
		t, err := dateparse.Parse(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("invalid updated_since date: %w", err)
		}
		opts.UpdatedSince = t.Format("2006-01-02T15:04:05Z")
	}

	tasks, err := client.ListAllTasks(ctx, opts)
	if err != nil {
		return fmt.Errorf("list tasks: %w", err)
	}

	return outputTasks(os.Stdout, tasks, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// TasksShowCmd shows a single task.
type TasksShowCmd struct {
	ID int64 `arg:"" help:"Task ID"`
}

func (c *TasksShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	task, err := client.GetTask(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	return outputTask(os.Stdout, task, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// TasksAddCmd creates a new task.
type TasksAddCmd struct {
	Name              string   `arg:"" help:"Task name"`
	BillableByDefault *bool    `help:"Billable by default"`
	DefaultHourlyRate *float64 `help:"Default hourly rate"`
	IsDefault         *bool    `help:"Add to new projects by default"`
	Active            *bool    `help:"Is active (default: true)"`
}

func (c *TasksAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.TaskInput{
		Name:              c.Name,
		BillableByDefault: c.BillableByDefault,
		DefaultHourlyRate: c.DefaultHourlyRate,
		IsDefault:         c.IsDefault,
		IsActive:          c.Active,
	}

	task, err := client.CreateTask(ctx, input)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, task)
	}

	fmt.Fprintf(os.Stdout, "Created task #%d: %s\n", task.ID, task.Name)
	return nil
}

// TasksEditCmd updates an existing task.
type TasksEditCmd struct {
	ID                int64    `arg:"" help:"Task ID"`
	Name              string   `help:"Task name"`
	BillableByDefault *bool    `help:"Billable by default"`
	DefaultHourlyRate *float64 `help:"Default hourly rate"`
	IsDefault         *bool    `help:"Add to new projects by default"`
	Active            *bool    `help:"Is active"`
}

func (c *TasksEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.TaskInput{}
	hasChanges := false

	if c.Name != "" {
		input.Name = c.Name
		hasChanges = true
	}
	if c.BillableByDefault != nil {
		input.BillableByDefault = c.BillableByDefault
		hasChanges = true
	}
	if c.DefaultHourlyRate != nil {
		input.DefaultHourlyRate = c.DefaultHourlyRate
		hasChanges = true
	}
	if c.IsDefault != nil {
		input.IsDefault = c.IsDefault
		hasChanges = true
	}
	if c.Active != nil {
		input.IsActive = c.Active
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	task, err := client.UpdateTask(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, task)
	}

	fmt.Fprintf(os.Stdout, "Updated task #%d: %s\n", task.ID, task.Name)
	return nil
}

// TasksRemoveCmd deletes a task.
type TasksRemoveCmd struct {
	ID    int64 `arg:"" help:"Task ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *TasksRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get task details for confirmation
	task, err := client.GetTask(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete task #%d (%s)?", task.ID, task.Name)
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

	if err := client.DeleteTask(ctx, c.ID); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted task #%d\n", c.ID)
	return nil
}

// outputTasks writes tasks in the specified format.
func outputTasks(w io.Writer, tasks []api.Task, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, tasks)
	case output.ModePlain:
		headers := []string{"ID", "Name", "Active", "Billable", "Default", "Rate"}
		rows := make([][]string, len(tasks))
		for i, t := range tasks {
			rate := ""
			if t.DefaultHourlyRate > 0 {
				rate = fmt.Sprintf("%.2f", t.DefaultHourlyRate)
			}
			rows[i] = []string{
				strconv.FormatInt(t.ID, 10),
				t.Name,
				strconv.FormatBool(t.IsActive),
				strconv.FormatBool(t.BillableByDefault),
				strconv.FormatBool(t.IsDefault),
				rate,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Name", "Active", "Billable", "Default", "Rate")
		for _, task := range tasks {
			rate := ""
			if task.DefaultHourlyRate > 0 {
				rate = fmt.Sprintf("%.2f", task.DefaultHourlyRate)
			}
			t.AddRow(
				strconv.FormatInt(task.ID, 10),
				task.Name,
				strconv.FormatBool(task.IsActive),
				strconv.FormatBool(task.BillableByDefault),
				strconv.FormatBool(task.IsDefault),
				rate,
			)
		}
		return t.Render()
	}
}

// outputTask writes a single task in the specified format.
func outputTask(w io.Writer, task *api.Task, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, task)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%t\t%t\t%t\t%.2f\n",
			task.ID, task.Name, task.IsActive, task.BillableByDefault, task.IsDefault, task.DefaultHourlyRate)
		return nil
	default:
		fmt.Fprintf(w, "ID:       %d\n", task.ID)
		fmt.Fprintf(w, "Name:     %s\n", task.Name)
		fmt.Fprintf(w, "Active:   %t\n", task.IsActive)
		fmt.Fprintf(w, "Billable: %t\n", task.BillableByDefault)
		fmt.Fprintf(w, "Default:  %t\n", task.IsDefault)
		if task.DefaultHourlyRate > 0 {
			fmt.Fprintf(w, "Rate:     %.2f\n", task.DefaultHourlyRate)
		}
		return nil
	}
}
