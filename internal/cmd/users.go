package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// UsersCmd groups user subcommands.
type UsersCmd struct {
	List   UsersListCmd   `cmd:"" help:"List all users"`
	Show   UsersShowCmd   `cmd:"" help:"Show a user by ID"`
	Me     UsersMeCmd     `cmd:"" help:"Show current authenticated user"`
	Add    UsersAddCmd    `cmd:"" help:"Create a new user"`
	Edit   UsersEditCmd   `cmd:"" help:"Update a user"`
	Remove UsersRemoveCmd `cmd:"" help:"Delete/deactivate a user"`
}

// UsersListCmd lists all users with optional filters.
type UsersListCmd struct {
	Active       *bool  `help:"Filter by active status"`
	UpdatedSince string `help:"Filter by updated_since (ISO 8601)"`
}

func (c *UsersListCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	opts := api.UserListOptions{
		IsActive:     c.Active,
		UpdatedSince: c.UpdatedSince,
	}

	users, err := client.ListAllUsers(ctx, opts)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	return outputUsers(os.Stdout, users, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// UsersShowCmd shows a single user by ID.
type UsersShowCmd struct {
	ID int64 `arg:"" help:"User ID"`
}

func (c *UsersShowCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	user, err := client.GetUser(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	return outputUser(os.Stdout, user, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// UsersMeCmd shows the current authenticated user.
type UsersMeCmd struct{}

func (c *UsersMeCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	return outputUser(os.Stdout, user, output.ModeFromFlags(cli.JSON, cli.Plain))
}

// UsersAddCmd creates a new user.
type UsersAddCmd struct {
	Email                        string   `help:"User email" required:""`
	FirstName                    string   `help:"First name" required:"" name:"first-name"`
	LastName                     string   `help:"Last name" required:"" name:"last-name"`
	Timezone                     string   `help:"Timezone"`
	HasAccessToAllFutureProjects *bool    `help:"Has access to all future projects" name:"access-all-projects"`
	IsContractor                 *bool    `help:"Is contractor" name:"contractor"`
	IsActive                     *bool    `help:"Is active" name:"active"`
	WeeklyCapacity               *int     `help:"Weekly capacity (seconds)" name:"weekly-capacity"`
	DefaultHourlyRate            *float64 `help:"Default hourly rate" name:"hourly-rate"`
	CostRate                     *float64 `help:"Cost rate" name:"cost-rate"`
	Roles                        []string `help:"Roles (comma-separated)"`
	AccessRoles                  []string `help:"Access roles (comma-separated)" name:"access-roles"`
}

func (c *UsersAddCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.UserInput{
		Email:                        c.Email,
		FirstName:                    c.FirstName,
		LastName:                     c.LastName,
		HasAccessToAllFutureProjects: c.HasAccessToAllFutureProjects,
		IsContractor:                 c.IsContractor,
		IsActive:                     c.IsActive,
		WeeklyCapacity:               c.WeeklyCapacity,
		DefaultHourlyRate:            c.DefaultHourlyRate,
		CostRate:                     c.CostRate,
		Roles:                        c.Roles,
		AccessRoles:                  c.AccessRoles,
	}

	if c.Timezone != "" {
		input.Timezone = &c.Timezone
	}

	user, err := client.CreateUser(ctx, input)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, user)
	}

	fmt.Fprintf(os.Stdout, "Created user #%d: %s (%s)\n", user.ID, user.FullName(), user.Email)
	return nil
}

// UsersEditCmd updates an existing user.
type UsersEditCmd struct {
	ID                           int64    `arg:"" help:"User ID"`
	Email                        string   `help:"User email"`
	FirstName                    string   `help:"First name" name:"first-name"`
	LastName                     string   `help:"Last name" name:"last-name"`
	Timezone                     string   `help:"Timezone"`
	HasAccessToAllFutureProjects *bool    `help:"Has access to all future projects" name:"access-all-projects"`
	IsContractor                 *bool    `help:"Is contractor" name:"contractor"`
	IsActive                     *bool    `help:"Is active" name:"active"`
	WeeklyCapacity               *int     `help:"Weekly capacity (seconds)" name:"weekly-capacity"`
	DefaultHourlyRate            *float64 `help:"Default hourly rate" name:"hourly-rate"`
	CostRate                     *float64 `help:"Cost rate" name:"cost-rate"`
	Roles                        []string `help:"Roles (comma-separated)"`
	AccessRoles                  []string `help:"Access roles (comma-separated)" name:"access-roles"`
}

func (c *UsersEditCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	input := &api.UserInput{}
	hasChanges := false

	if c.Email != "" {
		input.Email = c.Email
		hasChanges = true
	}
	if c.FirstName != "" {
		input.FirstName = c.FirstName
		hasChanges = true
	}
	if c.LastName != "" {
		input.LastName = c.LastName
		hasChanges = true
	}
	if c.Timezone != "" {
		input.Timezone = &c.Timezone
		hasChanges = true
	}
	if c.HasAccessToAllFutureProjects != nil {
		input.HasAccessToAllFutureProjects = c.HasAccessToAllFutureProjects
		hasChanges = true
	}
	if c.IsContractor != nil {
		input.IsContractor = c.IsContractor
		hasChanges = true
	}
	if c.IsActive != nil {
		input.IsActive = c.IsActive
		hasChanges = true
	}
	if c.WeeklyCapacity != nil {
		input.WeeklyCapacity = c.WeeklyCapacity
		hasChanges = true
	}
	if c.DefaultHourlyRate != nil {
		input.DefaultHourlyRate = c.DefaultHourlyRate
		hasChanges = true
	}
	if c.CostRate != nil {
		input.CostRate = c.CostRate
		hasChanges = true
	}
	if len(c.Roles) > 0 {
		input.Roles = c.Roles
		hasChanges = true
	}
	if len(c.AccessRoles) > 0 {
		input.AccessRoles = c.AccessRoles
		hasChanges = true
	}

	if !hasChanges {
		return fmt.Errorf("no changes specified")
	}

	user, err := client.UpdateUser(ctx, c.ID, input)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if cli.JSON {
		return output.WriteJSON(os.Stdout, user)
	}

	fmt.Fprintf(os.Stdout, "Updated user #%d: %s (%s)\n", user.ID, user.FullName(), user.Email)
	return nil
}

// UsersRemoveCmd deletes/deactivates a user.
type UsersRemoveCmd struct {
	ID    int64 `arg:"" help:"User ID"`
	Force bool  `help:"Skip confirmation" short:"f"`
}

func (c *UsersRemoveCmd) Run(cli *CLI) error {
	ctx := context.Background()
	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get user details for confirmation
	user, err := client.GetUser(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if !c.Force {
		msg := fmt.Sprintf("Delete user #%d (%s, %s)?", user.ID, user.FullName(), user.Email)
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

	if err := client.DeleteUser(ctx, c.ID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Deleted user #%d\n", c.ID)
	return nil
}

// outputUsers writes users in the specified format.
func outputUsers(w io.Writer, users []api.User, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, users)
	case output.ModePlain:
		headers := []string{"ID", "Name", "Email", "Active", "Roles"}
		rows := make([][]string, len(users))
		for i, u := range users {
			active := "no"
			if u.IsActive {
				active = "yes"
			}
			roles := ""
			if len(u.Roles) > 0 {
				roles = u.Roles[0]
				if len(u.Roles) > 1 {
					roles += fmt.Sprintf(" +%d", len(u.Roles)-1)
				}
			}
			rows[i] = []string{
				strconv.FormatInt(u.ID, 10),
				u.FullName(),
				u.Email,
				active,
				roles,
			}
		}
		return output.WriteTSV(w, headers, rows)
	default:
		t := output.NewTable(w, "ID", "Name", "Email", "Active", "Roles")
		for _, u := range users {
			active := "no"
			if u.IsActive {
				active = "yes"
			}
			roles := ""
			if len(u.Roles) > 0 {
				roles = u.Roles[0]
				if len(u.Roles) > 1 {
					roles += fmt.Sprintf(" +%d", len(u.Roles)-1)
				}
			}
			t.AddRow(
				strconv.FormatInt(u.ID, 10),
				u.FullName(),
				u.Email,
				active,
				roles,
			)
		}
		return t.Render()
	}
}

// outputUser writes a single user in the specified format.
func outputUser(w io.Writer, user *api.User, mode output.Mode) error {
	switch mode {
	case output.ModeJSON:
		return output.WriteJSON(w, user)
	case output.ModePlain:
		fmt.Fprintf(w, "%d\t%s\t%s\t%v\n",
			user.ID, user.FullName(), user.Email, user.IsActive)
		return nil
	default:
		fmt.Fprintf(w, "ID:         %d\n", user.ID)
		fmt.Fprintf(w, "Name:       %s\n", user.FullName())
		fmt.Fprintf(w, "Email:      %s\n", user.Email)
		fmt.Fprintf(w, "Active:     %v\n", user.IsActive)
		fmt.Fprintf(w, "Timezone:   %s\n", user.Timezone)
		fmt.Fprintf(w, "Contractor: %v\n", user.IsContractor)
		if len(user.Roles) > 0 {
			fmt.Fprintf(w, "Roles:      %v\n", user.Roles)
		}
		if len(user.AccessRoles) > 0 {
			fmt.Fprintf(w, "Access:     %v\n", user.AccessRoles)
		}
		if user.WeeklyCapacity > 0 {
			fmt.Fprintf(w, "Capacity:   %dh/week\n", user.WeeklyCapacity/3600)
		}
		if user.DefaultHourlyRate != nil {
			fmt.Fprintf(w, "Rate:       $%.2f/h\n", *user.DefaultHourlyRate)
		}
		return nil
	}
}
