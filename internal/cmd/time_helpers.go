package cmd

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// resolveProjectID resolves a project by ID or name.
func resolveProjectID(ctx context.Context, client *api.Client, input string) (int64, error) {
	// Try as ID first
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		return id, nil
	}

	// Search by name
	projects, err := client.ListAllProjects(ctx, api.ProjectListOptions{IsActive: boolPtr(true)})
	if err != nil {
		return 0, fmt.Errorf("fetch projects: %w", err)
	}

	input = strings.ToLower(input)
	for _, p := range projects {
		if strings.ToLower(p.Name) == input || strings.Contains(strings.ToLower(p.Name), input) {
			return p.ID, nil
		}
		if p.Code != "" && strings.ToLower(p.Code) == input {
			return p.ID, nil
		}
	}

	return 0, fmt.Errorf("project not found: %s", input)
}

// resolveClientID resolves a client by ID or name.
func resolveClientID(ctx context.Context, client *api.Client, input string) (int64, error) {
	// Try as ID first
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		return id, nil
	}

	// Search by name
	clients, err := client.ListAllClients(ctx, api.ClientListOptions{IsActive: boolPtr(true)})
	if err != nil {
		return 0, fmt.Errorf("fetch clients: %w", err)
	}

	input = strings.ToLower(input)
	for _, c := range clients {
		if strings.ToLower(c.Name) == input || strings.Contains(strings.ToLower(c.Name), input) {
			return c.ID, nil
		}
	}

	return 0, fmt.Errorf("client not found: %s", input)
}

// resolveTaskID resolves a task by ID or name within a project.
func resolveTaskID(ctx context.Context, client *api.Client, projectID int64, input string) (int64, error) {
	// Try as ID first
	if id, err := strconv.ParseInt(input, 10, 64); err == nil {
		return id, nil
	}

	// Get project's task assignments
	assignments, err := client.ListAllMyProjectAssignments(ctx)
	if err != nil {
		return 0, fmt.Errorf("fetch assignments: %w", err)
	}

	input = strings.ToLower(input)
	for _, pa := range assignments {
		if pa.Project.ID != projectID {
			continue
		}
		for _, ta := range pa.TaskAssignments {
			if strings.ToLower(ta.Task.Name) == input || strings.Contains(strings.ToLower(ta.Task.Name), input) {
				return ta.Task.ID, nil
			}
		}
	}

	return 0, fmt.Errorf("task not found: %s", input)
}

// fetchProjectsForWizard fetches projects for the TUI picker.
func fetchProjectsForWizard(ctx context.Context, client *api.Client) ([]ui.ProjectItem, error) {
	assignments, err := client.ListAllMyProjectAssignments(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ui.ProjectItem, 0, len(assignments))
	for _, pa := range assignments {
		if !pa.IsActive {
			continue
		}
		items = append(items, ui.ProjectItem{
			ProjectID:   pa.Project.ID,
			ProjectName: pa.Project.Name,
			Code:        pa.Project.Code,
			ClientName:  pa.Client.Name,
		})
	}

	return items, nil
}

// fetchTasksForProject fetches tasks for a project for the TUI picker.
func fetchTasksForProject(ctx context.Context, client *api.Client, projectID int64) ([]ui.TaskItem, error) {
	assignments, err := client.ListAllMyProjectAssignments(ctx)
	if err != nil {
		return nil, err
	}

	for _, pa := range assignments {
		if pa.Project.ID != projectID {
			continue
		}

		items := make([]ui.TaskItem, 0, len(pa.TaskAssignments))
		for _, ta := range pa.TaskAssignments {
			if !ta.IsActive {
				continue
			}
			items = append(items, ui.TaskItem{
				TaskID:   ta.Task.ID,
				TaskName: ta.Task.Name,
				Billable: ta.Billable,
			})
		}
		return items, nil
	}

	return nil, fmt.Errorf("project not found: %d", projectID)
}

// outputTimeEntries writes time entries in the specified format.
func outputTimeEntries(w io.Writer, entries []api.TimeEntry, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, entries)
	case output.ModePlain:
		headers := []string{"ID", "Date", "Project", "Task", "Hours", "ExtRef", "Notes"}
		rows := make([][]string, len(entries))
		for i, e := range entries {
			notes := e.Notes
			if len(notes) > 40 {
				notes = notes[:37] + "..."
			}
			extRef := ""
			if e.ExternalReference != nil && e.ExternalReference.ID != "" {
				extRef = e.ExternalReference.ID
			}
			rows[i] = []string{
				strconv.FormatInt(e.ID, 10),
				e.SpentDate,
				e.Project.Name,
				e.Task.Name,
				fmt.Sprintf("%.2f", e.Hours),
				extRef,
				notes,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Date", "Project", "Task", "Hours", "ExtRef", "Notes")
		for _, e := range entries {
			notes := e.Notes
			if len(notes) > 40 {
				notes = notes[:37] + "..."
			}
			extRef := ""
			if e.ExternalReference != nil && e.ExternalReference.ID != "" {
				extRef = e.ExternalReference.ID
			}
			t.AddRow(
				strconv.FormatInt(e.ID, 10),
				e.SpentDate,
				e.Project.Name,
				e.Task.Name,
				fmt.Sprintf("%.2f", e.Hours),
				extRef,
				notes,
			)
		}
		return t.Render()
	}
}

// outputTimeEntry writes a single time entry in the specified format.
func outputTimeEntry(w io.Writer, entry *api.TimeEntry, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, entry)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%.2f\t%s\n",
			entry.ID, entry.SpentDate, entry.Project.Name, entry.Task.Name, entry.Hours, entry.Notes)
		return nil
	default:
		fmt.Fprintf(w, "ID:      %d\n", entry.ID)
		fmt.Fprintf(w, "Date:    %s\n", entry.SpentDate)
		fmt.Fprintf(w, "Project: %s\n", entry.Project.Name)
		fmt.Fprintf(w, "Task:    %s\n", entry.Task.Name)
		fmt.Fprintf(w, "Hours:   %.2f\n", entry.Hours)
		if entry.Notes != "" {
			fmt.Fprintf(w, "Notes:   %s\n", entry.Notes)
		}
		if entry.IsRunning {
			fmt.Fprintf(w, "Status:  Running\n")
		}
		if entry.ExternalReference != nil {
			fmt.Fprintf(w, "External Ref:\n")
			if entry.ExternalReference.Service != "" {
				fmt.Fprintf(w, "  Service: %s\n", entry.ExternalReference.Service)
			}
			if entry.ExternalReference.ID != "" {
				fmt.Fprintf(w, "  ID:      %s\n", entry.ExternalReference.ID)
			}
			if entry.ExternalReference.GroupID != "" {
				fmt.Fprintf(w, "  Group:   %s\n", entry.ExternalReference.GroupID)
			}
			if entry.ExternalReference.Permalink != "" {
				fmt.Fprintf(w, "  URL:     %s\n", entry.ExternalReference.Permalink)
			}
		}
		return nil
	}
}

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool {
	return &b
}
