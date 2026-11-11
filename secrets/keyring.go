// Package secrets stores the credentials of the platforms in the keyring of the operating system,
// so that they do not have to be written in the configuration file.
//
// The configuration file only holds a reference to an entry of the keyring, as the "keyring." prefix
// followed by the name of the entry. Several platforms can point at the same entry, which is how one
// token shared by several hosts is rotated in one place.
package secrets

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/zalando/go-keyring"
)

// Service is the name Heimdall stores its entries under in the keyring of the operating system
const Service = "heimdall"

// probeKey is read, never written, to find out whether a keyring is reachable
const probeKey = "heimdall-availability-probe"

// MaxValueLength is the longest secret accepted. The keychain of macOS takes about 3000 bytes for the
// service, the name and the value together, and no token comes close to this.
const MaxValueLength = 2048

var (
	// ErrUnavailable is returned when the operating system has no keyring Heimdall can reach :
	// an unsupported platform, or a Linux box with no Secret Service running
	ErrUnavailable = errors.New("no keyring available on this system")
	// ErrNotFound is returned when the keyring holds no entry under this name
	ErrNotFound = keyring.ErrNotFound
	// ErrTooBig is returned when the value does not fit in the keyring of the operating system
	ErrTooBig = fmt.Errorf("a secret cannot be longer than %d characters", MaxValueLength)
	// ErrInvalidKey is returned for a name which is blank or holds unexpected characters
	ErrInvalidKey = errors.New("the name of a keyring entry may only hold letters, digits, dots, dashes and underscores")
)

// validKey guards the only user chosen string which is written into the configuration file
var validKey = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// ValidKey tells whether a name can be used for an entry of the keyring
func ValidKey(key string) bool { return validKey.MatchString(key) }

var (
	probeOnce sync.Once
	available bool
)

// Reading an entry costs a sub process on macOS, and a token is read once per repository on every
// fetch or pull, six of them at a time : without this cache, a fetch on fifty repositories would
// fork /usr/bin/security fifty times. Only values which were found are kept, so that an entry added
// from elsewhere is still picked up.
var (
	cacheMutex sync.RWMutex
	cache      = map[string]string{}
)

// Available tells whether a keyring of the operating system can be reached. It is probed once with a
// read of an entry which is never written, so it neither creates nor changes anything.
//
// Heimdall must stay usable when it returns false : the platforms configured with a literal value or
// an "env." reference keep working, only storing into the keyring is refused.
func Available() bool {
	probeOnce.Do(func() {
		_, err := keyring.Get(Service, probeKey)
		// Finding no entry means the keyring answered, which is all this probe asks of it
		available = err == nil || errors.Is(err, keyring.ErrNotFound)
	})
	return available
}

// Get reads the value of an entry of the keyring. It returns ErrNotFound when there is no such entry
// and ErrUnavailable when there is no keyring to read from.
func Get(key string) (string, error) {
	if err := checkKey(key); err != nil {
		return "", err
	}
	cacheMutex.RLock()
	value, cached := cache[key]
	cacheMutex.RUnlock()
	if cached {
		return value, nil
	}

	if !Available() {
		return "", ErrUnavailable
	}
	value, err := keyring.Get(Service, key)
	if err != nil {
		return "", wrap(key, err)
	}
	cacheMutex.Lock()
	cache[key] = value
	cacheMutex.Unlock()
	return value, nil
}

// Set writes the value of an entry of the keyring, replacing it if it already exists
func Set(key, value string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	if value == "" {
		return errors.New("refusing to store an empty secret")
	}
	if len(value) > MaxValueLength {
		return ErrTooBig
	}
	if !Available() {
		return ErrUnavailable
	}
	if err := keyring.Set(Service, key, value); err != nil {
		Forget(key)
		return wrap(key, err)
	}
	cacheMutex.Lock()
	cache[key] = value
	cacheMutex.Unlock()
	return nil
}

// Delete removes an entry from the keyring
func Delete(key string) error {
	if err := checkKey(key); err != nil {
		return err
	}
	Forget(key)
	if !Available() {
		return ErrUnavailable
	}
	return wrap(key, keyring.Delete(Service, key))
}

// Forget drops an entry from the cache, so that the next read goes to the keyring again
func Forget(key string) {
	cacheMutex.Lock()
	delete(cache, key)
	cacheMutex.Unlock()
}

// ForgetAll empties the cache
func ForgetAll() {
	cacheMutex.Lock()
	cache = map[string]string{}
	cacheMutex.Unlock()
}

func checkKey(key string) error {
	if !validKey.MatchString(strings.TrimSpace(key)) {
		return fmt.Errorf("%q : %w", key, ErrInvalidKey)
	}
	return nil
}

// wrap names the entry the error is about, and translates the ones of the keyring library into the
// ones of this package. The value of an entry is never part of an error of the library, so wrapping
// cannot leak a secret.
func wrap(key string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, keyring.ErrNotFound):
		return fmt.Errorf("%s : %w", key, ErrNotFound)
	case errors.Is(err, keyring.ErrSetDataTooBig):
		return fmt.Errorf("%s : %w", key, ErrTooBig)
	case errors.Is(err, keyring.ErrUnsupportedPlatform):
		return fmt.Errorf("%s : %w", key, ErrUnavailable)
	default:
		return fmt.Errorf("keyring entry %s : %w", key, err)
	}
}

// UseMockForTests replaces the keyring of the operating system with one held in memory, so that a
// test never reads or writes the real keychain of whoever runs it. It also makes the tests pass on a
// build machine which has no keyring at all.
func UseMockForTests() {
	keyring.MockInit()
	probeOnce = sync.Once{}
	available = false
	ForgetAll()
}

// FailForTests makes every operation fail with the given error, to exercise the paths where no
// keyring is available
func FailForTests(err error) {
	keyring.MockInitWithError(err)
	probeOnce = sync.Once{}
	available = false
	ForgetAll()
}
