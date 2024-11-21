package utils

import (
	"github.com/mitchellh/colorstring"
	"github.com/spf13/viper"
	"github.com/yodamad/heimdall/commons"
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

func GetToken(host string) string {
	rawValue := viper.GetString("tokens." + host)
	if strings.HasPrefix(rawValue, commons.ENV_VARIABLE) {
		return os.Getenv(strings.TrimPrefix(rawValue, commons.ENV_VARIABLE))
	} else {
		return rawValue
	}
}
