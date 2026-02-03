package ui

import (
	"fmt"
	"time"
)

// WizardStep represents a step in a multi-step wizard.
type WizardStep interface {
	Name() string
	Run(data map[string]any) (map[string]any, error)
}

// Wizard is a multi-step form for collecting data.
type Wizard struct {
	steps    []WizardStep
	current  int
	data     map[string]any
	done     bool
	canceled bool
}

// NewWizard creates a new wizard with the given steps.
func NewWizard(steps ...WizardStep) *Wizard {
	return &Wizard{
		steps: steps,
		data:  make(map[string]any),
	}
}

// Run executes all wizard steps sequentially.
func (w *Wizard) Run() (map[string]any, error) {
	for w.current < len(w.steps) {
		step := w.steps[w.current]

		result, err := step.Run(w.data)
		if err != nil {
			if err == ErrCanceled {
				w.canceled = true
				return nil, err
			}
			return nil, fmt.Errorf("step %s: %w", step.Name(), err)
		}

		// Merge step results into data
		for k, v := range result {
			w.data[k] = v
		}

		w.current++
	}

	w.done = true
	return w.data, nil
}

// Data returns the collected data.
func (w *Wizard) Data() map[string]any {
	return w.data
}

// Canceled returns true if the wizard was canceled.
func (w *Wizard) Canceled() bool {
	return w.canceled
}

// ProjectStep prompts for project selection.
type ProjectStep struct {
	Projects []ProjectItem
}

func (s *ProjectStep) Name() string { return "project" }

func (s *ProjectStep) Run(data map[string]any) (map[string]any, error) {
	proj, err := PickProject("Select Project", s.Projects)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, ErrCanceled
	}
	return map[string]any{
		"project_id":   proj.ProjectID,
		"project_name": proj.ProjectName,
		"client_name":  proj.ClientName,
	}, nil
}

// TaskStep prompts for task selection.
type TaskStep struct {
	TasksFn func(projectID int64) ([]TaskItem, error)
}

func (s *TaskStep) Name() string { return "task" }

func (s *TaskStep) Run(data map[string]any) (map[string]any, error) {
	projectID, ok := data["project_id"].(int64)
	if !ok {
		return nil, fmt.Errorf("project_id not found in data")
	}

	tasks, err := s.TasksFn(projectID)
	if err != nil {
		return nil, fmt.Errorf("fetch tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks found for project")
	}

	task, err := PickTask("Select Task", tasks)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrCanceled
	}
	return map[string]any{
		"task_id":   task.TaskID,
		"task_name": task.TaskName,
		"billable":  task.Billable,
	}, nil
}

// DateStep prompts for a date.
type DateStep struct {
	Message string
	Default time.Time
}

func (s *DateStep) Name() string { return "date" }

func (s *DateStep) Run(_ map[string]any) (map[string]any, error) {
	defaultStr := s.Default.Format("2006-01-02")
	if s.Default.IsZero() {
		defaultStr = time.Now().Format("2006-01-02")
	}

	dateStr, err := TextPrompt(s.Message, defaultStr)
	if err != nil {
		return nil, err
	}

	if dateStr == "" {
		dateStr = defaultStr
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (use YYYY-MM-DD): %w", err)
	}

	return map[string]any{
		"spent_date": date.Format("2006-01-02"),
	}, nil
}

// HoursStep prompts for hours.
type HoursStep struct {
	Message string
	Default float64
}

func (s *HoursStep) Name() string { return "hours" }

func (s *HoursStep) Run(_ map[string]any) (map[string]any, error) {
	hours, err := NumberPrompt(s.Message, s.Default)
	if err != nil {
		return nil, err
	}

	if hours <= 0 {
		return nil, fmt.Errorf("hours must be greater than 0")
	}
	if hours > 24 {
		return nil, fmt.Errorf("hours cannot exceed 24")
	}

	return map[string]any{
		"hours": hours,
	}, nil
}

// NotesStep prompts for notes.
type NotesStep struct {
	Message  string
	Required bool
}

func (s *NotesStep) Name() string { return "notes" }

func (s *NotesStep) Run(_ map[string]any) (map[string]any, error) {
	notes, err := TextPrompt(s.Message, "")
	if err != nil {
		return nil, err
	}

	if s.Required && notes == "" {
		return nil, fmt.Errorf("notes are required")
	}

	return map[string]any{
		"notes": notes,
	}, nil
}

// NewTimeEntryWizard creates a wizard for creating time entries.
func NewTimeEntryWizard(projects []ProjectItem, tasksFn func(int64) ([]TaskItem, error)) *Wizard {
	return NewWizard(
		&ProjectStep{Projects: projects},
		&TaskStep{TasksFn: tasksFn},
		&DateStep{Message: "Date (YYYY-MM-DD):", Default: time.Now()},
		&HoursStep{Message: "Hours:", Default: 1.0},
		&NotesStep{Message: "Notes (optional):"},
	)
}

// TimeEntryFromWizard extracts time entry data from wizard results.
type TimeEntryData struct {
	ProjectID   int64
	ProjectName string
	TaskID      int64
	TaskName    string
	SpentDate   string
	Hours       float64
	Notes       string
}

// ParseTimeEntryData extracts structured data from wizard results.
func ParseTimeEntryData(data map[string]any) (*TimeEntryData, error) {
	entry := &TimeEntryData{}

	if v, ok := data["project_id"].(int64); ok {
		entry.ProjectID = v
	} else {
		return nil, fmt.Errorf("missing project_id")
	}

	if v, ok := data["project_name"].(string); ok {
		entry.ProjectName = v
	}

	if v, ok := data["task_id"].(int64); ok {
		entry.TaskID = v
	} else {
		return nil, fmt.Errorf("missing task_id")
	}

	if v, ok := data["task_name"].(string); ok {
		entry.TaskName = v
	}

	if v, ok := data["spent_date"].(string); ok {
		entry.SpentDate = v
	} else {
		return nil, fmt.Errorf("missing spent_date")
	}

	if v, ok := data["hours"].(float64); ok {
		entry.Hours = v
	} else {
		return nil, fmt.Errorf("missing hours")
	}

	if v, ok := data["notes"].(string); ok {
		entry.Notes = v
	}

	return entry, nil
}
