package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/yodamad/heimdall/commons"
)

func Trace(msg string, isDebug bool) {
	if commons.TUIMode {
		if isDebug {
			log.Debug(CleanForLog(msg))
		} else {
			log.Info(CleanForLog(msg))
		}
		return
	}
	if isDebug {
		if commons.Verbose {
			fmt.Println(msg)
		}
		log.Debug(CleanForLog(msg))
	} else {
		fmt.Println(msg)
		log.Info(CleanForLog(msg))
	}
}

func TraceWarn(msg string) {
	if commons.TUIMode {
		log.Warn(CleanForLog(msg))
		return
	}
	if commons.NoColor {
		msg = CleanForLog(msg)
		fmt.Println("⚠  " + msg)
	} else {
		fmt.Println(ColorString("[light_yellow]⚠  " + msg + "[default]"))
	}
	log.Info(CleanForLog(msg))
}

func OverrideLogFile() {
	if commons.LogDir == commons.DefaultLogFolder {
		Trace(ColorString("📝 Log file written in [light_blue]"+commons.DefaultLogFolder+"heimdall.log"), false)
		return
	}

	path := filepath.Join(commons.LogDir, "heimdall.log")
	// The log records the hosts, the names of the keyring entries and the authentication failures, so
	// it is for its owner only. A log directory can be a shared one, where the file may already be a
	// symlink planted by someone else : appending to it would write as the user running Heimdall.
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		TraceWarn("Not writing the log in " + path + " : it is a symbolic link")
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		TraceWarn("Cannot write the log in " + path + " : " + err.Error())
		return
	}
	log.SetOutput(f)
	Trace(ColorString("📝 Log file written in [light_blue]"+path), false)
}

// Inspired from https://github.com/acarl005/stripansi/blob/master/stripansi.go
const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

func CleanForLog(str string) string {
	return re.ReplaceAllString(str, "")
}

func ColorString(str string) string {
	msg := colorstring.Color(str)
	if commons.NoColor {
		return CleanForLog(msg)
	}
	return msg
}
