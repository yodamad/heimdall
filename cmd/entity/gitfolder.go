package entity

type GitFolder struct {
	Path                 string
	CurrentBranch        string
	HasLocalChanges      bool
	DetailedLocalChanges string
	RemoteChanges        string
}
