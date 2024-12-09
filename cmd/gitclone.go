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
	"regexp"
	"strings"
)

var hostname, keepSuffix bool
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
		if keepSuffix && !hostname {
			utils.TraceWarn(colorstring.Color("[bold]keep-suffix[reset][light_yellow] option is ignored because [bold]host[reset][light_yellow] option is not enabled"))
		}
		if !strings.HasSuffix(cloneDir, "/") {
			cloneDir += "/"
		}
		cloneRepo(args[0])
	},
}

func init() {
	utils.UseConfig()
	GitClone.Flags().BoolVarP(&hostname, "host", "p", false, "Include hostname prefix in path created ?")
	GitClone.Flags().StringVarP(&cloneDir, "clone-dir", "c", commons.DefaultWorkDir, "Folder in which clone the repo, by default in configured workdir")
	GitClone.Flags().BoolVarP(&keepSuffix, "keep-hostname-suffix", "k", false, "Include hostname suffix (.com, .fr,...) in path created ?")
}

func cloneRepo(inputUrl string) {
	utils.Trace(colorstring.Color("[light_blue] Cloning "+inputUrl+"..."), false)

	parsedUrl, _ := url.Parse(inputUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	pathToRepo := parsedUrl.Path

	if hostname {
		if !keepSuffix {
			re := regexp.MustCompile(`\.[a-zA-Z]+$`)
			hostnameOfRepo = re.ReplaceAllString(hostnameOfRepo, "")
		}
		clone(inputUrl, cloneDir+hostnameOfRepo+pathToRepo)
	} else {
		clone(inputUrl, cloneDir+pathToRepo)
	}
}

func clone(inputUrl string, path string) {
	path = strings.ReplaceAll(path, "//", "/")
	utils.Trace("Create directory "+path, false)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		utils.TraceWarn("Cannot create path : [light_blue] " + err.Error())
	}
	_, err = git.PlainClone(path, false, &git.CloneOptions{
		URL:      inputUrl + ".git",
		Progress: os.Stderr,
	})
	if err != nil {
		utils.TraceWarn("Git clone failed: [light_blue] " + err.Error())
	}
}
