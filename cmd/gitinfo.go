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

	utils.PrintSeparation()
	if nbGitFolders > 0 {
		if nbSkippedFolders > 0 {
			utils.Trace("Found "+strconv.Itoa(nbGitFolders)+" folder(s) (Skip "+strconv.Itoa(nbSkippedFolders)+" folders because of errors, use '-v' to check in details)", false)
		} else {
			utils.Trace("Found "+strconv.Itoa(nbGitFolders)+" folder(s)", false)
		}
		utils.PrintTable(gitFolders)

		if interactiveMode {
			chooseInteractiveOption()
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

func chooseInteractiveOption() {
	utils.PrintSeparation()
	menu := utils.NewMenu("Interactive mode options")

	if checkIfAtLeastOne(gitFolders, func(folder entity.GitFolder) bool { return folder.HasLocalChanges }) {
		menu.AddItem("📤 Display local changes of a repository", "local")
	}
	if checkIfAtLeastOne(gitFolders, func(folder entity.GitFolder) bool { return entity.HasRemoteChanges(folder) }) {
		menu.AddItem("📥 Display remote commits of a repository", "remote")
	}
	menu.AddItem("🔃 Update one or several repositories ([dim]git pull[reset])", "pull")
	menu.AddItem("✅ I'm done", "end")

	choice := menu.Display()

	switch choice {
	case "local":
		folder := pickSingleItem(gitFolders, func(folder entity.GitFolder) bool { return folder.HasLocalChanges })
		utils.PrintSeparation()
		listLocalChanges(folder)
	case "remote":
		utils.PrintSeparation()
		folder := pickSingleItem(gitFolders, func(folder entity.GitFolder) bool { return entity.HasRemoteChanges(folder) })
		listRemoteChanges(folder)
	case "pull":
		utils.Trace("🚧 Not yet implemented...", false)
		chooseInteractiveOption()
	case "end":
		return
	}
	menu = utils.NewMenu("What to do next:")
	menu.AddItem("🔄 Check another folder", "restart")
	menu.AddItem("✅ I'm done", "end")
	choice = menu.Display()

	switch choice {
	case "restart":
		chooseInteractiveOption()
	case "end":
		return
	}
}

func listLocalChanges(path string) {
	repo, _ := git.PlainOpen(path)
	repo.Fetch(&git.FetchOptions{})
	w, _ := repo.Worktree()
	s, _ := w.Status()

	utils.Trace(colorstring.Color("🚦 [dark_gray]"+strconv.Itoa(len(s))+" files"), false)
	for filename, _ := range s {
		fileStatus := s.File(filename)
		fmt.Printf("%s - %s \n", filename, string(fileStatus.Worktree))
	}
	utils.PrintSeparation()
}

func listRemoteChanges(gitFolder string) {
	repo, _ := git.PlainOpen(gitFolder)
	ref, _ := repo.Head()
	out, _ := exec.Command("git", "-C", gitFolder, "log", "--oneline", ref.Name().Short()+"..origin/"+ref.Name().Short()).Output()

	fullOutput := string(out)

	utils.Trace(colorstring.Color("🚦 [dark_gray]"+strconv.Itoa(strings.Count(fullOutput, "\n"))+" commits"), false)
	utils.Trace(string(out), false)
}

type filterFolder func(folder entity.GitFolder) bool

func checkIfAtLeastOne(items []entity.GitFolder, fn filterFolder) bool {
	for _, item := range items {
		if fn(item) {
			return true
		}
	}
	return false
}

func pickSingleItem(items []entity.GitFolder, fn filterFolder) string {
	menu := utils.NewMenu("Pick one")
	for _, item := range items {
		if fn(item) {
			menu.AddItem(item.Path, item.Path)
		}
	}
	choice := menu.Display()
	return choice
}
