package ui

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ErrCanceled indicates user canceled the operation.
var ErrCanceled = errors.New("operation canceled")

// RunProgram executes a bubbletea program and returns any error.
func RunProgram(model tea.Model) error {
	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	_, err := p.Run()
	return err
}

// confirmModel for yes/no prompts.
type confirmModel struct {
	message  string
	selected bool // true = yes, false = no
	done     bool
	canceled bool
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y":
			m.selected = true
			m.done = true
			return m, tea.Quit
		case "n", "N":
			m.selected = false
			m.done = true
			return m, tea.Quit
		case "left", "h":
			m.selected = true
		case "right", "l":
			m.selected = false
		case "enter":
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c", "q":
			m.canceled = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	var yes, no string
	if m.selected {
		yes = SelectedStyle.Render("[Yes]")
		no = DimStyle.Render(" No ")
	} else {
		yes = DimStyle.Render(" Yes ")
		no = SelectedStyle.Render("[No]")
	}

	return fmt.Sprintf("%s %s %s\n%s",
		PromptStyle.Render(m.message),
		yes,
		no,
		HelpStyle.Render("y/n or ←/→ to select, enter to confirm, esc to cancel"),
	)
}

// ConfirmPrompt displays a yes/no confirmation prompt.
func ConfirmPrompt(message string) (bool, error) {
	model := confirmModel{
		message:  message,
		selected: true, // default to yes
	}

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	m := finalModel.(confirmModel)
	if m.canceled {
		return false, ErrCanceled
	}
	return m.selected, nil
}

// textPromptModel for text input.
type textPromptModel struct {
	input    textinput.Model
	message  string
	done     bool
	canceled bool
}

func (m textPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.canceled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textPromptModel) View() string {
	return fmt.Sprintf("%s\n%s\n%s",
		PromptStyle.Render(m.message),
		m.input.View(),
		HelpStyle.Render("enter to confirm, esc to cancel"),
	)
}

// TextPrompt displays a text input prompt.
func TextPrompt(message, placeholder string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 40

	model := textPromptModel{
		input:   ti,
		message: message,
	}

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	m := finalModel.(textPromptModel)
	if m.canceled {
		return "", ErrCanceled
	}
	return m.input.Value(), nil
}

// NumberPrompt displays a numeric input prompt.
func NumberPrompt(message string, defaultVal float64) (float64, error) {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("%.2f", defaultVal)
	ti.Focus()
	ti.CharLimit = 10
	ti.Width = 20

	model := textPromptModel{
		input:   ti,
		message: message,
	}

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return 0, err
	}

	m := finalModel.(textPromptModel)
	if m.canceled {
		return 0, ErrCanceled
	}

	value := m.input.Value()
	if value == "" {
		return defaultVal, nil
	}

	var result float64
	_, err = fmt.Sscanf(value, "%f", &result)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", value)
	}
	return result, nil
}
