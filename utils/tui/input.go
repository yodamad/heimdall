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
	// Cancelled tells that the question was left with escape or ctrl+c instead of being answered.
	// Without it an empty answer and a cancel look the same, which must never happen for a secret.
	Cancelled bool
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

// InitialSecretInputModel asks for a value which must not be displayed while it is typed, such as a
// token or the passphrase of an ssh key
func InitialSecretInputModel(q string) InputModel {
	ti := textinput.New()
	ti.Focus()
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	// A token can be long : the 200 characters of a plain question would truncate it silently
	ti.CharLimit = 2048
	ti.Width = 60
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
		case tea.KeyEnter:
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Cancelled = true
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
