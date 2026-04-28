package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zh1C/lazybrew/pkg/commands/brew"
	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// FocusArea indicates which major area is focused.
type FocusArea int

const (
	FocusSidePanel FocusArea = iota
	FocusMainPanel
)

// OverlayMode indicates what overlay is currently displayed.
type OverlayMode int

const (
	OverlayNone OverlayMode = iota
	OverlaySearch
	OverlayConfirm
	OverlayHelp
)

// App is the main bubbletea model.
type App struct {
	// Brew commands
	cmds *brew.BrewCommands

	// Terminal dimensions
	width  int
	height int

	// Panel state
	activePanel PanelID
	focusArea   FocusArea

	// Tab state per panel
	formulaeTab TabID
	caskTab     TabID
	serviceTab  TabID

	// Data
	formulae         []models.Formula
	casks            []models.Cask
	services         []models.Service
	taps             []models.Tap
	outdatedFormulae []models.Formula
	outdatedCasks    []models.Cask
	leaves           []string

	// List cursors (per panel/tab)
	formulaeCursor  int
	casksCursor     int
	tapsCursor      int
	servicesCursor  int
	searchCursor    int

	// Detail panel state
	detailInfo   string
	detailScroll int

	// Command log
	commandLog   []string
	maxLogLines  int

	// Overlay state
	overlay       OverlayMode
	searchInput   textinput.Model
	searchResults []string
	searchType    string // "formula" or "cask"
	confirmMsg    string
	confirmAction func() tea.Cmd

	// Loading state
	loading    bool
	loadingMsg string

	// Filtering
	filterInput textinput.Model
	filtering   bool
	filterText  string

	// Error messages
	errMsg string
}

// NewApp creates a new App instance.
func NewApp() *App {
	si := textinput.New()
	si.Placeholder = "Search formulae and casks..."
	si.CharLimit = 128

	fi := textinput.New()
	fi.Placeholder = "Filter..."
	fi.CharLimit = 64

	return &App{
		cmds:        brew.NewBrewCommands(),
		activePanel: FormulaePanel,
		focusArea:   FocusSidePanel,
		maxLogLines: 100,
		commandLog:  make([]string, 0),
		searchInput: si,
		filterInput: fi,
		loading:     true,
		loadingMsg:  "Loading Homebrew data...",
	}
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		loadAllData(a.cmds),
		loadOutdated(a.cmds),
		loadLeaves(a.cmds),
	)
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		return a.handleKeyPress(msg)

	case DataLoadedMsg:
		return a.handleDataLoaded(msg)

	case OutdatedLoadedMsg:
		return a.handleOutdatedLoaded(msg)

	case LeavesLoadedMsg:
		a.leaves = msg.Leaves
		return a, nil

	case FormulaInfoMsg:
		return a.handleFormulaInfo(msg)

	case CaskInfoMsg:
		return a.handleCaskInfo(msg)

	case CommandDoneMsg:
		return a.handleCommandDone(msg)

	case SearchResultMsg:
		return a.handleSearchResult(msg)

	case RefreshMsg:
		a.loading = true
		a.loadingMsg = "Refreshing..."
		return a, tea.Batch(
			loadAllData(a.cmds),
			loadOutdated(a.cmds),
			loadLeaves(a.cmds),
		)
	}

	return a, nil
}

// --- Message Handlers ---

func (a *App) handleDataLoaded(msg DataLoadedMsg) (tea.Model, tea.Cmd) {
	a.loading = false
	if msg.Err != nil {
		a.errMsg = fmt.Sprintf("Error loading data: %v", msg.Err)
		return a, nil
	}
	a.formulae = msg.Formulae
	a.casks = msg.Casks
	a.services = msg.Services
	a.taps = msg.Taps
	a.errMsg = ""

	// Load detail for first selected item
	var cmd tea.Cmd
	if len(a.formulae) > 0 && a.activePanel == FormulaePanel {
		cmd = loadFormulaInfo(a.cmds, a.formulae[0].Name)
	}
	return a, cmd
}

func (a *App) handleOutdatedLoaded(msg OutdatedLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err == nil {
		a.outdatedFormulae = msg.Formulae
		a.outdatedCasks = msg.Casks
		// Mark outdated flags on installed formulae
		outdatedMap := make(map[string]bool)
		for _, f := range a.outdatedFormulae {
			outdatedMap[f.Name] = true
		}
		for i := range a.formulae {
			a.formulae[i].Outdated = outdatedMap[a.formulae[i].Name]
		}
		outdatedCaskMap := make(map[string]bool)
		for _, c := range a.outdatedCasks {
			outdatedCaskMap[c.FullToken] = true
		}
		for i := range a.casks {
			if a.casks[i].FullToken != "" {
				a.casks[i].Outdated = outdatedCaskMap[a.casks[i].FullToken]
			}
		}
	}
	return a, nil
}

func (a *App) handleFormulaInfo(msg FormulaInfoMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.detailInfo = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil
	}
	a.detailInfo = a.renderFormulaDetail(msg.Formula, msg.Deps, msg.Uses)
	a.detailScroll = 0
	return a, nil
}

func (a *App) handleCaskInfo(msg CaskInfoMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.detailInfo = fmt.Sprintf("Error: %v", msg.Err)
		return a, nil
	}
	a.detailInfo = a.renderCaskDetail(msg.Cask)
	a.detailScroll = 0
	return a, nil
}

func (a *App) handleCommandDone(msg CommandDoneMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		a.addLog(cmdLogSuccess.Render("✓") + " " + msg.Command + " completed")
	} else {
		errStr := ""
		if msg.Err != nil {
			errStr = ": " + msg.Err.Error()
		}
		a.addLog(cmdLogError.Render("✗") + " " + msg.Command + " failed" + errStr)
	}
	// Refresh data after command completes
	return a, func() tea.Msg { return RefreshMsg{} }
}

func (a *App) handleSearchResult(msg SearchResultMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		a.errMsg = fmt.Sprintf("Search error: %v", msg.Err)
		return a, nil
	}
	a.searchResults = make([]string, 0)
	for _, f := range msg.Formulae {
		a.searchResults = append(a.searchResults, "📦 "+f)
	}
	for _, c := range msg.Casks {
		a.searchResults = append(a.searchResults, "🖥️ "+c)
	}
	a.searchCursor = 0
	return a, nil
}

// --- Key Handling ---

func (a *App) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle overlay modes first
	if a.overlay != OverlayNone {
		return a.handleOverlayKey(msg)
	}

	// Handle filtering mode
	if a.filtering {
		return a.handleFilterKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return a, tea.Quit

	// Panel navigation
	case "tab", "]":
		a.nextPanel()
		return a, a.loadDetailForCurrentItem()
	case "shift+tab", "[":
		a.prevPanel()
		return a, a.loadDetailForCurrentItem()

	// Focus switching
	case "h":
		if a.focusArea == FocusMainPanel {
			a.focusArea = FocusSidePanel
		}
		return a, nil
	case "l":
		if a.focusArea == FocusSidePanel {
			a.focusArea = FocusMainPanel
		}
		return a, nil

	// List navigation
	case "j", "down":
		a.moveCursorDown()
		return a, a.loadDetailForCurrentItem()
	case "k", "up":
		a.moveCursorUp()
		return a, a.loadDetailForCurrentItem()
	case "g":
		a.moveCursorToTop()
		return a, a.loadDetailForCurrentItem()
	case "G":
		a.moveCursorToBottom()
		return a, a.loadDetailForCurrentItem()

	// Tab switching within panel
	case "1":
		return a.switchTab(0)
	case "2":
		return a.switchTab(1)
	case "3":
		return a.switchTab(2)

	// Detail panel scrolling (when main focused)
	case "J":
		if a.focusArea == FocusMainPanel {
			a.detailScroll++
		}
		return a, nil
	case "K":
		if a.focusArea == FocusMainPanel && a.detailScroll > 0 {
			a.detailScroll--
		}
		return a, nil

	// Actions
	case "i":
		return a.startSearch()
	case "u":
		return a.uninstallSelected()
	case "U":
		return a.upgradeSelected()
	case "r":
		return a.handleReinstartOrReinstall()
	case "p":
		return a.pinSelected()
	case "P":
		return a.unpinSelected()
	case "d":
		return a, a.loadDetailForCurrentItem()
	case "o":
		return a.openHomepage()
	case "s":
		return a.startService()
	case "S":
		return a.stopService()
	case "z":
		return a.zapCask()

	// Global actions
	case "ctrl+u":
		return a.brewUpdate()
	case "ctrl+l":
		return a.brewCleanup()
	case "ctrl+d":
		return a.brewDoctor()
	case "ctrl+a":
		return a.brewAutoremove()

	// Search/Filter
	case "/":
		a.filtering = true
		a.filterInput.Focus()
		return a, nil

	// Help
	case "?":
		a.overlay = OverlayHelp
		return a, nil

	// Enter for detail or confirm
	case "enter":
		return a, a.loadDetailForCurrentItem()
	}

	return a, nil
}

func (a *App) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch a.overlay {
	case OverlaySearch:
		return a.handleSearchKey(msg)
	case OverlayConfirm:
		return a.handleConfirmKey(msg)
	case OverlayHelp:
		if msg.String() == "esc" || msg.String() == "?" || msg.String() == "q" {
			a.overlay = OverlayNone
		}
		return a, nil
	}
	return a, nil
}

func (a *App) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.overlay = OverlayNone
		a.searchResults = nil
		a.searchInput.Reset()
		return a, nil
	case "enter":
		if a.searchInput.Value() != "" && len(a.searchResults) == 0 {
			// Start search
			query := a.searchInput.Value()
			a.addLog(cmdLogPrefix.Render("$") + " brew search " + query)
			return a, searchPackages(a.cmds, query)
		}
		if len(a.searchResults) > 0 && a.searchCursor < len(a.searchResults) {
			// Install selected result
			return a.installSearchResult()
		}
		return a, nil
	case "up", "ctrl+p":
		if a.searchCursor > 0 {
			a.searchCursor--
		}
		return a, nil
	case "down", "ctrl+n":
		if a.searchCursor < len(a.searchResults)-1 {
			a.searchCursor++
		}
		return a, nil
	default:
		var cmd tea.Cmd
		a.searchInput, cmd = a.searchInput.Update(msg)
		return a, cmd
	}
}

func (a *App) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		a.overlay = OverlayNone
		if a.confirmAction != nil {
			cmd := a.confirmAction()
			a.confirmAction = nil
			return a, cmd
		}
		return a, nil
	case "n", "esc":
		a.overlay = OverlayNone
		a.confirmAction = nil
		return a, nil
	}
	return a, nil
}

func (a *App) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.filtering = false
		a.filterText = ""
		a.filterInput.Reset()
		a.filterInput.Blur()
		return a, nil
	case "enter":
		a.filterText = a.filterInput.Value()
		a.filtering = false
		a.filterInput.Blur()
		return a, nil
	default:
		var cmd tea.Cmd
		a.filterInput, cmd = a.filterInput.Update(msg)
		a.filterText = a.filterInput.Value()
		return a, cmd
	}
}

// --- Panel Navigation ---

func (a *App) nextPanel() {
	a.activePanel = PanelID((int(a.activePanel) + 1) % PanelCount)
	a.focusArea = FocusSidePanel
}

func (a *App) prevPanel() {
	a.activePanel = PanelID((int(a.activePanel) - 1 + PanelCount) % PanelCount)
	a.focusArea = FocusSidePanel
}

func (a *App) switchTab(tab int) (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		if tab <= int(FormulaeTabLeaves) {
			a.formulaeTab = TabID(tab)
			a.formulaeCursor = 0
		}
	case CasksPanel:
		if tab <= int(CaskTabOutdated) {
			a.caskTab = TabID(tab)
			a.casksCursor = 0
		}
	case ServicesPanel:
		if tab <= int(ServiceTabStopped) {
			a.serviceTab = TabID(tab)
			a.servicesCursor = 0
		}
	}
	return a, a.loadDetailForCurrentItem()
}

// --- Cursor Navigation ---

func (a *App) moveCursorDown() {
	max := a.currentListLen() - 1
	cursor := a.currentCursor()
	if cursor < max {
		a.setCurrentCursor(cursor + 1)
	}
}

func (a *App) moveCursorUp() {
	cursor := a.currentCursor()
	if cursor > 0 {
		a.setCurrentCursor(cursor - 1)
	}
}

func (a *App) moveCursorToTop() {
	a.setCurrentCursor(0)
}

func (a *App) moveCursorToBottom() {
	max := a.currentListLen() - 1
	if max >= 0 {
		a.setCurrentCursor(max)
	}
}

func (a *App) currentCursor() int {
	switch a.activePanel {
	case FormulaePanel:
		return a.formulaeCursor
	case CasksPanel:
		return a.casksCursor
	case TapsPanel:
		return a.tapsCursor
	case ServicesPanel:
		return a.servicesCursor
	default:
		return 0
	}
}

func (a *App) setCurrentCursor(v int) {
	switch a.activePanel {
	case FormulaePanel:
		a.formulaeCursor = v
	case CasksPanel:
		a.casksCursor = v
	case TapsPanel:
		a.tapsCursor = v
	case ServicesPanel:
		a.servicesCursor = v
	}
}

func (a *App) currentListLen() int {
	switch a.activePanel {
	case FormulaePanel:
		return len(a.getFilteredFormulae())
	case CasksPanel:
		return len(a.getFilteredCasks())
	case TapsPanel:
		return len(a.taps)
	case ServicesPanel:
		return len(a.getFilteredServices())
	default:
		return 0
	}
}

// --- Data Filtering ---

func (a *App) getFilteredFormulae() []models.Formula {
	var list []models.Formula
	switch a.formulaeTab {
	case FormulaeTabInstalled:
		list = a.formulae
	case FormulaeTabOutdated:
		list = a.outdatedFormulae
	case FormulaeTabLeaves:
		leafSet := make(map[string]bool)
		for _, l := range a.leaves {
			leafSet[l] = true
		}
		for _, f := range a.formulae {
			if leafSet[f.Name] {
				list = append(list, f)
			}
		}
	}

	if a.filterText == "" {
		return list
	}

	filtered := make([]models.Formula, 0)
	for _, f := range list {
		if strings.Contains(strings.ToLower(f.Name), strings.ToLower(a.filterText)) {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

func (a *App) getFilteredCasks() []models.Cask {
	var list []models.Cask
	switch a.caskTab {
	case CaskTabInstalled:
		list = a.casks
	case CaskTabOutdated:
		list = a.outdatedCasks
	}

	if a.filterText == "" {
		return list
	}

	filtered := make([]models.Cask, 0)
	for _, c := range list {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(a.filterText)) {
			filtered = append(filtered, c)
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
	for _, s := range list {
		if strings.Contains(strings.ToLower(s.Name), strings.ToLower(a.filterText)) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

// --- Actions ---

func (a *App) startSearch() (tea.Model, tea.Cmd) {
	a.overlay = OverlaySearch
	a.searchInput.Focus()
	a.searchInput.Reset()
	a.searchResults = nil
	return a, textinput.Blink
}

func (a *App) uninstallSelected() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		items := a.getFilteredFormulae()
		if a.formulaeCursor >= len(items) {
			return a, nil
		}
		name := items[a.formulaeCursor].Name
		a.confirmMsg = fmt.Sprintf("Uninstall formula '%s'?", name)
		a.confirmAction = func() tea.Cmd {
			a.addLog(cmdLogPrefix.Render("$") + " brew uninstall " + name)
			return runBrewCommand("brew uninstall "+name, func(cb func(string)) brew.CommandResult {
				return a.cmds.Formula.Uninstall(name, cb)
			})
		}
		a.overlay = OverlayConfirm

	case CasksPanel:
		items := a.getFilteredCasks()
		if a.casksCursor >= len(items) {
			return a, nil
		}
		name := items[a.casksCursor].FullToken
		if name == "" {
			name = items[a.casksCursor].Name
		}
		a.confirmMsg = fmt.Sprintf("Uninstall cask '%s'?", name)
		a.confirmAction = func() tea.Cmd {
			a.addLog(cmdLogPrefix.Render("$") + " brew uninstall --cask " + name)
			return runBrewCommand("brew uninstall --cask "+name, func(cb func(string)) brew.CommandResult {
				return a.cmds.Cask.Uninstall(name, cb)
			})
		}
		a.overlay = OverlayConfirm
	}
	return a, nil
}

func (a *App) upgradeSelected() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		items := a.getFilteredFormulae()
		if a.formulaeCursor >= len(items) {
			return a, nil
		}
		name := items[a.formulaeCursor].Name
		a.addLog(cmdLogPrefix.Render("$") + " brew upgrade " + name)
		return a, runBrewCommand("brew upgrade "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Formula.Upgrade(name, cb)
		})

	case CasksPanel:
		items := a.getFilteredCasks()
		if a.casksCursor >= len(items) {
			return a, nil
		}
		name := items[a.casksCursor].FullToken
		if name == "" {
			name = items[a.casksCursor].Name
		}
		a.addLog(cmdLogPrefix.Render("$") + " brew upgrade --cask " + name)
		return a, runBrewCommand("brew upgrade --cask "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Cask.Upgrade(name, cb)
		})
	}
	return a, nil
}

func (a *App) handleReinstartOrReinstall() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case ServicesPanel:
		return a.restartService()
	case FormulaePanel:
		items := a.getFilteredFormulae()
		if a.formulaeCursor >= len(items) {
			return a, nil
		}
		name := items[a.formulaeCursor].Name
		a.addLog(cmdLogPrefix.Render("$") + " brew reinstall " + name)
		return a, runBrewCommand("brew reinstall "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Formula.Reinstall(name, cb)
		})
	}
	return a, nil
}

func (a *App) pinSelected() (tea.Model, tea.Cmd) {
	if a.activePanel != FormulaePanel {
		return a, nil
	}
	items := a.getFilteredFormulae()
	if a.formulaeCursor >= len(items) {
		return a, nil
	}
	name := items[a.formulaeCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew pin " + name)
	result := a.cmds.Formula.Pin(name)
	if result.Err == nil {
		a.addLog(cmdLogSuccess.Render("✓") + " " + name + " pinned")
		// Update local state
		for i := range a.formulae {
			if a.formulae[i].Name == name {
				a.formulae[i].Pinned = true
				break
			}
		}
	} else {
		a.addLog(cmdLogError.Render("✗") + " pin failed: " + result.Err.Error())
	}
	return a, nil
}

func (a *App) unpinSelected() (tea.Model, tea.Cmd) {
	if a.activePanel != FormulaePanel {
		return a, nil
	}
	items := a.getFilteredFormulae()
	if a.formulaeCursor >= len(items) {
		return a, nil
	}
	name := items[a.formulaeCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew unpin " + name)
	result := a.cmds.Formula.Unpin(name)
	if result.Err == nil {
		a.addLog(cmdLogSuccess.Render("✓") + " " + name + " unpinned")
		for i := range a.formulae {
			if a.formulae[i].Name == name {
				a.formulae[i].Pinned = false
				break
			}
		}
	} else {
		a.addLog(cmdLogError.Render("✗") + " unpin failed: " + result.Err.Error())
	}
	return a, nil
}

func (a *App) openHomepage() (tea.Model, tea.Cmd) {
	switch a.activePanel {
	case FormulaePanel:
		items := a.getFilteredFormulae()
		if a.formulaeCursor >= len(items) {
			return a, nil
		}
		name := items[a.formulaeCursor].Name
		a.cmds.Runner.Run("home", name)
	case CasksPanel:
		items := a.getFilteredCasks()
		if a.casksCursor >= len(items) {
			return a, nil
		}
		name := items[a.casksCursor].FullToken
		if name == "" {
			name = items[a.casksCursor].Name
		}
		a.cmds.Runner.Run("home", "--cask", name)
	}
	return a, nil
}

func (a *App) startService() (tea.Model, tea.Cmd) {
	if a.activePanel != ServicesPanel {
		return a, nil
	}
	svcs := a.getFilteredServices()
	if a.servicesCursor >= len(svcs) {
		return a, nil
	}
	name := svcs[a.servicesCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew services start " + name)
	return a, runBrewCommand("brew services start "+name, func(cb func(string)) brew.CommandResult {
		return a.cmds.Service.Start(name, cb)
	})
}

func (a *App) stopService() (tea.Model, tea.Cmd) {
	if a.activePanel != ServicesPanel {
		return a, nil
	}
	svcs := a.getFilteredServices()
	if a.servicesCursor >= len(svcs) {
		return a, nil
	}
	name := svcs[a.servicesCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew services stop " + name)
	return a, runBrewCommand("brew services stop "+name, func(cb func(string)) brew.CommandResult {
		return a.cmds.Service.Stop(name, cb)
	})
}

func (a *App) restartService() (tea.Model, tea.Cmd) {
	if a.activePanel != ServicesPanel {
		return a, nil
	}
	svcs := a.getFilteredServices()
	if a.servicesCursor >= len(svcs) {
		return a, nil
	}
	name := svcs[a.servicesCursor].Name
	a.addLog(cmdLogPrefix.Render("$") + " brew services restart " + name)
	return a, runBrewCommand("brew services restart "+name, func(cb func(string)) brew.CommandResult {
		return a.cmds.Service.Restart(name, cb)
	})
}

func (a *App) zapCask() (tea.Model, tea.Cmd) {
	if a.activePanel != CasksPanel {
		return a, nil
	}
	items := a.getFilteredCasks()
	if a.casksCursor >= len(items) {
		return a, nil
	}
	name := items[a.casksCursor].FullToken
	if name == "" {
		name = items[a.casksCursor].Name
	}
	a.confirmMsg = fmt.Sprintf("Zap cask '%s'? This removes ALL associated files!", name)
	a.confirmAction = func() tea.Cmd {
		a.addLog(cmdLogPrefix.Render("$") + " brew uninstall --cask --zap " + name)
		return runBrewCommand("brew zap "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Cask.Zap(name, cb)
		})
	}
	a.overlay = OverlayConfirm
	return a, nil
}

func (a *App) brewUpdate() (tea.Model, tea.Cmd) {
	a.addLog(cmdLogPrefix.Render("$") + " brew update")
	return a, runBrewCommand("brew update", func(cb func(string)) brew.CommandResult {
		return a.cmds.System.Update(cb)
	})
}

func (a *App) brewCleanup() (tea.Model, tea.Cmd) {
	a.confirmMsg = "Run 'brew cleanup --prune=all'? This removes all cached downloads."
	a.confirmAction = func() tea.Cmd {
		a.addLog(cmdLogPrefix.Render("$") + " brew cleanup --prune=all")
		return runBrewCommand("brew cleanup", func(cb func(string)) brew.CommandResult {
			return a.cmds.System.Cleanup(cb)
		})
	}
	a.overlay = OverlayConfirm
	return a, nil
}

func (a *App) brewDoctor() (tea.Model, tea.Cmd) {
	a.addLog(cmdLogPrefix.Render("$") + " brew doctor")
	return a, runBrewCommand("brew doctor", func(cb func(string)) brew.CommandResult {
		return a.cmds.System.Doctor(cb)
	})
}

func (a *App) brewAutoremove() (tea.Model, tea.Cmd) {
	a.confirmMsg = "Run 'brew autoremove'? This removes orphaned dependencies."
	a.confirmAction = func() tea.Cmd {
		a.addLog(cmdLogPrefix.Render("$") + " brew autoremove")
		return runBrewCommand("brew autoremove", func(cb func(string)) brew.CommandResult {
			return a.cmds.System.Autoremove(cb)
		})
	}
	a.overlay = OverlayConfirm
	return a, nil
}

func (a *App) installSearchResult() (tea.Model, tea.Cmd) {
	if a.searchCursor >= len(a.searchResults) {
		return a, nil
	}
	selected := a.searchResults[a.searchCursor]
	// Remove icon prefix
	name := selected
	if strings.HasPrefix(name, "📦 ") {
		name = strings.TrimPrefix(name, "📦 ")
		a.overlay = OverlayNone
		a.addLog(cmdLogPrefix.Render("$") + " brew install " + name)
		return a, runBrewCommand("brew install "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Formula.Install(name, cb)
		})
	} else if strings.HasPrefix(name, "🖥️ ") {
		name = strings.TrimPrefix(name, "🖥️ ")
		a.overlay = OverlayNone
		a.addLog(cmdLogPrefix.Render("$") + " brew install --cask " + name)
		return a, runBrewCommand("brew install --cask "+name, func(cb func(string)) brew.CommandResult {
			return a.cmds.Cask.Install(name, cb)
		})
	}
	return a, nil
}

// loadDetailForCurrentItem loads detail info based on current panel and cursor.
func (a *App) loadDetailForCurrentItem() tea.Cmd {
	switch a.activePanel {
	case FormulaePanel:
		items := a.getFilteredFormulae()
		if a.formulaeCursor < len(items) {
			return loadFormulaInfo(a.cmds, items[a.formulaeCursor].Name)
		}
	case CasksPanel:
		items := a.getFilteredCasks()
		if a.casksCursor < len(items) {
			name := items[a.casksCursor].FullToken
			if name == "" {
				name = items[a.casksCursor].Name
			}
			return loadCaskInfo(a.cmds, name)
		}
	case TapsPanel:
		if a.tapsCursor < len(a.taps) {
			tap := a.taps[a.tapsCursor]
			a.detailInfo = a.renderTapDetail(&tap)
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

// --- Detail Renderers ---

func (a *App) renderFormulaDetail(f *models.Formula, deps string, uses []string) string {
	var b strings.Builder

	title := lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(f.Name)
	b.WriteString(title + "\n\n")

	fields := []struct{ key, val string }{
		{"Version", f.CurrentVersion()},
		{"Description", f.Desc},
		{"Homepage", f.Homepage},
		{"License", f.License},
	}
	if f.Pinned {
		fields = append(fields, struct{ key, val string }{"Status", pinnedStyle.Render("📌 Pinned")})
	}
	if f.Outdated {
		fields = append(fields, struct{ key, val string }{"Update", outdatedStyle.Render("▲ Update available")})
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
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("Dependencies:") + "\n")
		b.WriteString(deps)
	}

	if len(uses) > 0 {
		b.WriteString("\n" + lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("Used by:") + "\n")
		for _, u := range uses {
			b.WriteString("  " + u + "\n")
		}
	}

	return b.String()
}

func (a *App) renderCaskDetail(c *models.Cask) string {
	var b strings.Builder

	title := lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(c.Name)
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
		fields = append(fields, struct{ key, val string }{"Update", outdatedStyle.Render("▲ Update available")})
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

func (a *App) renderTapDetail(t *models.Tap) string {
	var b strings.Builder

	title := lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(t.Name)
	b.WriteString(title + "\n\n")

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

func (a *App) renderServiceDetail(s *models.Service) string {
	var b strings.Builder

	title := lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(s.Name)
	b.WriteString(title + "\n\n")

	statusStr := string(s.Status)
	if s.IsRunning() {
		statusStr = runningStyle.Render("● Running")
	} else {
		statusStr = stoppedStyle.Render("■ Stopped")
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

	title := lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("Homebrew Status")
	b.WriteString(title + "\n\n")

	b.WriteString(detailKeyStyle.Render("Formulae:") + " " + fmt.Sprintf("%d installed", len(a.formulae)) + "\n")
	b.WriteString(detailKeyStyle.Render("Casks:") + " " + fmt.Sprintf("%d installed", len(a.casks)) + "\n")
	b.WriteString(detailKeyStyle.Render("Taps:") + " " + fmt.Sprintf("%d", len(a.taps)) + "\n")
	b.WriteString(detailKeyStyle.Render("Services:") + " " + fmt.Sprintf("%d", len(a.services)) + "\n")

	outdatedCount := len(a.outdatedFormulae) + len(a.outdatedCasks)
	if outdatedCount > 0 {
		b.WriteString(detailKeyStyle.Render("Outdated:") + " " + outdatedStyle.Render(fmt.Sprintf("%d packages", outdatedCount)) + "\n")
	} else {
		b.WriteString(detailKeyStyle.Render("Outdated:") + " " + cmdLogSuccess.Render("All up to date ✓") + "\n")
	}

	b.WriteString(detailKeyStyle.Render("Leaves:") + " " + fmt.Sprintf("%d formulae", len(a.leaves)) + "\n")

	return b.String()
}

// --- Logging ---

func (a *App) addLog(line string) {
	a.commandLog = append(a.commandLog, line)
	if len(a.commandLog) > a.maxLogLines {
		a.commandLog = a.commandLog[len(a.commandLog)-a.maxLogLines:]
	}
}
