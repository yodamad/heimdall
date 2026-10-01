package gitops

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func IsGitDir(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// Discover lists the git folders found under root, down to depth levels.
// If root is itself a git folder, it is the only one returned.
func Discover(root string, depth int) []string {
	root = filepath.Clean(root)
	if IsGitDir(root) {
		return []string{root}
	}
	found := make([]string, 0)
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() || path == root {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if strings.Count(rel, string(os.PathSeparator))+1 > depth {
			return fs.SkipDir
		}
		if IsGitDir(path) {
			found = append(found, path)
			return fs.SkipDir
		}
		return nil
	})
	return found
}
