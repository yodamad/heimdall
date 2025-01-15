package utils

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/colorstring"
	"github.com/spf13/viper"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils/tui"
	"os"
	"strings"
)

type Platform struct {
	typeOf string
	token  string
}

var ConfiguredPlatforms = make(map[string]Platform)

func HasInputConfig() bool {
	_, err := os.Stat(commons.InputConfigFile)
	if err != nil {
		fmt.Println(colorstring.Color("[light_yellow]Cannot read input config file : [red]" + commons.InputConfigFile + "[light_yellow] Ignore it..."))
		fmt.Println("")
	}
	return commons.InputConfigFile != "" && err == nil
}

func UseConfig() {
	if HasInputConfig() {
		viper.SetConfigFile(commons.InputConfigFile)
		viper.SetConfigType("yaml")
		err := viper.ReadInConfig()
		BuildPlatform()
		if err != nil {
			fmt.Println(colorstring.Color("[light_yellow]Cannot read config in file : [red]"+commons.InputConfigFile) + "[light_yellow] Ignore it...")
		}
		if commons.WorkDir == commons.DefaultWorkDir {
			workDir := viper.GetString("work_dir")
			if workDir != "" {
				if info, err := os.Stat(workDir); err != nil || !info.IsDir() {
					fmt.Println(colorstring.Color("[light_yellow]The work_dir is not a valid directory: [red]" + workDir))
				} else {
					commons.WorkDir = workDir
				}
			}
		}
	}
	if !strings.HasSuffix(commons.WorkDir, "/") {
		commons.WorkDir += "/"
	}
}

func BuildPlatform() {
	platforms := viper.GetStringMap("platforms")

	// Iterating over the map
	for key := range platforms {
		infos := viper.GetStringMap("platforms." + key)
		platform := Platform{
			typeOf: infos["type"].(string),
			token:  infos["token"].(string),
		}
		ConfiguredPlatforms[key] = platform
	}
}

func GetToken(host string, spinner *tea.Program) string {
	if platform, isPresent := ConfiguredPlatforms[host]; isPresent {
		rawValue := platform.token
		if strings.HasPrefix(rawValue, commons.ENV_VARIABLE) {
			envValue := os.Getenv(strings.TrimPrefix(rawValue, commons.ENV_VARIABLE))
			if envValue == "" {
				TraceWarn(strings.TrimPrefix(rawValue, commons.ENV_VARIABLE) + " referenced in config-file is not set")
				if spinner != nil {
					spinner.Send(tui.ErrorMessage{Error: strings.TrimPrefix(rawValue, commons.ENV_VARIABLE) + " referenced in config-file is not set"})
				}
				return ""
			}
			return envValue
		} else {
			return rawValue
		}
	} else {
		TraceWarn(colorstring.Color("[yellow]" + host + "[light_yellow] is not configured, cannot retrieve a token. Operation may fail"))
		return ""
	}
}
