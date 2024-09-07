package cmd

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const DEFAULT_FOLDER = "/Users/admin_local/work"
const MAX_DEPTH = 3

var rootCmd = &cobra.Command{
	Use:   "hugo",
	Short: "Hugo is a very fast static site generator",
	Long: `A Fast and Flexible Static Site Generator built with
                love by spf13 and friends in Go.
                Complete documentation is available at http://hugo.spf13.com`,
	Run: func(cmd *cobra.Command, args []string) {
		println("Echo")
	},
}

var gitInfo = &cobra.Command{
	Use:   "git-info",
	Short: "Hugo is a very fast static site generator",
	Long: `A Fast and Flexible Static Site Generator built with
                love by spf13 and friends in Go.
                Complete documentation is available at http://hugo.spf13.com`,
	Run: func(cmd *cobra.Command, args []string) {
		listGitDirs()
	},
}

func init() {
	rootCmd.AddCommand(gitInfo)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func listGitDirs() {
	maxDepth := 2
	rootDir := DEFAULT_FOLDER
	nbIgnoreSlashes := strings.Count(rootDir, "/")

	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// handle possible path err, just in case...
			return err
		}
		if d.IsDir() && (strings.Count(path, string(os.PathSeparator))-nbIgnoreSlashes) > maxDepth {
			log.WithField("dir", path).Debug("Skip a dir")
			return fs.SkipDir
		} else if d.IsDir() {
			log.WithField("dir", path).Info("Inspecting dir")
			_, err := os.Stat(path + "/.git")
			if err == nil {
				log.WithField("dir", path).Info(" ✅ Found .git")
				return fs.SkipDir
			}
		}
		// ... process entry
		return nil
	})
}
