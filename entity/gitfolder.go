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
