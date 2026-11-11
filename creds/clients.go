package creds

import (
	"net/http"
	"time"

	"github.com/google/go-github/v68/github"
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// The hosts of the public instances, the only ones whose API lives somewhere else than on the host
// of the repositories
const (
	publicGitHub = "github.com"
	publicGitLab = "gitlab.com"
)

// apiURL builds the base URL of the API of a host. It is a variable so that the tests can point it
// at a local server.
var apiURL = func(host string) string { return "https://" + host }

// GitHubClient builds a client for a host, which is api.github.com for the public instance and the
// /api/v3/ endpoint of the host itself for a GitHub Enterprise one.
//
// Every call needing a GitHub client has to go through this : building the client without the base
// URL would silently talk to github.com with the token of another host.
func GitHubClient(host, token string, timeout time.Duration) (*github.Client, error) {
	client := github.NewClient(&http.Client{Timeout: timeout}).WithAuthToken(token)
	if host == publicGitHub {
		return client, nil
	}
	return client.WithEnterpriseURLs(apiURL(host), apiURL(host))
}

// GitLabClient builds a client for a host, pointing at the /api/v4 endpoint of the host itself for a
// self hosted instance.
//
// Retries are turned off on purpose : the client retries five times with a backoff by default, which
// would turn a short timeout into minutes of waiting on a host which is down.
func GitLabClient(host, token string, timeout time.Duration) (*gitlab.Client, error) {
	options := []gitlab.ClientOptionFunc{
		gitlab.WithHTTPClient(&http.Client{Timeout: timeout}),
		gitlab.WithoutRetries(),
	}
	if host != publicGitLab {
		options = append(options, gitlab.WithBaseURL(apiURL(host)))
	}
	return gitlab.NewClient(token, options...)
}
