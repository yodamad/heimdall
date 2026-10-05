package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/ui"
	"github.com/yodamad/heimdall/utils"
)

var Tui = &cobra.Command{
	Use:     "tui",
	Aliases: []string{"ui"},
	Short:   "Navigate through your git repositories and run actions on them",
	Run: func(cmd *cobra.Command, args []string) {
		RunTui()
	},
}

func init() {
	Tui.Flags().IntVarP(&searchDepth, "depth", "d", commons.MaxDepth, "search depth")
}

// IsTerminal checks that heimdall is run interactively, which is required by the TUI
func IsTerminal() bool {
	for _, f := range []*os.File{os.Stdin, os.Stdout} {
		if info, err := f.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
			return false
		}
	}
	return true
}

func RunTui() {
	utils.UseConfig()
	// From now on, nothing is printed on stdout, only in log file
	commons.TUIMode = true
	utils.OverrideLogFile()
	if commons.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	if err := ui.Run(commons.WorkDir, searchDepth); err != nil {
		fmt.Println("Alas, there's been an error: " + err.Error())
		os.Exit(1)
	}
}
