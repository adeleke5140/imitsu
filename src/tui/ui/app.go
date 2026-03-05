package ui

import (
	"fmt"
	"strings"

	"github.com/adeleke5140/imitsu/tui/api"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Tab int

const (
	TabSecrets Tab = iota
	TabTeams
	TabAccount
)

type AppModel struct {
	client *api.Client
	tab    Tab
	user   *api.User
	width  int
	height int

	secrets SecretsModel
	teams   TeamsModel
	account AccountModel

	loggedIn bool
}

func NewApp() AppModel {
	client := api.NewClient()
	m := AppModel{
		client:  client,
		account: NewAccountModel(client),
		secrets: NewSecretsModel(client),
		teams:   NewTeamsModel(client),
	}

	if client.IsLoggedIn() {
		m.loggedIn = true
		m.tab = TabSecrets
	} else {
		m.tab = TabAccount
	}

	return m
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{m.account.Init()}
	if m.loggedIn {
		cmds = append(cmds, m.secrets.Init(), m.teams.Init(), m.fetchUser())
	}
	return tea.Batch(cmds...)
}

func (m AppModel) fetchUser() tea.Cmd {
	return func() tea.Msg {
		user, err := m.client.WhoAmI()
		if err != nil {
			return loginErrMsg{err}
		}
		return loginSuccessMsg{*user}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if msg.String() == "q" && m.isTyping() {
				break
			}
			return m, tea.Quit

		case "s":
			if !m.isTyping() && m.loggedIn {
				m.tab = TabSecrets
				return m, nil
			}
		case "t":
			if !m.isTyping() && m.loggedIn {
				m.tab = TabTeams
				return m, nil
			}
		case "a":
			if !m.isTyping() {
				m.tab = TabAccount
				return m, nil
			}
		case "ctrl+l":
			if m.loggedIn {
				m.client.Logout()
				m.loggedIn = false
				m.user = nil
				m.account = NewAccountModel(m.client)
				m.tab = TabAccount
				return m, m.account.Init()
			}
		}

	case loginSuccessMsg:
		m.loggedIn = true
		m.user = &msg.user
		m.account.SetUser(&msg.user)
		m.secrets = NewSecretsModel(m.client)
		m.teams = NewTeamsModel(m.client)
		m.tab = TabSecrets
		return m, tea.Batch(m.secrets.Init(), m.teams.Init())

	case loginErrMsg:
		if m.loggedIn {
			m.loggedIn = false
			m.user = nil
			m.account = NewAccountModel(m.client)
			m.account.err = "session expired, please login again"
			m.tab = TabAccount
			return m, m.account.Init()
		}
	}

	// Key events only go to the active tab
	if _, ok := msg.(tea.KeyMsg); ok {
		switch m.tab {
		case TabSecrets:
			var cmd tea.Cmd
			m.secrets, cmd = m.secrets.Update(msg)
			return m, cmd
		case TabTeams:
			var cmd tea.Cmd
			m.teams, cmd = m.teams.Update(msg)
			return m, cmd
		case TabAccount:
			var cmd tea.Cmd
			m.account, cmd = m.account.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Data messages go to all models so background fetches aren't lost
	var cmd1, cmd2, cmd3 tea.Cmd
	m.secrets, cmd1 = m.secrets.Update(msg)
	m.teams, cmd2 = m.teams.Update(msg)
	m.account, cmd3 = m.account.Update(msg)
	return m, tea.Batch(cmd1, cmd2, cmd3)
}

func (m AppModel) isTyping() bool {
	if m.tab == TabAccount && m.account.focus == FocusForm {
		return true
	}
	if m.tab == TabSecrets && (m.secrets.view == SecretCreate || m.secrets.view == SecretExport) {
		return true
	}
	return false
}

func (m AppModel) View() string {
	w := m.width
	h := m.height
	if w == 0 {
		w = 80
	}
	if h == 0 {
		h = 24
	}

	tabBar := m.renderTabs()
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	block := lipgloss.JoinVertical(lipgloss.Center,
		tabBar,
		"",
		content,
		"",
		statusBar,
	)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, block)
}

func (m AppModel) renderTabs() string {
	tabs := []struct {
		key  string
		name string
		tab  Tab
	}{
		{"s", "secrets", TabSecrets},
		{"t", "teams", TabTeams},
		{"a", "account", TabAccount},
	}

	tabWidth := ContentWidth / len(tabs)

	activeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Width(tabWidth).
		Align(lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("62"))

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(tabWidth).
		Align(lipgloss.Center).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("237"))

	var rendered []string
	for _, tab := range tabs {
		label := fmt.Sprintf("%s %s", tab.key, tab.name)
		if tab.tab == m.tab {
			rendered = append(rendered, activeStyle.Render(label))
		} else {
			rendered = append(rendered, inactiveStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}

func (m AppModel) renderContent() string {
	content := ""
	switch m.tab {
	case TabSecrets:
		if !m.loggedIn {
			content = Subtle.Render("login to view secrets")
		} else {
			content = m.secrets.View()
		}
	case TabTeams:
		if !m.loggedIn {
			content = Subtle.Render("login to view teams")
		} else {
			content = m.teams.View()
		}
	case TabAccount:
		content = m.account.View()
	}

	return lipgloss.NewStyle().
		Width(ContentWidth).
		Render(content)
}

func (m AppModel) renderStatusBar() string {
	divider := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Render(strings.Repeat("─", ContentWidth))

	userInfo := ""
	if m.user != nil {
		userInfo = fmt.Sprintf("%s (%s)", m.user.Email, m.user.Role)
	} else {
		userInfo = "not logged in"
	}
	serverInfo := Subtle.Render(m.client.Config.ServerURL)

	helpItems := []string{
		Accent.Render("s") + Subtle.Render("/") + Accent.Render("t") + Subtle.Render("/") + Accent.Render("a") + Subtle.Render(" tabs"),
		Accent.Render("j") + Subtle.Render("/") + Accent.Render("k") + Subtle.Render(" navigate"),
		Accent.Render("q") + Subtle.Render(" quit"),
	}
	helpLine := strings.Join(helpItems, "    ")

	status := lipgloss.NewStyle().
		Width(ContentWidth).
		Align(lipgloss.Center).
		Render(serverInfo + Subtle.Render("  |  ") + Subtle.Render(userInfo))

	help := lipgloss.NewStyle().
		Width(ContentWidth).
		Align(lipgloss.Center).
		Render(helpLine)

	return lipgloss.JoinVertical(lipgloss.Center, divider, "", status, "", help)
}
