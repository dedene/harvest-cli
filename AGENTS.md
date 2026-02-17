# Repository Guidelines

## Project Structure

- `cmd/harvest/`: CLI entrypoint
- `internal/`: implementation
  - `cmd/`: command routing (kong CLI framework)
  - `api/`: Harvest API client
  - `auth/`: OAuth2 + PAT + keyring + account management
  - `config/`: accounts + credentials + paths
  - `ui/`: Charmbracelet bubbletea TUI components
  - `output/`: table rendering
  - `dateparse/`: custom date parsing
  - `errfmt/`: error formatting
- `bin/`: build outputs

## Build, Test, and Development Commands

- `make build`: compile to `bin/harvest` (CGO enabled on macOS for Keychain)
- `make install`: install to system
- `make fmt` / `make lint` / `make test` / `make ci`: format, lint, test, full local gate
- `make tools`: install pinned dev tools into `.tools/`
- `make clean`: remove bin/ and .tools/

## Coding Style & Naming Conventions

- Formatting: `make fmt` (goimports local prefix `github.com/dedene/harvest-cli` + gofumpt)
- Output: keep stdout parseable (`--json`); send human hints/progress to stderr
- Linting: golangci-lint with project config
- TUI: use Charmbracelet ecosystem (bubbletea + bubbles)

## Testing Guidelines

- Unit tests: stdlib `testing` (files: `*_test.go` next to code)
- Coverage areas: config (accounts/config/credentials/paths), auth (keyring/oauth/pat/token/accounts), output (table)
- 10+ test files; comprehensive coverage

## Config & Secrets

- **OAuth2**: full OAuth flow for Harvest authentication
- **PAT**: Personal Access Token support as alternative
- **Keyring**: 99designs/keyring (macOS Keychain, Linux SecretService, Windows Credential Manager)
- **Multi-account**: switch between multiple Harvest accounts
- **Token caching**: automatic token refresh

## Key Features

- Time tracking, timers, weekly dashboards
- Invoices, reports, expenses
- Interactive TUI for time entry

## Commit & Pull Request Guidelines

- Conventional Commits: `feat|fix|refactor|build|ci|chore|docs|style|perf|test`
- Group related changes; avoid bundling unrelated refactors
- PR review: use `gh pr view` / `gh pr diff`; don't switch branches

## Security Tips

- Never commit OAuth credentials or PAT tokens
- Prefer OS keychain; CGO required on macOS for Keychain access
- Multi-account credentials are isolated per account
