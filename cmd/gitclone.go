package cmd

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	"net/url"
	"os"
)

var hostname bool
var cloneDir string

var GitClone = &cobra.Command{
	Use:     "git-clone",
	Aliases: []string{"gc"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return err
		}
		_, err := url.ParseRequestURI(args[0])
		if err == nil {
			return nil
		}
		return fmt.Errorf("invalid color specified: %s", args[0])
	},
	Short: "Git clone given repository to a folder based on the path of the repo",
	Run: func(cmd *cobra.Command, args []string) {
		utils.OverrideLogFile()
		utils.PrintBanner()
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		cloneRepo(args[0])
	},
}

func init() {
	utils.UseConfig()
	GitClone.Flags().BoolVarP(&hostname, "host", "o", false, "Include hostname in path created ?")
	GitClone.Flags().StringVarP(&cloneDir, "clone-dir", "c", commons.DefaultWorkDir, "Folder in which clone the repo, by default in configured workdir")
}

func cloneRepo(inputUrl string) {
	utils.Trace(colorstring.Color("[light_blue] Cloning "+inputUrl+"..."), false)

	parsedUrl, _ := url.Parse(inputUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	pathToRepo := parsedUrl.Path

	if hostname {
		clone(inputUrl, cloneDir+hostnameOfRepo+pathToRepo)
	} else {
		clone(inputUrl, cloneDir+pathToRepo)
	}
}

func clone(inputUrl string, path string) {
	utils.Trace("Create directory "+path, false)
	err := os.MkdirAll(cloneDir+path, os.ModePerm)
	if err != nil {
		utils.TraceWarn("Cannot create path : [light_blue] " + err.Error())
	}
	_, err = git.PlainClone(cloneDir+path, false, &git.CloneOptions{
		URL:      inputUrl + ".git",
		Progress: os.Stderr,
	})
	if err != nil {
		utils.TraceWarn("Git clone failed: [light_blue] " + err.Error())
		fmt.Println(colorstring.Color("[yellow]Git clone failed: [red] " + err.Error()))
	}
}
