package tui

import "github.com/charmbracelet/lipgloss"

var (
	appStyle = lipgloss.NewStyle().
			Padding(1, 2)
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("81"))
	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("238")).
			Padding(1, 2)
	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229"))
	activeRowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)
	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))
	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)
	treeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63"))
	checkboxStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
	checkedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)
	disabledStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
	badgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")).
			Padding(0, 1)
	searchBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230")).
				Background(lipgloss.Color("63")).
				Bold(true).
				Padding(0, 1)
	projectBadgeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)
