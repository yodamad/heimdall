package utils

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/utils/tui"
	"os"
	"strings"
)

type Platform struct {
	typeOf            string
	token             string
	publicKey         string
	publicKeyPassword string
}

var ConfiguredPlatforms = make(map[string]Platform)

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
		if err != nil {
			fmt.Println(ColorString("[light_yellow]Cannot read config in file : [red]"+commons.InputConfigFile) + "[light_yellow] Ignore it...")
		}
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
	if !strings.HasSuffix(commons.WorkDir, "/") {
		commons.WorkDir += "/"
	}
}

func BuildPlatforms() {
	platforms := viper.GetStringMap("platforms")

	// Iterating over the map
	for key := range platforms {
		infos := viper.GetStringMap("platforms." + key)

		if infos["type"] == nil {
			TraceWarn(ColorString("[light_yellow]Platform [blue]" + key + "[light_yellow] is not correctly configured : missing [blue]type[light_yellow] field. Ignoring it..."))
			continue
		} else if infos["token"] == nil {
			TraceWarn(ColorString("[light_yellow]Platform [blue]" + key + "[light_yellow] is not correctly configured : missing [blue]token[light_yellow] field. Ignoring it..."))
			continue
		} else {
			platform := Platform{
				typeOf: infos["type"].(string),
				token:  infos["token"].(string),
			}
			if infos["public_key"] != nil {
				platform.publicKey = infos["public_key"].(string)
			}
			if infos["public_key_password"] != nil {
				platform.publicKeyPassword = infos["public_key_password"].(string)
			}
			ConfiguredPlatforms[key] = platform
		}
	}
}

func GetPlatformType(host string) string {
	if platform, isPresent := ConfiguredPlatforms[host]; isPresent {
		return platform.typeOf
	} else {
		return ""
	}
}

func GetToken(host string, spinner *tea.Program) string {
	if platform, isPresent := ConfiguredPlatforms[host]; isPresent {
		rawValue := platform.token
		if strings.HasPrefix(rawValue, commons.EnvVariable) {
			envValue := os.Getenv(strings.TrimPrefix(rawValue, commons.EnvVariable))
			if envValue == "" {
				if spinner != nil {
					spinner.Send(tui.ErrorMessage{Error: strings.TrimPrefix(rawValue, commons.EnvVariable) + " referenced in config-file is not set"})
				} else {
					TraceWarn(ColorString("[light_blue]" + strings.TrimPrefix(rawValue, commons.EnvVariable) + "[yellow] referenced in config-file is not set"))
				}
				return ""
			}
			return envValue
		} else {
			return rawValue
		}
	} else {
		TraceWarn(ColorString("[yellow]" + host + "[light_yellow] is not configured, cannot retrieve a token. Operation may fail"))
		return ""
	}
}

func GetPublicKey(host string, spinner *tea.Program) string {
	if platform, isPresent := ConfiguredPlatforms[host]; isPresent {
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
	if platform, isPresent := ConfiguredPlatforms[host]; isPresent {
		rawValue := platform.publicKeyPassword
		if strings.HasPrefix(rawValue, commons.EnvVariable) {
			envValue := os.Getenv(strings.TrimPrefix(rawValue, commons.EnvVariable))
			if envValue == "" {
				log := strings.TrimPrefix(rawValue, commons.EnvVariable) + " referenced in config-file is not set"
				if spinner != nil {
					spinner.Send(tui.ErrorMessage{Error: log})
				} else {
					TraceWarn(log)
				}
				return commons.SshkeyPassword
			}
			return envValue
		} else {
			return rawValue
		}
	} else {
		return commons.SshkeyPassword
	}
}
