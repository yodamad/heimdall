package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/colorstring"
	"github.com/yodamad/heimdall/cmd/entity"
)

type MenuModel struct {
	question string
	choices  []entity.GitFolder       // items on the to-do list
	cursor   int                      // which to-do list item our cursor is pointing at
	Selected map[int]entity.GitFolder // which to-do items are Selected
}

func InitialMenuModel(question string, choices []entity.GitFolder) MenuModel {
	return MenuModel{
		question: question,
		choices:  choices,
		Selected: map[int]entity.GitFolder{},
	}
}

func (m MenuModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.Selected[m.cursor]

			if ok {
				delete(m.Selected, m.cursor)
			} else {
				m.Selected[m.cursor] = m.choices[m.cursor]
			}
		}
	}

	// Return the updated menuModel to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m MenuModel) View() string {
	// The header
	s := m.question + "\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {
		//cursor := " " // no cursor
		if m.cursor == i {
			checked := " " // not selected
			if _, ok := m.Selected[i]; ok {
				checked = colorstring.Color("[light_green]x") // selected!
				s += fmt.Sprintf(colorstring.Color(" [bold]> [%s] [bold][light_green]%s[reset]\n"), checked, choice.Path)
			} else {
				s += fmt.Sprintf(colorstring.Color(" [bold]> [%s] [bold]%s[reset]\n"), checked, choice.Path)
			}

		} else {
			checked := " " // not selected
			if _, ok := m.Selected[i]; ok {
				checked = colorstring.Color("[light_green]x") // selected!
				s += fmt.Sprintf(colorstring.Color("  [%s] [light_green]%s[reset]\n"), checked, choice.Path)
			} else {
				s += fmt.Sprintf(colorstring.Color("  [%s] %s[reset]\n"), checked, choice.Path)
			}
		}
	}

	s += "\nPress q when done\n"
	return s
}
