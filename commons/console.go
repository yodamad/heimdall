package commons

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/utils/tui"
)

func AskQuestion(question string, defaultAnswer string) string {
	p := tea.NewProgram(tui.InitialInputModel(question, defaultAnswer))
	m, _ := p.Run()
	return m.(tui.InputModel).TextInput.Value()
}
