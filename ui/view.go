package ui

import (
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
)

// Max number of output lines kept per command in the detail pane
const maxOutputLines = 200

const (
	branchW  = 14
	stateW   = 28
	dividerW = 3

	// in the branches of the detail pane
	branchStateW = 17
	branchAgeW   = 10
)

const helpText = `Move
  ↑/↓ j/k      move
  g/G          first / last repository
  pgup/pgdown  previous / next page
  tab          go to the details, tab again to go back
  /            filter repositories by path
  esc          clear the filter, then the selection
  enter        collapse / expand the folder
  ← h          collapse the folder, or go to the folder the line is in
  → l          expand the folder, or go into it
  z            collapse / expand all the folders

Select
  space        select / unselect the repository, or all the ones of the folder
  a            select / unselect all the listed repositories
  u            select the repositories which can be pulled

Act on the selected repositories, or if none on the current one or folder
  r            read local status again
  f            fetch
  p            pull, skipped when there are local changes
  m            run the morning routine
  !            run a command

In the details
  ↑/↓ j/k      choose a branch of the repository, pgup/pgdown to scroll
  enter        switch to the chosen branch

  q           quit

A folder lists its folders, then its repositories, the ones needing attention first. Nothing is fetched at startup:
what the list knows about origin dates from the last fetch, press f to check origin.`

var busyVerbs = map[string]string{
	busyStatus:  "reading",
	busyRefresh: "reading",
	busyFetch:   "fetching",
	busyPull:    "pulling",
	busySwitch:  "switching",
	busyCmd:     "running",
}

func (m *model) View() string {
	if m.width == 0 {
		return ""
	}

	var body []string
	switch {
	case m.mode == modeHelp:
		lines := strings.Split(helpText, "\n")
		for _, line := range lines {
			// Short terminals get the help without its blank lines
			if line != "" || len(lines) <= m.paneH {
				body = append(body, " "+fit(line, m.width-1))
			}
		}
	case m.narrow() && m.detailFocus:
		for _, line := range m.detailLines() {
			body = append(body, " "+line)
		}
	case m.narrow():
		body = m.listLines()
	default:
		divider := dimStyle.Render(" │ ")
		if m.detailFocus {
			divider = lipgloss.NewStyle().Foreground(selectColor).Render(" │ ")
		}
		detail := m.detailLines()
		for i, line := range m.listLines() {
			if i < len(detail) {
				line += divider + detail[i]
			}
			body = append(body, line)
		}
	}
	for len(body) < m.paneH {
		body = append(body, "")
	}
	return m.headerLine() + "\n" + m.bridgeLine() + "\n\n" + strings.Join(body[:m.paneH], "\n") + "\n" + m.statusLine() + "\n" + m.keysLine()
}

func (m *model) headerLine() string {
	badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#10141C")).Background(selectColor)
	if commons.NoColor {
		badge = lipgloss.NewStyle().Bold(true).Reverse(true)
	}
	const badgeW = 12 // " Heimdall " and its margins

	// On the right, what is running or else where heimdall is looking
	right := strings.TrimSuffix(m.root, "/")
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(right, home) {
		right = "~" + strings.TrimPrefix(right, home)
	}
	if running := m.running(); running > 0 && m.batchTotal > 0 {
		right = busyVerbs[m.batchKind] + " " + strconv.Itoa(m.batchTotal-running) + " of " + strconv.Itoa(m.batchTotal)
	}

	// The count of repositories in each state, in the color of the state
	type count struct {
		text  string
		style lipgloss.Style
	}
	counts := []count{{plural(len(m.rows), "repository", "repositories"), boldStyle}}
	if m.discovering {
		counts[0].text = "looking for repositories"
	}
	byState := m.countByState()
	for st := stateBroken; st <= stateLoading; st++ {
		if byState[st] > 0 {
			counts = append(counts, count{strconv.Itoa(byState[st]) + " " + stateNames[st], lipgloss.NewStyle().Foreground(stateColors[st])})
		}
	}

	// The least urgent counts give way first on a narrow terminal, the right part before them
	width := func() int {
		w := badgeW
		for _, c := range counts {
			w += lipgloss.Width(c.text) + 2
		}
		return w
	}
	if width()+lipgloss.Width(right)+1 > m.width {
		right = ""
	}
	for len(counts) > 1 && width() > m.width {
		counts = counts[:len(counts)-1]
	}

	line := " " + badge.Render(" Heimdall ") + " "
	for _, c := range counts {
		line += c.style.Render(c.text) + "  "
	}
	if gap := m.width - width() - lipgloss.Width(right) - 1; right != "" && gap >= 0 {
		line += strings.Repeat(" ", gap) + dimStyle.Render(right)
	}
	return line
}

func (m *model) countByState() map[state]int {
	byState := map[state]int{}
	for _, r := range m.rows {
		byState[stateOf(r)]++
	}
	return byState
}

// bridgeLine draws the share of each state among the repositories as a colored band,
// from the most urgent on the left: the Bifröst that Heimdall watches over.
func (m *model) bridgeLine() string {
	width := m.width - 2
	if commons.NoColor || len(m.rows) == 0 || width < 10 {
		return ""
	}

	byState := m.countByState()
	cells, used, widest := map[state]int{}, 0, stateClean
	for st, nb := range byState {
		// Even a single repository is worth being seen
		cells[st] = max(1, nb*width/len(m.rows))
		used += cells[st]
		if cells[st] > cells[widest] {
			widest = st
		}
	}
	cells[widest] += width - used

	line := " "
	for st := stateBroken; st <= stateLoading; st++ {
		if cells[st] > 0 {
			line += lipgloss.NewStyle().Background(stateColors[st]).Render(strings.Repeat(" ", cells[st]))
		}
	}
	return line
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
		return dimStyle.Render(fit(" Filtered on \""+m.filter.Value()+"\", esc to list all", m.width))
	}
	return ""
}

// keysLine lists the keys which do something right now, and on how many repositories
func (m *model) keysLine() string {
	var keys []string
	switch {
	case m.mode == modeFilter:
		keys = []string{"enter keep filter", "esc list all"}
	case m.mode == modeCommand:
		keys = []string{"enter run", "esc cancel"}
	case m.mode == modeConfirm:
		keys = []string{"y confirm", "any other key cancel"}
	case m.mode == modeHelp:
		keys = []string{"any key go back"}
	case m.detailFocus && len(m.branchChoices()) > 0:
		keys = []string{"↑↓ choose a branch"}
		if branch := m.branchChoices()[m.branchCursor]; !branch.Current {
			keys = append(keys, "enter switch to "+branch.Name)
		}
		keys = append(keys, "pgup/pgdown scroll", "tab back to the list", "q quit")
	case m.detailFocus:
		keys = []string{"↑↓ scroll", "tab back to the list", "q quit"}
	case len(m.rows) == 0:
		keys = []string{"q quit"}
	default:
		targets := m.targets()
		nbSelected, nbPullable := 0, 0
		for _, r := range targets {
			if r.selected {
				nbSelected++
			}
			if r.loaded && r.gf.Err == "" && !r.gf.HasLocalChanges {
				nbPullable++
			}
		}
		on := ""
		if nbSelected > 0 {
			on = " " + strconv.Itoa(nbSelected) + " selected"
		}
		if folder := m.currentFolder(); folder != "" {
			if m.isCollapsed(folder) {
				keys = append(keys, "enter expand")
			} else {
				keys = append(keys, "enter collapse")
			}
			if nbSelected == 0 {
				on = " " + strconv.Itoa(len(targets)) + " in folder"
			}
		}
		if len(targets) > 0 {
			keys = append(keys, "f fetch"+on)
			switch {
			case nbPullable == 0:
			case nbPullable < len(targets):
				keys = append(keys, "p pull "+strconv.Itoa(nbPullable)+" of "+strconv.Itoa(len(targets)))
			default:
				keys = append(keys, "p pull"+on)
			}
			if len(m.morning) > 0 {
				keys = append(keys, "m morning routine")
			}
			keys = append(keys, "! run a command", "space select", "tab details")
		}
		keys = append(keys, "/ filter", "? all keys")
	}
	return dimStyle.Render(fit(" "+strings.Join(keys, "   "), m.width))
}

func (m *model) listLines() []string {
	if m.discovering {
		return []string{" " + m.spin.View() + " Looking for git folders in " + m.root}
	}
	if len(m.rows) == 0 {
		return []string{
			fit(" No git folder in "+m.root, m.listW),
			dimStyle.Render(fit(" Start heimdall with -w to look in another directory, or -d to look deeper.", m.listW)),
		}
	}
	if len(m.visible) == 0 {
		return []string{dimStyle.Render(fit(" No repository matches \""+m.filter.Value()+"\", esc to list all", m.listW))}
	}

	lines := make([]string, 0, m.paneH)
	lines = append(lines, m.columnsLine())
	end := min(m.offset+m.listRows(), len(m.lines))
	for i := m.offset; i < end; i++ {
		line := m.lines[i]
		switch {
		case i == m.offset && i > 0 && i+1 < len(m.lines) && m.lines[i+1].parent != "" && m.lines[i+1].parent != line.folder:
			// Where the list is scrolled to stays in sight: the whole path of the folder the next line is in
			parent := m.lines[i+1].parent
			lines = append(lines, m.folderLine(listLine{folder: parent, label: parent}, false))
		case line.r != nil:
			lines = append(lines, m.rowLine(line.r, line.depth, i == m.cursor))
		default:
			lines = append(lines, m.folderLine(line, i == m.cursor))
		}
	}
	for len(lines) < m.paneH {
		lines = append(lines, strings.Repeat(" ", m.listW))
	}
	return lines
}

// columns gives the width left to the name of a repository under guides, and if there
// is room for its branch: the name gives way to the guides, so that branches and
// states stay aligned
func (m *model) columns(guides string) (int, bool) {
	nameW := m.listW - 3 - lipgloss.Width(guides) - 2 - stateW
	if nameW-branchW-2 >= 16 {
		return nameW - branchW - 2, true
	}
	return nameW, false
}

// columnsLine names the columns of the list, above the repositories at the root
func (m *model) columnsLine() string {
	nameW, withBranch := m.columns("")
	line := "   " + fit("Repository", nameW) + "  "
	if withBranch {
		line += fit("Branch", branchW) + "  "
	}
	// As the titles of the details, they stand out from what they are above
	return boldStyle.Render(fit(line+"State", m.listW))
}

// folderStates counts the listed repositories of a folder in each state needing attention
func (m *model) folderStates(folder string) (map[state][]*row, int) {
	rows := m.folderRows(folder)
	byState := map[state][]*row{}
	for _, r := range rows {
		byState[stateOf(r)] = append(byState[stateOf(r)], r)
	}
	return byState, len(rows)
}

// guides are the lines drawn under the folders a line of the list is in. The deepest
// ones are not drawn when they would leave no room for a name.
func (m *model) guides(depth int) string {
	return strings.Repeat("│ ", max(0, min(depth, (m.listW-stateW-17)/2)))
}

// folderLine is a folder of the tree: the path to it stays quiet, its own name stands
// out, followed by how many repositories are listed in it and in the folders it holds.
// A collapsed folder tells what its hidden repositories need.
func (m *model) folderLine(at listLine, isCursor bool) string {
	folder := at.folder
	paint := func(style lipgloss.Style, text string) string {
		if isCursor && commons.NoColor {
			style = style.Reverse(true)
		} else if isCursor {
			style = style.Background(cursorColor)
		}
		return style.Render(text)
	}
	collapsed := m.isCollapsed(folder)
	byState, nb := m.folderStates(folder)

	marker := "▾"
	switch {
	case collapsed && commons.NoColor:
		marker = "+"
	case collapsed:
		marker = "▸"
	case commons.NoColor:
		marker = "-"
	}

	guides := m.guides(at.depth)
	indent := 3 + lipgloss.Width(guides)
	count := "  " + strconv.Itoa(nb)
	name := strings.TrimSpace(fitLeft(at.label, m.listW-indent-lipgloss.Width(count)))
	parent := ""
	if i := strings.LastIndex(name, sep); i >= 0 {
		parent, name = name[:i+1], name[i+1:]
	}
	line := paint(dimStyle, " "+guides+marker+" "+parent) + paint(boldStyle, name) + paint(dimStyle, count)
	used := indent + lipgloss.Width(parent+name+count)

	if collapsed {
		for st := stateBroken; st < stateClean; st++ {
			text := "   " + strconv.Itoa(len(byState[st])) + " " + stateNames[st]
			if len(byState[st]) > 0 && used+lipgloss.Width(text) <= m.listW {
				line += paint(lipgloss.NewStyle().Foreground(stateColors[st]), text)
				used += lipgloss.Width(text)
			}
		}
	}
	return line + paint(lipgloss.NewStyle(), strings.Repeat(" ", max(0, m.listW-used)))
}

// folderContent lists the repositories of a folder by state, the ones needing attention first
func (m *model) folderContent(folder string) string {
	byState, nb := m.folderStates(folder)
	var b strings.Builder
	b.WriteString(plural(nb, "repository", "repositories") + "\n")
	for st := stateBroken; st <= stateLoading; st++ {
		rows := byState[st]
		if len(rows) == 0 {
			continue
		}
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(stateColors[st]).Render(strconv.Itoa(len(rows))+" "+stateNames[st]) + "\n")
		for _, r := range rows {
			// The heading already tells the state of the ones which need nothing
			text := stateText(r)
			switch {
			case st == stateLoading:
				text = ""
			case text == "up to date":
				text = "on " + r.gf.CurrentBranch
			}
			b.WriteString(fit(strings.TrimPrefix(r.name, folder+sep), m.detail.Width-stateW-2) + "  " + dimStyle.Render(fit(text, stateW)) + "\n")
		}
	}
	return b.String()
}

// stateText says in plain words what state the repository is in
func stateText(r *row) string {
	if r.gf.Err != "" {
		return r.gf.Err
	}
	var parts []string
	if nb := len(r.gf.ChangedFiles); nb > 0 {
		parts = append(parts, strconv.Itoa(nb)+" changed")
	}
	if nb := behind(r.gf); nb > 0 {
		parts = append(parts, strconv.Itoa(nb)+" behind")
	}
	if r.gf.Ahead > 0 {
		parts = append(parts, strconv.Itoa(r.gf.Ahead)+" ahead")
	}
	switch {
	case len(parts) > 0:
		return strings.Join(parts, ", ")
	case r.gf.RemoteURL == "":
		return "no remote"
	case strings.TrimSpace(r.gf.RemoteChanges) == "":
		return "branch not on " + commons.RemoteName
	case stale(r.gf):
		return fetchAge(r.gf)
	}
	return "up to date"
}

func (m *model) rowLine(r *row, depth int, isCursor bool) string {
	st := stateOf(r)
	paint := func(style lipgloss.Style, text string) string {
		if isCursor && commons.NoColor {
			style = style.Reverse(true)
		} else if isCursor {
			style = style.Background(cursorColor)
		}
		return style.Render(text)
	}
	plain := lipgloss.NewStyle()

	mark := " "
	nameStyle := plain
	if st == stateClean || st == stateLoading {
		nameStyle = dimStyle
	}
	if r.selected {
		mark = "●"
		if commons.NoColor {
			mark = "*"
		}
		nameStyle = lipgloss.NewStyle().Foreground(selectColor).Bold(true)
	}

	text, textStyle := stateText(r), lipgloss.NewStyle().Foreground(stateColors[st])
	switch {
	case r.busy != "":
		text, textStyle = busyVerbs[r.busy]+"…", dimStyle
	case r.note != "" && r.noteOK:
		text, textStyle = r.note, okStyle
	case r.note != "":
		text, textStyle = r.note, koStyle
	}

	guides := m.guides(depth)
	nameW, withBranch := m.columns(guides)
	branch := ""
	if withBranch {
		branch = paint(dimStyle, fit(r.gf.CurrentBranch, branchW)+"  ")
	}

	return paint(dimStyle, " "+guides) +
		paint(lipgloss.NewStyle().Foreground(selectColor), mark) +
		paint(nameStyle, " "+fit(r.base, nameW)+"  ") +
		branch +
		paint(textStyle, fit(text, stateW))
}

func (m *model) detailLines() []string {
	name := m.currentFolder()
	if r := m.current(); r != nil {
		name = r.name
	}
	if name == "" {
		return nil
	}
	title := boldStyle
	if m.detailFocus {
		title = title.Foreground(selectColor)
	}
	return append([]string{title.Render(fitLeft(name, m.detailW))}, strings.Split(m.detail.View(), "\n")...)
}

// verdict tells what state the repository is in, and what can be done about it
func verdict(r *row) string {
	if r.gf.Err != "" {
		return koStyle.Render("This repository can't be read: " + r.gf.Err)
	}

	nbBehind, nbChanged := behind(r.gf), len(r.gf.ChangedFiles)
	// Each count has the color of the state it puts the repository in
	var parts []string
	part := func(st state, text string) {
		if len(parts) == 0 {
			text = strings.ToUpper(text[:1]) + text[1:]
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(stateColors[st]).Render(text))
	}
	if nbBehind > 0 {
		part(stateBehind, plural(nbBehind, "commit", "commits")+" behind "+commons.RemoteName)
	}
	if r.gf.Ahead > 0 {
		part(stateAhead, plural(r.gf.Ahead, "commit", "commits")+" not pushed")
	}
	if nbChanged > 0 {
		part(stateDirty, plural(nbChanged, "file", "files")+" changed")
	}

	var sentences []string
	switch {
	case len(parts) > 0:
		sentences = append(sentences, strings.Join(parts, ", ")+".")
	case r.gf.RemoteURL == "":
		sentences = append(sentences, "Nothing to commit. This repository has no "+commons.RemoteName+" remote.")
	case strings.TrimSpace(r.gf.RemoteChanges) == "":
		sentences = append(sentences, "Nothing to commit. This branch is not on "+commons.RemoteName+".")
	default:
		sentences = append(sentences, "Up to date with "+commons.RemoteName+", nothing to commit.")
	}

	switch {
	case nbBehind > 0 && r.gf.Ahead > 0:
		sentences = append(sentences, "Local and "+commons.RemoteName+" have diverged, merge or rebase from a shell.")
	case nbBehind > 0 && nbChanged > 0:
		sentences = append(sentences, "Pull is blocked until you commit or stash.")
	case nbBehind > 0:
		sentences = append(sentences, "Press p to pull.")
	}

	if stale(r.gf) {
		age := fetchAge(r.gf)
		sentences = append(sentences, dimStyle.Render(strings.ToUpper(age[:1])+age[1:]+", press f to check "+commons.RemoteName+"."))
	}
	return strings.Join(sentences, " ")
}

func (m *model) detailContent(r *row) string {
	var b strings.Builder
	section := func(title string, nb int) {
		b.WriteString("\n" + boldStyle.Render(title))
		if nb > 0 {
			b.WriteString(" " + dimStyle.Render(strconv.Itoa(nb)))
		}
		b.WriteString("\n")
	}
	// indented keeps a wrapped block of text apart from the command it is the output of
	indented := func(text string) {
		text = lipgloss.NewStyle().Width(m.detail.Width - 2).Render(text)
		for _, line := range strings.Split(text, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}

	if !r.loaded {
		return dimStyle.Render("Reading status…")
	}

	// Where the repository is: branch, remote and how fresh the knowledge of it is
	b.WriteString(boldStyle.Render(r.gf.CurrentBranch))
	if remoteW := m.detail.Width - lipgloss.Width(r.gf.CurrentBranch) - 2; r.gf.RemoteURL != "" && remoteW > 0 {
		b.WriteString("  " + dimStyle.Render(strings.TrimSpace(fit(shortRemote(r.gf.RemoteURL), remoteW))))
	}
	b.WriteString("\n\n")

	text := verdict(r)
	if age := fetchAge(r.gf); r.gf.RemoteURL != "" && !stale(r.gf) {
		text += " " + dimStyle.Render(strings.ToUpper(age[:1])+age[1:]+".")
	}
	if r.stashes > 0 {
		text += " " + plural(r.stashes, "stash", "stashes") + " kept aside."
	}
	if r.note != "" && !r.noteOK {
		text += "\n" + koStyle.Render(strings.ToUpper(r.note[:1])+r.note[1:])
	}
	b.WriteString(text + "\n")

	// What was just asked for comes before what is always there
	if len(r.cmds) > 0 {
		section("Commands", 0)
		for _, cmd := range r.cmds {
			if cmd.ExitCode == 0 {
				b.WriteString(okStyle.Render("✓ "+cmd.Cmd) + "\n")
			} else {
				b.WriteString(koStyle.Render("✗ "+cmd.Cmd) + dimStyle.Render("  exit code "+strconv.Itoa(cmd.ExitCode)) + "\n")
			}
			if output := cleanOutput(cmd.Output); output != "" {
				indented(output)
			}
		}
	}

	if len(r.gf.ChangedFiles) > 0 {
		section("Changed files", len(r.gf.ChangedFiles))
		for _, file := range r.gf.ChangedFiles {
			b.WriteString(changedFile(file) + "\n")
		}
	}

	commits := func(title string, nb int, st state, commits []string) {
		if len(commits) == 0 {
			return
		}
		section(title, nb)
		for _, commit := range commits {
			fields := strings.SplitN(commit, "\t", 3)
			if len(fields) != 3 {
				b.WriteString(commit + "\n")
				continue
			}
			// One line per commit: the subject gives way to the hash and the age
			hash, age := fields[0], fields[1]
			subjectW := m.detail.Width - lipgloss.Width(hash) - lipgloss.Width(age) - 3
			if subjectW < 10 {
				age, subjectW = "", m.detail.Width-lipgloss.Width(hash)-1
			}
			b.WriteString(lipgloss.NewStyle().Foreground(stateColors[st]).Render(hash) + " " + fit(fields[2], subjectW) + "  " + dimStyle.Render(age) + "\n")
		}
	}
	commits("Incoming", len(r.incoming), stateBehind, r.incoming)
	commits("Not pushed", len(r.outgoing), stateAhead, r.outgoing)

	// A single branch is the one already told above
	if len(r.branches) > 1 {
		section("Branches", len(r.branches))
		// The lines above may be wrapped, the branches are below what they become
		m.branchTop = strings.Count(lipgloss.NewStyle().Width(m.detail.Width).Render(b.String()), "\n")
		for i, branch := range r.branches {
			b.WriteString(m.branchLine(branch, m.detailFocus && i == m.branchCursor) + "\n")
		}
	}

	commits("Last commits", 0, stateClean, r.recent)
	return b.String()
}

// branchLine tells where a branch stands compared to the one it tracks, in the color
// of the state it would put the repository in, and how old its last commit is
func (m *model) branchLine(branch entity.Branch, isCursor bool) string {
	paint := func(style lipgloss.Style, text string) string {
		if isCursor && commons.NoColor {
			style = style.Reverse(true)
		} else if isCursor {
			style = style.Background(cursorColor)
		}
		return style.Render(text)
	}

	text, st := "up to date", stateClean
	switch {
	case branch.Upstream == "":
		text, st = "not on "+commons.RemoteName, stateAhead
	case branch.Gone:
		text = "gone from " + commons.RemoteName
	case branch.Ahead > 0 && branch.Behind > 0:
		text, st = strconv.Itoa(branch.Ahead)+" ahead, "+strconv.Itoa(branch.Behind)+" behind", stateBroken
	case branch.Behind > 0:
		text, st = strconv.Itoa(branch.Behind)+" behind", stateBehind
	case branch.Ahead > 0:
		text, st = strconv.Itoa(branch.Ahead)+" ahead", stateAhead
	}

	// As git does, a star marks the branch the repository is on
	mark, nameStyle := "  ", lipgloss.NewStyle()
	if branch.Current {
		mark, nameStyle = "* ", boldStyle
	}

	// The age gives way first, then the name
	nameW, age := m.detail.Width-2-2-branchStateW, ""
	if nameW-branchAgeW-2 >= 12 {
		nameW -= branchAgeW + 2
		age = paint(dimStyle, "  "+fit(branch.Age, branchAgeW))
	}
	return paint(nameStyle, mark+fit(branch.Name, nameW)+"  ") +
		paint(lipgloss.NewStyle().Foreground(stateColors[st]), fit(text, branchStateW)) +
		age
}

// shortRemote keeps the host and the path of a remote URL
func shortRemote(remoteURL string) string {
	remote := strings.TrimSuffix(remoteURL, ".git")
	if i := strings.Index(remote, "://"); i >= 0 {
		remote = remote[i+3:]
	} else if at := strings.Index(remote, "@"); at >= 0 {
		// scp-like syntax, user@host:path
		remote = strings.Replace(remote, ":", "/", 1)
	}
	if at := strings.Index(remote, "@"); at >= 0 {
		remote = remote[at+1:]
	}
	return remote
}

// changedFile turns a line of git status --porcelain into words, colored as the
// state they put the repository in: local work to commit, or work ready to be.
func changedFile(line string) string {
	if len(line) < 4 {
		return line
	}
	index, worktree, path := line[0], line[1], strings.Trim(line[3:], "\"")

	words := map[byte]string{'M': "modified", 'T': "modified", 'A': "added", 'D': "deleted", 'R': "renamed", 'C': "copied"}
	word, style := "changed", lipgloss.NewStyle().Foreground(stateColors[stateDirty])
	switch {
	case index == 'U' || worktree == 'U' || (index == worktree && (index == 'A' || index == 'D')):
		word, style = "conflict", koStyle
	case index == '?':
		word, style = "untracked", dimStyle
	case worktree != ' ':
		if w, ok := words[worktree]; ok {
			word = w
		}
	default:
		// Everything is in the index
		word, style = "staged", okStyle
		if w, ok := words[index]; ok && index != 'M' {
			word = w
		}
	}

	dir, file := "", path
	if i := strings.LastIndex(strings.TrimSuffix(path, "/"), "/"); i >= 0 {
		dir, file = path[:i+1], path[i+1:]
	}
	return style.Render(fit(word, 10)) + dimStyle.Render(dir) + file
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
