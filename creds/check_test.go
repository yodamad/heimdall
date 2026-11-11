package creds

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils"
)

// configure points Heimdall at a configuration file holding only what the test needs
func configure(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "heimdall.yml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	original := commons.InputConfigFile
	commons.InputConfigFile = path
	t.Cleanup(func() { commons.InputConfigFile = original })
	if err := utils.ReloadConfig(); err != nil {
		t.Fatalf("cannot read the test configuration : %v", err)
	}
}

// serve starts a stand-in for the API of a platform and makes the checker talk to it
func serve(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	original := apiURL
	apiURL = func(string) string { return server.URL }
	t.Cleanup(func() { apiURL = original })
}

const githubConfig = "platforms:\n  github.test:\n    type: github\n    token: a-token\n"
const gitlabConfig = "platforms:\n  gitlab.test:\n    type: gitlab\n    token: a-token\n"

func TestCheckAcceptsAValidGitHubToken(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/user" {
			t.Errorf("unexpected path %s : the enterprise base URL is wrong", r.URL.Path)
		}
		w.Write([]byte(`{"login":"yodamad"}`))
	})
	configure(t, githubConfig)

	result := Check(context.Background(), "github.test", 0)
	if result.Status != StatusValid {
		t.Errorf("expected StatusValid, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if result.User != "yodamad" {
		t.Errorf("expected the login of the token, got %q", result.User)
	}
}

func TestCheckReportsAGitHubTokenExpiringSoon(t *testing.T) {
	expiry := time.Now().Add(3 * 24 * time.Hour)
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Github-Authentication-Token-Expiration", expiry.UTC().Format("2006-01-02 15:04:05 MST"))
		w.Write([]byte(`{"login":"yodamad"}`))
	})
	configure(t, githubConfig)

	result := Check(context.Background(), "github.test", DefaultExpiryWindow)
	if result.Status != StatusExpiring {
		t.Errorf("expected StatusExpiring, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if result.ExpiresAt.IsZero() {
		t.Error("expected the expiry date to be reported")
	}
}

func TestCheckReportsAnExpiredGitHubToken(t *testing.T) {
	expiry := time.Now().Add(-24 * time.Hour)
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Github-Authentication-Token-Expiration", expiry.UTC().Format("2006-01-02 15:04:05 MST"))
		w.Write([]byte(`{"login":"yodamad"}`))
	})
	configure(t, githubConfig)

	result := Check(context.Background(), "github.test", 0)
	if result.Status != StatusRejected {
		t.Errorf("expected StatusRejected, got %s (%s)", result.Status.Label(), result.Detail)
	}
}

func TestCheckKeepsAGitHubTokenWithNoExpiryValid(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"login":"yodamad"}`))
	})
	configure(t, githubConfig)

	result := Check(context.Background(), "github.test", DefaultExpiryWindow)
	if result.Status != StatusValid {
		t.Errorf("a token which never expires is valid, got %s", result.Status.Label())
	}
}

func TestCheckRejectsARefusedToken(t *testing.T) {
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		serve(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			w.Write([]byte(`{"message":"Bad credentials"}`))
		})
		configure(t, githubConfig)

		result := Check(context.Background(), "github.test", 0)
		if result.Status != StatusRejected {
			t.Errorf("HTTP %d : expected StatusRejected, got %s (%s)", code, result.Status.Label(), result.Detail)
		}
	}
}

// The whole point of telling these apart : a platform which is down must not be reported as a dead
// credential, or the user revokes a token which was fine
func TestCheckDoesNotBlameTheCredentialForAServerProblem(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	configure(t, githubConfig)

	result := Check(context.Background(), "github.test", 0)
	if result.Status != StatusUnknown {
		t.Errorf("expected StatusUnknown, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if result.Status.NeedsAttention() {
		t.Error("an unreachable platform is not something the user has to fix")
	}
}

func TestCheckDoesNotBlameTheCredentialForAnUnreachableHost(t *testing.T) {
	original := apiURL
	// A port nothing listens on
	apiURL = func(string) string { return "http://127.0.0.1:1" }
	t.Cleanup(func() { apiURL = original })
	configure(t, githubConfig)

	result := Check(context.Background(), "github.test", 0)
	if result.Status != StatusUnknown {
		t.Errorf("expected StatusUnknown, got %s (%s)", result.Status.Label(), result.Detail)
	}
}

func TestCheckAcceptsAValidGitLabToken(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api/v4/user"):
			w.Write([]byte(`{"username":"yodamad"}`))
		case strings.HasSuffix(r.URL.Path, "/personal_access_tokens/self"):
			w.Write([]byte(`{"active":true,"revoked":false,"scopes":["api"],"expires_at":null}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	configure(t, gitlabConfig)

	result := Check(context.Background(), "gitlab.test", 0)
	if result.Status != StatusValid {
		t.Errorf("expected StatusValid, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if result.User != "yodamad" {
		t.Errorf("expected the username of the token, got %q", result.User)
	}
}

func TestCheckRejectsARevokedGitLabToken(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/api/v4/user"):
			w.Write([]byte(`{"username":"yodamad"}`))
		default:
			w.Write([]byte(`{"active":false,"revoked":true}`))
		}
	})
	configure(t, gitlabConfig)

	result := Check(context.Background(), "gitlab.test", 0)
	if result.Status != StatusRejected {
		t.Errorf("expected StatusRejected, got %s (%s)", result.Status.Label(), result.Detail)
	}
}

// A group or project token, or an older instance, has no /personal_access_tokens/self : the token is
// still valid, only its expiry is unknown
func TestCheckKeepsAGitLabTokenValidWhenItsExpiryCannotBeRead(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/api/v4/user") {
			w.Write([]byte(`{"username":"a-group-token"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	configure(t, gitlabConfig)

	result := Check(context.Background(), "gitlab.test", 0)
	if result.Status != StatusValid {
		t.Errorf("expected StatusValid, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if !strings.Contains(result.Detail, "expiry unknown") {
		t.Errorf("expected the detail to say the expiry is unknown, got %q", result.Detail)
	}
}

func TestCheckReportsAMissingToken(t *testing.T) {
	configure(t, "platforms:\n  github.test:\n    type: github\n")

	result := Check(context.Background(), "github.test", 0)
	if result.Status != StatusNoToken {
		t.Errorf("expected StatusNoToken, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if !result.Status.NeedsAttention() {
		t.Error("a missing token is something the user has to fix")
	}
}

func TestCheckReportsAnUnresolvedReference(t *testing.T) {
	os.Unsetenv("HEIMDALL_TEST_ABSENT")
	configure(t, "platforms:\n  github.test:\n    type: github\n    token: env.HEIMDALL_TEST_ABSENT\n")

	result := Check(context.Background(), "github.test", 0)
	if result.Status != StatusUnresolved {
		t.Errorf("expected StatusUnresolved, got %s (%s)", result.Status.Label(), result.Detail)
	}
	if !strings.Contains(result.Detail, "env: HEIMDALL_TEST_ABSENT") {
		t.Errorf("expected the detail to name where it looked, got %q", result.Detail)
	}
}

func TestCheckReportsAnUnsupportedType(t *testing.T) {
	configure(t, "platforms:\n  bitbucket.test:\n    type: bitbucket\n    token: a-token\n")

	result := Check(context.Background(), "bitbucket.test", 0)
	if result.Status != StatusUnsupported {
		t.Errorf("expected StatusUnsupported, got %s (%s)", result.Status.Label(), result.Detail)
	}
}

func TestCheckReportsAMissingType(t *testing.T) {
	configure(t, "platforms:\n  somewhere.test:\n    token: a-token\n")

	result := Check(context.Background(), "somewhere.test", 0)
	if result.Status != StatusUnsupported {
		t.Errorf("expected StatusUnsupported, got %s (%s)", result.Status.Label(), result.Detail)
	}
}

func TestCheckAllChecksEveryPlatformAndSortsThem(t *testing.T) {
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "personal_access_tokens") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"login":"yodamad","username":"yodamad"}`))
	})
	configure(t, "platforms:\n"+
		"  zzz.test:\n    type: github\n    token: a-token\n"+
		"  aaa.test:\n    type: gitlab\n    token: a-token\n"+
		"  mmm.test:\n    type: github\n")

	results := CheckAll(context.Background(), nil, 0)
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	if results[0].Host != "aaa.test" || results[1].Host != "mmm.test" || results[2].Host != "zzz.test" {
		t.Errorf("expected the results sorted by host, got %s, %s, %s", results[0].Host, results[1].Host, results[2].Host)
	}
	if results[1].Status != StatusNoToken {
		t.Errorf("expected the platform with no token to be reported, got %s", results[1].Status.Label())
	}
}

func TestCheckAllNarrowsToTheGivenHosts(t *testing.T) {
	configure(t, "platforms:\n"+
		"  one.test:\n    type: github\n"+
		"  two.test:\n    type: github\n")

	results := CheckAll(context.Background(), []string{"two.test"}, 0)
	if len(results) != 1 || results[0].Host != "two.test" {
		t.Errorf("expected only two.test, got %v", results)
	}
}

// The results are displayed and written into the log file
func TestResultsNeverHoldTheToken(t *testing.T) {
	const token = "sup3r-s3cret-token"
	serve(t, func(w http.ResponseWriter, r *http.Request) {
		// A platform echoing the token back must not make it into the result either
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"bad token ` + token + `"}`))
	})
	configure(t, "platforms:\n  github.test:\n    type: github\n    token: "+token+"\n")

	result := Check(context.Background(), "github.test", 0)
	for name, field := range map[string]string{
		"Detail":   result.Detail,
		"Location": result.Location,
		"User":     result.User,
		"Host":     result.Host,
		"Type":     result.Type,
	} {
		if strings.Contains(field, token) {
			t.Errorf("%s held the token : %q", name, field)
		}
	}
}
