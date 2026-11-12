package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/utils/tui"
)

func GetRootDir() string {
	rootDir := commons.WorkDir
	interactiveMode := commons.Interactive
	answer := "n"

	for interactiveMode && !strings.EqualFold(answer, "y") {
		answer = tui.AskQuestion(ColorString("🔍 Search in directory "+tui.PathColor+rootDir+"[default] [light_gray][Y/n][default] : "), "Y")
		if strings.EqualFold(answer, "n") {
			if strings.EqualFold(answer, "n") {
				answer = tui.AskQuestion(ColorString("➡️ Directory to search in : "), rootDir)
			}
			rootDir = answer
		} else if answer == "" {
			answer = "y"
		} else if !strings.EqualFold(answer, "y") {
			fmt.Println(ColorString("[yellow]Unknown option value : [light_gray]" + answer))
			// Reset answer
			answer = "n"
		}
	}

	if !interactiveMode {
		Trace(ColorString("🔍 Search in directory "+tui.PathColor+rootDir+"[default]"), false)
	}
	return rootDir
}

func ListGitDirectories(searchDepth int) []entity.GitFolder {

	rootDir := GetRootDir()

	// Initialize the spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = tui.SpinnerStyle

	// Create the model
	m := tui.SpinnerModel{
		Spinner: s,
	}

	if rootDir == commons.DefaultWorkDir {
		m.Text = ColorString("Searching in [bold]default directory[default] : " + tui.PathColor + "'" + rootDir + "'[default]")
	} else {
		m.Text = ColorString("Searching in " + tui.PathColor + "'" + rootDir + "'[default] ...")
	}

	// Start the spinner
	prg := tea.NewProgram(m)

	go func() {
		if _, err := prg.Run(); err != nil {
			err := prg.ReleaseTerminal()
			if err != nil {
				return
			}
		}
	}()

	return checkDirAndSubdirs(rootDir, searchDepth, prg)
}

// How often the walk says where it is : one message per visited entry would flood the program
const scanTickEvery = 200

func checkDirAndSubdirs(rootDir string, searchDepth int, spinner *tea.Program) []entity.GitFolder {
	nbIgnoreSlashes := strings.Count(rootDir, "/")
	if strings.HasPrefix(rootDir, "./") {
		nbIgnoreSlashes--
	}
	gitFoldersFound := make([]entity.GitFolder, 0)

	_, err := isGitDir(rootDir)
	if err != nil {
		// The walk does not know how many repositories there are before it is over, so the count
		// grows instead of being an X of N
		nbVisited := 0
		filepath.WalkDir(strings.TrimPrefix(rootDir, "./"), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// handle possible path err, just in case...
				return err
			}
			// Saying where the walk is on every single entry floods the program and slows it down
			nbVisited++
			if nbVisited%scanTickEvery == 0 {
				spinner.Send(tui.UpdateMessage{Message: fmt.Sprintf("🔍 %d repositories found, scanning %s", len(gitFoldersFound), path)})
			}
			if d.IsDir() && (strings.Count(path, string(os.PathSeparator))-nbIgnoreSlashes) >= searchDepth {
				Trace("Skip "+path, true)
				return fs.SkipDir
			} else if d.IsDir() {
				err = nil
				gitFolder, err := isGitDir(path)
				if err == nil && gitFolder.Path != "" {
					gitFoldersFound = append(gitFoldersFound, gitFolder)
					spinner.Send(tui.UpdateMessage{Message: fmt.Sprintf("🔍 %d repositories found, scanning %s", len(gitFoldersFound), path)})
					spinner.Send(tui.InfoMessage{Message: ColorString("[green]✔[default] " + path)})
					return fs.SkipDir
				}
			}
			// ... process entry
			return nil
		})

		spinner.Send(tui.TheEndMessage())
		spinner.ReleaseTerminal()
	} else {
		gitFoldersFound = append(gitFoldersFound, entity.GitFolder{Path: rootDir})
	}
	return gitFoldersFound
}

func isGitDir(path string) (entity.GitFolder, error) {
	Trace("Inspecting "+path, true)
	_, err := os.Stat(path + "/.git")
	if err == nil {
		Trace("Found a .git folder : "+path, true)
		return entity.GitFolder{Path: path}, nil
	}
	return entity.GitFolder{}, err
}
