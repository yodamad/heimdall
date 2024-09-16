package cmd

import (
	"github.com/go-git/go-git/v5"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"heimdall/cmd/entity"
	"heimdall/commons"
	"heimdall/utils"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var gitFolders = []entity.GitFolder{}

var GitInfo = &cobra.Command{
	Use:   "git-info",
	Short: "List all directories containing a `.git` folder",
	Run: func(cmd *cobra.Command, args []string) {
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		listGitDirs()
	},
}

func listGitDirs() {
	rootDir := commons.RootDir
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

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Path", "Branch", "IsDirty", "RemoteChanges"})
	for _, gf := range gitFolders {
		t.AppendRows([]table.Row{
			{gf.Path, gf.CurrentBranch, gf.HasLocalChanges, gf.RemoteChanges},
		})
		t.AppendSeparator()
	}
	t.Render()
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
	ref, _ := repo.Head()

	utils.Trace("Check "+path, true)
	utils.Trace("Is up-to-date ? "+s.String(), true)
	utils.Trace("Exec : git rev-list --count "+ref.Name().Short()+"..origin/"+ref.Name().Short(), true)

	out, err := exec.Command("git", "-C", path, "rev-list", "--count", ref.Name().Short()+"..origin/"+ref.Name().Short()).Output()
	gitFolders = append(gitFolders, entity.GitFolder{
		Path:                 path,
		CurrentBranch:        ref.Name().Short(),
		HasLocalChanges:      s.IsClean(),
		DetailedLocalChanges: s.String(),
		RemoteChanges:        string(out),
	})

	return s, err
}
