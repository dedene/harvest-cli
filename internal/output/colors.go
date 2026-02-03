package output

import (
	"os"

	"github.com/muesli/termenv"
	"golang.org/x/term"
)

// Colors provides terminal color support.
type Colors struct {
	output  *termenv.Output
	enabled bool
}

// NewColors creates a Colors instance based on the mode setting.
// mode: "auto" (detect), "always" (force), "never" (disable)
func NewColors(mode string) *Colors {
	enabled := IsColorEnabled(mode)

	var profile termenv.Profile
	if enabled {
		profile = termenv.NewOutput(os.Stdout).Profile
	} else {
		profile = termenv.Ascii
	}

	output := termenv.NewOutput(os.Stdout, termenv.WithProfile(profile))

	return &Colors{
		output:  output,
		enabled: enabled,
	}
}

// IsColorEnabled determines if color output should be enabled.
func IsColorEnabled(mode string) bool {
	switch mode {
	case "always":
		return true
	case "never":
		return false
	default: // "auto"
		// Check if stdout is a terminal
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			return false
		}
		// Check NO_COLOR env var
		if _, ok := os.LookupEnv("NO_COLOR"); ok {
			return false
		}
		// Check TERM=dumb
		if os.Getenv("TERM") == "dumb" {
			return false
		}
		return true
	}
}

// Enabled returns whether colors are enabled.
func (c *Colors) Enabled() bool {
	return c.enabled
}

// Success returns the string styled as a success message (green).
func (c *Colors) Success(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Foreground(c.output.Color("2")).String()
}

// Error returns the string styled as an error message (red).
func (c *Colors) Error(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Foreground(c.output.Color("1")).String()
}

// Warning returns the string styled as a warning message (yellow).
func (c *Colors) Warning(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Foreground(c.output.Color("3")).String()
}

// Dim returns the string in a dimmed style.
func (c *Colors) Dim(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Faint().String()
}

// Bold returns the string in bold.
func (c *Colors) Bold(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Bold().String()
}

// Cyan returns the string in cyan.
func (c *Colors) Cyan(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Foreground(c.output.Color("6")).String()
}

// Magenta returns the string in magenta.
func (c *Colors) Magenta(s string) string {
	if !c.enabled {
		return s
	}
	return c.output.String(s).Foreground(c.output.Color("5")).String()
}
