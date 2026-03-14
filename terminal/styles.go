package main

import "github.com/charmbracelet/lipgloss"

var (
	styleBorder    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	styleTitle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styleRoom      = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleActiveTab = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Underline(true)
	styleTab       = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleDM        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	styleSystem    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)
	styleError     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleNotif     = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	styleOnline    = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleOffline   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleName      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	stylePrompt    = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
)
