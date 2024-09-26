package utils

import (
	log "github.com/sirupsen/logrus"
	"heimdall/commons"
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
