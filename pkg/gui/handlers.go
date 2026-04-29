package gui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// ============================================================
// Message handlers — all tea.Msg → state update logic
// ============================================================

// --- Stage 1 Handlers (instant from file system) ---

func (a *App) handleFormulaeNames(msg FormulaeNamesMsg) (tea.Model, tea.Cmd) {
	a.formulaeLoading = false
	if msg.Err != nil {
		a.errMsg = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil
	}
	a.formulaeNames = msg.Names
	if msg.Versions != nil {
		a.formulaeVersions = msg.Versions
	}
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
	if msg.Versions != nil {
		a.caskVersions = msg.Versions
	}
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
// If so, transitions to Stage 2 and fires parallel enrichment:
//   - API cache loading (~0.2s) — formula/cask metadata, deps, leaves
//   - brew outdated (~1.5s) — needs precise version comparison
//   - brew services (~1.5s) — needs launchctl interaction
func (a *App) maybeStartStage2() tea.Cmd {
	// Still waiting for some Stage 1 data
	if a.formulaeLoading || a.casksLoading || a.tapsLoading {
		return nil
	}

	// Stage 1 complete → start Stage 2
	a.stage = StageEnriching
	a.cacheLoading = true
	a.servicesLoading = true
	a.stage2CacheDone = false
	a.stage2OutdatedDone = false
	a.stage2ServicesDone = false

	return tea.Batch(
		loadAPICache(a.cmds, a.formulaeNames), // ~0.2s (file system)
		loadOutdatedNames(a.cmds),             // ~1.5s (brew command, kept)
		loadServices(a.cmds),                  // ~1.5s (brew command, kept)
		spinnerTick(),                         // keep spinner alive
	)
}

// --- Stage 2 Handlers ---

func (a *App) handleCacheLoaded(msg CacheLoadedMsg) (tea.Model, tea.Cmd) {
	a.cacheLoading = false
	a.stage2CacheDone = true

	if msg.Err != nil {
		// Cache failed — not critical, detail will fallback to brew commands
		a.addLog(cmdLogError.Render("!") + " Cache load failed, using brew fallback")
	} else {
		a.formulaCache = msg.FormulaCache
		a.caskCache = msg.CaskCache
		a.receipts = msg.Receipts
		a.reverseDeps = msg.ReverseDeps
		if msg.Leaves != nil {
			a.leaves = msg.Leaves
		}
	}

	a.checkStageComplete()
	return a, nil
}

func (a *App) handleOutdatedNames(msg OutdatedNamesMsg) (tea.Model, tea.Cmd) {
	a.stage2OutdatedDone = true
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

func (a *App) handleServices(msg ServicesLoadedMsg) (tea.Model, tea.Cmd) {
	a.servicesLoading = false
	a.stage2ServicesDone = true
	if msg.Err == nil {
		a.services = msg.Services
	}
	a.checkStageComplete()
	return a, nil
}

// checkStageComplete marks Stage 2 as done when all enrichment data is in.
func (a *App) checkStageComplete() {
	if a.stage != StageEnriching {
		return
	}
	if a.stage2CacheDone && a.stage2OutdatedDone && a.stage2ServicesDone {
		a.stage = StageComplete
	}
}

// --- On-demand detail handlers (fallback when cache misses) ---

func (a *App) handleFormulaInfo(msg FormulaInfoMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.detailInfo = fmt.Sprintf("Error loading %s: %v", msg.Name, msg.Err)
		return a, nil
	}
	rendered := a.renderFormulaDetail(msg.Formula, msg.Deps, msg.Uses)
	a.detailInfo = rendered
	a.detailScroll = 0
	return a, nil
}

func (a *App) handleCaskInfo(msg CaskInfoMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.detailInfo = fmt.Sprintf("Error loading %s: %v", msg.Name, msg.Err)
		return a, nil
	}
	rendered := a.renderCaskDetail(msg.Cask)
	a.detailInfo = rendered
	a.detailScroll = 0
	return a, nil
}

// --- Action result handlers ---

func (a *App) handleCommandDone(msg CommandDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		a.addLog(cmdLogSuccess.Render("OK") + " " + msg.Command + " completed")
	} else {
		errStr := ""
		if msg.Err != nil {
			errStr = ": " + msg.Err.Error()
		}
		a.addLog(cmdLogError.Render("ERR") + " " + msg.Command + " failed" + errStr)
	}
	if msg.Output != "" {
		for _, line := range strings.Split(strings.TrimSpace(msg.Output), "\n") {
			if line != "" {
				a.addLog("  " + line)
			}
		}
	}
	// Refresh data after command
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
	a.formulaCache = nil
	a.caskCache = nil
	a.receipts = nil
	a.reverseDeps = nil
	return a, tea.Batch(
		loadFormulaeFromFS(a.cmds),
		loadCasksFromFS(a.cmds),
		loadTapsFromFS(a.cmds),
		spinnerTick(),
	)
}
