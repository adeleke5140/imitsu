package ui

import (
	"fmt"
	"strings"

	"github.com/adeleke5140/imitsu/tui/api"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type LoginModel struct {
	client   *api.Client
	inputs   []textinput.Model
	focused  int
	err      string
	loading  bool
	isRegister bool
	// register has 3 fields: email, name, password; login has 2: email, password
}

type loginSuccessMsg struct {
	user api.User
}

type loginErrMsg struct {
	err error
}

func NewLoginModel(client *api.Client) LoginModel {
	emailInput := textinput.New()
	emailInput.Placeholder = "email@example.com"
	emailInput.Focus()
	emailInput.CharLimit = 64
	emailInput.Width = 40

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password (min 12 chars)"
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '*'
	passwordInput.CharLimit = 128
	passwordInput.Width = 40

	return LoginModel{
		client:  client,
		inputs:  []textinput.Model{emailInput, passwordInput},
		focused: 0,
	}
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (LoginModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.isRegister {
				// In register mode, toggle between 3 fields
				m.focused = (m.focused + 1) % len(m.inputs)
			} else {
				m.focused = (m.focused + 1) % 2
			}
			return m, m.updateFocus()

		case "shift+tab":
			if m.isRegister {
				m.focused = (m.focused - 1 + len(m.inputs)) % len(m.inputs)
			} else {
				m.focused = (m.focused - 1 + 2) % 2
			}
			return m, m.updateFocus()

		case "ctrl+r":
			m.isRegister = !m.isRegister
			if m.isRegister && len(m.inputs) == 2 {
				nameInput := textinput.New()
				nameInput.Placeholder = "your name"
				nameInput.CharLimit = 64
				nameInput.Width = 40
				// Insert name input between email and password
				m.inputs = []textinput.Model{m.inputs[0], nameInput, m.inputs[1]}
			} else if !m.isRegister && len(m.inputs) == 3 {
				m.inputs = []textinput.Model{m.inputs[0], m.inputs[2]}
			}
			m.focused = 0
			m.err = ""
			return m, m.updateFocus()

		case "enter":
			if m.loading {
				return m, nil
			}
			return m, m.submit()
		}

	case loginSuccessMsg:
		m.loading = false
		return m, nil

	case loginErrMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil
	}

	// Update the focused input
	var cmd tea.Cmd
	if m.focused < len(m.inputs) {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	}
	return m, cmd
}

func (m LoginModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focused {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m LoginModel) submit() tea.Cmd {
	email := m.inputs[0].Value()

	if m.isRegister {
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
			// Auto-login after register
			resp, err := m.client.Login(email, password)
			if err != nil {
				return loginErrMsg{err}
			}
			return loginSuccessMsg{resp.User}
		}
	}

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

func (m LoginModel) View() string {
	var b strings.Builder

	mode := "Login"
	if m.isRegister {
		mode = "Register"
	}

	b.WriteString(Title.Render(fmt.Sprintf("imitsu - %s", mode)))
	b.WriteString("\n\n")

	for i, input := range m.inputs {
		if m.isRegister && i > 2 {
			break
		}
		if !m.isRegister && i > 1 {
			break
		}

		label := ""
		switch {
		case i == 0:
			label = "Email"
		case i == 1 && m.isRegister:
			label = "Name"
		case i == 1 && !m.isRegister:
			label = "Password"
		case i == 2:
			label = "Password"
		}

		if i == m.focused {
			b.WriteString(Highlight.Render(label))
		} else {
			b.WriteString(Subtle.Render(label))
		}
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n\n")
	}

	if m.err != "" {
		b.WriteString(ErrorText.Render(m.err))
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString(Subtle.Render("connecting..."))
		b.WriteString("\n")
	}

	helpItems := []string{
		Accent.Render("enter") + Subtle.Render(" submit"),
		Accent.Render("tab") + Subtle.Render(" next field"),
		Accent.Render("ctrl+r") + Subtle.Render(" toggle register/login"),
		Accent.Render("q") + Subtle.Render(" quit"),
	}
	b.WriteString(HelpStyle.Render(strings.Join(helpItems, "    ")))

	return BoxStyle.Width(ContentWidth - 4).Render(b.String())
}
