package utils

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"heimdall/cmd/entity"
	"os"
)

func PrintTable(gitFolders []entity.GitFolder) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path", "Branch", "IsDirty", "RemoteChanges"})
	for _, gf := range gitFolders {
		t.AppendRows([]table.Row{
			{gf.Path, gf.CurrentBranch, visualDisplayBool(gf.HasLocalChanges), displayRemoteChanges(gf.RemoteChanges)},
		})
		t.AppendSeparator()
	}
	t.Render()
}

func visualDisplayBool(status bool) string {
	if status {
		return "🔴"
	} else {
		return "🟢"
	}
}

func displayRemoteChanges(remoteChanges string) string {
	if len(remoteChanges) > 0 {
		return visualDisplayBool(true) + "(" + remoteChanges + ")"
	} else {
		return visualDisplayBool(false)
	}
}
