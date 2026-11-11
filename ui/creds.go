package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yodamad/heimdall/commons"
	"github.com/yodamad/heimdall/creds"
	"github.com/yodamad/heimdall/secrets"
)

// The steps of the credentials screen
type credStep int

const (
	credPick   credStep = iota // choosing the platforms to update
	credName                   // naming the entry of the keyring
	credSecret                 // typing the secret, which is never displayed
)

// openCreds shows the credentials screen and checks the platforms
func (m *model) openCreds() tea.Cmd {
	m.mode = modeCreds
	m.credStep = credPick
	m.credNote = ""
	m.credInput.SetValue("")
	m.credInput.Blur()
	if m.credPicked == nil {
		m.credPicked = map[string]bool{}
	}
	if len(m.creds) == 0 {
		m.credCursor = 0
	}
	return m.recheckCreds()
}

// openCredsFor shows the screen with one platform already chosen, which is what the offer made after
// an authentication failure leads to
func (m *model) openCredsFor(host string) tea.Cmd {
	cmd := m.openCreds()
	m.credPicked = map[string]bool{host: true}
	for index, result := range m.creds {
		if result.Host == host {
			m.credCursor = index
		}
	}
	return cmd
}

func (m *model) recheckCreds() tea.Cmd {
	m.credChecking = true
	return credCheckCmd()
}

// credTargets are the platforms an update applies to : the chosen ones, or the one under the cursor
func (m *model) credTargets() []string {
	var hosts []string
	for _, result := range m.creds {
		if m.credPicked[result.Host] {
			hosts = append(hosts, result.Host)
		}
	}
	if len(hosts) == 0 && m.credCursor < len(m.creds) {
		hosts = append(hosts, m.creds[m.credCursor].Host)
	}
	return hosts
}

func (m *model) handleCredsKey(key tea.KeyMsg) tea.Cmd {
	switch m.credStep {
	case credName, credSecret:
		return m.handleCredsInput(key)
	}

	switch key.String() {
	case "esc", "c", "q":
		m.mode = modeList
		m.credNote = ""
		return nil

	case "up", "k":
		if m.credCursor > 0 {
			m.credCursor--
		}

	case "down", "j":
		if m.credCursor < len(m.creds)-1 {
			m.credCursor++
		}

	case " ":
		if m.credCursor < len(m.creds) {
			host := m.creds[m.credCursor].Host
			if m.credPicked[host] {
				delete(m.credPicked, host)
			} else {
				m.credPicked[host] = true
			}
		}

	case "a":
		if len(m.credPicked) == len(m.creds) {
			m.credPicked = map[string]bool{}
		} else {
			for _, result := range m.creds {
				m.credPicked[result.Host] = true
			}
		}

	case "r":
		if !m.credChecking {
			return m.recheckCreds()
		}

	case "enter":
		return m.startCredUpdate()
	}
	return nil
}

// startCredUpdate moves on to naming the keyring entry, once there is somewhere to store a secret
func (m *model) startCredUpdate() tea.Cmd {
	targets := m.credTargets()
	if len(targets) == 0 {
		m.credNote = "No platform is configured, add one with heimdall credentials update <host>"
		return nil
	}
	if !secrets.Available() {
		m.credNote = "No keyring on this system : set the token in an environment variable and point " +
			"the config file at it with env.MY_TOKEN"
		return nil
	}
	if err := creds.EnsureWritable(commons.InputConfigFile); err != nil {
		m.credNote = "Cannot write the config file : " + err.Error()
		return nil
	}

	m.credStep = credName
	m.credNote = ""
	m.credInput.EchoMode = textinput.EchoNormal
	m.credInput.Prompt = "entry name: "
	m.credInput.SetValue(defaultEntry(targets))
	m.credInput.CursorEnd()
	m.credInput.Focus()
	return textinput.Blink
}

func defaultEntry(targets []string) string {
	if len(targets) == 1 && secrets.ValidKey(targets[0]) {
		return targets[0]
	}
	return "shared"
}

func (m *model) handleCredsInput(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc":
		// Back one step, and the secret typed so far is dropped
		m.credInput.SetValue("")
		if m.credStep == credSecret {
			m.credStep = credName
			m.credInput.EchoMode = textinput.EchoNormal
			m.credInput.Prompt = "entry name: "
			m.credInput.SetValue(m.credEntry)
			m.credInput.CursorEnd()
			return nil
		}
		m.credStep = credPick
		m.credInput.Blur()
		return nil

	case "enter":
		value := strings.TrimSpace(m.credInput.Value())
		if m.credStep == credName {
			if !secrets.ValidKey(value) {
				m.credNote = "An entry name may only hold letters, digits, dots, dashes and underscores"
				return nil
			}
			m.credEntry = value
			m.credNote = ""
			m.credStep = credSecret
			// From here on nothing typed is displayed
			m.credInput.EchoMode = textinput.EchoPassword
			m.credInput.EchoCharacter = '•'
			m.credInput.Prompt = "secret: "
			m.credInput.SetValue("")
			return nil
		}

		if value == "" {
			m.credNote = "An empty secret is not stored"
			return nil
		}
		// The secret leaves the screen here and lives only in the closure of the command
		m.credInput.SetValue("")
		m.credInput.Blur()
		m.credStep = credPick
		targets := m.credTargets()
		m.credNote = "Storing the secret..."
		return credSaveCmd(m.credEntry, value, targets)
	}

	var cmd tea.Cmd
	m.credInput, cmd = m.credInput.Update(key)
	return cmd
}

// credsLines renders the screen
func (m *model) credsLines() []string {
	title := " " + boldStyle.Render("Credentials")
	if len(m.creds) > 0 {
		title += dimStyle.Render("   " + plural(len(m.creds), "platform", "platforms"))
		if nb := len(m.credPicked); nb > 0 {
			title += dimStyle.Render(", " + strconv.Itoa(nb) + " selected")
		}
	}
	lines := []string{title, ""}

	switch {
	case m.credChecking && len(m.creds) == 0:
		lines = append(lines, " "+m.spin.View()+" "+dimStyle.Render("Checking the credentials of your platforms..."))
	case len(m.creds) == 0:
		lines = append(lines, " "+warnStyle.Render("No platform is configured."))
		lines = append(lines, "")
		lines = append(lines, " "+dimStyle.Render("Add one with  heimdall credentials update github.com"))
	default:
		for index, result := range m.creds {
			lines = append(lines, m.credLine(index, result))
		}
	}

	if m.credNote != "" {
		lines = append(lines, "", " "+warnStyle.Render(fit(m.credNote, m.width-2)))
	}
	return lines
}

func (m *model) credLine(index int, result creds.Result) string {
	box := "[ ]"
	if m.credPicked[result.Host] {
		box = "[x]"
	}

	hostWidth := min(34, max(14, m.width/3))
	host := fit(result.Host, hostWidth)
	if m.credPicked[result.Host] {
		host = lipgloss.NewStyle().Foreground(selectColor).Render(host)
	}

	state := credState(result)
	detail := result.Detail
	if detail == "" && result.User != "" {
		detail = "authenticates as " + result.User
	}
	if result.Location != "" {
		detail = result.Location + "   " + detail
	}

	line := " " + box + " " + host + " " + state
	if room := m.width - lipgloss.Width(line) - 3; room > 10 {
		line += "  " + dimStyle.Render(fit(detail, room))
	}
	if index == m.credCursor {
		return lipgloss.NewStyle().Background(cursorColor).Render(fit(line, m.width-1))
	}
	return line
}

func credState(result creds.Result) string {
	label := fit(result.Status.Label(), 12)
	switch result.Status {
	case creds.StatusValid:
		return okStyle.Render(label)
	case creds.StatusExpiring:
		return warnStyle.Render(label)
	case creds.StatusRejected, creds.StatusNoToken, creds.StatusUnresolved:
		return koStyle.Render(label)
	default:
		return dimStyle.Render(label)
	}
}

// credsKeys lists what can be done on the screen right now
func (m *model) credsKeys() []string {
	switch m.credStep {
	case credName:
		return []string{"enter continue", "esc back"}
	case credSecret:
		return []string{"enter store and save", "esc back"}
	}
	if len(m.creds) == 0 {
		return []string{"esc go back"}
	}
	keys := []string{"↑↓ move", "space select", "a all"}
	if nb := len(m.credTargets()); nb > 1 {
		keys = append(keys, "enter update "+plural(nb, "platform", "platforms"))
	} else {
		keys = append(keys, "enter update")
	}
	return append(keys, "r check again", "esc go back")
}

// credsPrompt is what the status line says on the screen
func (m *model) credsPrompt() string {
	switch m.credStep {
	case credName:
		return " " + m.credInput.View()
	case credSecret:
		return " " + dimStyle.Render("for "+strings.Join(m.credTargets(), ", ")+"  ") + m.credInput.View()
	}
	if m.credChecking {
		return " " + dimStyle.Render("checking...")
	}
	return ""
}
