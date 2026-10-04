package gitops

import (
	"errors"
	"net/url"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
	"github.com/yodamad/heimdall/utils/tui"
)

// ErrNoKey is returned when the ssh key configured for a platform cannot be found
var ErrNoKey = errors.New("configured ssh key does not exist")

var sshRemote = regexp.MustCompile(`(?P<User>[^@]+)@(?P<Host>[^:/]+)`)

func ConnectionType(remoteURL string) string {
	if strings.Contains(remoteURL, "@") {
		return "SSH"
	} else if strings.HasPrefix(remoteURL, "http") {
		if gitUrl, err := url.Parse(remoteURL); err == nil {
			return strings.ToUpper(gitUrl.Scheme)
		}
	}
	return ""
}

func notifyError(spinner *tea.Program, msg string, err error) {
	if commons.Verbose && spinner != nil {
		spinner.Send(tui.ErrorMessage{Error: err.Error()})
	} else {
		utils.TraceWarn(msg + err.Error())
	}
}

// authFor builds the authentication for the origin remote from the configured platforms.
// spinner may be nil.
func authFor(repo *git.Repository, spinner *tea.Program) (string, transport.AuthMethod, error) {
	remote, err := repo.Remote(commons.RemoteName)
	if err != nil {
		notifyError(spinner, "Cannot get remote : ", err)
		return "", nil, err
	}
	origin := remote.Config().URLs[0]
	connectionType := ConnectionType(origin)

	if strings.Contains(origin, "@") {
		user := sshRemote.FindStringSubmatch(strings.TrimPrefix(origin, "ssh://"))
		if user == nil {
			return connectionType, nil, nil
		}
		sshKey := utils.GetPublicKey(user[2], spinner)
		if sshKey == "" {
			return connectionType, nil, ErrNoKey
		}
		fileContent, _ := os.ReadFile(sshKey)
		publicKey, err := ssh.NewPublicKeys(user[1], fileContent, utils.GetPublicKeyPassword(user[2], spinner))
		if err != nil {
			// Let go-git fallback on its default (ssh-agent)
			return connectionType, nil, nil
		}
		return connectionType, publicKey, nil
	} else if strings.HasPrefix(origin, "http") {
		gitUrl, err := url.Parse(origin)
		if err != nil {
			notifyError(spinner, "Cannot parse URL : ", err)
			return "", nil, err
		}
		hostname := strings.TrimPrefix(gitUrl.Hostname(), "www.")
		if token := utils.GetToken(hostname, spinner); token != "" {
			return connectionType, &http.BasicAuth{Password: token}, nil
		}
	}
	return connectionType, nil, nil
}

// FetchRepo fetches origin and returns the connection type used.
// The go-git "already up-to-date" error is returned as is. spinner may be nil.
func FetchRepo(repo *git.Repository, spinner *tea.Program) (string, error) {
	connectionType, auth, err := authFor(repo, spinner)
	if errors.Is(err, ErrNoKey) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return connectionType, repo.Fetch(&git.FetchOptions{Auth: auth})
}

// Fetch fetches origin of the repository located in path
func Fetch(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return err
	}
	_, auth, err := authFor(repo, nil)
	if err != nil {
		return err
	}
	return fetched(path, repo.Fetch(&git.FetchOptions{Auth: auth}))
}

// Pull pulls origin in the repository located in path
func Pull(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return err
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}
	_, auth, err := authFor(repo, nil)
	if err != nil {
		return err
	}
	return fetched(path, worktree.Pull(&git.PullOptions{RemoteName: commons.RemoteName, Auth: auth}))
}

// fetched records a successful fetch, being already up-to-date is one
func fetched(path string, err error) error {
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	markFetched(path)
	return nil
}
