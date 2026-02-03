package dateparse

import (
	"testing"
	"time"
)

func TestParse_RelativeKeywords(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		input string
		want  time.Time
	}{
		{"today", today},
		{"Today", today},
		{"TODAY", today},
		{"yesterday", today.AddDate(0, 0, -1)},
		{"tomorrow", today.AddDate(0, 0, 1)},
		{"last week", today.AddDate(0, 0, -7)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_DaysAgo(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		input string
		want  time.Time
	}{
		{"1 day ago", today.AddDate(0, 0, -1)},
		{"2 days ago", today.AddDate(0, 0, -2)},
		{"7days ago", today.AddDate(0, 0, -7)},
		{"1 week ago", today.AddDate(0, 0, -7)},
		{"2 weeks ago", today.AddDate(0, 0, -14)},
		{"1 month ago", today.AddDate(0, -1, 0)},
		{"3 months ago", today.AddDate(0, -3, 0)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParse_ISO8601(t *testing.T) {
	tests := []struct {
		input string
		year  int
		month time.Month
		day   int
	}{
		{"2024-01-15", 2024, time.January, 15},
		{"2024-12-31", 2024, time.December, 31},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", tt.input, err)
			}
			if got.Year() != tt.year || got.Month() != tt.month || got.Day() != tt.day {
				t.Errorf("Parse(%q) = %v, want %d-%02d-%02d", tt.input, got, tt.year, tt.month, tt.day)
			}
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	invalid := []string{
		"not a date",
		"",
	}

	for _, input := range invalid {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("Parse(%q) should return error", input)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"1h", time.Hour},
		{"2h", 2 * time.Hour},
		{"1.5h", 90 * time.Minute},
		{"90m", 90 * time.Minute},
		{"30m", 30 * time.Minute},
		{"1h30m", 90 * time.Minute},
		{"2h15m", 2*time.Hour + 15*time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if err != nil {
				t.Fatalf("ParseDuration(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	invalid := []string{
		"not a duration",
		"",
		"abc",
	}

	for _, input := range invalid {
		t.Run(input, func(t *testing.T) {
			_, err := ParseDuration(input)
			if err == nil {
				t.Errorf("ParseDuration(%q) should return error", input)
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	date := time.Date(2024, time.January, 15, 10, 30, 0, 0, time.UTC)
	got := FormatDate(date)
	want := "2024-01-15"
	if got != want {
		t.Errorf("FormatDate() = %q, want %q", got, want)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{time.Hour, "1h"},
		{2 * time.Hour, "2h"},
		{90 * time.Minute, "1.5h"},
		{30 * time.Minute, "0.5h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.input)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		input  string
		hour   int
		minute int
	}{
		{"9:00", 9, 0},
		{"09:00", 9, 0},
		{"14:30", 14, 30},
		{"9:00am", 9, 0},
		{"9:00 am", 9, 0},
		{"9:00pm", 21, 0},
		{"12:00pm", 12, 0},
		{"12:00am", 0, 0},
		{"11:59pm", 23, 59},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			hour, minute, err := ParseTimeOfDay(tt.input)
			if err != nil {
				t.Fatalf("ParseTimeOfDay(%q) error: %v", tt.input, err)
			}
			if hour != tt.hour || minute != tt.minute {
				t.Errorf("ParseTimeOfDay(%q) = %d:%02d, want %d:%02d", tt.input, hour, minute, tt.hour, tt.minute)
			}
		})
	}
}

func TestParseTimeOfDay_Invalid(t *testing.T) {
	invalid := []string{
		"not a time",
		"25:00",
		"12:60",
		"",
	}

	for _, input := range invalid {
		t.Run(input, func(t *testing.T) {
			_, _, err := ParseTimeOfDay(input)
			if err == nil {
				t.Errorf("ParseTimeOfDay(%q) should return error", input)
			}
		})
	}
}

func TestParse_LastWeekday(t *testing.T) {
	// These tests are date-dependent but should not error
	weekdays := []string{
		"last monday",
		"last tuesday",
		"last wednesday",
		"last thursday",
		"last friday",
		"last saturday",
		"last sunday",
	}

	for _, input := range weekdays {
		t.Run(input, func(t *testing.T) {
			got, err := Parse(input)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", input, err)
			}
			// Just verify it's in the past
			if got.After(time.Now()) {
				t.Errorf("Parse(%q) = %v, should be in the past", input, got)
			}
		})
	}
}
