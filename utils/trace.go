package utils

import log "github.com/sirupsen/logrus"

func Trace(msg string, isDebug bool) {
	if isDebug {
		log.Debug(msg)
	} else {
		println(msg)
		log.Info(msg)
	}
}

func TraceWarn(msg string) {
	println(msg)
	log.Warn(msg)
}
