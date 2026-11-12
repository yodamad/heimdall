package ui

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/activity"
	"github.com/yodamad/heimdall/commons"
)

// How often what the workers recorded is moved onto the model. The workers write into a store of
// their own, and only Update reads it, so the model is never touched from another goroutine.
const activityPeriod = 200 * time.Millisecond

// Max number of activity lines kept on the model
const maxLogLines = 500

// Width of the repository column of the activity pane
const logScopeW = 18

type activityTickMsg time.Time

func activityTickCmd() tea.Cmd {
	return tea.Tick(activityPeriod, func(t time.Time) tea.Msg { return activityTickMsg(t) })
}

// logLine is one line of the activity pane
type logLine struct {
	at    time.Time
	level activity.Level
	scope string
	msg   string
}

// pullActivity moves what was recorded since the last call onto the model. It is called from
// Update, inside the single loop of bubbletea, so nothing is shared with the workers but the store.
func (m *model) pullActivity() {
	entries, seq := m.log.Since(m.logSeq)
	if len(entries) == 0 {
		m.logSeq = seq
		return
	}

	// The store is a ring : a jump in the sequence means the lines in between were overwritten
	// while nothing was reading them
	if dropped := entries[0].Seq - m.logSeq - 1; m.logSeq > 0 && dropped > 0 {
		m.logLines = append(m.logLines, logLine{
			at:    entries[0].At,
			level: activity.Debug,
			msg:   "… " + strconv.FormatUint(dropped, 10) + " lines dropped",
		})
	}

	for _, e := range entries {
		m.logLines = append(m.logLines, logLine{at: e.At, level: e.Level, scope: e.Scope, msg: e.Msg})
	}
	if len(m.logLines) > maxLogLines {
		m.logLines = append([]logLine(nil), m.logLines[len(m.logLines)-maxLogLines:]...)
	}
	m.logSeq = seq
	m.syncLogView()
}

// shown says if the line is displayed : the debug ones are only when --verbose was asked for
func (l logLine) shown() bool {
	return l.level != activity.Debug || commons.Verbose
}

// openLogs shows the activity pane, on its last lines
func (m *model) openLogs() tea.Cmd {
	m.mode = modeLogs
	m.logFollow = true
	m.syncLogView()
	m.logView.GotoBottom()
	return nil
}

func (m *model) handleLogsKey(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc", "L", "q":
		m.mode = modeList
		return nil
	case "G", "end":
		m.logFollow = true
		m.logView.GotoBottom()
		return nil
	}

	var cmd tea.Cmd
	m.logView, cmd = m.logView.Update(key)
	// Scrolling away stops following the last lines, coming back to the bottom follows again
	m.logFollow = m.logView.AtBottom()
	return cmd
}

// syncLogView rebuilds the content of the pane, and keeps it on the last lines while it follows
func (m *model) syncLogView() {
	if m.logView.Width <= 0 {
		return
	}

	var lines []string
	for _, l := range m.logLines {
		if l.shown() {
			lines = append(lines, m.logLineView(l, m.logView.Width))
		}
	}
	m.logView.SetContent(strings.Join(lines, "\n"))
	if m.logFollow {
		m.logView.GotoBottom()
	}
}

// glyphs tell the levels apart without a color, which --no-color needs
var glyphs = map[activity.Level]string{
	activity.Debug: "·",
	activity.Info:  "·",
	activity.Warn:  "⚠",
	activity.Error: "✖",
}

// logLineView renders one line, giving up the time then the repository on a narrow terminal
func (m *model) logLineView(l logLine, width int) string {
	msg := glyphs[l.level] + " " + l.msg

	// A line is plain by default, so that it reads as the normal state of things
	style := lipgloss.NewStyle()
	switch l.level {
	case activity.Debug:
		style = dimStyle
	case activity.Warn, activity.Error:
		style = warnStyle
	}

	switch {
	case width >= 46:
		return dimStyle.Render(l.at.Format("15:04:05")+" ") +
			dimStyle.Render(fitLeft(l.scope, logScopeW)+" ") +
			style.Render(fit(msg, width-8-1-logScopeW-1))
	case width >= 30:
		return dimStyle.Render(fitLeft(l.scope, logScopeW)+" ") + style.Render(fit(msg, width-logScopeW-1))
	default:
		return style.Render(fit(msg, width))
	}
}

// logPane is the body of the activity pane
func (m *model) logPane() []string {
	nb := 0
	for _, l := range m.logLines {
		if l.shown() {
			nb++
		}
	}

	title := "Activity · " + plural(nb, "line", "lines")
	if !m.logFollow {
		title += ", G to follow again"
	}
	lines := []string{boldStyle.Render(fit(" "+title, m.width-1))}

	if nb == 0 {
		return append(lines, dimStyle.Render(" Nothing has happened yet."))
	}
	for _, line := range strings.Split(m.logView.View(), "\n") {
		lines = append(lines, " "+line)
	}
	return lines
}

// logFooter is the last thing which happened, shown under the list while actions run. The header
// keeps the count of what is done, so it is not repeated here.
func (m *model) logFooter() string {
	// During a bulk run the panel says the same thing, repository by repository
	if m.runPanelOn() {
		return ""
	}
	for i := len(m.logLines) - 1; i >= 0; i-- {
		l := m.logLines[i]
		if !l.shown() {
			continue
		}
		line := l.msg
		if l.scope != "" {
			line = l.scope + " · " + l.msg
		}
		return dimStyle.Render(fit(" "+m.spin.View()+" "+line, m.width))
	}
	return ""
}

// rowActivity is the last lines about this repository, for its details
func (m *model) rowActivity(r *row, nb int) []string {
	var lines []string
	for i := len(m.logLines) - 1; i >= 0 && len(lines) < nb; i-- {
		l := m.logLines[i]
		if l.scope != r.name || !l.shown() {
			continue
		}
		lines = append([]string{m.logLineView(l, m.detailW)}, lines...)
	}
	return lines
}
