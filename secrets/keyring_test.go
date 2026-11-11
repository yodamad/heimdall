package secrets

import (
	"errors"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestSetGetDelete(t *testing.T) {
	UseMockForTests()

	if err := Set("corp-gitlab", "s3cret"); err != nil {
		t.Fatalf("Set failed : %v", err)
	}
	value, err := Get("corp-gitlab")
	if err != nil {
		t.Fatalf("Get failed : %v", err)
	}
	if value != "s3cret" {
		t.Errorf("expected the stored secret back, got %q", value)
	}

	if err := Delete("corp-gitlab"); err != nil {
		t.Fatalf("Delete failed : %v", err)
	}
	if _, err := Get("corp-gitlab"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after a delete, got %v", err)
	}
}

func TestGetReportsAMissingEntry(t *testing.T) {
	UseMockForTests()

	if _, err := Get("never-stored"); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// The entry is read once per repository on every fetch, and reading it costs a sub process on macOS
func TestGetIsCached(t *testing.T) {
	UseMockForTests()

	if err := Set("cached", "first"); err != nil {
		t.Fatal(err)
	}
	// Changing the entry behind the cache : the cached value has to be served
	if err := keyring.Set(Service, "cached", "second"); err != nil {
		t.Fatal(err)
	}
	if value, _ := Get("cached"); value != "first" {
		t.Errorf("expected the cached value, got %q", value)
	}

	Forget("cached")
	if value, _ := Get("cached"); value != "second" {
		t.Errorf("expected the new value after Forget, got %q", value)
	}
}

// A missing entry must not be cached, so that an entry added from elsewhere is picked up
func TestGetDoesNotCacheAMissingEntry(t *testing.T) {
	UseMockForTests()

	if _, err := Get("added-later"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := keyring.Set(Service, "added-later", "here-now"); err != nil {
		t.Fatal(err)
	}
	if value, err := Get("added-later"); err != nil || value != "here-now" {
		t.Errorf("expected the entry added behind our back, got %q (%v)", value, err)
	}
}

func TestNoKeyringAvailable(t *testing.T) {
	FailForTests(keyring.ErrUnsupportedPlatform)
	defer UseMockForTests()

	if Available() {
		t.Error("expected no keyring to be available")
	}
	if _, err := Get("anything"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("expected ErrUnavailable on read, got %v", err)
	}
	if err := Set("anything", "value"); !errors.Is(err, ErrUnavailable) {
		t.Errorf("expected ErrUnavailable on write, got %v", err)
	}
}

func TestInvalidEntryNames(t *testing.T) {
	UseMockForTests()

	// The name is written into the configuration file, so it is kept to characters which cannot
	// break it, and which stay readable
	for _, key := range []string{"", "   ", "a: b", "../escape", "with space", "new\nline", strings.Repeat("x", 65)} {
		if ValidKey(key) {
			t.Errorf("%q should not be a valid entry name", key)
		}
		if err := Set(key, "value"); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Set(%q) : expected ErrInvalidKey, got %v", key, err)
		}
		if _, err := Get(key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("Get(%q) : expected ErrInvalidKey, got %v", key, err)
		}
	}

	for _, key := range []string{"github.com", "corp-gitlab", "corp_gitlab", "GL2", strings.Repeat("x", 64)} {
		if !ValidKey(key) {
			t.Errorf("%q should be a valid entry name", key)
		}
	}
}

func TestSetRefusesAnEmptyOrOversizedSecret(t *testing.T) {
	UseMockForTests()

	if err := Set("empty", ""); err == nil {
		t.Error("expected an empty secret to be refused")
	}
	if err := Set("huge", strings.Repeat("x", MaxValueLength+1)); !errors.Is(err, ErrTooBig) {
		t.Errorf("expected ErrTooBig, got %v", err)
	}
}

// Whatever goes wrong, the value of an entry must never end up in an error : they are logged
func TestErrorsNeverHoldTheSecret(t *testing.T) {
	UseMockForTests()
	const secret = "sup3r-s3cret-token"

	if err := Set("held", secret); err != nil {
		t.Fatal(err)
	}
	_, getErr := Get("absent")
	errs := []error{
		getErr,
		Set("bad name", secret),
		Set("huge", strings.Repeat(secret, 500)),
		Delete("bad name"),
	}
	for _, err := range errs {
		if err != nil && strings.Contains(err.Error(), secret) {
			t.Errorf("an error held the secret : %v", err)
		}
	}
}
