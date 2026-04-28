package models

// Cask represents a Homebrew cask (GUI application).
type Cask struct {
	Name      string
	FullToken string
	Version   string
	Desc      string
	Homepage  string
	Installed string // installed version
	Outdated  bool
	AutoUpdates bool
	Artifacts []string
}

// IsInstalled returns true if the cask has an installed version.
func (c *Cask) IsInstalled() bool {
	return c.Installed != ""
}
