package gui

import (
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// ============================================================
// Stage 1 Messages — instant name lists (~0.03s each)
// ============================================================

// FormulaeNamesMsg arrives when formula names are loaded (Stage 1).
type FormulaeNamesMsg struct {
	Names []string
	Err   error
}

// CaskNamesMsg arrives when cask names are loaded (Stage 1).
type CaskNamesMsg struct {
	Names []string
	Err   error
}

// TapNamesMsg arrives when tap names are loaded (Stage 1).
type TapNamesMsg struct {
	Names []string
	Err   error
}

// ============================================================
// Stage 2 Messages — background enrichment (~1-2s each)
// ============================================================

// FormulaeVersionsMsg arrives when formula name→version map is loaded.
type FormulaeVersionsMsg struct {
	Versions map[string]string
	Err      error
}

// CaskVersionsMsg arrives when cask name→version map is loaded.
type CaskVersionsMsg struct {
	Versions map[string]string
	Err      error
}

// OutdatedNamesMsg arrives when outdated package names are loaded.
type OutdatedNamesMsg struct {
	Formulae []string
	Casks    []string
	Err      error
}

// LeavesLoadedMsg arrives when leaf formula names are loaded.
type LeavesLoadedMsg struct {
	Leaves []string
	Err    error
}

// ServicesLoadedMsg arrives when services list is loaded.
type ServicesLoadedMsg struct {
	Services []models.Service
	Err      error
}

// TapsDetailMsg arrives when detailed tap info is loaded.
type TapsDetailMsg struct {
	Taps []models.Tap
	Err  error
}

// ============================================================
// On-demand detail messages (triggered by user selection)
// ============================================================

// FormulaInfoMsg arrives when a single formula's detail is loaded.
type FormulaInfoMsg struct {
	Name    string
	Formula *models.Formula
	Deps    string
	Uses    []string
	Err     error
}

// CaskInfoMsg arrives when a single cask's detail is loaded.
type CaskInfoMsg struct {
	Name string
	Cask *models.Cask
	Err  error
}

// ============================================================
// Action messages
// ============================================================

// CommandDoneMsg is sent when a brew action command completes.
type CommandDoneMsg struct {
	Command string
	Output  string
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

// RefreshMsg triggers a full data refresh (Stage 1 + Stage 2).
type RefreshMsg struct{}

// SpinnerTickMsg triggers spinner animation update.
type SpinnerTickMsg struct{}
