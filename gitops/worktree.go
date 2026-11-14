package gitops

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yodamad/heimdall/entity"
)

// Worktrees lists the working trees of a repository, the main one first as git lists them,
// each with what it holds. The folder given is one of them, and is left out : the list is
// what this repository has besides the folder Heimdall knows it by.
func Worktrees(path string) []entity.Worktree {
	out, err := exec.Command("git", "-C", path, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil
	}

	var all []entity.Worktree
	for _, block := range strings.Split(strings.TrimSpace(string(out)), "\n\n") {
		wt, ok := parseWorktree(block)
		if !ok {
			continue
		}
		// The main working tree is the first one git lists
		wt.Main = len(all) == 0
		all = append(all, wt)
	}

	self := resolved(path)
	var others []entity.Worktree
	for _, wt := range all {
		if resolved(wt.Path) == self {
			continue
		}
		describe(path, &wt)
		others = append(others, wt)
	}
	return others
}

// parseWorktree reads the block of lines git tells about one working tree
func parseWorktree(block string) (entity.Worktree, bool) {
	var wt entity.Worktree
	for _, line := range strings.Split(block, "\n") {
		field, value, _ := strings.Cut(strings.TrimSpace(line), " ")
		switch field {
		case "worktree":
			wt.Path = value
		case "HEAD":
			wt.Head = value
			if len(wt.Head) > 7 {
				wt.Head = wt.Head[:7]
			}
		case "branch":
			wt.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "locked":
			wt.Locked, wt.Lock = true, value
		case "prunable":
			wt.Missing = true
		case "bare":
			// A bare repository has no working tree to look at, but it is still the main one
			wt.Missing = false
		}
	}
	return wt, wt.Path != ""
}

// describe reads what a worktree holds : its uncommitted changes, the commits only it has,
// and how old its last commit is. A worktree whose folder is gone holds nothing anymore.
func describe(repo string, wt *entity.Worktree) {
	if info, err := os.Stat(wt.Path); err != nil || !info.IsDir() {
		wt.Missing = true
	}
	if out, err := exec.Command("git", "-C", repo, "log", "-1", "--format=%ct", wt.Head).Output(); err == nil {
		wt.Age = age(string(out))
	}
	if wt.Missing {
		return
	}
	if out, err := exec.Command("git", "-C", wt.Path, "status", "--porcelain").Output(); err == nil {
		wt.Changes = len(lines(string(out)))
	}
	// Commits reachable from this worktree only : its own branch is left out, so what is
	// counted is what no other branch, no tag and no remote keeps. The refs are listed one
	// kind at a time and not with --all, which would hold HEAD itself and count nothing.
	args := []string{"-C", wt.Path, "rev-list", "--count", "HEAD", "--not"}
	if wt.Branch != "" {
		// Against --branches, the pattern git excludes is relative to refs/heads
		args = append(args, "--exclude="+wt.Branch)
	}
	args = append(args, "--branches", "--tags", "--remotes")
	if out, err := exec.Command("git", args...).Output(); err == nil {
		wt.Unmerged, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
}

// WorktreeCleanup says what removing the unused worktrees of a repository did
type WorktreeCleanup struct {
	Removed []string // paths git removed, or forgot when their folder was already gone
	Kept    int      // worktrees left alone : locked, or holding work
	Err     error    // what git said about the first worktree it refused to remove
}

// CleanWorktrees removes the worktrees of a repository nothing would be lost from, and
// forgets the ones whose folder is gone. It never touches the main working tree, a locked
// one, or one holding uncommitted changes or commits kept nowhere else. Should git refuse
// one all the same, the others are still removed and what it said is reported.
func CleanWorktrees(path string) WorktreeCleanup {
	var cleanup WorktreeCleanup
	pruned := false
	for _, wt := range Worktrees(path) {
		if !wt.Unused() {
			cleanup.Kept++
			continue
		}
		if wt.Missing {
			// One prune forgets every worktree whose folder is gone
			if !pruned {
				if err := runGit(path, "worktree", "prune"); err != nil && cleanup.Err == nil {
					cleanup.Err = err
				}
				pruned = true
			}
			cleanup.Removed = append(cleanup.Removed, wt.Path)
			continue
		}
		if err := runGit(path, "worktree", "remove", wt.Path); err != nil {
			cleanup.Kept++
			if cleanup.Err == nil {
				cleanup.Err = err
			}
			continue
		}
		cleanup.Removed = append(cleanup.Removed, wt.Path)
	}
	return cleanup
}

// git runs a command in a repository, and when it fails the error is what git said about it
func runGit(path string, args ...string) error {
	out, err := exec.Command("git", append([]string{"-C", path}, args...)...).CombinedOutput()
	if err == nil {
		return nil
	}
	return said(string(out), err)
}

// resolved is the real path of a folder, so that two names of the same one match
func resolved(path string) string {
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(real)
	}
	return filepath.Clean(path)
}
