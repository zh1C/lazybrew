package gui

import (
	"github.com/zh1C/lazybrew/pkg/commands/brew"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// ============================================================
// Stage 1 Messages — instant file system reads (~0.01s each)
// ============================================================

// FormulaeNamesMsg arrives when formula names are loaded (Stage 1).
type FormulaeNamesMsg struct {
	Names    []string
	Versions map[string]string
	Err      error
}

// CaskNamesMsg arrives when cask names are loaded (Stage 1).
type CaskNamesMsg struct {
	Names    []string
	Versions map[string]string
	Err      error
}

// TapNamesMsg arrives when tap names are loaded (Stage 1).
type TapNamesMsg struct {
	Names []string
	Err   error
}

// ============================================================
// Stage 2 Messages — API cache + background brew commands
// ============================================================

// CacheLoadedMsg arrives when the API cache has been fully parsed.
// Contains formula/cask metadata, reverse deps, leaves, receipts.
type CacheLoadedMsg struct {
	FormulaCache *brew.FormulaCache
	CaskCache    *brew.CaskCache
	Receipts     map[string]*brew.ReceiptInfo
	ReverseDeps  map[string][]string
	Leaves       map[string]bool
	Err          error
}

// OutdatedNamesMsg arrives when outdated package names are loaded (brew command).
type OutdatedNamesMsg struct {
	Formulae []string
	Casks    []string
	Err      error
}

// ServicesLoadedMsg arrives when services list is loaded (brew command).
type ServicesLoadedMsg struct {
	Services []models.Service
	Err      error
}

// ============================================================
// On-demand detail messages (no longer needed for formula/cask
// since we have cache, but kept for fallback)
// ============================================================

// FormulaInfoMsg arrives when a single formula's detail is loaded via brew.
type FormulaInfoMsg struct {
	Name    string
	Formula *models.Formula
	Deps    string
	Uses    []string
	Err     error
}

// CaskInfoMsg arrives when a single cask's detail is loaded via brew.
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
