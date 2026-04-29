package gui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zh1C/lazybrew/pkg/commands/brew"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// ============================================================
// Detail loading & rendering — cache-first, brew-fallback
// (like lazygit's per-item info loading pattern)
//
// New architecture:
//   - Formula/Cask details rendered INSTANTLY from API cache (0ms)
//   - Only falls back to brew commands if package not in cache
//   - No more detailCache map needed — data is always available
// ============================================================

// loadDetailForCurrentItem renders detail from cache data (instant)
// or triggers async brew command if cache miss.
func (a *App) loadDetailForCurrentItem() tea.Cmd {
	switch a.activePanel {
	case FormulaePanel:
		items := a.getFilteredFormulaeNames()
		if a.formulaeCursor < len(items) {
			name := items[a.formulaeCursor]
			// Try cache first (instant, 0ms)
			if a.formulaCache != nil {
				if f, ok := a.formulaCache.ByName[name]; ok {
					a.detailInfo = a.renderFormulaCacheDetail(name, f)
					a.detailScroll = 0
					return nil
				}
			}
			// Cache miss — fallback to brew command
			a.detailInfo = dimItemStyle.Render(" Loading " + name + "...")
			return loadFormulaInfo(a.cmds, name)
		}
	case CasksPanel:
		items := a.getFilteredCaskNames()
		if a.casksCursor < len(items) {
			name := items[a.casksCursor]
			// Try cache first (instant, 0ms)
			if a.caskCache != nil {
				if c, ok := a.caskCache.ByToken[name]; ok {
					a.detailInfo = a.renderCaskCacheDetail(name, c)
					a.detailScroll = 0
					return nil
				}
			}
			// Cache miss — fallback
			a.detailInfo = dimItemStyle.Render(" Loading " + name + "...")
			return loadCaskInfo(a.cmds, name)
		}
	case TapsPanel:
		items := a.getFilteredTapNames()
		if a.tapsCursor < len(items) {
			name := items[a.tapsCursor]
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
// Cache-based detail renderers (instant, no brew calls)
// ============================================================

func (a *App) renderFormulaCacheDetail(name string, f *brew.APICacheFormula) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render(f.Name)
	b.WriteString(title + "\n\n")

	// Version info
	version := a.formulaeVersions[name]
	if version == "" {
		version = f.Versions.Stable
	}

	fields := []struct{ key, val string }{
		{"Version", version},
		{"Description", f.Desc},
		{"Homepage", f.Homepage},
		{"License", f.License},
	}
	if f.KegOnly {
		fields = append(fields, struct{ key, val string }{"Keg-only", "Yes"})
	}
	if a.outdatedFormulae[name] {
		fields = append(fields, struct{ key, val string }{"Update", outdatedStyle.Render("* " + f.Versions.Stable + " available")})
	}
	if f.Deprecated {
		fields = append(fields, struct{ key, val string }{"Status", outdatedStyle.Render("Deprecated")})
	}
	if f.Caveats != nil && *f.Caveats != "" {
		// Truncate long caveats
		caveats := *f.Caveats
		if len(caveats) > 200 {
			caveats = caveats[:200] + "..."
		}
		fields = append(fields, struct{ key, val string }{"Caveats", caveats})
	}

	for _, field := range fields {
		if field.val != "" {
			b.WriteString(detailKeyStyle.Render(field.key+":") + " " + detailValueStyle.Render(field.val) + "\n")
		}
	}

	// Dependencies (from cache — instant!)
	if len(f.Dependencies) > 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Dependencies:") + "\n")
		for _, dep := range f.Dependencies {
			prefix := "  "
			// Mark installed deps
			if a.formulaeVersions[dep] != "" {
				prefix = "  " + cmdLogSuccess.Render("*") + " "
			}
			b.WriteString(prefix + dep + "\n")
		}
	}

	// Reverse dependencies (from our computed map — instant!)
	if a.reverseDeps != nil {
		if deps, ok := a.reverseDeps[name]; ok && len(deps) > 0 {
			b.WriteString("\n" + lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Used by:") + "\n")
			for _, d := range deps {
				b.WriteString("  " + d + "\n")
			}
		}
	}

	// Install info from receipt
	if a.receipts != nil {
		if ri, ok := a.receipts[name]; ok {
			b.WriteString("\n" + lipgloss.NewStyle().Foreground(accentColor).Bold(true).Render("Install info:") + "\n")
			if ri.InstalledOnRequest {
				b.WriteString("  Installed on request\n")
			} else {
				b.WriteString("  Installed as dependency\n")
			}
			if ri.PouredFromBottle {
				b.WriteString("  Poured from bottle\n")
			}
		}
	}

	return b.String()
}

func (a *App) renderCaskCacheDetail(name string, c *brew.APICacheCask) string {
	var b strings.Builder

	displayName := c.Token
	if len(c.Name) > 0 {
		displayName = c.Name[0]
	}
	title := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true).Render(displayName)
	b.WriteString(title + "\n\n")

	version := a.caskVersions[name]
	if version == "" {
		version = c.Version
	}

	fields := []struct{ key, val string }{
		{"Token", c.Token},
		{"Version", version},
		{"Description", c.Desc},
		{"Homepage", c.Homepage},
	}
	if a.outdatedCasks[name] {
		fields = append(fields, struct{ key, val string }{"Update", outdatedStyle.Render("* " + c.Version + " available")})
	}
	if c.AutoUpdates {
		fields = append(fields, struct{ key, val string }{"Auto-update", "Yes"})
	}
	if c.Deprecated {
		fields = append(fields, struct{ key, val string }{"Status", outdatedStyle.Render("Deprecated")})
	}

	for _, field := range fields {
		if field.val != "" {
			b.WriteString(detailKeyStyle.Render(field.key+":") + " " + detailValueStyle.Render(field.val) + "\n")
		}
	}

	return b.String()
}

// ============================================================
// Brew-based detail renderers (fallback for cache misses)
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

// ============================================================
// Other panel detail renderers
// ============================================================

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

	// Minimal info from tap name
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 2 {
		b.WriteString(detailKeyStyle.Render("User:") + " " + detailValueStyle.Render(parts[0]) + "\n")
		b.WriteString(detailKeyStyle.Render("Repo:") + " " + detailValueStyle.Render(parts[1]) + "\n")
	}
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
