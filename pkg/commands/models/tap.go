package models

// Tap represents a Homebrew third-party repository.
type Tap struct {
	Name        string
	Remote      string
	Installed   bool
	Official    bool
	FormulaCount int
	CaskCount   int
}
