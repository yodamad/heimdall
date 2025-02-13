package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/cmd"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "heimdall",
	Short: utils.ColorString("[yellow]Heimdall[default] helps you with your git folders"),
	Long: utils.ColorString(`
[yellow]Heimdall[default] is a CLI tool to help you with your git folders.
You can check, update, ... everything easily
          `),
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintBanner()
	},
	Example: utils.ColorString("[light_blue]heimdall -h"),
}

func init() {
	rootCmd.AddCommand(cmd.GitInfo)
	rootCmd.AddCommand(cmd.GitClone)
	rootCmd.AddCommand(cmd.EnvInfo)
	rootCmd.PersistentFlags().BoolVarP(&commons.Verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&commons.LogDir, "log-dir", "l", commons.DefaultLogFolder, "log directory")
	rootCmd.PersistentFlags().StringVarP(&commons.WorkDir, "work-dir", "w", commons.DefaultWorkDir, "work directory")
	rootCmd.PersistentFlags().StringVarP(&commons.InputConfigFile, "config-file", "c", commons.InputConfigFile, "config file")
	rootCmd.PersistentFlags().BoolVarP(&commons.NoColor, "no-color", "n", false, "no color output")

	// Create directory for configuration
	_, err := os.Stat(commons.DefaultConfFolder)
	if os.IsNotExist(err) {
		err := os.Mkdir(commons.DefaultConfFolder, os.ModePerm)
		if err != nil {
			fmt.Errorf("Cannot create dir " + err.Error())
			commons.DefaultConfFolder = os.TempDir()
		}
	}

	// Create cache directory for logs
	_, err = os.Stat(commons.DefaultLogFolder)
	if os.IsNotExist(err) {
		err := os.Mkdir(commons.DefaultLogFolder, os.ModePerm)
		if err != nil {
			fmt.Errorf("Cannot create dir " + err.Error())
			commons.DefaultLogFolder = os.TempDir()
		}
	}

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	f, _ := os.OpenFile(commons.DefaultLogFolder+"/heimdall.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
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
