package brew

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// FormulaCommands provides operations for Homebrew formulae.
type FormulaCommands struct {
	runner *Runner
}

// NewFormulaCommands creates a new FormulaCommands instance.
func NewFormulaCommands(runner *Runner) *FormulaCommands {
	return &FormulaCommands{runner: runner}
}

// jsonFormulaResponse is the top-level JSON structure from `brew info --json=v2`.
type jsonFormulaResponse struct {
	Formulae []jsonFormula `json:"formulae"`
	Casks    []interface{} `json:"casks"`
}

type jsonFormula struct {
	Name         string              `json:"name"`
	FullName     string              `json:"full_name"`
	Desc         string              `json:"desc"`
	Homepage     string              `json:"homepage"`
	License      string              `json:"license"`
	Versions     jsonFormulaVersions `json:"versions"`
	Deprecated   bool                `json:"deprecated"`
	Disabled     bool                `json:"disabled"`
	KegOnly      bool                `json:"keg_only"`
	Pinned       bool                `json:"pinned"`
	Outdated     bool                `json:"outdated"`
	Installed    []jsonInstalled     `json:"installed"`
	Dependencies []string            `json:"dependencies"`
}

type jsonFormulaVersions struct {
	Stable string `json:"stable"`
	Head   string `json:"head"`
}

type jsonInstalled struct {
	Version               string          `json:"version"`
	InstalledOnRequest    bool            `json:"installed_on_request"`
	InstalledAsDependency bool            `json:"installed_as_dependency"`
	RuntimeDeps           []jsonRuntimeDep `json:"runtime_dependencies"`
}

type jsonRuntimeDep struct {
	FullName string `json:"full_name"`
	Version  string `json:"version"`
	Revision int    `json:"revision"`
}

// ListNames returns just the installed formula names (extremely fast, ~0.03s).
func (fc *FormulaCommands) ListNames() ([]string, error) {
	result := fc.runner.Run("list", "--formula")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list formula names: %w", result.Err)
	}
	return parseLines(result.Stdout), nil
}

// ListNamesWithVersions returns formula names with versions (~1s).
func (fc *FormulaCommands) ListNamesWithVersions() (map[string]string, error) {
	result := fc.runner.Run("list", "--formula", "--versions")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list formula versions: %w", result.Err)
	}
	versionMap := make(map[string]string)
	for _, line := range parseLines(result.Stdout) {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			versionMap[parts[0]] = strings.TrimSpace(parts[1])
		} else if len(parts) == 1 {
			versionMap[parts[0]] = ""
		}
	}
	return versionMap, nil
}

// ListOutdatedNames returns just the names of outdated formulae (~1.5s).
func (fc *FormulaCommands) ListOutdatedNames() ([]string, error) {
	result := fc.runner.Run("outdated", "--formula", "--quiet")
	if result.Err != nil && result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list outdated formulae: %w", result.Err)
	}
	return parseLines(result.Stdout), nil
}

// ListOutdated returns all formulae that have updates available.
func (fc *FormulaCommands) ListOutdated() ([]models.Formula, error) {
	result := fc.runner.Run("outdated", "--formula", "--json=v2")
	if result.Err != nil && result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list outdated formulae: %w", result.Err)
	}

	var resp jsonFormulaResponse
	if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse outdated JSON: %w", err)
	}

	formulae := make([]models.Formula, 0, len(resp.Formulae))
	for _, jf := range resp.Formulae {
		f := convertFormula(jf)
		f.Outdated = true
		formulae = append(formulae, f)
	}
	return formulae, nil
}

// ListLeaves returns formulae not depended on by other installed formulae.
func (fc *FormulaCommands) ListLeaves() ([]string, error) {
	result := fc.runner.Run("leaves")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list leaves: %w", result.Err)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	leaves := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			leaves = append(leaves, line)
		}
	}
	return leaves, nil
}

// ListPinned returns pinned formulae names.
func (fc *FormulaCommands) ListPinned() ([]string, error) {
	result := fc.runner.Run("list", "--pinned")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list pinned: %w", result.Err)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	pinned := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			pinned = append(pinned, line)
		}
	}
	return pinned, nil
}

// Info returns detailed info for a specific formula.
func (fc *FormulaCommands) Info(name string) (*models.Formula, error) {
	result := fc.runner.Run("info", "--json=v2", name)
	if result.Err != nil {
		return nil, fmt.Errorf("failed to get formula info: %w", result.Err)
	}

	var resp jsonFormulaResponse
	if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse formula info JSON: %w", err)
	}

	if len(resp.Formulae) == 0 {
		return nil, fmt.Errorf("formula %q not found", name)
	}

	f := convertFormula(resp.Formulae[0])
	return &f, nil
}

// Install installs a formula.
func (fc *FormulaCommands) Install(name string, onOutput func(string)) CommandResult {
	return fc.runner.RunWithCallback(onOutput, "install", "--formula", name)
}

// Uninstall removes a formula.
func (fc *FormulaCommands) Uninstall(name string, onOutput func(string)) CommandResult {
	return fc.runner.RunWithCallback(onOutput, "uninstall", "--formula", name)
}

// Upgrade upgrades a specific formula or all formulae.
func (fc *FormulaCommands) Upgrade(name string, onOutput func(string)) CommandResult {
	if name == "" {
		return fc.runner.RunWithCallback(onOutput, "upgrade", "--formula")
	}
	return fc.runner.RunWithCallback(onOutput, "upgrade", "--formula", name)
}

// Reinstall reinstalls a formula.
func (fc *FormulaCommands) Reinstall(name string, onOutput func(string)) CommandResult {
	return fc.runner.RunWithCallback(onOutput, "reinstall", "--formula", name)
}

// Pin pins a formula to prevent upgrades.
func (fc *FormulaCommands) Pin(name string) CommandResult {
	return fc.runner.Run("pin", name)
}

// Unpin unpins a formula.
func (fc *FormulaCommands) Unpin(name string) CommandResult {
	return fc.runner.Run("unpin", name)
}

// Deps returns the dependency tree for a formula.
func (fc *FormulaCommands) Deps(name string) (string, error) {
	result := fc.runner.Run("deps", "--tree", name)
	if result.Err != nil {
		return "", fmt.Errorf("failed to get deps: %w", result.Err)
	}
	return result.Stdout, nil
}

// Uses returns formulae that depend on the given formula.
func (fc *FormulaCommands) Uses(name string) ([]string, error) {
	result := fc.runner.Run("uses", "--installed", name)
	if result.Err != nil {
		return nil, fmt.Errorf("failed to get uses: %w", result.Err)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	uses := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			uses = append(uses, line)
		}
	}
	return uses, nil
}

// Search searches for formulae matching a query.
func (fc *FormulaCommands) Search(query string) ([]string, error) {
	result := fc.runner.Run("search", "--formula", query)
	if result.Err != nil {
		return nil, fmt.Errorf("search failed: %w", result.Err)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	results := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "==>") {
			// split by whitespace in case multiple results per line
			fields := strings.Fields(line)
			results = append(results, fields...)
		}
	}
	return results, nil
}

func parseLines(s string) []string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func convertFormula(jf jsonFormula) models.Formula {
	f := models.Formula{
		Name:         jf.Name,
		FullName:     jf.FullName,
		Version:      jf.Versions.Stable,
		Desc:         jf.Desc,
		Homepage:     jf.Homepage,
		License:      jf.License,
		Outdated:     jf.Outdated,
		Pinned:       jf.Pinned,
		Dependencies: jf.Dependencies,
		KegOnly:      jf.KegOnly,
		Deprecated:   jf.Deprecated,
		Disabled:     jf.Disabled,
	}

	for _, ji := range jf.Installed {
		iv := models.InstalledVersion{
			Version:               ji.Version,
			InstalledOnRequest:    ji.InstalledOnRequest,
			InstalledAsDependency: ji.InstalledAsDependency,
		}
		for _, rd := range ji.RuntimeDeps {
			iv.RuntimeDeps = append(iv.RuntimeDeps, models.RuntimeDep{
				FullName: rd.FullName,
				Version:  rd.Version,
				Revision: rd.Revision,
			})
		}
		f.Installed = append(f.Installed, iv)
	}

	return f
}
