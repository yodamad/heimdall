package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
)

func run(t *testing.T, dir string, args ...string) {
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
	run(t, dir, "add", file)
	run(t, dir, "commit", "-m", "add "+file)
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"a/.git", "b/c/.git", "a/nested/.git", "w/x/y/z/.git"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
	}

	found := Discover(root+"/", 3)
	expected := []string{filepath.Join(root, "a"), filepath.Join(root, "b", "c")}
	if len(found) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, found)
	}
	for i := range expected {
		if found[i] != expected[i] {
			t.Errorf("expected %s, got %s", expected[i], found[i])
		}
	}

	if found = Discover(root, 4); len(found) != 3 {
		t.Errorf("expected 3 folders with depth 4, got %v", found)
	}
	if found = Discover(filepath.Join(root, "a"), 3); len(found) != 1 || found[0] != filepath.Join(root, "a") {
		t.Errorf("expected root itself, got %v", found)
	}
}

func TestStatusFetchPull(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	commons.TUIMode = true
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	first := filepath.Join(root, "first")
	second := filepath.Join(root, "second")

	run(t, root, "init", "--bare", remote)
	run(t, root, "clone", remote, first)
	run(t, first, "checkout", "-b", "main")
	commit(t, first, "one.txt")
	run(t, first, "push", "origin", "main")
	run(t, root, "clone", "-b", "main", remote, second)
	commit(t, first, "two.txt")
	run(t, first, "push", "origin", "main")

	gf := LocalStatus(second)
	if gf.Err != "" {
		t.Fatalf("unexpected error : %s", gf.Err)
	}
	if gf.CurrentBranch != "main" || gf.HasLocalChanges || entity.HasRemoteChanges(gf) || gf.RemoteURL != remote {
		t.Errorf("unexpected status before fetch : %+v", gf)
	}

	if err := Fetch(second); err != nil {
		t.Fatalf("fetch failed : %v", err)
	}
	if err := Fetch(second); err != nil {
		t.Fatalf("second fetch failed : %v", err)
	}
	gf = LocalStatus(second)
	if gf.RemoteChanges != "1" || !entity.CanPull(gf) {
		t.Errorf("expected 1 remote change after fetch : %+v", gf)
	}
	if commits := IncomingCommits(second, "main"); len(commits) != 1 {
		t.Errorf("expected 1 incoming commit, got %v", commits)
	}

	if err := os.WriteFile(filepath.Join(second, "local.txt"), []byte("local"), 0644); err != nil {
		t.Fatal(err)
	}
	gf = LocalStatus(second)
	if !gf.HasLocalChanges || len(gf.ChangedFiles) != 1 || entity.CanPull(gf) {
		t.Errorf("expected 1 local change : %+v", gf)
	}
	os.Remove(filepath.Join(second, "local.txt"))

	if err := Pull(second); err != nil {
		t.Fatalf("pull failed : %v", err)
	}
	gf = LocalStatus(second)
	if entity.HasRemoteChanges(gf) || gf.HasLocalChanges {
		t.Errorf("expected clean status after pull : %+v", gf)
	}
	if _, err := os.Stat(filepath.Join(second, "two.txt")); err != nil {
		t.Errorf("two.txt should have been pulled : %v", err)
	}
}
