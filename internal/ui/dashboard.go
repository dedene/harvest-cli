package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dedene/harvest-cli/internal/api"
)

// DashboardModel holds dashboard state and data.
type DashboardModel struct {
	Running    *api.TimeEntry
	TodayHours float64
	WeekHours  float64
	WeekTarget float64
	DailyHours map[string]float64 // key: "Mon", "Tue", etc.
	WeekStart  time.Time
}

// NewDashboard creates a dashboard model from entries.
func NewDashboard(entries []api.TimeEntry, running *api.TimeEntry, weekStart time.Time, weekTarget float64) *DashboardModel {
	m := &DashboardModel{
		Running:    running,
		WeekTarget: weekTarget,
		DailyHours: make(map[string]float64),
		WeekStart:  weekStart,
	}

	today := time.Now().Format("2006-01-02")

	// Initialize days
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	for _, d := range days {
		m.DailyHours[d] = 0
	}

	// Sum hours by day
	for _, e := range entries {
		m.WeekHours += e.Hours
		if e.SpentDate == today {
			m.TodayHours += e.Hours
		}

		// Parse date to get weekday
		if t, err := time.Parse("2006-01-02", e.SpentDate); err == nil {
			day := t.Weekday().String()[:3]
			m.DailyHours[day] += e.Hours
		}
	}

	return m
}

// View renders the dashboard as a string.
func (m *DashboardModel) View() string {
	var b strings.Builder

	// Header
	weekLabel := m.WeekStart.Format("Jan 2, 2006")
	header := TitleStyle.Render(fmt.Sprintf("harvest - Week of %s", weekLabel))
	b.WriteString(header)
	b.WriteString("\n\n")

	// Running timer
	if m.Running != nil {
		b.WriteString(m.renderRunningTimer())
		b.WriteString("\n\n")
	}

	// Today + Week summary
	b.WriteString(m.renderSummary())
	b.WriteString("\n\n")

	// Daily breakdown
	b.WriteString(m.renderDailyBreakdown())
	b.WriteString("\n")

	return b.String()
}

func (m *DashboardModel) renderRunningTimer() string {
	var b strings.Builder

	project := m.Running.Project.Name
	task := m.Running.Task.Name

	// Play symbol
	playIcon := SuccessStyle.Render("\u25b6")
	b.WriteString(playIcon)
	b.WriteString(" Running: ")
	b.WriteString(HighlightStyle.Render(project))
	b.WriteString(" - ")
	b.WriteString(task)
	b.WriteString("\n")

	// Elapsed time
	if m.Running.TimerStartedAt != nil {
		elapsed := time.Since(*m.Running.TimerStartedAt)
		startedStr := m.Running.TimerStartedAt.Local().Format("3:04 PM")
		elapsedStr := formatDuration(elapsed)

		b.WriteString("  Started ")
		b.WriteString(DimStyle.Render(startedStr))
		b.WriteString(" (")
		b.WriteString(SelectedStyle.Render(elapsedStr))
		b.WriteString(" elapsed)")
	}

	return b.String()
}

func (m *DashboardModel) renderSummary() string {
	var b strings.Builder

	// Today
	todayStr := fmt.Sprintf("%.1fh", m.TodayHours)
	b.WriteString("Today: ")
	b.WriteString(HighlightStyle.Render(todayStr))
	b.WriteString("\n")

	// Week progress
	weekStr := fmt.Sprintf("%.1fh / %.0fh", m.WeekHours, m.WeekTarget)
	b.WriteString("This Week: ")

	// Color based on progress
	pct := m.WeekHours / m.WeekTarget
	style := NormalStyle
	if pct >= 1.0 {
		style = SuccessStyle
	} else if pct >= 0.8 {
		style = lipgloss.NewStyle().Foreground(colorHighlight)
	}
	b.WriteString(style.Render(weekStr))

	return b.String()
}

func (m *DashboardModel) renderDailyBreakdown() string {
	var b strings.Builder

	// Ordered days (Mon-Sun for standard week)
	days := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

	// Header row
	for _, d := range days {
		b.WriteString(fmt.Sprintf("  %-6s", d))
	}
	b.WriteString("\n")

	// Hours row
	today := time.Now().Weekday().String()[:3]
	for _, d := range days {
		hours := m.DailyHours[d]
		var cell string
		if hours == 0 {
			cell = "-"
		} else {
			cell = fmt.Sprintf("%.1fh", hours)
		}

		// Highlight today
		if d == today {
			cell = SelectedStyle.Render(cell)
		} else {
			cell = DimStyle.Render(cell)
		}
		b.WriteString(fmt.Sprintf("  %-6s", cell))
	}

	return b.String()
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
