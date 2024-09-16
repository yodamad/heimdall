package cmd

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"heimdall/utils"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DEFAULT_FOLDER = "/Users/admin_local/work"
const MAX_DEPTH = 3

var RootDir string
var Verbose bool

var rootCmd = &cobra.Command{
	Use:   "heimdall",
	Short: "Heimdall helps you with your git folders",
	Long: `Heimdall is a CLI tool to help you with your git folders.
			You can check, update, ... everything easily
          `,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var gitInfo = &cobra.Command{
	Use:   "git-info",
	Short: "List all directories containing a `.git` folder",
	Run: func(cmd *cobra.Command, args []string) {
		if Verbose {
			log.SetLevel(log.DebugLevel)
		}
		listGitDirs()
	},
}

func init() {
	rootCmd.AddCommand(gitInfo)
	rootCmd.PersistentFlags().StringVarP(&RootDir, "root-dir", "r", DEFAULT_FOLDER, "root directory")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "verbose output")

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

func listGitDirs() {
	rootDir := RootDir
	utils.Trace("Searching in "+rootDir+"...", false)
	maxDepth := 2
	nbIgnoreSlashes := strings.Count(rootDir, "/")
	nbGitFolders := 0

	rootIsGit, _ := checkIsGitDir(rootDir)
	if !rootIsGit {
		filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// handle possible path err, just in case...
				return err
			}
			if d.IsDir() && (strings.Count(path, string(os.PathSeparator))-nbIgnoreSlashes) > maxDepth {
				utils.Trace("Skip "+path, true)
				return fs.SkipDir
			} else if d.IsDir() {
				foundGit, err := checkIsGitDir(path)
				if err == nil && foundGit {
					nbGitFolders++
					return fs.SkipDir
				}
			}
			// ... process entry
			return nil
		})
	}

	utils.Trace("Found "+strconv.Itoa(nbGitFolders)+" folders", false)
}

func checkIsGitDir(path string) (bool, error) {
	utils.Trace("Inspecting "+path, true)
	_, err := os.Stat(path + "/.git")
	if err == nil {
		utils.Trace("Found a .git folder : "+path, true)

		checkIfUpToDate(path)

		if err != nil {
			return false, err
		}

		return true, nil
	}
	return false, err
}

func checkIfUpToDate(path string) (git.Status, error) {
	repo, err := git.PlainOpen(path)
	repo.Fetch(&git.FetchOptions{})
	w, err := repo.Worktree()
	s, err := w.Status()

	utils.Trace(path+" is up-to-date ? "+strconv.FormatBool(s.IsClean()), true)

	return s, err
}
