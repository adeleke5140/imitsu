package ui

import (
	"fmt"
	"strings"

	"github.com/adeleke5140/imitsu/tui/api"
	tea "github.com/charmbracelet/bubbletea"
)

type TeamsView int

const (
	TeamsList TeamsView = iota
	TeamDetail
)

type TeamsModel struct {
	client  *api.Client
	teams   []api.Team
	cursor  int
	view    TeamsView
	err     string
	loading bool

	// detail view
	selected     *api.Team
	members      []api.TeamMember
	memberCursor int

	// prefetched team details: team ID -> members
	teamMembers map[string][]api.TeamMember
}

type teamsLoadedMsg struct {
	teams []api.Team
}

type teamDetailLoadedMsg struct {
	team    api.Team
	members []api.TeamMember
}

type allTeamDetailsFetchedMsg struct {
	teamMembers map[string][]api.TeamMember
}

type teamsErrMsg struct {
	err error
}

func NewTeamsModel(client *api.Client) TeamsModel {
	return TeamsModel{
		client: client,
	}
}

func (m TeamsModel) Init() tea.Cmd {
	return m.loadTeams()
}

func (m TeamsModel) loadTeams() tea.Cmd {
	return func() tea.Msg {
		teams, err := m.client.ListTeams()
		if err != nil {
			return teamsErrMsg{err}
		}
		return teamsLoadedMsg{teams}
	}
}

func (m TeamsModel) loadTeamDetail(id string) tea.Cmd {
	return func() tea.Msg {
		detail, err := m.client.GetTeamDetails(id)
		if err != nil {
			return teamsErrMsg{err}
		}
		return teamDetailLoadedMsg{detail.Team, detail.Members}
	}
}

func (m TeamsModel) prefetchAllDetails() tea.Cmd {
	teams := m.teams
	return func() tea.Msg {
		result := make(map[string][]api.TeamMember, len(teams))
		for _, t := range teams {
			detail, err := m.client.GetTeamDetails(t.ID)
			if err != nil {
				continue
			}
			result[t.ID] = detail.Members
		}
		return allTeamDetailsFetchedMsg{result}
	}
}

func (m TeamsModel) Update(msg tea.Msg) (TeamsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case teamsLoadedMsg:
		m.teams = msg.teams
		m.loading = false
		m.err = ""
		return m, m.prefetchAllDetails()

	case allTeamDetailsFetchedMsg:
		m.teamMembers = msg.teamMembers
		return m, nil

	case teamDetailLoadedMsg:
		m.selected = &msg.team
		m.members = msg.members
		m.view = TeamDetail
		m.memberCursor = 0
		m.loading = false
		m.err = ""
		if m.teamMembers == nil {
			m.teamMembers = make(map[string][]api.TeamMember)
		}
		m.teamMembers[msg.team.ID] = msg.members
		return m, nil

	case teamsErrMsg:
		m.err = msg.err.Error()
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case TeamsList:
			return m.handleListKey(msg)
		case TeamDetail:
			return m.handleDetailKey(msg)
		}
	}

	return m, nil
}

func (m TeamsModel) handleListKey(msg tea.KeyMsg) (TeamsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.teams)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.teams) > 0 {
			team := m.teams[m.cursor]
			if members, ok := m.teamMembers[team.ID]; ok {
				m.selected = &team
				m.members = members
				m.view = TeamDetail
				m.memberCursor = 0
				return m, nil
			}
			m.loading = true
			return m, m.loadTeamDetail(team.ID)
		}
	case "r":
		m.loading = true
		return m, m.loadTeams()
	}
	return m, nil
}

func (m TeamsModel) handleDetailKey(msg tea.KeyMsg) (TeamsModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.view = TeamsList
		m.selected = nil
		m.members = nil
		return m, nil
	case "up", "k":
		if m.memberCursor > 0 {
			m.memberCursor--
		}
	case "down", "j":
		if m.memberCursor < len(m.members)-1 {
			m.memberCursor++
		}
	case "r":
		if m.selected != nil {
			m.loading = true
			return m, m.loadTeamDetail(m.selected.ID)
		}
	}
	return m, nil
}

func (m TeamsModel) View() string {
	switch m.view {
	case TeamDetail:
		return m.viewDetail()
	default:
		return m.viewList()
	}
}

func (m TeamsModel) viewList() string {
	var b strings.Builder

	b.WriteString(Title.Render("Teams"))
	b.WriteString("\n")

	if m.loading {
		b.WriteString(Subtle.Render("loading..."))
		return b.String()
	}

	if m.err != "" {
		b.WriteString(ErrorText.Render(m.err))
		b.WriteString("\n")
	}

	if len(m.teams) == 0 {
		b.WriteString(Subtle.Render("no teams yet."))
	} else {
		header := fmt.Sprintf("  %-23s %-10s %s", "TEAM", "MEMBERS", "CREATED")
		b.WriteString(Subtle.Render(header))
		b.WriteString("\n")
		b.WriteString(Subtle.Render("  " + strings.Repeat("─", 50)))
		b.WriteString("\n")

		for i, t := range m.teams {
			line := fmt.Sprintf("%-23s %-10d %s", t.Name, t.MemberCount, t.CreatedAt)
			if i == m.cursor {
				b.WriteString(SelectedItem.Render("> " + line))
			} else {
				b.WriteString(ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(HelpStyle.Render("\nj/k: navigate | enter: view members | r: refresh"))

	return b.String()
}

func (m TeamsModel) viewDetail() string {
	var b strings.Builder

	if m.selected == nil {
		return "no team selected"
	}

	if m.loading {
		b.WriteString(Subtle.Render("loading..."))
		return b.String()
	}

	t := m.selected
	b.WriteString(Title.Render(fmt.Sprintf("Team: %s", t.Name)))
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("%s  %d\n", Subtle.Render("Members:"), t.MemberCount))
	b.WriteString(fmt.Sprintf("%s  %s\n", Subtle.Render("Created:"), t.CreatedAt))
	b.WriteString("\n")

	if len(m.members) == 0 {
		b.WriteString(Subtle.Render("no members."))
	} else {
		header := fmt.Sprintf("  %-28s %-18s %-10s %s", "EMAIL", "NAME", "ROLE", "JOINED")
		b.WriteString(Subtle.Render(header))
		b.WriteString("\n")
		b.WriteString(Subtle.Render("  " + strings.Repeat("─", 70)))
		b.WriteString("\n")

		for i, member := range m.members {
			line := fmt.Sprintf("%-28s %-18s %-10s %s", member.Email, member.Name, member.Role, member.JoinedAt)
			if i == m.memberCursor {
				b.WriteString(SelectedItem.Render("> " + line))
			} else {
				b.WriteString(ListItem.Render(line))
			}
			b.WriteString("\n")
		}
	}

	if m.err != "" {
		b.WriteString("\n")
		b.WriteString(ErrorText.Render(m.err))
	}

	b.WriteString(HelpStyle.Render("\nj/k: navigate | r: refresh | esc: back"))

	return b.String()
}
