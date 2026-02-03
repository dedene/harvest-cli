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

// ProjectsCmd groups project subcommands.
type ProjectsCmd struct {
	List   ProjectsListCmd   `cmd:"" help:"List all projects"`
	Show   ProjectsShowCmd   `cmd:"" help:"Show a project"`
	Add    ProjectsAddCmd    `cmd:"" help:"Create a project"`
	Edit   ProjectsEditCmd   `cmd:"" help:"Update a project"`
	Remove ProjectsRemoveCmd `cmd:"" help:"Delete a project"`
}

// ProjectsListCmd lists projects with filters.
type ProjectsListCmd struct {
	Active        string `help:"Filter by active status: true, false, all" default:"all" enum:"true,false,all"`
	HarvestClient string `help:"Filter by client ID or name" name:"harvest-client" short:"c"`
	UpdatedSince  string `help:"Filter by updated since date"`
}

func (c *ProjectsListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.ProjectListOptions{}

	// Parse active filter
	switch c.Active {
	case "true":
		opts.IsActive = boolPtr(true)
	case "false":
		opts.IsActive = boolPtr(false)
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

	projects, err := client.ListAllProjects(ctx, opts)
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	return outputProjects(os.Stdout, projects, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ProjectsShowCmd shows a single project.
type ProjectsShowCmd struct {
	ID int64 `arg:"" help:"Project ID"`
}

func (c *ProjectsShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	project, err := client.GetProject(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	return outputProject(os.Stdout, project, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ProjectsAddCmd creates a new project.
type ProjectsAddCmd struct {
	Name          string  `help:"Project name" short:"n" required:""`
	HarvestClient string  `help:"Client ID or name" name:"harvest-client" short:"c"`
	Code          string  `help:"Project code"`
	Billable      bool    `help:"Project is billable" default:"true"`
	BillBy        string  `help:"Bill by: none, People, Project, Tasks" default:"none" enum:"none,People,Project,Tasks"`
	BudgetBy      string  `help:"Budget by: none, person, project, task" default:"none" enum:"none,person,project,task"`
	HourlyRate    float64 `help:"Hourly rate"`
	Budget        float64 `help:"Budget amount"`
	Notes         string  `help:"Project notes"`
	StartsOn      string  `help:"Start date"`
	EndsOn        string  `help:"End date"`
	FixedFee      bool    `help:"Fixed fee project"`
}

func (c *ProjectsAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.ProjectInput{
		Name:   c.Name,
		BillBy: c.BillBy,
	}

	// Resolve client
	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		input.ClientID = clientID
	}

	if c.Code != "" {
		input.Code = &c.Code
	}

	input.IsBillable = &c.Billable

	if c.FixedFee {
		input.IsFixedFee = &c.FixedFee
	}

	if c.BudgetBy != "" {
		input.BudgetBy = c.BudgetBy
	}

	if c.HourlyRate > 0 {
		input.HourlyRate = &c.HourlyRate
	}

	if c.Budget > 0 {
		input.Budget = &c.Budget
	}

	if c.Notes != "" {
		input.Notes = &c.Notes
	}

	if c.StartsOn != "" {
		t, err := dateparse.Parse(c.StartsOn)
		if err != nil {
			return fmt.Errorf("invalid starts_on date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.StartsOn = &d
	}

	if c.EndsOn != "" {
		t, err := dateparse.Parse(c.EndsOn)
		if err != nil {
			return fmt.Errorf("invalid ends_on date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.EndsOn = &d
	}

	project, err := client.CreateProject(ctx, input)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, project)
	}

	fmt.Fprintf(os.Stdout, "Created project #%d: %s\n", project.ID, project.Name)
	return nil
}

// ProjectsEditCmd updates an existing project.
type ProjectsEditCmd struct {
	ID            int64   `arg:"" help:"Project ID"`
	Name          string  `help:"Project name" short:"n"`
	HarvestClient string  `help:"Client ID or name" name:"harvest-client" short:"c"`
	Code          string  `help:"Project code"`
	Active        string  `help:"Is active: true, false" default:"" enum:",true,false"`
	Billable      string  `help:"Is billable: true, false" default:"" enum:",true,false"`
	BillBy        string  `help:"Bill by: none, People, Project, Tasks" default:"" enum:",none,People,Project,Tasks"`
	BudgetBy      string  `help:"Budget by: none, person, project, task" default:"" enum:",none,person,project,task"`
	HourlyRate    float64 `help:"Hourly rate"`
	Budget        float64 `help:"Budget amount"`
	Notes         string  `help:"Project notes"`
	StartsOn      string  `help:"Start date"`
	EndsOn        string  `help:"End date"`
	FixedFee      string  `help:"Fixed fee: true, false" default:"" enum:",true,false"`
}

func (c *ProjectsEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.ProjectInput{}
	hasChanges := false

	if c.Name != "" {
		input.Name = c.Name
		hasChanges = true
	}

	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		input.ClientID = clientID
		hasChanges = true
	}

	if c.Code != "" {
		input.Code = &c.Code
		hasChanges = true
	}

	if c.Active != "" {
		active := c.Active == "true"
		input.IsActive = &active
		hasChanges = true
	}

	if c.Billable != "" {
		billable := c.Billable == "true"
		input.IsBillable = &billable
		hasChanges = true
	}

	if c.BillBy != "" {
		input.BillBy = c.BillBy
		hasChanges = true
	}

	if c.BudgetBy != "" {
		input.BudgetBy = c.BudgetBy
		hasChanges = true
	}

	if c.HourlyRate > 0 {
		input.HourlyRate = &c.HourlyRate
		hasChanges = true
	}

	if c.Budget > 0 {
		input.Budget = &c.Budget
		hasChanges = true
	}

	if c.Notes != "" {
		input.Notes = &c.Notes
		hasChanges = true
	}

	if c.StartsOn != "" {
		t, err := dateparse.Parse(c.StartsOn)
		if err != nil {
			return fmt.Errorf("invalid starts_on date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.StartsOn = &d
		hasChanges = true
	}

	if c.EndsOn != "" {
		t, err := dateparse.Parse(c.EndsOn)
		if err != nil {
			return fmt.Errorf("invalid ends_on date: %w", err)
		}
		d := dateparse.FormatDate(t)
		input.EndsOn = &d
		hasChanges = true
	}

	if c.FixedFee != "" {
		fixedFee := c.FixedFee == "true"
		input.IsFixedFee = &fixedFee
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	project, err := client.UpdateProject(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, project)
	}

	fmt.Fprintf(os.Stdout, "Updated project #%d: %s\n", project.ID, project.Name)
	return nil
}

// ProjectsRemoveCmd deletes a project.
type ProjectsRemoveCmd struct {
	ID    int64 `arg:"" help:"Project ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *ProjectsRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get project details for confirmation
	project, err := client.GetProject(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete project #%d (%s)?", project.ID, project.Name)
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

	if err := client.DeleteProject(ctx, c.ID); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted project #%d\n", c.ID)
	return nil
}

// outputProjects writes projects in the specified format.
func outputProjects(w io.Writer, projects []api.Project, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, projects)
	case output.ModePlain:
		headers := []string{"ID", "Name", "Client", "Code", "Active", "Billable"}
		rows := make([][]string, len(projects))
		for i, p := range projects {
			rows[i] = []string{
				strconv.FormatInt(p.ID, 10),
				p.Name,
				p.Client.Name,
				p.Code,
				strconv.FormatBool(p.IsActive),
				strconv.FormatBool(p.IsBillable),
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Name", "Client", "Code", "Active", "Billable")
		for _, p := range projects {
			active := "Yes"
			if !p.IsActive {
				active = "No"
			}
			billable := "Yes"
			if !p.IsBillable {
				billable = "No"
			}
			t.AddRow(
				strconv.FormatInt(p.ID, 10),
				p.Name,
				p.Client.Name,
				p.Code,
				active,
				billable,
			)
		}
		return t.Render()
	}
}

// outputProject writes a single project in the specified format.
func outputProject(w io.Writer, project *api.Project, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, project)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%t\t%t\n",
			project.ID, project.Name, project.Client.Name, project.Code, project.IsActive, project.IsBillable)
		return nil
	default:
		fmt.Fprintf(w, "ID:       %d\n", project.ID)
		fmt.Fprintf(w, "Name:     %s\n", project.Name)
		fmt.Fprintf(w, "Client:   %s\n", project.Client.Name)
		if project.Code != "" {
			fmt.Fprintf(w, "Code:     %s\n", project.Code)
		}
		fmt.Fprintf(w, "Active:   %t\n", project.IsActive)
		fmt.Fprintf(w, "Billable: %t\n", project.IsBillable)
		fmt.Fprintf(w, "Bill By:  %s\n", project.BillBy)
		if project.HourlyRate != nil {
			fmt.Fprintf(w, "Rate:     %.2f\n", *project.HourlyRate)
		}
		fmt.Fprintf(w, "Budget By: %s\n", project.BudgetBy)
		if project.Budget != nil {
			fmt.Fprintf(w, "Budget:   %.2f\n", *project.Budget)
		}
		if project.Notes != "" {
			fmt.Fprintf(w, "Notes:    %s\n", project.Notes)
		}
		if project.StartsOn != nil {
			fmt.Fprintf(w, "Starts:   %s\n", *project.StartsOn)
		}
		if project.EndsOn != nil {
			fmt.Fprintf(w, "Ends:     %s\n", *project.EndsOn)
		}
		return nil
	}
}
