package cmd

import (
	"context"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/creds"
	"github.com/yodamad/heimdall/secrets"
	"github.com/yodamad/heimdall/utils"
	"github.com/yodamad/heimdall/utils/tui"
)

var (
	allPlatforms   bool
	keyringEntry   string
	valueFromStdin bool
	reference      string
	sshPassphrase  bool
	platformType   string
	expiringWithin time.Duration
	quietCheck     bool
	strictCheck    bool
)

// Exit codes of `credentials check`, so that it can be used in a script
const (
	exitEverythingFine = 0
	exitNeedsAttention = 1
	exitCouldNotBeTold = 2
)

var Credentials = &cobra.Command{
	Use:     "credentials",
	Aliases: []string{"creds", "cred"},
	Short:   "Check and update the credentials of your platforms",
	Long: utils.ColorString(`
Check the tokens of the platforms of your [light_blue]configuration file[default] against their API, and
update the ones which are missing, expired or about to expire.

A new credential is stored in the [light_blue]keyring of your operating system[default], and the
configuration file only keeps a reference to it. Several platforms can share one
entry, so that a token used by several hosts is rotated in one place.
`),
	Run: func(cmd *cobra.Command, args []string) {
		start()
		// Without a subcommand, say what is wrong and offer to fix it : the useful default
		results := runCheck(args)
		if !anyNeedsAttention(results) {
			return
		}
		utils.PrintSeparation()
		if answer := tui.AskQuestion(utils.ColorString("🔧 Do you want to update them now ? [light_gray][Y/n][default] : "), "Y"); answer != "y" && answer != "Y" {
			utils.Trace(utils.ColorString("❌ [red]Nothing changed"), false)
			return
		}
		runUpdate(hostsNeedingAttention(results))
	},
}

var credentialsList = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List the platforms and where their credentials are read from",
	Run: func(cmd *cobra.Command, args []string) {
		start()
		runList()
	},
}

var credentialsCheck = &cobra.Command{
	Use:   "check [host...]",
	Short: "Check the token of every platform, or of the given ones, against its API",
	Run: func(cmd *cobra.Command, args []string) {
		start()
		results := runCheck(args)
		os.Exit(exitCodeFor(results))
	},
}

var credentialsUpdate = &cobra.Command{
	Use:   "update [host...]",
	Short: "Store a new credential for one or several platforms",
	Run: func(cmd *cobra.Command, args []string) {
		start()
		runUpdate(args)
	},
}

var credentialsMigrate = &cobra.Command{
	Use:   "migrate",
	Short: "Move the tokens written in the configuration file into the keyring",
	Run: func(cmd *cobra.Command, args []string) {
		start()
		runMigrate()
	},
}

func init() {
	credentialsCheck.Flags().DurationVar(&expiringWithin, "expiring-within", creds.DefaultExpiryWindow, "report a token expiring within this duration")
	credentialsCheck.Flags().BoolVarP(&quietCheck, "quiet", "q", false, "print nothing, only set the exit code")
	credentialsCheck.Flags().BoolVar(&strictCheck, "strict", false, "fail when a token is about to expire too")

	credentialsUpdate.Flags().BoolVarP(&allPlatforms, "all", "a", false, "update every configured platform")
	credentialsUpdate.Flags().StringVarP(&keyringEntry, "entry", "e", "", "name of the keyring entry to store the secret under")
	credentialsUpdate.Flags().BoolVar(&valueFromStdin, "from-stdin", false, "read the secret from the standard input instead of asking for it")
	credentialsUpdate.Flags().StringVar(&reference, "reference", "", "point the platforms at an existing secret (env.MY_VAR or keyring.my-entry) instead of storing one")
	credentialsUpdate.Flags().BoolVar(&sshPassphrase, "ssh-passphrase", false, "update the passphrase of the ssh key instead of the token")
	credentialsUpdate.Flags().StringVar(&platformType, "type", "", "type of a platform which is not configured yet (github or gitlab)")

	Credentials.AddCommand(credentialsList, credentialsCheck, credentialsUpdate, credentialsMigrate)
}

// start does what every command of Heimdall does before its work. The banner comes first here, so
// that a warning about an incomplete platform is not printed above it.
func start() {
	utils.PrintBanner()
	utils.UseConfig()
	utils.OverrideLogFile()
	if commons.Verbose {
		log.SetLevel(log.DebugLevel)
	}
}

func runList() {
	platforms := utils.Platforms()
	if len(platforms) == 0 {
		noPlatformConfigured()
		return
	}

	writer := table.NewWriter()
	writer.SetOutputMirror(os.Stdout)
	writer.AppendHeader(table.Row{"Platform", "Type", "Token", "SSH key", "Passphrase"})
	for _, platform := range platforms {
		writer.AppendRow(table.Row{
			platform.Host,
			orDash(platform.Type),
			platform.TokenLocation(),
			orDash(platform.PublicKey),
			platform.PassphraseLocation(),
		})
	}
	writer.SetColumnConfigs([]table.ColumnConfig{{Name: "Type", Align: text.AlignCenter}})
	writer.Render()

	if !secrets.Available() {
		utils.TraceWarn("No keyring available on this system : a new credential can only be stored as an env. reference")
	}
}

func runCheck(hosts []string) []creds.Result {
	if len(utils.Platforms()) == 0 {
		noPlatformConfigured()
		return nil
	}
	if !quietCheck {
		utils.Trace(utils.ColorString("🔎 Checking the credentials of your platforms..."), false)
	}

	results := creds.CheckAll(context.Background(), hosts, expiringWithin)
	if quietCheck {
		return results
	}

	writer := table.NewWriter()
	writer.SetOutputMirror(os.Stdout)
	writer.AppendHeader(table.Row{"Platform", "Type", "Token", "State", "Details"})
	for _, result := range results {
		writer.AppendRow(table.Row{
			result.Host,
			orDash(result.Type),
			result.Location,
			stateOf(result),
			detailOf(result),
		})
	}
	writer.SetColumnConfigs([]table.ColumnConfig{{Name: "Type", Align: text.AlignCenter}})
	writer.Render()
	return results
}

// stateOf renders the verdict, with a symbol unless colors are turned off
func stateOf(result creds.Result) string {
	if commons.NoColor {
		return result.Status.Label()
	}
	switch result.Status {
	case creds.StatusValid:
		return utils.ColorString("[green]✔ valid")
	case creds.StatusExpiring:
		return utils.ColorString("[yellow]⏳ expiring")
	case creds.StatusRejected:
		return utils.ColorString("[red]✘ rejected")
	case creds.StatusNoToken:
		return utils.ColorString("[red]✘ no token")
	case creds.StatusUnresolved:
		return utils.ColorString("[red]✘ unresolved")
	case creds.StatusUnsupported:
		return utils.ColorString("[dark_gray]- unsupported")
	default:
		return utils.ColorString("[dark_gray]? unreachable")
	}
}

func detailOf(result creds.Result) string {
	if result.Detail != "" {
		return result.Detail
	}
	if result.User != "" {
		return "authenticates as " + result.User
	}
	return ""
}

func exitCodeFor(results []creds.Result) int {
	code := exitEverythingFine
	for _, result := range results {
		switch {
		case result.Status.NeedsAttention():
			return exitNeedsAttention
		case strictCheck && result.Status == creds.StatusExpiring:
			return exitNeedsAttention
		case result.Status == creds.StatusUnknown:
			code = exitCouldNotBeTold
		}
	}
	return code
}

func anyNeedsAttention(results []creds.Result) bool {
	return len(hostsNeedingAttention(results)) > 0
}

func hostsNeedingAttention(results []creds.Result) []string {
	var hosts []string
	for _, result := range results {
		if result.Status.NeedsAttention() || result.Status == creds.StatusExpiring {
			hosts = append(hosts, result.Host)
		}
	}
	return hosts
}

func runUpdate(hosts []string) {
	// Refuse before asking for a secret which could not be saved afterwards
	if err := creds.EnsureWritable(commons.InputConfigFile); err != nil {
		utils.TraceWarn(utils.ColorString("❌ Cannot write the configuration file : [red]" + err.Error()))
		return
	}

	targets, ok := chooseTargets(hosts)
	if !ok || len(targets) == 0 {
		utils.Trace(utils.ColorString("❌ [red]Nothing changed"), false)
		return
	}

	field := creds.FieldToken
	what := "token"
	if sshPassphrase {
		field, what = creds.FieldPassphrase, "ssh key passphrase"
	}

	value, ok := referenceFor(targets, what)
	if !ok {
		return
	}

	// A platform which is not configured yet needs its type, or Heimdall cannot talk to it
	types, ok := typesFor(targets, field)
	if !ok {
		return
	}

	for host, kind := range types {
		if err := creds.SetPlatformField(commons.InputConfigFile, host, creds.FieldType, kind); err != nil {
			utils.TraceWarn(utils.ColorString("❌ Cannot write the type of " + host + " : [red]" + err.Error()))
			return
		}
	}
	if err := creds.SetPlatformFields(commons.InputConfigFile, targets, field, value); err != nil {
		utils.TraceWarn(utils.ColorString("❌ Cannot write the configuration file : [red]" + err.Error()))
		return
	}

	utils.Trace(utils.ColorString("✅ "+what+" of [light_blue]"+strings.Join(targets, "[default], [light_blue]")+
		"[default] now read from [light_blue]"+value), false)

	if err := utils.ReloadConfig(); err != nil {
		utils.TraceWarn("Cannot read the configuration file again : " + err.Error())
		return
	}
	if sshPassphrase {
		// There is nothing to check a passphrase against
		return
	}
	utils.PrintSeparation()
	expiringWithin = creds.DefaultExpiryWindow
	runCheck(targets)
}

// chooseTargets works out which platforms to update : the ones named on the command line, all of
// them, or the ones picked from a list where whatever is broken is already selected
func chooseTargets(hosts []string) ([]string, bool) {
	platforms := utils.Platforms()

	if len(hosts) > 0 {
		// A host which is not configured yet is accepted : this is how a platform is added
		return hosts, true
	}
	if len(platforms) == 0 {
		noPlatformConfigured()
		return nil, false
	}
	if allPlatforms {
		var all []string
		for _, platform := range platforms {
			all = append(all, platform.Host)
		}
		return all, true
	}

	// Checking first, so that the list says what is wrong with each platform
	results := creds.CheckAll(context.Background(), nil, expiringWithin)
	items := make([]tui.MultiItem, 0, len(results))
	preselected := map[int]bool{}
	for index, result := range results {
		items = append(items, tui.MultiItem{Label: result.Host, Hint: result.Status.Label() + hintSuffix(result)})
		if result.Status.NeedsAttention() || result.Status == creds.StatusExpiring {
			preselected[index] = true
		}
	}
	return tui.PickSeveral("Platforms to update", items, preselected)
}

func hintSuffix(result creds.Result) string {
	if result.Status == creds.StatusValid && result.User != "" {
		return " as " + result.User
	}
	if result.Detail != "" {
		return " - " + result.Detail
	}
	return ""
}

// referenceFor returns the value to write in the configuration file for these platforms : either a
// reference given on the command line, or a new entry of the keyring holding the secret it asks for
func referenceFor(targets []string, what string) (string, bool) {
	if reference != "" {
		switch {
		case strings.HasPrefix(reference, commons.EnvVariable):
			if strings.TrimPrefix(reference, commons.EnvVariable) == "" {
				utils.TraceWarn("--reference env. needs the name of the variable, as env.MY_TOKEN")
				return "", false
			}
		case strings.HasPrefix(reference, commons.KeyringPrefix):
			if !secrets.ValidKey(strings.TrimPrefix(reference, commons.KeyringPrefix)) {
				utils.TraceWarn("--reference keyring. needs a name of letters, digits, dots, dashes or underscores")
				return "", false
			}
		default:
			utils.TraceWarn("--reference has to start with " + commons.EnvVariable + " or " + commons.KeyringPrefix)
			return "", false
		}
		return reference, true
	}

	if !secrets.Available() {
		utils.TraceWarn("No keyring available on this system, so there is nowhere safe to store a secret")
		utils.Trace(utils.ColorString("💡 Set it in an environment variable and point Heimdall at it with "+
			"[light_blue]--reference env.MY_TOKEN"), false)
		return "", false
	}

	entry := keyringEntry
	if entry == "" {
		entry = defaultEntryName(targets)
		if !valueFromStdin {
			entry = tui.AskQuestion(utils.ColorString("🔑 Name of the keyring entry to store the "+what+
				" under [dark_gray]("+entry+")[default] : "), entry)
		}
	}
	if !secrets.ValidKey(entry) {
		utils.TraceWarn("The name of a keyring entry may only hold letters, digits, dots, dashes and underscores")
		return "", false
	}

	value, ok := readSecret(what, entry)
	if !ok {
		return "", false
	}
	if err := secrets.Set(entry, value); err != nil {
		utils.TraceWarn(utils.ColorString("❌ Cannot store the secret in the keyring : [red]" + err.Error()))
		return "", false
	}
	utils.Trace(utils.ColorString("🔐 Stored in the keyring as [light_blue]"+entry), false)
	return commons.KeyringPrefix + entry, true
}

// defaultEntryName is the host when there is only one, and a shared name when the same secret is
// about to be used by several of them
func defaultEntryName(targets []string) string {
	if len(targets) == 1 && secrets.ValidKey(targets[0]) {
		return targets[0]
	}
	return "shared"
}

// readSecret gets the secret from the standard input, for a script, or asks for it twice without
// displaying it
func readSecret(what, entry string) (string, bool) {
	if valueFromStdin {
		raw, err := io.ReadAll(os.Stdin)
		if err != nil {
			utils.TraceWarn("Cannot read the secret from the standard input : " + err.Error())
			return "", false
		}
		value := strings.TrimSpace(string(raw))
		if value == "" {
			utils.TraceWarn("Nothing was given on the standard input")
			return "", false
		}
		return value, true
	}

	// The prompt hides what is typed only when it owns a terminal. Without one the shell keeps
	// echoing, and the secret would be printed on screen and into any transcript of the session.
	if !IsTerminal() {
		utils.TraceWarn("Not running in a terminal, so the " + what + " cannot be typed without being displayed")
		utils.Trace(utils.ColorString("💡 Give it on the standard input instead : "+
			"[light_blue]echo \"$NEW_TOKEN\" | heimdall credentials update <host> --from-stdin"), false)
		return "", false
	}

	value, ok := tui.AskSecret(utils.ColorString("🔑 " + strings.ToUpper(what[:1]) + what[1:] + " for [light_blue]" + entry + "[default] (not displayed) : "))
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		utils.TraceWarn("An empty secret is not stored")
		return "", false
	}
	again, ok := tui.AskSecret(utils.ColorString("🔑 Once more, to be sure : "))
	if !ok {
		return "", false
	}
	if strings.TrimSpace(again) != value {
		utils.TraceWarn("The two answers are different, nothing was stored")
		return "", false
	}
	return value, true
}

// typesFor asks for the type of the platforms which have none, as Heimdall cannot use them otherwise
func typesFor(targets []string, field string) (map[string]string, bool) {
	if field != creds.FieldToken {
		return nil, true
	}
	known := map[string]string{}
	for _, platform := range utils.Platforms() {
		known[platform.Host] = platform.Type
	}

	types := map[string]string{}
	for _, host := range targets {
		if known[host] != "" {
			continue
		}
		if platformType != "" {
			if platformType != "github" && platformType != "gitlab" {
				utils.TraceWarn("--type has to be github or gitlab")
				return nil, false
			}
			types[host] = platformType
			continue
		}
		answer := tui.AskQuestion(utils.ColorString("🏷️  Type of [light_blue]"+host+
			"[default] [dark_gray](github or gitlab)[default] : "), "")
		if answer != "github" && answer != "gitlab" {
			utils.TraceWarn("The type of " + host + " has to be github or gitlab")
			return nil, false
		}
		types[host] = answer
	}
	return types, true
}

func runMigrate() {
	var literals []string
	for _, platform := range utils.Platforms() {
		if platform.TokenSource == utils.SourceLiteral {
			literals = append(literals, platform.Host)
		}
	}
	sort.Strings(literals)

	if len(literals) == 0 {
		utils.Trace(utils.ColorString("✅ No token is written in [light_blue]"+commons.InputConfigFile+
			"[default], nothing to migrate"), false)
		return
	}
	if !secrets.Available() {
		utils.TraceWarn("No keyring available on this system, the tokens cannot be moved out of the configuration file")
		return
	}
	if err := creds.EnsureWritable(commons.InputConfigFile); err != nil {
		utils.TraceWarn(utils.ColorString("❌ Cannot write the configuration file : [red]" + err.Error()))
		return
	}

	utils.TraceWarn(strconv.Itoa(len(literals)) + " token(s) are written in " + commons.InputConfigFile + " : " + strings.Join(literals, ", "))
	// Being honest about what this does and does not achieve
	utils.Trace(utils.ColorString("⚠️  Moving them into the keyring removes them from the file, but [bold]not[reset] from "+
		"your backups,\n   nor from the history of the repository if this file is versioned. For a token which was "+
		"exposed,\n   revoking it and issuing a new one is the only real fix."), false)
	if answer := tui.AskQuestion(utils.ColorString("🔐 Move them into the keyring ? [light_gray][Y/n][default] : "), "Y"); answer != "y" && answer != "Y" {
		utils.Trace(utils.ColorString("❌ [red]Nothing changed"), false)
		return
	}

	for _, host := range literals {
		token, found := utils.LiteralToken(host)
		if !found {
			continue
		}
		entry := host
		if !secrets.ValidKey(entry) {
			entry = tui.AskQuestion(utils.ColorString("🔑 Name of the keyring entry for [light_blue]"+host+"[default] : "), "")
			if !secrets.ValidKey(entry) {
				utils.TraceWarn("Skipping " + host + " : the name of the entry is not usable")
				continue
			}
		}
		if err := secrets.Set(entry, token); err != nil {
			utils.TraceWarn(utils.ColorString("❌ " + host + " : [red]" + err.Error()))
			continue
		}
		if err := creds.SetPlatformField(commons.InputConfigFile, host, creds.FieldToken, commons.KeyringPrefix+entry); err != nil {
			utils.TraceWarn(utils.ColorString("❌ " + host + " : [red]" + err.Error()))
			continue
		}
		utils.Trace(utils.ColorString("✅ [light_blue]"+host+"[default] now reads its token from the keyring entry [light_blue]"+entry), false)
	}

	if err := utils.ReloadConfig(); err != nil {
		utils.TraceWarn("Cannot read the configuration file again : " + err.Error())
	}
}

func noPlatformConfigured() {
	utils.TraceWarn("No platform is configured in " + commons.InputConfigFile)
	utils.Trace(utils.ColorString("💡 Add one with [light_blue]heimdall credentials update <host>[default], "+
		"for instance [light_blue]heimdall credentials update github.com"), false)
}

func orDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
