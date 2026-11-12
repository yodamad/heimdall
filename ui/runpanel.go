package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/activity"
	"github.com/yodamad/heimdall/commons"
)

// While an action runs on many repositories, one line cannot say what is going on : it shows the
// last thing recorded, whichever repository it came from, and hides every other. The panel docks
// under the list for the length of the run and answers the three questions a run raises : how far
// along it is, what each worker is busy with right now, and what has already failed.
//
// Its top edge is the progress meter, drawn in the same filled cells as the band under the header :
// the run reads as a second bridge being crossed.

// Repositories in flight listed at once, before the rest are counted on one line
const maxFlightLines = 4

// Failed repositories recalled under them, newest first
const maxFailedLines = 2

// The panel is only worth its lines when several repositories are involved
const minBatchForPanel = 2

// A pane shorter than this keeps the list, and the single line under it
const minPaneForPanel = 8

// Width of the repository column of the panel
const flightScopeW = 22

// runPanelOn says if a bulk run is going on, and has room to be shown
func (m *model) runPanelOn() bool {
	switch m.mode {
	case modeHelp, modeCreds, modeLogs:
		// These take the whole pane, and the activity pane already says all of it
		return false
	}
	return m.batchTotal >= minBatchForPanel && m.running() > 0 && m.paneH >= minPaneForPanel
}

// inFlight are the repositories an action is running or queued on, in the order of the list
func (m *model) inFlight() []*row {
	var rows []*row
	for _, r := range m.rows {
		if r.busy != "" && r.busy != busyStatus {
			rows = append(rows, r)
		}
	}
	return rows
}

// liveLine is the last thing recorded about this repository since the run started, which for a
// fetch or a pull is what git says of its progress. It is empty while the repository waits for a
// worker slot : nothing is recorded until one is held.
func (m *model) liveLine(name string) string {
	for i := len(m.logLines) - 1; i >= 0; i-- {
		l := m.logLines[i]
		if l.at.Before(m.batchStartedAt) {
			return ""
		}
		if l.scope == name {
			return l.msg
		}
	}
	return ""
}

// runFailures are the warnings and errors recorded since the run started, newest first
func (m *model) runFailures(nb int) []logLine {
	var lines []logLine
	for i := len(m.logLines) - 1; i >= 0 && len(lines) < nb; i-- {
		l := m.logLines[i]
		if l.at.Before(m.batchStartedAt) {
			break
		}
		if l.level >= activity.Warn {
			lines = append(lines, l)
		}
	}
	return lines
}

// runMeter draws how far the run is along : what is done, what holds a worker right now, and what
// waits for one. Only maxWorkers repositories are worked on at a time, so the three shares are
// always worth telling apart. The filled cells need a background, so glyphs of their own weight
// take over when colors are off.
func (m *model) runMeter(done, running, total, width int) string {
	if width < 4 || total <= 0 {
		return ""
	}
	// A share which is not empty is worth at least one cell : a single repository being worked on
	// among sixty is the whole point of the line
	cells := func(nb int) int {
		if nb <= 0 {
			return 0
		}
		return max(1, nb*width/total)
	}
	doneW, runW := cells(done), min(cells(running), width)
	if doneW+runW > width {
		doneW = width - runW
	}
	queuedW := width - doneW - runW

	if commons.NoColor {
		return strings.Repeat("━", doneW) + strings.Repeat("┉", runW) + strings.Repeat("┄", queuedW)
	}
	fill := func(w int, color lipgloss.TerminalColor) string {
		if w <= 0 {
			return ""
		}
		return lipgloss.NewStyle().Background(color).Render(strings.Repeat(" ", w))
	}
	return fill(doneW, stateColors[stateClean]) + fill(runW, flightColor) + fill(queuedW, trackColor)
}

// runElapsed is how long the run has been going on, in as few characters as it takes
func runElapsed(since time.Time) string {
	if since.IsZero() {
		return ""
	}
	d := time.Since(since).Round(time.Second)
	if d < time.Minute {
		return strconv.Itoa(int(d.Seconds())) + "s"
	}
	return strconv.Itoa(int(d.Minutes())) + "m " + strconv.Itoa(int(d.Seconds())%60) + "s"
}

// runPanel is the panel shown under the list while a bulk action runs, the meter first
func (m *model) runPanel() []string {
	flight := m.inFlight()
	done := max(0, m.batchTotal-len(flight))

	// What each repository in flight is busy with. A repository waiting for a free worker has
	// nothing recorded about it yet : that is how the queued ones are told from the running ones.
	live := make([]string, len(flight))
	running := 0
	for i, r := range flight {
		if live[i] = m.liveLine(r.name); live[i] != "" {
			running++
		}
	}

	lines := []string{" " + m.runMeter(done, running, m.batchTotal, m.width-2)}

	// What the run is, how far along, and how long it has been going on. The header badge keeps the
	// same count, so this line adds what the badge has no room for.
	verb := busyVerbs[m.batchKind]
	if verb == "" {
		verb = "working"
	}
	sep := dimStyle.Render(" · ")
	head := boldStyle.Render(strings.ToUpper(verb[:1])+verb[1:]+" "+strconv.Itoa(done)+" of "+strconv.Itoa(m.batchTotal)) +
		sep + dimStyle.Render(strconv.Itoa(running)+" running")
	// Why some of them are not moving : only maxWorkers repositories are worked on at a time
	if queued := len(flight) - running; queued > 0 {
		head += sep + dimStyle.Render(strconv.Itoa(queued)+" queued")
	}
	if m.batchFailed > 0 {
		head += sep + koStyle.Render(plural(m.batchFailed, "failure", "failures"))
	}
	if elapsed := runElapsed(m.batchStartedAt); elapsed != "" {
		head += sep + dimStyle.Render(elapsed)
	}
	// The way to the whole history, while the panel keeps to the live part of it
	if hint := "L for the whole activity"; lipgloss.Width(head)+lipgloss.Width(hint)+3 <= m.width {
		head += strings.Repeat(" ", m.width-lipgloss.Width(head)-lipgloss.Width(hint)-2) + dimStyle.Render(hint)
	}
	lines = append(lines, " "+head)

	// How many lines the body can have : the failures are kept room for, they are the ones not to
	// be missed, and the panel never takes more than half of the pane
	failures := m.runFailures(maxFailedLines)
	body := min(maxFlightLines+len(failures), max(1, m.paneH/2-2))
	nbFlight := min(len(flight), max(1, body-len(failures)))
	// One line goes to the count of the ones left out, unless it would hide the last repository
	if len(flight) > nbFlight && nbFlight > 1 {
		nbFlight--
	}

	// The ones holding a worker come first : they are the ones with something to say, and the ones
	// left out of a short panel are then the ones which have not started
	order := make([]int, 0, len(flight))
	for i := range flight {
		if live[i] != "" {
			order = append(order, i)
		}
	}
	for i := range flight {
		if live[i] == "" {
			order = append(order, i)
		}
	}

	scopeW := min(flightScopeW, max(8, m.width/3))
	for _, i := range order[:nbFlight] {
		text, style := live[i], lipgloss.NewStyle()
		if text == "" {
			text, style = "waiting for a free worker", dimStyle
		} else {
			text = m.spin.View() + " " + text
		}
		lines = append(lines, " "+
			lipgloss.NewStyle().Foreground(flightColor).Render(fitLeft(flight[i].name, scopeW)+" ")+
			style.Render(fit(text, m.width-scopeW-3)))
	}
	if rest := len(flight) - nbFlight; rest > 0 {
		lines = append(lines, " "+dimStyle.Render(fit(strconv.Itoa(rest)+" more in flight", m.width-2)))
	}

	// Oldest failure last : the newest one sits closest to the lines which are still moving
	for i := len(failures) - 1; i >= 0; i-- {
		l := failures[i]
		// Everything listed here went wrong, whatever level it was recorded at : one glyph for all
		// of them, so that the eye catches the column and not three different marks
		lines = append(lines, " "+koStyle.Render(fit("✖ "+fitLeft(l.scope, scopeW-2)+" "+l.msg, m.width-2)))
	}
	return lines
}
