package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/colorstring"
	"github.com/yodamad/heimdall/commons"
)

// MultiItem is one line of a list several of which can be picked
type MultiItem struct {
	// Label is what is picked, and what Picked returns
	Label string
	// Hint is displayed next to the label, dimmed. It must never hold a secret.
	Hint string
}

// MultiModel lets several items of a list be picked. MenuModel does the same for the git folders of
// the interactive git-info, but it is tied to that type.
type MultiModel struct {
	title     string
	items     []MultiItem
	cursor    int
	selected  map[int]bool
	Cancelled bool
}

// InitialMultiModel builds the list. The items of preselected start out picked, which is how the
// platforms needing attention are offered first.
func InitialMultiModel(title string, items []MultiItem, preselected map[int]bool) MultiModel {
	if !commons.NoColor {
		title = TitleColor + title + ":[default]"
	}
	selected := map[int]bool{}
	for index, isSelected := range preselected {
		if isSelected && index >= 0 && index < len(items) {
			selected[index] = true
		}
	}
	return MultiModel{
		title:    colorstring.Color(title),
		items:    items,
		selected: selected,
	}
}

func (m MultiModel) Init() tea.Cmd { return nil }

func (m MultiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			m.Cancelled = true
			return m, tea.Quit

		case "enter":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case " ":
			if m.selected[m.cursor] {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = true
			}

		case "a":
			if len(m.selected) == len(m.items) {
				m.selected = map[int]bool{}
			} else {
				for index := range m.items {
					m.selected[index] = true
				}
			}
		}
	}
	return m, nil
}

func (m MultiModel) View() string {
	view := m.title + "\n\n"
	for index, item := range m.items {
		box := " "
		if m.selected[index] {
			box = "x"
		}
		line := fmt.Sprintf(" [%s] %s", box, item.Label)
		if item.Hint != "" {
			line += "  " + colorstring.Color("[dark_gray]"+item.Hint+"[reset]")
		}
		if m.cursor == index {
			view += colorstring.Color("[bold]>[reset]") + line + "\n"
		} else {
			view += " " + line + "\n"
		}
	}
	view += colorstring.Color("\n[dark_gray]space select   a all   enter confirm   esc cancel[reset]\n")
	return view
}

// Picked returns the labels of the picked items, in the order of the list
func (m MultiModel) Picked() []string {
	if m.Cancelled {
		return nil
	}
	picked := make([]string, 0, len(m.selected))
	for index, item := range m.items {
		if m.selected[index] {
			picked = append(picked, item.Label)
		}
	}
	return picked
}
