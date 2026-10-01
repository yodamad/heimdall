package gitops

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
)

// LocalStatus reads the state of a repository without any network access.
// Remote changes are computed against the last fetched origin ref, so they
// may be stale until Fetch is called.
func LocalStatus(path string) entity.GitFolder {
	gf := entity.GitFolder{Path: path}

	repo, err := git.PlainOpen(path)
	if err != nil {
		gf.Err = err.Error()
		return gf
	}
	if remote, err := repo.Remote(commons.RemoteName); err == nil && len(remote.Config().URLs) > 0 {
		gf.RemoteURL = remote.Config().URLs[0]
		gf.ConnectionType = ConnectionType(gf.RemoteURL)
	}
	ref, err := repo.Head()
	if err != nil {
		gf.Err = err.Error()
		return gf
	}
	gf.CurrentBranch = ref.Name().Short()

	out, err := exec.Command("git", "-C", path, "status", "--porcelain").Output()
	if err != nil {
		gf.Err = "git status : " + err.Error()
		return gf
	}
	gf.DetailedLocalChanges = string(out)
	gf.ChangedFiles = lines(string(out))
	gf.HasLocalChanges = len(gf.ChangedFiles) > 0

	branch := gf.CurrentBranch
	out, err = exec.Command("git", "-C", path, "rev-list", "--left-right", "--count", branch+"..."+commons.RemoteName+"/"+branch).Output()
	if err == nil {
		counts := strings.Fields(string(out))
		if len(counts) == 2 {
			gf.Ahead, _ = strconv.Atoi(counts[0])
			gf.RemoteChanges = counts[1]
		}
	}
	return gf
}

// IncomingCommits lists the commits of origin/<branch> missing locally
func IncomingCommits(path string, branch string) []string {
	out, err := exec.Command("git", "-C", path, "log", "--oneline", branch+".."+commons.RemoteName+"/"+branch).Output()
	if err != nil {
		return nil
	}
	return lines(string(out))
}

func lines(s string) []string {
	var res []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			res = append(res, l)
		}
	}
	return res
}
