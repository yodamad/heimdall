package cmd

import (
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/build"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	"os"
)

var EnvInfo = &cobra.Command{
	Use:     "env-info",
	Aliases: []string{"env"},
	Short:   "Get information about the environment",
	Run: func(cmd *cobra.Command, args []string) {
		commons.Verbose = true
		utils.PrintBanner()
		printInfo()
	},
}

func printInfo() {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	// Heimdall
	t.AppendRow(table.Row{header("Heimdall"), "", ""})
	t.AppendRow(table.Row{"", "version", colorValue(build.BuildInfos.GitVersion)})
	t.AppendRow(table.Row{"", "commit", colorValue(build.BuildInfos.GitCommit)})
	t.AppendSeparator()

	// OS
	hostInfo, _ := host.Info()
	cpuInfo, _ := cpu.Info()
	t.AppendRow(table.Row{header("OS"), "", ""})
	t.AppendRow(table.Row{"", "type", colorValue(hostInfo.OS)})
	t.AppendRow(table.Row{"", "arch", colorValue(hostInfo.KernelArch)})
	t.AppendRow(table.Row{"", "platform", colorValue(hostInfo.Platform)})
	t.AppendRow(table.Row{"", "family", colorValue(hostInfo.PlatformFamily)})
	t.AppendRow(table.Row{"", "version", colorValue(hostInfo.PlatformVersion)})
	t.AppendSeparator()

	if len(cpuInfo) > 0 {
		t.AppendRow(table.Row{header("CPU"), "", ""})
		t.AppendRow(table.Row{"", "cpu", colorValue(cpuInfo[0].ModelName)})
		t.AppendRow(table.Row{"", "model", colorValue(cpuInfo[0].Model)})
		t.AppendRow(table.Row{"", "family", colorValue(cpuInfo[0].Family)})
	}
	t.Render()
}

func header(value string) string {
	return utils.ColorString("[bold][yellow]" + value + "[default]")
}

func colorValue(value string) string {
	return utils.ColorString("[blue]" + value)
}
