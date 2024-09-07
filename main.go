package main

import (
	log "github.com/sirupsen/logrus"
	"heimdall/cmd"
	"os"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
		Di
	})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func main() {
	cmd.Execute()
}
