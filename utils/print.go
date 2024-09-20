package utils

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"heimdall/cmd/entity"
	"os"
	"strings"
)

func PrintTable(gitFolders []entity.GitFolder) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path", "Branch", "Local_Changes", "Remote_Changes"})
	for _, gf := range gitFolders {
		t.AppendRows([]table.Row{
			{gf.Path, gf.CurrentBranch, visualDisplayBool(gf.HasLocalChanges), displayRemoteChanges(gf.RemoteChanges)},
		})
		t.AppendSeparator()
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:  "Branch",
			Align: text.AlignCenter,
		},
		{
			Name:  "Local_Changes",
			Align: text.AlignCenter,
		},
		{
			Name:  "Remote_Changes",
			Align: text.AlignCenter,
		},
	})
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
	changes := strings.TrimSuffix(remoteChanges, "\n")
	if len(changes) > 0 && changes != "0" {
		return visualDisplayBool(true) + "(" + changes + ")"
	} else {
		return visualDisplayBool(false)
	}
}
