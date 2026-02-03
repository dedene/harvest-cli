package config

import (
	"fmt"
	"os"
	"strings"
)

// EnvAccount is the environment variable for specifying the account.
const EnvAccount = "HARVEST_ACCOUNT"

// ResolveAccount determines which account to use.
// Priority: flag > env > config default.
func ResolveAccount(flagAccount string) (string, error) {
	// 1. Flag takes precedence
	if flagAccount != "" {
		return resolveAlias(flagAccount)
	}

	// 2. Environment variable
	if env := os.Getenv(EnvAccount); env != "" {
		return resolveAlias(env)
	}

	// 3. Config default
	cfg, err := ReadConfig()
	if err != nil {
		return "", fmt.Errorf("reading config: %w", err)
	}
	if cfg.DefaultAccount != "" {
		return resolveAlias(cfg.DefaultAccount)
	}

	return "", fmt.Errorf("no account specified: use --account flag, %s env, or set default_account in config", EnvAccount)
}

// resolveAlias looks up an alias and returns the actual email.
// If not an alias, returns the input unchanged.
func resolveAlias(account string) (string, error) {
	cfg, err := ReadConfig()
	if err != nil {
		return "", err
	}
	if cfg.AccountAliases != nil {
		if email, ok := cfg.AccountAliases[account]; ok {
			return email, nil
		}
	}
	return account, nil
}

// ResolveClientForAccount determines which OAuth client to use for an account.
// Priority: override > account mapping > domain mapping > default.
func ResolveClientForAccount(email, override string) (string, error) {
	if override != "" {
		return override, nil
	}

	cfg, err := ReadConfig()
	if err != nil {
		return DefaultClientName, nil
	}

	// Check account-specific mapping
	if cfg.AccountClients != nil {
		if client, ok := cfg.AccountClients[email]; ok {
			return client, nil
		}
	}

	// Check domain mapping
	domain := DomainFromEmail(email)
	if domain != "" && cfg.ClientDomains != nil {
		if client, ok := cfg.ClientDomains[domain]; ok {
			return client, nil
		}
	}

	return DefaultClientName, nil
}

// SetAccountAlias creates or updates an account alias.
func SetAccountAlias(alias, email string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}

	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	cfg.initMaps()
	cfg.AccountAliases[alias] = email
	return WriteConfig(cfg)
}

// DeleteAccountAlias removes an account alias.
func DeleteAccountAlias(alias string) error {
	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}

	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	if cfg.AccountAliases == nil {
		return fmt.Errorf("alias %q not found", alias)
	}
	if _, ok := cfg.AccountAliases[alias]; !ok {
		return fmt.Errorf("alias %q not found", alias)
	}
	delete(cfg.AccountAliases, alias)
	return WriteConfig(cfg)
}

// ListAccountAliases returns all configured aliases.
func ListAccountAliases() map[string]string {
	cfg, err := ReadConfig()
	if err != nil {
		return nil
	}
	if cfg.AccountAliases == nil {
		return make(map[string]string)
	}
	// Return a copy
	result := make(map[string]string, len(cfg.AccountAliases))
	for k, v := range cfg.AccountAliases {
		result[k] = v
	}
	return result
}

// SetDefaultAccount sets the default account in config.
func SetDefaultAccount(account string) error {
	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	cfg.DefaultAccount = account
	return WriteConfig(cfg)
}

// DomainFromEmail extracts the domain from an email address.
func DomainFromEmail(email string) string {
	idx := strings.LastIndex(email, "@")
	if idx < 0 || idx == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[idx+1:])
}

// NormalizeDomain normalizes a domain for comparison.
func NormalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

// SetClientDomain maps a domain to a specific OAuth client.
func SetClientDomain(domain, client string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if client == "" {
		return fmt.Errorf("client cannot be empty")
	}

	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	cfg.initMaps()
	cfg.ClientDomains[NormalizeDomain(domain)] = client
	return WriteConfig(cfg)
}

// SetAccountClient maps an account to a specific OAuth client.
func SetAccountClient(email, client string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if client == "" {
		return fmt.Errorf("client cannot be empty")
	}

	cfg, err := ReadConfig()
	if err != nil {
		return err
	}
	cfg.initMaps()
	cfg.AccountClients[email] = client
	return WriteConfig(cfg)
}
