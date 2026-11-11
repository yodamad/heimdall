package creds

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const configWithComments = `work_dir: /tmp/work
platforms:
  gitlab.com:
    token: OLD_TOKEN # Bad !!
    type: gitlab
  github.com:
    token: env.ENV_VAR_TOKEN
    type: github
morning_routine:
  shell: zsh
  commands: git fetch, git pull
`

func write(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "heimdall.yml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSetPlatformFieldKeepsCommentsAndTheRestOfTheFile(t *testing.T) {
	path := write(t, configWithComments)

	if err := SetPlatformField(path, "gitlab.com", FieldToken, "keyring.corp"); err != nil {
		t.Fatalf("SetPlatformField failed : %v", err)
	}

	after := read(t, path)
	if !strings.Contains(after, "token: keyring.corp") {
		t.Errorf("the token was not updated :\n%s", after)
	}
	if strings.Contains(after, "OLD_TOKEN") {
		t.Errorf("the previous token is still there :\n%s", after)
	}
	if !strings.Contains(after, "# Bad !!") {
		t.Errorf("the comment of the line was dropped :\n%s", after)
	}
	// The rest of the file has to survive untouched
	for _, kept := range []string{"work_dir: /tmp/work", "env.ENV_VAR_TOKEN", "shell: zsh", "commands: git fetch, git pull"} {
		if !strings.Contains(after, kept) {
			t.Errorf("%q was lost :\n%s", kept, after)
		}
	}
	// Two space indentation, as in the documented sample
	if !strings.Contains(after, "\n  gitlab.com:\n") {
		t.Errorf("expected a two space indentation :\n%s", after)
	}
}

func TestSetPlatformFieldsWritesEveryHostInOnePass(t *testing.T) {
	path := write(t, configWithComments)

	hosts := []string{"gitlab.com", "github.com"}
	if err := SetPlatformFields(path, hosts, FieldToken, "keyring.shared"); err != nil {
		t.Fatalf("SetPlatformFields failed : %v", err)
	}

	after := read(t, path)
	if count := strings.Count(after, "token: keyring.shared"); count != 2 {
		t.Errorf("expected both hosts to point at the same entry, got %d :\n%s", count, after)
	}
}

func TestSetPlatformFieldAddsAnUnknownHost(t *testing.T) {
	path := write(t, configWithComments)

	if err := SetPlatformField(path, "gitlab.corp.com", FieldType, "gitlab"); err != nil {
		t.Fatalf("SetPlatformField failed : %v", err)
	}
	if err := SetPlatformField(path, "gitlab.corp.com", FieldToken, "keyring.corp"); err != nil {
		t.Fatalf("SetPlatformField failed : %v", err)
	}

	after := read(t, path)
	if !strings.Contains(after, "gitlab.corp.com:") || !strings.Contains(after, "keyring.corp") {
		t.Errorf("the new platform was not added :\n%s", after)
	}
}

// The configuration is read through viper, which lowercases the keys, so a host written with capitals
// in the file has to be found instead of being added a second time
func TestSetPlatformFieldMatchesAnExistingHostWithoutCase(t *testing.T) {
	path := write(t, "platforms:\n  GitLab.com:\n    type: gitlab\n    token: OLD\n")

	if err := SetPlatformField(path, "gitlab.com", FieldToken, "keyring.corp"); err != nil {
		t.Fatalf("SetPlatformField failed : %v", err)
	}

	after := read(t, path)
	if strings.Contains(after, "\n  gitlab.com:") {
		t.Errorf("a second entry was added for the same host :\n%s", after)
	}
	if !strings.Contains(after, "GitLab.com:") || !strings.Contains(after, "keyring.corp") {
		t.Errorf("the existing entry was not updated :\n%s", after)
	}
}

func TestSetPlatformFieldCreatesTheFileAndKeepsItPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "heimdall.yml")

	if err := SetPlatformField(path, "github.com", FieldToken, "keyring.gh"); err != nil {
		t.Fatalf("SetPlatformField failed : %v", err)
	}

	after := read(t, path)
	if !strings.Contains(after, "platforms:") || !strings.Contains(after, "keyring.gh") {
		t.Errorf("unexpected content :\n%s", after)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected the config file to be 0600, got %#o", perm)
	}
}

func TestSetPlatformFieldRefusesAnUnknownField(t *testing.T) {
	path := write(t, configWithComments)
	before := read(t, path)

	err := SetPlatformField(path, "gitlab.com", "password", "whatever")
	if !errors.Is(err, ErrUnknownField) {
		t.Errorf("expected ErrUnknownField, got %v", err)
	}
	if read(t, path) != before {
		t.Error("the file was changed although the field was refused")
	}
}

func TestSetPlatformFieldsLeavesTheFileAloneOnAnInvalidHost(t *testing.T) {
	path := write(t, configWithComments)
	before := read(t, path)

	if err := SetPlatformFields(path, []string{"gitlab.com", "  "}, FieldToken, "keyring.corp"); err == nil {
		t.Error("expected an error for an empty host")
	}
	if read(t, path) != before {
		t.Error("a bulk update which failed must not change the file at all")
	}
}

func TestSetPlatformFieldReportsABrokenFile(t *testing.T) {
	path := write(t, "platforms:\n\tthis is not: [valid\n")

	if err := SetPlatformField(path, "gitlab.com", FieldToken, "keyring.corp"); err == nil {
		t.Error("expected an error for a file which is not yaml")
	}
}

func TestEnsureWritable(t *testing.T) {
	path := write(t, configWithComments)
	if err := EnsureWritable(path); err != nil {
		t.Errorf("expected a writable file, got %v", err)
	}

	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)
	if err := EnsureWritable(path); err == nil {
		t.Error("expected a read-only file to be refused")
	}
}

func TestEnsureWritableRefusesAReadOnlyDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes in a read-only directory anyway")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0700)

	if err := EnsureWritable(filepath.Join(dir, "heimdall.yml")); err == nil {
		t.Error("expected a read-only directory to be refused")
	}
}

// A configuration file linked from a dotfiles repository has to be updated where it really lives :
// replacing the link with a regular file would leave the old token in the dotfiles copy
func TestSetPlatformFieldFollowsASymlinkedConfig(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real-heimdall.yml")
	if err := os.WriteFile(real, []byte(configWithComments), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "heimdall.yml")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks not available : %v", err)
	}

	if err := SetPlatformField(link, "gitlab.com", FieldToken, "keyring.corp"); err != nil {
		t.Fatalf("SetPlatformField failed : %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced by a regular file")
	}
	if !strings.Contains(read(t, real), "token: keyring.corp") {
		t.Errorf("the file the link points at was not updated :\n%s", read(t, real))
	}
}

// Nothing must be left holding a copy of the configuration once a write is done
func TestSetPlatformFieldLeavesNoTemporaryFileBehind(t *testing.T) {
	path := write(t, configWithComments)

	if err := SetPlatformField(path, "gitlab.com", FieldToken, "keyring.corp"); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".heimdall-") {
			t.Errorf("a temporary copy of the configuration was left behind : %s", entry.Name())
		}
	}
}
