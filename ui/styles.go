package ui

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/entity"
)

// state of a repository, ordered from the most to the least urgent
type state int

const (
	stateBroken state = iota // error, or diverged from origin
	stateBehind
	stateDirty
	stateAhead
	stateClean
	stateLoading
)

// A fetch older than this is not trusted anymore
const staleAfter = 24 * time.Hour

// The state colors, from warm to cold as urgency decreases. They are used for the state only.
var stateColors = map[state]lipgloss.Color{
	stateBroken:  lipgloss.Color("#E5484D"),
	stateBehind:  lipgloss.Color("#F5A524"),
	stateDirty:   lipgloss.Color("#E9D66B"),
	stateAhead:   lipgloss.Color("#5BC8AF"),
	stateClean:   lipgloss.Color("#6E7B8B"),
	stateLoading: lipgloss.Color("#6E7B8B"),
}

var stateNames = map[state]string{
	stateBroken:  "in trouble",
	stateBehind:  "behind",
	stateDirty:   "changed",
	stateAhead:   "not pushed",
	stateClean:   "up to date",
	stateLoading: "being read",
}

var (
	dimStyle    = lipgloss.NewStyle().Foreground(stateColors[stateClean])
	okStyle     = lipgloss.NewStyle().Foreground(stateColors[stateAhead])
	koStyle     = lipgloss.NewStyle().Foreground(stateColors[stateBroken])
	warnStyle   = lipgloss.NewStyle().Foreground(stateColors[stateBehind])
	boldStyle   = lipgloss.NewStyle().Bold(true)
	selectColor = lipgloss.AdaptiveColor{Light: "#2F5FD0", Dark: "#9DB8FF"}
	cursorColor = lipgloss.AdaptiveColor{Light: "#DDE3EC", Dark: "#2B3340"}
)

func behind(gf entity.GitFolder) int {
	nb, _ := strconv.Atoi(strings.TrimSpace(gf.RemoteChanges))
	return nb
}

func stateOf(r *row) state {
	switch {
	case !r.loaded:
		return stateLoading
	case r.gf.Err != "", behind(r.gf) > 0 && r.gf.Ahead > 0:
		return stateBroken
	case behind(r.gf) > 0:
		return stateBehind
	case r.gf.HasLocalChanges:
		return stateDirty
	case r.gf.Ahead > 0:
		return stateAhead
	}
	return stateClean
}

// stale repositories have not been fetched recently, their remote changes may be outdated
func stale(gf entity.GitFolder) bool {
	return gf.RemoteURL != "" && time.Since(gf.FetchedAt) > staleAfter
}

func plural(nb int, one string, many string) string {
	if nb == 1 {
		return "1 " + one
	}
	return strconv.Itoa(nb) + " " + many
}

func fetchAge(gf entity.GitFolder) string {
	if gf.FetchedAt.IsZero() {
		return "never fetched"
	}
	age := time.Since(gf.FetchedAt)
	switch {
	case age < time.Minute:
		return "fetched just now"
	case age < time.Hour:
		return "fetched " + plural(int(age.Minutes()), "minute", "minutes") + " ago"
	case age < 24*time.Hour:
		return "fetched " + plural(int(age.Hours()), "hour", "hours") + " ago"
	case age < 60*24*time.Hour:
		return "fetched " + plural(int(age.Hours()/24), "day", "days") + " ago"
	}
	return "fetched " + plural(int(age.Hours()/24/30), "month", "months") + " ago"
}

// A multi-line error message must not break the row it is displayed in
var oneLine = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ")

// fit truncates or pads s to exactly w cells
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = oneLine.Replace(s)
	if lipgloss.Width(s) > w {
		r := []rune(s)
		for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
			r = r[:len(r)-1]
		}
		s = string(r) + "…"
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

// fitLeft is like fit but keeps the end of s, which is the meaningful part of a path
func fitLeft(s string, w int) string {
	if w <= 0 {
		return ""
	}
	s = oneLine.Replace(s)
	if lipgloss.Width(s) > w {
		r := []rune(s)
		for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
			r = r[1:]
		}
		s = "…" + string(r)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}
