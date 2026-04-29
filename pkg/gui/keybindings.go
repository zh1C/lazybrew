package gui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================
// Key Handling — all keyboard event dispatch lives here
// ============================================================

func (a *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.overlay != OverlayNone {
		return a.handleOverlayKey(msg)
	}
	if a.filtering {
		return a.handleFilterKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit

	// Panel navigation
	case "tab":
		a.nextPanel()
		return a, a.loadDetailForCurrentItem()
	case "shift+tab":
		a.prevPanel()
		return a, a.loadDetailForCurrentItem()

	// Tab switching within panel
	case "]":
		return a.nextTab()
	case "[":
		return a.prevTab()

	// Focus switching
	case "h":
		if a.focusArea == FocusMainPanel {
			a.focusArea = FocusSidePanel
		}
		return a, nil
	case "l":
		if a.focusArea == FocusSidePanel {
			a.focusArea = FocusMainPanel
		}
		return a, nil

	// List navigation
	case "j", "down":
		a.moveCursorDown()
		return a, a.loadDetailForCurrentItem()
	case "k", "up":
		a.moveCursorUp()
		return a, a.loadDetailForCurrentItem()
	case "g":
		a.moveCursorToTop()
		return a, a.loadDetailForCurrentItem()
	case "G":
		a.moveCursorToBottom()
		return a, a.loadDetailForCurrentItem()

	// Tab switching by number
	case "1":
		return a.switchTab(0)
	case "2":
		return a.switchTab(1)
	case "3":
		return a.switchTab(2)

	// Detail panel scrolling
	case "J":
		if a.focusArea == FocusMainPanel {
			a.detailScroll++
		}
		return a, nil
	case "K":
		if a.focusArea == FocusMainPanel && a.detailScroll > 0 {
			a.detailScroll--
		}
		return a, nil

	// Actions
	case "i":
		return a.startSearch()
	case "u":
		return a.uninstallSelected()
	case "U":
		return a.upgradeSelected()
	case "r":
		return a.handleReinstartOrReinstall()
	case "p":
		return a.pinSelected()
	case "P":
		return a.unpinSelected()
	case "d":
		return a, a.loadDetailForCurrentItem()
	case "o":
		return a.openHomepage()
	case "s":
		return a.startService()
	case "S":
		return a.stopService()
	case "z":
		return a.zapCask()

	// Global actions
	case "ctrl+u":
		return a.brewUpdate()
	case "ctrl+l":
		return a.brewCleanup()
	case "ctrl+d":
		return a.brewDoctor()
	case "ctrl+a":
		return a.brewAutoremove()

	case "/":
		a.filtering = true
		a.filterInput.Focus()
		return a, nil

	case "?":
		a.overlay = OverlayHelp
		return a, nil

	case "enter":
		return a, a.loadDetailForCurrentItem()
	}

	return a, nil
}

func (a *App) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.overlay {
	case OverlaySearch:
		return a.handleSearchKey(msg)
	case OverlayConfirm:
		return a.handleConfirmKey(msg)
	case OverlayHelp:
		if msg.String() == "esc" || msg.String() == "?" || msg.String() == "q" {
			a.overlay = OverlayNone
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.overlay = OverlayNone
		a.searchResults = nil
		a.searchInput.Reset()
		return a, nil
	case "enter":
		if a.searchInput.Value() != "" && len(a.searchResults) == 0 {
			query := a.searchInput.Value()
			// Use cache search if available (instant), fallback to brew search
			if a.formulaCache != nil || a.caskCache != nil {
				a.addLog(cmdLogPrefix.Render("$") + " search " + query + " (cache)")
				return a, searchFromCache(a, query)
			}
			a.addLog(cmdLogPrefix.Render("$") + " brew search " + query)
			return a, searchPackages(a.cmds, query)
		}
		if len(a.searchResults) > 0 && a.searchCursor < len(a.searchResults) {
			return a.installSearchResult()
		}
		return a, nil
	case "up", "ctrl+p":
		if a.searchCursor > 0 {
			a.searchCursor--
		}
		return a, nil
	case "down", "ctrl+n":
		if a.searchCursor < len(a.searchResults)-1 {
			a.searchCursor++
		}
		return a, nil
	default:
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		return a, cmd
	}
}

func (a *App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		a.overlay = OverlayNone
		if a.confirmAction != nil {
			cmd := a.confirmAction()
			a.confirmAction = nil
			return a, cmd
		}
		return a, nil
	case "n", "esc":
		a.overlay = OverlayNone
		a.confirmAction = nil
		return a, nil
	}
	return a, nil
}

func (a *App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.filtering = false
		a.filterText = ""
		a.filterInput.Reset()
		a.filterInput.Blur()
		return a, nil
	case "enter":
		a.filterText = a.filterInput.Value()
		a.filtering = false
		a.filterInput.Blur()
		return a, nil
	default:
		var cmd tea.Cmd
		a.filterInput, cmd = a.filterInput.Update(msg)
		a.filterText = a.filterInput.Value()
		return a, cmd
	}
}
