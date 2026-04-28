package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zh1C/lazybrew/pkg/commands/brew"
)

// ============================================================
// User actions — brew operation methods
// (like lazygit's gui/controllers/*.go)
// ============================================================

func (a *App) startSearch() (tea.Model, tea.Cmd) {
	a.overlay = OverlaySearch
	a.searchInput.Focus()
	a.searchInput.Reset()
	a.searchResults = nil
	return a, textinput.Blink
}

func (a *App) uninstallSelected() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		name := a.selectedFormulaName()
		if name == "" {
			return a, nil
		}
		a.confirmMsg = fmt.Sprintf("Uninstall formula '%s'?", name)
		a.confirmAction = func() tea.Cmd {
			a.addLog(cmdLogPrefix.Render("$") + " brew uninstall " + name)
			return runBrewCommand("brew uninstall "+name, func(cb func(string)) brew.CommandResult {
				return a.cmds.Formula.Uninstall(name, cb)
			})
		}
		a.overlay = OverlayConfirm

	case CasksPanel:
		name := a.selectedCaskName()
		if name == "" {
			return a, nil
		}
		a.confirmMsg = fmt.Sprintf("Uninstall cask '%s'?", name)
		a.confirmAction = func() tea.Cmd {
			a.addLog(cmdLogPrefix.Render("$") + " brew uninstall --cask " + name)
			return runBrewCommand("brew uninstall --cask "+name, func(cb func(string)) brew.CommandResult {
				return a.cmds.Cask.Uninstall(name, cb)
			})
		}
		a.overlay = OverlayConfirm
	}
	return a, nil
}

func (a *App) upgradeSelected() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		name := a.selectedFormulaName()
		if name == "" {
			return a, nil
		}
		a.addLog(cmdLogPrefix.Render("$") + " brew upgrade " + name)
		return a, runBrewCommand("brew upgrade "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Formula.Upgrade(name, cb)
		})
	case CasksPanel:
		name := a.selectedCaskName()
		if name == "" {
			return a, nil
		}
		a.addLog(cmdLogPrefix.Render("$") + " brew upgrade --cask " + name)
		return a, runBrewCommand("brew upgrade --cask "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Cask.Upgrade(name, cb)
		})
	}
	return a, nil
}

func (a *App) handleReinstartOrReinstall() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case ServicesPanel:
		return a.restartService()
	case FormulaePanel:
		name := a.selectedFormulaName()
		if name == "" {
			return a, nil
		}
		a.addLog(cmdLogPrefix.Render("$") + " brew reinstall " + name)
		return a, runBrewCommand("brew reinstall "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Formula.Reinstall(name, cb)
		})
	}
	return a, nil
}

func (a *App) pinSelected() (tea.Model, tea.Cmd) {
	if a.activePanel != FormulaePanel {
		return a, nil
	}
	name := a.selectedFormulaName()
	if name == "" {
		return a, nil
	}
	a.addLog(cmdLogPrefix.Render("$") + " brew pin " + name)
	result := a.cmds.Formula.Pin(name)
	if result.Err == nil {
		a.addLog(cmdLogSuccess.Render("✓") + " " + name + " pinned")
	} else {
		a.addLog(cmdLogError.Render("✗") + " pin failed: " + result.Err.Error())
	}
	return a, nil
}

func (a *App) unpinSelected() (tea.Model, tea.Cmd) {
	if a.activePanel != FormulaePanel {
		return a, nil
	}
	name := a.selectedFormulaName()
	if name == "" {
		return a, nil
	}
	a.addLog(cmdLogPrefix.Render("$") + " brew unpin " + name)
	result := a.cmds.Formula.Unpin(name)
	if result.Err == nil {
		a.addLog(cmdLogSuccess.Render("✓") + " " + name + " unpinned")
	} else {
		a.addLog(cmdLogError.Render("✗") + " unpin failed: " + result.Err.Error())
	}
	return a, nil
}

func (a *App) openHomepage() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		name := a.selectedFormulaName()
		if name != "" {
			a.cmds.Runner.Run("home", name)
		}
	case CasksPanel:
		name := a.selectedCaskName()
		if name != "" {
			a.cmds.Runner.Run("home", "--cask", name)
		}
	}
	return a, nil
}

func (a *App) startService() (tea.Model, tea.Cmd) {
	if a.activePanel != ServicesPanel {
		return a, nil
	}
	svcs := a.getFilteredServices()
	if a.servicesCursor >= len(svcs) {
		return a, nil
	}
	name := svcs[a.servicesCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew services start " + name)
	return a, runBrewCommand("brew services start "+name, func(cb func(string)) brew.CommandResult {
		return a.cmds.Service.Start(name, cb)
	})
}

func (a *App) stopService() (tea.Model, tea.Cmd) {
	if a.activePanel != ServicesPanel {
		return a, nil
	}
	svcs := a.getFilteredServices()
	if a.servicesCursor >= len(svcs) {
		return a, nil
	}
	name := svcs[a.servicesCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew services stop " + name)
	return a, runBrewCommand("brew services stop "+name, func(cb func(string)) brew.CommandResult {
		return a.cmds.Service.Stop(name, cb)
	})
}

func (a *App) restartService() (tea.Model, tea.Cmd) {
	if a.activePanel != ServicesPanel {
		return a, nil
	}
	svcs := a.getFilteredServices()
	if a.servicesCursor >= len(svcs) {
		return a, nil
	}
	name := svcs[a.servicesCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew services restart " + name)
	return a, runBrewCommand("brew services restart "+name, func(cb func(string)) brew.CommandResult {
		return a.cmds.Service.Restart(name, cb)
	})
}

func (a *App) zapCask() (tea.Model, tea.Cmd) {
	if a.activePanel != CasksPanel {
		return a, nil
	}
	name := a.selectedCaskName()
	if name == "" {
		return a, nil
	}
	a.confirmMsg = fmt.Sprintf("Zap cask '%s'? This removes ALL associated files!", name)
	a.confirmAction = func() tea.Cmd {
		a.addLog(cmdLogPrefix.Render("$") + " brew uninstall --cask --zap " + name)
		return runBrewCommand("brew zap "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Cask.Zap(name, cb)
		})
	}
	a.overlay = OverlayConfirm
	return a, nil
}

func (a *App) brewUpdate() (tea.Model, tea.Cmd) {
	a.addLog(cmdLogPrefix.Render("$") + " brew update")
	return a, runBrewCommand("brew update", func(cb func(string)) brew.CommandResult {
		return a.cmds.System.Update(cb)
	})
}

func (a *App) brewCleanup() (tea.Model, tea.Cmd) {
	a.confirmMsg = "Run 'brew cleanup --prune=all'?"
	a.confirmAction = func() tea.Cmd {
		a.addLog(cmdLogPrefix.Render("$") + " brew cleanup --prune=all")
		return runBrewCommand("brew cleanup", func(cb func(string)) brew.CommandResult {
			return a.cmds.System.Cleanup(cb)
		})
	}
	a.overlay = OverlayConfirm
	return a, nil
}

func (a *App) brewDoctor() (tea.Model, tea.Cmd) {
	a.addLog(cmdLogPrefix.Render("$") + " brew doctor")
	return a, runBrewCommand("brew doctor", func(cb func(string)) brew.CommandResult {
		return a.cmds.System.Doctor(cb)
	})
}

func (a *App) brewAutoremove() (tea.Model, tea.Cmd) {
	a.confirmMsg = "Run 'brew autoremove'?"
	a.confirmAction = func() tea.Cmd {
		a.addLog(cmdLogPrefix.Render("$") + " brew autoremove")
		return runBrewCommand("brew autoremove", func(cb func(string)) brew.CommandResult {
			return a.cmds.System.Autoremove(cb)
		})
	}
	a.overlay = OverlayConfirm
	return a, nil
}

func (a *App) installSearchResult() (tea.Model, tea.Cmd) {
	if a.searchCursor >= len(a.searchResults) {
		return a, nil
	}
	selected := a.searchResults[a.searchCursor]
	name := selected
	if strings.HasPrefix(name, "[formula] ") {
		name = strings.TrimPrefix(name, "[formula] ")
		a.overlay = OverlayNone
		a.addLog(cmdLogPrefix.Render("$") + " brew install " + name)
		return a, runBrewCommand("brew install "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Formula.Install(name, cb)
		})
	} else if strings.HasPrefix(name, "[cask] ") {
		name = strings.TrimPrefix(name, "[cask] ")
		a.overlay = OverlayNone
		a.addLog(cmdLogPrefix.Render("$") + " brew install --cask " + name)
		return a, runBrewCommand("brew install --cask "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Cask.Install(name, cb)
		})
	}
	return a, nil
}
