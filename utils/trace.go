package utils

import (
	"fmt"
	"os"
	"regexp"

	"github.com/mitchellh/colorstring"
	log "github.com/sirupsen/logrus"
	"github.com/yodamad/heimdall/commons"
)

func Trace(msg string, isDebug bool) {
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
	if commons.NoColor {
		msg = CleanForLog(msg)
		fmt.Println("⚠  " + msg)
	} else {
		fmt.Println(ColorString("[light_yellow]⚠  " + msg + "[default]"))
	}
	log.Info(CleanForLog(msg))
}

func OverrideLogFile() {
	if commons.LogDir != commons.DefaultLogFolder {
		os.RemoveAll("heimdall.log")
		f, _ := os.OpenFile(commons.LogDir+"/heimdall.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		log.SetOutput(f)
		Trace(ColorString("📝 Log file written in [light_blue]"+commons.LogDir+"/heimdall.log"), false)
	} else {
		Trace(ColorString("📝 Log file written in [light_blue]"+commons.DefaultLogFolder+"heimdall.log"), false)
	}
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
