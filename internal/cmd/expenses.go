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
	"github.com/dedene/harvest-cli/internal/dateparse"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// ExpensesCmd groups expense subcommands.
type ExpensesCmd struct {
	List       ExpensesListCmd       `cmd:"" help:"List expenses"`
	Show       ExpensesShowCmd       `cmd:"" help:"Show an expense"`
	Add        ExpensesAddCmd        `cmd:"" help:"Create an expense"`
	Edit       ExpensesEditCmd       `cmd:"" help:"Update an expense"`
	Remove     ExpensesRemoveCmd     `cmd:"" help:"Delete an expense"`
	Receipt    ExpensesReceiptCmd    `cmd:"" help:"Upload receipt to expense"`
	Categories ExpensesCategoriesCmd `cmd:"" help:"List expense categories"`
}

// ExpensesListCmd lists expenses with filters.
type ExpensesListCmd struct {
	User          string `help:"Filter by user ID or 'me'"`
	HarvestClient string `help:"Filter by client ID or name" name:"harvest-client" short:"c"`
	Project       string `help:"Filter by project ID or name" short:"p"`
	Billed        bool   `help:"Only billed expenses"`
	Unbilled      bool   `help:"Only unbilled expenses"`
	UpdatedSince  string `help:"Filter by updated since (ISO datetime)"`
	From          string `help:"Start date (YYYY-MM-DD or 'today')" short:"f"`
	To            string `help:"End date" short:"t"`
}

func (c *ExpensesListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.ExpenseListOptions{}

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

	// Parse client filter
	if c.HarvestClient != "" {
		clientID, err := resolveClientID(ctx, client, c.HarvestClient)
		if err != nil {
			return err
		}
		opts.ClientID = clientID
	}

	// Parse project filter
	if c.Project != "" {
		projectID, err := resolveProjectID(ctx, client, c.Project)
		if err != nil {
			return err
		}
		opts.ProjectID = projectID
	}

	// Handle billed/unbilled filters
	if c.Billed {
		t := true
		opts.IsBilled = &t
	} else if c.Unbilled {
		f := false
		opts.IsBilled = &f
	}

	// Parse date filters
	if c.UpdatedSince != "" {
		t, err := dateparse.Parse(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("invalid updated_since date: %w", err)
		}
		opts.UpdatedSince = t.Format("2006-01-02T15:04:05Z")
	}

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

	expenses, err := client.ListAllExpenses(ctx, opts)
	if err != nil {
		return fmt.Errorf("list expenses: %w", err)
	}

	return outputExpenses(os.Stdout, expenses, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ExpensesShowCmd shows a single expense.
type ExpensesShowCmd struct {
	ID int64 `arg:"" help:"Expense ID"`
}

func (c *ExpensesShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	expense, err := client.GetExpense(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get expense: %w", err)
	}

	return outputExpense(os.Stdout, expense, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ExpensesAddCmd creates a new expense.
type ExpensesAddCmd struct {
	Project   string  `help:"Project ID or name" short:"p" required:""`
	Category  string  `help:"Expense category ID or name" required:""`
	Date      string  `help:"Date (default: today)" short:"d"`
	TotalCost float64 `help:"Total cost amount" required:""`
	Notes     string  `help:"Notes" short:"n"`
	Units     int     `help:"Units (for unit-based categories)"`
	Billable  *bool   `help:"Whether expense is billable"`
	Receipt   string  `help:"Path to receipt file"`
}

func (c *ExpensesAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	projectID, err := resolveProjectID(ctx, client, c.Project)
	if err != nil {
		return err
	}

	categoryID, err := resolveExpenseCategoryID(ctx, client, c.Category)
	if err != nil {
		return err
	}

	input := &api.ExpenseInput{
		ProjectID:         projectID,
		ExpenseCategoryID: categoryID,
		TotalCost:         &c.TotalCost,
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

	if c.Notes != "" {
		input.Notes = &c.Notes
	}

	if c.Units > 0 {
		input.Units = &c.Units
	}

	if c.Billable != nil {
		input.Billable = c.Billable
	}

	expense, err := client.CreateExpense(ctx, input)
	if err != nil {
		return fmt.Errorf("create expense: %w", err)
	}

	// Upload receipt if provided
	if c.Receipt != "" {
		expense, err = client.UploadExpenseReceipt(ctx, expense.ID, c.Receipt)
		if err != nil {
			return fmt.Errorf("upload receipt: %w", err)
		}
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, expense)
	}

	fmt.Fprintf(os.Stdout, "Created expense #%d: %s - %.2f on %s\n",
		expense.ID, expense.ExpenseCategory.Name, expense.TotalCost, expense.SpentDate)
	return nil
}

// ExpensesEditCmd updates an existing expense.
type ExpensesEditCmd struct {
	ID            int64   `arg:"" help:"Expense ID"`
	Project       string  `help:"Project ID or name"`
	Category      string  `help:"Expense category ID or name"`
	Date          string  `help:"Date"`
	TotalCost     float64 `help:"Total cost amount"`
	Notes         string  `help:"Notes"`
	Units         int     `help:"Units (for unit-based categories)"`
	Billable      *bool   `help:"Whether expense is billable"`
	DeleteReceipt bool    `help:"Delete the attached receipt"`
}

func (c *ExpensesEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.ExpenseInput{}
	hasChanges := false

	if c.Project != "" {
		projectID, err := resolveProjectID(ctx, client, c.Project)
		if err != nil {
			return err
		}
		input.ProjectID = projectID
		hasChanges = true
	}

	if c.Category != "" {
		categoryID, err := resolveExpenseCategoryID(ctx, client, c.Category)
		if err != nil {
			return err
		}
		input.ExpenseCategoryID = categoryID
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

	if c.TotalCost > 0 {
		input.TotalCost = &c.TotalCost
		hasChanges = true
	}

	if c.Notes != "" {
		input.Notes = &c.Notes
		hasChanges = true
	}

	if c.Units > 0 {
		input.Units = &c.Units
		hasChanges = true
	}

	if c.Billable != nil {
		input.Billable = c.Billable
		hasChanges = true
	}

	if c.DeleteReceipt {
		t := true
		input.DeleteReceipt = &t
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	expense, err := client.UpdateExpense(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update expense: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, expense)
	}

	fmt.Fprintf(os.Stdout, "Updated expense #%d: %s - %.2f\n",
		expense.ID, expense.ExpenseCategory.Name, expense.TotalCost)
	return nil
}

// ExpensesRemoveCmd deletes an expense.
type ExpensesRemoveCmd struct {
	ID    int64 `arg:"" help:"Expense ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *ExpensesRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get expense details for confirmation
	expense, err := client.GetExpense(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get expense: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete expense #%d (%s - %.2f on %s)?",
			expense.ID, expense.ExpenseCategory.Name, expense.TotalCost, expense.SpentDate)
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

	if err := client.DeleteExpense(ctx, c.ID); err != nil {
		return fmt.Errorf("delete expense: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted expense #%d\n", c.ID)
	return nil
}

// ExpensesReceiptCmd uploads a receipt to an expense.
type ExpensesReceiptCmd struct {
	ID      int64  `arg:"" help:"Expense ID"`
	Receipt string `arg:"" help:"Path to receipt file"`
}

func (c *ExpensesReceiptCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Verify file exists
	if _, err := os.Stat(c.Receipt); os.IsNotExist(err) {
		return fmt.Errorf("receipt file not found: %s", c.Receipt)
	}

	expense, err := client.UploadExpenseReceipt(ctx, c.ID, c.Receipt)
	if err != nil {
		return fmt.Errorf("upload receipt: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, expense)
	}

	fmt.Fprintf(os.Stdout, "Uploaded receipt to expense #%d\n", expense.ID)
	if expense.Receipt != nil {
		fmt.Fprintf(os.Stdout, "  File: %s\n", expense.Receipt.FileName)
	}
	return nil
}

// ExpensesCategoriesCmd lists expense categories.
type ExpensesCategoriesCmd struct {
	Active *bool `help:"Filter by active status"`
}

func (c *ExpensesCategoriesCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.ExpenseCategoryListOptions{
		IsActive: c.Active,
	}

	categories, err := client.ListAllExpenseCategories(ctx, opts)
	if err != nil {
		return fmt.Errorf("list expense categories: %w", err)
	}

	return outputExpenseCategories(os.Stdout, categories, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// resolveExpenseCategoryID resolves a category identifier (ID or name) to an ID.
func resolveExpenseCategoryID(ctx context.Context, client *api.Client, identifier string) (int64, error) {
	// Try parsing as ID first
	if id, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return id, nil
	}

	// Search by name
	categories, err := client.ListAllExpenseCategories(ctx, api.ExpenseCategoryListOptions{})
	if err != nil {
		return 0, fmt.Errorf("list expense categories: %w", err)
	}

	identifierLower := strings.ToLower(identifier)
	for _, cat := range categories {
		if strings.ToLower(cat.Name) == identifierLower {
			return cat.ID, nil
		}
	}

	return 0, fmt.Errorf("expense category not found: %s", identifier)
}

// outputExpenses writes expenses in the specified format.
func outputExpenses(w io.Writer, expenses []api.Expense, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, expenses)
	case output.ModePlain:
		headers := []string{"ID", "Date", "Project", "Category", "Cost", "Billed", "Notes"}
		rows := make([][]string, len(expenses))
		for i, e := range expenses {
			notes := e.Notes
			if len(notes) > 30 {
				notes = notes[:27] + "..."
			}
			rows[i] = []string{
				strconv.FormatInt(e.ID, 10),
				e.SpentDate,
				e.Project.Name,
				e.ExpenseCategory.Name,
				fmt.Sprintf("%.2f", e.TotalCost),
				strconv.FormatBool(e.IsBilled),
				notes,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Date", "Project", "Category", "Cost", "Billed", "Notes")
		for _, e := range expenses {
			notes := e.Notes
			if len(notes) > 30 {
				notes = notes[:27] + "..."
			}
			t.AddRow(
				strconv.FormatInt(e.ID, 10),
				e.SpentDate,
				e.Project.Name,
				e.ExpenseCategory.Name,
				fmt.Sprintf("%.2f", e.TotalCost),
				strconv.FormatBool(e.IsBilled),
				notes,
			)
		}
		return t.Render()
	}
}

// outputExpense writes a single expense in the specified format.
func outputExpense(w io.Writer, e *api.Expense, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, e)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%.2f\t%t\t%s\n",
			e.ID, e.SpentDate, e.Project.Name, e.ExpenseCategory.Name,
			e.TotalCost, e.IsBilled, e.Notes)
		return nil
	default:
		fmt.Fprintf(w, "ID:       %d\n", e.ID)
		fmt.Fprintf(w, "Date:     %s\n", e.SpentDate)
		fmt.Fprintf(w, "Project:  %s\n", e.Project.Name)
		fmt.Fprintf(w, "Client:   %s\n", e.Client.Name)
		fmt.Fprintf(w, "Category: %s\n", e.ExpenseCategory.Name)
		fmt.Fprintf(w, "Cost:     %.2f\n", e.TotalCost)
		fmt.Fprintf(w, "Units:    %.2f\n", e.Units)
		fmt.Fprintf(w, "Billable: %t\n", e.Billable)
		fmt.Fprintf(w, "Billed:   %t\n", e.IsBilled)
		fmt.Fprintf(w, "Status:   %s\n", e.ApprovalStatus)
		if e.IsLocked {
			fmt.Fprintf(w, "Locked:   %s\n", e.LockedReason)
		}
		if e.Notes != "" {
			fmt.Fprintf(w, "Notes:    %s\n", e.Notes)
		}
		if e.Receipt != nil {
			fmt.Fprintf(w, "Receipt:  %s\n", e.Receipt.FileName)
		}
		if e.Invoice != nil {
			fmt.Fprintf(w, "Invoice:  #%s\n", e.Invoice.Number)
		}
		return nil
	}
}

// outputExpenseCategories writes expense categories in the specified format.
func outputExpenseCategories(w io.Writer, categories []api.ExpenseCategory, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, categories)
	case output.ModePlain:
		headers := []string{"ID", "Name", "Active", "Unit Name", "Unit Price"}
		rows := make([][]string, len(categories))
		for i, c := range categories {
			unitName := ""
			if c.UnitName != nil {
				unitName = *c.UnitName
			}
			unitPrice := ""
			if c.UnitPrice != nil {
				unitPrice = fmt.Sprintf("%.2f", *c.UnitPrice)
			}
			rows[i] = []string{
				strconv.FormatInt(c.ID, 10),
				c.Name,
				strconv.FormatBool(c.IsActive),
				unitName,
				unitPrice,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Name", "Active", "Unit Name", "Unit Price")
		for _, c := range categories {
			unitName := ""
			if c.UnitName != nil {
				unitName = *c.UnitName
			}
			unitPrice := ""
			if c.UnitPrice != nil {
				unitPrice = fmt.Sprintf("%.2f", *c.UnitPrice)
			}
			t.AddRow(
				strconv.FormatInt(c.ID, 10),
				c.Name,
				strconv.FormatBool(c.IsActive),
				unitName,
				unitPrice,
			)
		}
		return t.Render()
	}
}
