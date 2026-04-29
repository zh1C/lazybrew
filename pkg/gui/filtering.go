package gui

import (
	"strings"

	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// ============================================================
// Data filtering — work on name lists, not full objects
// ============================================================

func (a *App) getFilteredFormulaeNames() []string {
	var list []string
	switch a.formulaeTab {
	case FormulaeTabInstalled:
		list = a.formulaeNames
	case FormulaeTabOutdated:
		for _, name := range a.formulaeNames {
			if a.outdatedFormulae[name] {
				list = append(list, name)
			}
		}
	case FormulaeTabLeaves:
		for _, name := range a.formulaeNames {
			if a.leaves[name] {
				list = append(list, name)
			}
		}
	}

	if a.filterText == "" {
		return list
	}
	filtered := make([]string, 0)
	lower := strings.ToLower(a.filterText)
	for _, name := range list {
		if strings.Contains(strings.ToLower(name), lower) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func (a *App) getFilteredCaskNames() []string {
	var list []string
	switch a.caskTab {
	case CaskTabInstalled:
		list = a.caskNames
	case CaskTabOutdated:
		for _, name := range a.caskNames {
			if a.outdatedCasks[name] {
				list = append(list, name)
			}
		}
	}

	if a.filterText == "" {
		return list
	}
	filtered := make([]string, 0)
	lower := strings.ToLower(a.filterText)
	for _, name := range list {
		if strings.Contains(strings.ToLower(name), lower) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func (a *App) getFilteredTapNames() []string {
	if a.filterText == "" {
		return a.tapNames
	}
	filtered := make([]string, 0)
	lower := strings.ToLower(a.filterText)
	for _, name := range a.tapNames {
		if strings.Contains(strings.ToLower(name), lower) {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func (a *App) getFilteredServices() []models.Service {
	var list []models.Service
	switch a.serviceTab {
	case ServiceTabAll:
		list = a.services
	case ServiceTabRunning:
		for _, s := range a.services {
			if s.IsRunning() {
				list = append(list, s)
			}
		}
	case ServiceTabStopped:
		for _, s := range a.services {
			if !s.IsRunning() {
				list = append(list, s)
			}
		}
	}

	if a.filterText == "" {
		return list
	}
	filtered := make([]models.Service, 0)
	lower := strings.ToLower(a.filterText)
	for _, s := range list {
		if strings.Contains(strings.ToLower(s.Name), lower) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}
