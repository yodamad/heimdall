package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/commons"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=test", "-c", "user.email=test@test.io", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed : %v\n%s", args, err, out)
	}
}

func commit(t *testing.T, dir string, file string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, file), []byte(file), 0644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", file)
	git(t, dir, "commit", "-m", "add "+file)
}

// drain runs a command and all the ones it triggers, as the bubbletea runtime would do
func drain(m *model, cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		for _, c := range msg {
			drain(m, c)
		}
	case spinner.TickMsg:
	default:
		_, next := m.Update(msg)
		drain(m, next)
	}
}

func press(m *model, keys ...string) {
	for _, key := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		switch key {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case " ":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		}
		_, cmd := m.Update(msg)
		drain(m, cmd)
	}
}

func TestNavigateAndRunActions(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	commons.TUIMode = true
	base := t.TempDir()
	root := filepath.Join(base, "work")
	remote := filepath.Join(base, "remote.git")
	source := filepath.Join(base, "source")
	behind := filepath.Join(root, "group", "behind")
	dirty := filepath.Join(root, "dirty")

	git(t, base, "init", "--bare", remote)
	git(t, base, "clone", remote, source)
	git(t, source, "checkout", "-b", "main")
	commit(t, source, "one.txt")
	git(t, source, "push", "origin", "main")
	os.MkdirAll(filepath.Join(root, "group"), 0755)
	git(t, base, "clone", "-b", "main", remote, behind)
	git(t, base, "clone", "-b", "main", remote, dirty)
	os.WriteFile(filepath.Join(dirty, "wip.txt"), []byte("wip"), 0644)
	commit(t, source, "two.txt")
	git(t, source, "push", "origin", "main")

	m := newModel(root+"/", 3)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	drain(m, m.Init())

	if len(m.rows) != 2 || m.rows[0].name != "dirty" || m.rows[1].name != filepath.Join("group", "behind") {
		t.Fatalf("unexpected repositories : %+v", m.rows)
	}
	view := m.View()
	for _, expected := range []string{"2 repositories", "dirty", "group/behind", "main", "Local changes (1)", "wip.txt"} {
		if !strings.Contains(view, expected) {
			t.Errorf("view should contain %q :\n%s", expected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if w := len([]rune(stripAnsi(line))); w > 120 {
			t.Errorf("line is wider than the terminal (%d) : %q", w, line)
		}
	}
	if nb := strings.Count(view, "\n") + 1; nb != 30 {
		t.Errorf("view should fill the 30 lines of the terminal, got %d", nb)
	}

	// Filter
	press(m, "/", "b", "e", "h", "enter")
	if len(m.visible) != 1 || m.current().name != filepath.Join("group", "behind") {
		t.Fatalf("filter should keep only group/behind : %+v", m.visible)
	}
	press(m, "esc")
	if len(m.visible) != 2 {
		t.Fatalf("esc should clear the filter")
	}

	// Nothing is fetched at startup, bulk fetch needs no confirmation
	if m.rows[1].fetched || m.rows[1].gf.RemoteChanges != "0" {
		t.Errorf("no fetch expected at startup : %+v", m.rows[1])
	}
	press(m, "a", "f")
	for _, r := range m.rows {
		if !r.fetched || r.gf.RemoteChanges != "1" || len(r.incoming) != 1 || r.busy != "" {
			t.Errorf("%s should be fetched with 1 remote change : %+v", r.name, r)
		}
	}

	// Bulk pull is confirmed, and skips the repository with local changes
	press(m, "p")
	if m.mode != modeConfirm || !strings.Contains(m.View(), "Pull on 2 repositories") {
		t.Fatalf("pull on several repositories should ask for a confirmation")
	}
	press(m, "y")
	if r := m.rows[0]; r.noteOK || !strings.Contains(r.note, "skipped") || r.gf.RemoteChanges != "1" {
		t.Errorf("dirty should not have been pulled : %+v", r)
	}
	if r := m.rows[1]; !r.noteOK || r.gf.RemoteChanges != "0" {
		t.Errorf("behind should have been pulled : %+v", r)
	}
	if _, err := os.Stat(filepath.Join(behind, "two.txt")); err != nil {
		t.Errorf("two.txt should have been pulled : %v", err)
	}

	// Cancelled bulk command
	press(m, "!", "l", "s", "enter")
	if m.mode != modeConfirm {
		t.Fatalf("command on several repositories should ask for a confirmation")
	}
	press(m, "n")
	if len(m.rows[0].cmds) != 0 {
		t.Errorf("cancelled command should not be run")
	}

	// Command on the current repository only
	press(m, "esc", "u", "g")
	if m.rows[0].selected || m.rows[1].selected {
		t.Errorf("no repository should be pullable anymore")
	}
	press(m, "!", "l", "s", "enter")
	if cmds := m.rows[0].cmds; len(cmds) != 1 || cmds[0].ExitCode != 0 || !strings.Contains(cmds[0].Output, "wip.txt") {
		t.Errorf("ls should have been run in dirty : %+v", cmds)
	}
	if len(m.rows[1].cmds) != 0 {
		t.Errorf("ls should not have been run in behind")
	}
	if view = m.View(); !strings.Contains(view, "✓ ls") {
		t.Errorf("view should contain the command result :\n%s", view)
	}

	// Narrow terminal displays one pane
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	for _, line := range strings.Split(m.View(), "\n") {
		if w := len([]rune(stripAnsi(line))); w > 60 {
			t.Errorf("line is wider than the narrow terminal (%d) : %q", w, line)
		}
	}
}

func stripAnsi(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape:
			inEscape = r != 'm'
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
