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
)

// ReportsCmd groups report subcommands.
type ReportsCmd struct {
	Time       ReportsTimeCmd       `cmd:"" help:"Time reports"`
	Expenses   ReportsExpensesCmd   `cmd:"" help:"Expense reports"`
	Uninvoiced ReportsUninvoicedCmd `cmd:"" help:"Uninvoiced amounts report"`
	Budget     ReportsBudgetCmd     `cmd:"" help:"Project budget report"`
}

// ReportsTimeCmd generates time reports.
type ReportsTimeCmd struct {
	By   string `help:"Group by: clients, projects, tasks, team" default:"projects" enum:"clients,projects,tasks,team"`
	From string `help:"Start date (required)" short:"f" required:""`
	To   string `help:"End date (required)" short:"t" required:""`
}

func (c *ReportsTimeCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Parse dates
	fromDate, err := dateparse.Parse(c.From)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}
	toDate, err := dateparse.Parse(c.To)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}

	opts := api.ReportListOptions{
		From: dateparse.FormatDate(fromDate),
		To:   dateparse.FormatDate(toDate),
	}

	var results []api.TimeReportResult

	switch c.By {
	case "clients":
		results, err = client.ListAllTimeReportsByClients(ctx, opts)
	case "projects":
		results, err = client.ListAllTimeReportsByProjects(ctx, opts)
	case "tasks":
		results, err = client.ListAllTimeReportsByTasks(ctx, opts)
	case "team":
		results, err = client.ListAllTimeReportsByTeam(ctx, opts)
	default:
		return fmt.Errorf("invalid group by: %s", c.By)
	}

	if err != nil {
		return fmt.Errorf("get time report: %w", err)
	}

	// Warn if approaching rate limit
	if warn := client.WarnIfNearReportsLimit(); warn != "" {
		fmt.Fprintln(os.Stderr, warn)
	}

	return outputTimeReport(os.Stdout, results, c.By, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ReportsExpensesCmd generates expense reports.
type ReportsExpensesCmd struct {
	By   string `help:"Group by: clients, projects, categories, team" default:"projects" enum:"clients,projects,categories,team"`
	From string `help:"Start date (required)" short:"f" required:""`
	To   string `help:"End date (required)" short:"t" required:""`
}

func (c *ReportsExpensesCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Parse dates
	fromDate, err := dateparse.Parse(c.From)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}
	toDate, err := dateparse.Parse(c.To)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}

	opts := api.ReportListOptions{
		From: dateparse.FormatDate(fromDate),
		To:   dateparse.FormatDate(toDate),
	}

	var results []api.ExpenseReportResult

	switch c.By {
	case "clients":
		results, err = client.ListAllExpenseReportsByClients(ctx, opts)
	case "projects":
		results, err = client.ListAllExpenseReportsByProjects(ctx, opts)
	case "categories":
		results, err = client.ListAllExpenseReportsByCategories(ctx, opts)
	case "team":
		results, err = client.ListAllExpenseReportsByTeam(ctx, opts)
	default:
		return fmt.Errorf("invalid group by: %s", c.By)
	}

	if err != nil {
		return fmt.Errorf("get expense report: %w", err)
	}

	// Warn if approaching rate limit
	if warn := client.WarnIfNearReportsLimit(); warn != "" {
		fmt.Fprintln(os.Stderr, warn)
	}

	return outputExpenseReport(os.Stdout, results, c.By, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ReportsUninvoicedCmd generates uninvoiced amounts report.
type ReportsUninvoicedCmd struct {
	From string `help:"Start date (required)" short:"f" required:""`
	To   string `help:"End date (required)" short:"t" required:""`
}

func (c *ReportsUninvoicedCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Parse dates
	fromDate, err := dateparse.Parse(c.From)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}
	toDate, err := dateparse.Parse(c.To)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}

	opts := api.ReportListOptions{
		From: dateparse.FormatDate(fromDate),
		To:   dateparse.FormatDate(toDate),
	}

	results, err := client.ListAllUninvoicedReport(ctx, opts)
	if err != nil {
		return fmt.Errorf("get uninvoiced report: %w", err)
	}

	// Warn if approaching rate limit
	if warn := client.WarnIfNearReportsLimit(); warn != "" {
		fmt.Fprintln(os.Stderr, warn)
	}

	return outputUninvoicedReport(os.Stdout, results, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ReportsBudgetCmd generates project budget report.
type ReportsBudgetCmd struct {
	Active   bool `help:"Only active projects"`
	Inactive bool `help:"Only inactive projects"`
}

func (c *ReportsBudgetCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.ProjectBudgetReportOptions{}

	if c.Active {
		t := true
		opts.IsActive = &t
	} else if c.Inactive {
		f := false
		opts.IsActive = &f
	}

	results, err := client.ListAllProjectBudgetReport(ctx, opts)
	if err != nil {
		return fmt.Errorf("get budget report: %w", err)
	}

	// Warn if approaching rate limit
	if warn := client.WarnIfNearReportsLimit(); warn != "" {
		fmt.Fprintln(os.Stderr, warn)
	}

	return outputBudgetReport(os.Stdout, results, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// outputTimeReport writes time report results in the specified format.
func outputTimeReport(w io.Writer, results []api.TimeReportResult, groupBy string, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, results)
	case output.ModePlain:
		return outputTimeReportTSV(w, results, groupBy)
	default:
		return outputTimeReportTable(w, results, groupBy)
	}
}

func outputTimeReportTSV(w io.Writer, results []api.TimeReportResult, groupBy string) error {
	var headers []string
	var rows [][]string

	switch groupBy {
	case "clients":
		headers = []string{"ClientID", "Client", "TotalHours", "BillableHours", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.ClientID, 10),
				r.ClientName,
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	case "projects":
		headers = []string{"ProjectID", "Project", "Client", "TotalHours", "BillableHours", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.ProjectID, 10),
				r.ProjectName,
				r.ClientName,
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	case "tasks":
		headers = []string{"TaskID", "Task", "TotalHours", "BillableHours", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.TaskID, 10),
				r.TaskName,
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	case "team":
		headers = []string{"UserID", "User", "TotalHours", "BillableHours", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.UserID, 10),
				r.UserName,
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	}

	return output.WriteTSV(w, headers, rows)
}

func outputTimeReportTable(w io.Writer, results []api.TimeReportResult, groupBy string) error {
	var t *output.Table

	switch groupBy {
	case "clients":
		t = output.NewTable(w, "ID", "Client", "Total Hours", "Billable Hours", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.ClientID, 10),
				r.ClientName,
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	case "projects":
		t = output.NewTable(w, "ID", "Project", "Client", "Total Hours", "Billable Hours", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.ProjectID, 10),
				truncate(r.ProjectName, 25),
				truncate(r.ClientName, 20),
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	case "tasks":
		t = output.NewTable(w, "ID", "Task", "Total Hours", "Billable Hours", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.TaskID, 10),
				truncate(r.TaskName, 30),
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	case "team":
		t = output.NewTable(w, "ID", "User", "Total Hours", "Billable Hours", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.UserID, 10),
				r.UserName,
				fmt.Sprintf("%.2f", r.TotalHours),
				fmt.Sprintf("%.2f", r.BillableHours),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	}

	return t.Render()
}

// outputExpenseReport writes expense report results in the specified format.
func outputExpenseReport(w io.Writer, results []api.ExpenseReportResult, groupBy string, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, results)
	case output.ModePlain:
		return outputExpenseReportTSV(w, results, groupBy)
	default:
		return outputExpenseReportTable(w, results, groupBy)
	}
}

func outputExpenseReportTSV(w io.Writer, results []api.ExpenseReportResult, groupBy string) error {
	var headers []string
	var rows [][]string

	switch groupBy {
	case "clients":
		headers = []string{"ClientID", "Client", "TotalAmount", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.ClientID, 10),
				r.ClientName,
				fmt.Sprintf("%.2f", r.TotalAmount),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	case "projects":
		headers = []string{"ProjectID", "Project", "Client", "TotalAmount", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.ProjectID, 10),
				r.ProjectName,
				r.ClientName,
				fmt.Sprintf("%.2f", r.TotalAmount),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	case "categories":
		headers = []string{"CategoryID", "Category", "TotalAmount", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.ExpenseCategoryID, 10),
				r.ExpenseCategoryName,
				fmt.Sprintf("%.2f", r.TotalAmount),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	case "team":
		headers = []string{"UserID", "User", "TotalAmount", "BillableAmount", "Currency"}
		for _, r := range results {
			rows = append(rows, []string{
				strconv.FormatInt(r.UserID, 10),
				r.UserName,
				fmt.Sprintf("%.2f", r.TotalAmount),
				fmt.Sprintf("%.2f", r.BillableAmount),
				r.Currency,
			})
		}
	}

	return output.WriteTSV(w, headers, rows)
}

func outputExpenseReportTable(w io.Writer, results []api.ExpenseReportResult, groupBy string) error {
	var t *output.Table

	switch groupBy {
	case "clients":
		t = output.NewTable(w, "ID", "Client", "Total Amount", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.ClientID, 10),
				r.ClientName,
				formatAmount(r.TotalAmount, r.Currency),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	case "projects":
		t = output.NewTable(w, "ID", "Project", "Client", "Total Amount", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.ProjectID, 10),
				truncate(r.ProjectName, 25),
				truncate(r.ClientName, 20),
				formatAmount(r.TotalAmount, r.Currency),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	case "categories":
		t = output.NewTable(w, "ID", "Category", "Total Amount", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.ExpenseCategoryID, 10),
				truncate(r.ExpenseCategoryName, 30),
				formatAmount(r.TotalAmount, r.Currency),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	case "team":
		t = output.NewTable(w, "ID", "User", "Total Amount", "Billable Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.UserID, 10),
				r.UserName,
				formatAmount(r.TotalAmount, r.Currency),
				formatAmount(r.BillableAmount, r.Currency),
			)
		}
	}

	return t.Render()
}

// outputUninvoicedReport writes uninvoiced report results in the specified format.
func outputUninvoicedReport(w io.Writer, results []api.UninvoicedReportResult, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, results)
	case output.ModePlain:
		headers := []string{"ProjectID", "Project", "Client", "UninvoicedHours", "UninvoicedExpenses", "UninvoicedAmount", "Currency"}
		rows := make([][]string, len(results))
		for i, r := range results {
			rows[i] = []string{
				strconv.FormatInt(r.ProjectID, 10),
				r.ProjectName,
				r.ClientName,
				fmt.Sprintf("%.2f", r.UninvoicedHours),
				fmt.Sprintf("%.2f", r.UninvoicedExpenses),
				fmt.Sprintf("%.2f", r.UninvoicedAmount),
				r.Currency,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Project", "Client", "Uninv. Hours", "Uninv. Expenses", "Uninv. Amount")
		for _, r := range results {
			t.AddRow(
				strconv.FormatInt(r.ProjectID, 10),
				truncate(r.ProjectName, 20),
				truncate(r.ClientName, 15),
				fmt.Sprintf("%.2f", r.UninvoicedHours),
				formatAmount(r.UninvoicedExpenses, r.Currency),
				formatAmount(r.UninvoicedAmount, r.Currency),
			)
		}
		return t.Render()
	}
}

// outputBudgetReport writes budget report results in the specified format.
func outputBudgetReport(w io.Writer, results []api.ProjectBudgetReportResult, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, results)
	case output.ModePlain:
		headers := []string{"ProjectID", "Project", "Client", "BudgetBy", "Budget", "Spent", "Remaining", "Active"}
		rows := make([][]string, len(results))
		for i, r := range results {
			budget := "-"
			if r.Budget != nil {
				budget = fmt.Sprintf("%.2f", *r.Budget)
			}
			rows[i] = []string{
				strconv.FormatInt(r.ProjectID, 10),
				r.ProjectName,
				r.ClientName,
				r.BudgetBy,
				budget,
				fmt.Sprintf("%.2f", r.BudgetSpent),
				fmt.Sprintf("%.2f", r.BudgetRemaining),
				strconv.FormatBool(r.IsActive),
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Project", "Client", "Budget By", "Budget", "Spent", "Remaining", "Active")
		for _, r := range results {
			budget := "-"
			if r.Budget != nil {
				budget = fmt.Sprintf("%.2f", *r.Budget)
			}
			active := "No"
			if r.IsActive {
				active = "Yes"
			}
			t.AddRow(
				strconv.FormatInt(r.ProjectID, 10),
				truncate(r.ProjectName, 20),
				truncate(r.ClientName, 15),
				r.BudgetBy,
				budget,
				fmt.Sprintf("%.2f", r.BudgetSpent),
				fmt.Sprintf("%.2f", r.BudgetRemaining),
				active,
			)
		}
		return t.Render()
	}
}

// formatAmount formats an amount with currency.
func formatAmount(amount float64, currency string) string {
	if currency == "" {
		return fmt.Sprintf("%.2f", amount)
	}
	return fmt.Sprintf("%.2f %s", amount, currency)
}

// truncate shortens a string to max length with ellipsis.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
