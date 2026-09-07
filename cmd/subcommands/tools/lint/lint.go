package lint

import (
	"nezha/cmd/internal/langcheck"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax/grammar"
)

// Diagnostic is the structural diagnostic used by the lint CLI.
type Diagnostic = langcheck.Diagnostic

// LintSource is the CLI compatibility seam over command-side structural
// language analysis. New consumers should use cmd/internal/langcheck.
func LintSource(g *grammar.Grammar, file string, source string, lintScope value.Scope) ([]Diagnostic, error) {
	return langcheck.AnalyzeStructure(g, file, source, lintScope, langcheck.NewWorkspaceCatalog(g))
}
