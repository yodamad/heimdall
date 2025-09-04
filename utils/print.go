package utils

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/yodamad/heimdall/build"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
)

func PrintBanner() {
	if commons.NoColor {
		PrintBannerWithoutColor()
	} else {
		PrintBannerWithColor()
	}
}

func PrintBannerWithColor() {
	fmt.Print(ColorString(
		`[light_blue]            _               _       _ _
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _` + "`" + ` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|
[default]
`,
	))

	if commons.Verbose {
		fmt.Printf(ColorString("Version [bold][light_gray]%s[reset] (commit %s), built at %s, compiled with %s\n"),
			build.BuildInfos.GitVersion, build.BuildInfos.GitCommit[:7], build.BuildInfos.BuildDate, build.BuildInfos.GoVersion)
	} else {
		fmt.Printf(ColorString("Version [bold][light_gray]%s[reset]\n"), build.BuildInfos.GitVersion)
	}

	fmt.Println("  ")
}

func PrintBannerWithoutColor() {
	fmt.Print(
		`            _               _       _ _
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _` + "`" + ` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|
`,
	)
	fmt.Println()

	if commons.Verbose {
		fmt.Printf("Version %s (commit %s), built at %s, compiled with %s\n",
			build.BuildInfos.GitVersion, build.BuildInfos.GitCommit[:7], build.BuildInfos.BuildDate, build.BuildInfos.GoVersion)
	} else {
		fmt.Printf("Version %s\n", build.BuildInfos.GitVersion)
	}

	fmt.Println("  ")
}

func PrintSeparation() {
	fmt.Println("...")
}

func PrintSimpleTable(gitFolders []entity.GitFolder) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path"})
	for _, gf := range gitFolders {
		t.AppendRows([]table.Row{
			{VisualDisplayRepo(gf.Path)},
		})
		t.AppendSeparator()
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:  "Branch",
			Align: text.AlignCenter,
		},
	})
	t.Render()
}

func PrintMorningTable(gitFolders []entity.GitFolderWithCmdInfos) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	row := table.Row{}
	colsConfig := []table.ColumnConfig{}
	row = append(row, "Path")
	colsConfig = append(colsConfig, table.ColumnConfig{
		Name:  "Path",
		Align: text.AlignLeft,
	})
	for _, cmd := range GetMorningRoutine().Cmds {
		row = append(row, cmd)
		colsConfig = append(colsConfig, table.ColumnConfig{
			Name:     cmd,
			Align:    text.AlignCenter,
			WidthMin: len(cmd),
		})
	}
	t.SetColumnConfigs(colsConfig)

	t.AppendHeader(row)
	for _, gf := range gitFolders {
		var values table.Row
		values = append(values, VisualDisplayRepo(gf.Path))
		for _, cmdInfo := range gf.Cmds {
			values = append(values, codeDisplay(cmdInfo.ExitCode))
		}
		t.AppendRows([]table.Row{values})
		t.AppendSeparator()
	}

	t.Render()
}

func codeDisplay(code int) string {
	if code != 0 {
		return "❌ (" + strconv.Itoa(code) + ")"
	}
	return "✅"
}

func PrintTable(gitFolders []entity.GitFolder) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path", "Branch", "Connection Type", "Local_Changes", "Remote_Changes"})
	for _, gf := range gitFolders {
		t.AppendRows([]table.Row{
			{VisualDisplayRepo(gf.Path), gf.CurrentBranch, gf.ConnectionType, visualDisplayBool(gf.HasLocalChanges), displayRemoteChanges(gf.RemoteChanges)},
		})
		t.AppendSeparator()
	}
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:  "Branch",
			Align: text.AlignCenter,
		},
		{
			Name:  "Connection Type",
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

func VisualDisplayRepo(repo string) string {
	cleanRootDir := strings.TrimPrefix(commons.WorkDir, "./")
	coloredRepo := strings.Replace(repo, cleanRootDir, ColorString("[dark_gray]"+cleanRootDir+"[default]"), -1)
	return coloredRepo
}

func visualDisplayBool(status bool) string {
	if commons.NoColor {
		if status {
			return "KO"
		} else {
			return "OK"
		}
	} else {
		if status {
			return "🔴"
		} else {
			return "🟢"
		}
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
