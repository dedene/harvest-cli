// Package dateparse provides flexible date and time parsing utilities.
package dateparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/araddon/dateparse"
)

var (
	// Relative date patterns
	daysAgoRe     = regexp.MustCompile(`^(\d+)\s*days?\s*ago$`)
	weeksAgoRe    = regexp.MustCompile(`^(\d+)\s*weeks?\s*ago$`)
	monthsAgoRe   = regexp.MustCompile(`^(\d+)\s*months?\s*ago$`)
	lastWeekdayRe = regexp.MustCompile(`^last\s+(monday|tuesday|wednesday|thursday|friday|saturday|sunday)$`)

	// Time of day pattern
	timeOfDayRe = regexp.MustCompile(`^(\d{1,2}):(\d{2})(?:\s*(am|pm))?$`)

	// Duration patterns
	hoursMinutesRe = regexp.MustCompile(`^(\d+)h(\d+)m?$`)
	hoursOnlyRe    = regexp.MustCompile(`^(\d+(?:\.\d+)?)h$`)
	minutesOnlyRe  = regexp.MustCompile(`^(\d+)m$`)
)

// Parse parses a date string with flexible formats.
// Supports:
//   - "today", "yesterday", "tomorrow"
//   - "N days ago", "N weeks ago", "N months ago"
//   - "last week", "last monday", etc.
//   - ISO 8601: "2024-01-15"
//   - Common formats via dateparse library
func Parse(s string) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Handle relative keywords
	switch s {
	case "today":
		return today, nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	case "last week":
		return today.AddDate(0, 0, -7), nil
	case "this week":
		// Start of this week (Monday)
		weekday := int(today.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday
		}
		return today.AddDate(0, 0, -(weekday - 1)), nil
	}

	// Handle "N days/weeks/months ago"
	if m := daysAgoRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return today.AddDate(0, 0, -n), nil
	}
	if m := weeksAgoRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return today.AddDate(0, 0, -n*7), nil
	}
	if m := monthsAgoRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		return today.AddDate(0, -n, 0), nil
	}

	// Handle "last monday", "last tuesday", etc.
	if m := lastWeekdayRe.FindStringSubmatch(s); m != nil {
		targetDay := parseWeekday(m[1])
		return lastWeekday(today, targetDay), nil
	}

	// Fall back to dateparse library
	t, err := dateparse.ParseLocal(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot parse date %q: %w", s, err)
	}
	return t, nil
}

// ParseDuration parses duration strings.
// Supports:
//   - "1.5h", "2h" (hours)
//   - "1h30m" (hours and minutes)
//   - "90m" (minutes only)
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Try hours+minutes format: 1h30m
	if m := hoursMinutesRe.FindStringSubmatch(s); m != nil {
		hours, _ := strconv.Atoi(m[1])
		minutes, _ := strconv.Atoi(m[2])
		return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute, nil
	}

	// Try hours only: 1.5h or 2h
	if m := hoursOnlyRe.FindStringSubmatch(s); m != nil {
		hours, _ := strconv.ParseFloat(m[1], 64)
		return time.Duration(hours * float64(time.Hour)), nil
	}

	// Try minutes only: 90m
	if m := minutesOnlyRe.FindStringSubmatch(s); m != nil {
		minutes, _ := strconv.Atoi(m[1])
		return time.Duration(minutes) * time.Minute, nil
	}

	// Try standard Go duration
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("cannot parse duration %q", s)
	}
	return d, nil
}

// FormatDate formats a date for display (YYYY-MM-DD).
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDuration formats a duration as hours (e.g., "1.5h").
func FormatDuration(d time.Duration) string {
	hours := d.Hours()
	if hours == float64(int(hours)) {
		return fmt.Sprintf("%dh", int(hours))
	}
	return fmt.Sprintf("%.2gh", hours)
}

// ParseTimeOfDay parses time strings: "9:00", "9:00am", "14:30".
func ParseTimeOfDay(s string) (hour, minute int, err error) {
	s = strings.TrimSpace(strings.ToLower(s))

	m := timeOfDayRe.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, fmt.Errorf("cannot parse time %q", s)
	}

	hour, _ = strconv.Atoi(m[1])
	minute, _ = strconv.Atoi(m[2])
	ampm := m[3]

	// Handle 12-hour format
	if ampm == "pm" && hour < 12 {
		hour += 12
	} else if ampm == "am" && hour == 12 {
		hour = 0
	}

	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, fmt.Errorf("invalid time %q", s)
	}

	return hour, minute, nil
}

// parseWeekday converts a weekday name to time.Weekday.
func parseWeekday(name string) time.Weekday {
	switch strings.ToLower(name) {
	case "sunday":
		return time.Sunday
	case "monday":
		return time.Monday
	case "tuesday":
		return time.Tuesday
	case "wednesday":
		return time.Wednesday
	case "thursday":
		return time.Thursday
	case "friday":
		return time.Friday
	case "saturday":
		return time.Saturday
	default:
		return time.Sunday
	}
}

// lastWeekday returns the most recent occurrence of the given weekday before today.
func lastWeekday(today time.Time, target time.Weekday) time.Time {
	current := today.Weekday()
	daysBack := int(current) - int(target)
	if daysBack <= 0 {
		daysBack += 7
	}
	return today.AddDate(0, 0, -daysBack)
}
