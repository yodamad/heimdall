package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/activity"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/gitops"
	"github.com/yodamad/heimdall/utils"
)

// Max number of repositories processed at the same time
const maxWorkers = 6

// Number of commits of the current branch listed in the detail pane
const nbRecentCommits = 5

var workers = make(chan struct{}, maxWorkers)

// doneVerbs says what happened, once it has happened : busyVerbs (ui/view.go) is the running form
var doneVerbs = map[string]string{
	busyStatus:  "read",
	busyRefresh: "read",
	busyFetch:   "fetched",
	busyPull:    "pulled",
	busySwitch:  "switched",
	busyCmd:     "ran",
}

// logStart records that a repository is being worked on. It is called once the worker slot is
// held, so the log says what is actually running and not the whole queue at once.
//
// Reading a status is recorded as a debug line : it happens for every repository at startup and
// again after every action, and would bury the lines the user is waiting for.
func logStart(kind, name string) {
	level := activity.Info
	if kind == busyStatus || kind == busyRefresh {
		level = activity.Debug
	}
	activity.Record(level, name, busyVerbs[kind]+"…")
}

// logEnd records how it went. The text of the error is recorded, as it already is in the log file :
// nothing here is a secret, the credentials never leave their closure.
func logEnd(kind, name string, err error) {
	if err != nil {
		activity.Record(activity.Warn, name, doneVerbs[kind]+" failed : "+err.Error())
		return
	}
	level := activity.Info
	if kind == busyStatus || kind == busyRefresh {
		level = activity.Debug
	}
	activity.Record(level, name, doneVerbs[kind])
}

type discoveredMsg struct {
	paths []string
}

type statusMsg struct {
	gf       entity.GitFolder
	incoming []string
	outgoing []string
	recent   []string
	branches []entity.Branch
	stashes  int
}

type opMsg struct {
	path string
	kind string
	err  error
}

type cmdMsg struct {
	path    string
	results []entity.CmdInfo
}

func discoverCmd(root string, depth int) tea.Cmd {
	return func() tea.Msg {
		return discoveredMsg{paths: gitops.Discover(root, depth)}
	}
}

func statusCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		logStart(busyStatus, name)

		msg := statusMsg{gf: gitops.LocalStatus(path)}
		if msg.gf.Err != "" {
			activity.Record(activity.Warn, name, "can't be read : "+msg.gf.Err)
			return msg
		}
		msg.recent = gitops.RecentCommits(path, nbRecentCommits)
		msg.branches = gitops.Branches(path)
		msg.stashes = gitops.Stashes(path)
		if entity.HasRemoteChanges(msg.gf) {
			msg.incoming = gitops.IncomingCommits(path, msg.gf.CurrentBranch)
		}
		if msg.gf.Ahead > 0 {
			msg.outgoing = gitops.OutgoingCommits(path, msg.gf.CurrentBranch)
		}
		logEnd(busyStatus, name, nil)
		return msg
	}
}

func fetchCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		logStart(busyFetch, name)
		err := gitops.Fetch(path)
		logEnd(busyFetch, name, err)
		return opMsg{path: path, kind: busyFetch, err: err}
	}
}

func pullCmd(name, path string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		logStart(busyPull, name)
		err := gitops.Pull(path)
		logEnd(busyPull, name, err)
		return opMsg{path: path, kind: busyPull, err: err}
	}
}

func switchCmd(name, path string, branch string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		activity.Record(activity.Info, name, "switching to "+branch+"…")
		err := gitops.Switch(path, branch)
		logEnd(busySwitch, name, err)
		return opMsg{path: path, kind: busySwitch, err: err}
	}
}

func execCmd(name string, gf entity.GitFolder, cmds []string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		logStart(busyCmd, name)

		msg := cmdMsg{path: gf.Path}
		failed := 0
		for _, cmd := range cmds {
			result := utils.ExecCmd(cmd, gf)
			if result.ExitCode != 0 {
				failed++
				activity.Record(activity.Warn, name, cmd+" failed")
			}
			msg.results = append(msg.results, result)
		}
		if failed == 0 {
			activity.Record(activity.Info, name, doneVerbs[busyCmd]+" "+plural(len(cmds), "command", "commands"))
		}
		return msg
	}
}
