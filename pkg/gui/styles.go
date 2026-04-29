package gui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// Active border: green bold
	activeBorderColor = lipgloss.Color("2") // ANSI green

	// Inactive border: white/default
	inactiveBorderColor = lipgloss.Color("7") // ANSI white

	// Options text: blue
	optionsColor = lipgloss.Color("4") // ANSI blue

	// Selected line bg: blue
	selectedBgColor = lipgloss.Color("4") // ANSI blue

	// Cherry-pick / accent: cyan
	accentColor = lipgloss.Color("6") // ANSI cyan

	// Unstaged changes / error: red
	errorColor = lipgloss.Color("1") // ANSI red

	// Warning / outdated: yellow
	warningColor = lipgloss.Color("3") // ANSI yellow

	// Success: green
	successColor = lipgloss.Color("2") // ANSI green

	// Dimmed text
	dimColor = lipgloss.Color("8") // ANSI bright black (gray)

	// Default foreground
	defaultFgColor = lipgloss.Color("7") // ANSI white
)

// Panel styles
var (
	// Active panel border
	activePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(activeBorderColor)

	// Inactive panel border
	inactivePanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(inactiveBorderColor)

	// Panel title style (active) — green bold
	activeTitleStyle = lipgloss.NewStyle().
				Foreground(activeBorderColor).
				Bold(true)

	// Panel title style (inactive) — default
	inactiveTitleStyle = lipgloss.NewStyle().
				Foreground(defaultFgColor)

	// Selected item in list — blue background
	selectedItemStyle = lipgloss.NewStyle().
				Background(selectedBgColor).
				Bold(true)

	// Normal item in list
	normalItemStyle = lipgloss.NewStyle().
			Foreground(defaultFgColor)

	// Dimmed item
	dimItemStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// Status bar style — options in blue
	statusBarStyle = lipgloss.NewStyle().
			Foreground(optionsColor)

	// Keybinding style — blue
	keyStyle = lipgloss.NewStyle().
			Foreground(optionsColor).
			Bold(true)

	// Keybinding description style
	keyDescStyle = lipgloss.NewStyle().
			Foreground(optionsColor)

	// Outdated/update marker — yellow
	outdatedStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	// Pinned marker — cyan
	pinnedStyle = lipgloss.NewStyle().
			Foreground(accentColor)

	// Service status styles
	runningStyle = lipgloss.NewStyle().
			Foreground(successColor)
	stoppedStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	// Tab styles
	activeTabStyle = lipgloss.NewStyle().
			Foreground(activeBorderColor).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(defaultFgColor)

	// Detail panel styles
	detailKeyStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true).
			Width(14)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(defaultFgColor)

	// Command log styles
	cmdLogPrefix = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	cmdLogSuccess = lipgloss.NewStyle().
			Foreground(successColor)

	cmdLogError = lipgloss.NewStyle().
			Foreground(errorColor)

	// Search input style
	searchPromptStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	// Help title
	helpTitleStyle = lipgloss.NewStyle().
			Foreground(activeBorderColor).
			Bold(true)
)
