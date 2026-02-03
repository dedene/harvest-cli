package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// Spinner displays a loading indicator with a message.
type Spinner struct {
	spinner spinner.Model
	message string
	program *tea.Program
	done    chan struct{}
}

type spinnerModel struct {
	spinner spinner.Model
	message string
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// NewSpinner creates a new spinner with the given message.
func NewSpinner(message string) *Spinner {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return &Spinner{
		spinner: s,
		message: message,
		done:    make(chan struct{}),
	}
}

// Start begins displaying the spinner.
func (s *Spinner) Start() {
	model := spinnerModel{
		spinner: s.spinner,
		message: s.message,
	}
	s.program = tea.NewProgram(model, tea.WithOutput(os.Stderr))

	go func() {
		_, _ = s.program.Run()
		close(s.done)
	}()
}

// Stop terminates the spinner display.
func (s *Spinner) Stop() {
	if s.program != nil {
		s.program.Quit()
		<-s.done
	}
}

// WithSpinner executes a function while displaying a spinner.
func WithSpinner[T any](message string, fn func() (T, error)) (T, error) {
	s := NewSpinner(message)
	s.Start()
	defer s.Stop()
	return fn()
}
