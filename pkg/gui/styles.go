package gui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	primaryColor   = lipgloss.Color("#7D56F4")
	secondaryColor = lipgloss.Color("#6C71C4")
	accentColor    = lipgloss.Color("#F25D94")
	successColor   = lipgloss.Color("#73D216")
	warningColor   = lipgloss.Color("#F5A623")
	errorColor     = lipgloss.Color("#FF5555")
	dimColor       = lipgloss.Color("#666666")
	textColor      = lipgloss.Color("#FAFAFA")
	subtleColor    = lipgloss.Color("#999999")
	borderColor    = lipgloss.Color("#444444")
	activeBorder   = lipgloss.Color("#7D56F4")
)

// Panel styles
var (
	// Active panel border
	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(activeBorder)

	// Inactive panel border
	inactivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(borderColor)

	// Panel title style (active)
	activeTitleStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Panel title style (inactive)
	inactiveTitleStyle = lipgloss.NewStyle().
				Foreground(dimColor)

	// Selected item in list
	selectedItemStyle = lipgloss.NewStyle().
				Foreground(textColor).
				Background(primaryColor).
				Bold(true).
				PaddingLeft(1).
				PaddingRight(1)

	// Normal item in list
	normalItemStyle = lipgloss.NewStyle().
			Foreground(textColor).
			PaddingLeft(1)

	// Dimmed item
	dimItemStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			PaddingLeft(1)

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Background(lipgloss.Color("#1A1A2E"))

	// Keybinding style
	keyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	// Keybinding description style
	keyDescStyle = lipgloss.NewStyle().
			Foreground(subtleColor)

	// Outdated/update available marker
	outdatedStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// Pinned marker
	pinnedStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	// Service status styles
	runningStyle = lipgloss.NewStyle().
			Foreground(successColor)
	stoppedStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
			Foreground(textColor).
			Background(primaryColor).
			Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(subtleColor).
				Padding(0, 1)

	// Detail panel styles
	detailKeyStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Width(14)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(textColor)

	// Command log styles
	cmdLogPrefix = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	cmdLogSuccess = lipgloss.NewStyle().
			Foreground(successColor)

	cmdLogError = lipgloss.NewStyle().
			Foreground(errorColor)

	// Popup/dialog overlay style
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Width(50)

	// Search input style
	searchPromptStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	// Help title
	helpTitleStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)
)
