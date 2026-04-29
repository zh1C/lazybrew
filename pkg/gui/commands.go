package gui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/zh1C/lazybrew/pkg/commands/brew"
)

// Stage 1: File system reads

func loadFormulaeFromFS(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		names, err := cmds.Cache.InstalledFormulaNames()
		if err != nil {
			// Fallback to brew command
			names, err = cmds.Formula.ListNames()
			if err != nil {
				return FormulaeNamesMsg{Err: err}
			}
		}
		versions, _ := cmds.Cache.InstalledFormulaVersions()
		return FormulaeNamesMsg{Names: names, Versions: versions}
	}
}

func loadCasksFromFS(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		names, err := cmds.Cache.InstalledCaskNames()
		if err != nil {
			names, err = cmds.Cask.ListNames()
			if err != nil {
				return CaskNamesMsg{Err: err}
			}
		}
		versions, _ := cmds.Cache.InstalledCaskVersions()
		return CaskNamesMsg{Names: names, Versions: versions}
	}
}

func loadTapsFromFS(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		names, err := cmds.Cache.TapNames()
		if err != nil {
			names, err = cmds.Tap.ListNames()
			if err != nil {
				return TapNamesMsg{Err: err}
			}
		}
		return TapNamesMsg{Names: names}
	}
}

// Stage 2: API cache + brew commands

// loadAPICache loads formula/cask metadata from local API cache files.
// This provides desc, homepage, license, deps for all 8000+ formulae in ~0.2s.
func loadAPICache(cmds *brew.BrewCommands, installedNames []string) tea.Cmd {
	return func() tea.Msg {
		fc, fcErr := cmds.Cache.LoadFormulaCache()
		cc, ccErr := cmds.Cache.LoadCaskCache()
		receipts, _ := cmds.Cache.AllReceipts()

		if fcErr != nil && ccErr != nil {
			return CacheLoadedMsg{Err: fcErr}
		}

		var reverseDeps map[string][]string
		var leaves map[string]bool

		if fc != nil && len(installedNames) > 0 {
			reverseDeps = brew.BuildReverseDeps(fc, installedNames)
			if receipts != nil {
				leaves = brew.ComputeLeaves(receipts, reverseDeps)
			}
		}

		return CacheLoadedMsg{
			FormulaCache: fc,
			CaskCache:    cc,
			Receipts:     receipts,
			ReverseDeps:  reverseDeps,
			Leaves:       leaves,
		}
	}
}

// loadOutdatedNames still uses brew command (needs precise version comparison).
func loadOutdatedNames(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		formulae, _ := cmds.Formula.ListOutdatedNames()
		casks, _ := cmds.Cask.ListOutdatedNames()
		return OutdatedNamesMsg{Formulae: formulae, Casks: casks}
	}
}

// loadServices still uses brew command (needs launchctl interaction).
func loadServices(cmds *brew.BrewCommands) tea.Cmd {
	return func() tea.Msg {
		services, err := cmds.Service.List()
		return ServicesLoadedMsg{Services: services, Err: err}
	}
}

// On-demand detail loading — fallback for packages not in cache

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

// Action commands

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

// searchFromCache searches the API cache for matching formula/cask names.
func searchFromCache(app *App, query string) tea.Cmd {
	return func() tea.Msg {
		lower := strings.ToLower(query)
		var formulae, casks []string

		// Search formula cache
		if app.formulaCache != nil {
			for _, f := range app.formulaCache.All {
				if strings.Contains(strings.ToLower(f.Name), lower) ||
					strings.Contains(strings.ToLower(f.Desc), lower) {
					formulae = append(formulae, f.Name)
				}
			}
		}

		// Search cask cache
		if app.caskCache != nil {
			for _, c := range app.caskCache.All {
				if strings.Contains(strings.ToLower(c.Token), lower) ||
					strings.Contains(strings.ToLower(c.Desc), lower) {
					casks = append(casks, c.Token)
				}
			}
		}

		// Limit results
		maxResults := 50
		if len(formulae) > maxResults {
			formulae = formulae[:maxResults]
		}
		if len(casks) > maxResults {
			casks = casks[:maxResults]
		}

		return SearchResultMsg{
			Query:    query,
			Formulae: formulae,
			Casks:    casks,
		}
	}
}

// searchPackages falls back to brew search when cache is not available.
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
