package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type (
	errMsg error
)

type InputModel struct {
	question  string
	TextInput textinput.Model
	err       error
}

func InitialInputModel(q string, ph string) InputModel {
	ti := textinput.New()
	ti.Placeholder = ph
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 200
	ti.Prompt = ""

	return InputModel{
		question:  q,
		TextInput: ti,
		err:       nil,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	return fmt.Sprintf(
		m.question+"%s",
		m.TextInput.View(),
	) + "\n"
}
