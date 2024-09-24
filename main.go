package main

import (
	"fmt"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"heimdall/cmd"
	"heimdall/commons"
	"heimdall/utils"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "heimdall",
	Short: colorstring.Color("[yellow]Heimdall[default] helps you with your git folders"),
	Long: colorstring.Color(`
[yellow]Heimdall[default] is a CLI tool to help you with your git folders.
You can check, update, ... everything easily
          `),
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintBanner()
	},
	Example: colorstring.Color("[light_blue]heimdall -h"),
}

func init() {
	rootCmd.AddCommand(cmd.GitInfo)
	rootCmd.PersistentFlags().StringVarP(&commons.RootDir, "root-dir", "r", commons.DEFAULT_FOLDER, "root directory")
	rootCmd.PersistentFlags().BoolVarP(&commons.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&commons.Interactive, "i", "i", false, "interactive mode")

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	f, _ := os.OpenFile("heimdall.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetOutput(f)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
	rootCmd.SetHelpTemplate(colorstring.Color(`            _               _       _ _ 
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _` + "`" + ` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|

` + fmt.Sprintf(colorstring.Color("Version [bold][light_gray]%s[reset]\n"), utils.Version) + `
[bold]Usage[reset]:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

[bold]	Aliases[reset]:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

[bold]Examples[reset]:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

[bold]Available Commands[reset]:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

[bold]Flags[reset]:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

[bold]Global Flags[reset]:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
	`))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
