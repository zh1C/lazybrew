package brew

// SystemCommands provides system-level Homebrew operations.
type SystemCommands struct {
	runner *Runner
}

// NewSystemCommands creates a new SystemCommands instance.
func NewSystemCommands(runner *Runner) *SystemCommands {
	return &SystemCommands{runner: runner}
}

// Update runs `brew update` to fetch the latest package info.
func (sc *SystemCommands) Update(onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "update")
}

// Cleanup runs `brew cleanup` to remove outdated files.
func (sc *SystemCommands) Cleanup(onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "cleanup", "--prune=all")
}

// Doctor runs `brew doctor` to check system health.
func (sc *SystemCommands) Doctor(onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "doctor")
}

// Autoremove runs `brew autoremove` to remove orphaned dependencies.
func (sc *SystemCommands) Autoremove(onOutput func(string)) CommandResult {
	return sc.runner.RunWithCallback(onOutput, "autoremove")
}

// BrewCommands aggregates all brew command groups.
type BrewCommands struct {
	Runner  *Runner
	Formula *FormulaCommands
	Cask    *CaskCommands
	Service *ServiceCommands
	Tap     *TapCommands
	System  *SystemCommands
	Cache   *CacheReader // Local file system + API cache (fast path)
}

// NewBrewCommands creates a fully initialized BrewCommands.
func NewBrewCommands() *BrewCommands {
	runner := NewRunner()
	return &BrewCommands{
		Runner:  runner,
		Formula: NewFormulaCommands(runner),
		Cask:    NewCaskCommands(runner),
		Service: NewServiceCommands(runner),
		Tap:     NewTapCommands(runner),
		System:  NewSystemCommands(runner),
		Cache:   NewCacheReader(),
	}
}
