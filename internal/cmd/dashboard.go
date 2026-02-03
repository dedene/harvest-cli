package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dedene/harvest-cli/internal/api"
	"github.com/dedene/harvest-cli/internal/output"
	"github.com/dedene/harvest-cli/internal/ui"
)

// DashboardCmd shows a weekly summary view.
type DashboardCmd struct {
	Week string `help:"Week to show (default: current)" short:"w"`
}

// Run executes the dashboard command.
func (c *DashboardCmd) Run(cli *CLI) error {
	ctx := context.Background()

	client, err := NewClientFromFlags(ctx, &cli.RootFlags)
	if err != nil {
		return err
	}

	// Get company settings for week_start_day
	company, err := client.GetCompany(ctx)
	if err != nil {
		return fmt.Errorf("get company: %w", err)
	}

	// Calculate week boundaries
	weekStart, weekEnd := c.calculateWeekBoundaries(company.WeekStartDay)

	// Fetch time entries for the week
	entries, err := client.ListAllTimeEntries(ctx, api.TimeEntryListOptions{
		From: weekStart.Format("2006-01-02"),
		To:   weekEnd.Format("2006-01-02"),
	})
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}

	// Check for running timer
	running, _ := client.GetRunningTimeEntry(ctx)

	// Get week target (company.WeeklyCapacity is in seconds, default 40h = 144000)
	weekTarget := float64(company.WeeklyCapacity) / 3600.0
	if weekTarget <= 0 {
		weekTarget = 40.0
	}

	// Build dashboard model
	dashboard := ui.NewDashboard(entries, running, weekStart, weekTarget)

	// Render output
	if cli.JSON {
		return c.outputJSON(dashboard)
	}

	fmt.Fprint(os.Stdout, dashboard.View())
	return nil
}

// calculateWeekBoundaries returns start (Monday by default) and end of week.
func (c *DashboardCmd) calculateWeekBoundaries(weekStartDay string) (time.Time, time.Time) {
	now := time.Now()

	// Parse --week flag if provided (format: 2024-01-15 or YYYY-Www)
	if c.Week != "" {
		if t, err := time.Parse("2006-01-02", c.Week); err == nil {
			now = t
		} else if strings.HasPrefix(c.Week, "20") && strings.Contains(c.Week, "W") {
			// ISO week format: 2024-W03
			var year, week int
			if _, err := fmt.Sscanf(c.Week, "%d-W%d", &year, &week); err == nil {
				// Find first day of ISO week
				jan1 := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
				daysToMonday := int(time.Monday - jan1.Weekday())
				if daysToMonday > 0 {
					daysToMonday -= 7
				}
				firstMonday := jan1.AddDate(0, 0, daysToMonday)
				now = firstMonday.AddDate(0, 0, (week-1)*7)
			}
		}
	}

	// Map Harvest week_start_day to Go weekday
	startWeekday := parseWeekStartDay(weekStartDay)

	// Find start of week
	daysBack := int(now.Weekday()) - int(startWeekday)
	if daysBack < 0 {
		daysBack += 7
	}

	weekStart := time.Date(now.Year(), now.Month(), now.Day()-daysBack, 0, 0, 0, 0, time.Local)
	weekEnd := weekStart.AddDate(0, 0, 6)

	return weekStart, weekEnd
}

func parseWeekStartDay(day string) time.Weekday {
	switch strings.ToLower(day) {
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
		return time.Monday
	}
}

func (c *DashboardCmd) outputJSON(d *ui.DashboardModel) error {
	data := map[string]any{
		"week_start":   d.WeekStart.Format("2006-01-02"),
		"today_hours":  d.TodayHours,
		"week_hours":   d.WeekHours,
		"week_target":  d.WeekTarget,
		"daily_hours":  d.DailyHours,
	}

	if d.Running != nil {
		data["running"] = map[string]any{
			"id":               d.Running.ID,
			"project":          d.Running.Project.Name,
			"task":             d.Running.Task.Name,
			"notes":            d.Running.Notes,
			"timer_started_at": d.Running.TimerStartedAt,
			"hours":            d.Running.Hours,
		}
	}

	return output.WriteJSON(os.Stdout, data)
}
