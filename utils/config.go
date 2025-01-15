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

func GetToken(host string, spinner *tea.Program) string {
	rawValue := viper.GetString("tokens." + host)
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
}
