package gui

import (
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// --- Tea Messages ---

// DataLoadedMsg is sent when initial data loading completes.
type DataLoadedMsg struct {
	Formulae []models.Formula
	Casks    []models.Cask
	Services []models.Service
	Taps     []models.Tap
	Err      error
}

// FormulaInfoMsg is sent when formula detail info is loaded.
type FormulaInfoMsg struct {
	Formula *models.Formula
	Deps    string
	Uses    []string
	Err     error
}

// CaskInfoMsg is sent when cask detail info is loaded.
type CaskInfoMsg struct {
	Cask *models.Cask
	Err  error
}

// CommandOutputMsg is sent when a brew command produces output.
type CommandOutputMsg struct {
	Line string
}

// CommandDoneMsg is sent when a brew command completes.
type CommandDoneMsg struct {
	Command string
	Success bool
	Err     error
}

// SearchResultMsg is sent when search results are ready.
type SearchResultMsg struct {
	Query    string
	Formulae []string
	Casks    []string
	Err      error
}

// RefreshMsg triggers a data refresh.
type RefreshMsg struct{}

// OutdatedLoadedMsg is sent when outdated data is loaded.
type OutdatedLoadedMsg struct {
	Formulae []models.Formula
	Casks    []models.Cask
	Err      error
}

// LeavesLoadedMsg is sent when leaves data is loaded.
type LeavesLoadedMsg struct {
	Leaves []string
	Err    error
}
