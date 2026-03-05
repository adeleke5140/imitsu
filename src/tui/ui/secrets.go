package ui

import (
	"fmt"
	"strings"

	"github.com/adeleke5140/imitsu/tui/api"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type SecretsView int

const (
	SecretsList SecretsView = iota
	SecretDetail
	SecretCreate
)

type SecretsModel struct {
	client   *api.Client
	secrets  []api.Secret
	cursor   int
	view     SecretsView
	err      string
	loading  bool
	selected *api.Secret // currently viewed secret

	// create form
	createInputs []textinput.Model
	createFocus  int

	// confirm delete
	confirmDelete bool
}

type secretsLoadedMsg struct {
	secrets []api.Secret
}

type secretDetailMsg struct {
	secret api.Secret
}

type secretCreatedMsg struct{}
type secretDeletedMsg struct{}

type secretErrMsg struct {
	err error
}

func NewSecretsModel(client *api.Client) SecretsModel {
	return SecretsModel{
		client: client,
		view:   SecretsList,
	}
}

func (m SecretsModel) Init() tea.Cmd {
	return m.loadSecrets()
}

func (m SecretsModel) loadSecrets() tea.Cmd {
	m.loading = true
	return func() tea.Msg {
		secrets, err := m.client.ListSecrets()
		if err != nil {
			return secretErrMsg{err}
		}
		return secretsLoadedMsg{secrets}
	}
}

func (m SecretsModel) Update(msg tea.Msg) (SecretsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case secretsLoadedMsg:
		m.secrets = msg.secrets
		m.loading = false
		m.err = ""
		return m, nil

	case secretDetailMsg:
		m.selected = &msg.secret
		m.view = SecretDetail
		m.loading = false
		return m, nil

	case secretCreatedMsg:
		m.view = SecretsList
		m.loading = false
		return m, m.loadSecrets()

	case secretDeletedMsg:
		m.view = SecretsList
		m.selected = nil
		m.confirmDelete = false
		m.loading = false
		return m, m.loadSecrets()

	case secretErrMsg:
		m.err = msg.err.Error()
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Update text inputs if in create view
	if m.view == SecretCreate && m.createFocus < len(m.createInputs) {
		var cmd tea.Cmd
		m.createInputs[m.createFocus], cmd = m.createInputs[m.createFocus].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m SecretsModel) handleKey(msg tea.KeyMsg) (SecretsModel, tea.Cmd) {
	switch m.view {
	case SecretsList:
		return m.handleListKey(msg)
	case SecretDetail:
		return m.handleDetailKey(msg)
	case SecretCreate:
		return m.handleCreateKey(msg)
	}
	return m, nil
}

func (m SecretsModel) handleListKey(msg tea.KeyMsg) (SecretsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.secrets)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.secrets) > 0 {
			m.loading = true
			id := m.secrets[m.cursor].ID
			return m, func() tea.Msg {
				secret, err := m.client.GetSecret(id)
				if err != nil {
					return secretErrMsg{err}
				}
				return secretDetailMsg{*secret}
			}
		}
	case "n":
		m.view = SecretCreate
		m.createInputs = makeCreateInputs()
		m.createFocus = 0
		return m, m.createInputs[0].Focus()
	case "r":
		return m, m.loadSecrets()
	}
	return m, nil
}

func (m SecretsModel) handleDetailKey(msg tea.KeyMsg) (SecretsModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		if m.confirmDelete {
			m.confirmDelete = false
			return m, nil
		}
		m.view = SecretsList
		m.selected = nil
		return m, nil
	case "d":
		if !m.confirmDelete {
			m.confirmDelete = true
		}
		return m, nil
	case "y":
		if m.confirmDelete && m.selected != nil {
			m.loading = true
			id := m.selected.ID
			return m, func() tea.Msg {
				err := m.client.DeleteSecret(id)
				if err != nil {
					return secretErrMsg{err}
				}
				return secretDeletedMsg{}
			}
		}
	case "n":
		if m.confirmDelete {
			m.confirmDelete = false
		}
	}
	return m, nil
}

func (m SecretsModel) handleCreateKey(msg tea.KeyMsg) (SecretsModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.view = SecretsList
		return m, nil
	case "tab":
		m.createFocus = (m.createFocus + 1) % len(m.createInputs)
		return m, m.focusCreateInput()
	case "shift+tab":
		m.createFocus = (m.createFocus - 1 + len(m.createInputs)) % len(m.createInputs)
		return m, m.focusCreateInput()
	case "enter":
		name := m.createInputs[0].Value()
		value := m.createInputs[1].Value()
		category := m.createInputs[2].Value()
		if name == "" || value == "" {
			m.err = "name and value are required"
			return m, nil
		}
		if category == "" {
			category = "general"
		}
		m.loading = true
		return m, func() tea.Msg {
			_, err := m.client.CreateSecret(name, value, category)
			if err != nil {
				return secretErrMsg{err}
			}
			return secretCreatedMsg{}
		}
	}

	var cmd tea.Cmd
	m.createInputs[m.createFocus], cmd = m.createInputs[m.createFocus].Update(msg)
	return m, cmd
}

func (m SecretsModel) focusCreateInput() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.createInputs))
	for i := range m.createInputs {
		if i == m.createFocus {
			cmds[i] = m.createInputs[i].Focus()
		} else {
			m.createInputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func makeCreateInputs() []textinput.Model {
	nameInput := textinput.New()
	nameInput.Placeholder = "SECRET_NAME"
	nameInput.CharLimit = 128
	nameInput.Width = 40

	valueInput := textinput.New()
	valueInput.Placeholder = "secret value"
	valueInput.EchoMode = textinput.EchoPassword
	valueInput.EchoCharacter = '*'
	valueInput.CharLimit = 1024
	valueInput.Width = 40

	categoryInput := textinput.New()
	categoryInput.Placeholder = "general"
	categoryInput.CharLimit = 64
	categoryInput.Width = 40

	return []textinput.Model{nameInput, valueInput, categoryInput}
}

func (m SecretsModel) View() string {
	switch m.view {
	case SecretDetail:
		return m.viewDetail()
	case SecretCreate:
		return m.viewCreate()
	default:
		return m.viewList()
	}
}

func (m SecretsModel) viewList() string {
	var b strings.Builder

	b.WriteString(Title.Render("Secrets"))
	b.WriteString("\n")

	if m.loading {
		b.WriteString(Subtle.Render("loading..."))
		return b.String()
	}

	if m.err != "" {
		b.WriteString(ErrorText.Render(m.err))
		b.WriteString("\n")
	}

	if len(m.secrets) == 0 {
		b.WriteString(Subtle.Render("no secrets yet. press n to create one."))
	} else {
		header := fmt.Sprintf("  %-28s %-12s %-5s %s", "NAME", "CATEGORY", "VER", "UPDATED")
		b.WriteString(Subtle.Render(header))
		b.WriteString("\n")
		b.WriteString(Subtle.Render("  " + strings.Repeat("─", 65)))
		b.WriteString("\n")

		for i, s := range m.secrets {
			line := fmt.Sprintf("%-28s %-12s v%-4d %s", s.Name, s.Category, s.Version, s.UpdatedAt)
			if i == m.cursor {
				b.WriteString(SelectedItem.Render("> " + line))
			} else {
				b.WriteString(ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(HelpStyle.Render("\nj/k: navigate | enter: view | n: new | r: refresh | d: delete"))

	return b.String()
}

func (m SecretsModel) viewDetail() string {
	var b strings.Builder

	if m.selected == nil {
		return "no secret selected"
	}

	s := m.selected
	b.WriteString(Title.Render(fmt.Sprintf("Secret: %s", s.Name)))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Value:"), s.Value))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Category:"), s.Category))
	b.WriteString(fmt.Sprintf("%s  v%d\n", Subtle.Render("Version:"), s.Version))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Created:"), s.CreatedAt))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Updated:"), s.UpdatedAt))

	if m.confirmDelete {
		b.WriteString("\n")
		b.WriteString(Warning.Render("Delete this secret? (y/n)"))
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(ErrorText.Render(m.err))
	}

	b.WriteString(HelpStyle.Render("\nesc: back | d: delete"))

	return b.String()
}

func (m SecretsModel) viewCreate() string {
	var b strings.Builder

	b.WriteString(Title.Render("New Secret"))
	b.WriteString("\n\n")

	labels := []string{"Name", "Value", "Category"}
	for i, input := range m.createInputs {
		if i == m.createFocus {
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

	b.WriteString(HelpStyle.Render("enter: create | tab: next field | esc: cancel"))

	return b.String()
}
