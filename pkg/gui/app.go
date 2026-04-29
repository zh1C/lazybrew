package gui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
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

// StartupStage tracks the two-phase loading progress
type StartupStage int

const (
	StageLoading   StartupStage = iota // Stage 1: loading from file system
	StageEnriching                     // Stage 2: loading API cache + brew commands
	StageComplete                      // All data loaded
)

// App is the main bubbletea model.
type App struct {
	// Brew commands
	cmds *brew.BrewCommands

	// Terminal dimensions
	width  int
	height int

	// Startup stage
	stage StartupStage

	// Panel state
	activePanel PanelID
	focusArea   FocusArea

	// Tab state per panel
	formulaeTab TabID
	caskTab     TabID
	serviceTab  TabID

	// Data — Stage 1: names + versions from file system (instant)
	formulaeNames    []string
	formulaeVersions map[string]string
	caskNames        []string
	caskVersions     map[string]string
	tapNames         []string

	// Data — Stage 2: enrichment from API cache + brew commands
	formulaCache     *brew.FormulaCache           // all 8000+ formulae metadata
	caskCache        *brew.CaskCache              // all 7000+ casks metadata
	receipts         map[string]*brew.ReceiptInfo // install receipts
	reverseDeps      map[string][]string          // pkg → dependents
	outdatedFormulae map[string]bool
	outdatedCasks    map[string]bool
	leaves           map[string]bool
	services         []models.Service
	taps             []models.Tap

	// Per-panel loading flags
	formulaeLoading bool
	casksLoading    bool
	tapsLoading     bool
	servicesLoading bool
	cacheLoading    bool // API cache loading in progress

	// Stage 2 completion tracking
	stage2CacheDone    bool
	stage2OutdatedDone bool
	stage2ServicesDone bool

	// List cursors
	formulaeCursor int
	casksCursor    int
	tapsCursor     int
	servicesCursor int
	searchCursor   int

	// Detail panel — rendered from cache data (instant)
	detailInfo   string
	detailScroll int

	// Command log
	commandLog  []string
	maxLogLines int

	// Overlay state
	overlay       OverlayMode
	searchInput   textinput.Model
	searchResults []string
	confirmMsg    string
	confirmAction func() tea.Cmd

	// Filtering
	filterInput textinput.Model
	filtering   bool
	filterText  string

	// Spinner
	spinnerFrame int

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
		cmds:             brew.NewBrewCommands(),
		stage:            StageLoading,
		activePanel:      FormulaePanel,
		focusArea:        FocusSidePanel,
		maxLogLines:      100,
		commandLog:       make([]string, 0),
		searchInput:      si,
		filterInput:      fi,
		formulaeVersions: make(map[string]string),
		caskVersions:     make(map[string]string),
		outdatedFormulae: make(map[string]bool),
		outdatedCasks:    make(map[string]bool),
		leaves:           make(map[string]bool),
		formulaeLoading:  true,
		casksLoading:     true,
		tapsLoading:      true,
		servicesLoading:  true,
		cacheLoading:     true,
	}
}

// Init implements tea.Model.
// Stage 1: Fire 3 instant file-system reads in parallel
// Spinner tick for loading animation.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		loadFormulaeFromFS(a.cmds),
		loadCasksFromFS(a.cmds),
		loadTapsFromFS(a.cmds),
		spinnerTick(),
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

	// --- Stage 1 handlers (instant from file system) ---
	case FormulaeNamesMsg:
		return a.handleFormulaeNames(msg)
	case CaskNamesMsg:
		return a.handleCaskNames(msg)
	case TapNamesMsg:
		return a.handleTapNames(msg)

	// --- Stage 2 handlers (API cache + brew commands) ---
	case CacheLoadedMsg:
		return a.handleCacheLoaded(msg)
	case OutdatedNamesMsg:
		return a.handleOutdatedNames(msg)
	case ServicesLoadedMsg:
		return a.handleServices(msg)

	// --- On-demand detail fallback ---
	case FormulaInfoMsg:
		return a.handleFormulaInfo(msg)
	case CaskInfoMsg:
		return a.handleCaskInfo(msg)

	// --- Actions ---
	case CommandDoneMsg:
		return a.handleCommandDone(msg)
	case SearchResultMsg:
		return a.handleSearchResult(msg)
	case RefreshMsg:
		return a.handleRefresh()

	// --- Spinner ---
	case SpinnerTickMsg:
		a.spinnerFrame++
		if a.stage != StageComplete {
			return a, spinnerTick()
		}
		return a, nil
	}

	return a, nil
}

// Panel & Cursor Navigation

func (a *App) nextPanel() {
	a.activePanel = PanelID((int(a.activePanel) + 1) % PanelCount)
	a.focusArea = FocusSidePanel
}

func (a *App) prevPanel() {
	a.activePanel = PanelID((int(a.activePanel) - 1 + PanelCount) % PanelCount)
	a.focusArea = FocusSidePanel
}

// tabCount returns how many tabs the current panel has (0 means no tabs).
// Panels without tabs (Status, Taps) return 0.
func (a *App) tabCount() int {
	switch a.activePanel {
	case FormulaePanel:
		return 3 // Installed, Outdated, Leaves
	case CasksPanel:
		return 2 // Installed, Outdated
	case ServicesPanel:
		return 3 // All, Running, Stopped
	default:
		return 0
	}
}

// currentTab returns the current tab index for the active panel.
func (a *App) currentTab() int {
	switch a.activePanel {
	case FormulaePanel:
		return int(a.formulaeTab)
	case CasksPanel:
		return int(a.caskTab)
	case ServicesPanel:
		return int(a.serviceTab)
	default:
		return 0
	}
}

// nextTab switches to the next tab within the current panel
// Wraps around from last → first (ModuloWithWrap pattern).
func (a *App) nextTab() (tea.Model, tea.Cmd) {
	count := a.tabCount()
	if count == 0 {
		return a, nil
	}
	newTab := (a.currentTab() + 1) % count
	return a.switchTab(newTab)
}

// prevTab switches to the previous tab within the current panel
// Wraps around from first → last (ModuloWithWrap pattern).
func (a *App) prevTab() (tea.Model, tea.Cmd) {
	count := a.tabCount()
	if count == 0 {
		return a, nil
	}
	newTab := (a.currentTab() - 1 + count) % count
	return a.switchTab(newTab)
}

// switchTab sets the active tab to the given index
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
		return len(a.getFilteredFormulaeNames())
	case CasksPanel:
		return len(a.getFilteredCaskNames())
	case TapsPanel:
		return len(a.getFilteredTapNames())
	case ServicesPanel:
		return len(a.getFilteredServices())
	default:
		return 0
	}
}

// Selected item helpers

func (a *App) selectedFormulaName() string {
	items := a.getFilteredFormulaeNames()
	if a.formulaeCursor < len(items) {
		return items[a.formulaeCursor]
	}
	return ""
}

func (a *App) selectedCaskName() string {
	items := a.getFilteredCaskNames()
	if a.casksCursor < len(items) {
		return items[a.casksCursor]
	}
	return ""
}

// Spinner & Logging helpers

// spinnerChar returns the current spinner frame character.
func (a *App) spinnerChar() string {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	return frames[a.spinnerFrame%len(frames)]
}

func (a *App) addLog(line string) {
	a.commandLog = append(a.commandLog, line)
	if len(a.commandLog) > a.maxLogLines {
		a.commandLog = a.commandLog[len(a.commandLog)-a.maxLogLines:]
	}
}
