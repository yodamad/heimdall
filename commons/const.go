package commons

import (
	"fmt"
	"os"
	"runtime"

	"github.com/mitchellh/colorstring"
	"github.com/yodamad/heimdall/build"
)

var DefaultConfFolderFunc = func() string {
	home, _ := os.UserConfigDir()
	if runtime.GOOS == "darwin" {
		home = os.Getenv("HOME")
	}
	return home + "/.heimdall/"
}

// DefaultFolder /* Default folder for git-info search */
var DefaultConfFolder = DefaultConfFolderFunc()

// DefaultConfigFile /* Default config file name */
var DefaultConfigFile = "heimdall.yml"

var DefaultWorkDirFunc = func() string { home, _ := os.UserHomeDir(); return home }

var DefaultLogFolder = DefaultLogDirFunc()
var DefaultLogDirFunc = func() string {
	home, _ := os.UserCacheDir()
	if runtime.GOOS == "darwin" {
		home = DefaultConfFolder
	}
	return home
}

// DefaultWorkDir /* Default work folder */
var DefaultWorkDir = DefaultWorkDirFunc()

// MAX_DEPTH /* Depth of search */
const MAX_DEPTH = 3

// PUBLICKEY_PATH /* Path for public key for ssh authentication */
var PUBLICKEY_PATH = os.Getenv("HOME") + "/.ssh/id_rsa"

// SSHKEY_PASSWORD /* Default password for ssh authentication key opening */
const SSHKEY_PASSWORD = ""

// REMOTE_NAME /* Name of the remote URL */
const REMOTE_NAME = "origin"

// WorkDir /* Which directory to start from */
var WorkDir string

// LogDir /* Which directory to log in */
var LogDir string

// Verbose /* Verbose log */
var Verbose bool

// Interactive /* Interactive mode */
var Interactive bool

// InputConfigFile /* The config file to use */
var InputConfigFile = DefaultConfFolder + DefaultConfigFile

// ENV_VARIABLE /* Prefix for env. variable in config file */
const ENV_VARIABLE = "env."

var HelpMessageTemplate = colorstring.Color(`[light_blue]            _               _       _ _
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ `+"`"+` _ \ / _`+"`"+` |/ _`+"`"+` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|

[default]`) + fmt.Sprintf(colorstring.Color("Version [bold][light_gray]%s[reset]\n"), build.BuildInfos.GitVersion) + colorstring.Color(`
[bold]Usage[reset]:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

[bold]Aliases[reset]:
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

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}`) + fmt.Sprintf("\n")
