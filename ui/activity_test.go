package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/activity"
	"github.com/yodamad/heimdall/commons"
)

// quiet returns a model looking at an empty folder, with a log of its own : the tests must not
// depend on what the rest of Heimdall recorded into the default one
func quiet(t *testing.T, width, height int) *model {
	t.Helper()
	m := newModel(t.TempDir()+"/", 3)
	m.log = activity.New(activity.DefaultCapacity)
	m.discovering = false
	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return m
}

// oneGroup returns a model on a folder holding group/repo, so that the folder keys can be tested
func oneGroup(t *testing.T) *model {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	commons.TUIMode = true

	base := t.TempDir()
	root := filepath.Join(base, "work")
	repo := filepath.Join(root, "group", "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, base, "init", repo)
	commit(t, repo, "one.txt")

	// The workers record into the default log, so that is the one read here, emptied first
	m := newModel(root+"/", 3)
	activity.Default.Reset()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	drain(m, m.Init())
	return m
}

func TestActivityLogRecordsActions(t *testing.T) {
	m := oneGroup(t)
	name := filepath.Join("group", "repo")

	// The discovery line is recorded from Update, the repository ones from the workers
	m.pullActivity()
	if !strings.Contains(lines(m), "1 repository found under ") {
		t.Errorf("the discovery should have been recorded :\n%s", lines(m))
	}

	m.logLines = nil
	press(m, "f")
	m.pullActivity()

	var started, ended bool
	for _, l := range m.logLines {
		if l.scope != name {
			continue
		}
		started = started || strings.Contains(l.msg, "fetching")
		ended = ended || strings.Contains(l.msg, "fetched")
	}
	if !started || !ended {
		t.Errorf("the fetch of %s should have been recorded :\n%s", name, lines(m))
	}
}

func TestActivityFooter(t *testing.T) {
	m := quiet(t, 100, 24)
	m.log.Info("repo", "fetched")
	m.pullActivity()

	if footer := stripAnsi(m.statusLine()); !strings.Contains(footer, "repo · fetched") {
		t.Errorf("the status line should show the last activity, got %q", footer)
	}

	// What Heimdall has to say about the run itself still comes first
	m.status = "boom"
	if footer := stripAnsi(m.statusLine()); !strings.Contains(footer, "boom") {
		t.Errorf("the status should win over the activity, got %q", footer)
	}

	m.status = ""
	m.filter.SetValue("something")
	if footer := stripAnsi(m.statusLine()); !strings.Contains(footer, "Filtered on") {
		t.Errorf("the filter hint should win over the activity, got %q", footer)
	}
}

func TestLogPaneToggle(t *testing.T) {
	m := quiet(t, 100, 24)
	m.log.Info("repo", "pulled 3 commits")
	m.pullActivity()

	press(m, "L")
	if m.mode != modeLogs {
		t.Fatalf("L should open the activity pane, mode is %v", m.mode)
	}
	view := stripAnsi(m.View())
	if !strings.Contains(view, "Activity") || !strings.Contains(view, "pulled 3 commits") {
		t.Errorf("the pane should list the activity :\n%s", view)
	}
	if !strings.Contains(stripAnsi(m.keysLine()), "G follow") {
		t.Errorf("the keys should tell how to follow : %q", stripAnsi(m.keysLine()))
	}

	// Scrolling away stops following the last lines, G follows again
	for i := 0; i < 80; i++ {
		m.log.Info("repo", "line "+strconv.Itoa(i))
	}
	m.pullActivity()
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.logFollow {
		t.Error("scrolling up should stop following the last lines")
	}
	press(m, "G")
	if !m.logFollow {
		t.Error("G should follow the last lines again")
	}

	press(m, "esc")
	if m.mode != modeList {
		t.Errorf("esc should go back to the list, mode is %v", m.mode)
	}
}

// TestKeyLStillExpandsFolder is the regression this feature has to avoid : l was already taken
func TestKeyLStillExpandsFolder(t *testing.T) {
	m := oneGroup(t)

	press(m, "g", "enter")
	if len(m.lines) != 1 || m.currentFolder() != "group" {
		t.Fatalf("group should be collapsed with the cursor on it : %+v", m.lines)
	}

	press(m, "l")
	if m.mode == modeLogs {
		t.Fatal("the lowercase l must not open the activity pane")
	}
	if len(m.lines) != 2 {
		t.Fatalf("l should have expanded group : %+v", m.lines)
	}

	press(m, "L")
	if m.mode != modeLogs {
		t.Fatalf("L should open the activity pane, mode is %v", m.mode)
	}
}

func TestDebugLinesNeedVerbose(t *testing.T) {
	was := commons.Verbose
	t.Cleanup(func() { commons.Verbose = was })

	commons.Verbose = false
	m := quiet(t, 100, 24)
	m.log.Debug("repo", "reading…")
	m.pullActivity()
	if strings.Contains(lines(m), "reading…") {
		t.Errorf("a debug line should be hidden without --verbose :\n%s", lines(m))
	}

	commons.Verbose = true
	if !strings.Contains(lines(m), "reading…") {
		t.Errorf("a debug line should show with --verbose :\n%s", lines(m))
	}
}

func TestLogPaneNarrow(t *testing.T) {
	for _, width := range []int{40, 60} {
		m := quiet(t, width, 12)
		m.log.Warn("a/very/long/repository/name/indeed", "pulled failed : a long explanation of what went wrong")
		m.pullActivity()
		press(m, "L")

		for _, line := range strings.Split(m.View(), "\n") {
			if w := lipgloss.Width(stripAnsi(line)); w > width {
				t.Errorf("at %d columns, a line is %d wide : %q", width, w, stripAnsi(line))
			}
		}
	}
}

func TestDroppedLinesAreSaid(t *testing.T) {
	m := quiet(t, 100, 24)
	m.log = activity.New(4)

	m.log.Info("repo", "read")
	m.pullActivity()
	for i := 0; i < 10; i++ {
		m.log.Info("repo", "line "+strconv.Itoa(i))
	}
	m.pullActivity()

	// The dropped line is a debug one : it only matters to whoever asked to be told everything
	was := commons.Verbose
	t.Cleanup(func() { commons.Verbose = was })
	commons.Verbose = true
	if !strings.Contains(lines(m), "lines dropped") {
		t.Errorf("the overwritten lines should be said :\n%s", lines(m))
	}
}

// lines is what the pane shows, without its styling
func lines(m *model) string {
	var b strings.Builder
	for _, l := range m.logLines {
		if l.shown() {
			b.WriteString(stripAnsi(m.logLineView(l, 120)) + "\n")
		}
	}
	return b.String()
}
