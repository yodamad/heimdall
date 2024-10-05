package utils

import (
	log "github.com/sirupsen/logrus"
	"github.com/yodamad/heimdall/commons"
	"os"
)

func Trace(msg string, isDebug bool) {
	if isDebug {
		log.Debug(msg)
	} else {
		println(msg)
		log.Info(msg)
	}
}

func TraceWarn(msg string) {
	if commons.Verbose {
		println(msg)
	}
	log.Warn(msg)
}

func OverrideLogFile() {
	if commons.LogDir != commons.DEFAULT_FOLDER {
		os.RemoveAll("heimdall.log")
		f, _ := os.OpenFile(commons.LogDir+"/heimdall.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		log.SetOutput(f)
	}
}
