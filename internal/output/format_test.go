package output

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMode_String(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeTable, "table"},
		{ModeJSON, "json"},
		{ModePlain, "plain"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("Mode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestModeFromFlags(t *testing.T) {
	tests := []struct {
		json  bool
		plain bool
		want  Mode
	}{
		{false, false, ModeTable},
		{true, false, ModeJSON},
		{false, true, ModePlain},
		{true, true, ModeJSON}, // JSON takes precedence
	}

	for _, tt := range tests {
		got := ModeFromFlags(tt.json, tt.plain)
		if got != tt.want {
			t.Errorf("ModeFromFlags(%v, %v) = %v, want %v", tt.json, tt.plain, got, tt.want)
		}
	}
}

func TestContextMode(t *testing.T) {
	ctx := context.Background()

	// Default should be ModeTable
	if got := GetMode(ctx); got != ModeTable {
		t.Errorf("GetMode(empty ctx) = %v, want ModeTable", got)
	}

	// Set and get JSON mode
	ctx = WithMode(ctx, ModeJSON)
	if got := GetMode(ctx); got != ModeJSON {
		t.Errorf("GetMode(ctx with JSON) = %v, want ModeJSON", got)
	}

	// Set and get Plain mode
	ctx = WithMode(ctx, ModePlain)
	if got := GetMode(ctx); got != ModePlain {
		t.Errorf("GetMode(ctx with Plain) = %v, want ModePlain", got)
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{
		"name":  "test",
		"count": 42,
	}

	err := WriteJSON(&buf, data)
	if err != nil {
		t.Fatalf("WriteJSON error: %v", err)
	}

	// Verify it's valid JSON
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if decoded["name"] != "test" {
		t.Errorf("name = %v, want test", decoded["name"])
	}

	// Verify indentation
	if !strings.Contains(buf.String(), "  ") {
		t.Error("Output should be indented")
	}
}

func TestWriteTSV(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"foo", "1"},
		{"bar", "2"},
	}

	err := WriteTSV(&buf, headers, rows)
	if err != nil {
		t.Fatalf("WriteTSV error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	if lines[0] != "Name\tValue" {
		t.Errorf("Header line = %q, want %q", lines[0], "Name\tValue")
	}
	if lines[1] != "foo\t1" {
		t.Errorf("Row 1 = %q, want %q", lines[1], "foo\t1")
	}
}

func TestWriteTSV_NoHeaders(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"foo", "1"},
		{"bar", "2"},
	}

	err := WriteTSV(&buf, nil, rows)
	if err != nil {
		t.Fatalf("WriteTSV error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
}

func TestFormatter_Output(t *testing.T) {
	type testData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	data := testData{Name: "test", Age: 30}
	headers := []string{"Name", "Age"}
	rows := [][]string{{"test", "30"}}

	tests := []struct {
		name     string
		mode     Mode
		contains string
	}{
		{"json mode", ModeJSON, `"name": "test"`},
		{"plain mode", ModePlain, "test\t30"},
		{"table mode", ModeTable, "Name"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewFormatter(&buf, tt.mode)
			err := f.Output(data, headers, rows)
			if err != nil {
				t.Fatalf("Output error: %v", err)
			}

			if !strings.Contains(buf.String(), tt.contains) {
				t.Errorf("Output = %q, should contain %q", buf.String(), tt.contains)
			}
		})
	}
}
