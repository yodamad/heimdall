package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/creds"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/gitops"
	"github.com/yodamad/heimdall/secrets"
	"github.com/yodamad/heimdall/utils"
)

// configureCreds points Heimdall at a configuration file of its own and uses a keyring held in memory,
// so that a test never touches the real keychain nor the real configuration
func configureCreds(t *testing.T, content string) string {
	t.Helper()
	secrets.UseMockForTests()

	path := filepath.Join(t.TempDir(), "heimdall.yml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	original := commons.InputConfigFile
	commons.InputConfigFile = path
	t.Cleanup(func() { commons.InputConfigFile = original })
	if err := utils.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	return path
}

func credsModel(t *testing.T) *model {
	t.Helper()
	commons.TUIMode = true
	m := newModel(t.TempDir()+"/", 1)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	return m
}

func TestCredentialsScreenOpensAndListsThePlatforms(t *testing.T) {
	configureCreds(t, "platforms:\n"+
		"  github.test:\n    type: github\n    token: env.HEIMDALL_TEST_ABSENT\n"+
		"  gitlab.test:\n    type: gitlab\n")
	os.Unsetenv("HEIMDALL_TEST_ABSENT")

	m := credsModel(t)
	press(m, "c")

	if m.mode != modeCreds {
		t.Fatalf("c should open the credentials screen, mode is %v", m.mode)
	}
	if len(m.creds) != 2 {
		t.Fatalf("expected the two platforms to be checked, got %d", len(m.creds))
	}

	view := m.View()
	for _, expected := range []string{"github.test", "gitlab.test", "unresolved", "no token", "credentials"} {
		if !strings.Contains(strings.ToLower(view), strings.ToLower(expected)) {
			t.Errorf("the screen should mention %q :\n%s", expected, view)
		}
	}

	press(m, "esc")
	if m.mode != modeList {
		t.Errorf("esc should go back to the list, mode is %v", m.mode)
	}
}

// The screen is displayed and its note goes into the log file : no secret may reach it
func TestCredentialsScreenNeverShowsTheSecret(t *testing.T) {
	const token = "sup3r-s3cret-token"
	configureCreds(t, "platforms:\n  github.test:\n    type: github\n    token: "+token+"\n")

	m := credsModel(t)
	press(m, "c")

	if view := m.View(); strings.Contains(view, token) {
		t.Errorf("the token written in the config file is displayed :\n%s", view)
	}
	if !strings.Contains(m.View(), "config file") {
		t.Errorf("the screen should say the token comes from the config file :\n%s", m.View())
	}
}

func TestCredentialsScreenStoresASecretForSeveralPlatforms(t *testing.T) {
	path := configureCreds(t, "platforms:\n"+
		"  gitlab.one:\n    type: gitlab\n    token: OLD # keep me\n"+
		"  gitlab.two:\n    type: gitlab\n")

	m := credsModel(t)
	press(m, "c")
	if len(m.creds) != 2 {
		t.Fatalf("expected two platforms, got %d", len(m.creds))
	}

	// Both platforms, one shared secret
	press(m, "a")
	if len(m.credPicked) != 2 {
		t.Fatalf("a should pick every platform, got %d", len(m.credPicked))
	}
	press(m, "enter")
	if m.credStep != credName {
		t.Fatalf("enter should ask for the name of the keyring entry, step is %v", m.credStep)
	}

	// Name the entry, then type the secret
	m.credInput.SetValue("corp")
	press(m, "enter")
	if m.credStep != credSecret {
		t.Fatalf("expected the secret to be asked for, step is %v", m.credStep)
	}
	m.credInput.SetValue("a-brand-new-token")
	press(m, "enter")

	// The secret must not be left anywhere on the model
	if value := m.credInput.Value(); value != "" {
		t.Errorf("the input should be emptied as soon as the secret is read, holds %q", value)
	}

	stored, err := secrets.Get("corp")
	if err != nil {
		t.Fatalf("the secret should be in the keyring : %v", err)
	}
	if stored != "a-brand-new-token" {
		t.Errorf("unexpected secret stored : %q", stored)
	}

	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(written)
	if count := strings.Count(content, "token: keyring.corp"); count != 2 {
		t.Errorf("both platforms should point at the entry, got %d :\n%s", count, content)
	}
	if strings.Contains(content, "a-brand-new-token") {
		t.Errorf("the secret must never be written in the config file :\n%s", content)
	}
	if !strings.Contains(content, "# keep me") {
		t.Errorf("the comments of the file should survive :\n%s", content)
	}
}

func TestCredentialsScreenRefusesAnInvalidEntryName(t *testing.T) {
	configureCreds(t, "platforms:\n  github.test:\n    type: github\n")

	m := credsModel(t)
	press(m, "c", " ", "enter")
	m.credInput.SetValue("not a name")
	press(m, "enter")

	if m.credStep != credName {
		t.Errorf("an invalid name should keep asking, step is %v", m.credStep)
	}
	if !strings.Contains(m.credNote, "letters, digits") {
		t.Errorf("the screen should say why, note is %q", m.credNote)
	}
}

func TestCredentialsScreenRefusesAnEmptySecret(t *testing.T) {
	configureCreds(t, "platforms:\n  github.test:\n    type: github\n")

	m := credsModel(t)
	press(m, "c", " ", "enter")
	m.credInput.SetValue("github.test")
	press(m, "enter")
	m.credInput.SetValue("")
	press(m, "enter")

	if m.credStep != credSecret {
		t.Errorf("an empty secret should keep asking, step is %v", m.credStep)
	}
	if _, err := secrets.Get("github.test"); err == nil {
		t.Error("an empty secret must not be stored")
	}
}

// Escaping out of the secret must not store anything : the step before used to be indistinguishable
func TestCredentialsScreenCancelsWithoutStoringAnything(t *testing.T) {
	path := configureCreds(t, "platforms:\n  github.test:\n    type: github\n")
	before, _ := os.ReadFile(path)

	m := credsModel(t)
	press(m, "c", " ", "enter")
	m.credInput.SetValue("github.test")
	press(m, "enter")
	m.credInput.SetValue("typed-then-cancelled")
	press(m, "esc")

	if m.credStep != credName {
		t.Errorf("esc should go back one step, step is %v", m.credStep)
	}
	press(m, "esc")
	if m.credStep != credPick {
		t.Errorf("esc should go back to the list of platforms, step is %v", m.credStep)
	}
	if _, err := secrets.Get("github.test"); err == nil {
		t.Error("a cancelled prompt must not store anything")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("a cancelled prompt must not change the config file")
	}
}

// A fetch which failed on the credential offers to fix it instead of only showing a git error
func TestAuthenticationFailureOffersToUpdateTheCredential(t *testing.T) {
	configureCreds(t, "platforms:\n  github.test:\n    type: github\n    token: a-token\n")

	m := credsModel(t)
	m.rows = []*row{{gf: gitFolderOn("https://github.test/yodamad/heimdall.git"), name: "heimdall", loaded: true}}
	m.byPath = map[string]*row{m.rows[0].gf.Path: m.rows[0]}

	authErr := &gitops.AuthError{Host: "github.test", Kind: gitops.KindToken, Sure: true, Err: transport.ErrAuthorizationFailed}
	m.Update(opMsg{path: m.rows[0].gf.Path, kind: busyFetch, err: authErr})

	if m.credOffer != "github.test" {
		t.Fatalf("the failure should offer to update github.test, got %q", m.credOffer)
	}
	if !strings.Contains(m.status, "Press c") || !strings.Contains(m.status, "github.test") {
		t.Errorf("the status should offer the fix, got %q", m.status)
	}

	press(m, "c")
	if m.mode != modeCreds {
		t.Fatalf("c should open the credentials screen, mode is %v", m.mode)
	}
	if !m.credPicked["github.test"] {
		t.Error("the platform which failed should already be picked")
	}
}

// A repository which is simply gone must not be reported as an expired token
func TestRepositoryNotFoundIsOnlyAHint(t *testing.T) {
	configureCreds(t, "platforms:\n  github.test:\n    type: github\n    token: a-token\n")

	authErr := gitops.HostOf("https://github.test/yodamad/gone.git")
	if authErr != "github.test" {
		t.Fatalf("unexpected host %q", authErr)
	}
	// The classifier marks it as not sure, so the wording hedges
	hint := &gitops.AuthError{Host: "github.test", Kind: gitops.KindToken, Sure: false, Err: transport.ErrRepositoryNotFound}
	if !strings.Contains(hint.Error(), "may have expired") {
		t.Errorf("a 404 should only be a hint, got %q", hint.Error())
	}
	sure := &gitops.AuthError{Host: "github.test", Kind: gitops.KindToken, Sure: true, Err: transport.ErrAuthorizationFailed}
	if strings.Contains(sure.Error(), "may have expired") {
		t.Errorf("a refused token is not a hint, got %q", sure.Error())
	}
}

func TestCredentialsScreenSaysWhenNoPlatformIsConfigured(t *testing.T) {
	configureCreds(t, "work_dir: /tmp\n")

	m := credsModel(t)
	press(m, "c")

	if len(m.creds) != 0 {
		t.Fatalf("expected no platform, got %d", len(m.creds))
	}
	if view := m.View(); !strings.Contains(view, "No platform is configured") {
		t.Errorf("the screen should say there is nothing configured :\n%s", view)
	}
	// Asking for an update with nothing to update must not crash nor prompt
	press(m, "enter")
	if m.credStep != credPick {
		t.Errorf("there is nothing to update, step is %v", m.credStep)
	}
}

func TestCheckResultLabels(t *testing.T) {
	for status, expected := range map[creds.Status]string{
		creds.StatusValid:       "valid",
		creds.StatusExpiring:    "expiring",
		creds.StatusRejected:    "rejected",
		creds.StatusNoToken:     "no token",
		creds.StatusUnresolved:  "unresolved",
		creds.StatusUnsupported: "unsupported",
		creds.StatusUnknown:     "unreachable",
	} {
		if label := status.Label(); label != expected {
			t.Errorf("expected %q, got %q", expected, label)
		}
	}
}

// gitFolderOn builds a repository with an origin on this URL, which is all these tests need
func gitFolderOn(remoteURL string) entity.GitFolder {
	return entity.GitFolder{Path: "/tmp/heimdall-test-repo", RemoteURL: remoteURL, CurrentBranch: "main"}
}
