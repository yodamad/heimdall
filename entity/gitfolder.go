package entity

import (
	"strings"
	"time"
)

type GitFolder struct {
	Path                 string
	CurrentBranch        string
	HasLocalChanges      bool
	DetailedLocalChanges string
	RemoteChanges        string
	ConnectionType       string
	RemoteURL            string
	Ahead                int
	ChangedFiles         []string
	FetchedAt            time.Time
	Err                  string
}

// Branch is a local branch, and where it stands compared to the branch it tracks
type Branch struct {
	Name     string
	Current  bool
	Upstream string
	Gone     bool // the tracked branch is not on the remote anymore
	Ahead    int
	Behind   int
	Age      string // of its last commit
}

// Worktree is a working tree of a repository: where it is, what it holds, and whether
// anything would be lost by removing it. A repository has at least one, its main working
// tree, and git can add others, each on a branch of its own.
type Worktree struct {
	Path     string
	Branch   string // short name, empty when the worktree is detached
	Head     string // short hash of its HEAD
	Main     bool   // the main working tree, the one git cannot remove
	Locked   bool
	Lock     string // why it is locked, when git was told
	Missing  bool   // its folder is gone, only the bookkeeping is left
	Changes  int    // uncommitted changes
	Unmerged int    // commits of its HEAD no other branch nor remote holds
	Age      string // of its last commit, as in "3 days"
}

// Unused tells a worktree nothing would be lost from: git can remove it, nobody locked it,
// and it holds neither uncommitted changes nor commits kept nowhere else. One whose folder
// is gone is unused too : what is left of it is bookkeeping.
func (w Worktree) Unused() bool {
	if w.Main || w.Locked {
		return false
	}
	return w.Missing || (w.Changes == 0 && w.Unmerged == 0)
}

// UnusedWorktrees are the worktrees of a repository which can be removed
func UnusedWorktrees(worktrees []Worktree) []Worktree {
	var unused []Worktree
	for _, w := range worktrees {
		if w.Unused() {
			unused = append(unused, w)
		}
	}
	return unused
}

type CmdInfo struct {
	Cmd      string
	ExitCode int
	Output   string
}

type GitFolderWithCmdInfos struct {
	GitFolder
	Cmds []CmdInfo
}

func HasRemoteChanges(gitFolder GitFolder) bool {
	return len(gitFolder.RemoteChanges) > 0 && strings.TrimSuffix(gitFolder.RemoteChanges, "\n") != "0"
}

func CanPull(gitFolder GitFolder) bool {
	return HasRemoteChanges(gitFolder) && !gitFolder.HasLocalChanges
}
