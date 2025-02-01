package cmd

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/cmd/entity"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	"github.com/yodamad/heimdall/utils/tui"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var searchDepth = commons.MAX_DEPTH
var gitFolders []entity.GitFolder

var GitInfo = &cobra.Command{
	Use:     "git-info",
	Aliases: []string{"gi"},
	Short:   "List all directories containing a `.git` folder",
	Run: func(cmd *cobra.Command, args []string) {
		utils.UseConfig()
		utils.PrintBanner()
		utils.OverrideLogFile()
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		listGitDirs()
	},
}

func init() {
	GitInfo.Flags().IntVarP(&searchDepth, "depth", "d", commons.MAX_DEPTH, "search depth")
	GitInfo.Flags().BoolVarP(&commons.Interactive, "interactive-mode", "i", false, "interactive mode")
}

func listGitDirs() {
	rootDir := commons.WorkDir
	interactiveMode := commons.Interactive
	answer := "n"

	for interactiveMode && !strings.EqualFold(answer, "y") {
		answer = commons.AskQuestion(colorstring.Color("🔍 Search in directory "+tui.PathColor+rootDir+"[default] [light_gray][Y/n][default] : "), "Y")
		if strings.EqualFold(answer, "n") {
			if strings.EqualFold(answer, "n") {
				answer = commons.AskQuestion(colorstring.Color("➡️ Directory to search in : "), rootDir)
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

	if !interactiveMode {
		utils.Trace(colorstring.Color("🔍 Search in directory "+tui.PathColor+rootDir+"[default]"), false)
	}

	// Initialize the spinner
	s := spinner.New()
	s.Spinner = spinner.Points
	s.Style = tui.SpinnerStyle

	// Create the model
	m := tui.SpinnerModel{
		Spinner: s,
	}

	if rootDir == commons.DefaultWorkDir {
		m.Text = colorstring.Color("Searching in [bold]default directory[default] : " + tui.PathColor + "'" + rootDir + "'[default]")
	} else {
		m.Text = colorstring.Color("Searching in " + tui.PathColor + "'" + rootDir + "'[default] ...")
	}

	// Start the spinner
	prg := tea.NewProgram(m)

	go func() {
		if _, err := prg.Run(); err != nil {
			prg.ReleaseTerminal()
		}
	}()

	checkDir(rootDir, prg)
}

func checkDir(rootDir string, spinner *tea.Program) tea.Cmd {
	nbIgnoreSlashes := strings.Count(rootDir, "/")
	if strings.HasPrefix(rootDir, "./") {
		nbIgnoreSlashes--
	}
	nbGitFolders := 0
	nbSkippedFolders := 0

	rootIsGit, err := checkIsGitDir(rootDir, spinner)
	if err != nil && rootIsGit {
		utils.Trace("Skip .git folder "+rootDir, false)
		nbSkippedFolders++
	}
	if !rootIsGit {
		filepath.WalkDir(strings.TrimPrefix(rootDir, "./"), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// handle possible path err, just in case...
				return err
			}
			if d.IsDir() && (strings.Count(path, string(os.PathSeparator))-nbIgnoreSlashes) >= searchDepth {
				utils.Trace("Skip "+path, true)
				return fs.SkipDir
			} else if d.IsDir() {
				err = nil
				foundGit, err := checkIsGitDir(path, spinner)
				if err == nil && foundGit {
					nbGitFolders++
					return fs.SkipDir
				} else if err != nil && foundGit {
					spinner.Send(tui.PrintMessage{Path: path})
					if commons.Verbose {
						spinner.Send(tui.ErrorMessage{Error: err.Error()})
					}
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
	spinner.Send(tui.TheEndMessage())
	spinner.ReleaseTerminal()

	utils.PrintSeparation()
	if nbGitFolders > 0 {
		utils.PrintTable(gitFolders)
		if nbSkippedFolders > 0 {
			utils.Trace(colorstring.Color("Found [green]"+strconv.Itoa(nbGitFolders)+"[default] folder(s) (Skip [yellow]"+strconv.Itoa(nbSkippedFolders)+"[default] folders because of errors, use '-v' to check in details)"), false)
		} else {
			utils.Trace(colorstring.Color("Found [green]"+strconv.Itoa(nbGitFolders)+"[default] folder(s)"), false)
		}
		if commons.Interactive {
			chooseInteractiveOption(spinner)
		}
	} else {
		if nbSkippedFolders > 0 {
			utils.Trace(colorstring.Color("😕 [red]No git folder found[default] (Skip [yellow]"+strconv.Itoa(nbSkippedFolders)+"[default] folders because of errors, use '-v' to check in details)"), false)
		} else {
			utils.Trace(colorstring.Color("😕 [red]No git folder found"), false)
		}
		utils.Trace(colorstring.Color("🤔 Is "+tui.PathColor+rootDir+"[default] the correct path ?"), false)
	}

	return tea.Quit
}

func checkIsGitDir(path string, spinner *tea.Program) (bool, error) {
	utils.Trace("Inspecting "+path, true)
	_, err := os.Stat(path + "/.git")
	if err == nil {
		utils.Trace("Found a .git folder : "+path, true)
		_, err = checkIfUpToDate(path, spinner)

		if err != nil {
			return true, err
		}

		return true, nil
	}
	return false, err
}

func checkIfUpToDate(path string, spinner *tea.Program) (git.Status, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	} else {
		connectionType, err := gitFetch(repo, spinner)
		if err != nil && err.Error() != "already up-to-date" {
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
				ConnectionType:       connectionType,
			})

			return s, err
		}
	}
}

func chooseInteractiveOption(spinner *tea.Program) {
	utils.PrintSeparation()

	var choices = []string{}
	if checkIfAtLeastOne(gitFolders, func(folder entity.GitFolder) bool { return folder.HasLocalChanges }) {
		choices = append(choices, "📤 Display local changes of a repository")
	}
	if checkIfAtLeastOne(gitFolders, func(folder entity.GitFolder) bool { return entity.HasRemoteChanges(folder) }) {
		choices = append(choices, "📥 Display remote commits of a repository")
	}
	if checkIfAtLeastOne(gitFolders, func(folder entity.GitFolder) bool { return entity.CanPull(folder) }) {
		choices = append(choices, colorstring.Color("🔃 Update one or several repositories ([dim]git pull[reset])"))
	}
	choices = append(choices, "✅ I'm done")

	p := tea.NewProgram(tui.InitialChoiceModel("Interactive mode options", choices))
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	choice := m.(tui.ChoiceModel).Picked()
	switch choice {
	case "📤 Display local changes of a repository":
		folder := pickSingleItem(gitFolders, func(folder entity.GitFolder) bool { return folder.HasLocalChanges })
		utils.PrintSeparation()
		listLocalChanges(folder, spinner)
	case "📥 Display remote commits of a repository":
		utils.PrintSeparation()
		folder := pickSingleItem(gitFolders, func(folder entity.GitFolder) bool { return entity.HasRemoteChanges(folder) })
		listRemoteChanges(folder)
	case colorstring.Color("🔃 Update one or several repositories ([dim]git pull[reset])"):
		toUpdate := selectItems(gitFolders, func(folder entity.GitFolder) bool { return entity.CanPull(folder) })
		utils.PrintSeparation()
		if len(toUpdate) > 0 {
			utils.Trace(colorstring.Color(tui.TitleColor+"Pulling repositories :[default]\n"), false)
		}
		for _, folder := range toUpdate {
			gitPull(folder)
		}
		chooseInteractiveOption(spinner)
	case "✅ I'm done":
		os.Exit(0)
	}

	choices = []string{"🔄 Check another folder", "✅ I'm done"}
	p = tea.NewProgram(tui.InitialChoiceModel("What to do next", choices))
	m, err = p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	choice = m.(tui.ChoiceModel).Picked()

	switch choice {
	case "🔄 Check another folder":
		chooseInteractiveOption(spinner)
	case "✅ I'm done":
		return
	}
}

func listLocalChanges(path string, spinner *tea.Program) {
	repo, _ := git.PlainOpen(path)
	_, err := gitFetch(repo, spinner)
	if err != nil {
		utils.TraceWarn("Impossible to fetch : " + err.Error())
	}
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
	var choices []string
	for _, item := range items {
		if fn(item) {
			choices = append(choices, item.Path)
		}
	}
	p := tea.NewProgram(tui.InitialChoiceModel("Pick one", choices))
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	choice := m.(tui.ChoiceModel).Picked()
	return choice
}

func selectItems(items []entity.GitFolder, fn filterFolder) []entity.GitFolder {
	utils.PrintSeparation()
	var q = colorstring.Color(tui.TitleColor + "Pick repositories to update:[default]")

	var filteredItems []entity.GitFolder
	for _, item := range items {
		if fn(item) {
			filteredItems = append(filteredItems, item)
		}
	}

	p := tea.NewProgram(tui.InitialMenuModel(q, filteredItems))
	m, err := p.Run()
	if err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	var picked []entity.GitFolder
	for index, _ := range m.(tui.MenuModel).Selected {
		picked = append(picked, m.(tui.MenuModel).Selected[index])
	}
	return picked
}

func gitFetch(repo *git.Repository, spinner *tea.Program) (string, error) {
	connectionType := ""
	fetchOptions := &git.FetchOptions{}
	remote, _ := repo.Remote(commons.REMOTE_NAME)
	origin := remote.Config().URLs[0]
	if strings.Contains(origin, "@") {
		connectionType = "SSH"
		re := regexp.MustCompile(`\w+`)
		user := re.FindStringSubmatch(origin)

		var publicKey *ssh.PublicKeys
		sshKey, err := os.ReadFile(commons.PUBLICKEY_PATH)
		if err != nil {
			if spinner != nil {
				spinner.Send(tui.ErrorMessage{Error: err.Error()})
			} else {
				utils.TraceWarn("Cannot read file : " + err.Error())
			}
			return "", err
		}
		publicKey, _ = ssh.NewPublicKeys(user[0], sshKey, commons.SSHKEY_PASSWORD)
		fetchOptions.Auth = publicKey
	} else if strings.HasPrefix(origin, "http") {
		gitUrl, err := url.Parse(origin)
		if err != nil {
			if commons.Verbose && spinner != nil {
				spinner.Send(tui.ErrorMessage{Error: err.Error()})
			} else {
				utils.TraceWarn("Cannot parse URL : " + err.Error())
			}
			return "", err
		}
		hostname := strings.TrimPrefix(gitUrl.Hostname(), "www.")
		connectionType = strings.ToUpper(gitUrl.Scheme)
		if utils.GetToken(hostname, spinner) != "" {
			fetchOptions.Auth = &http.BasicAuth{Password: utils.GetToken(hostname, spinner)}
		}
	}
	return connectionType, repo.Fetch(fetchOptions)
}

func gitPull(folder entity.GitFolder) {
	repo, _ := git.PlainOpen(folder.Path)
	worktree, _ := repo.Worktree()
	err := worktree.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil {
		utils.TraceWarn(colorstring.Color("Cannot pull : [red]" + err.Error()))
	}
	utils.Trace(colorstring.Color("✅ [bold]"+folder.Path+"[reset] pulled"), false)
}
