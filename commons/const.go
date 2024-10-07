package commons

import (
	"fmt"
	"github.com/mitchellh/colorstring"
	"os"
)

// ldflags vars
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// DefaultFolder /* Default folder for git-info search */
var DefaultFolder = os.TempDir()

// MAX_DEPTH /* Depth of search */
const MAX_DEPTH = 3

// PUBLICKEY_PATH /* Path for public key for ssh authentication */
var PUBLICKEY_PATH = os.Getenv("HOME") + "/.ssh/id_rsa"

// SSHKEY_PASSWORD /* Default password for ssh authentication key opening */
const SSHKEY_PASSWORD = ""

// REMOTE_NAME /* Name of the remote URL */
const REMOTE_NAME = "origin"

// RootDir /* Which directory to start from */
var RootDir string

// LogDir /* Which directory to log in */
var LogDir string

// Verbose /* Verbose log */
var Verbose bool

// Interactive /* Interactive mode */
var Interactive bool

var HelpMessageTemplate = colorstring.Color(`            _               _       _ _ 
  /\  /\___(_)_ __ ___   __| | __ _| | |
 / /_/ / _ \ | '_ ` + "`" + ` _ \ / _` + "`" + ` |/ _` + "`" + ` | | |
/ __  /  __/ | | | | | | (_| | (_| | | |
\/ /_/ \___|_|_| |_| |_|\__,_|\__,_|_|_|

` + fmt.Sprintf(colorstring.Color("Version [bold][light_gray]%s[reset]\n"), Version) + `
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
	`)
