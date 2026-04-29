package brew

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zh1C/lazybrew/pkg/commands/models"
)

// CaskCommands provides operations for Homebrew casks.
type CaskCommands struct {
	runner *Runner
}

// NewCaskCommands creates a new CaskCommands instance.
func NewCaskCommands(runner *Runner) *CaskCommands {
	return &CaskCommands{runner: runner}
}

type jsonCaskResponse struct {
	Formulae []interface{} `json:"formulae"`
	Casks    []jsonCask    `json:"casks"`
}

type jsonCask struct {
	Token       string   `json:"token"`
	FullToken   string   `json:"full_token"`
	Name        []string `json:"name"`
	Desc        string   `json:"desc"`
	Homepage    string   `json:"homepage"`
	Version     string   `json:"version"`
	Installed   string   `json:"installed"`
	Outdated    bool     `json:"outdated"`
	AutoUpdates bool     `json:"auto_updates"`
}

// ListNames returns just the installed cask names
func (cc *CaskCommands) ListNames() ([]string, error) {
	result := cc.runner.Run("list", "--cask")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list cask names: %w", result.Err)
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

// ListNamesWithVersions returns cask names with versions
func (cc *CaskCommands) ListNamesWithVersions() (map[string]string, error) {
	result := cc.runner.Run("list", "--cask", "--versions")
	if result.Err != nil {
		return nil, fmt.Errorf("failed to list cask versions: %w", result.Err)
	}
	versionMap := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(result.Stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			versionMap[parts[0]] = strings.TrimSpace(parts[1])
		} else if len(parts) == 1 {
			versionMap[parts[0]] = ""
		}
	}
	return versionMap, nil
}

// ListOutdatedNames returns just the names of outdated casks
func (cc *CaskCommands) ListOutdatedNames() ([]string, error) {
	result := cc.runner.Run("outdated", "--cask", "--quiet")
	if result.Err != nil && result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list outdated casks: %w", result.Err)
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

// ListOutdated returns all outdated casks.
func (cc *CaskCommands) ListOutdated() ([]models.Cask, error) {
	result := cc.runner.Run("outdated", "--cask", "--json=v2")
	if result.Err != nil && result.ExitCode != 0 {
		return nil, fmt.Errorf("failed to list outdated casks: %w", result.Err)
	}

	if strings.TrimSpace(result.Stdout) == "" {
		return []models.Cask{}, nil
	}

	var resp jsonCaskResponse
	if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse outdated casks JSON: %w", err)
	}

	casks := make([]models.Cask, 0, len(resp.Casks))
	for _, jc := range resp.Casks {
		c := convertCask(jc)
		c.Outdated = true
		casks = append(casks, c)
	}
	return casks, nil
}

// Info returns detailed info for a specific cask.
func (cc *CaskCommands) Info(name string) (*models.Cask, error) {
	result := cc.runner.Run("info", "--json=v2", "--cask", name)
	if result.Err != nil {
		return nil, fmt.Errorf("failed to get cask info: %w", result.Err)
	}

	var resp jsonCaskResponse
	if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse cask info JSON: %w", err)
	}

	if len(resp.Casks) == 0 {
		return nil, fmt.Errorf("cask %q not found", name)
	}

	c := convertCask(resp.Casks[0])
	return &c, nil
}

// Install installs a cask.
func (cc *CaskCommands) Install(name string, onOutput func(string)) CommandResult {
	return cc.runner.RunWithCallback(onOutput, "install", "--cask", name)
}

// Uninstall removes a cask.
func (cc *CaskCommands) Uninstall(name string, onOutput func(string)) CommandResult {
	return cc.runner.RunWithCallback(onOutput, "uninstall", "--cask", name)
}

// Upgrade upgrades a cask.
func (cc *CaskCommands) Upgrade(name string, onOutput func(string)) CommandResult {
	if name == "" {
		return cc.runner.RunWithCallback(onOutput, "upgrade", "--cask")
	}
	return cc.runner.RunWithCallback(onOutput, "upgrade", "--cask", name)
}

// Zap completely removes a cask including all associated files.
func (cc *CaskCommands) Zap(name string, onOutput func(string)) CommandResult {
	return cc.runner.RunWithCallback(onOutput, "uninstall", "--cask", "--zap", name)
}

// Search searches for casks matching a query.
func (cc *CaskCommands) Search(query string) ([]string, error) {
	result := cc.runner.Run("search", "--cask", query)
	if result.Err != nil {
		return nil, fmt.Errorf("search failed: %w", result.Err)
	}
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	results := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "==>") {
			fields := strings.Fields(line)
			results = append(results, fields...)
		}
	}
	return results, nil
}

func convertCask(jc jsonCask) models.Cask {
	name := jc.Token
	if len(jc.Name) > 0 {
		name = jc.Name[0]
	}
	return models.Cask{
		Name:        name,
		FullToken:   jc.FullToken,
		Version:     jc.Version,
		Desc:        jc.Desc,
		Homepage:    jc.Homepage,
		Installed:   jc.Installed,
		Outdated:    jc.Outdated,
		AutoUpdates: jc.AutoUpdates,
	}
}
