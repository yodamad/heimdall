package creds

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/google/go-github/v68/github"
	"github.com/yodamad/heimdall/utils"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// Status is the verdict on the credential of a platform
type Status int

const (
	// StatusUnknown : the platform could not be reached, so nothing can be said about its credential.
	// A flaky network must never be reported as a bad credential.
	StatusUnknown Status = iota
	// StatusNoToken : no token is configured for this platform
	StatusNoToken
	// StatusUnresolved : the configuration references an entry of the keyring or an environment
	// variable which holds nothing
	StatusUnresolved
	// StatusRejected : the platform refused the token, it is expired or revoked
	StatusRejected
	// StatusUnsupported : the type of the platform is neither github nor gitlab
	StatusUnsupported
	// StatusValid : the platform accepted the token
	StatusValid
	// StatusExpiring : the platform accepted the token, which expires soon
	StatusExpiring
)

// Label names a status in plain words
func (s Status) Label() string {
	switch s {
	case StatusNoToken:
		return "no token"
	case StatusUnresolved:
		return "unresolved"
	case StatusRejected:
		return "rejected"
	case StatusUnsupported:
		return "unsupported"
	case StatusValid:
		return "valid"
	case StatusExpiring:
		return "expiring"
	default:
		return "unreachable"
	}
}

// NeedsAttention tells whether the user has something to fix. An unreachable platform does not count:
// it is most probably the network.
func (s Status) NeedsAttention() bool {
	return s == StatusNoToken || s == StatusUnresolved || s == StatusRejected
}

// Result is what a check found out about one platform. Nothing in it ever holds a secret.
type Result struct {
	Host string
	Type string
	// Location says where the token is read from, always safe to display
	Location  string
	Status    Status
	User      string    // who the token authenticates as, when the platform says so
	ExpiresAt time.Time // zero when there is no expiry or it could not be read
	Scopes    []string  // gitlab only
	Detail    string    // sanitized, never the token
}

// Expired tells whether the expiry date is known and in the past
func (r Result) Expired() bool {
	return !r.ExpiresAt.IsZero() && r.ExpiresAt.Before(time.Now())
}

const (
	// DefaultExpiryWindow is how long before its expiry a token starts being reported as expiring
	DefaultExpiryWindow = 14 * 24 * time.Hour
	// checkTimeout caps one platform, so that a host which is down does not hold the whole check
	checkTimeout = 8 * time.Second
	// maxChecks is how many platforms are checked at the same time, as the TUI does for repositories
	maxChecks = 6
)

// CheckAll checks the credential of each given host, or of every configured platform when no host is
// given. The results are sorted by host.
//
// It only reads : nothing is written, in the configuration file or in the keyring.
func CheckAll(ctx context.Context, hosts []string, window time.Duration) []Result {
	wanted := make(map[string]bool, len(hosts))
	for _, host := range hosts {
		wanted[host] = true
	}

	var platforms []utils.PlatformInfo
	for _, info := range utils.Platforms() {
		if len(wanted) == 0 || wanted[info.Host] {
			platforms = append(platforms, info)
		}
	}

	results := make([]Result, len(platforms))
	workers := make(chan struct{}, maxChecks)
	var waitGroup sync.WaitGroup
	for i, info := range platforms {
		waitGroup.Add(1)
		go func(i int, info utils.PlatformInfo) {
			defer waitGroup.Done()
			workers <- struct{}{}
			defer func() { <-workers }()
			results[i] = check(ctx, info, window)
		}(i, info)
	}
	waitGroup.Wait()

	sort.Slice(results, func(i, j int) bool { return results[i].Host < results[j].Host })
	return results
}

// Check checks the credential of one host
func Check(ctx context.Context, host string, window time.Duration) Result {
	for _, info := range utils.Platforms() {
		if info.Host == host {
			return check(ctx, info, window)
		}
	}
	return Result{Host: host, Status: StatusNoToken, Location: "not set", Detail: host + " is not configured"}
}

func check(ctx context.Context, info utils.PlatformInfo, window time.Duration) Result {
	if window <= 0 {
		window = DefaultExpiryWindow
	}
	result := Result{Host: info.Host, Type: info.Type, Location: info.TokenLocation()}

	if info.TokenSource == utils.SourceNone {
		result.Status = StatusNoToken
		result.Detail = "no token configured for this platform"
		return result
	}
	if info.Type != "github" && info.Type != "gitlab" {
		result.Status = StatusUnsupported
		if info.Type == "" {
			result.Detail = "no type configured, expected github or gitlab"
		} else {
			result.Detail = "type " + info.Type + " is not supported, expected github or gitlab"
		}
		return result
	}

	// Resolving reads the keyring or the environment. A reference pointing at nothing is a problem
	// the user can fix, and it is told apart from a token the platform rejected. It is resolved
	// quietly : this function reports the outcome of every platform itself.
	token, _ := utils.TokenQuietly(info.Host)
	if token == "" {
		result.Status = StatusUnresolved
		result.Detail = "nothing to read at " + info.TokenLocation()
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	var err error
	if info.Type == "github" {
		err = checkGitHub(ctx, info.Host, token, &result)
	} else {
		err = checkGitLab(ctx, info.Host, token, &result)
	}
	if err != nil {
		result.Status, result.Detail = classify(err, info.Host)
		return result
	}

	switch {
	case result.Status == StatusRejected:
		// Set by the probe itself, for a token the platform reports as revoked or inactive
	case result.Expired():
		result.Status = StatusRejected
		result.Detail = "expired on " + result.ExpiresAt.Format(time.DateOnly)
	case !result.ExpiresAt.IsZero() && time.Until(result.ExpiresAt) <= window:
		result.Status = StatusExpiring
		result.Detail = "expires on " + result.ExpiresAt.Format(time.DateOnly) +
			", in " + strconv.Itoa(int(time.Until(result.ExpiresAt).Hours()/24)) + " days"
	default:
		result.Status = StatusValid
	}
	return result
}

func checkGitHub(ctx context.Context, host, token string, result *Result) error {
	client, err := GitHubClient(host, token, checkTimeout)
	if err != nil {
		return err
	}
	user, response, err := client.Users.Get(ctx, "")
	if err != nil {
		return err
	}
	if user != nil {
		result.User = user.GetLogin()
	}
	// GitHub reports the expiry of a personal access token in a header of every answer. The zero
	// value means the token does not expire, which is not a problem.
	if response != nil && !response.TokenExpiration.Equal(github.Timestamp{}) {
		result.ExpiresAt = response.TokenExpiration.Time
	}
	return nil
}

func checkGitLab(ctx context.Context, host, token string, result *Result) error {
	client, err := GitLabClient(host, token, checkTimeout)
	if err != nil {
		return err
	}
	user, _, err := client.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		return err
	}
	if user != nil {
		result.User = user.Username
	}

	// The expiry needs a second call, which only works for a personal access token : a group or
	// project token, or an older instance, answers 403 or 404. The token is valid either way, so this
	// is best effort and never downgrades the verdict.
	personalToken, _, err := client.PersonalAccessTokens.GetSinglePersonalAccessToken(gitlab.WithContext(ctx))
	if err != nil || personalToken == nil {
		result.Detail = "valid, expiry unknown"
		return nil
	}
	result.Scopes = personalToken.Scopes
	if personalToken.Revoked || !personalToken.Active {
		result.Status = StatusRejected
		result.Detail = "the platform reports this token as revoked"
		return nil
	}
	if personalToken.ExpiresAt != nil {
		result.ExpiresAt = time.Time(*personalToken.ExpiresAt)
	}
	return nil
}

// classify turns the error of a probe into a verdict. Only an answer from the platform refusing the
// token blames the credential : anything else is reported as unreachable, because telling a user
// their token is dead when the wifi dropped would be worse than saying nothing.
func classify(err error, host string) (Status, string) {
	if status, ok := statusCode(err); ok {
		switch status {
		case http.StatusUnauthorized, http.StatusForbidden:
			return StatusRejected, "the platform refused this token (HTTP " + strconv.Itoa(status) + ")"
		case http.StatusNotFound:
			// The API of the host answered, but not where expected : most often not a GitLab or
			// GitHub instance, or a wrong type in the configuration
			return StatusUnknown, "the API of " + host + " answered HTTP 404, check the type of this platform"
		default:
			return StatusUnknown, host + " answered HTTP " + strconv.Itoa(status)
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return StatusUnknown, host + " did not answer in time"
	}
	return StatusUnknown, "could not reach " + host
}

// statusCode digs the HTTP status out of the error types of the two clients
func statusCode(err error) (int, bool) {
	var githubErr *github.ErrorResponse
	if errors.As(err, &githubErr) && githubErr.Response != nil {
		return githubErr.Response.StatusCode, true
	}
	var gitlabErr *gitlab.ErrorResponse
	if errors.As(err, &gitlabErr) && gitlabErr.Response != nil {
		return gitlabErr.Response.StatusCode, true
	}
	// The gitlab client turns a 404 into a sentinel instead of an ErrorResponse
	if errors.Is(err, gitlab.ErrNotFound) {
		return http.StatusNotFound, true
	}
	return 0, false
}
