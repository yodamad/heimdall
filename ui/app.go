package ui

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/utils"
)

type mode int

const (
	modeList mode = iota
	modeFilter
	modeCommand
	modeConfirm
	modeHelp
)

const (
	busyStatus  = "status" // status read after an action, not an action by itself
	busyRefresh = "refresh"
	busyFetch   = "fetch"
	busyPull    = "pull"
	busySwitch  = "switch"
	busyCmd     = "cmd"
)

type row struct {
	gf       entity.GitFolder
	name     string // path relative to the root directory
	dir      string // folder of the repository in name, empty at the root
	base     string
	loaded   bool
	busy     string
	selected bool
	note     string // result of the last action
	noteOK   bool
	incoming []string
	outgoing []string
	recent   []string // last commits of the current branch
	branches []entity.Branch
	stashes  int
	cmds     []entity.CmdInfo
}

const sep = string(filepath.Separator)

// listLine is a line of the list: a repository, or a folder of the tree they are in
type listLine struct {
	r      *row
	folder string // path of the folder, relative to the root directory
	label  string // name of the folder, after the ones of the folders it is alone in
	parent string // folder the line is in, empty at the root
	depth  int
}

// treeNode is a folder while the tree is built
type treeNode struct {
	path    string
	folders []*treeNode
	rows    []*row
}

type pendingAction struct {
	question string
	run      func() tea.Cmd
}

type model struct {
	root  string
	depth int

	rows      []*row
	byPath    map[string]*row
	visible   []*row // rows matching the filter
	lines     []listLine
	folders   map[string]listLine // folders of the tree, listed or hidden in a collapsed one
	collapsed map[string]bool     // folders whose content is hidden
	cursor    int
	offset    int

	mode        mode
	discovering bool
	detailFocus bool
	detailPath  string
	// the branch of the current repository chosen in the details, the line of the
	// first one in the details, and if the chosen one has to be scrolled to
	branchCursor int
	branchTop    int
	followBranch bool
	pending      *pendingAction
	status       string
	sorted       bool
	morning      []string // commands of the morning routine

	// progress of the running actions
	batchKind  string
	batchTotal int

	filter   textinput.Model
	cmdInput textinput.Model
	detail   viewport.Model
	spin     spinner.Model

	width, height int
	paneH         int
	listW         int
	detailW       int
}

// Run starts the TUI on the git folders found under root
func Run(root string, depth int) error {
	if commons.NoColor {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	_, err := tea.NewProgram(newModel(root, depth), tea.WithAltScreen()).Run()
	return err
}

func newModel(root string, depth int) *model {
	filter := textinput.New()
	filter.Prompt = "/ "
	cmdInput := textinput.New()
	cmdInput.Prompt = "! "
	cmdInput.Placeholder = "command to run"

	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = dimStyle

	var morning []string
	for _, cmd := range utils.GetMorningRoutine().Cmds {
		if cmd = strings.TrimSpace(cmd); cmd != "" {
			morning = append(morning, cmd)
		}
	}

	return &model{
		morning:     morning,
		root:        root,
		depth:       depth,
		byPath:      map[string]*row{},
		collapsed:   map[string]bool{},
		discovering: true,
		filter:      filter,
		cmdInput:    cmdInput,
		detail:      viewport.New(0, 0),
		spin:        spin,
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, discoverCmd(m.root, m.depth))
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case spinner.TickMsg:
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()

	case discoveredMsg:
		m.discovering = false
		var cmds []tea.Cmd
		for _, path := range msg.paths {
			r := &row{gf: entity.GitFolder{Path: path}, name: m.relName(path), busy: busyRefresh}
			r.base, r.dir = filepath.Base(r.name), parentDir(r.name)
			m.rows = append(m.rows, r)
			m.byPath[path] = r
			cmds = append(cmds, statusCmd(path))
		}
		m.batchKind, m.batchTotal = busyRefresh, len(m.rows)
		m.sortRows()
		m.applyFilter()
		cmd = tea.Batch(cmds...)

	case statusMsg:
		if r, ok := m.byPath[msg.gf.Path]; ok {
			r.gf = msg.gf
			r.incoming = msg.incoming
			r.outgoing = msg.outgoing
			r.recent = msg.recent
			r.branches = msg.branches
			r.stashes = msg.stashes
			r.loaded = true
			r.busy = ""
		}
		m.resort()

	case opMsg:
		if r, ok := m.byPath[msg.path]; ok {
			if msg.err != nil {
				r.note, r.noteOK = msg.kind+" failed: "+msg.err.Error(), false
			}
			r.busy = busyStatus
			cmd = statusCmd(msg.path)
		}

	case cmdMsg:
		if r, ok := m.byPath[msg.path]; ok {
			r.cmds = msg.results
			failed := 0
			for _, c := range msg.results {
				if c.ExitCode != 0 {
					failed++
				}
			}
			r.noteOK = failed == 0
			switch {
			case !r.noteOK && len(msg.results) == 1:
				r.note = msg.results[0].Cmd + " failed"
			case !r.noteOK:
				r.note = strconv.Itoa(failed) + " of " + strconv.Itoa(len(msg.results)) + " commands failed"
			case len(msg.results) == 1:
				r.note = "ran " + msg.results[0].Cmd
			default:
				r.note = "ran " + strconv.Itoa(len(msg.results)) + " commands"
			}
			r.busy = busyStatus
			cmd = statusCmd(msg.path)
		}

	case tea.KeyMsg:
		cmd = m.handleKey(msg)
	}
	m.syncDetail()
	return m, cmd
}

func (m *model) relName(path string) string {
	rel, err := filepath.Rel(filepath.Clean(m.root), path)
	if err != nil || rel == "." {
		return filepath.Base(path)
	}
	return rel
}

func (m *model) handleKey(key tea.KeyMsg) tea.Cmd {
	if key.String() == "ctrl+c" {
		return tea.Quit
	}

	switch m.mode {
	case modeFilter:
		switch key.String() {
		case "enter":
			m.filter.Blur()
			m.mode = modeList
		case "esc":
			m.filter.SetValue("")
			m.filter.Blur()
			m.mode = modeList
			m.applyFilter()
		default:
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(key)
			m.applyFilter()
			return cmd
		}
		return nil

	case modeCommand:
		switch key.String() {
		case "enter":
			cmd := strings.TrimSpace(m.cmdInput.Value())
			m.cmdInput.SetValue("")
			m.cmdInput.Blur()
			m.mode = modeList
			if cmd != "" {
				return m.runCommands([]string{cmd})
			}
		case "esc":
			m.cmdInput.SetValue("")
			m.cmdInput.Blur()
			m.mode = modeList
		default:
			var cmd tea.Cmd
			m.cmdInput, cmd = m.cmdInput.Update(key)
			return cmd
		}
		return nil

	case modeConfirm:
		pending := m.pending
		m.pending = nil
		m.mode = modeList
		if key.String() == "y" || key.String() == "Y" || key.String() == "enter" {
			return pending.run()
		}
		m.status = "Cancelled"
		return nil

	case modeHelp:
		m.mode = modeList
		return nil
	}

	if m.detailFocus {
		branches := m.branchChoices()
		switch key.String() {
		case "tab", "esc":
			m.detailFocus = false
		case "q":
			return tea.Quit
		case "up", "k", "down", "j":
			if len(branches) == 0 {
				var cmd tea.Cmd
				m.detail, cmd = m.detail.Update(key)
				return cmd
			}
			if key.String() == "up" || key.String() == "k" {
				m.branchCursor--
			} else {
				m.branchCursor++
			}
			m.branchCursor = max(0, min(m.branchCursor, len(branches)-1))
			m.followBranch = true
		case "enter":
			return m.switchBranch()
		default:
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(key)
			return cmd
		}
		return nil
	}

	m.status = ""
	switch key.String() {
	case "q":
		return tea.Quit
	case "?":
		m.mode = modeHelp
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "pgup":
		m.moveCursor(-m.listRows())
	case "pgdown":
		m.moveCursor(m.listRows())
	case "home", "g":
		m.moveCursor(-len(m.lines))
	case "end", "G":
		m.moveCursor(len(m.lines))
	case "enter":
		if folder := m.currentFolder(); folder != "" {
			m.collapse(folder, !m.collapsed[folder])
		}
	case "left", "h":
		// As in a file navigator: an open folder is collapsed, anything else leads to its folder
		if folder := m.currentFolder(); folder != "" && !m.isCollapsed(folder) {
			m.collapse(folder, true)
		} else {
			m.goTo(m.currentLine().parent)
		}
	case "right", "l":
		// A collapsed folder is expanded, an open one is entered
		if folder := m.currentFolder(); m.isCollapsed(folder) {
			m.collapse(folder, false)
		} else if folder != "" {
			m.moveCursor(1)
		}
	case "z":
		if m.filter.Value() != "" {
			m.status = "Folders stay open while the list is filtered, esc to list all"
			break
		}
		// Collapse all the folders, or expand them all when they already are
		all := true
		for folder := range m.folders {
			all = all && m.collapsed[folder]
		}
		for folder := range m.folders {
			m.collapsed[folder] = !all
		}
		m.applyFilter()
	case "tab":
		m.detailFocus = len(m.lines) > 0
		m.branchCursor, m.followBranch = 0, true
	case "/":
		m.mode = modeFilter
		m.filter.Focus()
	case "esc":
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.applyFilter()
		} else {
			for _, r := range m.rows {
				r.selected = false
			}
		}
	case " ":
		if r := m.current(); r != nil {
			r.selected = !r.selected
			m.moveCursor(1)
		} else if rows := m.folderRows(m.currentFolder()); len(rows) > 0 {
			// On a folder, all its repositories at once
			all := true
			for _, r := range rows {
				all = all && r.selected
			}
			for _, r := range rows {
				r.selected = !all
			}
		}
	case "a":
		all := true
		for _, r := range m.visible {
			all = all && r.selected
		}
		for _, r := range m.visible {
			r.selected = !all
		}
	case "u":
		nb := 0
		for _, r := range m.rows {
			r.selected = false
		}
		for _, r := range m.visible {
			if r.loaded && entity.CanPull(r.gf) {
				r.selected = true
				nb++
			}
		}
		m.status = plural(nb, "repository", "repositories") + " can be pulled"
	case "r":
		return m.start(busyRefresh, func(r *row) tea.Cmd { return statusCmd(r.gf.Path) })
	case "f":
		return m.start(busyFetch, func(r *row) tea.Cmd { return fetchCmd(r.gf.Path) })
	case "p":
		return m.confirm("Pull", func() tea.Cmd {
			return m.start(busyPull, func(r *row) tea.Cmd {
				if !r.loaded || r.gf.Err != "" {
					return nil
				}
				if r.gf.HasLocalChanges {
					r.note, r.noteOK = "pull skipped, local changes", false
					return nil
				}
				return pullCmd(r.gf.Path)
			})
		})
	case "m":
		if len(m.morning) == 0 {
			m.status = "No morning routine yet. Set morning_routine.commands in the config file, or press ! to run a command"
			return nil
		}
		return m.runCommands(m.morning)
	case "!":
		if len(m.targets()) > 0 {
			m.mode = modeCommand
			m.cmdInput.Focus()
		}
	}
	return nil
}

// targets are the selected repositories or, if none, the one under the cursor
// or all the ones of the folder under the cursor
func (m *model) targets() []*row {
	var targets []*row
	for _, r := range m.rows {
		if r.selected {
			targets = append(targets, r)
		}
	}
	if len(targets) == 0 && m.current() != nil {
		targets = append(targets, m.current())
	} else if len(targets) == 0 {
		targets = m.folderRows(m.currentFolder())
	}
	return targets
}

// branchChoices are the branches which can be chosen in the details: the ones of the
// current repository, when it has more than the one it is on
func (m *model) branchChoices() []entity.Branch {
	if r := m.current(); r != nil && r.loaded && len(r.branches) > 1 {
		return r.branches
	}
	return nil
}

// switchBranch puts the current repository on the branch chosen in the details
func (m *model) switchBranch() tea.Cmd {
	r, branches := m.current(), m.branchChoices()
	if m.branchCursor >= len(branches) || branches[m.branchCursor].Current || r.busy != "" {
		return nil
	}
	if m.running() == 0 {
		m.batchTotal = 0
	}
	m.batchKind = busySwitch
	m.batchTotal++
	r.note, r.busy = "", busySwitch
	cmd := switchCmd(r.gf.Path, branches[m.branchCursor].Name)
	// The branch the repository is on comes first
	m.branchCursor, m.followBranch = 0, true
	return cmd
}

// start launches an action on each target which is not already busy
func (m *model) start(kind string, action func(r *row) tea.Cmd) tea.Cmd {
	if m.running() == 0 {
		m.batchTotal = 0
	}
	var cmds []tea.Cmd
	for _, r := range m.targets() {
		if r.busy != "" {
			continue
		}
		r.note = ""
		if cmd := action(r); cmd != nil {
			r.busy = kind
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) > 0 {
		m.batchKind = kind
		m.batchTotal += len(cmds)
	}
	return tea.Batch(cmds...)
}

// confirm asks for a confirmation when an action targets several repositories
func (m *model) confirm(label string, run func() tea.Cmd) tea.Cmd {
	nb := len(m.targets())
	if nb <= 1 {
		return run()
	}
	m.pending = &pendingAction{
		question: label + " on " + strconv.Itoa(nb) + " repositories? [y/N]",
		run:      run,
	}
	m.mode = modeConfirm
	return nil
}

func (m *model) runCommands(cmds []string) tea.Cmd {
	return m.confirm("Run '"+strings.Join(cmds, "', '")+"'", func() tea.Cmd {
		return m.start(busyCmd, func(r *row) tea.Cmd { return execCmd(r.gf, cmds) })
	})
}

// running counts the repositories with an action in progress
func (m *model) running() int {
	nb := 0
	for _, r := range m.rows {
		if r.busy != "" && r.busy != busyStatus {
			nb++
		}
	}
	return nb
}

// resort puts the repositories needing attention first. It waits for the running
// actions to be done, so that rows do not move while they are being updated.
func (m *model) resort() {
	for _, r := range m.rows {
		if r.busy != "" {
			return
		}
	}
	m.sortRows()
	m.applyFilter()
	if !m.sorted {
		m.sorted = true
		m.firstRow()
	}
}

// parentDir is the folder a folder or a repository is in, empty at the root
func parentDir(path string) string {
	if dir := filepath.Dir(path); dir != "." {
		return dir
	}
	return ""
}

// dirLess orders the folders as a tree: a folder comes with what it holds, and the
// folders it holds come before its own repositories
func dirLess(a, b string) bool {
	split := func(dir string) []string {
		return strings.FieldsFunc(dir, func(r rune) bool { return r == filepath.Separator })
	}
	as, bs := split(a), split(b)
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return as[i] < bs[i]
		}
	}
	return len(as) > len(bs)
}

// sortRows puts the repositories in the order of the tree, the ones needing attention
// first in their folder
func (m *model) sortRows() {
	sort.SliceStable(m.rows, func(i, j int) bool {
		ri, rj := m.rows[i], m.rows[j]
		if ri.dir != rj.dir {
			return dirLess(ri.dir, rj.dir)
		}
		if si, sj := stateOf(ri), stateOf(rj); si != sj {
			return si < sj
		}
		return ri.base < rj.base
	})
}

// currentLine is the line of the list the cursor is on: a repository or a folder
func (m *model) currentLine() listLine {
	if m.cursor < 0 || m.cursor >= len(m.lines) {
		return listLine{}
	}
	return m.lines[m.cursor]
}

func (m *model) current() *row {
	return m.currentLine().r
}

func (m *model) currentFolder() string {
	return m.currentLine().folder
}

// folderRows are the listed repositories of a folder and of the folders it holds
func (m *model) folderRows(folder string) []*row {
	var rows []*row
	for _, r := range m.visible {
		if r.dir == folder || strings.HasPrefix(r.dir, folder+sep) {
			rows = append(rows, r)
		}
	}
	return rows
}

// isCollapsed tells if the content of a folder is hidden. A filter opens all the folders.
func (m *model) isCollapsed(folder string) bool {
	return m.collapsed[folder] && m.filter.Value() == ""
}

// goTo puts the cursor on a folder, if it is listed
func (m *model) goTo(folder string) bool {
	for i, line := range m.lines {
		if folder != "" && line.folder == folder {
			m.cursor = i
			m.moveCursor(0)
			return true
		}
	}
	return false
}

// collapse hides or shows the content of a folder
func (m *model) collapse(folder string, collapsed bool) {
	if folder == "" {
		return
	}
	if m.filter.Value() != "" {
		m.status = "Folders stay open while the list is filtered, esc to list all"
		return
	}
	m.collapsed[folder] = collapsed
	m.applyFilter()
}

// firstRow puts the cursor on the first repository of the list
func (m *model) firstRow() {
	m.cursor, m.offset = 0, 0
	for i, line := range m.lines {
		if line.r != nil {
			m.cursor = i
			break
		}
	}
	m.moveCursor(0)
}

// moveCursor moves the cursor by delta lines
func (m *model) moveCursor(delta int) {
	m.cursor = max(0, min(m.cursor+delta, len(m.lines)-1))

	// The line above the cursor is kept in sight: it is the folder of the line,
	// or the line this folder is pinned on while its content is scrolled
	rows := m.listRows()
	if m.cursor <= m.offset {
		m.offset = max(0, m.cursor-1)
	} else if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
	// No room is left empty under a list which got shorter
	m.offset = max(0, min(m.offset, len(m.lines)-rows))
}

func (m *model) applyFilter() {
	current := m.currentLine()
	filter := strings.ToLower(m.filter.Value())
	m.visible = m.visible[:0]
	for _, r := range m.rows {
		if filter == "" || strings.Contains(strings.ToLower(r.name), filter) {
			m.visible = append(m.visible, r)
		}
	}

	m.buildLines()

	// The cursor stays where it was, or goes to the closest folder its line is now hidden in
	folder := current.folder
	if current.r != nil {
		folder = current.r.dir
	}
	for i, line := range m.lines {
		if current.r != nil && line.r == current.r {
			m.cursor = i
			m.moveCursor(0)
			return
		}
	}
	for ; folder != ""; folder = parentDir(folder) {
		if m.goTo(folder) {
			return
		}
	}
	m.firstRow()
}

// buildLines lists the repositories as the tree of the folders they are in. A folder
// holding nothing but another folder shares its line, a collapsed one hides what it holds.
func (m *model) buildLines() {
	// The repositories are sorted in the order of the tree, so are the folders found
	root := &treeNode{}
	nodes := map[string]*treeNode{"": root}
	var nodeOf func(dir string) *treeNode
	nodeOf = func(dir string) *treeNode {
		if n, ok := nodes[dir]; ok {
			return n
		}
		n, parent := &treeNode{path: dir}, nodeOf(parentDir(dir))
		parent.folders = append(parent.folders, n)
		nodes[dir] = n
		return n
	}
	for _, r := range m.visible {
		n := nodeOf(r.dir)
		n.rows = append(n.rows, r)
	}

	m.lines, m.folders = m.lines[:0], map[string]listLine{}
	var list func(n *treeNode, depth int, hidden bool)
	list = func(n *treeNode, depth int, hidden bool) {
		for _, f := range n.folders {
			for len(f.rows) == 0 && len(f.folders) == 1 {
				f = f.folders[0]
			}
			line := listLine{folder: f.path, label: strings.TrimPrefix(f.path, n.path+sep), parent: n.path, depth: depth}
			m.folders[f.path] = line
			if !hidden {
				m.lines = append(m.lines, line)
			}
			list(f, depth+1, hidden || m.isCollapsed(f.path))
		}
		for _, r := range n.rows {
			if !hidden {
				m.lines = append(m.lines, listLine{r: r, parent: n.path, depth: depth})
			}
		}
	}
	list(root, 0, false)
}

// narrow terminals display only one pane at a time
func (m *model) narrow() bool {
	return m.width < 80
}

func (m *model) listRows() int {
	return m.paneH
}

func (m *model) layout() {
	// header, bridge and its margin, status and keys lines
	m.paneH = m.height - 5
	if m.paneH < 3 {
		m.paneH = 3
	}
	if m.narrow() {
		m.listW, m.detailW = m.width, m.width-2
	} else {
		m.listW = m.width * 58 / 100
		m.detailW = m.width - m.listW - dividerW
	}
	m.detail.Width = m.detailW
	// title
	m.detail.Height = m.paneH - 1
	m.filter.Width = m.width - 10
	m.cmdInput.Width = m.width - 10
	m.detailPath = ""
	m.moveCursor(0)
}

func (m *model) syncDetail() {
	if m.width == 0 {
		return
	}
	content, path := "", ""
	if r := m.current(); r != nil {
		path = r.gf.Path
	} else if folder := m.currentFolder(); folder != "" {
		content, path = m.folderContent(folder), folder+"/"
	} else {
		m.detailFocus = false
	}
	if m.detailPath != path {
		m.branchCursor = 0
	}
	m.branchCursor = max(0, min(m.branchCursor, len(m.branchChoices())-1))
	m.branchTop = -1
	if r := m.current(); r != nil {
		// Built once the chosen branch is known to be one of the repository
		content = m.detailContent(r)
	}
	m.detail.SetContent(lipgloss.NewStyle().Width(m.detail.Width).Render(content))
	if m.detailPath != path {
		m.detailPath = path
		m.detail.GotoTop()
	}

	// The chosen branch stays in sight, with the title of the branches when it is the first
	if line := m.branchTop + m.branchCursor; m.followBranch && m.detailFocus && m.branchTop >= 0 {
		if top := max(0, line-1); top < m.detail.YOffset {
			m.detail.SetYOffset(top)
		} else if line >= m.detail.YOffset+m.detail.Height {
			m.detail.SetYOffset(line - m.detail.Height + 1)
		}
	}
	m.followBranch = false
}
