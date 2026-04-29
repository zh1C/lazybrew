package brew

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Local cache reader — reads Homebrew's local file system and
// API cache to avoid slow Ruby startup overhead

// CacheReader reads Homebrew data from local files instead of brew CLI.
type CacheReader struct {
	prefix   string // e.g. /opt/homebrew
	cacheDir string // e.g. ~/Library/Caches/Homebrew/api
}

// NewCacheReader creates a CacheReader by detecting Homebrew paths.
func NewCacheReader() *CacheReader {
	prefix := detectBrewPrefix()
	cacheDir := detectAPICacheDir()
	return &CacheReader{prefix: prefix, cacheDir: cacheDir}
}

func detectBrewPrefix() string {
	// Try common paths
	for _, p := range []string{"/opt/homebrew", "/usr/local", "/home/linuxbrew/.linuxbrew"} {
		if fi, err := os.Stat(filepath.Join(p, "Cellar")); err == nil && fi.IsDir() {
			return p
		}
	}
	return "/opt/homebrew"
}

func detectAPICacheDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Caches", "Homebrew", "api")
}

// Layer 0: Direct file system reads (0ms level)

// InstalledFormulaNames reads Cellar directory for installed formula names.
func (c *CacheReader) InstalledFormulaNames() ([]string, error) {
	cellar := filepath.Join(c.prefix, "Cellar")
	entries, err := os.ReadDir(cellar)
	if err != nil {
		return nil, fmt.Errorf("read Cellar: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// InstalledFormulaVersions reads Cellar subdirectories for installed versions.
func (c *CacheReader) InstalledFormulaVersions() (map[string]string, error) {
	cellar := filepath.Join(c.prefix, "Cellar")
	entries, err := os.ReadDir(cellar)
	if err != nil {
		return nil, fmt.Errorf("read Cellar: %w", err)
	}
	versions := make(map[string]string, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		subEntries, err := os.ReadDir(filepath.Join(cellar, name))
		if err != nil {
			continue
		}
		// Pick the latest version (last alphabetically)
		latest := ""
		for _, se := range subEntries {
			if se.IsDir() && se.Name() > latest {
				latest = se.Name()
			}
		}
		versions[name] = latest
	}
	return versions, nil
}

// InstalledCaskNames reads Caskroom directory for installed cask names.
func (c *CacheReader) InstalledCaskNames() ([]string, error) {
	caskroom := filepath.Join(c.prefix, "Caskroom")
	entries, err := os.ReadDir(caskroom)
	if err != nil {
		return nil, fmt.Errorf("read Caskroom: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

// InstalledCaskVersions reads Caskroom subdirectories for installed versions.
func (c *CacheReader) InstalledCaskVersions() (map[string]string, error) {
	caskroom := filepath.Join(c.prefix, "Caskroom")
	entries, err := os.ReadDir(caskroom)
	if err != nil {
		return nil, fmt.Errorf("read Caskroom: %w", err)
	}
	versions := make(map[string]string, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		subEntries, err := os.ReadDir(filepath.Join(caskroom, name))
		if err != nil {
			continue
		}
		latest := ""
		for _, se := range subEntries {
			if se.IsDir() && se.Name() > latest {
				latest = se.Name()
			}
		}
		versions[name] = latest
	}
	return versions, nil
}

// TapNames reads Library/Taps directory for installed tap names.
func (c *CacheReader) TapNames() ([]string, error) {
	tapsDir := filepath.Join(c.prefix, "Library", "Taps")
	userEntries, err := os.ReadDir(tapsDir)
	if err != nil {
		return nil, fmt.Errorf("read Taps: %w", err)
	}
	var names []string
	for _, ue := range userEntries {
		if !ue.IsDir() {
			continue
		}
		user := ue.Name()
		repoEntries, err := os.ReadDir(filepath.Join(tapsDir, user))
		if err != nil {
			continue
		}
		for _, re := range repoEntries {
			if !re.IsDir() {
				continue
			}
			repo := re.Name()
			// Remove "homebrew-" prefix from repo name
			tapName := user + "/" + strings.TrimPrefix(repo, "homebrew-")
			names = append(names, tapName)
		}
	}
	sort.Strings(names)
	return names, nil
}

// Layer 0.5: INSTALL_RECEIPT.json batch read (0.07s for all)

// ReceiptInfo holds data from INSTALL_RECEIPT.json.
type ReceiptInfo struct {
	InstalledOnRequest    bool
	InstalledAsDependency bool
	RuntimeDeps           []string // full_name list
	PouredFromBottle      bool
}

// AllReceipts reads INSTALL_RECEIPT.json for all installed formulae.
func (c *CacheReader) AllReceipts() (map[string]*ReceiptInfo, error) {
	cellar := filepath.Join(c.prefix, "Cellar")
	entries, err := os.ReadDir(cellar)
	if err != nil {
		return nil, fmt.Errorf("read Cellar: %w", err)
	}

	receipts := make(map[string]*ReceiptInfo, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		pkgDir := filepath.Join(cellar, name)
		subEntries, err := os.ReadDir(pkgDir)
		if err != nil {
			continue
		}
		// Find latest version
		latest := ""
		for _, se := range subEntries {
			if se.IsDir() && se.Name() > latest {
				latest = se.Name()
			}
		}
		if latest == "" {
			continue
		}
		receiptPath := filepath.Join(pkgDir, latest, "INSTALL_RECEIPT.json")
		data, err := os.ReadFile(receiptPath)
		if err != nil {
			continue
		}
		var raw struct {
			InstalledOnRequest    bool `json:"installed_on_request"`
			InstalledAsDependency bool `json:"installed_as_dependency"`
			PouredFromBottle      bool `json:"poured_from_bottle"`
			RuntimeDeps           []struct {
				FullName string `json:"full_name"`
			} `json:"runtime_dependencies"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		ri := &ReceiptInfo{
			InstalledOnRequest:    raw.InstalledOnRequest,
			InstalledAsDependency: raw.InstalledAsDependency,
			PouredFromBottle:      raw.PouredFromBottle,
		}
		for _, rd := range raw.RuntimeDeps {
			ri.RuntimeDeps = append(ri.RuntimeDeps, rd.FullName)
		}
		receipts[name] = ri
	}
	return receipts, nil
}

// Layer 1: API Cache — full formula/cask metadata (~200ms)

// APICacheFormula represents a formula from the API cache.
type APICacheFormula struct {
	Name              string           `json:"name"`
	FullName          string           `json:"full_name"`
	Desc              string           `json:"desc"`
	Homepage          string           `json:"homepage"`
	License           string           `json:"license"`
	Versions          APICacheVersions `json:"versions"`
	KegOnly           bool             `json:"keg_only"`
	Pinned            bool             `json:"pinned"`
	Outdated          bool             `json:"outdated"`
	Deprecated        bool             `json:"deprecated"`
	Disabled          bool             `json:"disabled"`
	Dependencies      []string         `json:"dependencies"`
	BuildDependencies []string         `json:"build_dependencies"`
	Caveats           *string          `json:"caveats"`
	Service           *json.RawMessage `json:"service"`
	ConflictsWith     []string         `json:"conflicts_with"`
}

// APICacheVersions holds version info from the API cache.
type APICacheVersions struct {
	Stable string `json:"stable"`
	Head   string `json:"head"`
	Bottle bool   `json:"bottle"`
}

// APICacheCask represents a cask from the API cache.
type APICacheCask struct {
	Token       string   `json:"token"`
	Name        []string `json:"name"`
	Desc        string   `json:"desc"`
	Homepage    string   `json:"homepage"`
	Version     string   `json:"version"`
	AutoUpdates bool     `json:"auto_updates"`
	Deprecated  bool     `json:"deprecated"`
	Disabled    bool     `json:"disabled"`
}

// FormulaCache holds the parsed formula API cache indexed by name.
type FormulaCache struct {
	All    []APICacheFormula           // all 8000+ formulae
	ByName map[string]*APICacheFormula // index by name
}

// CaskCache holds the parsed cask API cache indexed by token.
type CaskCache struct {
	All     []APICacheCask           // all 7000+ casks
	ByToken map[string]*APICacheCask // index by token
}

// LoadFormulaCache parses formula.jws.json from Homebrew's API cache.
func (c *CacheReader) LoadFormulaCache() (*FormulaCache, error) {
	path := filepath.Join(c.cacheDir, "formula.jws.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read formula cache: %w", err)
	}

	// JWS format: {"payload": "<JSON array>", "signatures": [...]}
	var jws struct {
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(data, &jws); err != nil {
		return nil, fmt.Errorf("parse formula JWS: %w", err)
	}

	var formulae []APICacheFormula
	if err := json.Unmarshal([]byte(jws.Payload), &formulae); err != nil {
		return nil, fmt.Errorf("parse formula payload: %w", err)
	}

	fc := &FormulaCache{
		All:    formulae,
		ByName: make(map[string]*APICacheFormula, len(formulae)),
	}
	for i := range fc.All {
		fc.ByName[fc.All[i].Name] = &fc.All[i]
	}
	return fc, nil
}

// LoadCaskCache parses cask.jws.json from Homebrew's API cache.
func (c *CacheReader) LoadCaskCache() (*CaskCache, error) {
	path := filepath.Join(c.cacheDir, "cask.jws.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read cask cache: %w", err)
	}

	var jws struct {
		Payload string `json:"payload"`
	}
	if err := json.Unmarshal(data, &jws); err != nil {
		return nil, fmt.Errorf("parse cask JWS: %w", err)
	}

	var casks []APICacheCask
	if err := json.Unmarshal([]byte(jws.Payload), &casks); err != nil {
		return nil, fmt.Errorf("parse cask payload: %w", err)
	}

	cc := &CaskCache{
		All:     casks,
		ByToken: make(map[string]*APICacheCask, len(casks)),
	}
	for i := range cc.All {
		cc.ByToken[cc.All[i].Token] = &cc.All[i]
	}
	return cc, nil
}

// Derived data — computed from cache (replaces brew deps/uses)

// BuildReverseDeps builds a reverse dependency map from formula cache.
// Returns: package name → list of installed packages that depend on it.
func BuildReverseDeps(fc *FormulaCache, installedNames []string) map[string][]string {
	installed := make(map[string]bool, len(installedNames))
	for _, n := range installedNames {
		installed[n] = true
	}

	reverse := make(map[string][]string)
	for _, name := range installedNames {
		f, ok := fc.ByName[name]
		if !ok {
			continue
		}
		for _, dep := range f.Dependencies {
			reverse[dep] = append(reverse[dep], name)
		}
	}
	return reverse
}

// ComputeLeaves determines which installed formulae are not dependencies of
// any other installed formula (i.e., "leaf" packages).
func ComputeLeaves(receipts map[string]*ReceiptInfo, reverseDeps map[string][]string) map[string]bool {
	leaves := make(map[string]bool)
	for name, ri := range receipts {
		// A leaf is either:
		// 1. installed_on_request AND no installed packages depend on it, OR
		// 2. not depended on by anything installed
		dependents := reverseDeps[name]
		if len(dependents) == 0 && ri.InstalledOnRequest {
			leaves[name] = true
		}
	}
	return leaves
}
