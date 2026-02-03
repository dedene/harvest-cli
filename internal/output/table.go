package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Table is a simple table renderer using tabwriter.
type Table struct {
	w       *tabwriter.Writer
	headers []string
	rows    [][]string
}

// NewTable creates a new table with the given headers.
func NewTable(w io.Writer, headers ...string) *Table {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	return &Table{
		w:       tw,
		headers: headers,
		rows:    make([][]string, 0),
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cells ...string) {
	t.rows = append(t.rows, cells)
}

// Render writes the table to the underlying writer.
func (t *Table) Render() error {
	// Write headers
	if len(t.headers) > 0 {
		if _, err := fmt.Fprintln(t.w, strings.Join(t.headers, "\t")); err != nil {
			return err
		}
		// Write separator
		sep := make([]string, len(t.headers))
		for i, h := range t.headers {
			sep[i] = strings.Repeat("-", len(h))
		}
		if _, err := fmt.Fprintln(t.w, strings.Join(sep, "\t")); err != nil {
			return err
		}
	}

	// Write rows
	for _, row := range t.rows {
		if _, err := fmt.Fprintln(t.w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}

	return t.w.Flush()
}

// RowCount returns the number of rows added.
func (t *Table) RowCount() int {
	return len(t.rows)
}

// TableBuilder provides a fluent interface for building tables.
type TableBuilder struct {
	table   *Table
	headers []string
}

// NewTableBuilder creates a new table builder.
func NewTableBuilder(w io.Writer) *TableBuilder {
	return &TableBuilder{
		table: &Table{
			w:    tabwriter.NewWriter(w, 0, 0, 2, ' ', 0),
			rows: make([][]string, 0),
		},
	}
}

// Headers sets the table headers.
func (b *TableBuilder) Headers(headers ...string) *TableBuilder {
	b.table.headers = headers
	return b
}

// Row adds a row to the table.
func (b *TableBuilder) Row(cells ...string) *TableBuilder {
	b.table.AddRow(cells...)
	return b
}

// Build returns the configured table.
func (b *TableBuilder) Build() *Table {
	return b.table
}

// Render writes the table immediately.
func (b *TableBuilder) Render() error {
	return b.table.Render()
}

// SimpleTable is a convenience function for quick table output.
func SimpleTable(w io.Writer, headers []string, rows [][]string) error {
	t := NewTable(w, headers...)
	for _, row := range rows {
		t.AddRow(row...)
	}
	return t.Render()
}
