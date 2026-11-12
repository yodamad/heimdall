package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/activity"
	"github.com/yodamad/heimdall/commons"
)

// runningModel is a model of nb repositories, a pull having been launched on all of them and the
// first nbBusy of them not being done yet
func runningModel(t *testing.T, w, h, nb, nbBusy int) *model {
	t.Helper()
	m := newModel("/work", 5)
	m.log = activity.New(100)
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})

	var paths []string
	for i := range nb {
		paths = append(paths, filepath.Join("/work", "group", "repo"+string(rune('a'+i))))
	}
	m.Update(discoveredMsg{paths: paths})
	for _, r := range m.rows {
		r.loaded, r.busy = true, ""
		r.gf.CurrentBranch = "main"
	}

	m.batchKind, m.batchTotal = busyPull, nb
	m.batchStartedAt, m.batchFailed = time.Now().Add(-9*time.Second), 0
	for i := range nbBusy {
		m.rows[i].busy = busyPull
	}
	return m
}

func TestRunPanelShownForBulkRuns(t *testing.T) {
	m := runningModel(t, 110, 24, 8, 5)
	// Two of the five hold a worker, the three others wait for one
	m.log.Info(m.rows[0].name, "Receiving objects:  45% (560/1243)")
	m.log.Info(m.rows[1].name, "Resolving deltas:  82% (410/500)")
	m.pullActivity()

	if !m.runPanelOn() {
		t.Fatal("the panel should be shown while a pull runs on five repositories")
	}
	view := m.View()
	for _, want := range []string{
		"Pulling 3 of 8",           // three of the eight are done
		"2 running",                // the ones holding a worker
		"3 queued",                 // and the ones waiting for one
		"9s",                       // how long it has been going on
		"Receiving objects:  45%",  // what each worker is busy with, attributed
		"Resolving deltas:  82%",   // which one line under the list could never show
		"L for the whole activity", // the way to the rest
	} {
		if !strings.Contains(view, want) {
			t.Errorf("the panel should say %q, view :\n%s", want, view)
		}
	}
	// The header badge says the same thing : it gives way to the panel. The rows of the list keep
	// their own ⟳ pulling…, which is about one repository and not about the run.
	if strings.Contains(view, "⟳ pulling 3 of 8") {
		t.Errorf("the header should not repeat the count of the panel, view :\n%s", view)
	}
	// And so does the single line under the list
	if m.logFooter() != "" {
		t.Errorf("the line under the list should be empty while the panel is shown, got %q", m.logFooter())
	}
}

func TestRunPanelListsFailures(t *testing.T) {
	m := runningModel(t, 110, 24, 8, 5)
	m.batchFailed = 2
	m.log.Warn("group/repog", "pulled failed : authentication failed")
	m.log.Warn("group/repoh", "pulled failed : local changes would be overwritten")
	// Something recorded before the run started is not of this run
	m.logLines = append([]logLine{{at: m.batchStartedAt.Add(-time.Minute), level: activity.Warn, scope: "old", msg: "from a previous run"}}, m.logLines...)
	m.pullActivity()

	view := m.View()
	if !strings.Contains(view, "2 failures") {
		t.Errorf("the panel should count the failures, view :\n%s", view)
	}
	for _, want := range []string{"authentication failed", "local changes would be overwritten"} {
		if !strings.Contains(view, "✖") || !strings.Contains(view, want) {
			t.Errorf("the panel should list the failure %q, view :\n%s", want, view)
		}
	}
	if strings.Contains(view, "from a previous run") {
		t.Errorf("the panel should only list the failures of this run, view :\n%s", view)
	}
}

func TestRunPanelGivesWay(t *testing.T) {
	// A single repository is not a bulk run : the line under the list is enough
	m := runningModel(t, 110, 24, 1, 1)
	if m.runPanelOn() {
		t.Error("a run on one repository should keep to the single line")
	}

	// A short pane keeps the list whole, and the header badge says what runs
	m = runningModel(t, 110, 11, 8, 5)
	if m.runPanelOn() {
		t.Error("a pane of a few lines has no room for the panel")
	}
	if !strings.Contains(m.View(), "⟳ pulling 3 of 8") {
		t.Errorf("the header badge should say what runs when the panel cannot, view :\n%s", m.View())
	}

	// The screens taking the whole pane say it their own way
	for _, mode := range []mode{modeHelp, modeCreds, modeLogs} {
		m = runningModel(t, 110, 24, 8, 5)
		m.mode = mode
		if m.runPanelOn() {
			t.Errorf("the panel should not be shown in mode %d", mode)
		}
	}

	// The panel never takes more than half of the pane
	m = runningModel(t, 110, 18, 20, 20)
	m.batchFailed = 3
	m.log.Warn("group/repoa", "pulled failed : nope")
	m.pullActivity()
	if nb := len(m.runPanel()); nb > m.paneH/2 {
		t.Errorf("the panel takes %d lines of a pane of %d", nb, m.paneH)
	}
	// And it takes its lines from the list, which leaves the screen the height of the terminal
	if nb := len(strings.Split(m.View(), "\n")); nb != m.height {
		t.Errorf("the screen is %d lines high, expected %d", nb, m.height)
	}
}

func TestRunMeter(t *testing.T) {
	m := runningModel(t, 110, 24, 12, 12)

	// The three shares fill the width exactly, whatever the rounding
	for _, done := range []int{0, 1, 5, 11, 12} {
		for _, running := range []int{0, 1, 6} {
			if done+running > 12 {
				continue
			}
			meter := m.runMeter(done, running, 12, 100)
			if w := lipgloss.Width(meter); w != 100 {
				t.Errorf("the meter of %d done and %d running is %d cells wide, expected 100", done, running, w)
			}
		}
	}
	// A width which cannot say anything says nothing
	if meter := m.runMeter(1, 1, 12, 2); meter != "" {
		t.Errorf("a meter of two cells should be empty, got %q", meter)
	}

	// Without colors the shares are told apart by the weight of their glyphs
	commons.NoColor = true
	defer func() { commons.NoColor = false }()
	meter := m.runMeter(6, 3, 12, 100)
	if got := strings.Count(meter, "━"); got != 50 {
		t.Errorf("half of the meter should be done, got %d cells", got)
	}
	if got := strings.Count(meter, "┉"); got != 25 {
		t.Errorf("a quarter of the meter should be running, got %d cells", got)
	}
	if got := strings.Count(meter, "┄"); got != 25 {
		t.Errorf("a quarter of the meter should be queued, got %d cells", got)
	}
	// One repository among many is still worth a cell of its own
	if !strings.Contains(m.runMeter(199, 1, 200, 100), "┉") {
		t.Error("the only repository running should be seen on the meter")
	}
}
