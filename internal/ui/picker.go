package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// PickerItem interface for items displayed in the picker.
type PickerItem interface {
	ID() int64
	Title() string
	Description() string
}

// listItem wraps a PickerItem for the bubbles list.
type listItem struct {
	item PickerItem
}

func (i listItem) FilterValue() string {
	return i.item.Title() + " " + i.item.Description()
}

// itemDelegate renders list items.
type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	li, ok := item.(listItem)
	if !ok {
		return
	}

	title := li.item.Title()
	desc := li.item.Description()

	var line string
	if desc != "" {
		line = fmt.Sprintf("%s - %s", title, desc)
	} else {
		line = title
	}

	// Truncate if too long
	maxWidth := m.Width() - 4
	if maxWidth > 0 && len(line) > maxWidth {
		line = line[:maxWidth-3] + "..."
	}

	if index == m.Index() {
		fmt.Fprint(w, SelectedStyle.Render("> "+line))
	} else {
		fmt.Fprint(w, NormalStyle.Render("  "+line))
	}
}

// Picker is a searchable list picker for selecting items.
type Picker struct {
	list     list.Model
	selected PickerItem
	done     bool
	canceled bool
}

// NewPicker creates a new picker with the given title and items.
func NewPicker(title string, items []PickerItem) *Picker {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = listItem{item: item}
	}

	l := list.New(listItems, itemDelegate{}, 60, 15)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(true)
	l.Styles.Title = TitleStyle
	l.Styles.FilterPrompt = PromptStyle
	l.Styles.FilterCursor = SelectedStyle

	return &Picker{list: l}
}

// Init implements tea.Model.
func (p *Picker) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (p *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.list.SetWidth(msg.Width)
		h := msg.Height - 4
		if h < 5 {
			h = 5
		}
		p.list.SetHeight(h)
		return p, nil

	case tea.KeyMsg:
		// Don't intercept keys when filtering
		if p.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "enter":
			if item, ok := p.list.SelectedItem().(listItem); ok {
				p.selected = item.item
				p.done = true
				return p, tea.Quit
			}
		case "esc", "ctrl+c", "q":
			p.canceled = true
			return p, tea.Quit
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View implements tea.Model.
func (p *Picker) View() string {
	if p.done || p.canceled {
		return ""
	}
	return p.list.View()
}

// Run executes the picker and returns the selected item.
func (p *Picker) Run() (PickerItem, error) {
	program := tea.NewProgram(p, tea.WithOutput(os.Stderr))
	finalModel, err := program.Run()
	if err != nil {
		return nil, err
	}

	picker := finalModel.(*Picker)
	if picker.canceled {
		return nil, ErrCanceled
	}
	return picker.selected, nil
}

// Selected returns the selected item (nil if none).
func (p *Picker) Selected() PickerItem {
	return p.selected
}

// Canceled returns true if the picker was canceled.
func (p *Picker) Canceled() bool {
	return p.canceled
}

// ProjectItem implements PickerItem for Harvest projects.
type ProjectItem struct {
	ProjectID   int64
	ProjectName string
	ClientName  string
	Code        string
}

func (p ProjectItem) ID() int64     { return p.ProjectID }
func (p ProjectItem) Title() string { return p.ProjectName }
func (p ProjectItem) Description() string {
	parts := []string{}
	if p.ClientName != "" {
		parts = append(parts, p.ClientName)
	}
	if p.Code != "" {
		parts = append(parts, p.Code)
	}
	return strings.Join(parts, " | ")
}

// TaskItem implements PickerItem for Harvest tasks.
type TaskItem struct {
	TaskID   int64
	TaskName string
	Billable bool
}

func (t TaskItem) ID() int64     { return t.TaskID }
func (t TaskItem) Title() string { return t.TaskName }
func (t TaskItem) Description() string {
	if t.Billable {
		return "billable"
	}
	return ""
}

// ClientItem implements PickerItem for Harvest clients.
type ClientItem struct {
	ClientID   int64
	ClientName string
}

func (c ClientItem) ID() int64           { return c.ClientID }
func (c ClientItem) Title() string       { return c.ClientName }
func (c ClientItem) Description() string { return "" }

// UserItem implements PickerItem for Harvest users.
type UserItem struct {
	UserID    int64
	FirstName string
	LastName  string
	Email     string
}

func (u UserItem) ID() int64           { return u.UserID }
func (u UserItem) Title() string       { return fmt.Sprintf("%s %s", u.FirstName, u.LastName) }
func (u UserItem) Description() string { return u.Email }

// PickProject shows a project picker and returns the selected project.
func PickProject(title string, projects []ProjectItem) (*ProjectItem, error) {
	items := make([]PickerItem, len(projects))
	for i := range projects {
		items[i] = projects[i]
	}

	picker := NewPicker(title, items)
	selected, err := picker.Run()
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return nil, nil
	}

	proj := selected.(ProjectItem)
	return &proj, nil
}

// PickTask shows a task picker and returns the selected task.
func PickTask(title string, tasks []TaskItem) (*TaskItem, error) {
	items := make([]PickerItem, len(tasks))
	for i := range tasks {
		items[i] = tasks[i]
	}

	picker := NewPicker(title, items)
	selected, err := picker.Run()
	if err != nil {
		return nil, err
	}
	if selected == nil {
		return nil, nil
	}

	task := selected.(TaskItem)
	return &task, nil
}
