# ⏱️ harvest-cli - Harvest in your terminal

A powerful command-line interface for [Harvest](https://www.getharvest.com/) time tracking.

<!-- Badges placeholder -->

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

## Features

- **Full Harvest API v2 coverage** - Access all Harvest functionality from the command line
- **Time tracking** - Create, edit, list, and delete time entries with flexible date parsing
- **Timer management** - Start, stop, restart, and toggle timers with TUI project/task picker
- **Weekly dashboard** - Visual summary of your time tracking week
- **Financial management** - Expenses with receipt uploads, invoices with full workflow, estimates
- **Resource management** - Projects, clients, tasks, and users
- **Reports** - Time, expense, uninvoiced, and budget reports with rate limit awareness
- **Approval workflows** - Submit, approve, and reject time entries
- **Bulk operations** - CSV import/export for time entries
- **External references** - Link time entries to JIRA, Asana, GitHub, and other tools
- **TUI interactive pickers** - Fuzzy-search project and task selection
- **Multi-account support** - Manage multiple Harvest accounts with aliases
- **Secure credential storage** - System keyring integration (macOS Keychain, Linux Secret Service,
  Windows Credential Manager)
- **Multiple output formats** - Table, JSON, or TSV for scripting

## Installation

### Homebrew (macOS/Linux)

```bash
brew install dedene/tap/harvestcli
```

### Using Go

```bash
go install github.com/dedene/harvest-cli/cmd/harvest@latest
```

### Binary Downloads

Download pre-built binaries from the [Releases](https://github.com/dedene/harvest-cli/releases)
page.

## Quick Start

### 1. Set up OAuth credentials

Register an OAuth application in your
[Harvest Developer Settings](https://id.getharvest.com/oauth2/access_tokens/new), then:

```bash
harvest auth setup <client_id> <client_secret>
```

### 2. Authenticate

```bash
harvest auth login
```

This opens your browser for OAuth authorization. For headless environments, use `--manual` mode.

Alternatively, use a Personal Access Token:

```bash
harvest auth login --pat
```

### 3. Start tracking time

```bash
# Start a timer (interactive project/task picker)
harvest timer start

# Log time directly
harvest time add -p "My Project" --task "Development" -h 2 -n "Worked on feature X"

# View your dashboard
harvest dashboard

# Stop running timer
harvest timer stop
```

## Commands

| Command      | Description                                                                     |
| ------------ | ------------------------------------------------------------------------------- |
| `auth`       | Authentication: login, logout, status, list, switch accounts                    |
| `config`     | Configuration: show, set, unset, path                                           |
| `time`       | Time entries: list, show, add, edit, remove, log                                |
| `timer`      | Timer control: status, start, stop, restart, toggle                             |
| `dashboard`  | Weekly time tracking summary                                                    |
| `projects`   | Projects: list, show, add, edit, remove                                         |
| `clients`    | Clients: list, show, add, edit, remove                                          |
| `tasks`      | Tasks: list, show, add, edit, remove                                            |
| `users`      | Users: list, show, me, add, edit, remove                                        |
| `expenses`   | Expenses: list, show, add, edit, remove (with receipt upload)                   |
| `invoices`   | Invoices: list, show, add, edit, remove, send, mark-sent/closed/draft, payments |
| `estimates`  | Estimates: list, show, add, edit, remove, send, mark-sent/accepted/declined     |
| `reports`    | Reports: time, expenses, uninvoiced, budget                                     |
| `approvals`  | Approvals: pending, submit, approve, reject                                     |
| `bulk`       | Bulk operations: export, import (CSV)                                           |
| `company`    | Show company information                                                        |
| `completion` | Generate shell completions (bash, zsh, fish)                                    |
| `version`    | Show version information                                                        |

## Configuration

Configuration is stored in `~/.config/harvest/` (or `$XDG_CONFIG_HOME/harvest/`).

### Config File

```bash
# View current config
harvest config show

# Set default account
harvest config set default_account user@example.com

# Set timezone
harvest config set default_timezone America/New_York

# Create account alias
harvest config set alias.work work@company.com
```

### Environment Variables

| Variable                | Description                    |
| ----------------------- | ------------------------------ |
| `HARVESTCLI_ACCOUNT`    | Default account email or alias |
| `HARVESTCLI_ACCOUNT_ID` | Harvest account ID override    |

### Global Flags

| Flag            | Description                       |
| --------------- | --------------------------------- |
| `-a, --account` | Account email or alias            |
| `--account-id`  | Harvest account ID override       |
| `-j, --json`    | Output as JSON                    |
| `--plain`       | Output as TSV (plain text)        |
| `-v, --verbose` | Verbose output                    |
| `--color`       | Color output: auto, always, never |

## Authentication

### OAuth (Recommended)

OAuth provides secure, token-based authentication with automatic refresh:

```bash
# Setup OAuth client (one-time)
harvest auth setup <client_id> <client_secret>

# Login (opens browser)
harvest auth login

# Login in headless/SSH environment
harvest auth login --manual
```

### Personal Access Token

For simpler setups or CI/CD, use a PAT from
[Harvest Developer Settings](https://id.getharvest.com/developers):

```bash
harvest auth login --pat
```

### Multi-Account Support

```bash
# List all authenticated accounts
harvest auth list

# Switch default account
harvest auth switch user@example.com

# Use specific account for a command
harvest -a work@company.com time list

# Create an alias
harvest config set alias.personal me@gmail.com
harvest -a personal dashboard
```

## Examples

### Time Tracking

```bash
# List today's entries
harvest time list -f today

# List this week's entries for a project
harvest time list -f "monday" -t "today" -p "Client Project"

# Quick time log with wizard
harvest time log

# Add time with external reference (JIRA)
harvest time add -p "Project" --task "Dev" -h 2 --external-ref-id "JIRA-123" --external-ref-service jira
```

### Timer

```bash
# Start timer (interactive picker)
harvest timer start

# Start timer for specific project/task
harvest timer start -p "My Project" --task "Meetings"

# Toggle (stop if running, restart last if not)
harvest timer toggle

# Check status
harvest timer
```

### Reports

```bash
# Time report by project
harvest reports time -f "2024-01-01" -t "2024-01-31" --by projects

# Expense report by category
harvest reports expenses -f "2024-01-01" -t "2024-01-31" --by categories

# Uninvoiced amounts
harvest reports uninvoiced -f "2024-01-01" -t "2024-01-31"

# Project budgets
harvest reports budget --active
```

### Bulk Operations

```bash
# Export time entries to CSV
harvest bulk export -f "2024-01-01" -t "2024-01-31" -o timesheet.csv

# Import time entries from CSV
harvest bulk import timesheet.csv

# Dry run (preview without creating)
harvest bulk import timesheet.csv --dry-run
```

### Invoices

```bash
# Create invoice
harvest invoices add -c "Client Name" --subject "January 2024"

# Send invoice
harvest invoices send 12345 -r "billing@client.com"

# Record payment
harvest invoices payments add 12345 --amount 1500.00
```

## Shell Completions

```bash
# Bash
harvest completion bash > /etc/bash_completion.d/harvest

# Zsh
harvest completion zsh > "${fpath[1]}/_harvest"

# Fish
harvest completion fish > ~/.config/fish/completions/harvest.fish
```

## Agent Skill

This CLI is available as an [open agent skill](https://skills.sh/) for AI assistants including [Claude Code](https://claude.ai/code), [OpenClaw](https://openclaw.ai/), Cursor, and GitHub Copilot:

```bash
npx skills add dedene/harvest-cli
```

## License

MIT License - see [LICENSE](LICENSE) for details.
