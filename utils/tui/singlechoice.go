package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/colorstring"
	"github.com/yodamad/heimdall/commons"
)

type ChoiceModel struct {
	title    string
	choices  []string       // items on the to-do list
	cursor   int            // which to-do list item our cursor is pointing at
	selected map[int]string // which to-do items are Selected
}

func InitialChoiceModel(title string, choices []string) ChoiceModel {
	if !commons.NoColor {
		title = TitleColor + title + ":[default]"
	}
	return ChoiceModel{
		title: colorstring.Color(title),
		// Our to-do list is a grocery list
		choices: choices,

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: map[int]string{},
	}
}

func (m ChoiceModel) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m ChoiceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			m.selected[m.cursor] = m.choices[m.cursor]
			return m, tea.Quit
		}
	}

	// Return the updated choiceModel to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m ChoiceModel) View() string {
	// The header
	s := m.title + "\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
			s += fmt.Sprintf(colorstring.Color("[bold]%s %s[reset]\n"), cursor, choice)
		} else {
			s += fmt.Sprintf("%s %s\n", cursor, choice)
		}
	}

	// The footer
	//s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

func (m ChoiceModel) Picked() string {
	var picked string
	for i, _ := range m.selected {
		picked = m.selected[i]
	}
	return picked
}
