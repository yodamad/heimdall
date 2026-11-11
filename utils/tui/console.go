package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func AskQuestion(question string, defaultAnswer string) string {
	p := tea.NewProgram(InitialInputModel(question, defaultAnswer))
	m, _ := p.Run()
	if m.(InputModel).TextInput.Value() == "" {
		return defaultAnswer
	}
	return m.(InputModel).TextInput.Value()
}

// AskSecret asks for a value which is not displayed while it is typed. The second result is false
// when the question was cancelled, which has to be told apart from an empty answer : escaping out of
// a prompt must never store an empty token.
func AskSecret(question string) (string, bool) {
	model, err := tea.NewProgram(InitialSecretInputModel(question)).Run()
	if err != nil {
		return "", false
	}
	input := model.(InputModel)
	if input.Cancelled {
		return "", false
	}
	return input.TextInput.Value(), true
}

// PickSeveral asks for several items of a list to be picked. The second result is false when the
// list was cancelled.
func PickSeveral(title string, items []MultiItem, preselected map[int]bool) ([]string, bool) {
	model, err := tea.NewProgram(InitialMultiModel(title, items, preselected)).Run()
	if err != nil {
		return nil, false
	}
	list := model.(MultiModel)
	if list.Cancelled {
		return nil, false
	}
	return list.Picked(), true
}
