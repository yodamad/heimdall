package ui

import (
	"path/filepath"
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
	busyStatus = "status"
	busyFetch  = "fetch"
	busyPull   = "pull"
	busyCmd    = "cmd"
)

type row struct {
	gf       entity.GitFolder
	name     string // path relative to the root directory
	loaded   bool
	busy     string
	fetched  bool // fetched during this session, remote changes are reliable
	selected bool
	note     string // result of the last action
	noteOK   bool
	incoming []string
	cmds     []entity.CmdInfo
}

type pendingAction struct {
	question string
	run      func() tea.Cmd
}

type model struct {
	root  string
	depth int

	rows    []*row
	byPath  map[string]*row
	visible []*row // rows matching the filter
	cursor  int
	offset  int

	mode        mode
	discovering bool
	detailFocus bool
	detailPath  string
	pending     *pendingAction
	status      string

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
	spin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))

	return &model{
		root:        root,
		depth:       depth,
		byPath:      map[string]*row{},
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
			r := &row{gf: entity.GitFolder{Path: path}, name: m.relName(path), busy: busyStatus}
			m.rows = append(m.rows, r)
			m.byPath[path] = r
			cmds = append(cmds, statusCmd(path))
		}
		m.applyFilter()
		cmd = tea.Batch(cmds...)

	case statusMsg:
		if r, ok := m.byPath[msg.gf.Path]; ok {
			r.gf = msg.gf
			r.incoming = msg.incoming
			r.loaded = true
			r.busy = ""
		}

	case opMsg:
		if r, ok := m.byPath[msg.path]; ok {
			if msg.err != nil {
				r.note, r.noteOK = msg.kind+" failed : "+msg.err.Error(), false
			} else {
				r.fetched = true
				r.note, r.noteOK = msg.kind+" done", true
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
			if r.noteOK {
				r.note = strconv.Itoa(len(msg.results)) + " command(s) succeeded"
			} else {
				r.note = strconv.Itoa(failed) + "/" + strconv.Itoa(len(msg.results)) + " command(s) failed"
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
		switch key.String() {
		case "tab", "esc":
			m.detailFocus = false
		case "q":
			return tea.Quit
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
		m.moveCursor(-len(m.visible))
	case "end", "G":
		m.moveCursor(len(m.visible))
	case "tab":
		m.detailFocus = m.current() != nil
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
		m.status = strconv.Itoa(nb) + " repositories can be pulled"
	case "r":
		return m.start(busyStatus, func(r *row) tea.Cmd { return statusCmd(r.gf.Path) })
	case "f":
		return m.start(busyFetch, func(r *row) tea.Cmd { return fetchCmd(r.gf.Path) })
	case "p":
		return m.confirm("Pull", func() tea.Cmd {
			return m.start(busyPull, func(r *row) tea.Cmd {
				if !r.loaded || r.gf.Err != "" {
					return nil
				}
				if r.gf.HasLocalChanges {
					r.note, r.noteOK = "pull skipped : local changes", false
					return nil
				}
				return pullCmd(r.gf.Path)
			})
		})
	case "m":
		var cmds []string
		for _, cmd := range utils.GetMorningRoutine().Cmds {
			if cmd = strings.TrimSpace(cmd); cmd != "" {
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) == 0 {
			m.status = "No morning routine configured (morning_routine.commands in config file), use ! to run a command"
			return nil
		}
		return m.runCommands(cmds)
	case "!":
		if m.current() != nil {
			m.mode = modeCommand
			m.cmdInput.Focus()
		}
	}
	return nil
}

// targets are the selected repositories or, if none, the one under the cursor
func (m *model) targets() []*row {
	var targets []*row
	for _, r := range m.rows {
		if r.selected {
			targets = append(targets, r)
		}
	}
	if len(targets) == 0 && m.current() != nil {
		targets = append(targets, m.current())
	}
	return targets
}

// start launches an action on each target which is not already busy
func (m *model) start(kind string, action func(r *row) tea.Cmd) tea.Cmd {
	var cmds []tea.Cmd
	for _, r := range m.targets() {
		if r.busy != "" {
			continue
		}
		if cmd := action(r); cmd != nil {
			r.busy = kind
			cmds = append(cmds, cmd)
		}
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
		question: label + " on " + strconv.Itoa(nb) + " repositories ? [y/N]",
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

func (m *model) current() *row {
	if m.cursor < 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return m.visible[m.cursor]
}

func (m *model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor > len(m.visible)-1 {
		m.cursor = len(m.visible) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	rows := m.listRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	} else if m.cursor >= m.offset+rows {
		m.offset = m.cursor - rows + 1
	}
}

func (m *model) applyFilter() {
	current := m.current()
	filter := strings.ToLower(m.filter.Value())
	m.visible = m.visible[:0]
	m.cursor = 0
	for _, r := range m.rows {
		if filter == "" || strings.Contains(strings.ToLower(r.name), filter) {
			if r == current {
				m.cursor = len(m.visible)
			}
			m.visible = append(m.visible, r)
		}
	}
	m.offset = 0
	m.moveCursor(0)
}

// narrow terminals display only one pane at a time
func (m *model) narrow() bool {
	return m.width < 80
}

func (m *model) listRows() int {
	// borders and columns header
	if rows := m.paneH - 3; rows > 1 {
		return rows
	}
	return 1
}

func (m *model) layout() {
	// header, status and help lines
	m.paneH = m.height - 3
	if m.paneH < 5 {
		m.paneH = 5
	}
	if m.narrow() {
		m.listW, m.detailW = m.width, m.width
	} else {
		m.listW = m.width * 55 / 100
		m.detailW = m.width - m.listW
	}
	// borders and padding
	m.detail.Width = m.detailW - 4
	// borders and title
	m.detail.Height = m.paneH - 3
	m.filter.Width = m.width - 10
	m.cmdInput.Width = m.width - 10
	m.detailPath = ""
	m.moveCursor(0)
}

func (m *model) syncDetail() {
	if m.width == 0 {
		return
	}
	r := m.current()
	if r == nil {
		m.detail.SetContent("")
		m.detailPath = ""
		m.detailFocus = false
		return
	}
	m.detail.SetContent(lipgloss.NewStyle().Width(m.detail.Width).Render(m.detailContent(r)))
	if m.detailPath != r.gf.Path {
		m.detailPath = r.gf.Path
		m.detail.GotoTop()
	}
}
