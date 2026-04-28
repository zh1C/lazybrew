package gui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zh1C/lazybrew/pkg/commands/brew"
)

// loadAllData fetches all homebrew data concurrently.
func loadAllData(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		formulae, fErr := cmds.Formula.ListInstalled()
		if fErr != nil {
			return DataLoadedMsg{Err: fErr}
		}

		casks, cErr := cmds.Cask.ListInstalled()
		if cErr != nil {
			return DataLoadedMsg{Formulae: formulae, Err: cErr}
		}

		services, sErr := cmds.Service.List()
		if sErr != nil {
			// Services may not be available, don't fail
			services = nil
		}

		taps, tErr := cmds.Tap.List()
		if tErr != nil {
			taps = nil
		}

		return DataLoadedMsg{
			Formulae: formulae,
			Casks:    casks,
			Services: services,
			Taps:     taps,
		}
	}
}

// loadFormulaInfo fetches detailed info for a formula.
func loadFormulaInfo(cmds *brew.BrewCommands, name string) tea.Cmd {
	return func() tea.Msg {
		formula, err := cmds.Formula.Info(name)
		if err != nil {
			return FormulaInfoMsg{Err: err}
		}

		deps, _ := cmds.Formula.Deps(name)
		uses, _ := cmds.Formula.Uses(name)

		return FormulaInfoMsg{
			Formula: formula,
			Deps:    deps,
			Uses:    uses,
		}
	}
}

// loadCaskInfo fetches detailed info for a cask.
func loadCaskInfo(cmds *brew.BrewCommands, name string) tea.Cmd {
	return func() tea.Msg {
		cask, err := cmds.Cask.Info(name)
		if err != nil {
			return CaskInfoMsg{Err: err}
		}
		return CaskInfoMsg{Cask: cask}
	}
}

// loadOutdated fetches outdated packages.
func loadOutdated(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		formulae, fErr := cmds.Formula.ListOutdated()
		casks, cErr := cmds.Cask.ListOutdated()
		if fErr != nil {
			return OutdatedLoadedMsg{Err: fErr}
		}
		if cErr != nil {
			return OutdatedLoadedMsg{Formulae: formulae, Err: cErr}
		}
		return OutdatedLoadedMsg{Formulae: formulae, Casks: casks}
	}
}

// loadLeaves fetches leaf packages.
func loadLeaves(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		leaves, err := cmds.Formula.ListLeaves()
		return LeavesLoadedMsg{Leaves: leaves, Err: err}
	}
}

// runBrewCommand runs a brew command that produces live output.
func runBrewCommand(cmdName string, cmdFunc func(func(string)) brew.CommandResult) tea.Cmd {
	return func() tea.Msg {
		result := cmdFunc(func(line string) {
			// Note: in bubbletea we can't send messages from within a Cmd callback
			// The output will be captured and displayed after completion
		})
		return CommandDoneMsg{
			Command: cmdName,
			Success: result.Err == nil,
			Err:     result.Err,
		}
	}
}

// searchPackages searches for both formulae and casks.
func searchPackages(cmds *brew.BrewCommands, query string) tea.Cmd {
	return func() tea.Msg {
		formulae, fErr := cmds.Formula.Search(query)
		casks, cErr := cmds.Cask.Search(query)
		if fErr != nil && cErr != nil {
			return SearchResultMsg{Query: query, Err: fErr}
		}
		return SearchResultMsg{
			Query:    query,
			Formulae: formulae,
			Casks:    casks,
		}
	}
}
