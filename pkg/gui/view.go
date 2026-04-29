package gui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// View implements tea.Model - renders the entire TUI.
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	// Calculate layout dimensions
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

	// Status: always Size:3
	statusH := 3

	// Services: collapsed Size:3 when not active, Weight:1 when active
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
// is now computed inline in renderSidePanel

// --- Panel Renderers ---

func (a *App) renderStatusPanel(width, height int) string {
	isActive := a.activePanel == StatusPanel

	outdatedCount := len(a.outdatedFormulae) + len(a.outdatedCasks)

	// Single-line content, adaptive to available width
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

		// Outdated marker: *
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
// Format: "Install: i | Uninstall: u | Keybindings: ?"

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

// ============================================================
// Popup System — reusable overlay rendering
// ============================================================

// popupWidth calculates popup panel width的的
func (a *App) popupWidth(maxWidth int) int {
	panelWidth := 4 * a.width / 7
	if panelWidth > maxWidth {
		panelWidth = maxWidth
	}
	minWidth := 80
	if panelWidth < minWidth {
		panelWidth = a.width - 2
		if panelWidth > minWidth {
			panelWidth = minWidth
		}
	}
	if panelWidth < 20 {
		panelWidth = 20
	}
	return panelWidth
}

// renderPopupBox renders a bordered popup box with title, body content, and footer hints.
// This is the reusable building block for all popups (lazygit: CreatePopupPanel).
//
//	╭─ Title ──────────────────────╮
//	│ body content                 │
//	│                              │
//	│ [key] hint  [key] hint       │
//	╰──────────────────────────────╯
func (a *App) renderPopupBox(title, body, footer string, maxWidth int) string {
	panelW := a.popupWidth(maxWidth)
	innerW := panelW - 4 // 2 border + 2 padding
	if innerW < 10 {
		innerW = 10
	}

	// Wrap body lines to innerW
	var content strings.Builder
	if body != "" {
		for _, line := range strings.Split(body, "\n") {
			content.WriteString(wrapText(line, innerW) + "\n")
		}
	}
	if footer != "" {
		content.WriteString("\n" + footer)
	}

	// Cap height at 75% of terminal
	contentStr := strings.TrimRight(content.String(), "\n")
	contentLines := strings.Split(contentStr, "\n")
	maxH := a.height * 3 / 4
	if maxH < 5 {
		maxH = 5
	}
	if len(contentLines) > maxH {
		contentLines = contentLines[:maxH]
		contentStr = strings.Join(contentLines, "\n")
	}

	// Build the box with title in border
	borderColor := activeBorderColor
	colorStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleRendered := lipgloss.NewStyle().Foreground(borderColor).Bold(true).Render(" " + title + " ")

	// Render content inside a bordered box
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(innerW + 2) // innerW + 2 padding

	rendered := boxStyle.Render(contentStr)

	// Replace top border to embed title (same as wrapPanel pattern)
	renderedLines := strings.Split(rendered, "\n")
	if len(renderedLines) > 0 {
		left := colorStyle.Render("╭─")
		right := colorStyle.Render("─╮")
		totalW := panelW
		fillLen := totalW - lipgloss.Width(left) - lipgloss.Width(titleRendered) - lipgloss.Width(right)
		if fillLen < 1 {
			fillLen = 1
		}
		fill := colorStyle.Render(strings.Repeat("─", fillLen))
		renderedLines[0] = left + titleRendered + fill + right
	}

	return strings.Join(renderedLines, "\n")
}

// placePopup composites a popup string on top of the base view using
// ANSI-safe string splicing. Unlike lipgloss.Place (which replaces the base
// entirely with whitespace), this preserves the background content on both
// sides of the popup, similar to gocui's painter's algorithm but at the
// string level.
//
// For each line in the popup's vertical range:
//
//	result = base[0:offsetX] + popupLine + base[offsetX+popupW:]
//
// All cuts use ansi.Truncate / ansi.TruncateLeft so ANSI escape sequences
// are never broken.
func (a *App) placePopup(base, popup string) string {
	baseLines := strings.Split(base, "\n")
	popupLines := strings.Split(popup, "\n")

	popupW := lipgloss.Width(popup)
	popupH := len(popupLines)

	// Center the popup
	offsetY := (a.height - popupH) / 2
	offsetX := (a.width - popupW) / 2
	if offsetY < 0 {
		offsetY = 0
	}
	if offsetX < 0 {
		offsetX = 0
	}

	// Ensure base has enough lines
	for len(baseLines) < offsetY+popupH {
		baseLines = append(baseLines, strings.Repeat(" ", a.width))
	}

	for i, pLine := range popupLines {
		y := offsetY + i
		if y >= len(baseLines) {
			break
		}

		bLine := baseLines[y]
		bLineW := ansi.StringWidth(bLine)

		// Pad base line to at least cover the popup area
		if bLineW < offsetX+popupW {
			bLine = bLine + strings.Repeat(" ", offsetX+popupW-bLineW)
		}

		// Three-segment splice: left | popup | right
		left := ansi.Truncate(bLine, offsetX, "")
		right := ansi.TruncateLeft(bLine, offsetX+popupW, "")

		baseLines[y] = left + pLine + right
	}

	return strings.Join(baseLines, "\n")
}

// wrapText does simple word-boundary line wrapping to fit within maxWidth.
func wrapText(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth || maxWidth <= 0 {
		return s
	}
	var result strings.Builder
	words := strings.Fields(s)
	lineLen := 0
	for i, w := range words {
		wLen := lipgloss.Width(w)
		if i > 0 && lineLen+1+wLen > maxWidth {
			result.WriteString("\n")
			lineLen = 0
		}
		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}
		result.WriteString(w)
		lineLen += wLen
	}
	return result.String()
}

// --- Overlay Dispatch ---

func (a *App) renderOverlay(base string) string {
	var popup string

	switch a.overlay {
	case OverlaySearch:
		popup = a.renderSearchPopup()
	case OverlayConfirm:
		popup = a.renderConfirmPopup()
	case OverlayHelp:
		popup = a.renderHelpPopup()
	default:
		return base
	}

	return a.placePopup(base, popup)
}

// --- Popup Content Builders ---

func (a *App) renderSearchPopup() string {
	var body strings.Builder

	body.WriteString(a.searchInput.View() + "\n")

	if len(a.searchResults) > 0 {
		body.WriteString("\n" + dimItemStyle.Render("Results:") + "\n")
		maxShow := 15
		if len(a.searchResults) < maxShow {
			maxShow = len(a.searchResults)
		}
		for i := 0; i < maxShow; i++ {
			prefix := "  "
			if i == a.searchCursor {
				prefix = selectedItemStyle.Render("> ")
			}
			body.WriteString(prefix + a.searchResults[i] + "\n")
		}
		if len(a.searchResults) > maxShow {
			body.WriteString(dimItemStyle.Render(fmt.Sprintf("  ... and %d more", len(a.searchResults)-maxShow)) + "\n")
		}
	}

	var footer string
	if len(a.searchResults) > 0 {
		footer = keyDescStyle.Render("[Enter] install  [↑/↓] navigate  [Esc] cancel")
	} else if a.searchInput.Value() != "" {
		footer = keyDescStyle.Render("[Enter] search  [Esc] cancel")
	} else {
		footer = keyDescStyle.Render("Type to search, [Enter] to search, [Esc] to cancel")
	}

	return a.renderPopupBox("Search Packages", body.String(), footer, 80)
}

func (a *App) renderConfirmPopup() string {
	body := a.confirmMsg
	footer := keyStyle.Render("[y/Enter]") + keyDescStyle.Render(" confirm  ") +
		keyStyle.Render("[n/Esc]") + keyDescStyle.Render(" cancel")

	return a.renderPopupBox("Confirm", body, footer, 60)
}

func (a *App) renderHelpPopup() string {
	var body strings.Builder

	sections := []struct {
		title string
		keys  [][2]string
	}{
		{
			"Navigation",
			[][2]string{
				{"j/k", "Move up/down"},
				{"h/l", "Switch side/main focus"},
				{"Tab", "Next panel"},
				{"Shift+Tab", "Previous panel"},
				{"] / [", "Next/Prev tab"},
				{"1/2/3", "Jump to tab"},
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
		body.WriteString(helpTitleStyle.Render(sec.title) + "\n")
		for _, k := range sec.keys {
			body.WriteString(fmt.Sprintf("  %s  %s\n",
				lipgloss.NewStyle().Foreground(accentColor).Width(12).Render(k[0]),
				keyDescStyle.Render(k[1]),
			))
		}
		body.WriteString("\n")
	}

	footer := keyDescStyle.Render("[Esc/?/q] close help")

	return a.renderPopupBox("Keybindings", body.String(), footer, 60)
}

// --- Panel Wrapping Helpers ---

// wrapPanel renders a panel with title embedded in the border.
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

	// Build the top border with embedded title
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

	// Right side: "N of M" counter
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

// buildBorderTitle builds a top border line with embedded title
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
