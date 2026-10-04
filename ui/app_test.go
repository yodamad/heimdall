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
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
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

	// Folders come before the repositories next to them
	if len(m.rows) != 2 || m.rows[0].name != filepath.Join("group", "behind") || m.rows[1].name != "dirty" {
		t.Fatalf("unexpected repositories : %+v", m.rows)
	}
	behindRow, dirtyRow := m.rows[0], m.rows[1]
	if m.current() != behindRow {
		t.Fatalf("the cursor should be on the first repository : %+v", m.lines)
	}
	press(m, "down")
	view := m.View()
	for _, expected := range []string{"2 repositories", "dirty", " ▾ group  1", " │   behind", "main", "1 changed", "1 file changed.", "Never fetched", "wip.txt", "never fetched"} {
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
	if !behindRow.gf.FetchedAt.IsZero() || behindRow.gf.RemoteChanges != "0" {
		t.Errorf("no fetch expected at startup : %+v", behindRow)
	}
	press(m, "a", "f")
	for _, r := range m.rows {
		if r.gf.FetchedAt.IsZero() || r.gf.RemoteChanges != "1" || len(r.incoming) != 1 || r.busy != "" {
			t.Errorf("%s should be fetched with 1 remote change : %+v", r.name, r)
		}
	}
	view = m.View()
	for _, expected := range []string{"2 repositories", "2 behind", "1 changed, 1 behind", "1 commit behind origin. Press p to pull.", "p pull 1 of 2", "f fetch 2 selected"} {
		if !strings.Contains(view, expected) {
			t.Errorf("view should contain %q :\n%s", expected, view)
		}
	}

	// Bulk pull is confirmed, and skips the repository with local changes
	press(m, "p")
	if m.mode != modeConfirm || !strings.Contains(m.View(), "Pull on 2 repositories") {
		t.Fatalf("pull on several repositories should ask for a confirmation")
	}
	press(m, "y")
	if r := dirtyRow; r.noteOK || !strings.Contains(r.note, "skipped") || r.gf.RemoteChanges != "1" {
		t.Errorf("dirty should not have been pulled : %+v", r)
	}
	if r := behindRow; r.note != "" || r.gf.RemoteChanges != "0" {
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
	if len(dirtyRow.cmds) != 0 {
		t.Errorf("cancelled command should not be run")
	}

	// Command on the current repository only
	press(m, "esc", "u", "G")
	if dirtyRow.selected || behindRow.selected {
		t.Errorf("no repository should be pullable anymore")
	}
	press(m, "!", "l", "s", "enter")
	if cmds := dirtyRow.cmds; len(cmds) != 1 || cmds[0].ExitCode != 0 || !strings.Contains(cmds[0].Output, "wip.txt") {
		t.Errorf("ls should have been run in dirty : %+v", cmds)
	}
	if len(behindRow.cmds) != 0 {
		t.Errorf("ls should not have been run in behind")
	}
	if view = m.View(); !strings.Contains(view, "✓ ls") {
		t.Errorf("view should contain the command result :\n%s", view)
	}

	// The details list the last commits, and the branches when there are several
	if strings.Contains(view, "Branches") || !strings.Contains(view, "Last commits") || !strings.Contains(view, "add one.txt") {
		t.Errorf("view should contain the last commits and no branches :\n%s", view)
	}
	git(t, dirty, "branch", "spike")
	git(t, dirty, "stash", "-u")
	press(m, "r")
	if view = m.View(); !strings.Contains(view, "Branches 2") || !strings.Contains(view, "* main") || !strings.Contains(view, "not on origin") || !strings.Contains(view, "1 stash kept aside.") {
		t.Errorf("view should contain the branches and the stash :\n%s", view)
	}
	press(m, "tab", "enter")
	if dirtyRow.gf.CurrentBranch != "main" {
		t.Errorf("enter on the current branch should do nothing : %+v", dirtyRow)
	}
	press(m, "j")
	if view = m.View(); !strings.Contains(view, "enter switch to spike") {
		t.Errorf("view should tell how to switch to spike :\n%s", view)
	}
	press(m, "enter")
	if r := dirtyRow; r.gf.CurrentBranch != "spike" || !r.branches[0].Current || r.note != "" || m.branchCursor != 0 {
		t.Errorf("dirty should be on spike : %+v", r)
	}
	press(m, "j", "enter", "tab")
	if dirtyRow.gf.CurrentBranch != "main" || m.detailFocus {
		t.Errorf("dirty should be back on main, and the list focused : %+v", dirtyRow)
	}
	git(t, dirty, "stash", "pop")
	press(m, "r")

	// A collapsed folder hides its repositories, actions on it target them all
	press(m, "g", "enter")
	if m.current() != nil || m.currentFolder() != "group" || len(m.lines) != 2 {
		t.Fatalf("group should be collapsed with the cursor on it : %+v", m.lines)
	}
	if view = m.View(); !strings.Contains(view, "1 up to date") || !strings.Contains(view, "on main") {
		t.Errorf("details should list the repositories of group :\n%s", view)
	}
	if view = m.View(); !strings.Contains(view, "▸ group  1") || strings.Contains(view, "   behind ") {
		t.Errorf("view should display group collapsed :\n%s", view)
	}
	press(m, "!", "p", "w", "d", "enter")
	if cmds := behindRow.cmds; len(cmds) != 1 || cmds[0].Cmd != "pwd" || len(dirtyRow.cmds) != 1 {
		t.Errorf("pwd should have been run in the repositories of group only : %+v", cmds)
	}
	press(m, "enter", "down")
	if m.current() != behindRow {
		t.Errorf("group should be expanded again : %+v", m.lines)
	}

	// As in a file navigator, left goes to the folder then collapses it, right does the opposite
	press(m, "h")
	if m.currentFolder() != "group" || len(m.lines) != 3 {
		t.Errorf("the cursor should be on group, still expanded : %+v", m.lines)
	}
	press(m, "h")
	if m.currentFolder() != "group" || len(m.lines) != 2 {
		t.Errorf("group should be collapsed : %+v", m.lines)
	}
	press(m, "l", "l")
	if m.current() != behindRow {
		t.Errorf("group should be expanded, with the cursor on its repository : %+v", m.lines)
	}
	press(m, "h", "h", "z", "z")
	if len(m.lines) != 2 {
		t.Errorf("all folders should be collapsed again : %+v", m.lines)
	}

	// Narrow terminal displays one pane
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	for _, line := range strings.Split(m.View(), "\n") {
		if w := len([]rune(stripAnsi(line))); w > 60 {
			t.Errorf("line is wider than the narrow terminal (%d) : %q", w, line)
		}
	}
}

// tree renders the lines of the list, indented as they are displayed
func tree(m *model) string {
	var lines []string
	for _, line := range m.lines {
		name := line.label
		if line.r != nil {
			name = line.r.base
		}
		lines = append(lines, strings.Repeat("  ", line.depth)+name)
	}
	return strings.Join(lines, "\n")
}

func TestFolderTree(t *testing.T) {
	m := newModel("/work", 5)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	var paths []string
	for _, name := range []string{"zeta", "clients-old/legacy", "clients/globex/site", "clients/tooling", "clients/acme/web", "clients/acme/api", "perso/go/src/tool"} {
		paths = append(paths, filepath.Join("/work", filepath.FromSlash(name)))
	}
	m.Update(discoveredMsg{paths: paths})

	// Folders before repositories, and a folder holding only a folder shares its line
	expected := strings.Join([]string{
		"clients",
		"  acme",
		"    api",
		"    web",
		"  globex",
		"    site",
		"  tooling",
		"clients-old",
		"  legacy",
		filepath.FromSlash("perso/go/src"),
		"  tool",
		"zeta",
	}, "\n")
	if got := tree(m); got != expected {
		t.Fatalf("unexpected tree :\n%s", got)
	}
	if m.current() == nil || m.current().base != "api" {
		t.Fatalf("the cursor should be on the first repository : %+v", m.currentLine())
	}
	if view := m.View(); !strings.Contains(view, " ▾ clients  4") || !strings.Contains(view, " │ ▾ acme  2") || !strings.Contains(view, " │ │   api") {
		t.Errorf("view should display the tree with its guides :\n%s", view)
	}

	// Collapsing everything leaves the cursor on the closest folder still listed
	press(m, "z")
	if got := tree(m); got != "clients\nclients-old\n"+filepath.FromSlash("perso/go/src")+"\nzeta" || m.currentFolder() != "clients" {
		t.Fatalf("all folders should be collapsed with the cursor on clients :\n%s", got)
	}
	if nb := len(m.targets()); nb != 4 {
		t.Errorf("a folder should target the repositories of the folders it holds, got %d", nb)
	}
	press(m, "l", "l")
	if got := tree(m); !strings.HasPrefix(got, "clients\n  acme\n  globex\n  tooling\nclients-old") || m.currentFolder() != filepath.FromSlash("clients/acme") {
		t.Fatalf("clients should be expanded with the cursor on acme, still collapsed :\n%s", got)
	}
	press(m, "h")
	if m.currentFolder() != "clients" {
		t.Errorf("left on a collapsed folder should go to its folder : %+v", m.currentLine())
	}

	// Scrolled, the first line tells the whole path of the folder the next ones are in
	press(m, "z", "z")
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 9})
	press(m, "g", "down", "down", "down", "down", "down")
	if view := m.View(); !strings.Contains(view, " ▾ "+filepath.FromSlash("clients/acme")+"  2") {
		t.Errorf("view should pin the folder of the first lines :\n%s", view)
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
