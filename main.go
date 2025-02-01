package main

import (
	"fmt"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/cmd"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
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
	rootCmd.AddCommand(cmd.GitClone)
	rootCmd.AddCommand(cmd.EnvInfo)
	rootCmd.PersistentFlags().BoolVarP(&commons.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&commons.LogDir, "log-dir", "l", commons.DefaultFolder, "log directory")
	rootCmd.PersistentFlags().StringVarP(&commons.WorkDir, "work-dir", "w", commons.DefaultWorkDir, "work directory")
	rootCmd.PersistentFlags().StringVarP(&commons.InputConfigFile, "config-file", "c", commons.InputConfigFile, "config file")

	// Init .heimdall folder
	_, err := os.Stat(commons.DefaultFolder)
	if os.IsNotExist(err) {
		err := os.Mkdir(commons.DefaultFolder, os.ModePerm)
		if err != nil {
			fmt.Errorf("Cannot create dir " + err.Error())
			commons.DefaultFolder = os.TempDir()
		}
	}

	// Create config dir.
	_, err = os.Stat(commons.DefaultConfigFolder)
	if os.IsNotExist(err) {
		err := os.Mkdir(commons.DefaultConfigFolder, os.ModePerm)
		if err != nil {
			fmt.Errorf("Cannot create dir " + err.Error())
		}
	}

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	f, _ := os.OpenFile(commons.DefaultFolder+"heimdall.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	log.SetOutput(f)

	// Only log the warning severity or above.
	log.SetLevel(log.InfoLevel)
	rootCmd.SetHelpTemplate(commons.HelpMessageTemplate)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		utils.OverrideLogFile()
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
