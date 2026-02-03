package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/dateparse"
)

// BulkCmd groups bulk operation subcommands.
type BulkCmd struct {
	Export BulkExportCmd `cmd:"" help:"Export time entries to CSV"`
	Import BulkImportCmd `cmd:"" help:"Import time entries from CSV"`
}

// BulkExportCmd exports time entries to CSV.
type BulkExportCmd struct {
	From    string `help:"Start date (required)" short:"f" required:""`
	To      string `help:"End date (required)" short:"t" required:""`
	Project string `help:"Filter by project ID or name" short:"p"`
	User    string `help:"Filter by user ID or 'me'" short:"u"`
	Output  string `help:"Output file path (default: stdout)" short:"o"`
}

func (c *BulkExportCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.TimeEntryListOptions{}

	// Parse date filters
	fromDate, err := dateparse.Parse(c.From)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}
	opts.From = dateparse.FormatDate(fromDate)

	toDate, err := dateparse.Parse(c.To)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}
	opts.To = dateparse.FormatDate(toDate)

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

	entries, err := client.ListAllTimeEntries(ctx, opts)
	if err != nil {
		return fmt.Errorf("list time entries: %w", err)
	}

	// Determine output writer
	var w io.Writer = os.Stdout
	if c.Output != "" {
		f, err := os.Create(c.Output)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		w = f
	}

	return writeTimeEntriesCSV(w, entries)
}

// writeTimeEntriesCSV writes time entries as CSV.
func writeTimeEntriesCSV(w io.Writer, entries []api.TimeEntry) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Write header
	header := []string{
		"date",
		"project_id",
		"project_name",
		"task_id",
		"task_name",
		"hours",
		"notes",
		"external_ref_id",
	}
	if err := cw.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, e := range entries {
		extRefID := ""
		if e.ExternalReference != nil {
			extRefID = e.ExternalReference.ID
		}

		row := []string{
			e.SpentDate,
			strconv.FormatInt(e.Project.ID, 10),
			e.Project.Name,
			strconv.FormatInt(e.Task.ID, 10),
			e.Task.Name,
			fmt.Sprintf("%.2f", e.Hours),
			e.Notes,
			extRefID,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	return cw.Error()
}

// BulkImportCmd imports time entries from CSV.
type BulkImportCmd struct {
	File   string `arg:"" help:"CSV file path"`
	DryRun bool   `help:"Show what would be created without creating" short:"n"`
}

func (c *BulkImportCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Open and parse CSV
	f, err := os.Open(c.File)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	rows, err := parseImportCSV(f)
	if err != nil {
		return err
	}

	if len(rows) == 0 {
		fmt.Fprintln(os.Stdout, "No entries to import")
		return nil
	}

	// Validate all rows first
	validatedRows, err := validateImportRows(ctx, client, rows)
	if err != nil {
		return err
	}

	// Show summary
	fmt.Fprintf(os.Stdout, "%d entries will be created\n\n", len(validatedRows))

	if c.DryRun {
		fmt.Fprintln(os.Stdout, "Dry run - preview of entries:")
		for i, r := range validatedRows {
			fmt.Fprintf(os.Stdout, "  %d. %s: %s - %s (%.2fh)",
				i+1, r.SpentDate, r.ProjectName, r.TaskName, *r.Input.Hours)
			if r.Input.Notes != nil && *r.Input.Notes != "" {
				fmt.Fprintf(os.Stdout, " - %s", truncateNotes(*r.Input.Notes, 30))
			}
			fmt.Fprintln(os.Stdout)
		}
		return nil
	}

	// Create entries one by one with progress
	created := 0
	for i, r := range validatedRows {
		entry, err := client.CreateTimeEntry(ctx, r.Input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating entry %d: %v\n", i+1, err)
			continue
		}
		created++
		fmt.Fprintf(os.Stdout, "[%d/%d] Created #%d: %s - %s (%.2fh)\n",
			i+1, len(validatedRows), entry.ID, entry.Project.Name, entry.Task.Name, entry.Hours)
	}

	fmt.Fprintf(os.Stdout, "\nImport complete: %d/%d entries created\n", created, len(validatedRows))
	return nil
}

// importRow represents a parsed CSV row.
type importRow struct {
	LineNum int
	Date    string
	Project string
	Task    string
	Hours   string
	Notes   string
}

// validatedRow represents a validated import row ready for creation.
type validatedRow struct {
	SpentDate   string
	ProjectName string
	TaskName    string
	Input       *api.TimeEntryInput
}

// parseImportCSV parses the import CSV file.
func parseImportCSV(r io.Reader) ([]importRow, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	// Read header
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Find column indices
	colMap := make(map[string]int)
	for i, h := range header {
		colMap[strings.ToLower(strings.TrimSpace(h))] = i
	}

	// Required columns
	required := []string{"date", "project", "task", "hours"}
	for _, col := range required {
		if _, ok := colMap[col]; !ok {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}

	var rows []importRow
	lineNum := 1 // header is line 1

	for {
		lineNum++
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}

		row := importRow{
			LineNum: lineNum,
			Date:    getCol(record, colMap, "date"),
			Project: getCol(record, colMap, "project"),
			Task:    getCol(record, colMap, "task"),
			Hours:   getCol(record, colMap, "hours"),
			Notes:   getCol(record, colMap, "notes"),
		}

		rows = append(rows, row)
	}

	return rows, nil
}

// getCol safely gets a column value.
func getCol(record []string, colMap map[string]int, name string) string {
	if idx, ok := colMap[name]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

// validateImportRows validates all rows and resolves IDs.
func validateImportRows(ctx context.Context, client *api.Client, rows []importRow) ([]validatedRow, error) {
	var validated []validatedRow
	var errors []string

	// Cache for resolved IDs
	projectCache := make(map[string]int64)
	projectNames := make(map[int64]string)
	taskCache := make(map[string]int64) // key: "projectID:task"
	taskNames := make(map[int64]string)

	for _, row := range rows {
		err := validateImportRow(row)
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: %v", row.LineNum, err))
			continue
		}

		// Parse date
		date, err := dateparse.Parse(row.Date)
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: invalid date %q", row.LineNum, row.Date))
			continue
		}
		spentDate := dateparse.FormatDate(date)

		// Resolve project
		projectID, ok := projectCache[row.Project]
		if !ok {
			projectID, err = resolveProjectID(ctx, client, row.Project)
			if err != nil {
				errors = append(errors, fmt.Sprintf("line %d: %v", row.LineNum, err))
				continue
			}
			projectCache[row.Project] = projectID
			// Get project name for display
			projects, _ := client.ListAllProjects(ctx, api.ProjectListOptions{})
			for _, p := range projects {
				if p.ID == projectID {
					projectNames[projectID] = p.Name
					break
				}
			}
		}

		// Resolve task
		taskKey := fmt.Sprintf("%d:%s", projectID, row.Task)
		taskID, ok := taskCache[taskKey]
		if !ok {
			taskID, err = resolveTaskID(ctx, client, projectID, row.Task)
			if err != nil {
				errors = append(errors, fmt.Sprintf("line %d: %v", row.LineNum, err))
				continue
			}
			taskCache[taskKey] = taskID
			// Get task name for display
			assignments, _ := client.ListAllMyProjectAssignments(ctx)
			for _, pa := range assignments {
				if pa.Project.ID == projectID {
					for _, ta := range pa.TaskAssignments {
						if ta.Task.ID == taskID {
							taskNames[taskID] = ta.Task.Name
							break
						}
					}
					break
				}
			}
		}

		// Parse hours
		hours, err := strconv.ParseFloat(row.Hours, 64)
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: invalid hours %q", row.LineNum, row.Hours))
			continue
		}

		input := &api.TimeEntryInput{
			ProjectID: projectID,
			TaskID:    taskID,
			SpentDate: spentDate,
			Hours:     &hours,
		}
		if row.Notes != "" {
			input.Notes = &row.Notes
		}

		projectName := projectNames[projectID]
		if projectName == "" {
			projectName = row.Project
		}
		taskName := taskNames[taskID]
		if taskName == "" {
			taskName = row.Task
		}

		validated = append(validated, validatedRow{
			SpentDate:   spentDate,
			ProjectName: projectName,
			TaskName:    taskName,
			Input:       input,
		})
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("validation errors:\n  %s", strings.Join(errors, "\n  "))
	}

	return validated, nil
}

// validateImportRow validates a single import row.
func validateImportRow(row importRow) error {
	if row.Date == "" {
		return fmt.Errorf("date is required")
	}
	if row.Project == "" {
		return fmt.Errorf("project is required")
	}
	if row.Task == "" {
		return fmt.Errorf("task is required")
	}
	if row.Hours == "" {
		return fmt.Errorf("hours is required")
	}
	return nil
}

// truncateNotes truncates notes for display (rune-safe for UTF-8).
func truncateNotes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}
