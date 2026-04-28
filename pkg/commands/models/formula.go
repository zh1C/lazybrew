package models

// Formula represents a Homebrew formula (CLI tool/library).
type Formula struct {
	Name         string
	FullName     string
	Version      string
	Desc         string
	Homepage     string
	License      string
	Installed    []InstalledVersion
	Outdated     bool
	Pinned       bool
	Dependencies []string
	Dependents   []string
	KegOnly      bool
	Deprecated   bool
	Disabled     bool
	// Computed display fields
	InstalledPath string
	InstalledSize string
}

// InstalledVersion represents a specific installed version of a formula.
type InstalledVersion struct {
	Version          string
	InstalledOnRequest bool
	InstalledAsDependency bool
	RuntimeDeps      []RuntimeDep
}

// RuntimeDep represents a runtime dependency.
type RuntimeDep struct {
	FullName string
	Version  string
	Revision int
}

// CurrentVersion returns the latest installed version string.
func (f *Formula) CurrentVersion() string {
	if len(f.Installed) > 0 {
		return f.Installed[0].Version
	}
	return f.Version
}

// IsInstalledOnRequest returns true if the formula was explicitly installed by user.
func (f *Formula) IsInstalledOnRequest() bool {
	for _, v := range f.Installed {
		if v.InstalledOnRequest {
			return true
		}
	}
	return false
}
