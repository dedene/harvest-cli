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

// ClientsCmd groups client subcommands.
type ClientsCmd struct {
	List   ClientsListCmd   `cmd:"" help:"List all clients"`
	Show   ClientsShowCmd   `cmd:"" help:"Show a client"`
	Add    ClientsAddCmd    `cmd:"" help:"Create a client"`
	Edit   ClientsEditCmd   `cmd:"" help:"Update a client"`
	Remove ClientsRemoveCmd `cmd:"" help:"Delete a client"`
}

// ClientsListCmd lists clients with filters.
type ClientsListCmd struct {
	Active       *bool  `help:"Filter by active status"`
	UpdatedSince string `help:"Filter by updated since (ISO datetime)"`
}

func (c *ClientsListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.ClientListOptions{
		IsActive: c.Active,
	}
	if c.UpdatedSince != "" {
		t, err := dateparse.Parse(c.UpdatedSince)
		if err != nil {
			return fmt.Errorf("invalid updated_since date: %w", err)
		}
		opts.UpdatedSince = t.Format("2006-01-02T15:04:05Z")
	}

	clients, err := client.ListAllClients(ctx, opts)
	if err != nil {
		return fmt.Errorf("list clients: %w", err)
	}

	return outputClients(os.Stdout, clients, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ClientsShowCmd shows a single client.
type ClientsShowCmd struct {
	ID int64 `arg:"" help:"Client ID"`
}

func (c *ClientsShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	hc, err := client.GetClient(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}

	return outputClient(os.Stdout, hc, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// ClientsAddCmd creates a new client.
type ClientsAddCmd struct {
	Name     string `arg:"" help:"Client name"`
	Address  string `help:"Client address"`
	Currency string `help:"Currency code (e.g., USD, EUR)"`
	Active   *bool  `help:"Is active (default: true)"`
}

func (c *ClientsAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.ClientInput{
		Name:     c.Name,
		IsActive: c.Active,
	}
	if c.Address != "" {
		input.Address = &c.Address
	}
	if c.Currency != "" {
		input.Currency = &c.Currency
	}

	hc, err := client.CreateClient(ctx, input)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, hc)
	}

	fmt.Fprintf(os.Stdout, "Created client #%d: %s\n", hc.ID, hc.Name)
	return nil
}

// ClientsEditCmd updates an existing client.
type ClientsEditCmd struct {
	ID       int64  `arg:"" help:"Client ID"`
	Name     string `help:"Client name"`
	Address  string `help:"Client address"`
	Currency string `help:"Currency code"`
	Active   *bool  `help:"Is active"`
}

func (c *ClientsEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.ClientInput{}
	hasChanges := false

	if c.Name != "" {
		input.Name = c.Name
		hasChanges = true
	}
	if c.Address != "" {
		input.Address = &c.Address
		hasChanges = true
	}
	if c.Currency != "" {
		input.Currency = &c.Currency
		hasChanges = true
	}
	if c.Active != nil {
		input.IsActive = c.Active
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	hc, err := client.UpdateClient(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update client: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, hc)
	}

	fmt.Fprintf(os.Stdout, "Updated client #%d: %s\n", hc.ID, hc.Name)
	return nil
}

// ClientsRemoveCmd deletes a client.
type ClientsRemoveCmd struct {
	ID    int64 `arg:"" help:"Client ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *ClientsRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get client details for confirmation
	hc, err := client.GetClient(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get client: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete client #%d (%s)?", hc.ID, hc.Name)
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

	if err := client.DeleteClient(ctx, c.ID); err != nil {
		return fmt.Errorf("delete client: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted client #%d\n", c.ID)
	return nil
}

// outputClients writes clients in the specified format.
func outputClients(w io.Writer, clients []api.HarvestClient, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, clients)
	case output.ModePlain:
		headers := []string{"ID", "Name", "Active", "Currency"}
		rows := make([][]string, len(clients))
		for i, c := range clients {
			rows[i] = []string{
				strconv.FormatInt(c.ID, 10),
				c.Name,
				strconv.FormatBool(c.IsActive),
				c.Currency,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Name", "Active", "Currency")
		for _, c := range clients {
			t.AddRow(
				strconv.FormatInt(c.ID, 10),
				c.Name,
				strconv.FormatBool(c.IsActive),
				c.Currency,
			)
		}
		return t.Render()
	}
}

// outputClient writes a single client in the specified format.
func outputClient(w io.Writer, c *api.HarvestClient, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, c)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%t\t%s\n", c.ID, c.Name, c.IsActive, c.Currency)
		return nil
	default:
		fmt.Fprintf(w, "ID:       %d\n", c.ID)
		fmt.Fprintf(w, "Name:     %s\n", c.Name)
		fmt.Fprintf(w, "Active:   %t\n", c.IsActive)
		fmt.Fprintf(w, "Currency: %s\n", c.Currency)
		if c.Address != "" {
			fmt.Fprintf(w, "Address:  %s\n", c.Address)
		}
		return nil
	}
}
