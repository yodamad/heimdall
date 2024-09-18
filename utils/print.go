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
			{gf.Path, gf.CurrentBranch, gf.HasLocalChanges, gf.RemoteChanges},
		})
		t.AppendSeparator()
	}
	t.Render()
}
