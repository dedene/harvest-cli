// Package output provides output formatting for CLI commands.
package output

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Mode represents the output format mode.
type Mode int

const (
	// ModeTable outputs formatted tables (default).
	ModeTable Mode = iota
	// ModeJSON outputs JSON.
	ModeJSON
	// ModePlain outputs tab-separated values.
	ModePlain
)

// String returns the string representation of the mode.
func (m Mode) String() string {
	switch m {
	case ModeJSON:
		return "json"
	case ModePlain:
		return "plain"
	default:
		return "table"
	}
}

type contextKey string

const modeKey contextKey = "output_mode"

// WithMode returns a context with the given output mode.
func WithMode(ctx context.Context, mode Mode) context.Context {
	return context.WithValue(ctx, modeKey, mode)
}

// GetMode returns the output mode from context, defaulting to ModeTable.
func GetMode(ctx context.Context) Mode {
	if mode, ok := ctx.Value(modeKey).(Mode); ok {
		return mode
	}
	return ModeTable
}

// ModeFromFlags returns the output mode based on command flags.
// JSON takes precedence over plain.
func ModeFromFlags(jsonFlag, plainFlag bool) Mode {
	if jsonFlag {
		return ModeJSON
	}
	if plainFlag {
		return ModePlain
	}
	return ModeTable
}

// WriteJSON writes v as indented JSON to w.
func WriteJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// WriteTSV writes rows as tab-separated values.
// If headers is non-empty, it's written as the first row.
func WriteTSV(w io.Writer, headers []string, rows [][]string) error {
	if len(headers) > 0 {
		if _, err := fmt.Fprintln(w, strings.Join(headers, "\t")); err != nil {
			return err
		}
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}

// Formatter provides a unified interface for outputting data.
type Formatter struct {
	Mode   Mode
	Writer io.Writer
}

// NewFormatter creates a formatter with the given mode.
func NewFormatter(w io.Writer, mode Mode) *Formatter {
	return &Formatter{Mode: mode, Writer: w}
}

// Output writes data in the appropriate format.
// For JSON mode, v is encoded directly.
// For table/plain modes, headers and rows are used.
func (f *Formatter) Output(v any, headers []string, rows [][]string) error {
	switch f.Mode {
	case ModeJSON:
		return WriteJSON(f.Writer, v)
	case ModePlain:
		return WriteTSV(f.Writer, headers, rows)
	default:
		t := NewTable(f.Writer, headers...)
		for _, row := range rows {
			t.AddRow(row...)
		}
		return t.Render()
	}
}
