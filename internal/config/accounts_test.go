package config

import (
	"os"
	"testing"
)

func TestResolveAccountFromFlag(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv(EnvAccount, "")

	account, err := ResolveAccount("flag@example.com")
	if err != nil {
		t.Fatalf("ResolveAccount() error: %v", err)
	}
	if account != "flag@example.com" {
		t.Errorf("ResolveAccount() = %q, want flag@example.com", account)
	}
}

func TestResolveAccountFromEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv(EnvAccount, "env@example.com")

	account, err := ResolveAccount("")
	if err != nil {
		t.Fatalf("ResolveAccount() error: %v", err)
	}
	if account != "env@example.com" {
		t.Errorf("ResolveAccount() = %q, want env@example.com", account)
	}
}

func TestResolveAccountFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv(EnvAccount, "")

	cfg := &File{DefaultAccount: "config@example.com"}
	WriteConfig(cfg)

	account, err := ResolveAccount("")
	if err != nil {
		t.Fatalf("ResolveAccount() error: %v", err)
	}
	if account != "config@example.com" {
		t.Errorf("ResolveAccount() = %q, want config@example.com", account)
	}
}

func TestResolveAccountWithAlias(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv(EnvAccount, "")

	cfg := &File{
		AccountAliases: map[string]string{"work": "work@company.com"},
	}
	WriteConfig(cfg)

	account, err := ResolveAccount("work")
	if err != nil {
		t.Fatalf("ResolveAccount() error: %v", err)
	}
	if account != "work@company.com" {
		t.Errorf("ResolveAccount() = %q, want work@company.com", account)
	}
}

func TestResolveAccountNoAccount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Setenv(EnvAccount, "")

	_, err := ResolveAccount("")
	if err == nil {
		t.Error("ResolveAccount() should error when no account available")
	}
}

func TestResolveClientForAccount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfg := &File{
		AccountClients: map[string]string{"specific@example.com": "specific-client"},
		ClientDomains:  map[string]string{"company.com": "company-client"},
	}
	WriteConfig(cfg)

	tests := []struct {
		email    string
		override string
		want     string
	}{
		{"any@any.com", "override-client", "override-client"},
		{"specific@example.com", "", "specific-client"},
		{"user@company.com", "", "company-client"},
		{"user@unknown.com", "", DefaultClientName},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got, err := ResolveClientForAccount(tt.email, tt.override)
			if err != nil {
				t.Fatalf("ResolveClientForAccount() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ResolveClientForAccount(%q, %q) = %q, want %q", tt.email, tt.override, got, tt.want)
			}
		})
	}
}

func TestSetDeleteAccountAlias(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set alias
	if err := SetAccountAlias("test", "test@example.com"); err != nil {
		t.Fatalf("SetAccountAlias() error: %v", err)
	}

	aliases := ListAccountAliases()
	if aliases["test"] != "test@example.com" {
		t.Errorf("alias not set correctly")
	}

	// Delete alias
	if err := DeleteAccountAlias("test"); err != nil {
		t.Fatalf("DeleteAccountAlias() error: %v", err)
	}

	aliases = ListAccountAliases()
	if _, ok := aliases["test"]; ok {
		t.Error("alias should be deleted")
	}

	// Delete nonexistent
	if err := DeleteAccountAlias("nonexistent"); err == nil {
		t.Error("DeleteAccountAlias(nonexistent) should error")
	}
}

func TestSetDefaultAccount(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	os.Unsetenv(EnvAccount)

	if err := SetDefaultAccount("default@example.com"); err != nil {
		t.Fatalf("SetDefaultAccount() error: %v", err)
	}

	account, err := ResolveAccount("")
	if err != nil {
		t.Fatalf("ResolveAccount() error: %v", err)
	}
	if account != "default@example.com" {
		t.Errorf("default account = %q, want default@example.com", account)
	}
}

func TestDomainFromEmail(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{"user@example.com", "example.com"},
		{"user@Sub.Domain.COM", "sub.domain.com"},
		{"nodomain", ""},
		{"trailing@", ""},
		{"@nodomain", "nodomain"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := DomainFromEmail(tt.email)
			if got != tt.want {
				t.Errorf("DomainFromEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}

func TestNormalizeDomain(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"EXAMPLE.COM", "example.com"},
		{"  example.com  ", "example.com"},
		{"Example.Com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			got := NormalizeDomain(tt.domain)
			if got != tt.want {
				t.Errorf("NormalizeDomain(%q) = %q, want %q", tt.domain, got, tt.want)
			}
		})
	}
}

func TestSetClientDomain(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SetClientDomain("COMPANY.COM", "company-client"); err != nil {
		t.Fatalf("SetClientDomain() error: %v", err)
	}

	cfg, _ := ReadConfig()
	if cfg.ClientDomains["company.com"] != "company-client" {
		t.Errorf("domain mapping not set correctly")
	}
}

func TestSetAccountClient(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SetAccountClient("user@example.com", "user-client"); err != nil {
		t.Fatalf("SetAccountClient() error: %v", err)
	}

	cfg, _ := ReadConfig()
	if cfg.AccountClients["user@example.com"] != "user-client" {
		t.Errorf("account client mapping not set correctly")
	}
}

func TestAliasValidation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SetAccountAlias("", "test@example.com"); err == nil {
		t.Error("SetAccountAlias with empty alias should error")
	}
	if err := SetAccountAlias("test", ""); err == nil {
		t.Error("SetAccountAlias with empty email should error")
	}
	if err := DeleteAccountAlias(""); err == nil {
		t.Error("DeleteAccountAlias with empty alias should error")
	}
}
