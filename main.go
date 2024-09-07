package main

import (
	log "github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const DEFAULT_FOLDER = "/Users/admin_local/work"
const MAX_DEPTH = 3

func init() {

	log.SetFormatter(&log.TextFormatter{
		DisableLevelTruncation: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func main() {
	maxDepth := 2
	rootDir := DEFAULT_FOLDER
	nbIgnoreSlashes := strings.Count(rootDir, "/")

	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// handle possible path err, just in case...
			return err
		}
		if d.IsDir() && (strings.Count(path, string(os.PathSeparator))-nbIgnoreSlashes) > maxDepth {
			log.WithField("dir", path).Debug("Skip a dir")
			return fs.SkipDir
		} else if d.IsDir() {
			log.WithField("dir", path).Info("Inspecting dir")
			_, err := os.Stat(path + "/.git")
			if err == nil {
				log.WithField("dir", path).Info(" ✅ Found .git")
				return fs.SkipDir
			}
		}
		// ... process entry
		return nil
	})
}
