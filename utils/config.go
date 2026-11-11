package utils

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/secrets"
	"github.com/yodamad/heimdall/utils/tui"
)

type Platform struct {
	typeOf            string
	token             string
	publicKey         string
	publicKeyPassword string
}

type MorningRoutine struct {
	Cmds                 []string
	Shell                string
	initialized, ohmyzsh bool
}

var ConfiguredPlatforms = make(map[string]Platform)
var MorningRoutineConfig MorningRoutine

// platformsMutex guards ConfiguredPlatforms. The TUI reads it from several workers at a time while
// a credential update can rebuild it, so every read and the rebuild have to be guarded.
var platformsMutex sync.RWMutex

// alreadyWarned remembers the platforms whose configuration was already reported as incomplete, so
// that rebuilding the platforms after an update does not report them again and again
var alreadyWarned = make(map[string]bool)

// platform reads one platform under the lock
func platform(host string) (Platform, bool) {
	platformsMutex.RLock()
	defer platformsMutex.RUnlock()
	found, isPresent := ConfiguredPlatforms[host]
	return found, isPresent
}

func HasInputConfig() bool {
	_, err := os.Stat(commons.InputConfigFile)
	if err != nil {
		fmt.Println(ColorString("[light_yellow]Cannot read input config file : [red]" + commons.InputConfigFile + "[light_yellow] Ignore it..."))
		fmt.Println("")
	}
	return commons.InputConfigFile != "" && err == nil
}

func UseConfig() {
	if HasInputConfig() {
		viper.SetConfigFile(commons.InputConfigFile)
		viper.SetConfigType("yaml")
		err := viper.ReadInConfig()
		BuildPlatforms()
		BuildMorningRoutine()
		if err != nil {
			fmt.Println(ColorString("[light_yellow]Cannot read config in file : [red]"+commons.InputConfigFile) + "[light_yellow] Ignore it...")
		}
		SetWorkDir()
	}
	if !strings.HasSuffix(commons.WorkDir, "/") {
		commons.WorkDir += "/"
	}
}

func SetWorkDir() {
	if commons.WorkDir == "" || commons.WorkDir == commons.DefaultWorkDir {
		workDir := viper.GetString("work_dir")
		if workDir != "" {
			if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
				fmt.Println(ColorString("[light_yellow]The work_dir is not a valid directory: [red]" + workDir))
				commons.WorkDir = commons.DefaultWorkDir
			} else {
				commons.WorkDir = workDir
			}
		}
	}
}

func BuildMorningRoutine() MorningRoutine {
	if MorningRoutineConfig.initialized {
		return MorningRoutineConfig
	}
	MorningRoutineConfig := MorningRoutine{
		Cmds:    []string{},
		Shell:   "bash",
		ohmyzsh: false,
	}
	if viper.IsSet("morning_routine.shell") {
		MorningRoutineConfig.Shell = viper.GetString("morning_routine.shell")
	}
	if viper.IsSet("morning_routine.ohmyzsh") {
		MorningRoutineConfig.ohmyzsh = viper.GetBool("morning_routine.ohmyzsh")
	}
	if viper.IsSet("morning_routine.commands") {
		MorningRoutineConfig.Cmds = strings.Split(strings.TrimSpace(viper.GetString("morning_routine.commands")), ",")
	}
	MorningRoutineConfig.initialized = true
	return MorningRoutineConfig
}

func GetMorningRoutine() MorningRoutine {
	return BuildMorningRoutine()
}

func BuildPlatforms() {
	platforms := viper.GetStringMap("platforms")

	built := make(map[string]Platform, len(platforms))
	var incomplete []string

	// Iterating over the map
	for key := range platforms {
		infos := viper.GetStringMap("platforms." + key)

		platform := Platform{
			typeOf:    asString(infos["type"]),
			token:     asString(infos["token"]),
			publicKey: asString(infos["public_key"]),
		}
		// public_key_passphrase is the documented key, public_key_password the one the first version
		// read. Both are accepted, so that a configuration file written from the documentation stops
		// having its passphrase silently ignored.
		if infos["public_key_passphrase"] != nil {
			platform.publicKeyPassword = asString(infos["public_key_passphrase"])
		} else {
			platform.publicKeyPassword = asString(infos["public_key_password"])
		}

		// A platform missing its type or its token is kept, so that the credentials command can list
		// it and offer to fix it, but it is still reported as incomplete
		if platform.typeOf == "" {
			incomplete = append(incomplete, "Platform [blue]"+key+"[light_yellow] is not correctly configured : missing [blue]type[light_yellow] field")
		} else if platform.token == "" {
			incomplete = append(incomplete, "Platform [blue]"+key+"[light_yellow] is not correctly configured : missing [blue]token[light_yellow] field")
		}
		built[key] = platform
	}

	// The map is replaced rather than added to : a reload, after a credential was updated, must not
	// keep a platform which is no longer in the configuration file
	platformsMutex.Lock()
	ConfiguredPlatforms = built
	platformsMutex.Unlock()

	// Only the first time for a given platform : the TUI rebuilds the platforms after every update,
	// and the credentials screen says this much better anyway. alreadyWarned is guarded by the same
	// mutex, so that nothing relies on BuildPlatforms having a single caller.
	var toWarn []string
	platformsMutex.Lock()
	for _, warning := range incomplete {
		if !alreadyWarned[warning] {
			alreadyWarned[warning] = true
			toWarn = append(toWarn, warning)
		}
	}
	platformsMutex.Unlock()

	// Reported outside the lock
	for _, warning := range toWarn {
		TraceWarn(ColorString("[light_yellow]" + warning))
	}
}

// asString reads a value of the configuration file as a string, without panicking on a value which
// is not one (a token written as a number, for instance)
func asString(value any) string {
	if value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprintf("%v", value)
}

// ReloadConfig reads the configuration file again and builds the platforms from it, so that a
// credential updated while Heimdall is running is taken into account without restarting it.
// It writes nothing on stdout, which the TUI forbids.
func ReloadConfig() error {
	if commons.InputConfigFile == "" {
		return nil
	}
	if _, err := os.Stat(commons.InputConfigFile); err != nil {
		return err
	}
	viper.SetConfigFile(commons.InputConfigFile)
	viper.SetConfigType("yaml")
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	BuildPlatforms()
	return nil
}

func GetPlatformType(host string) string {
	if found, isPresent := platform(host); isPresent {
		return found.typeOf
	} else {
		return ""
	}
}

// IsComplete tells whether a platform has both the type and the token Heimdall needs to use it. A
// platform missing one of them is still listed, so that it can be fixed, but it cannot be used.
func IsComplete(host string) bool {
	found, isPresent := platform(host)
	return isPresent && found.typeOf != "" && found.token != ""
}

// LiteralToken returns the token of a platform when it is written in the configuration file itself,
// which is what the migration into the keyring needs to read. It is deliberately the only way to get
// a secret out of the configuration without resolving it, and deliberately easy to grep for.
func LiteralToken(host string) (string, bool) {
	found, isPresent := platform(host)
	if !isPresent || SourceOf(found.token) != SourceLiteral {
		return "", false
	}
	return found.token, true
}

func GetToken(host string, spinner *tea.Program) string {
	found, isPresent := platform(host)
	if !isPresent {
		TraceWarn(ColorString("[yellow]" + host + "[light_yellow] is not configured, cannot retrieve a token. Operation may fail"))
		return ""
	}
	if found.token == "" {
		TraceWarn(ColorString("[yellow]" + host + "[light_yellow] has no token configured. Operation may fail"))
		return ""
	}
	value, _ := resolveSecret(found.token, spinner)
	return value
}

// resolveSecret turns a raw value of the configuration file into the secret it stands for : the entry
// of the keyring a "keyring." reference names, the environment variable an "env." reference names,
// or the value itself. The second result tells whether a secret was found.
//
// Nothing it reports ever holds the secret, only the name of where it was looked for.
func resolveSecret(rawValue string, spinner *tea.Program) (string, bool) {
	value, found, problem := resolveQuietly(rawValue)
	if !found && problem != "" {
		notify(spinner, problem)
	}
	return value, found
}

// resolveQuietly does the resolution and describes what went wrong instead of reporting it, so that a
// caller which reports the outcome itself does not say it twice. The description never holds the
// secret, only the name of where it was looked for.
func resolveQuietly(rawValue string) (value string, found bool, problem string) {
	switch {
	case rawValue == "":
		return "", false, ""

	case strings.HasPrefix(rawValue, commons.KeyringPrefix):
		key := strings.TrimPrefix(rawValue, commons.KeyringPrefix)
		value, err := secrets.Get(key)
		switch {
		case errors.Is(err, secrets.ErrNotFound):
			return "", false, ColorString("[light_blue]" + key + "[yellow] is not in the keyring")
		case errors.Is(err, secrets.ErrUnavailable):
			return "", false, ColorString("[yellow]No keyring available to read [light_blue]" + key + "[yellow] from")
		case err != nil:
			return "", false, ColorString("[yellow]Cannot read [light_blue]" + key + "[yellow] from the keyring")
		case value == "":
			return "", false, ColorString("[light_blue]" + key + "[yellow] in the keyring is empty")
		}
		return value, true, ""

	case strings.HasPrefix(rawValue, commons.EnvVariable):
		name := strings.TrimPrefix(rawValue, commons.EnvVariable)
		value := os.Getenv(name)
		if value == "" {
			return "", false, ColorString("[light_blue]" + name + "[yellow] referenced in config-file is not set")
		}
		return value, true, ""

	default:
		return rawValue, true, ""
	}
}

// TokenQuietly resolves the token of a platform without reporting anything. It is what the credential
// check uses, because it reports the outcome of every platform itself.
func TokenQuietly(host string) (string, bool) {
	found, isPresent := platform(host)
	if !isPresent {
		return "", false
	}
	value, resolved, _ := resolveQuietly(found.token)
	return value, resolved
}

// notify reports a problem to the running UI when there is one, and to the log otherwise
func notify(spinner *tea.Program, message string) {
	if spinner != nil {
		spinner.Send(tui.ErrorMessage{Error: message})
	} else {
		TraceWarn(message)
	}
}

func GetPublicKey(host string, spinner *tea.Program) string {
	if platform, isPresent := platform(host); isPresent {
		rawValue := platform.publicKey
		if rawValue == "" {
			log := ColorString("⚠️ [yellow]" + host + "[light_yellow] publickey is not configured. Using default one: [light_blue]" + commons.PublickeyPath)
			if spinner != nil {
				spinner.Send(tui.InfoMessage{Message: log})
			} else {
				TraceWarn(log)
			}
			return commons.PublickeyPath
		} else if _, err := os.Stat(rawValue); err != nil {
			log := ColorString("⚠️ [light_yellow]Public key [blue]" + rawValue + "[light_yellow] does not exist. Operation may fail")
			if spinner != nil {
				spinner.Send(tui.ErrorMessage{Error: log})
			} else {
				TraceWarn(log)
			}
			return ""
		}
		return rawValue
	} else {
		log := ColorString("⚠️ [yellow]" + host + "[light_yellow] publickey is not configured. Using default one: [light_blue]" + commons.PublickeyPath)
		if spinner != nil {
			spinner.Send(tui.InfoMessage{Message: log})
		} else {
			TraceWarn(log)
		}
		return commons.PublickeyPath
	}
}

func GetPublicKeyPassword(host string, spinner *tea.Program) string {
	found, isPresent := platform(host)
	if !isPresent {
		return commons.SshkeyPassword
	}
	value, ok := resolveSecret(found.publicKeyPassword, spinner)
	if !ok {
		return commons.SshkeyPassword
	}
	return value
}

// Source tells where the value of a credential is read from
type Source int

const (
	// SourceNone : the credential is not configured at all
	SourceNone Source = iota
	// SourceLiteral : the value is written in the configuration file
	SourceLiteral
	// SourceEnv : the configuration file references an environment variable
	SourceEnv
	// SourceKeyring : the configuration file references an entry of the keyring
	SourceKeyring
)

// SourceOf tells where a raw value of the configuration file reads its secret from
func SourceOf(rawValue string) Source {
	switch {
	case rawValue == "":
		return SourceNone
	case strings.HasPrefix(rawValue, commons.KeyringPrefix):
		return SourceKeyring
	case strings.HasPrefix(rawValue, commons.EnvVariable):
		return SourceEnv
	default:
		return SourceLiteral
	}
}

// PlatformInfo describes how a platform is configured. It never holds a secret : when the value is
// written in the configuration file, the reference is left empty and only the source says there is
// one, so that a PlatformInfo can be displayed and logged freely.
type PlatformInfo struct {
	Host string
	Type string
	// TokenRef is the name of the keyring entry or of the environment variable the token is read
	// from. It is empty when the token is written in the configuration file, because that value is
	// the token itself : use TokenLocation to display where the token comes from.
	TokenRef         string
	TokenSource      Source
	PublicKey        string
	PassphraseRef    string
	PassphraseSource Source
}

// TokenLocation names where the token is read from, in a form which is always safe to display
func (p PlatformInfo) TokenLocation() string { return location(p.TokenRef, p.TokenSource) }

// PassphraseLocation names where the passphrase of the ssh key is read from, safe to display
func (p PlatformInfo) PassphraseLocation() string {
	return location(p.PassphraseRef, p.PassphraseSource)
}

// Complete tells whether the platform has the type and the token Heimdall needs to use it
func (p PlatformInfo) Complete() bool { return p.Type != "" && p.TokenSource != SourceNone }

func location(rawValue string, source Source) string {
	switch source {
	case SourceKeyring:
		return "keyring: " + strings.TrimPrefix(rawValue, commons.KeyringPrefix)
	case SourceEnv:
		return "env: " + strings.TrimPrefix(rawValue, commons.EnvVariable)
	case SourceLiteral:
		// Deliberately not the value : this is displayed, and printed into the log
		return "config file"
	default:
		return "not set"
	}
}

// Platforms lists the configured platforms, sorted by host, without exposing any secret
func Platforms() []PlatformInfo {
	platformsMutex.RLock()
	defer platformsMutex.RUnlock()

	infos := make([]PlatformInfo, 0, len(ConfiguredPlatforms))
	for host, platform := range ConfiguredPlatforms {
		infos = append(infos, PlatformInfo{
			Host:             host,
			Type:             platform.typeOf,
			TokenRef:         reference(platform.token),
			TokenSource:      SourceOf(platform.token),
			PublicKey:        platform.publicKey,
			PassphraseRef:    reference(platform.publicKeyPassword),
			PassphraseSource: SourceOf(platform.publicKeyPassword),
		})
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].Host < infos[j].Host })
	return infos
}

// reference keeps the value only when it points somewhere else : a value written in the
// configuration file is the secret itself and must not travel in a PlatformInfo
func reference(rawValue string) string {
	if SourceOf(rawValue) == SourceLiteral {
		return ""
	}
	return rawValue
}
