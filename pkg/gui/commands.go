package gui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zh1C/lazybrew/pkg/commands/brew"
)

// ============================================================
// Stage 1: Instant name lists (each ~0.03s, parallel = ~0.06s)
// ============================================================

func loadFormulaeNames(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		names, err := cmds.Formula.ListNames()
		return FormulaeNamesMsg{Names: names, Err: err}
	}
}

func loadCaskNames(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		names, err := cmds.Cask.ListNames()
		return CaskNamesMsg{Names: names, Err: err}
	}
}

func loadTapNames(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		names, err := cmds.Tap.ListNames()
		return TapNamesMsg{Names: names, Err: err}
	}
}

// ============================================================
// Stage 2: Background enrichment (each ~1-2s, parallel)
// ============================================================

func loadFormulaeVersions(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		versions, err := cmds.Formula.ListNamesWithVersions()
		return FormulaeVersionsMsg{Versions: versions, Err: err}
	}
}

func loadCaskVersions(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		versions, err := cmds.Cask.ListNamesWithVersions()
		return CaskVersionsMsg{Versions: versions, Err: err}
	}
}

func loadOutdatedNames(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		formulae, _ := cmds.Formula.ListOutdatedNames()
		casks, _ := cmds.Cask.ListOutdatedNames()
		return OutdatedNamesMsg{Formulae: formulae, Casks: casks}
	}
}

func loadLeaves(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		leaves, err := cmds.Formula.ListLeaves()
		return LeavesLoadedMsg{Leaves: leaves, Err: err}
	}
}

func loadServices(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		services, err := cmds.Service.List()
		return ServicesLoadedMsg{Services: services, Err: err}
	}
}

func loadTapsDetail(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		taps, err := cmds.Tap.List()
		return TapsDetailMsg{Taps: taps, Err: err}
	}
}

// ============================================================
// On-demand detail loading (triggered by user cursor movement)
// ============================================================

func loadFormulaInfo(cmds *brew.BrewCommands, name string) tea.Cmd {
	return func() tea.Msg {
		formula, err := cmds.Formula.Info(name)
		if err != nil {
			return FormulaInfoMsg{Name: name, Err: err}
		}
		deps, _ := cmds.Formula.Deps(name)
		uses, _ := cmds.Formula.Uses(name)
		return FormulaInfoMsg{
			Name:    name,
			Formula: formula,
			Deps:    deps,
			Uses:    uses,
		}
	}
}

func loadCaskInfo(cmds *brew.BrewCommands, name string) tea.Cmd {
	return func() tea.Msg {
		cask, err := cmds.Cask.Info(name)
		return CaskInfoMsg{Name: name, Cask: cask, Err: err}
	}
}

// ============================================================
// Action commands
// ============================================================

func runBrewCommand(cmdName string, cmdFunc func(func(string)) brew.CommandResult) tea.Cmd {
	return func() tea.Msg {
		var output string
		result := cmdFunc(func(line string) {
			output += line + "\n"
		})
		return CommandDoneMsg{
			Command: cmdName,
			Output:  output,
			Success: result.Err == nil,
			Err:     result.Err,
		}
	}
}

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

// spinnerTick sends a tick message for spinner animation.
func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg{}
	})
}
