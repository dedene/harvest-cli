package output

import (
	"testing"
)

func TestIsColorEnabled(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{"always", true},
		{"never", false},
		// "auto" depends on environment, so we test it separately
	}

	for _, tt := range tests {
		got := IsColorEnabled(tt.mode)
		if got != tt.want {
			t.Errorf("IsColorEnabled(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestColors_Disabled(t *testing.T) {
	c := NewColors("never")

	if c.Enabled() {
		t.Error("Colors should be disabled with 'never' mode")
	}

	// When disabled, methods should return the original string unchanged
	tests := []struct {
		name string
		fn   func(string) string
		want string
	}{
		{"Success", c.Success, "test"},
		{"Error", c.Error, "test"},
		{"Warning", c.Warning, "test"},
		{"Dim", c.Dim, "test"},
		{"Bold", c.Bold, "test"},
		{"Cyan", c.Cyan, "test"},
		{"Magenta", c.Magenta, "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("test")
			if got != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, "test", got, tt.want)
			}
		})
	}
}

func TestColors_Enabled(t *testing.T) {
	c := NewColors("always")

	if !c.Enabled() {
		t.Error("Colors should be enabled with 'always' mode")
	}

	// When enabled, methods may return styled strings
	// We just verify they don't panic and return something
	methods := []func(string) string{
		c.Success,
		c.Error,
		c.Warning,
		c.Dim,
		c.Bold,
		c.Cyan,
		c.Magenta,
	}

	for _, fn := range methods {
		result := fn("test")
		if result == "" {
			t.Error("Method returned empty string")
		}
	}
}

func TestNewColors_Modes(t *testing.T) {
	// Test all mode strings work without panic
	modes := []string{"auto", "always", "never", ""}

	for _, mode := range modes {
		c := NewColors(mode)
		if c == nil {
			t.Errorf("NewColors(%q) returned nil", mode)
		}
	}
}
