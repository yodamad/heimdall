package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/colorstring"
)

var (
	textStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render
	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
)

type SpinnerModel struct {
	Spinner spinner.Model
	tick    int
	Text    string
	Warns   []string
}

func (m SpinnerModel) Init() tea.Cmd {
	return m.Spinner.Tick
}

func (m SpinnerModel) Update(msg2 tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg2.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		// Update spinner
		var cmd tea.Cmd
		m.Spinner, cmd = m.Spinner.Update(msg)
		return m, cmd
	case PrintMessage:
		m.Warns = append(m.Warns, colorstring.Color("[light_yellow]Cannot fetch "+msg2.(PrintMessage).Path+". Skip it...[default]"))
	}
	return m, nil
}

// View renders the spinner and instructions
func (m SpinnerModel) View() string {
	var s = ""
	for _, w := range m.Warns {
		s += w + "\n"
	}
	return fmt.Sprintf("%s%s %s\n", s, m.Spinner.View(), textStyle(m.Text))
}

type PrintMessage struct {
	Path string
}

func TheEndMessage() tea.Cmd {
	return tea.Quit
}
