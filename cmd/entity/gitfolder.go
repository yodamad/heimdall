package entity

import "strings"

type GitFolder struct {
	Path                 string
	CurrentBranch        string
	HasLocalChanges      bool
	DetailedLocalChanges string
	RemoteChanges        string
}

func HasRemoteChanges(gitFolder GitFolder) bool {
	return len(gitFolder.RemoteChanges) > 0 && strings.TrimSuffix(gitFolder.RemoteChanges, "\n") != "0"
}

func CanPull(gitFolder GitFolder) bool {
	return HasRemoteChanges(gitFolder) && !gitFolder.HasLocalChanges
}
