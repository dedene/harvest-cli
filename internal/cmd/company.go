package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/output"
)

// CompanyCmd shows company information.
type CompanyCmd struct {
	Edit               bool `help:"Edit company settings" short:"e"`
	WantsTimestamps    *bool `help:"Enable timestamp timers (with --edit)" name:"timestamps"`
	WeeklyCapacity     *int  `help:"Weekly capacity in seconds (with --edit)" name:"weekly-capacity"`
}

func (c *CompanyCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	if c.Edit {
		return c.runEdit(ctx, client, cli)
	}

	company, err := client.GetCompany(ctx)
	if err != nil {
		return fmt.Errorf("get company: %w", err)
	}

	return outputCompany(os.Stdout, company, output.ModeFromFlags(cli.JSON, cli.Plain))
}

func (c *CompanyCmd) runEdit(ctx context.Context, client *api.Client, cli *CLI) error {
	input := &api.CompanyUpdateInput{}
	hasChanges := false

	if c.WantsTimestamps != nil {
		input.WantsTimestampTimers = c.WantsTimestamps
		hasChanges = true
	}
	if c.WeeklyCapacity != nil {
		input.WeeklyCapacity = c.WeeklyCapacity
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified; use --timestamps or --weekly-capacity with --edit")
	}

	company, err := client.UpdateCompany(ctx, input)
	if err != nil {
		return fmt.Errorf("update company: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, company)
	}

	fmt.Fprintf(os.Stdout, "Updated company: %s\n", company.Name)
	return nil
}

func outputCompany(w io.Writer, company *api.Company, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, company)
	case output.ModePlain:
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n",
			company.Name,
			company.WeekStartDay,
			company.TimeFormat,
			company.PlanType,
			company.WeeklyCapacity/3600,
		)
		return nil
	default:
		fmt.Fprintf(w, "Name:             %s\n", company.Name)
		fmt.Fprintf(w, "Domain:           %s\n", company.FullDomain)
		fmt.Fprintf(w, "Active:           %v\n", company.IsActive)
		fmt.Fprintf(w, "Plan:             %s\n", company.PlanType)
		fmt.Fprintf(w, "Week Start:       %s\n", company.WeekStartDay)
		fmt.Fprintf(w, "Time Format:      %s\n", company.TimeFormat)
		fmt.Fprintf(w, "Date Format:      %s\n", company.DateFormat)
		fmt.Fprintf(w, "Clock:            %s\n", company.Clock)
		fmt.Fprintf(w, "Weekly Capacity:  %dh\n", company.WeeklyCapacity/3600)
		fmt.Fprintf(w, "Timestamp Timers: %v\n", company.WantsTimestampTimers)
		fmt.Fprintf(w, "\nFeatures:\n")
		fmt.Fprintf(w, "  Expenses:       %v\n", company.ExpenseFeature)
		fmt.Fprintf(w, "  Invoices:       %v\n", company.InvoiceFeature)
		fmt.Fprintf(w, "  Estimates:      %v\n", company.EstimateFeature)
		fmt.Fprintf(w, "  Approvals:      %v\n", company.ApprovalFeature)
		fmt.Fprintf(w, "  Team:           %v\n", company.TeamFeature)
		return nil
	}
}
