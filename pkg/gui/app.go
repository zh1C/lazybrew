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

// StartupStage tracks the two-phase loading progress (like lazygit).
type StartupStage int

const (
	StageLoading   StartupStage = iota // Stage 1: loading name lists
	StageEnriching                     // Stage 2: enriching with versions/outdated/leaves
	StageComplete                      // All data loaded
)

// App is the main bubbletea model.
type App struct {
	// Brew commands
	cmds *brew.BrewCommands

	// Terminal dimensions
	width  int
	height int

	// Startup stage (like lazygit's INITIAL/COMPLETE pattern)
	stage StartupStage

	// Panel state
	activePanel PanelID
	focusArea   FocusArea

	// Tab state per panel
	formulaeTab TabID
	caskTab     TabID
	serviceTab  TabID

	// Data — Stage 1: just names (instant)
	formulaeNames []string
	caskNames     []string
	tapNames      []string

	// Data — Stage 2: enrichment (background)
	formulaeVersions map[string]string // name → version
	caskVersions     map[string]string // name → version
	outdatedFormulae map[string]bool   // name set
	outdatedCasks    map[string]bool   // name set
	leaves           map[string]bool   // name set
	services         []models.Service
	taps             []models.Tap

	// Per-panel loading flags
	formulaeLoading bool
	casksLoading    bool
	tapsLoading     bool
	servicesLoading bool

	// List cursors
	formulaeCursor int
	casksCursor    int
	tapsCursor     int
	servicesCursor int
	searchCursor   int

	// Detail panel — on-demand loading with cache
	detailInfo    string
	detailScroll  int
	detailLoading bool
	detailCache   map[string]string // "formula:name" or "cask:name" → rendered detail

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
		detailCache:      make(map[string]string),
		formulaeLoading:  true,
		casksLoading:     true,
		tapsLoading:      true,
		servicesLoading:  true,
	}
}

// Init implements tea.Model.
// Stage 1: Fire 3 instant name-list commands in parallel (~0.06s total).
// Spinner tick for loading animation.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		loadFormulaeNames(a.cmds), // ~0.03s
		loadCaskNames(a.cmds),     // ~0.03s
		loadTapNames(a.cmds),      // ~0.06s
		spinnerTick(),             // animate loading
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

	// --- Stage 1 handlers (instant) ---
	case FormulaeNamesMsg:
		return a.handleFormulaeNames(msg)
	case CaskNamesMsg:
		return a.handleCaskNames(msg)
	case TapNamesMsg:
		return a.handleTapNames(msg)

	// --- Stage 2 handlers (background) ---
	case FormulaeVersionsMsg:
		return a.handleFormulaeVersions(msg)
	case CaskVersionsMsg:
		return a.handleCaskVersions(msg)
	case OutdatedNamesMsg:
		return a.handleOutdatedNames(msg)
	case LeavesLoadedMsg:
		return a.handleLeaves(msg)
	case ServicesLoadedMsg:
		return a.handleServices(msg)
	case TapsDetailMsg:
		return a.handleTapsDetail(msg)

	// --- On-demand detail ---
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

// ============================================================
// Panel & Cursor Navigation
// ============================================================

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

// ============================================================
// Selected item helpers
// ============================================================

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

// ============================================================
// Spinner & Logging helpers
// ============================================================

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
