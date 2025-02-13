package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mitchellh/colorstring"
	"github.com/yodamad/heimdall/commons"
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
		if commons.NoColor {
			m.Warns = append(m.Warns, "Cannot fetch "+msg2.(PrintMessage).Path+". Skip it...")
		} else {
			m.Warns = append(m.Warns, colorstring.Color("[light_yellow]Cannot fetch "+msg2.(PrintMessage).Path+". Skip it...[default]"))
		}
	case ErrorMessage:
		if commons.NoColor {
			m.Warns = append(m.Warns, "Error "+msg2.(ErrorMessage).Error)
		} else {
			m.Warns = append(m.Warns, colorstring.Color("[red]Error "+msg2.(ErrorMessage).Error+"[default]"))
		}
	case InfoMessage:
		m.Warns = append(m.Warns, msg2.(InfoMessage).Message)
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

type ErrorMessage struct {
	Error string
}

type InfoMessage struct {
	Message string
}

func TheEndMessage() tea.Cmd {
	return tea.Quit
}
