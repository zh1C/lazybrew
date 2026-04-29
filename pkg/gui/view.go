package gui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// View implements tea.Model - renders the entire TUI.
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	// Calculate layout dimensions — lazygit uses SidePanelWidth: 0.3333
	// sideSectionWeight = Round(120 * 0.3333) = 40, mainSectionWeight = Round(120 * 0.6667) = 80
	const sidePanelRatio = 0.3333
	const maxColumnCount = 120
	sideSectionWeight := int(math.Round(maxColumnCount * sidePanelRatio))
	mainSectionWeight := int(math.Round(maxColumnCount * (1 - sidePanelRatio)))
	sideWidth := a.width * sideSectionWeight / (sideSectionWeight + mainSectionWeight)
	if sideWidth < 24 {
		sideWidth = 24
	}
	mainWidth := a.width - sideWidth
	bottomHeight := 1
	contentHeight := a.height - bottomHeight

	// Render side panels
	sidePanel := a.renderSidePanel(sideWidth, contentHeight)

	// Render main area (detail + log)
	mainPanel := a.renderMainArea(mainWidth, contentHeight)

	// Compose the body
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidePanel, mainPanel)

	// Render bottom bar
	bottomBar := a.renderBottomBar()

	// Compose full view
	view := lipgloss.JoinVertical(lipgloss.Left, body, bottomBar)

	// Render overlay if active
	if a.overlay != OverlayNone {
		view = a.renderOverlay(view)
	}

	return view
}

// --- Side Panel ---

func (a *App) renderSidePanel(width, height int) string {
	innerWidth := width - 2

	// Status: always Size:3 (like lazygit — border + 1 line content + border)
	statusH := 3

	// Services: collapsed Size:3 when not active, Weight:1 when active (like lazygit's Stash)
	servicesActive := a.activePanel == ServicesPanel
	var servicesH int
	var formulaeH, casksH, tapsH int

	if servicesActive {
		// Four panels share remaining space equally (all Weight:1)
		remainH := height - statusH
		if remainH < 12 {
			remainH = 12
		}
		eachH := remainH / 4
		extra := remainH % 4
		formulaeH = eachH
		casksH = eachH
		tapsH = eachH
		servicesH = eachH
		// Distribute remainder to first panels
		if extra > 0 {
			formulaeH++
			extra--
		}
		if extra > 0 {
			casksH++
			extra--
		}
		if extra > 0 {
			tapsH++
		}
	} else {
		// Services collapsed (Size:3), three panels share remaining space equally
		servicesH = 3
		remainH := height - statusH - servicesH
		if remainH < 9 {
			remainH = 9
		}
		eachH := remainH / 3
		extra := remainH % 3
		formulaeH = eachH
		casksH = eachH
		tapsH = eachH
		// Distribute remainder
		if extra > 0 {
			formulaeH++
			extra--
		}
		if extra > 0 {
			casksH++
		}
	}

	var sections []string
	sections = append(sections, a.renderStatusPanel(innerWidth, statusH))
	sections = append(sections, a.renderFormulaePanel(innerWidth, formulaeH))
	sections = append(sections, a.renderCasksPanel(innerWidth, casksH))
	sections = append(sections, a.renderTapsPanel(innerWidth, tapsH))
	sections = append(sections, a.renderServicesPanel(innerWidth, servicesH))

	combined := lipgloss.JoinVertical(lipgloss.Left, sections...)

	lines := strings.Split(combined, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return lipgloss.NewStyle().Width(width).Render(strings.Join(lines, "\n"))
}

// calculatePanelHeights and distributePanelSpace removed — height distribution
// is now computed inline in renderSidePanel using lazygit's Weight/Size pattern.

// --- Panel Renderers ---

func (a *App) renderStatusPanel(width, height int) string {
	isActive := a.activePanel == StatusPanel

	outdatedCount := len(a.outdatedFormulae) + len(a.outdatedCasks)

	// Single-line content, adaptive to available width (like lazygit's "repo → branch")
	// Width tiers: full → compact → minimal
	availW := width - 2 // inner content width

	var content string
	switch {
	case a.stage == StageLoading:
		content = fmt.Sprintf(" %s Loading...", a.spinnerChar())
	case a.stage == StageEnriching:
		// Show counts + spinner
		full := fmt.Sprintf(" Formulae: %d  Casks: %d  Taps: %d  %s",
			len(a.formulaeNames), len(a.caskNames), len(a.tapNames), a.spinnerChar())
		compact := fmt.Sprintf(" F: %d  C: %d  T: %d  %s",
			len(a.formulaeNames), len(a.caskNames), len(a.tapNames), a.spinnerChar())
		if len(full) <= availW {
			content = full
		} else {
			content = compact
		}
	default:
		// Stage complete — show full stats with outdated
		outStr := fmt.Sprintf("%d", outdatedCount)
		full := fmt.Sprintf(" Formulae: %d  Casks: %d  Taps: %d  Outdated: %s",
			len(a.formulaeNames), len(a.caskNames), len(a.tapNames), outStr)
		medium := fmt.Sprintf(" Formulae: %d  Casks: %d  Outdated: %s",
			len(a.formulaeNames), len(a.caskNames), outStr)
		compact := fmt.Sprintf(" F: %d  C: %d  T: %d  Out: %s",
			len(a.formulaeNames), len(a.caskNames), len(a.tapNames), outStr)

		switch {
		case len(full) <= availW:
			content = full
		case len(medium) <= availW:
			content = medium
		default:
			content = compact
		}
	}

	return a.wrapPanel(StatusPanel, nil, content, width, height, isActive, 0, 0)
}

func (a *App) renderFormulaePanel(width, height int) string {
	isActive := a.activePanel == FormulaePanel
	tabs := []string{"Installed", "Outdated", "Leaves"}

	items := a.getFilteredFormulaeNames()
	content := a.renderNameList(items, a.formulaeCursor, a.formulaeVersions, a.outdatedFormulae, width-2, height-2)

	if a.formulaeLoading {
		content = dimItemStyle.Render(" " + a.spinnerChar() + " Loading formulae...")
	}

	cursor := a.formulaeCursor
	total := len(items)
	return a.wrapPanel(FormulaePanel, tabs, content, width, height, isActive, cursor, total)
}

func (a *App) renderCasksPanel(width, height int) string {
	isActive := a.activePanel == CasksPanel
	tabs := []string{"Installed", "Outdated"}

	items := a.getFilteredCaskNames()
	content := a.renderNameList(items, a.casksCursor, a.caskVersions, a.outdatedCasks, width-2, height-2)

	if a.casksLoading {
		content = dimItemStyle.Render(" " + a.spinnerChar() + " Loading casks...")
	}

	cursor := a.casksCursor
	total := len(items)
	return a.wrapPanel(CasksPanel, tabs, content, width, height, isActive, cursor, total)
}

func (a *App) renderTapsPanel(width, height int) string {
	isActive := a.activePanel == TapsPanel

	items := a.getFilteredTapNames()
	content := a.renderTapNameList(items, a.tapsCursor, width-2, height-2)

	if a.tapsLoading {
		content = dimItemStyle.Render(" " + a.spinnerChar() + " Loading taps...")
	}

	cursor := a.tapsCursor
	total := len(items)
	return a.wrapPanel(TapsPanel, nil, content, width, height, isActive, cursor, total)
}

func (a *App) renderServicesPanel(width, height int) string {
	isActive := a.activePanel == ServicesPanel
	tabs := []string{"All", "Running", "Stopped"}

	items := a.getFilteredServices()
	content := a.renderServiceList(items, a.servicesCursor, width-2, height-2)

	if a.servicesLoading {
		content = dimItemStyle.Render(" " + a.spinnerChar() + " Loading services...")
	}

	cursor := a.servicesCursor
	total := len(items)
	return a.wrapPanel(ServicesPanel, tabs, content, width, height, isActive, cursor, total)
}

// --- List Renderers ---

// renderNameList renders a list of package names with optional version and outdated markers.
func (a *App) renderNameList(items []string, cursor int, versions map[string]string, outdated map[string]bool, width, maxLines int) string {
	if len(items) == 0 {
		return dimItemStyle.Render("  (empty)")
	}

	var lines []string
	start, end := a.visibleRange(cursor, len(items), maxLines)

	for i := start; i < end; i++ {
		name := items[i]
		version := versions[name]

		// Truncate name if needed
		maxNameLen := width - len(version) - 6
		if maxNameLen < 8 {
			maxNameLen = 8
		}
		if len(name) > maxNameLen {
			name = name[:maxNameLen-1] + "…"
		}

		// Outdated marker: * like lazygit uses for modified files
		marker := " "
		if outdated[items[i]] {
			marker = outdatedStyle.Render("*")
		}

		// Pad version to right-align
		padding := width - len(name) - len(version) - 4
		if padding < 1 {
			padding = 1
		}

		versionDisplay := dimItemStyle.Render(version)

		line := fmt.Sprintf(" %s %s%s%s", marker, name, strings.Repeat(" ", padding), versionDisplay)

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf(" %s %s%s%s", marker, name, strings.Repeat(" ", padding), version))
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (a *App) renderTapNameList(items []string, cursor, width, maxLines int) string {
	if len(items) == 0 {
		return dimItemStyle.Render("  (empty)")
	}

	var lines []string
	start, end := a.visibleRange(cursor, len(items), maxLines)

	for i := start; i < end; i++ {
		name := items[i]
		// Try to find detail info for this tap
		info := ""
		for _, t := range a.taps {
			if t.Name == name {
				info = fmt.Sprintf("F:%d C:%d", t.FormulaCount, t.CaskCount)
				break
			}
		}

		padding := width - len(name) - len(info) - 3
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf("  %s%s%s", name, strings.Repeat(" ", padding), dimItemStyle.Render(info))

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf("  %s%s%s", name, strings.Repeat(" ", padding), info))
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
		statusStr := string(s.Status)

		// Status marker: * for running, x for stopped
		var marker string
		if s.IsRunning() {
			marker = runningStyle.Render("*")
		} else {
			marker = stoppedStyle.Render("x")
		}

		padding := width - len(s.Name) - len(statusStr) - 5
		if padding < 1 {
			padding = 1
		}

		line := fmt.Sprintf(" %s %s%s%s", marker, s.Name, strings.Repeat(" ", padding), dimItemStyle.Render(statusStr))

		if i == cursor {
			line = selectedItemStyle.Width(width).Render(fmt.Sprintf(" %s %s%s%s", marker, s.Name, strings.Repeat(" ", padding), statusStr))
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

	innerWidth := width - 2

	detailPanel := a.renderDetailPanel(innerWidth, detailHeight)
	logPanel := a.renderCommandLog(innerWidth, logHeight)

	return lipgloss.JoinVertical(lipgloss.Left, detailPanel, logPanel)
}

func (a *App) renderDetailPanel(width, height int) string {
	isActive := a.focusArea == FocusMainPanel

	content := a.detailInfo
	if content == "" {
		if a.stage == StageLoading {
			content = dimItemStyle.Render(" " + a.spinnerChar() + " Loading...")
		} else {
			content = dimItemStyle.Render(" Select an item to view details")
		}
	}

	// Apply scrolling
	contentLines := strings.Split(content, "\n")
	if a.detailScroll > 0 && a.detailScroll < len(contentLines) {
		contentLines = contentLines[a.detailScroll:]
	}

	maxLines := height - 2
	if maxLines < 1 {
		maxLines = 1
	}
	if len(contentLines) > maxLines {
		contentLines = contentLines[:maxLines]
	}

	content = strings.Join(contentLines, "\n")

	// Use wrapPanelRaw for detail (no panel ID, just a title string)
	return a.wrapPanelRaw("Detail", content, width, height, isActive)
}

func (a *App) renderCommandLog(width, height int) string {
	var content string
	if len(a.commandLog) == 0 {
		content = dimItemStyle.Render(" No commands executed yet")
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

	return a.wrapPanelRaw("Command log", content, width, height, false)
}

// --- Bottom Bar ---
// Format: "Install: i | Uninstall: u | Keybindings: ?" (like lazygit)

func (a *App) renderBottomBar() string {
	if a.filtering {
		filterView := searchPromptStyle.Render("Filter: ") + a.filterInput.View()
		return statusBarStyle.Width(a.width).Render(filterView)
	}

	var items []string

	switch a.activePanel {
	case FormulaePanel:
		items = []string{
			"Install: i",
			"Uninstall: u",
			"Upgrade: U",
			"Reinstall: r",
			"Pin: p",
			"Open: o",
		}
	case CasksPanel:
		items = []string{
			"Install: i",
			"Uninstall: u",
			"Upgrade: U",
			"Zap: z",
			"Open: o",
		}
	case ServicesPanel:
		items = []string{
			"Start: s",
			"Stop: S",
			"Restart: r",
		}
	case TapsPanel:
		items = []string{}
	default:
		items = []string{
			"Install: i",
		}
	}

	items = append(items,
		"Filter: /",
		"Update: ^u",
		"Cleanup: ^l",
		"Keybindings: ?",
	)

	// Truncate if too wide
	separator := " | "
	text := strings.Join(items, separator)
	if lipgloss.Width(text)+2 > a.width {
		// Try shorter format
		for lipgloss.Width(text)+2 > a.width && len(items) > 1 {
			items = items[:len(items)-1]
		}
		text = strings.Join(items, separator)
	}

	return statusBarStyle.Width(a.width).Render(" " + text)
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

	b.WriteString(helpTitleStyle.Render("Search Packages") + "\n\n")
	b.WriteString(a.searchInput.View() + "\n")

	if len(a.searchResults) > 0 {
		b.WriteString("\n" + dimItemStyle.Render("Results:") + "\n")
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
		b.WriteString("\n" + keyDescStyle.Render("[Enter] install  [up/down] navigate  [Esc] cancel"))
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

	b.WriteString(helpTitleStyle.Render("Confirm") + "\n\n")
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

	b.WriteString(helpTitleStyle.Render("Keybindings") + "\n\n")

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
		b.WriteString(helpTitleStyle.Render(sec.title) + "\n")
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
		baseRunes := []rune(baseLine)
		overRunes := []rune(oLine)

		for len(baseRunes) < startX+len(overRunes) {
			baseRunes = append(baseRunes, ' ')
		}

		result := string(baseRunes[:startX]) + string(overRunes)
		if startX+len(overRunes) < len(baseRunes) {
			result += string(baseRunes[startX+len(overRunes):])
		}
		baseLines[y] = result
	}

	return strings.Join(baseLines, "\n")
}

// --- Panel Wrapping Helpers ---

// wrapPanel renders a panel with lazygit-style title embedded in the border.
// Format: ╭─[1]─Formulae─ Installed - Outdated - Leaves──────── N of M─╮
func (a *App) wrapPanel(id PanelID, tabs []string, content string, width, height int, active bool, cursor, total int) string {
	style := inactivePanelStyle
	titleStyle := inactiveTitleStyle
	borderColor := inactiveBorderColor
	if active {
		style = activePanelStyle
		titleStyle = activeTitleStyle
		borderColor = activeBorderColor
	}

	innerW := width - 2
	innerH := height - 2

	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	// Build the top border with embedded title (lazygit style)
	// Format: ╭─[1]─Title─ Tab1 - Tab2 - Tab3────────── N of M─╮
	prefix := PanelIndex(id)
	name := PanelName(id)

	titleText := prefix + "─" + name
	if tabs != nil && active {
		activeTab := a.activeTabForPanel(id)
		tabParts := make([]string, len(tabs))
		for i, t := range tabs {
			if i == activeTab {
				tabParts[i] = activeTabStyle.Render(t)
			} else {
				tabParts[i] = inactiveTabStyle.Render(t)
			}
		}
		titleText += "─ " + strings.Join(tabParts, " - ") + " "
	}

	// Right side: "N of M" counter like lazygit
	counterText := ""
	if total > 0 {
		counterText = fmt.Sprintf(" %d of %d", cursor+1, total)
	}

	// We'll render the title into the top border
	renderedTitle := titleStyle.Render(titleText)

	// Build content lines
	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerH {
		contentLines = contentLines[:innerH]
	}
	for len(contentLines) < innerH {
		contentLines = append(contentLines, "")
	}

	fullContent := strings.Join(contentLines, "\n")

	// Render the panel with border
	rendered := style.Width(innerW).Height(innerH).Render(fullContent)

	// Now replace the top border line to embed the title
	renderedLines := strings.Split(rendered, "\n")
	if len(renderedLines) > 0 {
		renderedLines[0] = a.buildBorderTitle(renderedTitle, counterText, width, borderColor)
	}

	return strings.Join(renderedLines, "\n")
}

// wrapPanelRaw wraps content with a simple title string (for detail/log panels).
func (a *App) wrapPanelRaw(title, content string, width, height int, active bool) string {
	style := inactivePanelStyle
	titleStyle := inactiveTitleStyle
	borderColor := inactiveBorderColor
	if active {
		style = activePanelStyle
		titleStyle = activeTitleStyle
		borderColor = activeBorderColor
	}

	innerW := width - 2
	innerH := height - 2

	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	renderedTitle := titleStyle.Render(title)

	contentLines := strings.Split(content, "\n")
	if len(contentLines) > innerH {
		contentLines = contentLines[:innerH]
	}
	for len(contentLines) < innerH {
		contentLines = append(contentLines, "")
	}

	fullContent := strings.Join(contentLines, "\n")
	rendered := style.Width(innerW).Height(innerH).Render(fullContent)

	renderedLines := strings.Split(rendered, "\n")
	if len(renderedLines) > 0 {
		renderedLines[0] = a.buildBorderTitle(renderedTitle, "", width, borderColor)
	}

	return strings.Join(renderedLines, "\n")
}

// buildBorderTitle builds a top border line with embedded title, like lazygit:
// ╭─[1]─Formulae─ Installed - Outdated──────── 3 of 42─╮
func (a *App) buildBorderTitle(renderedTitle, counterText string, width int, borderColor lipgloss.Color) string {
	colorStyle := lipgloss.NewStyle().Foreground(borderColor)

	left := colorStyle.Render("╭─")
	right := colorStyle.Render("─╮")

	counterRendered := ""
	if counterText != "" {
		counterRendered = dimItemStyle.Render(counterText) + colorStyle.Render("─")
	}

	// Calculate fill width
	titleWidth := lipgloss.Width(renderedTitle)
	counterWidth := lipgloss.Width(counterRendered)
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)

	fillLen := width - leftWidth - titleWidth - counterWidth - rightWidth
	if fillLen < 1 {
		fillLen = 1
	}

	fill := colorStyle.Render(strings.Repeat("─", fillLen))

	return left + renderedTitle + fill + counterRendered + right
}

// activeTabForPanel returns the active tab index for the given panel.
func (a *App) activeTabForPanel(id PanelID) int {
	switch id {
	case FormulaePanel:
		return int(a.formulaeTab)
	case CasksPanel:
		return int(a.caskTab)
	case ServicesPanel:
		return int(a.serviceTab)
	default:
		return 0
	}
}

func (a *App) formatOutdatedCount(count int) string {
	if count == 0 {
		return cmdLogSuccess.Render("0")
	}
	return outdatedStyle.Render(fmt.Sprintf("%d", count))
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
