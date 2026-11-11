package utils

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/secrets"
	"github.com/zalando/go-keyring"
)

func TestHasInputConfigOK(t *testing.T) {
	// Test case where the input config file exists
	commons.InputConfigFile = "test_config.yaml"
	_, err := os.Create(commons.InputConfigFile)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(commons.InputConfigFile)

	if !HasInputConfig() {
		t.Errorf("Expected HasInputConfig to return true, got false")
	}
}

func TestConfigFileNotSet(t *testing.T) {
	// Test case where the input config file is an empty string
	commons.InputConfigFile = ""
	if HasInputConfig() {
		t.Errorf("Expected HasInputConfig to return false, got true")
	}
}

func TestIncorrectConfig(t *testing.T) {
	// Test case where the input config file does not exist
	commons.InputConfigFile = "non_existent_config.yaml"
	if HasInputConfig() {
		t.Errorf("Expected HasInputConfig to return false, got true")
	}

	// Test case where the input config file is an empty string
	commons.InputConfigFile = ""
	if HasInputConfig() {
		t.Errorf("Expected HasInputConfig to return false, got true")
	}
}

func TestUseConfigOK(t *testing.T) {
	// Setup
	originalInputConfigFile := commons.InputConfigFile
	originalWorkDir := commons.WorkDir
	defer func() {
		commons.InputConfigFile = originalInputConfigFile
		commons.WorkDir = originalWorkDir
	}()

	// Test case where the input config file exists and is valid
	commons.InputConfigFile = "test_config.yaml"
	commons.WorkDir = commons.DefaultWorkDir
	configContent := `
work_dir: /tmp
platforms:
  github.com:
    type: github
    token: env.GITHUB_TOKEN
`
	err := os.WriteFile(commons.InputConfigFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(commons.InputConfigFile)

	UseConfig()

	if commons.WorkDir != "/tmp/" {
		t.Errorf("Expected WorkDir to be '/tmp', got '%s'", commons.WorkDir)
	}

	if len(ConfiguredPlatforms) != 1 {
		t.Errorf("Expected 1 configured platform, got %d", len(ConfiguredPlatforms))
	}

	if ConfiguredPlatforms["github.com"].typeOf != "github" {
		t.Errorf("Expected platform type to be 'github', got '%s'", ConfiguredPlatforms["github"].typeOf)
	}

	if ConfiguredPlatforms["github.com"].token != "env.GITHUB_TOKEN" {
		t.Errorf("Expected platform token to be 'env.GITHUB_TOKEN', got '%s'", ConfiguredPlatforms["github"].token)
	}
}

func TestDefaultConfigForIncorrectFile(t *testing.T) {
	// Test case where the input config file does not exist
	commons.InputConfigFile = "non_existent_config.yaml"
	commons.WorkDir = commons.DefaultWorkDir

	UseConfig()

	if commons.WorkDir != commons.DefaultWorkDir+"/" {
		t.Errorf("Expected WorkDir to be '%s', got '%s'", commons.DefaultWorkDir, commons.WorkDir)
	}
}

func TestIncorrectConfigFileContent(t *testing.T) {
	// Test case where the work_dir is not a valid directory
	originalInputConfigFile := commons.InputConfigFile
	originalWorkDir := commons.WorkDir
	defer func() {
		commons.InputConfigFile = originalInputConfigFile
		commons.WorkDir = originalWorkDir
	}()
	commons.WorkDir = commons.DefaultWorkDir
	commons.InputConfigFile = "test_invalid_work_dir_config.yaml"
	configContent := `
work_dir: /invalid_dir
`
	err := os.WriteFile(commons.InputConfigFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(commons.InputConfigFile)

	UseConfig()

	if commons.WorkDir != commons.DefaultWorkDir+"/" {
		t.Errorf("Expected WorkDir to be '%s', got '%s'", commons.DefaultWorkDir, commons.WorkDir)
	}
}

func TestBuildPlatformsOK(t *testing.T) {

	commons.InputConfigFile = "test_config.yaml"
	commons.WorkDir = commons.DefaultWorkDir
	configContent := `
work_dir: /tmp
platforms:
  github.com:
    type: github
    token: env.GITHUB_TOKEN
  gitlab.com:
    type: gitlab
    token: env.GITLAB_TOKEN
`
	err := os.WriteFile(commons.InputConfigFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(commons.InputConfigFile)

	UseConfig()

	BuildPlatforms()
	if len(ConfiguredPlatforms) != 2 {
		t.Errorf("Expected 2 configured platforms, got %d", len(ConfiguredPlatforms))
	}

	if platform, ok := ConfiguredPlatforms["github.com"]; !ok || platform.typeOf != "github" || platform.token != "env.GITHUB_TOKEN" {
		t.Errorf("Expected platform 'github.com' to have type 'github' and token 'env.GITHUB_TOKEN', got type '%s' and token '%s'", platform.typeOf, platform.token)
	}

	if platform, ok := ConfiguredPlatforms["gitlab.com"]; !ok || platform.typeOf != "gitlab" || platform.token != "env.GITLAB_TOKEN" {
		t.Errorf("Expected platform 'gitlab.com' to have type 'gitlab' and token 'env.GITLAB_TOKEN', got type '%s' and token '%s'", platform.typeOf, platform.token)
	}
}

func TestBuildPlatformsIncorrectStructure(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		// viper is global : do not leak the platforms set here into the other tests
		viper.Reset()
	}()
	viper.Reset()

	viper.Set("platforms", map[string]interface{}{
		"github": map[string]interface{}{
			"type": "github",
		},
		"gitlab": map[string]interface{}{
			"token": "env.GITLAB_TOKEN",
		},
	})
	BuildPlatforms()

	// Both are kept, so that they can be listed and fixed, but none is complete
	if len(ConfiguredPlatforms) != 2 {
		t.Errorf("Expected 2 configured platforms, got %d", len(ConfiguredPlatforms))
	}
	for _, host := range []string{"github", "gitlab"} {
		if _, ok := ConfiguredPlatforms[host]; !ok {
			t.Errorf("Expected platform '%s' to be kept", host)
		}
		if IsComplete(host) {
			t.Errorf("Expected platform '%s' to be incomplete", host)
		}
	}
}

func TestGetPlatformTypeOK(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()

	// Test case where the platform is present
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "env.GITHUB_TOKEN"},
	}
	if platformType := GetPlatformType("github.com"); platformType != "github" {
		t.Errorf("Expected platform type to be 'github', got '%s'", platformType)
	}

	// Test case where the platform is not present
	ConfiguredPlatforms = map[string]Platform{}
	if platformType := GetPlatformType("gitlab.com"); platformType != "" {
		t.Errorf("Expected platform type to be '', got '%s'", platformType)
	}
}

func TestGetPlatformTypeKO(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()

	// Test case where the platform is not present
	ConfiguredPlatforms = map[string]Platform{}
	if platformType := GetPlatformType("gitlab.com"); platformType != "" {
		t.Errorf("Expected platform type to be '', got '%s'", platformType)
	}
}

func TestGetTokenInClear(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()

	// Test case where the platform is present and the token is a direct value
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "direct_token"},
	}
	if token := GetToken("github.com", nil); token != "direct_token" {
		t.Errorf("Expected token to be 'direct_token', got '%s'", token)
	}
}

func TestGetTokenAsEnvVar(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()

	// Test case where the platform is present and the token is an environment variable
	os.Setenv("GITHUB_TOKEN", "env_token")
	defer os.Unsetenv("GITHUB_TOKEN")
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "env.GITHUB_TOKEN"},
	}
	if token := GetToken("github.com", nil); token != "env_token" {
		t.Errorf("Expected token to be 'env_token', got '%s'", token)
	}
}

func TestGetMissingToken(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()

	// Test case where the platform is not present
	ConfiguredPlatforms = map[string]Platform{}
	if token := GetToken("gitlab.com", nil); token != "" {
		t.Errorf("Expected token to be '', got '%s'", token)
	}

	// Test case where the environment variable referenced in the token is not set
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "env.NON_EXISTENT_ENV"},
	}
	if token := GetToken("github.com", nil); token != "" {
		t.Errorf("Expected token to be '', got '%s'", token)
	}
}

func TestGetTokenFromKeyring(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		secrets.UseMockForTests()
	}()
	secrets.UseMockForTests()
	if err := secrets.Set("shared-token", "keyring_token"); err != nil {
		t.Fatalf("Failed to seed the keyring: %v", err)
	}

	// Test case where the entry is in the keyring, and shared by two hosts
	ConfiguredPlatforms = map[string]Platform{
		"github.com":         {typeOf: "github", token: "keyring.shared-token"},
		"github.example.org": {typeOf: "github", token: "keyring.shared-token"},
	}
	for _, host := range []string{"github.com", "github.example.org"} {
		if token := GetToken(host, nil); token != "keyring_token" {
			t.Errorf("Expected token of '%s' to be 'keyring_token', got '%s'", host, token)
		}
	}

	// Test case where the entry is not in the keyring
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "keyring.missing-entry"},
	}
	if token := GetToken("github.com", nil); token != "" {
		t.Errorf("Expected token to be '', got '%s'", token)
	}
}

func TestGetTokenWithoutKeyring(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		secrets.UseMockForTests()
	}()
	secrets.FailForTests(keyring.ErrUnsupportedPlatform)

	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "keyring.shared-token"},
	}
	if token := GetToken("github.com", nil); token != "" {
		t.Errorf("Expected token to be '', got '%s'", token)
	}
}

func TestGetPublicKeyPasswordFromKeyring(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		secrets.UseMockForTests()
	}()
	secrets.UseMockForTests()
	if err := secrets.Set("ssh-passphrase", "keyring_passphrase"); err != nil {
		t.Fatalf("Failed to seed the keyring: %v", err)
	}

	// Test case where the entry is in the keyring
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "t", publicKeyPassword: "keyring.ssh-passphrase"},
	}
	if password := GetPublicKeyPassword("github.com", nil); password != "keyring_passphrase" {
		t.Errorf("Expected passphrase to be 'keyring_passphrase', got '%s'", password)
	}

	// Test case where the entry is missing : the default one is used
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "t", publicKeyPassword: "keyring.missing-entry"},
	}
	if password := GetPublicKeyPassword("github.com", nil); password != commons.SshkeyPassword {
		t.Errorf("Expected passphrase to be '%s', got '%s'", commons.SshkeyPassword, password)
	}

	// Test case where the platform is not configured
	if password := GetPublicKeyPassword("unknown.com", nil); password != commons.SshkeyPassword {
		t.Errorf("Expected passphrase to be '%s', got '%s'", commons.SshkeyPassword, password)
	}
}

func TestGetPublicKeyPasswordWithoutKeyring(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		secrets.UseMockForTests()
	}()
	secrets.FailForTests(keyring.ErrUnsupportedPlatform)

	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "t", publicKeyPassword: "keyring.ssh-passphrase"},
	}
	if password := GetPublicKeyPassword("github.com", nil); password != commons.SshkeyPassword {
		t.Errorf("Expected passphrase to be '%s', got '%s'", commons.SshkeyPassword, password)
	}
}

func TestPrefixInTheMiddleOfAValueIsLiteral(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		secrets.UseMockForTests()
	}()
	secrets.UseMockForTests()

	ConfiguredPlatforms = map[string]Platform{
		"gitlab.com": {typeOf: "gitlab", token: "glpat-env.something"},
		"github.com": {typeOf: "github", token: "ghp_keyring.something"},
	}
	if token := GetToken("gitlab.com", nil); token != "glpat-env.something" {
		t.Errorf("Expected token to be 'glpat-env.something', got '%s'", token)
	}
	if token := GetToken("github.com", nil); token != "ghp_keyring.something" {
		t.Errorf("Expected token to be 'ghp_keyring.something', got '%s'", token)
	}
	if source := SourceOf("glpat-env.something"); source != SourceLiteral {
		t.Errorf("Expected source to be SourceLiteral, got %v", source)
	}
	if source := SourceOf("ghp_keyring.something"); source != SourceLiteral {
		t.Errorf("Expected source to be SourceLiteral, got %v", source)
	}
}

func TestSourceOf(t *testing.T) {
	if source := SourceOf(""); source != SourceNone {
		t.Errorf("Expected SourceNone, got %v", source)
	}
	if source := SourceOf("glpat-abc"); source != SourceLiteral {
		t.Errorf("Expected SourceLiteral, got %v", source)
	}
	if source := SourceOf("env.MY_TOKEN"); source != SourceEnv {
		t.Errorf("Expected SourceEnv, got %v", source)
	}
	if source := SourceOf("keyring.my-token"); source != SourceKeyring {
		t.Errorf("Expected SourceKeyring, got %v", source)
	}
}

func TestTokenLocation(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()
	ConfiguredPlatforms = map[string]Platform{
		"a.com": {typeOf: "github", token: "keyring.my-token"},
		"b.com": {typeOf: "github", token: "env.MY_TOKEN"},
		"c.com": {typeOf: "github", token: "glpat-abc"},
		"d.com": {typeOf: "github"},
	}

	expected := map[string]string{
		"a.com": "keyring: my-token",
		"b.com": "env: MY_TOKEN",
		"c.com": "config file",
		"d.com": "not set",
	}
	for _, info := range Platforms() {
		if location := info.TokenLocation(); location != expected[info.Host] {
			t.Errorf("Expected location of '%s' to be '%s', got '%s'", info.Host, expected[info.Host], location)
		}
	}
}

func TestPassphraseLocation(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()
	ConfiguredPlatforms = map[string]Platform{
		"a.com": {typeOf: "github", token: "t", publicKeyPassword: "keyring.my-phrase"},
		"b.com": {typeOf: "github", token: "t"},
	}

	infos := Platforms()
	if location := infos[0].PassphraseLocation(); location != "keyring: my-phrase" {
		t.Errorf("Expected location to be 'keyring: my-phrase', got '%s'", location)
	}
	if location := infos[1].PassphraseLocation(); location != "not set" {
		t.Errorf("Expected location to be 'not set', got '%s'", location)
	}
}

func TestPlatformsNeverExposeALiteralSecret(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()
	const secretToken = "glpat-super-secret-token"
	const secretPassphrase = "super-secret-passphrase"
	ConfiguredPlatforms = map[string]Platform{
		"gitlab.com": {typeOf: "gitlab", token: secretToken, publicKey: "/path/key.pub", publicKeyPassword: secretPassphrase},
	}

	infos := Platforms()
	if len(infos) != 1 {
		t.Fatalf("Expected 1 platform, got %d", len(infos))
	}
	info := infos[0]
	if info.TokenRef != "" {
		t.Errorf("Expected TokenRef to be empty, got '%s'", info.TokenRef)
	}
	if info.TokenSource != SourceLiteral {
		t.Errorf("Expected TokenSource to be SourceLiteral, got %v", info.TokenSource)
	}
	if info.PassphraseRef != "" {
		t.Errorf("Expected PassphraseRef to be empty, got '%s'", info.PassphraseRef)
	}
	if info.PassphraseSource != SourceLiteral {
		t.Errorf("Expected PassphraseSource to be SourceLiteral, got %v", info.PassphraseSource)
	}

	// Nothing of the PlatformInfo, however it is displayed, may hold a secret
	for _, secret := range []string{secretToken, secretPassphrase} {
		for _, field := range []string{info.Host, info.Type, info.TokenRef, info.PublicKey, info.PassphraseRef, info.TokenLocation(), info.PassphraseLocation()} {
			if strings.Contains(field, secret) {
				t.Errorf("A field of PlatformInfo holds the secret '%s' : '%s'", secret, field)
			}
		}
		// And not through a formatted dump either
		if dump := fmt.Sprintf("%+v %#v", info, info); strings.Contains(dump, secret) {
			t.Errorf("The dump of PlatformInfo holds the secret '%s' : '%s'", secret, dump)
		}
	}
}

func TestPlatformsAreSortedByHost(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()
	ConfiguredPlatforms = map[string]Platform{
		"gitlab.com": {typeOf: "gitlab", token: "t"},
		"a.com":      {typeOf: "github", token: "t"},
		"github.com": {typeOf: "github", token: "t"},
	}

	infos := Platforms()
	expected := []string{"a.com", "github.com", "gitlab.com"}
	if len(infos) != len(expected) {
		t.Fatalf("Expected %d platforms, got %d", len(expected), len(infos))
	}
	for index, host := range expected {
		if infos[index].Host != host {
			t.Errorf("Expected platform %d to be '%s', got '%s'", index, host, infos[index].Host)
		}
	}
}

func TestLiteralToken(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()
	ConfiguredPlatforms = map[string]Platform{
		"literal.com": {typeOf: "github", token: "ghp_abc"},
		"env.com":     {typeOf: "github", token: "env.MY_TOKEN"},
		"keyring.com": {typeOf: "github", token: "keyring.my-token"},
		"none.com":    {typeOf: "github"},
	}

	if token, ok := LiteralToken("literal.com"); !ok || token != "ghp_abc" {
		t.Errorf("Expected 'ghp_abc' and true, got '%s' and %v", token, ok)
	}
	for _, host := range []string{"env.com", "keyring.com", "none.com", "unknown.com"} {
		if token, ok := LiteralToken(host); ok || token != "" {
			t.Errorf("Expected '' and false for '%s', got '%s' and %v", host, token, ok)
		}
	}
}

func TestPlatformWithoutTokenIsKept(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		viper.Reset()
	}()
	viper.Reset()
	viper.Set("platforms", map[string]interface{}{
		"notoken": map[string]interface{}{"type": "github"},
	})
	BuildPlatforms()

	if _, ok := ConfiguredPlatforms["notoken"]; !ok {
		t.Fatalf("Expected 'notoken' to be kept")
	}
	if IsComplete("notoken") {
		t.Errorf("Expected 'notoken' to be incomplete")
	}
	infos := Platforms()
	if len(infos) != 1 || infos[0].Host != "notoken" {
		t.Fatalf("Expected 'notoken' to be listed, got %v", infos)
	}
	if infos[0].Complete() {
		t.Errorf("Expected the listed platform to be incomplete")
	}
	if infos[0].TokenSource != SourceNone {
		t.Errorf("Expected TokenSource to be SourceNone, got %v", infos[0].TokenSource)
	}
}

func TestPlatformWithoutTypeIsKept(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		viper.Reset()
	}()
	viper.Reset()
	viper.Set("platforms", map[string]interface{}{
		"notype": map[string]interface{}{"token": "env.MY_TOKEN"},
	})
	BuildPlatforms()

	if _, ok := ConfiguredPlatforms["notype"]; !ok {
		t.Fatalf("Expected 'notype' to be kept")
	}
	if IsComplete("notype") {
		t.Errorf("Expected 'notype' to be incomplete")
	}
	infos := Platforms()
	if len(infos) != 1 || infos[0].Host != "notype" {
		t.Fatalf("Expected 'notype' to be listed, got %v", infos)
	}
	if infos[0].Complete() {
		t.Errorf("Expected the listed platform to be incomplete")
	}
}

func TestIsComplete(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()
	ConfiguredPlatforms = map[string]Platform{
		"ok.com": {typeOf: "github", token: "env.MY_TOKEN"},
	}
	if !IsComplete("ok.com") {
		t.Errorf("Expected 'ok.com' to be complete")
	}
	if IsComplete("unknown.com") {
		t.Errorf("Expected an unknown platform not to be complete")
	}
}

func TestBuildPlatformsReadsBothPassphraseKeys(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		viper.Reset()
	}()
	viper.Reset()
	viper.Set("platforms", map[string]interface{}{
		"documented": map[string]interface{}{"type": "github", "token": "t", "public_key_passphrase": "documented"},
		"legacy":     map[string]interface{}{"type": "github", "token": "t", "public_key_password": "legacy"},
		"both": map[string]interface{}{"type": "github", "token": "t",
			"public_key_passphrase": "documented", "public_key_password": "legacy"},
	})
	BuildPlatforms()

	expected := map[string]string{
		"documented": "documented",
		"legacy":     "legacy",
		"both":       "documented",
	}
	for host, passphrase := range expected {
		if got := ConfiguredPlatforms[host].publicKeyPassword; got != passphrase {
			t.Errorf("Expected passphrase of '%s' to be '%s', got '%s'", host, passphrase, got)
		}
	}
}

func TestBuildPlatformsClearsStaleEntries(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		viper.Reset()
	}()
	viper.Reset()

	viper.Set("platforms", map[string]interface{}{
		"github": map[string]interface{}{"type": "github", "token": "t"},
		"gitlab": map[string]interface{}{"type": "gitlab", "token": "t"},
	})
	BuildPlatforms()
	if len(ConfiguredPlatforms) != 2 {
		t.Fatalf("Expected 2 configured platforms, got %d", len(ConfiguredPlatforms))
	}

	viper.Set("platforms", map[string]interface{}{
		"github": map[string]interface{}{"type": "github", "token": "t"},
	})
	BuildPlatforms()
	if len(ConfiguredPlatforms) != 1 {
		t.Errorf("Expected 1 configured platform, got %d", len(ConfiguredPlatforms))
	}
	if _, ok := ConfiguredPlatforms["gitlab"]; ok {
		t.Errorf("Expected 'gitlab.com' to be gone after the rebuild")
	}
}

func TestBuildPlatformsNonStringValue(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
		viper.Reset()
	}()
	viper.Reset()
	viper.Set("platforms", map[string]interface{}{
		"github": map[string]interface{}{"type": "github", "token": 12345},
	})

	// A token written as a number must not panic
	BuildPlatforms()
	if got := ConfiguredPlatforms["github"].token; got != "12345" {
		t.Errorf("Expected token to be '12345', got '%s'", got)
	}
}

func TestAsString(t *testing.T) {
	if got := asString(nil); got != "" {
		t.Errorf("Expected '' for nil, got '%s'", got)
	}
	if got := asString("value"); got != "value" {
		t.Errorf("Expected 'value', got '%s'", got)
	}
	if got := asString(42); got != "42" {
		t.Errorf("Expected '42', got '%s'", got)
	}
	if got := asString(true); got != "true" {
		t.Errorf("Expected 'true', got '%s'", got)
	}
}

func TestReloadConfigPicksUpChanges(t *testing.T) {
	// Setup
	originalInputConfigFile := commons.InputConfigFile
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		commons.InputConfigFile = originalInputConfigFile
		ConfiguredPlatforms = originalConfiguredPlatforms
		viper.Reset()
	}()
	viper.Reset()

	commons.InputConfigFile = "test_reload_config.yaml"
	defer os.Remove(commons.InputConfigFile)

	first := `
platforms:
  github.com:
    type: github
    token: env.OLD_TOKEN
`
	if err := os.WriteFile(commons.InputConfigFile, []byte(first), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	if err := ReloadConfig(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if got := ConfiguredPlatforms["github.com"].token; got != "env.OLD_TOKEN" {
		t.Errorf("Expected token to be 'env.OLD_TOKEN', got '%s'", got)
	}

	second := `
platforms:
  github.com:
    type: github
    token: keyring.new-token
`
	if err := os.WriteFile(commons.InputConfigFile, []byte(second), 0644); err != nil {
		t.Fatalf("Failed to update test config file: %v", err)
	}
	if err := ReloadConfig(); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if got := ConfiguredPlatforms["github.com"].token; got != "keyring.new-token" {
		t.Errorf("Expected token to be 'keyring.new-token', got '%s'", got)
	}
}

func TestReloadConfigMissingFile(t *testing.T) {
	// Setup
	originalInputConfigFile := commons.InputConfigFile
	defer func() {
		commons.InputConfigFile = originalInputConfigFile
		viper.Reset()
	}()

	commons.InputConfigFile = "non_existent_reload_config.yaml"
	if err := ReloadConfig(); err == nil {
		t.Errorf("Expected an error for a missing file, got nil")
	}
}

func TestGetUnsetToken(t *testing.T) {
	// Setup
	originalConfiguredPlatforms := ConfiguredPlatforms
	defer func() {
		ConfiguredPlatforms = originalConfiguredPlatforms
	}()

	// Test case where the environment variable referenced in the token is not set
	ConfiguredPlatforms = map[string]Platform{
		"github.com": {typeOf: "github", token: "env.NON_EXISTENT_ENV"},
	}
	if token := GetToken("github.com", nil); token != "" {
		t.Errorf("Expected token to be '', got '%s'", token)
	}
}
