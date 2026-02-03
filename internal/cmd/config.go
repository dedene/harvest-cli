package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dedene/harvest-cli/internal/config"
)

// ConfigCmd groups configuration subcommands.
type ConfigCmd struct {
	Show  ConfigShowCmd  `cmd:"" default:"1" help:"Show current configuration"`
	Set   ConfigSetCmd   `cmd:"" help:"Set a configuration value"`
	Unset ConfigUnsetCmd `cmd:"" help:"Remove a configuration value"`
	Path  ConfigPathCmd  `cmd:"" help:"Show configuration directory path"`
}

// ConfigShowCmd shows current configuration.
type ConfigShowCmd struct{}

func (c *ConfigShowCmd) Run(cli *CLI) error {
	cfg, err := config.ReadConfig()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	if cli.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	}

	// Human-readable output
	fmt.Fprintf(os.Stdout, "Config file: %s\n\n", config.ConfigPath())

	if cfg.DefaultAccount != "" {
		fmt.Fprintf(os.Stdout, "default_account:   %s\n", cfg.DefaultAccount)
	}
	if cfg.DefaultTimezone != "" {
		fmt.Fprintf(os.Stdout, "default_timezone:  %s\n", cfg.DefaultTimezone)
	}
	if cfg.WeekStart != "" {
		fmt.Fprintf(os.Stdout, "week_start:        %s\n", cfg.WeekStart)
	}
	if cfg.Color != "" {
		fmt.Fprintf(os.Stdout, "color:             %s\n", cfg.Color)
	}
	if cfg.KeyringBackend != "" {
		fmt.Fprintf(os.Stdout, "keyring_backend:   %s\n", cfg.KeyringBackend)
	}
	if cfg.ContactEmail != "" {
		fmt.Fprintf(os.Stdout, "contact_email:     %s\n", cfg.ContactEmail)
	}

	if len(cfg.AccountAliases) > 0 {
		fmt.Fprintln(os.Stdout, "\nAccount aliases:")
		for alias, email := range cfg.AccountAliases {
			fmt.Fprintf(os.Stdout, "  %s -> %s\n", alias, email)
		}
	}

	if len(cfg.AccountClients) > 0 {
		fmt.Fprintln(os.Stdout, "\nAccount clients:")
		for email, client := range cfg.AccountClients {
			fmt.Fprintf(os.Stdout, "  %s -> %s\n", email, client)
		}
	}

	if len(cfg.ClientDomains) > 0 {
		fmt.Fprintln(os.Stdout, "\nClient domains:")
		for domain, client := range cfg.ClientDomains {
			fmt.Fprintf(os.Stdout, "  %s -> %s\n", domain, client)
		}
	}

	return nil
}

// ConfigSetCmd sets a configuration value.
type ConfigSetCmd struct {
	Key   string `arg:"" help:"Configuration key"`
	Value string `arg:"" help:"Configuration value"`
}

// Allowed configuration keys.
var allowedConfigKeys = map[string]bool{
	"default_account":  true,
	"default_timezone": true,
	"week_start":       true,
	"color":            true,
	"keyring_backend":  true,
	"contact_email":    true,
}

func (c *ConfigSetCmd) Run() error {
	key := strings.ToLower(strings.TrimSpace(c.Key))

	// Handle aliases specially
	if strings.HasPrefix(key, "alias.") {
		alias := strings.TrimPrefix(key, "alias.")
		return config.SetAccountAlias(alias, c.Value)
	}

	// Handle client domains
	if strings.HasPrefix(key, "domain.") {
		domain := strings.TrimPrefix(key, "domain.")
		return config.SetClientDomain(domain, c.Value)
	}

	// Handle account clients
	if strings.HasPrefix(key, "client.") {
		email := strings.TrimPrefix(key, "client.")
		return config.SetAccountClient(email, c.Value)
	}

	if !allowedConfigKeys[key] {
		return fmt.Errorf("unknown config key: %q\nAllowed keys: %s",
			key, strings.Join(sortedKeys(allowedConfigKeys), ", "))
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	switch key {
	case "default_account":
		cfg.DefaultAccount = c.Value
	case "default_timezone":
		cfg.DefaultTimezone = c.Value
	case "week_start":
		cfg.WeekStart = c.Value
	case "color":
		cfg.Color = c.Value
	case "keyring_backend":
		cfg.KeyringBackend = c.Value
	case "contact_email":
		cfg.ContactEmail = c.Value
	}

	if err := config.WriteConfig(cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Set %s = %s\n", key, c.Value)

	return nil
}

// ConfigUnsetCmd removes a configuration value.
type ConfigUnsetCmd struct {
	Key string `arg:"" help:"Configuration key to remove"`
}

func (c *ConfigUnsetCmd) Run() error {
	key := strings.ToLower(strings.TrimSpace(c.Key))

	// Handle aliases specially
	if strings.HasPrefix(key, "alias.") {
		alias := strings.TrimPrefix(key, "alias.")
		if err := config.DeleteAccountAlias(alias); err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Removed alias %s\n", alias)
		return nil
	}

	if !allowedConfigKeys[key] {
		return fmt.Errorf("unknown config key: %q", key)
	}

	cfg, err := config.ReadConfig()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	switch key {
	case "default_account":
		cfg.DefaultAccount = ""
	case "default_timezone":
		cfg.DefaultTimezone = ""
	case "week_start":
		cfg.WeekStart = ""
	case "color":
		cfg.Color = ""
	case "keyring_backend":
		cfg.KeyringBackend = ""
	case "contact_email":
		cfg.ContactEmail = ""
	}

	if err := config.WriteConfig(cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Unset %s\n", key)

	return nil
}

// ConfigPathCmd shows configuration paths.
type ConfigPathCmd struct{}

func (c *ConfigPathCmd) Run() error {
	dir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("resolve config dir: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Config dir:  %s\n", dir)
	fmt.Fprintf(os.Stdout, "Config file: %s\n", config.ConfigPath())
	fmt.Fprintf(os.Stdout, "Clients dir: %s\n", config.ClientsDir())
	fmt.Fprintf(os.Stdout, "Keyring dir: %s\n", config.KeyringDir())

	return nil
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
