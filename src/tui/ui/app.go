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
)

type AppModel struct {
	client *api.Client
	tab    Tab
	user   *api.User
	width  int
	height int

	login   LoginModel
	secrets SecretsModel
	teams   TeamsModel

	loggedIn bool
}

func NewApp() AppModel {
	client := api.NewClient()
	m := AppModel{
		client: client,
		login:  NewLoginModel(client),
	}

	if client.IsLoggedIn() {
		m.loggedIn = true
		m.secrets = NewSecretsModel(client)
		m.teams = NewTeamsModel(client)
	}

	return m
}

func (m AppModel) Init() tea.Cmd {
	if m.loggedIn {
		return tea.Batch(
			m.secrets.Init(),
			m.teams.Init(),
			m.fetchUser(),
		)
	}
	return m.login.Init()
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
			if m.loggedIn && !m.isTyping() {
				m.tab = TabSecrets
				return m, nil
			}
		case "t":
			if m.loggedIn && !m.isTyping() {
				m.tab = TabTeams
				return m, nil
			}
		case "ctrl+l":
			if m.loggedIn {
				m.client.Logout()
				m.loggedIn = false
				m.user = nil
				m.login = NewLoginModel(m.client)
				return m, m.login.Init()
			}
		}

	case loginSuccessMsg:
		m.loggedIn = true
		m.user = &msg.user
		m.secrets = NewSecretsModel(m.client)
		m.teams = NewTeamsModel(m.client)
		return m, tea.Batch(m.secrets.Init(), m.teams.Init())

	case loginErrMsg:
		if m.loggedIn {
			m.loggedIn = false
			m.user = nil
			m.login = NewLoginModel(m.client)
			m.login.err = "session expired, please login again"
			return m, m.login.Init()
		}
	}

	if !m.loggedIn {
		var cmd tea.Cmd
		m.login, cmd = m.login.Update(msg)
		return m, cmd
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
		}
		return m, nil
	}

	// Data messages go to both models so background fetches aren't lost
	var cmd1, cmd2 tea.Cmd
	m.secrets, cmd1 = m.secrets.Update(msg)
	m.teams, cmd2 = m.teams.Update(msg)
	return m, tea.Batch(cmd1, cmd2)
}

func (m AppModel) isTyping() bool {
	if !m.loggedIn {
		return true
	}
	if m.tab == TabSecrets && m.secrets.view == SecretCreate {
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

	if !m.loggedIn {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, m.login.View())
	}

	// Build all three parts as one unified block, then center the whole thing
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
	}{
		{"s", "secrets"},
		{"t", "teams"},
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
	for i, tab := range tabs {
		label := fmt.Sprintf("%s %s", tab.key, tab.name)
		if Tab(i) == m.tab {
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
		content = m.secrets.View()
	case TabTeams:
		content = m.teams.View()
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
	}
	serverInfo := Subtle.Render(m.client.Config.ServerURL)

	helpItems := []string{
		Accent.Render("s") + Subtle.Render("/") + Accent.Render("t") + Subtle.Render(" tabs"),
		Accent.Render("j") + Subtle.Render("/") + Accent.Render("k") + Subtle.Render(" navigate"),
		Accent.Render("ctrl+l") + Subtle.Render(" logout"),
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
