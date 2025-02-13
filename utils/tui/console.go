package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func AskQuestion(question string, defaultAnswer string) string {
	p := tea.NewProgram(InitialInputModel(question, defaultAnswer))
	m, _ := p.Run()
	return m.(InputModel).TextInput.Value()
}
