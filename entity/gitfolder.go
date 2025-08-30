package entity

import "strings"

type GitFolder struct {
	Path                 string
	CurrentBranch        string
	HasLocalChanges      bool
	DetailedLocalChanges string
	RemoteChanges        string
	ConnectionType       string
}

type CmdInfo struct {
	Cmd      string
	ExitCode int
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
