package cmd

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v68/github"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var hostname, keepSuffix, isGroupClone bool

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
		utils.UseConfig()
		utils.PrintBanner()
		utils.OverrideLogFile()
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		if keepSuffix && !hostname {
			utils.TraceWarn(utils.ColorString("[bold]keep-suffix[reset][light_yellow] option is ignored because [bold]host[reset][light_yellow] option is not enabled"))
		}
		if !strings.HasSuffix(commons.WorkDir, "/") {
			commons.WorkDir += "/"
		}
		clone(args[0])
	},
}

func init() {
	GitClone.Flags().BoolVarP(&hostname, "include-hostname", "i", false, "Include hostname in path created ?")
	GitClone.Flags().BoolVarP(&keepSuffix, "keep-hostname-suffix", "k", false, "Include hostname suffix (.com, .fr,...) in path created ?")
	GitClone.Flags().BoolVarP(&isGroupClone, "clone-group", "g", false, "Clone all repositories under the given URL")
}

func clone(urlArg string) {
	if isGroupClone {
		parsedUrl, _ := url.Parse(urlArg)
		hostnameOfRepo := parsedUrl.Hostname()
		switch utils.GetPlatformType(hostnameOfRepo) {
		case "gitlab":
			cloneGitlabGroup(urlArg)
		case "github":
			cloneGithubGroup(urlArg)
		default:
			utils.TraceWarn("Platform type not supported yet (only gitlab & github for now)")
		}
	} else {
		cloneRepo(urlArg, 1, 1)
	}
}

func cloneGitlabGroup(groupUrl string) {
	parsedUrl, _ := url.Parse(groupUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	groupPath := parsedUrl.Path

	utils.Trace(utils.ColorString("[light_blue] Listing projects in [yellow]GitLab[light_blue] group [cyan]"+groupPath), false)

	gitlabClient, err := gitlab.NewClient(utils.GetToken(hostnameOfRepo, nil))
	if err != nil {
		utils.TraceWarn("Impossible to log to " + hostnameOfRepo)
	}

	projects, _, err := gitlabClient.Groups.ListGroupProjects(strings.TrimPrefix(groupPath, "/"), &gitlab.ListGroupProjectsOptions{
		ListOptions:      gitlab.ListOptions{},
		Archived:         gitlab.Ptr(false),
		IncludeSubGroups: gitlab.Ptr(true),
	})
	if err != nil {
		utils.TraceWarn(utils.ColorString("Cannot retrieve projects from group : [red]" + err.Error()))
	}

	urls := make([]string, 0, len(projects))
	for _, project := range projects {
		urls = append(urls, project.WebURL)
	}
	cloneGroup(urls)
}

func cloneGithubGroup(orgUrl string) {
	parsedUrl, _ := url.Parse(orgUrl)
	hostnameOfOrg := parsedUrl.Hostname()
	orgPath := parsedUrl.Path

	utils.Trace(utils.ColorString("[light_blue] Listing projects in [yellow]GitHub[light_blue] organization [cyan]"+orgUrl), false)

	token := utils.GetToken(hostnameOfOrg, nil)
	githubClient := github.NewClient(nil).WithAuthToken(token)
	cleanUrl := strings.TrimSuffix(strings.TrimPrefix(orgPath, "/"), "/")
	repos, _, err := githubClient.Repositories.ListByOrg(context.Background(), cleanUrl, &github.RepositoryListByOrgOptions{})
	if err != nil {
		utils.TraceWarn(utils.ColorString("Cannot retrieve projects from group : [red]" + err.Error()))
	}
	urls := make([]string, 0, len(repos))
	for _, project := range repos {
		urls = append(urls, strings.TrimSuffix(project.GetCloneURL(), ".git"))
	}
	cloneGroup(urls)
}

// cloneGroup clones the repositories of a group one after the other, saying how far along it is.
//
// It stays sequential on purpose : asking for a token may prompt, and several clones prompting at
// once would fight over the terminal. What was missing is the count, not the speed.
func cloneGroup(urls []string) {
	utils.Trace(utils.ColorString("[light_blue]📦 "+strconv.Itoa(len(urls))+" repositories to clone"), false)

	failed := 0
	for i, projectUrl := range urls {
		if err := cloneRepo(projectUrl, i+1, len(urls)); err != nil {
			failed++
		}
	}

	done := len(urls) - failed
	summary := "[light_blue]🏁 " + strconv.Itoa(done) + " cloned"
	if failed > 0 {
		summary += "[light_blue], [red]" + strconv.Itoa(failed) + " failed"
	}
	utils.Trace(utils.ColorString(summary), false)
}

// progressPrefix numbers a repository in a group clone. It is empty when there is only one, so
// that cloning a single repository prints exactly what it printed before.
func progressPrefix(index, total int) string {
	if total < 2 {
		return ""
	}
	return "[" + strconv.Itoa(index) + "/" + strconv.Itoa(total) + "] "
}

func cloneRepo(inputUrl string, index, total int) error {
	prefix := progressPrefix(index, total)
	utils.Trace(utils.ColorString("[light_blue]🧬 "+prefix+"Cloning [cyan]"+inputUrl+"..."), false)

	parsedUrl, _ := url.Parse(inputUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	pathToRepo := parsedUrl.Path

	var err error
	if hostname {
		if !keepSuffix {
			re := regexp.MustCompile(`\.[a-zA-Z]+$`)
			hostnameOfRepo = re.ReplaceAllString(hostnameOfRepo, "")
		}
		err = doClone(inputUrl, commons.WorkDir+hostnameOfRepo+pathToRepo)
	} else {
		err = doClone(inputUrl, commons.WorkDir+pathToRepo)
	}
	if err != nil {
		// doClone has already said what went wrong
		return err
	}
	utils.Trace(utils.ColorString("[light_blue]✅ "+prefix+"[cyan]"+inputUrl+"[light_blue] cloned"), false)
	return nil
}

func doClone(inputUrl string, path string) error {
	parsedUrl, _ := url.Parse(inputUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	path = strings.ReplaceAll(path, "//", "/")
	utils.Trace("Create directory "+path, true)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		utils.TraceWarn(utils.ColorString("❌ Cannot create path : [red] " + err.Error()))
		return err
	}

	// The progress of git is a sideband redrawing itself with carriage returns : it is only asked
	// for when the user asked to be told everything, and goes through a writer cutting it in lines
	var progress io.Writer
	if commons.Verbose {
		lines := utils.NewLineWriter(func(line string) { utils.Trace(line, true) })
		defer lines.Close()
		progress = lines
	}

	_, err = git.PlainClone(path, false, &git.CloneOptions{
		Auth:     &http.BasicAuth{Password: utils.GetToken(hostnameOfRepo, nil)},
		URL:      inputUrl + ".git",
		Progress: progress,
	})
	if err != nil {
		utils.TraceWarn(utils.ColorString("❌ Git clone failed: [red] " + err.Error()))
		return err
	}
	return nil
}
