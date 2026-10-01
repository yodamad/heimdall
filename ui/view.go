package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
)

// Max number of output lines kept per command in the detail pane
const maxOutputLines = 200

const (
	branchW = 16
	localW  = 3
	remoteW = 5
	stateW  = 1
)

const helpText = `Navigation
  ↑/↓ j/k      move
  g/G          first / last repository
  pgup/pgdown  previous / next page
  tab          focus details pane (scroll with ↑/↓), tab again to go back
  /            filter repositories by path
  esc          clear filter, then selection

Selection
  space        select / unselect repository
  a            select / unselect all displayed repositories
  u            select repositories which can be pulled

Actions (on selected repositories, or the current one if none is selected)
  r            refresh local status
  f            git fetch
  p            git pull (skipped if there are local changes)
  m            run morning routine
  !            run a command

  ?            this help
  q            quit

Remote changes are computed from the last fetch : they are displayed
with a ~ until the repository is fetched from here.`

func (m *model) View() string {
	if m.width == 0 {
		return ""
	}

	var body string
	switch {
	case m.mode == modeHelp:
		body = paneStyle.Width(m.width-2).Height(m.paneH-2).Padding(0, 1).Render(helpText)
	case m.narrow() && m.detailFocus:
		body = m.detailPane()
	case m.narrow():
		body = m.listPane()
	default:
		body = lipgloss.JoinHorizontal(lipgloss.Top, m.listPane(), m.detailPane())
	}
	return m.headerLine() + "\n" + body + "\n" + m.statusLine() + "\n" + m.helpLine()
}

func (m *model) headerLine() string {
	nbSelected, nbBusy := 0, 0
	for _, r := range m.rows {
		if r.selected {
			nbSelected++
		}
		if r.busy != "" {
			nbBusy++
		}
	}
	infos := strconv.Itoa(len(m.rows)) + " repositories"
	if len(m.visible) != len(m.rows) {
		infos += " · " + strconv.Itoa(len(m.visible)) + " displayed"
	}
	if nbSelected > 0 {
		infos += " · " + strconv.Itoa(nbSelected) + " selected"
	}
	if nbBusy > 0 {
		infos += " · " + strconv.Itoa(nbBusy) + " running"
	}
	infos += " · " + m.root
	return titleStyle.Render(" Heimdall ") + dimStyle.Render(fit(infos, m.width-10))
}

func (m *model) statusLine() string {
	switch m.mode {
	case modeFilter:
		return " " + m.filter.View()
	case modeCommand:
		return " " + m.cmdInput.View()
	case modeConfirm:
		return warnStyle.Render(fit(" "+m.pending.question, m.width))
	}
	if m.status != "" {
		return warnStyle.Render(fit(" "+m.status, m.width))
	}
	if m.filter.Value() != "" {
		return dimStyle.Render(fit(" filter: "+m.filter.Value(), m.width))
	}
	return ""
}

func (m *model) helpLine() string {
	help := "? help · ↑↓ move · space select · / filter · f fetch · p pull · m morning · ! cmd · r refresh · tab details · q quit"
	switch {
	case m.mode == modeFilter:
		help = "enter apply · esc clear"
	case m.mode == modeCommand:
		help = "enter run · esc cancel"
	case m.mode == modeConfirm:
		help = "y confirm · any other key cancel"
	case m.mode == modeHelp:
		help = "press any key to go back"
	case m.detailFocus:
		help = "↑↓ scroll · tab back to repositories · q quit"
	}
	return dimStyle.Render(fit(" "+help, m.width))
}

func (m *model) listPane() string {
	width := m.listW - 2
	style := paneStyle
	if !m.detailFocus {
		style = focusStyle
	}
	style = style.Width(width).Height(m.paneH - 2)

	if m.discovering {
		return style.Render(" " + m.spin.View() + " Looking for git folders in " + m.root)
	}
	if len(m.rows) == 0 {
		return style.Render(fit(" No git folder found, is "+m.root+" the correct path ?", width))
	}

	selW := 2
	if commons.NoColor {
		selW = 4
	}
	// cursor, selection mark and a space between each column
	nameW := width - 2 - selW - branchW - localW - remoteW - stateW - 4
	if nameW < 8 {
		nameW = 8
	}

	lines := []string{dimStyle.Render(fit(strings.Repeat(" ", 2+selW)+fit("repository", nameW)+" "+fit("branch", branchW)+" "+fit("loc", localW)+" "+fit("rem", remoteW), width))}
	end := m.offset + m.listRows()
	if end > len(m.visible) {
		end = len(m.visible)
	}
	for i := m.offset; i < end; i++ {
		lines = append(lines, m.rowLine(m.visible[i], i == m.cursor, nameW))
	}
	if len(m.visible) == 0 {
		lines = append(lines, dimStyle.Render(" No repository matches the filter"))
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m *model) rowLine(r *row, isCursor bool, nameW int) string {
	cursor := "  "
	if isCursor {
		cursor = "▸ "
		if commons.NoColor {
			cursor = "> "
		}
	}

	selected := "  "
	if commons.NoColor {
		selected = "[ ] "
		if r.selected {
			selected = "[x] "
		}
	} else if r.selected {
		selected = "● "
	}

	nameStyle := lipgloss.NewStyle()
	if r.selected {
		nameStyle = selectStyle
	}
	if isCursor {
		nameStyle = nameStyle.Bold(true)
	}

	state := " "
	switch {
	case r.busy != "":
		state = m.spin.View()
	case r.note != "" && r.noteOK:
		state = okStyle.Render("✓")
	case r.note != "":
		state = koStyle.Render("✗")
	}

	return titleStyle.Render(cursor) + selectStyle.Render(selected) +
		nameStyle.Render(fitLeft(r.name, nameW)) + " " +
		dimStyle.Render(fit(r.gf.CurrentBranch, branchW)) + " " +
		localCell(r) + " " + remoteCell(r) + " " + state
}

func localCell(r *row) string {
	switch {
	case !r.loaded:
		return dimStyle.Render(fit("…", localW))
	case r.gf.Err != "":
		return koStyle.Render(fit("!", localW))
	case r.gf.HasLocalChanges && commons.NoColor:
		return fit("KO", localW)
	case r.gf.HasLocalChanges:
		return koStyle.Render(fit("●", localW))
	case commons.NoColor:
		return fit("OK", localW)
	}
	return okStyle.Render(fit("●", localW))
}

func remoteCell(r *row) string {
	behind := strings.TrimSpace(r.gf.RemoteChanges)
	if !r.loaded || behind == "" {
		return dimStyle.Render(fit("-", remoteW))
	}
	text, style := "✓", okStyle
	if commons.NoColor {
		text = "OK"
	}
	if entity.HasRemoteChanges(r.gf) {
		text, style = "↓"+behind, koStyle
	}
	if !r.fetched {
		text, style = "~"+text, dimStyle
	}
	return style.Render(fit(text, remoteW))
}

func (m *model) detailPane() string {
	style := paneStyle
	if m.detailFocus {
		style = focusStyle
	}
	title := ""
	if r := m.current(); r != nil {
		title = r.name
	}
	return style.Width(m.detailW-2).Height(m.paneH-2).Padding(0, 1).
		Render(titleStyle.Render(fitLeft(title, m.detail.Width)) + "\n" + m.detail.View())
}

func (m *model) detailContent(r *row) string {
	var b strings.Builder
	field := func(label string, value string) {
		b.WriteString(dimStyle.Render(fit(label, 8)) + value + "\n")
	}
	section := func(title string) {
		b.WriteString("\n" + boldStyle.Render(title) + "\n")
	}

	field("Path", r.gf.Path)
	if !r.loaded {
		b.WriteString("\n" + m.spin.View() + " Loading...")
		return b.String()
	}
	if r.gf.CurrentBranch != "" {
		field("Branch", r.gf.CurrentBranch)
	}
	if r.gf.RemoteURL != "" {
		field("Remote", strings.TrimSpace(r.gf.ConnectionType+" "+r.gf.RemoteURL))
	} else {
		field("Remote", dimStyle.Render("no "+commons.RemoteName+" remote"))
	}
	if r.gf.Err != "" {
		field("Error", koStyle.Render(r.gf.Err))
	}

	if behind := strings.TrimSpace(r.gf.RemoteChanges); behind != "" {
		sync := "↑" + strconv.Itoa(r.gf.Ahead) + " ↓" + behind
		if !r.fetched {
			sync += dimStyle.Render("  since last fetch, press f to update")
		}
		field("Sync", sync)
	}
	if r.note != "" {
		style := koStyle
		if r.noteOK {
			style = okStyle
		}
		field("Last", style.Render(r.note))
	}

	if len(r.gf.ChangedFiles) > 0 {
		section("Local changes (" + strconv.Itoa(len(r.gf.ChangedFiles)) + ")")
		b.WriteString(strings.Join(r.gf.ChangedFiles, "\n") + "\n")
	} else if r.gf.Err == "" {
		b.WriteString("\n" + okStyle.Render("No local changes") + "\n")
	}

	if len(r.incoming) > 0 {
		section("Incoming commits (" + strconv.Itoa(len(r.incoming)) + ")")
		b.WriteString(strings.Join(r.incoming, "\n") + "\n")
	}

	if len(r.cmds) > 0 {
		section("Commands")
		for _, cmd := range r.cmds {
			if cmd.ExitCode == 0 {
				b.WriteString(okStyle.Render("✓ "+cmd.Cmd) + "\n")
			} else {
				b.WriteString(koStyle.Render("✗ "+cmd.Cmd+" (exit code "+strconv.Itoa(cmd.ExitCode)+")") + "\n")
			}
			if output := cleanOutput(cmd.Output); output != "" {
				b.WriteString(dimStyle.Render(output) + "\n")
			}
		}
	}
	return b.String()
}

func cleanOutput(output string) string {
	output = strings.ReplaceAll(strings.ReplaceAll(output, "\r", ""), "\t", "    ")
	output = strings.TrimRight(output, "\n ")
	lines := strings.Split(output, "\n")
	if len(lines) > maxOutputLines {
		lines = append([]string{"..."}, lines[len(lines)-maxOutputLines:]...)
	}
	return strings.Join(lines, "\n")
}
