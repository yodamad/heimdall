package cmd

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/mitchellh/colorstring"
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
	Use:     "git-info",
	Aliases: []string{"gi"},
	Short:   "List all directories containing a `.git` folder",

	Run: func(cmd *cobra.Command, args []string) {
		utils.PrintBanner()
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		listGitDirs()
	},
}

func listGitDirs() {
	rootDir := commons.RootDir
	interactiveMode := commons.Interactive
	answer := "n"

	for interactiveMode && !strings.EqualFold(answer, "y") {
		answer = commons.AskQuestion(colorstring.Color("🔍 Search in directory [light_blue]" + rootDir + "[default] [light_gray][Y/n][default] : "))
		if strings.EqualFold(answer, "n") {
			if strings.EqualFold(answer, "n") {
				answer = commons.AskQuestion(colorstring.Color("➡️ Directory to search in : "))
			}
			rootDir = answer
		} else if answer == "" {
			answer = "y"
		} else if !strings.EqualFold(answer, "y") {
			fmt.Println(colorstring.Color("[yellow]Unknown option value : [light_gray]" + answer))
			// Reset answer
			answer = "n"
		}
	}

	if rootDir == commons.DEFAULT_FOLDER {
		utils.Trace(colorstring.Color("Searching in [bold]default directory[default] : [light_blue]'"+rootDir+"'[default]"), false)
	} else {
		utils.Trace(colorstring.Color("Searching in [light_blue]'"+rootDir+"'[default] ..."), false)
	}

	maxDepth := commons.MAX_DEPTH
	nbIgnoreSlashes := strings.Count(rootDir, "/")
	nbGitFolders := 0
	nbSkippedFolders := 0

	rootIsGit, err := checkIsGitDir(rootDir)
	if err != nil && rootIsGit {
		utils.Trace("Skip .git folder "+rootDir, false)
		nbSkippedFolders++
	}
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
				err = nil
				foundGit, err := checkIsGitDir(path)
				if err == nil && foundGit {
					nbGitFolders++
					return fs.SkipDir
				} else if err != nil && foundGit {
					utils.Trace(colorstring.Color("⚠️ [light_yellow]Error analyzing [light_blue]"+path+"[light_yellow], skip it..."), false)
					utils.TraceWarn("Skip .git folder " + path + " (root cause : " + err.Error() + ")")
					nbSkippedFolders++
					return fs.SkipDir
				}
			}
			// ... process entry
			return nil
		})
	} else {
		nbGitFolders++
	}

	utils.Trace("---------------", false)
	if nbGitFolders > 0 {
		if nbSkippedFolders > 0 {
			utils.Trace("Found "+strconv.Itoa(nbGitFolders)+" folder(s) (Skip "+strconv.Itoa(nbSkippedFolders)+" folders because of errors, use '-v' to check in details)", false)
		} else {
			utils.Trace("Found "+strconv.Itoa(nbGitFolders)+" folder(s)", false)
		}
		utils.PrintTable(gitFolders)

		if interactiveMode {
			fmt.Println("...")
			menu := utils.NewMenu("Interactive mode options")

			menu.AddItem("Display local changes of a repository", "local")
			menu.AddItem("Display remote changes of a repository", "remote")
			menu.AddItem("Update one or several repositories ([dim]git pull[reset])", "pull")

			choice := menu.Display()

			fmt.Printf("Choice: %s\n", choice)
		}
	} else {
		if nbSkippedFolders > 0 {
			utils.Trace(colorstring.Color("😕 [red]No git folder found[default] (Skip "+strconv.Itoa(nbSkippedFolders)+" folders because of errors, use '-v' to check in details)"), false)
		} else {
			utils.Trace(colorstring.Color("😕 [red]No git folder found"), false)
		}
		utils.Trace(colorstring.Color("🤔 Is [light_blue]"+rootDir+"[default] the correct path ?"), false)
	}
}

func checkIsGitDir(path string) (bool, error) {
	utils.Trace("Inspecting "+path, true)
	_, err := os.Stat(path + "/.git")
	if err == nil {
		utils.Trace("Found a .git folder : "+path, true)

		_, err = checkIfUpToDate(path)

		if err != nil {
			return true, err
		}

		return true, nil
	}
	return false, err
}

func checkIfUpToDate(path string) (git.Status, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		utils.TraceWarn("Cannot check " + path + ". Skip it... (" + err.Error() + ")")
		return nil, err
	} else {
		err := repo.Fetch(&git.FetchOptions{})

		if err != nil && err.Error() != "already up-to-date" {
			utils.TraceWarn("Cannot fetch " + path + ". Skip it... (" + err.Error() + ")")
			return nil, err
		} else {
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
				HasLocalChanges:      !s.IsClean(),
				DetailedLocalChanges: s.String(),
				RemoteChanges:        string(out),
			})

			return s, err
		}
	}
}

func listLocalChanges(s git.Status) {
	for filename, _ := range s {
		fileStatus := s.File(filename)
		fmt.Printf("%s - %s - %s \n", s.IsUntracked(filename), fileStatus.Worktree, fileStatus.Staging)
	}
}
