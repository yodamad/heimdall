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

// Kinds of credential an AuthError can be about
const (
	KindToken      = "token"
	KindSshKey     = "ssh key"
	KindPassphrase = "ssh key passphrase"
)

// AuthError says that an operation failed because of the credential of a platform, and which one.
// The UI uses it to offer to update that credential instead of only displaying a git error.
type AuthError struct {
	// Host is the platform the credential belongs to, as the configuration keys them
	Host string
	// Kind is which credential is at fault : KindToken, KindSshKey or KindPassphrase
	Kind string
	// Sure is false when the error is merely consistent with a bad credential, which must be
	// reported as a hint and never as a verdict
	Sure bool
	Err  error
}

func (e *AuthError) Error() string {
	if e.Host == "" {
		return e.Err.Error()
	}
	if !e.Sure {
		return e.Err.Error() + " — the " + e.Kind + " for " + e.Host + " may have expired"
	}
	return e.Err.Error() + " — check the " + e.Kind + " for " + e.Host
}

func (e *AuthError) Unwrap() error { return e.Err }

// HostOf returns the host of a remote URL, in the form the configured platforms are keyed by
func HostOf(remoteURL string) string {
	// The scheme first, as in ConnectionType : an https remote can hold a user too
	if !strings.HasPrefix(remoteURL, "http") && strings.Contains(remoteURL, "@") {
		if user := sshRemote.FindStringSubmatch(strings.TrimPrefix(remoteURL, "ssh://")); user != nil {
			return user[2]
		}
		return ""
	}
	if gitUrl, err := url.Parse(remoteURL); err == nil {
		return strings.TrimPrefix(gitUrl.Hostname(), "www.")
	}
	return ""
}

// OriginHost returns the host of the origin remote of a repository
func OriginHost(repo *git.Repository) string {
	remote, err := repo.Remote(commons.RemoteName)
	if err != nil || len(remote.Config().URLs) == 0 {
		return ""
	}
	return HostOf(remote.Config().URLs[0])
}

// asAuthError wraps an error of a fetch or a pull into an AuthError when it is about a credential, so
// that the caller can offer to fix it. Anything else is passed through untouched.
func asAuthError(err error, repo *git.Repository, connectionType string) error {
	if err == nil {
		return nil
	}
	host := OriginHost(repo)
	kind := KindToken
	if connectionType == "SSH" {
		kind = KindSshKey
	}

	switch {
	case errors.Is(err, ErrNoKey):
		return &AuthError{Host: host, Kind: KindSshKey, Sure: true, Err: err}

	case errors.Is(err, transport.ErrAuthenticationRequired), errors.Is(err, transport.ErrAuthorizationFailed):
		return &AuthError{Host: host, Kind: kind, Sure: true, Err: err}

	case errors.Is(err, transport.ErrRepositoryNotFound):
		// A private repository read with a token which is no longer valid answers 404, but so does a
		// repository which was really deleted or renamed. Only a hint, never a verdict : telling the
		// user their token is dead every time a repository disappears would be worse than silence.
		if host != "" && connectionType != "SSH" && utils.IsComplete(host) {
			return &AuthError{Host: host, Kind: KindToken, Sure: false, Err: err}
		}
	}
	return err
}

// ConnectionType tells how a remote is reached. The scheme is looked at first : an https remote may
// hold a user as well (https://user@host/repo), and taking it for an ssh one would send the user
// looking for a key instead of a token.
func ConnectionType(remoteURL string) string {
	if strings.HasPrefix(remoteURL, "http") {
		if gitUrl, err := url.Parse(remoteURL); err == nil {
			return strings.ToUpper(gitUrl.Scheme)
		}
		return ""
	}
	if strings.Contains(remoteURL, "@") {
		return "SSH"
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

	// The scheme decides, before the user : https://user@host/repo is authenticated with a token, not
	// with an ssh key
	if !strings.HasPrefix(origin, "http") && strings.Contains(origin, "@") {
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
//
// A missing ssh key used to be returned as a success, which made a repository Heimdall could not read
// look up to date. It is now reported as an AuthError, which the caller is expected to display
// without dropping the repository.
func FetchRepo(repo *git.Repository, spinner *tea.Program) (string, error) {
	connectionType, auth, err := authFor(repo, spinner)
	if err != nil {
		return connectionType, asAuthError(err, repo, connectionType)
	}
	return connectionType, asAuthError(repo.Fetch(&git.FetchOptions{Auth: auth}), repo, connectionType)
}

// Fetch fetches origin of the repository located in path
func Fetch(path string) error {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return err
	}
	connectionType, auth, err := authFor(repo, nil)
	if err != nil {
		return asAuthError(err, repo, connectionType)
	}
	return asAuthError(fetched(path, repo.Fetch(&git.FetchOptions{Auth: auth})), repo, connectionType)
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
	connectionType, auth, err := authFor(repo, nil)
	if err != nil {
		return asAuthError(err, repo, connectionType)
	}
	options := &git.PullOptions{RemoteName: commons.RemoteName, Auth: auth}
	// Without a reference, go-git pulls the HEAD of the remote instead of the current branch
	if head, err := repo.Head(); err == nil && head.Name().IsBranch() {
		options.ReferenceName = head.Name()
		if branch, err := repo.Branch(head.Name().Short()); err == nil && branch.Merge != "" {
			options.ReferenceName = branch.Merge
		}
	}
	return asAuthError(fetched(path, worktree.Pull(options)), repo, connectionType)
}

// fetched records a successful fetch, being already up-to-date is one
func fetched(path string, err error) error {
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	markFetched(path)
	return nil
}
