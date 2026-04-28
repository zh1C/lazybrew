package gui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================
// Message handlers — all tea.Msg → state update logic
// (like lazygit's gui/refresh.go + gui/state.go)
// ============================================================

// --- Stage 1 Handlers (instant name lists) ---

func (a *App) handleFormulaeNames(msg FormulaeNamesMsg) (tea.Model, tea.Cmd) {
	a.formulaeLoading = false
	if msg.Err != nil {
		a.errMsg = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil
	}
	a.formulaeNames = msg.Names
	a.errMsg = ""

	// Check if Stage 1 is complete → start Stage 2
	return a, a.maybeStartStage2()
}

func (a *App) handleCaskNames(msg CaskNamesMsg) (tea.Model, tea.Cmd) {
	a.casksLoading = false
	if msg.Err != nil {
		a.errMsg = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil
	}
	a.caskNames = msg.Names
	return a, a.maybeStartStage2()
}

func (a *App) handleTapNames(msg TapNamesMsg) (tea.Model, tea.Cmd) {
	a.tapsLoading = false
	if msg.Err != nil {
		// Non-critical
	}
	a.tapNames = msg.Names
	return a, a.maybeStartStage2()
}

// maybeStartStage2 checks if all Stage 1 data has arrived.
// If so, transitions to Stage 2 and fires background enrichment commands.
func (a *App) maybeStartStage2() tea.Cmd {
	// Still waiting for some Stage 1 data
	if a.formulaeLoading || a.casksLoading || a.tapsLoading {
		return nil
	}

	// Stage 1 complete → start Stage 2
	a.stage = StageEnriching
	a.servicesLoading = true

	return tea.Batch(
		loadFormulaeVersions(a.cmds), // ~1s
		loadCaskVersions(a.cmds),     // ~1s
		loadOutdatedNames(a.cmds),    // ~1.5s
		loadLeaves(a.cmds),           // ~1.7s
		loadServices(a.cmds),         // ~1.5s
		loadTapsDetail(a.cmds),       // ~0.5s
		spinnerTick(),                // keep spinner alive
	)
}

// --- Stage 2 Handlers (background enrichment) ---

func (a *App) handleFormulaeVersions(msg FormulaeVersionsMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.formulaeVersions = msg.Versions
	}
	a.checkStageComplete()
	return a, nil
}

func (a *App) handleCaskVersions(msg CaskVersionsMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.caskVersions = msg.Versions
	}
	a.checkStageComplete()
	return a, nil
}

func (a *App) handleOutdatedNames(msg OutdatedNamesMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.outdatedFormulae = make(map[string]bool, len(msg.Formulae))
		for _, name := range msg.Formulae {
			a.outdatedFormulae[name] = true
		}
		a.outdatedCasks = make(map[string]bool, len(msg.Casks))
		for _, name := range msg.Casks {
			a.outdatedCasks[name] = true
		}
	}
	a.checkStageComplete()
	return a, nil
}

func (a *App) handleLeaves(msg LeavesLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.leaves = make(map[string]bool, len(msg.Leaves))
		for _, name := range msg.Leaves {
			a.leaves[name] = true
		}
	}
	a.checkStageComplete()
	return a, nil
}

func (a *App) handleServices(msg ServicesLoadedMsg) (tea.Model, tea.Cmd) {
	a.servicesLoading = false
	if msg.Err == nil {
		a.services = msg.Services
	}
	a.checkStageComplete()
	return a, nil
}

func (a *App) handleTapsDetail(msg TapsDetailMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.taps = msg.Taps
	}
	a.checkStageComplete()
	return a, nil
}

// checkStageComplete marks Stage 2 as done when all enrichment data is in.
func (a *App) checkStageComplete() {
	if a.stage != StageEnriching {
		return
	}
	// Check if all enrichment data has arrived
	if len(a.formulaeVersions) > 0 && len(a.caskVersions) > 0 && !a.servicesLoading {
		a.stage = StageComplete
	}
}

// --- On-demand detail handlers ---

func (a *App) handleFormulaInfo(msg FormulaInfoMsg) (tea.Model, tea.Cmd) {
	a.detailLoading = false
	if msg.Err != nil {
		a.detailInfo = fmt.Sprintf("Error loading %s: %v", msg.Name, msg.Err)
		return a, nil
	}
	rendered := a.renderFormulaDetail(msg.Formula, msg.Deps, msg.Uses)
	a.detailInfo = rendered
	a.detailScroll = 0
	// Cache it
	a.detailCache["formula:"+msg.Name] = rendered
	return a, nil
}

func (a *App) handleCaskInfo(msg CaskInfoMsg) (tea.Model, tea.Cmd) {
	a.detailLoading = false
	if msg.Err != nil {
		a.detailInfo = fmt.Sprintf("Error loading %s: %v", msg.Name, msg.Err)
		return a, nil
	}
	rendered := a.renderCaskDetail(msg.Cask)
	a.detailInfo = rendered
	a.detailScroll = 0
	a.detailCache["cask:"+msg.Name] = rendered
	return a, nil
}

// --- Action result handlers ---

func (a *App) handleCommandDone(msg CommandDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		a.addLog(cmdLogSuccess.Render("✓") + " " + msg.Command + " completed")
	} else {
		errStr := ""
		if msg.Err != nil {
			errStr = ": " + msg.Err.Error()
		}
		a.addLog(cmdLogError.Render("✗") + " " + msg.Command + " failed" + errStr)
	}
	if msg.Output != "" {
		for _, line := range strings.Split(strings.TrimSpace(msg.Output), "\n") {
			if line != "" {
				a.addLog("  " + line)
			}
		}
	}
	// Refresh data after command — clear cache
	a.detailCache = make(map[string]string)
	return a, func() tea.Msg { return RefreshMsg{} }
}

func (a *App) handleSearchResult(msg SearchResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.errMsg = fmt.Sprintf("Search error: %v", msg.Err)
		return a, nil
	}
	a.searchResults = make([]string, 0)
	for _, f := range msg.Formulae {
		a.searchResults = append(a.searchResults, "[formula] "+f)
	}
	for _, c := range msg.Casks {
		a.searchResults = append(a.searchResults, "[cask] "+c)
	}
	a.searchCursor = 0
	return a, nil
}

func (a *App) handleRefresh() (tea.Model, tea.Cmd) {
	// Reset to Stage 1 for a full refresh
	a.stage = StageLoading
	a.formulaeLoading = true
	a.casksLoading = true
	a.tapsLoading = true
	a.formulaeVersions = make(map[string]string)
	a.caskVersions = make(map[string]string)
	a.outdatedFormulae = make(map[string]bool)
	a.outdatedCasks = make(map[string]bool)
	a.leaves = make(map[string]bool)
	return a, tea.Batch(
		loadFormulaeNames(a.cmds),
		loadCaskNames(a.cmds),
		loadTapNames(a.cmds),
		spinnerTick(),
	)
}
