package langcheck

import "nezha/engine/semantics/value"

// HelpEntry is authored help/documentation projected into lint diagnostics.
type HelpEntry struct {
	Name     string
	Template string
	Summary  string
	Doc      string
}

// Catalog supplies prelude/runtime facts needed by command-side structural checks
// without moving tool-specific analysis into engine packages.
type Catalog interface {
	VisibleNames(scope value.Scope) []string
	Help(name string) (HelpEntry, bool)
}
