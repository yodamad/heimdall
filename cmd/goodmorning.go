package cmd

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/utils"
	"github.com/yodamad/heimdall/utils/tui"
)

var forceCmd bool
var overrideCmds bool
var newCmds string

// GoodMorningCmd represents the good-morning command

var GoodMorning = &cobra.Command{
	Use:     "good-morning",
	Aliases: []string{"gm"},
	Short:   "Run your morning routine on your git repos",
	Run: func(cmd *cobra.Command, args []string) {
		utils.UseConfig()
		utils.PrintBanner()
		utils.OverrideLogFile()
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		WakeUp()
	},
}

func init() {
	GoodMorning.Flags().IntVarP(&searchDepth, "depth", "d", commons.MaxDepth, "search depth")
	GoodMorning.Flags().BoolVarP(&forceCmd, "force", "f", false, "Don't ask for confirmation before execution commands")
	GoodMorning.Flags().BoolVarP(&overrideCmds, "override-cmds", "o", false, "Override configured commands")
	GoodMorning.Flags().StringVarP(&newCmds, "run-cmds", "r", "", "New commands to execute (comma separated)")
}

func WakeUp() {
	rootDir := utils.GetRootDir()
	gitFoldersFound := utils.ListGitDirectories(searchDepth)
	utils.PrintSeparation()
	if len(gitFoldersFound) > 0 {
		utils.PrintSimpleTable(gitFoldersFound)
		utils.Trace(utils.ColorString("Found [green]"+strconv.Itoa(len(gitFoldersFound))+"[default] folder(s)"), false)
	} else {
		utils.Trace(utils.ColorString("😕 [red]No git folder found"), false)
		utils.Trace(utils.ColorString("🤔 Is "+tui.PathColor+rootDir+"[default] the correct path ?"), false)
	}
	utils.PrintSeparation()

	morningCmds := utils.GetMorningRoutine().Cmds

	if len(morningCmds) == 0 || overrideCmds {
		// Ask for commands
		var answer string
		if newCmds != "" {
			morningCmds = strings.Split(newCmds, ",")
			for i := range morningCmds {
				morningCmds[i] = strings.TrimSpace(morningCmds[i])
			}
			utils.Trace(utils.ColorString("🏗️  Commands to be executed [dark_gray]"+strings.Join(morningCmds, "[default],[dark_gray]")+"[default]"), false)
			answer = "n"
		} else {
			if len(morningCmds) == 0 {
				answer = tui.AskQuestion("⚠️  No morning routine configured, do you want to configure it now ? [Y/n] : ", "Y")
			} else {
				answer = tui.AskQuestion("⚠️  Do you want to override your morning routine ? [Y/n] : ", "Y")
			}

			if answer == "y" || answer == "Y" || answer == "" {
				answer = tui.AskQuestion(utils.ColorString("➡️  Commands to execute [dark_gray](separated by comma)[default]: "), "")
				if answer != "" {
					morningCmds = strings.Split(answer, ",")
					for i := range morningCmds {
						morningCmds[i] = strings.TrimSpace(morningCmds[i])
					}
					utils.Trace(utils.ColorString("✅ Morning routine configured with [dark_gray]"+strings.Join(morningCmds, "[default],[dark_gray]")+"[default]"), false)
				} else {
					utils.Trace(utils.ColorString("❌ [red]Abort, see you tomorrow..."), false)
					return
				}
			} else {
				utils.Trace(utils.ColorString("❌ [red]Abort, see you tomorrow..."), false)
				return
			}
		}
	}

	if !forceCmd {
		answer := tui.AskQuestion(utils.ColorString("☕️  Run your morning routine ([dark_gray]"+strings.Join(morningCmds, "[default],[dark_gray]")+"[default]) on these [green]"+strconv.Itoa(len(gitFoldersFound))+"[default] folders ? [light_gray][Y/n][default] : "), "Y")
		if answer != "y" && answer != "Y" {
			utils.Trace(utils.ColorString("❌ [red]Abort, see you tomorrow..."), false)
			return
		}
		utils.PrintSeparation()
	}

	// Initialize the spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = tui.SpinnerStyle
	m := tui.SpinnerModel{
		Spinner: s,
	}
	m.Text = utils.ColorString("Good Morning!")
	// Start the spinner
	prg := tea.NewProgram(m)

	go func() {
		if _, err := prg.Run(); err != nil {
			err := prg.ReleaseTerminal()
			if err != nil {
				utils.TraceWarn("Failed release terminal " + err.Error())
			}
			utils.Trace("Terminal released", false)
			return
		}
	}()

	updatedFolders := make([]entity.GitFolderWithCmdInfos, 0)
	for _, gf := range gitFoldersFound {

		prg.Send(tui.UpdateMessage{Message: "☀️ Good Morning " + utils.VisualDisplayRepo(gf.Path)})
		// Execute commands
		updatedFolder := entity.GitFolderWithCmdInfos{
			GitFolder: gf,
		}
		for _, cmd := range morningCmds {
			updatedFolder.Cmds = append(updatedFolder.Cmds, utils.ExecCmd(cmd, gf))
		}
		updatedFolders = append(updatedFolders, updatedFolder)
	}
	prg.Send(tui.UpdateMessage{Message: "☕️ All done !"})
	err := prg.ReleaseTerminal()
	if err != nil {
		utils.TraceWarn("Failed release terminal " + err.Error())
	}

	utils.PrintSeparation()
	utils.PrintMorningTable(updatedFolders)
	utils.PrintSeparation()
	utils.Trace("☀️ Have a good day !", false)
	utils.PrintSeparation()
}
