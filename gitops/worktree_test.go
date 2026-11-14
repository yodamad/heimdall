package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// worktreeOf finds a worktree by the base name of its folder
func worktreeOf(t *testing.T, path string, base string) (int, bool) {
	t.Helper()
	all := Worktrees(path)
	for i, wt := range all {
		if filepath.Base(wt.Path) == base {
			return i, true
		}
	}
	return 0, false
}

func TestWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "init", "-b", "main", repo)
	commit(t, repo, "first.txt")

	// A repository alone has nothing besides the folder Heimdall knows it by
	if all := Worktrees(repo); len(all) != 0 {
		t.Fatalf("expected no other worktree, got %v", all)
	}

	// Four worktrees : one clean, one with a change, one with a commit of its own, one locked
	for _, name := range []string{"clean", "dirty", "feat-ahead", "held"} {
		run(t, repo, "worktree", "add", filepath.Join(base, name), "-b", name)
	}
	if err := os.WriteFile(filepath.Join(base, "dirty", "wip.txt"), []byte("wip"), 0644); err != nil {
		t.Fatal(err)
	}
	commit(t, filepath.Join(base, "feat-ahead"), "new.txt")
	run(t, repo, "worktree", "lock", filepath.Join(base, "held"))

	all := Worktrees(repo)
	if len(all) != 4 {
		t.Fatalf("expected 4 worktrees, got %v", all)
	}
	for _, wt := range all {
		if wt.Main {
			t.Errorf("%s should not be the main worktree", wt.Path)
		}
		if wt.Branch != filepath.Base(wt.Path) {
			t.Errorf("expected %s on its own branch, got %q", wt.Path, wt.Branch)
		}
		if wt.Head == "" || wt.Age == "" {
			t.Errorf("expected a head and an age for %s, got %q and %q", wt.Path, wt.Head, wt.Age)
		}
		switch filepath.Base(wt.Path) {
		case "clean":
			if !wt.Unused() {
				t.Errorf("a clean worktree should be unused : %+v", wt)
			}
		case "dirty":
			if wt.Changes != 1 || wt.Unused() {
				t.Errorf("a worktree with a change should be kept : %+v", wt)
			}
		case "feat-ahead":
			if wt.Unmerged != 1 || wt.Unused() {
				t.Errorf("a worktree with a commit of its own should be kept : %+v", wt)
			}
		case "held":
			if !wt.Locked || wt.Unused() {
				t.Errorf("a locked worktree should be kept : %+v", wt)
			}
		}
	}

	// The main worktree is listed, and never unused, when a linked one is the folder looked at
	fromLinked := Worktrees(filepath.Join(base, "clean"))
	if len(fromLinked) != 4 {
		t.Fatalf("expected the main worktree and the 3 others, got %v", fromLinked)
	}
	if !fromLinked[0].Main || fromLinked[0].Unused() {
		t.Errorf("the main worktree should be listed first and kept : %+v", fromLinked[0])
	}
}

func TestCleanWorktrees(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	base := t.TempDir()
	repo := filepath.Join(base, "repo")
	if err := os.Mkdir(repo, 0755); err != nil {
		t.Fatal(err)
	}
	run(t, repo, "init", "-b", "main", repo)
	commit(t, repo, "first.txt")

	for _, name := range []string{"clean", "dirty", "gone"} {
		run(t, repo, "worktree", "add", filepath.Join(base, name), "-b", name)
	}
	if err := os.WriteFile(filepath.Join(base, "dirty", "wip.txt"), []byte("wip"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(base, "gone")); err != nil {
		t.Fatal(err)
	}

	if _, found := worktreeOf(t, repo, "gone"); !found {
		t.Fatal("the worktree whose folder is gone should still be listed")
	}

	cleanup := CleanWorktrees(repo)
	if cleanup.Err != nil {
		t.Fatalf("nothing should have been refused : %v", cleanup.Err)
	}
	if len(cleanup.Removed) != 2 || cleanup.Kept != 1 {
		t.Fatalf("expected the clean and the gone ones removed and the dirty one kept, got %+v", cleanup)
	}
	if _, err := os.Stat(filepath.Join(base, "clean")); !os.IsNotExist(err) {
		t.Errorf("the clean worktree folder should be gone : %v", err)
	}
	if _, found := worktreeOf(t, repo, "gone"); found {
		t.Error("the worktree whose folder was gone should have been forgotten")
	}
	left := Worktrees(repo)
	if len(left) != 1 || filepath.Base(left[0].Path) != "dirty" {
		t.Errorf("only the worktree holding a change should be left, got %v", left)
	}

	// Nothing left to remove, and the worktree holding a change is still there
	if cleanup = CleanWorktrees(repo); len(cleanup.Removed) != 0 || cleanup.Kept != 1 || cleanup.Err != nil {
		t.Errorf("expected nothing more to remove, got %+v", cleanup)
	}
}
