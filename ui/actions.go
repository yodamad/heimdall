package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/yodamad/heimdall/entity"
	"github.com/yodamad/heimdall/gitops"
	"github.com/yodamad/heimdall/utils"
)

// Max number of repositories processed at the same time
const maxWorkers = 6

var workers = make(chan struct{}, maxWorkers)

type discoveredMsg struct {
	paths []string
}

type statusMsg struct {
	gf       entity.GitFolder
	incoming []string
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
		if entity.HasRemoteChanges(msg.gf) {
			msg.incoming = gitops.IncomingCommits(path, msg.gf.CurrentBranch)
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
