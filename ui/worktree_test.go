package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/commons"
)

// TestWorktreesDisplayedAndCleaned checks that the details say what worktrees a repository
// has, and that w removes the ones nothing would be lost from and keeps the others
func TestWorktreesDisplayedAndCleaned(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	commons.TUIMode = true
	base := t.TempDir()
	root := filepath.Join(base, "work")
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	git(t, base, "init", "-b", "main", repo)
	commit(t, repo, "one.txt")

	// Three worktrees next to the repository : one clean, one with a change, one gone
	for _, name := range []string{"spike", "wip", "ghost"} {
		git(t, repo, "worktree", "add", filepath.Join(base, name), "-b", name)
	}
	if err := os.WriteFile(filepath.Join(base, "wip", "wip.txt"), []byte("wip"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(base, "ghost")); err != nil {
		t.Fatal(err)
	}

	m := newModel(root+"/", 3)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	drain(m, m.Init())

	r := m.current()
	if r == nil || len(r.worktrees) != 3 {
		t.Fatalf("the repository should have 3 other worktrees : %+v", r)
	}

	view := m.View()
	for _, expected := range []string{"Worktrees 3", "spike", "nothing kept here", "wip", "1 changed", "ghost", "folder gone",
		"3 other worktrees.", "2 unused,", "w remove 2 unused worktrees",
		// the list has a column of its own counting the worktrees
		"WT"} {
		if !strings.Contains(stripAnsi(view), expected) {
			t.Errorf("view should contain %q :\n%s", expected, view)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if w := len([]rune(stripAnsi(line))); w > 120 {
			t.Errorf("line is wider than the terminal (%d) : %q", w, line)
		}
	}

	// w asks before removing anything, even for a single repository
	press(m, "w")
	if m.mode != modeConfirm || !strings.Contains(m.pending.question, "Remove 2 unused worktrees") {
		t.Fatalf("w should ask first : %v %+v", m.mode, m.pending)
	}
	press(m, "n")
	if m.mode != modeList || len(m.current().worktrees) != 3 {
		t.Fatalf("a refused confirmation should leave the worktrees alone : %+v", m.current())
	}

	press(m, "w", "y")
	if r = m.current(); len(r.worktrees) != 1 || r.worktrees[0].Branch != "wip" {
		t.Fatalf("only the worktree holding a change should be left : %+v", r.worktrees)
	}
	if r.note != "removed 2 worktrees" || !r.noteOK {
		t.Errorf("the repository should say what was removed : %q", r.note)
	}
	if _, err := os.Stat(filepath.Join(base, "spike")); !os.IsNotExist(err) {
		t.Errorf("the clean worktree folder should be gone : %v", err)
	}
	if view = m.View(); !strings.Contains(stripAnsi(view), "Worktrees 1") || strings.Contains(stripAnsi(view), "unused,") {
		t.Errorf("view should list the worktree left and offer nothing to remove :\n%s", view)
	}
	// The column counts the one left, and nothing is marked for removal anymore
	if got := treesCount(r); got != "1" {
		t.Errorf("the column should count the worktree left, got %q", got)
	}

	// Nothing left to remove: w says so instead of asking
	press(m, "w")
	if m.mode != modeList || !strings.Contains(m.status, "No worktree to remove") {
		t.Errorf("w should say there is nothing to remove : %v %q", m.mode, m.status)
	}

	// The activity log tells which worktrees were removed
	m.pullActivity()
	var logged int
	for _, line := range m.logLines {
		if strings.Contains(line.msg, "worktree removed") {
			logged++
		}
	}
	if logged != 2 {
		t.Errorf("the log should name the 2 removed worktrees, got %d :\n%+v", logged, m.logLines)
	}
}
