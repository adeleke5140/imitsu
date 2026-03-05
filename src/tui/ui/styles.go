package ui

import "github.com/charmbracelet/lipgloss"

const ContentWidth = 76

var (
	Subtle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	Highlight = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	Special   = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	ErrorText = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	Success   = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	Warning   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	Accent    = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		MarginBottom(1)

	StatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	ListItem = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItem = lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(lipgloss.Color("212")).
			Bold(true)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
