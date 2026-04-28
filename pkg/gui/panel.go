package gui

import "fmt"

// PanelID identifies each side panel.
type PanelID int

const (
	StatusPanel PanelID = iota
	FormulaePanel
	CasksPanel
	TapsPanel
	ServicesPanel
)

// PanelCount is the total number of side panels.
const PanelCount = 5

// PanelName returns the display name for a panel.
func PanelName(id PanelID) string {
	switch id {
	case StatusPanel:
		return "Status"
	case FormulaePanel:
		return "Formulae"
	case CasksPanel:
		return "Casks"
	case TapsPanel:
		return "Taps"
	case ServicesPanel:
		return "Services"
	default:
		return ""
	}
}

// PanelIndex returns the jump-key prefix like lazygit's [1], [2], etc.
func PanelIndex(id PanelID) string {
	return fmt.Sprintf("[%d]", int(id)+1)
}

// TabID identifies tabs within a panel.
type TabID int

// Formulae tabs
const (
	FormulaeTabInstalled TabID = iota
	FormulaeTabOutdated
	FormulaeTabLeaves
)

// Cask tabs
const (
	CaskTabInstalled TabID = iota
	CaskTabOutdated
)

// Service tabs
const (
	ServiceTabAll TabID = iota
	ServiceTabRunning
	ServiceTabStopped
)
