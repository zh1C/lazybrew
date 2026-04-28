package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// View implements tea.Model - renders the entire TUI.
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	// Calculate layout dimensions
	sideWidth := a.width * 30 / 100
	if sideWidth < 24 {
		sideWidth = 24
	}
	if sideWidth > 40 {
		sideWidth = 40
	}
	mainWidth := a.width - sideWidth
	bottomHeight := 1
	contentHeight := a.height - bottomHeight - 1 // -1 for title bar

	// Render title bar
	titleBar := a.renderTitleBar()

	// Render side panels
	sidePanel := a.renderSidePanel(sideWidth, contentHeight)

	// Render main area (detail + log)
	mainPanel := a.renderMainArea(mainWidth, contentHeight)

	// Compose the body
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidePanel, mainPanel)

	// Render bottom bar
	bottomBar := a.renderBottomBar()

	// Compose full view
	view := lipgloss.JoinVertical(lipgloss.Left, titleBar, body, bottomBar)

	// Render overlay if active
	if a.overlay != OverlayNone {
		view = a.renderOverlay(view)
	}

	return view
}

// --- Title Bar ---

func (a *App) renderTitleBar() string {
	title := lipgloss.NewStyle().
		Foreground(primaryColor).
		Bold(true).
		Render(" 🍺 lazybrew")

	loadingIndicator := ""
	if a.loading {
		loadingIndicator = lipgloss.NewStyle().
			Foreground(warningColor).
			Render(" ⏳ " + a.loadingMsg)
	}

	errIndicator := ""
	if a.errMsg != "" {
		errIndicator = lipgloss.NewStyle().
			Foreground(errorColor).
			Render(" ⚠ " + a.errMsg)
	}

	right := lipgloss.NewStyle().
		Foreground(dimColor).
		Render("? help  q quit ")

	left := title + loadingIndicator + errIndicator
	spacer := strings.Repeat(" ", max(0, a.width-lipgloss.Width(left)-lipgloss.Width(right)))

	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1A1A2E")).
		Width(a.width).
		Render(left + spacer + right)
}

// --- Side Panel ---

func (a *App) renderSidePanel(width, height int) string {
	innerWidth := width - 2 // border takes 2 chars

	// Calculate heights for each panel section
	statusH := 5
	remainH := height - statusH
	if remainH < 4 {
		remainH = 4
	}

	// Distribute remaining height among panels
	panelHeights := a.calculatePanelHeights(remainH)

	// Render each panel section
	var sections []string
	sections = append(sections, a.renderStatusPanel(innerWidth, statusH))
	sections = append(sections, a.renderFormulaePanel(innerWidth, panelHeights[0]))
	sections = append(sections, a.renderCasksPanel(innerWidth, panelHeights[1]))
	sections = append(sections, a.renderTapsPanel(innerWidth, panelHeights[2]))
	sections = append(sections, a.renderServicesPanel(innerWidth, panelHeights[3]))

	combined := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Fit to exact height
	lines := strings.Split(combined, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

func (a *App) calculatePanelHeights(total int) [4]int {
	// Formulae, Casks, Taps, Services
	heights := [4]int{}

	// Active panel gets more space
	switch a.activePanel {
	case FormulaePanel:
		heights = distributePanelSpace(total, 0)
	case CasksPanel:
		heights = distributePanelSpace(total, 1)
	case TapsPanel:
		heights = distributePanelSpace(total, 2)
	case ServicesPanel:
		heights = distributePanelSpace(total, 3)
	default:
		heights = distributePanelSpace(total, 0)
	}

	return heights
}

func distributePanelSpace(total int, activeIdx int) [4]int {
	heights := [4]int{}
	minH := 3 // minimum height per panel (title + 1 item + border)

	// Give active panel more space
	remaining := total - minH*4
	if remaining < 0 {
		remaining = 0
		minH = total / 4
		if minH < 2 {
			minH = 2
		}
	}

	for i := range heights {
		heights[i] = minH
	}

	// Distribute remaining to active panel
	heights[activeIdx] += remaining

	return heights
}

// --- Panel Renderers ---

func (a *App) renderStatusPanel(width, height int) string {
	isActive := a.activePanel == StatusPanel
	title := a.panelTitle(StatusPanel, isActive)

	outdatedCount := len(a.outdatedFormulae) + len(a.outdatedCasks)
	content := fmt.Sprintf(
		" Formulae: %d  Casks: %d\n Outdated: %s  Taps: %d",
		len(a.formulae),
		len(a.casks),
		a.formatOutdatedCount(outdatedCount),
		len(a.taps),
	)

	return a.wrapPanel(title, content, width, height, isActive)
}

func (a *App) renderFormulaePanel(width, height int) string {
	isActive := a.activePanel == FormulaePanel
	title := a.panelTitleWithTabs(FormulaePanel, isActive, []string{"Installed", "Outdated", "Leaves"}, int(a.formulaeTab))

	items := a.getFilteredFormulae()
	content := a.renderFormulaList(items, a.formulaeCursor, width-2, height-2)

	return a.wrapPanel(title, content, width, height, isActive)
}

func (a *App) renderCasksPanel(width, height int) string {
	isActive := a.activePanel == CasksPanel
	title := a.panelTitleWithTabs(CasksPanel, isActive, []string{"Installed", "Outdated"}, int(a.caskTab))

	items := a.getFilteredCasks()
	content := a.renderCaskList(items, a.casksCursor, width-2, height-2)

	return a.wrapPanel(title, content, width, height, isActive)
}

func (a *App) renderTapsPanel(width, height int) string {
	isActive := a.activePanel == TapsPanel
	title := a.panelTitle(TapsPanel, isActive)

	content := a.renderTapList(a.taps, a.tapsCursor, width-2, height-2)

	return a.wrapPanel(title, content, width, height, isActive)
}

func (a *App) renderServicesPanel(width, height int) string {
	isActive := a.activePanel == ServicesPanel
	title := a.panelTitleWithTabs(ServicesPanel, isActive, []string{"All", "Running", "Stopped"}, int(a.serviceTab))

	items := a.getFilteredServices()
	content := a.renderServiceList(items, a.servicesCursor, width-2, height-2)

	return a.wrapPanel(title, content, width, height, isActive)
}

// --- List Renderers ---

func (a *App) renderFormulaList(items []models.Formula, cursor, width, maxLines int) string {
	if len(items) == 0 {
		return dimItemStyle.Render("  (empty)")
	}

	var lines []string
	start, end := a.visibleRange(cursor, len(items), maxLines)

	for i := start; i < end; i++ {
		f := items[i]
		name := f.Name
		version := f.CurrentVersion()

		// Truncate name if needed
		maxNameLen := width - len(version) - 6
		if maxNameLen < 8 {
			maxNameLen = 8
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "…"
		}

		// Build markers
		marker := " "
		if f.Outdated {
			marker = outdatedStyle.Render("▲")
		} else if f.Pinned {
			marker = pinnedStyle.Render("📌")
		}

		// Pad version to right-align
		padding := width - len(name) - len(version) - 4
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf(" %s %s%s%s", marker, name, strings.Repeat(" ", padding), lipgloss.NewStyle().Foreground(dimColor).Render(version))

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf(" %s %s%s%s", marker, name, strings.Repeat(" ", padding), version))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (a *App) renderCaskList(items []models.Cask, cursor, width, maxLines int) string {
	if len(items) == 0 {
		return dimItemStyle.Render("  (empty)")
	}

	var lines []string
	start, end := a.visibleRange(cursor, len(items), maxLines)

	for i := start; i < end; i++ {
		c := items[i]
		name := c.Name
		version := c.Version

		maxNameLen := width - len(version) - 6
		if maxNameLen < 8 {
			maxNameLen = 8
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "…"
		}

		marker := " "
		if c.Outdated {
			marker = outdatedStyle.Render("▲")
		}

		padding := width - len(name) - len(version) - 4
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf(" %s %s%s%s", marker, name, strings.Repeat(" ", padding), lipgloss.NewStyle().Foreground(dimColor).Render(version))

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf(" %s %s%s%s", marker, name, strings.Repeat(" ", padding), version))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (a *App) renderTapList(items []models.Tap, cursor, width, maxLines int) string {
	if len(items) == 0 {
		return dimItemStyle.Render("  (empty)")
	}

	var lines []string
	start, end := a.visibleRange(cursor, len(items), maxLines)

	for i := start; i < end; i++ {
		t := items[i]
		info := fmt.Sprintf("F:%d C:%d", t.FormulaCount, t.CaskCount)

		padding := width - len(t.Name) - len(info) - 3
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf("  %s%s%s", t.Name, strings.Repeat(" ", padding), lipgloss.NewStyle().Foreground(dimColor).Render(info))

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf("  %s%s%s", t.Name, strings.Repeat(" ", padding), info))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (a *App) renderServiceList(items []models.Service, cursor, width, maxLines int) string {
	if len(items) == 0 {
		return dimItemStyle.Render("  (empty)")
	}

	var lines []string
	start, end := a.visibleRange(cursor, len(items), maxLines)

	for i := start; i < end; i++ {
		s := items[i]
		icon := s.StatusIcon()
		statusStr := string(s.Status)

		var styledIcon string
		if s.IsRunning() {
			styledIcon = runningStyle.Render(icon)
		} else {
			styledIcon = stoppedStyle.Render(icon)
		}

		padding := width - len(s.Name) - len(statusStr) - 5
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf(" %s %s%s%s", styledIcon, s.Name, strings.Repeat(" ", padding), lipgloss.NewStyle().Foreground(dimColor).Render(statusStr))

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf(" %s %s%s%s", icon, s.Name, strings.Repeat(" ", padding), statusStr))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// --- Main Area ---

func (a *App) renderMainArea(width, height int) string {
	logHeight := 8
	if len(a.commandLog) == 0 {
		logHeight = 3
	}
	detailHeight := height - logHeight

	if detailHeight < 5 {
		detailHeight = 5
		logHeight = height - detailHeight
	}

	innerWidth := width - 2 // border

	// Detail panel
	detailPanel := a.renderDetailPanel(innerWidth, detailHeight)

	// Command log panel
	logPanel := a.renderCommandLog(innerWidth, logHeight)

	return lipgloss.JoinVertical(lipgloss.Left, detailPanel, logPanel)
}

func (a *App) renderDetailPanel(width, height int) string {
	isActive := a.focusArea == FocusMainPanel
	title := "Detail"

	content := a.detailInfo
	if content == "" {
		if a.loading {
			content = dimItemStyle.Render("  Loading...")
		} else {
			content = dimItemStyle.Render("  Select an item to view details")
		}
	}

	// Apply scrolling
	lines := strings.Split(content, "\n")
	if a.detailScroll > 0 && a.detailScroll < len(lines) {
		lines = lines[a.detailScroll:]
	}

	// Truncate to fit
	maxLines := height - 2
	if maxLines < 1 {
		maxLines = 1
	}
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	content = strings.Join(lines, "\n")

	return a.wrapPanel(title, content, width, height, isActive)
}

func (a *App) renderCommandLog(width, height int) string {
	title := "Command Log"

	var content string
	if len(a.commandLog) == 0 {
		content = dimItemStyle.Render("  No commands executed yet")
	} else {
		maxLines := height - 2
		if maxLines < 1 {
			maxLines = 1
		}
		start := len(a.commandLog) - maxLines
		if start < 0 {
			start = 0
		}
		visible := a.commandLog[start:]
		content = strings.Join(visible, "\n")
	}

	return a.wrapPanel(title, content, width, height, false)
}

// --- Bottom Bar ---

func (a *App) renderBottomBar() string {
	var keys []string

	if a.filtering {
		filterView := searchPromptStyle.Render("Filter: ") + a.filterInput.View()
		return statusBarStyle.Width(a.width).Render(filterView)
	}

	switch a.activePanel {
	case FormulaePanel:
		keys = []string{
			keyStyle.Render("i") + keyDescStyle.Render("nstall"),
			keyStyle.Render("u") + keyDescStyle.Render("ninstall"),
			keyStyle.Render("U") + keyDescStyle.Render("pgrade"),
			keyStyle.Render("r") + keyDescStyle.Render("einstall"),
			keyStyle.Render("p") + keyDescStyle.Render("in"),
			keyStyle.Render("o") + keyDescStyle.Render("pen"),
			keyStyle.Render("/") + keyDescStyle.Render("filter"),
		}
	case CasksPanel:
		keys = []string{
			keyStyle.Render("i") + keyDescStyle.Render("nstall"),
			keyStyle.Render("u") + keyDescStyle.Render("ninstall"),
			keyStyle.Render("U") + keyDescStyle.Render("pgrade"),
			keyStyle.Render("z") + keyDescStyle.Render("ap"),
			keyStyle.Render("o") + keyDescStyle.Render("pen"),
			keyStyle.Render("/") + keyDescStyle.Render("filter"),
		}
	case ServicesPanel:
		keys = []string{
			keyStyle.Render("s") + keyDescStyle.Render("tart"),
			keyStyle.Render("S") + keyDescStyle.Render("top"),
			keyStyle.Render("r") + keyDescStyle.Render("estart"),
			keyStyle.Render("/") + keyDescStyle.Render("filter"),
		}
	case TapsPanel:
		keys = []string{
			keyStyle.Render("/") + keyDescStyle.Render("filter"),
		}
	default:
		keys = []string{
			keyStyle.Render("i") + keyDescStyle.Render("nstall"),
			keyStyle.Render("/") + keyDescStyle.Render("filter"),
		}
	}

	// Add global keys
	keys = append(keys,
		keyStyle.Render("^u") + keyDescStyle.Render("pdate"),
		keyStyle.Render("^l") + keyDescStyle.Render("cleanup"),
		keyStyle.Render("?") + keyDescStyle.Render("help"),
	)

	return statusBarStyle.Width(a.width).Render(" " + strings.Join(keys, "  "))
}

// --- Overlay Rendering ---

func (a *App) renderOverlay(base string) string {
	var overlay string

	switch a.overlay {
	case OverlaySearch:
		overlay = a.renderSearchOverlay()
	case OverlayConfirm:
		overlay = a.renderConfirmOverlay()
	case OverlayHelp:
		overlay = a.renderHelpOverlay()
	default:
		return base
	}

	return a.placeOverlay(base, overlay)
}

func (a *App) renderSearchOverlay() string {
	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("🔍 Search Packages") + "\n\n")
	b.WriteString(a.searchInput.View() + "\n")

	if len(a.searchResults) > 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(dimColor).Render("Results:") + "\n")
		maxShow := 15
		if len(a.searchResults) < maxShow {
			maxShow = len(a.searchResults)
		}
		for i := 0; i < maxShow; i++ {
			prefix := "  "
			if i == a.searchCursor {
				prefix = selectedItemStyle.Render("> ")
			}
			b.WriteString(prefix + a.searchResults[i] + "\n")
		}
		if len(a.searchResults) > maxShow {
			b.WriteString(dimItemStyle.Render(fmt.Sprintf("  ... and %d more", len(a.searchResults)-maxShow)) + "\n")
		}
		b.WriteString("\n" + keyDescStyle.Render("[Enter] install  [↑↓] navigate  [Esc] cancel"))
	} else if a.searchInput.Value() != "" {
		b.WriteString("\n" + keyDescStyle.Render("[Enter] search  [Esc] cancel"))
	} else {
		b.WriteString("\n" + keyDescStyle.Render("Type to search, [Enter] to search, [Esc] to cancel"))
	}

	width := 56
	if a.width < 60 {
		width = a.width - 4
	}

	return dialogBoxStyle.Width(width).Render(b.String())
}

func (a *App) renderConfirmOverlay() string {
	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("⚠️  Confirm") + "\n\n")
	b.WriteString(a.confirmMsg + "\n\n")
	b.WriteString(keyStyle.Render("[y/Enter]") + keyDescStyle.Render(" confirm  ") +
		keyStyle.Render("[n/Esc]") + keyDescStyle.Render(" cancel"))

	width := 50
	if a.width < 54 {
		width = a.width - 4
	}

	return dialogBoxStyle.Width(width).Render(b.String())
}

func (a *App) renderHelpOverlay() string {
	var b strings.Builder

	b.WriteString(helpTitleStyle.Render("⌨️  Keyboard Shortcuts") + "\n\n")

	sections := []struct {
		title string
		keys  [][2]string
	}{
		{
			"Navigation",
			[][2]string{
				{"j/k", "Move up/down"},
				{"h/l", "Switch side/main focus"},
				{"Tab/]", "Next panel"},
				{"Shift+Tab/[", "Previous panel"},
				{"1/2/3", "Switch tab"},
				{"g/G", "Go to top/bottom"},
				{"J/K", "Scroll detail panel"},
			},
		},
		{
			"Formulae & Casks",
			[][2]string{
				{"i", "Install (search)"},
				{"u", "Uninstall"},
				{"U", "Upgrade"},
				{"r", "Reinstall"},
				{"p/P", "Pin / Unpin"},
				{"o", "Open homepage"},
				{"z", "Zap cask"},
			},
		},
		{
			"Services",
			[][2]string{
				{"s", "Start service"},
				{"S", "Stop service"},
				{"r", "Restart service"},
			},
		},
		{
			"Global",
			[][2]string{
				{"Ctrl+u", "brew update"},
				{"Ctrl+l", "brew cleanup"},
				{"Ctrl+d", "brew doctor"},
				{"Ctrl+a", "brew autoremove"},
				{"/", "Filter list"},
				{"?", "Toggle help"},
				{"q", "Quit"},
			},
		},
	}

	for _, sec := range sections {
		b.WriteString(lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(sec.title) + "\n")
		for _, k := range sec.keys {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				lipgloss.NewStyle().Foreground(accentColor).Width(12).Render(k[0]),
				keyDescStyle.Render(k[1]),
			))
		}
		b.WriteString("\n")
	}

	b.WriteString(keyDescStyle.Render("[Esc/?/q] close help"))

	width := 48
	if a.width < 52 {
		width = a.width - 4
	}

	return dialogBoxStyle.Width(width).Render(b.String())
}

// placeOverlay centers an overlay on top of the base view.
func (a *App) placeOverlay(base, overlay string) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayW := lipgloss.Width(overlay)
	overlayH := len(overlayLines)

	startY := (a.height - overlayH) / 2
	startX := (a.width - overlayW) / 2

	if startY < 0 {
		startY = 0
	}
	if startX < 0 {
		startX = 0
	}

	for i, oLine := range overlayLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}

		baseLine := baseLines[y]
		// Simple overlay: replace characters in the base line
		baseRunes := []rune(baseLine)
		overRunes := []rune(oLine)

		// Ensure base line is wide enough
		for len(baseRunes) < startX+len(overRunes) {
			baseRunes = append(baseRunes, ' ')
		}

		// Overlay
		result := string(baseRunes[:startX]) + string(overRunes)
		if startX+len(overRunes) < len(baseRunes) {
			result += string(baseRunes[startX+len(overRunes):])
		}
		baseLines[y] = result
	}

	return strings.Join(baseLines, "\n")
}

// --- Helpers ---

func (a *App) panelTitle(id PanelID, active bool) string {
	icon := PanelIcon(id)
	name := PanelName(id)
	if active {
		return activeTitleStyle.Render(icon + " " + name)
	}
	return inactiveTitleStyle.Render(icon + " " + name)
}

func (a *App) panelTitleWithTabs(id PanelID, active bool, tabs []string, activeTab int) string {
	icon := PanelIcon(id)
	name := PanelName(id)

	var titleParts []string
	if active {
		titleParts = append(titleParts, activeTitleStyle.Render(icon+" "+name))
	} else {
		titleParts = append(titleParts, inactiveTitleStyle.Render(icon+" "+name))
	}

	if active {
		tabStr := " "
		for i, t := range tabs {
			if i == activeTab {
				tabStr += activeTabStyle.Render(t) + " "
			} else {
				tabStr += inactiveTabStyle.Render(t) + " "
			}
		}
		titleParts = append(titleParts, tabStr)
	}

	return strings.Join(titleParts, "")
}

func (a *App) wrapPanel(title, content string, width, height int, active bool) string {
	style := inactivePanelStyle
	if active {
		style = activePanelStyle
	}

	// Calculate inner dimensions
	innerW := width - 2 // borders
	innerH := height - 2 // borders

	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	// Build content with title
	titleLine := title
	contentLines := strings.Split(content, "\n")

	// Truncate content to fit
	if len(contentLines) > innerH-1 {
		contentLines = contentLines[:innerH-1]
	}

	// Pad content lines to fill height
	for len(contentLines) < innerH-1 {
		contentLines = append(contentLines, "")
	}

	fullContent := titleLine + "\n" + strings.Join(contentLines, "\n")

	return style.Width(innerW).Height(innerH).Render(fullContent)
}

func (a *App) formatOutdatedCount(count int) string {
	if count == 0 {
		return cmdLogSuccess.Render("0 ✓")
	}
	return outdatedStyle.Render(fmt.Sprintf("%d ▲", count))
}

func (a *App) visibleRange(cursor, total, maxVisible int) (int, int) {
	if total <= maxVisible {
		return 0, total
	}

	start := cursor - maxVisible/2
	if start < 0 {
		start = 0
	}

	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	return start, end
}
