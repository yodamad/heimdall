package cmd

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v68/github"
	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"net/url"
	"os"
	"regexp"
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
		utils.OverrideLogFile()
		utils.PrintBanner()
		if commons.Verbose {
			log.SetLevel(log.DebugLevel)
		}
		if keepSuffix && !hostname {
			utils.TraceWarn(colorstring.Color("[bold]keep-suffix[reset][light_yellow] option is ignored because [bold]host[reset][light_yellow] option is not enabled"))
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
		if strings.Contains(hostnameOfRepo, "gitlab") {
			cloneGitlabGroup(urlArg)
		} else if strings.Contains(hostnameOfRepo, "github") {
			cloneGithubGroup(urlArg)
		} else {
			utils.TraceWarn("Platform not supported yet (only gitlab.com & github.com)")
		}
	} else {
		cloneRepo(urlArg)
	}
}

func cloneGitlabGroup(groupUrl string) {
	parsedUrl, _ := url.Parse(groupUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	groupPath := parsedUrl.Path

	utils.Trace(colorstring.Color("[light_blue] Listing projects in [yellow]GitLab[light_blue] group [cyan]"+groupPath), false)

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
		utils.TraceWarn("Cannot retrieve projects from group : " + err.Error())
	}

	for _, project := range projects {
		projectUrl := project.WebURL
		cloneRepo(projectUrl)
	}
}

func cloneGithubGroup(orgUrl string) {
	parsedUrl, _ := url.Parse(orgUrl)
	hostnameOfOrg := parsedUrl.Hostname()
	orgPath := parsedUrl.Path

	utils.Trace(colorstring.Color("[light_blue] Listing projects in [yellow]GitHub[light_blue] organization [cyan]"+orgUrl), false)

	token := utils.GetToken(hostnameOfOrg, nil)
	githubClient := github.NewClient(nil).WithAuthToken(token)
	cleanUrl := strings.TrimSuffix(strings.TrimPrefix(orgPath, "/"), "/")
	repos, _, err := githubClient.Repositories.ListByOrg(context.Background(), cleanUrl, &github.RepositoryListByOrgOptions{})
	if err != nil {
		utils.TraceWarn("Cannot retrieve projects from group : " + err.Error())
	}
	for _, project := range repos {
		projectUrl := project.GetCloneURL()
		cloneRepo(strings.TrimSuffix(projectUrl, ".git"))
	}
}

func cloneRepo(inputUrl string) {
	utils.Trace(colorstring.Color("[light_blue] Cloning [cyan]"+inputUrl+"..."), false)

	parsedUrl, _ := url.Parse(inputUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	pathToRepo := parsedUrl.Path

	if hostname {
		if !keepSuffix {
			re := regexp.MustCompile(`\.[a-zA-Z]+$`)
			hostnameOfRepo = re.ReplaceAllString(hostnameOfRepo, "")
		}
		doClone(inputUrl, commons.WorkDir+hostnameOfRepo+pathToRepo)
	} else {
		doClone(inputUrl, commons.WorkDir+pathToRepo)
	}
}

func doClone(inputUrl string, path string) {
	parsedUrl, _ := url.Parse(inputUrl)
	hostnameOfRepo := parsedUrl.Hostname()
	path = strings.ReplaceAll(path, "//", "/")
	utils.Trace("Create directory "+path, true)
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		utils.TraceWarn("Cannot create path : [light_blue] " + err.Error())
	}
	_, err = git.PlainClone(path, false, &git.CloneOptions{
		Auth:     &http.BasicAuth{Password: utils.GetToken(hostnameOfRepo, nil)},
		URL:      inputUrl + ".git",
		Progress: nil,
	})
	if err != nil {
		utils.TraceWarn("Git clone failed: [light_blue] " + err.Error())
	}
}
