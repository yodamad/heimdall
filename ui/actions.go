package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/gitops"
	"github.com/yodamad/heimdall/utils"
)

// Max number of repositories processed at the same time
const maxWorkers = 6

// Number of commits of the current branch listed in the detail pane
const nbRecentCommits = 5

var workers = make(chan struct{}, maxWorkers)

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

func statusCmd(path string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()

		msg := statusMsg{gf: gitops.LocalStatus(path)}
		if msg.gf.Err != "" {
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
		return msg
	}
}

func fetchCmd(path string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		return opMsg{path: path, kind: busyFetch, err: gitops.Fetch(path)}
	}
}

func pullCmd(path string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		return opMsg{path: path, kind: busyPull, err: gitops.Pull(path)}
	}
}

func switchCmd(path string, branch string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()
		return opMsg{path: path, kind: busySwitch, err: gitops.Switch(path, branch)}
	}
}

func execCmd(gf entity.GitFolder, cmds []string) tea.Cmd {
	return func() tea.Msg {
		workers <- struct{}{}
		defer func() { <-workers }()

		msg := cmdMsg{path: gf.Path}
		for _, cmd := range cmds {
			msg.results = append(msg.results, utils.ExecCmd(cmd, gf))
		}
		return msg
	}
}
