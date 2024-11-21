package utils

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mitchellh/colorstring"
	"github.com/spf13/viper"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils/tui"
	"os"
	"strings"
)

func HasInputConfig() bool {
	return commons.InputConfigFile != ""
}

func UseConfig() {
	if HasInputConfig() {
		viper.SetConfigFile(commons.InputConfigFile)
		viper.SetConfigType("yaml")
		err := viper.ReadInConfig()
		if err != nil {
			TraceWarn(colorstring.Color("Cannot read input config file : [red] " + commons.InputConfigFile))
			TraceWarn("Try run command without it")
		}
	}
}

func GetToken(host string, spinner *tea.Program) string {
	rawValue := viper.GetString("tokens." + host)
	if strings.HasPrefix(rawValue, commons.ENV_VARIABLE) {
		envValue := os.Getenv(strings.TrimPrefix(rawValue, commons.ENV_VARIABLE))
		if envValue == "" {
			TraceWarn(strings.TrimPrefix(rawValue, commons.ENV_VARIABLE) + " referenced in config-file is not set")
			spinner.Send(tui.ErrorMessage{Error: strings.TrimPrefix(rawValue, commons.ENV_VARIABLE) + " referenced in config-file is not set"})
			return ""
		}
		return envValue
	} else {
		return rawValue
	}
}
