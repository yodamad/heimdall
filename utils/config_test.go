package utils

import (
	"github.com/spf13/viper"
	"os"
	"testing"

	"github.com/yodamad/heimdall/commons"
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
	ConfiguredPlatforms := make(map[string]Platform)

	viper.Set("platforms", map[string]interface{}{
		"github.com": map[string]interface{}{
			"type": "github",
		},
		"gitlab.com": map[string]interface{}{
			"token": "env.GITLAB_TOKEN",
		},
	})
	BuildPlatforms()

	if len(ConfiguredPlatforms) != 0 {
		t.Errorf("Expected 0 configured platforms, got %d", len(ConfiguredPlatforms))
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
