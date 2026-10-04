package gitops

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/entity"
)

// LocalStatus reads the state of a repository without any network access.
// Remote changes are computed against the last fetched origin ref, so they
// may be stale until Fetch is called.
func LocalStatus(path string) entity.GitFolder {
	gf := entity.GitFolder{Path: path}
	if info, err := os.Stat(fetchHead(path)); err == nil {
		gf.FetchedAt = info.ModTime()
	}

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

// IncomingCommits lists the commits of origin/<branch> missing locally.
// Each line holds the hash, the age and the subject of a commit, separated by tabs.
// The age is told as in "3 days".
func IncomingCommits(path string, branch string) []string {
	return commits(path, branch+".."+commons.RemoteName+"/"+branch)
}

// OutgoingCommits lists the local commits missing in origin/<branch>, as IncomingCommits does
func OutgoingCommits(path string, branch string) []string {
	return commits(path, commons.RemoteName+"/"+branch+".."+branch)
}

// RecentCommits lists the last commits of the current branch, as IncomingCommits does
func RecentCommits(path string, nb int) []string {
	return commits(path, "-"+strconv.Itoa(nb))
}

// Branches lists the local branches: the current one, then the most recently committed first
func Branches(path string) []entity.Branch {
	out, err := exec.Command("git", "-C", path, "for-each-ref", "--sort=-committerdate",
		"--format=%(HEAD)%09%(refname:short)%09%(upstream:short)%09%(upstream:track,nobracket)%09%(committerdate:unix)",
		"refs/heads").Output()
	if err != nil {
		return nil
	}
	var branches []entity.Branch
	for _, line := range lines(string(out)) {
		fields := strings.Split(line, "\t")
		if len(fields) != 5 {
			continue
		}
		branch := entity.Branch{
			Name:     fields[1],
			Current:  fields[0] == "*",
			Upstream: fields[2],
			Gone:     fields[3] == "gone",
			Age:      age(fields[4]),
		}
		for _, part := range strings.Split(fields[3], ", ") {
			if nb, ok := strings.CutPrefix(part, "ahead "); ok {
				branch.Ahead, _ = strconv.Atoi(nb)
			} else if nb, ok := strings.CutPrefix(part, "behind "); ok {
				branch.Behind, _ = strconv.Atoi(nb)
			}
		}
		if branch.Current {
			branches = append([]entity.Branch{branch}, branches...)
		} else {
			branches = append(branches, branch)
		}
	}
	return branches
}

// Stashes counts the stashes of the repository
func Stashes(path string) int {
	out, err := exec.Command("git", "-C", path, "stash", "list").Output()
	if err != nil {
		return 0
	}
	return len(lines(string(out)))
}

// Switch checks a local branch out. When git refuses, the error is what it says about it.
func Switch(path string, branch string) error {
	out, err := exec.Command("git", "-C", path, "switch", branch).CombinedOutput()
	if err == nil {
		return nil
	}
	said := strings.TrimSpace(strings.ReplaceAll(string(out), "\t", "  "))
	said = strings.TrimPrefix(strings.TrimPrefix(said, "error: "), "fatal: ")
	if said == "" {
		return err
	}
	return errors.New(said)
}

func commits(path string, revisions string) []string {
	out, err := exec.Command("git", "-C", path, "log", "--format=%h%x09%ct%x09%s", revisions).Output()
	if err != nil {
		return nil
	}
	commits := lines(string(out))
	for i, commit := range commits {
		if fields := strings.SplitN(commit, "\t", 3); len(fields) == 3 {
			commits[i] = fields[0] + "\t" + age(fields[1]) + "\t" + fields[2]
		}
	}
	return commits
}

// age tells how long ago a unix date was. Git would tell it in the language of the
// user and at length, this one is the same for everybody and fits in a column.
func age(unix string) string {
	seconds, err := strconv.ParseInt(strings.TrimSpace(unix), 10, 64)
	if err != nil {
		return ""
	}
	count := func(nb float64, one string) string {
		if int(nb) == 1 {
			return "1 " + one
		}
		return strconv.Itoa(int(nb)) + " " + one + "s"
	}
	since := time.Since(time.Unix(seconds, 0))
	switch {
	case since < time.Minute:
		return "just now"
	case since < time.Hour:
		return count(since.Minutes(), "minute")
	case since < 24*time.Hour:
		return count(since.Hours(), "hour")
	case since < 60*24*time.Hour:
		return count(since.Hours()/24, "day")
	case since < 2*365*24*time.Hour:
		return count(since.Hours()/24/30, "month")
	}
	return count(since.Hours()/24/365, "year")
}

func fetchHead(path string) string {
	return filepath.Join(path, ".git", "FETCH_HEAD")
}

// markFetched updates the date of FETCH_HEAD as git does, go-git does not maintain it
func markFetched(path string) {
	now := time.Now()
	if os.Chtimes(fetchHead(path), now, now) != nil {
		if f, err := os.OpenFile(fetchHead(path), os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			f.Close()
		}
	}
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
