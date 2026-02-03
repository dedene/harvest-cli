package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/dedene/harvest-cli/internal/errfmt"
)

// RootFlags are global flags available to all commands.
type RootFlags struct {
	Account   string `help:"Account email or alias" short:"a" env:"HARVESTCLI_ACCOUNT"`
	AccountID int64  `help:"Harvest account ID override" env:"HARVESTCLI_ACCOUNT_ID"`
	Client    string `help:"OAuth client name override"`
	JSON      bool   `help:"Output as JSON" short:"j"`
	Plain     bool   `help:"Output as TSV (plain text)"`
	Verbose   bool   `help:"Verbose output" short:"v"`
	Color     string `help:"Color output: auto, always, never" default:"auto" enum:"auto,always,never"`
}

// CLI is the root command structure.
type CLI struct {
	RootFlags `embed:""`

	Version    kong.VersionFlag `help:"Print version and exit"`
	VersionCmd VersionCmd       `cmd:"" name:"version" help:"Show version information"`
	Auth       AuthCmd          `cmd:"" help:"Authentication commands"`
	Config     ConfigCmd        `cmd:"" help:"Configuration commands"`
	Time       TimeCmd          `cmd:"" help:"Time entry commands"`
	Timer      TimerCmd         `cmd:"" help:"Timer commands"`
	Projects   ProjectsCmd      `cmd:"" help:"Project commands"`
	Clients    ClientsCmd       `cmd:"" help:"Client commands"`
	Tasks      TasksCmd         `cmd:"" help:"Task commands"`
	Users      UsersCmd         `cmd:"" help:"User management commands"`
	Expenses   ExpensesCmd      `cmd:"" help:"Expense commands"`
	Estimates  EstimatesCmd     `cmd:"" help:"Estimate commands"`
	Invoices   InvoicesCmd      `cmd:"" help:"Invoice commands"`
	Reports    ReportsCmd       `cmd:"" help:"Report commands"`
	Company    CompanyCmd       `cmd:"" help:"Show company information"`
	Approvals  ApprovalsCmd     `cmd:"" help:"Approval workflow commands"`
	Bulk       BulkCmd          `cmd:"" help:"Bulk import/export operations"`
	Completion CompletionCmd    `cmd:"" help:"Generate shell completions"`
	Dashboard  DashboardCmd     `cmd:"" help:"Show weekly time tracking summary"`
}

type exitPanic struct{ code int }

// Execute parses args and runs the appropriate command.
func Execute(args []string) (err error) {
	parser, err := newParser()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			if ep, ok := r.(exitPanic); ok {
				if ep.code == 0 {
					err = nil
					return
				}
				err = &ExitError{Code: ep.code, Err: errors.New("exited")}
				return
			}
			panic(r)
		}
	}()

	// Show help when no command provided
	if len(args) == 0 {
		args = []string{"--help"}
	}

	kctx, err := parser.Parse(args)
	if err != nil {
		parsedErr := wrapParseError(err)
		_, _ = fmt.Fprintln(os.Stderr, parsedErr)
		return parsedErr
	}

	err = kctx.Run()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, errfmt.FormatError(err))
		return err
	}

	return nil
}

func wrapParseError(err error) error {
	if err == nil {
		return nil
	}

	var parseErr *kong.ParseError
	if errors.As(err, &parseErr) {
		return &ExitError{Code: 2, Err: parseErr}
	}

	return err
}

func newParser() (*kong.Kong, error) {
	cli := &CLI{}
	parser, err := kong.New(
		cli,
		kong.Name("harvest"),
		kong.Description("Harvest time tracking CLI"),
		kong.Vars{"version": VersionString()},
		kong.Exit(func(code int) { panic(exitPanic{code: code}) }),
		kong.BindTo(cli, (*CLI)(nil)),
		kong.Help(helpPrinter),
		kong.ConfigureHelp(helpOptions()),
	)
	if err != nil {
		return nil, err
	}

	return parser, nil
}
