package brew

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// TapCommands provides operations for Homebrew taps.
type TapCommands struct {
	runner *Runner
}

// NewTapCommands creates a new TapCommands instance.
func NewTapCommands(runner *Runner) *TapCommands {
	return &TapCommands{runner: runner}
}

type jsonTap struct {
	Name          string `json:"name"`
	Remote        string `json:"remote"`
	Installed     bool   `json:"installed"`
	Official      bool   `json:"official"`
	FormulaCount  int    `json:"formula_count"`
	CaskCount     int    `json:"cask_count"`
}

// List returns all taps.
func (tc *TapCommands) List() ([]models.Tap, error) {
	result := tc.runner.Run("tap-info", "--installed", "--json")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list taps: %w", result.Err)
	}

	var jsonTaps []jsonTap
	if err := json.Unmarshal([]byte(result.Stdout), &jsonTaps); err != nil {
		return nil, fmt.Errorf("failed to parse taps JSON: %w", err)
	}

	taps := make([]models.Tap, 0, len(jsonTaps))
	for _, jt := range jsonTaps {
		t := models.Tap{
			Name:         jt.Name,
			Remote:       jt.Remote,
			Installed:    jt.Installed,
			Official:     jt.Official,
			FormulaCount: jt.FormulaCount,
			CaskCount:    jt.CaskCount,
		}
		taps = append(taps, t)
	}
	return taps, nil
}

// Add adds a new tap.
func (tc *TapCommands) Add(name string, onOutput func(string)) CommandResult {
	return tc.runner.RunWithCallback(onOutput, "tap", name)
}

// Remove removes a tap.
func (tc *TapCommands) Remove(name string, onOutput func(string)) CommandResult {
	return tc.runner.RunWithCallback(onOutput, "untap", name)
}

// ListNames returns just the tap names.
func (tc *TapCommands) ListNames() ([]string, error) {
	result := tc.runner.Run("tap")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list tap names: %w", result.Err)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}
