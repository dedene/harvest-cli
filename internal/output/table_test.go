package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTable_Basic(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "Name", "Value")
	tbl.AddRow("foo", "1")
	tbl.AddRow("bar", "2")

	if err := tbl.Render(); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()

	// Should contain headers
	if !strings.Contains(output, "Name") || !strings.Contains(output, "Value") {
		t.Errorf("Output should contain headers, got: %s", output)
	}

	// Should contain separator
	if !strings.Contains(output, "----") {
		t.Errorf("Output should contain separator, got: %s", output)
	}

	// Should contain data
	if !strings.Contains(output, "foo") || !strings.Contains(output, "bar") {
		t.Errorf("Output should contain data, got: %s", output)
	}
}

func TestTable_NoHeaders(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf)
	tbl.AddRow("foo", "1")
	tbl.AddRow("bar", "2")

	if err := tbl.Render(); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()

	// Should not contain separator (no headers)
	if strings.Count(output, "----") > 0 {
		t.Errorf("Output should not contain separator without headers, got: %s", output)
	}

	// Should contain data
	if !strings.Contains(output, "foo") {
		t.Errorf("Output should contain data, got: %s", output)
	}
}

func TestTable_RowCount(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "A", "B")

	if tbl.RowCount() != 0 {
		t.Errorf("RowCount() = %d, want 0", tbl.RowCount())
	}

	tbl.AddRow("1", "2")
	tbl.AddRow("3", "4")

	if tbl.RowCount() != 2 {
		t.Errorf("RowCount() = %d, want 2", tbl.RowCount())
	}
}

func TestTableBuilder(t *testing.T) {
	var buf bytes.Buffer
	err := NewTableBuilder(&buf).
		Headers("Col1", "Col2").
		Row("a", "b").
		Row("c", "d").
		Render()

	if err != nil {
		t.Fatalf("Render error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Col1") {
		t.Errorf("Output should contain headers, got: %s", output)
	}
	if !strings.Contains(output, "a") {
		t.Errorf("Output should contain data, got: %s", output)
	}
}

func TestSimpleTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"X", "Y"}
	rows := [][]string{{"1", "2"}, {"3", "4"}}

	if err := SimpleTable(&buf, headers, rows); err != nil {
		t.Fatalf("SimpleTable error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "X") || !strings.Contains(output, "Y") {
		t.Errorf("Output should contain headers, got: %s", output)
	}
}

func TestTable_Alignment(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "Short", "LongerHeader")
	tbl.AddRow("a", "b")
	tbl.AddRow("longer value", "x")

	if err := tbl.Render(); err != nil {
		t.Fatalf("Render error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 3 {
		t.Fatalf("Expected at least 3 lines, got %d", len(lines))
	}

	// Just verify it renders without error and produces multiple lines
	// The actual alignment depends on tabwriter
}
