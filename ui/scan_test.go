package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/commons"
)

// fakeRepo makes a folder which Discover takes for a git repository. The status read on it
// fails, which is of no consequence here : these tests are about what is found, not about git.
func fakeRepo(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

// scanModel is a model looking under root, down to depth, with its first scan done
func scanModel(t *testing.T, root string, depth int) *model {
	t.Helper()
	commons.TUIMode = true
	m := newModel(root, depth)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	drain(m, m.Init())
	return m
}

func names(m *model) []string {
	out := make([]string, 0, len(m.rows))
	for _, r := range m.rows {
		out = append(out, r.name)
	}
	return out
}

// contains looks for a text in a rendered view, colors aside
func contains(s, want string) bool {
	return strings.Contains(stripAnsi(s), want)
}

// goToFolder puts the cursor on the line of a folder, and says if it found one
func goToFolder(m *model, folder string) bool {
	for i, line := range m.lines {
		if line.r == nil && line.folder == folder {
			m.cursor = i
			m.moveCursor(0)
			return true
		}
	}
	return false
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	for _, tc := range []struct{ in, want string }{
		{"~", home},
		{"~/work", filepath.Join(home, "work")},
		{"  ~/work  ", filepath.Join(home, "work")},
		{"/tmp/../etc", "/etc"},
		{"/a/b/", "/a/b"},
	} {
		got, err := expandPath(tc.in)
		if err != nil {
			t.Fatalf("expandPath(%q) : %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("expandPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	// A path starting with ~ but not at the home directory is left alone
	if got, _ := expandPath("/x/~foo"); got != "/x/~foo" {
		t.Errorf("~ should only stand for the home directory at the start of a path, got %q", got)
	}
}

func TestDirPromptPrefill(t *testing.T) {
	root := t.TempDir()
	fakeRepo(t, filepath.Join(root, "group", "one"))
	fakeRepo(t, filepath.Join(root, "two"))
	m := scanModel(t, root, 3)

	// On a repository, the prompt starts from the root
	if !goToFolder(m, "group") {
		t.Fatal("no folder line for group")
	}
	press(m, "right") // into the folder, onto the repository it holds
	if m.current() == nil {
		t.Fatalf("the cursor should be on a repository : %+v", m.currentLine())
	}
	press(m, "d")
	if m.mode != modeDir {
		t.Fatalf("d should open the dir prompt, mode is %v", m.mode)
	}
	if got := m.dirInput.Value(); got != shortPath(root) {
		t.Errorf("prompt on a repository should start from the root %q, got %q", shortPath(root), got)
	}
	press(m, "esc")
	if m.mode != modeList || m.dirInput.Value() != "" {
		t.Error("esc should close the prompt and clear it")
	}

	// On a folder, it starts from that folder
	if !goToFolder(m, "group") {
		t.Fatal("no folder line for group")
	}
	press(m, "d")
	if want := shortPath(filepath.Join(root, "group")); m.dirInput.Value() != want {
		t.Errorf("prompt on a folder should start from it, want %q, got %q", want, m.dirInput.Value())
	}
}

func TestChangeDir(t *testing.T) {
	root := t.TempDir()
	fakeRepo(t, filepath.Join(root, "a", "one"))
	fakeRepo(t, filepath.Join(root, "b", "two"))
	m := scanModel(t, root, 3)
	if len(m.rows) != 2 {
		t.Fatalf("two repositories expected under the root, got %v", names(m))
	}

	// A folder is collapsed, to check the state of the old tree does not survive the scan
	if !goToFolder(m, "a") {
		t.Fatal("no folder line for a")
	}
	press(m, "enter")
	if len(m.collapsed) == 0 {
		t.Fatal("enter on a folder should collapse it")
	}

	press(m, "d")
	m.dirInput.SetValue(filepath.Join(root, "a"))
	press(m, "enter")

	if m.mode != modeDir && m.dirNote != "" {
		t.Fatalf("the folder should have been accepted : %q", m.dirNote)
	}
	if m.root != filepath.Join(root, "a") {
		t.Errorf("root should be %q, got %q", filepath.Join(root, "a"), m.root)
	}
	if len(m.rows) != 1 || m.rows[0].name != "one" {
		t.Fatalf("one repository named after the new root expected, got %v", names(m))
	}
	if len(m.collapsed) != 0 {
		t.Errorf("the collapsed folders of the old root should be gone, got %v", m.collapsed)
	}
	if m.discovering {
		t.Error("the scan is over")
	}
	// The header says where it is looking now, and how deep. A long path is cut from the left,
	// so only its end is looked for, and the line still has to hold in the terminal.
	view := m.View()
	if !contains(view, filepath.Join(filepath.Base(root), "a")+" · depth 3") {
		t.Errorf("the header should say the new root and the depth :\n%s", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if w := len([]rune(stripAnsi(line))); w > 120 {
			t.Errorf("line is wider than the terminal (%d) : %q", w, line)
		}
	}
}

func TestChangeDirRefusesWhatCannotBeLookedIn(t *testing.T) {
	root := t.TempDir()
	fakeRepo(t, filepath.Join(root, "one"))
	file := filepath.Join(root, "a-file.txt")
	if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	m := scanModel(t, root, 3)

	for _, tc := range []struct{ typed, note string }{
		{filepath.Join(root, "nope"), "No such folder"},
		{file, "Not a folder"},
	} {
		press(m, "d")
		m.dirInput.SetValue(tc.typed)
		press(m, "enter")

		if m.mode != modeDir {
			t.Fatalf("%q : the prompt should stay up to be fixed, mode is %v", tc.typed, m.mode)
		}
		if !contains(m.dirNote, tc.note) {
			t.Errorf("%q : note should say %q, got %q", tc.typed, tc.note, m.dirNote)
		}
		if m.dirInput.Value() != tc.typed {
			t.Errorf("%q : what was typed should be kept, got %q", tc.typed, m.dirInput.Value())
		}
		if m.root != root || len(m.rows) != 1 {
			t.Errorf("%q : nothing should have been scanned again, root %q rows %v", tc.typed, m.root, names(m))
		}
		// The note is shown, and typing again clears it
		if view := m.View(); !contains(view, tc.note) {
			t.Errorf("%q : the note should be on screen :\n%s", tc.typed, view)
		}
		press(m, "x")
		if m.dirNote != "" {
			t.Errorf("%q : typing again should clear the note", tc.typed)
		}
		press(m, "esc")
	}

	// An empty prompt just closes
	press(m, "d")
	m.dirInput.SetValue("  ")
	press(m, "enter")
	if m.mode != modeList || m.root != root {
		t.Errorf("an empty folder should cancel, mode %v root %q", m.mode, m.root)
	}
}

func TestDepthKeys(t *testing.T) {
	root := t.TempDir()
	fakeRepo(t, filepath.Join(root, "x", "y", "deep"))
	m := scanModel(t, root, 3)
	if len(m.rows) != 1 {
		t.Fatalf("the deep repository should be found at depth 3, got %v", names(m))
	}

	press(m, "-")
	if m.depth != 2 {
		t.Fatalf("- should look one level less deep, depth is %d", m.depth)
	}
	if len(m.rows) != 0 {
		t.Fatalf("nothing should be found at depth 2, got %v", names(m))
	}
	if view := m.View(); !contains(view, "depth 2") || !contains(view, "No git folder") {
		t.Errorf("the screen should say the depth and that nothing was found :\n%s", view)
	}

	press(m, "+")
	if m.depth != 3 || len(m.rows) != 1 {
		t.Fatalf("+ should find it again, depth %d rows %v", m.depth, names(m))
	}
	// = is the unshifted +
	press(m, "=")
	if m.depth != 4 {
		t.Fatalf("= should look deeper too, depth is %d", m.depth)
	}

	// The bounds hold, and nothing is scanned again at them
	m.depth = minDepth
	press(m, "-")
	if m.depth != minDepth || m.status == "" {
		t.Errorf("the depth should not go under %d, depth %d status %q", minDepth, m.depth, m.status)
	}
	m.depth = maxDepth
	press(m, "+")
	if m.depth != maxDepth || m.status == "" {
		t.Errorf("the depth should not go over %d, depth %d status %q", maxDepth, m.depth, m.status)
	}
}

func TestRescanDropsTheAnswerOfAReplacedScan(t *testing.T) {
	root := t.TempDir()
	fakeRepo(t, filepath.Join(root, "a", "one"))
	fakeRepo(t, filepath.Join(root, "b", "two"))
	m := scanModel(t, root, 3)

	// Two scans in a row : the answer of the first one arrives late, and says nothing about
	// where Heimdall is looking now
	stale := m.rescan(filepath.Join(root, "a"), m.depth)
	fresh := m.rescan(filepath.Join(root, "b"), m.depth)
	drain(m, fresh)
	drain(m, stale)

	if m.root != filepath.Join(root, "b") {
		t.Fatalf("root should be the one of the last scan, got %q", m.root)
	}
	if len(m.rows) != 1 || m.rows[0].name != "two" {
		t.Fatalf("only the repositories of the last scan should be listed, got %v", names(m))
	}
}

func TestRescanWaitsForTheRunningActions(t *testing.T) {
	root := t.TempDir()
	fakeRepo(t, filepath.Join(root, "one"))
	m := scanModel(t, root, 3)
	m.rows[0].busy = busyFetch

	press(m, "+")
	if m.depth != 3 || m.status == "" {
		t.Errorf("a scan should be refused while an action runs, depth %d status %q", m.depth, m.status)
	}
	press(m, "d")
	m.dirInput.SetValue(root)
	press(m, "enter")
	if m.discovering {
		t.Error("no scan should have started while an action runs")
	}
}
