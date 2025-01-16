package utils

import (
	"fmt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/mitchellh/colorstring"
	"github.com/yodamad/heimdall/build"
	"github.com/yodamad/heimdall/cmd/entity"
	"github.com/yodamad/heimdall/commons"
	"os"
	"strings"
)

func PrintBanner() {
	fmt.Print(colorstring.Color(
		`[light_blue]            _               _       _ _ 
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _` + "`" + ` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|
[default]
`,
	))

	if commons.Verbose {
		fmt.Printf(colorstring.Color("Version [bold][light_gray]%s[reset] (commit %s), built at %s\n"), build.Version, build.Commit, build.Date)
	} else {
		fmt.Printf(colorstring.Color("Version [bold][light_gray]%s[reset]\n"), build.Version)
	}

	fmt.Println("  ")
}

func PrintSeparation() {
	fmt.Println("...")
}

func PrintTable(gitFolders []entity.GitFolder) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path", "Branch", "Local_Changes", "Remote_Changes"})
	for _, gf := range gitFolders {
		t.AppendRows([]table.Row{
			{visualDisplayRepo(gf.Path), gf.CurrentBranch, visualDisplayBool(gf.HasLocalChanges), displayRemoteChanges(gf.RemoteChanges)},
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

func visualDisplayRepo(repo string) string {
	cleanRootDir := strings.TrimPrefix(commons.WorkDir, "./")
	coloredRepo := strings.Replace(repo, cleanRootDir, colorstring.Color("[dark_gray]"+cleanRootDir+"[default]"), -1)
	return coloredRepo
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
		return visualDisplayBool(true) + " (" + changes + ")"
	} else {
		return visualDisplayBool(false)
	}
}
