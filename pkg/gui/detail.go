package gui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// ============================================================
// Detail loading & rendering — on-demand with cache
// (like lazygit's per-item info loading pattern)
// ============================================================

// loadDetailForCurrentItem triggers on-demand detail loading for the
// currently selected item, using cache when available.
func (a *App) loadDetailForCurrentItem() tea.Cmd {
	switch a.activePanel {
	case FormulaePanel:
		items := a.getFilteredFormulaeNames()
		if a.formulaeCursor < len(items) {
			name := items[a.formulaeCursor]
			// Check cache first
			if cached, ok := a.detailCache["formula:"+name]; ok {
				a.detailInfo = cached
				a.detailScroll = 0
				return nil
			}
			a.detailLoading = true
			a.detailInfo = "" // clear while loading
			return loadFormulaInfo(a.cmds, name)
		}
	case CasksPanel:
		items := a.getFilteredCaskNames()
		if a.casksCursor < len(items) {
			name := items[a.casksCursor]
			if cached, ok := a.detailCache["cask:"+name]; ok {
				a.detailInfo = cached
				a.detailScroll = 0
				return nil
			}
			a.detailLoading = true
			a.detailInfo = ""
			return loadCaskInfo(a.cmds, name)
		}
	case TapsPanel:
		items := a.getFilteredTapNames()
		if a.tapsCursor < len(items) {
			name := items[a.tapsCursor]
			// Taps use locally available data (no brew command needed)
			a.detailInfo = a.renderTapDetail(name)
			a.detailScroll = 0
		}
	case ServicesPanel:
		svcs := a.getFilteredServices()
		if a.servicesCursor < len(svcs) {
			svc := svcs[a.servicesCursor]
			a.detailInfo = a.renderServiceDetail(&svc)
			a.detailScroll = 0
		}
	case StatusPanel:
		a.detailInfo = a.renderStatusDetail()
		a.detailScroll = 0
	}
	return nil
}

// ============================================================
// Detail Renderers
// ============================================================

func (a *App) renderFormulaDetail(f *models.Formula, deps string, uses []string) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render(f.Name)
	b.WriteString(title + "\n\n")

	fields := []struct{ key, val string }{
		{"Version", f.CurrentVersion()},
		{"Description", f.Desc},
		{"Homepage", f.Homepage},
		{"License", f.License},
	}
	if f.Pinned {
		fields = append(fields, struct{ key, val string }{"Status", pinnedStyle.Render("Pinned")})
	}
	if f.Outdated {
		fields = append(fields, struct{ key, val string }{"Update", outdatedStyle.Render("* Update available")})
	}
	if f.KegOnly {
		fields = append(fields, struct{ key, val string }{"Keg-only", "Yes"})
	}
	for _, field := range fields {
		if field.val != "" {
			b.WriteString(detailKeyStyle.Render(field.key+":") + " " + detailValueStyle.Render(field.val) + "\n")
		}
	}
	if deps != "" {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Dependencies:") + "\n")
		b.WriteString(deps)
	}
	if len(uses) > 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Used by:") + "\n")
		for _, u := range uses {
			b.WriteString("  " + u + "\n")
		}
	}
	return b.String()
}

func (a *App) renderCaskDetail(c *models.Cask) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render(c.Name)
	b.WriteString(title + "\n\n")

	fields := []struct{ key, val string }{
		{"Token", c.FullToken},
		{"Version", c.Version},
		{"Description", c.Desc},
		{"Homepage", c.Homepage},
	}
	if c.Installed != "" {
		fields = append(fields, struct{ key, val string }{"Installed", c.Installed})
	}
	if c.Outdated {
		fields = append(fields, struct{ key, val string }{"Update", outdatedStyle.Render("* Update available")})
	}
	if c.AutoUpdates {
		fields = append(fields, struct{ key, val string }{"Auto-update", "Yes"})
	}
	for _, field := range fields {
		if field.val != "" {
			b.WriteString(detailKeyStyle.Render(field.key+":") + " " + detailValueStyle.Render(field.val) + "\n")
		}
	}
	return b.String()
}

func (a *App) renderTapDetail(name string) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render(name)
	b.WriteString(title + "\n\n")

	// Try to find the tap in detailed taps data
	for _, t := range a.taps {
		if t.Name == name {
			fields := []struct{ key, val string }{
				{"Remote", t.Remote},
				{"Formulae", fmt.Sprintf("%d", t.FormulaCount)},
				{"Casks", fmt.Sprintf("%d", t.CaskCount)},
			}
			if t.Official {
				fields = append(fields, struct{ key, val string }{"Type", "Official"})
			} else {
				fields = append(fields, struct{ key, val string }{"Type", "Third-party"})
			}
			for _, field := range fields {
				if field.val != "" {
					b.WriteString(detailKeyStyle.Render(field.key+":") + " " + detailValueStyle.Render(field.val) + "\n")
				}
			}
			return b.String()
		}
	}

	b.WriteString(dimItemStyle.Render("Loading tap details..."))
	return b.String()
}

func (a *App) renderServiceDetail(s *models.Service) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render(s.Name)
	b.WriteString(title + "\n\n")

	statusStr := string(s.Status)
	if s.IsRunning() {
		statusStr = runningStyle.Render("Running")
	} else {
		statusStr = stoppedStyle.Render("Stopped")
	}

	fields := []struct{ key, val string }{
		{"Status", statusStr},
		{"User", s.User},
		{"File", s.File},
	}
	for _, field := range fields {
		if field.val != "" {
			b.WriteString(detailKeyStyle.Render(field.key+":") + " " + detailValueStyle.Render(field.val) + "\n")
		}
	}
	return b.String()
}

func (a *App) renderStatusDetail() string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render("Homebrew Status")
	b.WriteString(title + "\n\n")

	b.WriteString(detailKeyStyle.Render("Formulae:") + " " + fmt.Sprintf("%d installed", len(a.formulaeNames)) + "\n")
	b.WriteString(detailKeyStyle.Render("Casks:") + " " + fmt.Sprintf("%d installed", len(a.caskNames)) + "\n")
	b.WriteString(detailKeyStyle.Render("Taps:") + " " + fmt.Sprintf("%d", len(a.tapNames)) + "\n")
	b.WriteString(detailKeyStyle.Render("Services:") + " " + fmt.Sprintf("%d", len(a.services)) + "\n")

	outdatedCount := len(a.outdatedFormulae) + len(a.outdatedCasks)
	if outdatedCount > 0 {
		b.WriteString(detailKeyStyle.Render("Outdated:") + " " + outdatedStyle.Render(fmt.Sprintf("%d packages", outdatedCount)) + "\n")
	} else if a.stage == StageComplete {
		b.WriteString(detailKeyStyle.Render("Outdated:") + " " + cmdLogSuccess.Render("All up to date") + "\n")
	} else {
		b.WriteString(detailKeyStyle.Render("Outdated:") + " " + dimItemStyle.Render("checking...") + "\n")
	}

	b.WriteString(detailKeyStyle.Render("Leaves:") + " " + fmt.Sprintf("%d formulae", len(a.leaves)) + "\n")

	if a.stage != StageComplete {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(warningColor).Render("Loading additional data..."))
	}

	return b.String()
}
