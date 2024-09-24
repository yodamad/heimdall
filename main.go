package main

import (
	"fmt"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"heimdall/cmd"
	"heimdall/commons"
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
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	fmt.Println("            _               _       _ _")
	fmt.Println("  /\\  /\\___(_)_ __ ___   __| | __ _| | |")
	fmt.Println(" / /_/ / _ \\ | '_ ` _ \\ / _` |/ _` | | |")
	fmt.Println("/ __  /  __/ | | | | | | (_| | (_| | | |")
	fmt.Println("\\/ /_/ \\___|_|_| |_| |_|\\__,_|\\__,_|_|_|")
	fmt.Println("                                        ")
	Execute()
}
