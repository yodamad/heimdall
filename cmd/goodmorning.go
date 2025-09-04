package cmd

import (
	"strconv"

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
	GoodMorning.Flags().BoolVarP(&forceCmd, "force", "f", false, "Don\\'t ask for confirmation before executino commands")
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

	if !forceCmd {
		utils.PrintSeparation()
		answer := tui.AskQuestion(utils.ColorString("☕️ Run your morning routine on these [green]"+strconv.Itoa(len(gitFoldersFound))+"[default] folders ? [light_gray][Y/n][default] : "), "Y")
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
			prg.ReleaseTerminal()
		}
	}()

	updatedFolders := make([]entity.GitFolderWithCmdInfos, 0)
	for _, gf := range gitFoldersFound {

		updatedFolder := entity.GitFolderWithCmdInfos{
			GitFolder: gf,
		}
		for _, cmd := range utils.GetMorningRoutine().Cmds {
			updatedFolder.Cmds = append(updatedFolder.Cmds, utils.ExecCmd(cmd, gf))
		}
		updatedFolders = append(updatedFolders, updatedFolder)
	}
	prg.Quit()
	utils.PrintMorningTable(updatedFolders)
}
