package ui

import (
	"fmt"
	"strings"

	"github.com/adeleke5140/imitsu/tui/api"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type AccountPane int

const (
	PaneLogin AccountPane = iota
	PaneRegister
	PaneProfile
	PaneServer
)

type AccountFocus int

const (
	FocusSidebar AccountFocus = iota
	FocusForm
)

type AccountModel struct {
	client *api.Client
	user   *api.User

	pane   AccountPane
	focus  AccountFocus
	cursor int // sidebar cursor

	// form inputs
	inputs  []textinput.Model
	formIdx int

	err     string
	loading bool
	message string
}

type loginSuccessMsg struct {
	user api.User
}

type loginErrMsg struct {
	err error
}

func NewAccountModel(client *api.Client) AccountModel {
	m := AccountModel{
		client: client,
		pane:   PaneLogin,
		focus:  FocusSidebar,
	}
	m.inputs = m.makeLoginInputs()
	return m
}

func (m *AccountModel) SetUser(user *api.User) {
	m.user = user
	if user != nil {
		m.pane = PaneProfile
		m.cursor = 2
	}
}

func (m AccountModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AccountModel) sidebarItems() []struct {
	label string
	pane  AccountPane
} {
	items := []struct {
		label string
		pane  AccountPane
	}{
		{"login", PaneLogin},
		{"register", PaneRegister},
		{"server", PaneServer},
	}
	if m.user != nil {
		items = append(items, struct {
			label string
			pane  AccountPane
		}{"profile", PaneProfile})
	}
	return items
}

func (m AccountModel) Update(msg tea.Msg) (AccountModel, tea.Cmd) {
	switch msg := msg.(type) {
	case loginSuccessMsg:
		m.loading = false
		m.user = &msg.user
		m.err = ""
		m.message = fmt.Sprintf("logged in as %s", msg.user.Email)
		m.pane = PaneProfile
		m.cursor = 2
		return m, nil

	case loginErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		m.message = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Update text inputs
	if m.focus == FocusForm && m.formIdx < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.formIdx], cmd = m.inputs[m.formIdx].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AccountModel) handleKey(msg tea.KeyMsg) (AccountModel, tea.Cmd) {
	switch m.focus {
	case FocusSidebar:
		return m.handleSidebarKey(msg)
	case FocusForm:
		return m.handleFormKey(msg)
	}
	return m, nil
}

func (m AccountModel) handleSidebarKey(msg tea.KeyMsg) (AccountModel, tea.Cmd) {
	items := m.sidebarItems()
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case "enter", "l", "right":
		m.pane = items[m.cursor].pane
		m.err = ""
		m.message = ""
		switch m.pane {
		case PaneLogin:
			m.inputs = m.makeLoginInputs()
		case PaneRegister:
			m.inputs = m.makeRegisterInputs()
		case PaneServer:
			m.inputs = m.makeServerInputs()
		case PaneProfile:
			return m, nil
		}
		m.focus = FocusForm
		m.formIdx = 0
		return m, m.inputs[0].Focus()
	}
	return m, nil
}

func (m AccountModel) handleFormKey(msg tea.KeyMsg) (AccountModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "left", "h":
		// Only go back to sidebar on esc or left/h if input is empty or it's left/h
		if msg.String() == "esc" {
			m.focus = FocusSidebar
			for i := range m.inputs {
				m.inputs[i].Blur()
			}
			return m, nil
		}
	case "tab":
		m.formIdx = (m.formIdx + 1) % len(m.inputs)
		return m, m.focusFormInput()
	case "shift+tab":
		m.formIdx = (m.formIdx - 1 + len(m.inputs)) % len(m.inputs)
		return m, m.focusFormInput()
	case "enter":
		if m.loading {
			return m, nil
		}
		if m.pane == PaneServer {
			url := m.inputs[0].Value()
			if url == "" {
				m.err = "server URL is required"
				return m, nil
			}
			m.client.Config.ServerURL = strings.TrimRight(url, "/")
			m.client.SaveConfig()
			m.message = fmt.Sprintf("server set to %s", m.client.Config.ServerURL)
			m.err = ""
			return m, nil
		}
		return m, m.submit()
	}

	var cmd tea.Cmd
	if m.formIdx < len(m.inputs) {
		m.inputs[m.formIdx], cmd = m.inputs[m.formIdx].Update(msg)
	}
	return m, cmd
}

func (m AccountModel) focusFormInput() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.formIdx {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m AccountModel) submit() tea.Cmd {
	if m.pane == PaneLogin {
		email := m.inputs[0].Value()
		password := m.inputs[1].Value()
		if email == "" || password == "" {
			m.err = "email and password are required"
			return nil
		}
		m.loading = true
		return func() tea.Msg {
			resp, err := m.client.Login(email, password)
			if err != nil {
				return loginErrMsg{err}
			}
			return loginSuccessMsg{resp.User}
		}
	}

	if m.pane == PaneRegister {
		email := m.inputs[0].Value()
		name := m.inputs[1].Value()
		password := m.inputs[2].Value()
		if email == "" || name == "" || password == "" {
			m.err = "all fields are required"
			return nil
		}
		m.loading = true
		return func() tea.Msg {
			_, err := m.client.Register(email, name, password)
			if err != nil {
				return loginErrMsg{err}
			}
			resp, err := m.client.Login(email, password)
			if err != nil {
				return loginErrMsg{err}
			}
			return loginSuccessMsg{resp.User}
		}
	}

	return nil
}

func (m AccountModel) makeLoginInputs() []textinput.Model {
	email := textinput.New()
	email.Placeholder = "email@example.com"
	email.CharLimit = 64
	email.Width = 36

	password := textinput.New()
	password.Placeholder = "password"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'
	password.CharLimit = 128
	password.Width = 36

	return []textinput.Model{email, password}
}

func (m AccountModel) makeRegisterInputs() []textinput.Model {
	email := textinput.New()
	email.Placeholder = "email@example.com"
	email.CharLimit = 64
	email.Width = 36

	name := textinput.New()
	name.Placeholder = "your name"
	name.CharLimit = 64
	name.Width = 36

	password := textinput.New()
	password.Placeholder = "password (min 12 chars)"
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'
	password.CharLimit = 128
	password.Width = 36

	return []textinput.Model{email, name, password}
}

func (m AccountModel) makeServerInputs() []textinput.Model {
	url := textinput.New()
	url.Placeholder = "http://localhost:3100"
	url.SetValue(m.client.Config.ServerURL)
	url.CharLimit = 256
	url.Width = 36

	return []textinput.Model{url}
}

func (m AccountModel) View() string {
	sidebar := m.viewSidebar()
	content := m.viewContent()

	sidebarWidth := 20
	contentWidth := ContentWidth - sidebarWidth - 3 // 3 for separator

	sidebarStyled := lipgloss.NewStyle().
		Width(sidebarWidth).
		Render(sidebar)

	separator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Render(strings.Repeat("│\n", max(lipgloss.Height(sidebarStyled), lipgloss.Height(content))))

	contentStyled := lipgloss.NewStyle().
		Width(contentWidth).
		PaddingLeft(2).
		Render(content)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarStyled, separator, contentStyled)
}

func (m AccountModel) viewSidebar() string {
	var b strings.Builder
	items := m.sidebarItems()

	for i, item := range items {
		if i == m.cursor && m.focus == FocusSidebar {
			b.WriteString(Highlight.Bold(true).Render(item.label))
		} else if items[i].pane == m.pane {
			b.WriteString(Highlight.Render(item.label))
		} else {
			b.WriteString(Subtle.Render(item.label))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m AccountModel) viewContent() string {
	switch m.pane {
	case PaneLogin:
		return m.viewForm("Login", []string{"Email", "Password"})
	case PaneRegister:
		return m.viewForm("Register", []string{"Email", "Name", "Password"})
	case PaneServer:
		return m.viewForm("Server", []string{"URL"})
	case PaneProfile:
		return m.viewProfile()
	}
	return ""
}

func (m AccountModel) viewForm(title string, labels []string) string {
	var b strings.Builder

	b.WriteString(Title.Render(title))
	b.WriteString("\n\n")

	for i, input := range m.inputs {
		if i >= len(labels) {
			break
		}
		if m.focus == FocusForm && i == m.formIdx {
			b.WriteString(Highlight.Render(labels[i]))
		} else {
			b.WriteString(Subtle.Render(labels[i]))
		}
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(ErrorText.Render(m.err))
		b.WriteString("\n")
	}

	if m.message != "" {
		b.WriteString(Success.Render(m.message))
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString(Subtle.Render("connecting..."))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render(
		Accent.Render("enter") + Subtle.Render(" submit") + "    " +
			Accent.Render("tab") + Subtle.Render(" next field") + "    " +
			Accent.Render("esc") + Subtle.Render(" sidebar"),
	))

	return b.String()
}

func (m AccountModel) viewProfile() string {
	var b strings.Builder

	b.WriteString(Title.Render("Profile"))
	b.WriteString("\n\n")

	if m.user == nil {
		b.WriteString(Subtle.Render("not logged in"))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Email:"), m.user.Email))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Name:"), m.user.Name))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Role:"), m.user.Role))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Joined:"), m.user.CreatedAt))

	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(Success.Render(m.message))
	}

	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render(Accent.Render("ctrl+l") + Subtle.Render(" logout")))

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
