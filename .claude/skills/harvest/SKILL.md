---
name: harvest-cli
description: >
  Track time and manage Harvest projects via the harvest CLI. Use when the user wants to
  log time entries, start/stop timers, view weekly dashboards, manage invoices, run reports,
  or handle expenses. Triggered by mentions of Harvest, time tracking, timesheets, billable hours,
  or timesheet workflows.
license: MIT
homepage: https://github.com/dedene/harvest-cli
metadata:
  author: dedene
  version: "1.1.0"
  openclaw:
    primaryEnv: HARVESTCLI_ACCOUNT
    requires:
      env:
        - HARVESTCLI_ACCOUNT
        - HARVESTCLI_ACCOUNT_ID
      bins:
        - harvest
    install:
      - kind: brew
        tap: dedene/tap
        formula: harvestcli
        bins: [harvest]
      - kind: go
        package: github.com/dedene/harvest-cli/cmd/harvest
        bins: [harvest]
---

# harvest-cli

CLI for [Harvest](https://www.getharvest.com/) time tracking and invoicing.

## Quick Start

```bash
# Verify auth
harvest auth status

# View weekly dashboard
harvest dashboard

# Start a timer
harvest timer start

# List today's time entries
harvest time list -f today
```

## Authentication

Requires OAuth setup or Personal Access Token from [Harvest Developer Settings](https://id.getharvest.com/developers).

```bash
# Check auth status
harvest auth status

# List accounts
harvest auth list
```

If not authenticated, user must run `harvest auth login` interactively (browser OAuth). Do not attempt auth setup on behalf of the user.

## Core Rules

1. **Always use `--json`** when parsing output programmatically
2. **Read before write** - check current timer/entries before modifying
3. **Timer is singleton** - only one timer runs at a time; starting new stops old
4. **Multi-account** - use `-a email` or `-a alias` for specific accounts

## Output Formats

| Flag | Format | Use case |
|------|--------|----------|
| (default) | Table | User-facing display |
| `--json` | JSON | Agent parsing, scripting |
| `--plain` | TSV | Pipe to awk/cut |

## Workflows

### Timer Control

```bash
# Check running timer
harvest timer

# Start timer (interactive project/task picker)
harvest timer start

# Start for specific project/task
harvest timer start -p "My Project" --task "Development"

# Stop running timer
harvest timer stop

# Toggle (stop if running, restart last if not)
harvest timer toggle

# Restart last timer
harvest timer restart
```

### Time Entries

```bash
# List today's entries
harvest time list -f today

# List this week
harvest time list -f monday -t today

# List by project
harvest time list -p "Client Project"

# Add time entry
harvest time add -p "Project" --task "Development" -h 2 -n "Worked on feature"

# Add with external reference (JIRA, GitHub)
harvest time add -p "Project" --task "Dev" -h 1.5 \
  --external-ref-id "JIRA-123" --external-ref-service jira

# Edit entry
harvest time edit <entry_id> -h 3 -n "Updated notes"

# Delete entry
harvest time remove <entry_id>

# Quick log with wizard
harvest time log
```

### Dashboard

```bash
# Weekly summary
harvest dashboard

# JSON for parsing
harvest dashboard --json
```

### Reports

```bash
# Time report by project
harvest reports time -f "2024-01-01" -t "2024-01-31" --by projects

# Time report by task
harvest reports time -f "2024-01-01" -t "2024-01-31" --by tasks

# Expense report
harvest reports expenses -f "2024-01-01" -t "2024-01-31" --by categories

# Uninvoiced amounts
harvest reports uninvoiced -f "2024-01-01" -t "2024-01-31"

# Project budgets
harvest reports budget --active
```

### Projects & Clients

```bash
# List projects
harvest projects list
harvest projects list --active

# Get project details
harvest projects show <id>

# List clients
harvest clients list

# Get client details
harvest clients show <id>
```

### Invoices

```bash
# List invoices
harvest invoices list
harvest invoices list --status draft

# Show invoice
harvest invoices show <id>

# Create invoice
harvest invoices add -c "Client Name" --subject "January 2024"

# Send invoice
harvest invoices send <id> -r "billing@client.com"

# Record payment
harvest invoices payments add <id> --amount 1500.00

# Mark status
harvest invoices mark-sent <id>
harvest invoices mark-closed <id>
```

### Expenses

```bash
# List expenses
harvest expenses list

# Add expense with receipt
harvest expenses add -p "Project" --category "Travel" \
  --amount 50.00 --receipt /path/to/receipt.pdf

# Show expense
harvest expenses show <id>
```

### Bulk Operations

```bash
# Export to CSV
harvest bulk export -f "2024-01-01" -t "2024-01-31" -o timesheet.csv

# Import from CSV
harvest bulk import timesheet.csv

# Dry run (preview)
harvest bulk import timesheet.csv --dry-run
```

### Multi-Account

```bash
# List accounts
harvest auth list

# Switch default account
harvest auth switch user@example.com

# Use specific account for command
harvest -a work@company.com time list

# Create alias
harvest config set alias.work work@company.com
harvest -a work dashboard
```

## Scripting Examples

```bash
# Get running timer project
harvest timer --json | jq -r '.project.name'

# Get today's total hours
harvest time list -f today --json | jq '[.[].hours] | add'

# Find project ID by name
harvest projects list --json | jq -r '.[] | select(.name == "My Project") | .id'

# Stop timer if running
if harvest timer --json | jq -e '.is_running' > /dev/null; then
  harvest timer stop
fi
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `HARVESTCLI_ACCOUNT` | Default account email or alias |
| `HARVESTCLI_ACCOUNT_ID` | Harvest account ID override |

## Global Flags

| Flag | Description |
|------|-------------|
| `-a, --account` | Account email or alias |
| `--account-id` | Harvest account ID override |
| `-j, --json` | Output as JSON |
| `--plain` | Output as TSV |
| `-v, --verbose` | Verbose output |
| `--color` | Color: auto, always, never |

## Command Reference

| Command | Description |
|---------|-------------|
| `auth` | login, logout, status, list, switch |
| `config` | show, set, unset, path |
| `time` | list, show, add, edit, remove, log |
| `timer` | status, start, stop, restart, toggle |
| `dashboard` | Weekly summary |
| `projects` | list, show, add, edit, remove |
| `clients` | list, show, add, edit, remove |
| `tasks` | list, show, add, edit, remove |
| `users` | list, show, me, add, edit, remove |
| `expenses` | list, show, add, edit, remove |
| `invoices` | list, show, add, edit, remove, send, payments |
| `estimates` | list, show, add, edit, remove, send |
| `reports` | time, expenses, uninvoiced, budget |
| `approvals` | pending, submit, approve, reject |
| `bulk` | export, import |
| `company` | Show company info |

## Guidelines

- Never expose OAuth tokens or credentials
- Confirm destructive operations (remove, bulk import) with user first
- Timer commands affect real billable time - confirm before stopping/modifying
- Rate limits handled automatically with backoff


## Installation

```bash
brew install dedene/tap/harvestcli
```
