// Package language provides editor-independent Nezha language services.
// It depends inward on Nezha language authorities; protocol/editor adapters
// depend on this package, never the reverse.
package language

import (
	"errors"

	"nezha/engine"
	"nezha/engine/formatter"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// Document is the neutral source unit consumed by language services.
type Document struct {
	ID      string
	Path    string
	Source  string
	Version int
}

// Service provides editor-independent queries over Nezha source.
type Service struct {
	grammar *grammar.Grammar
	catalog Catalog
}

func NewService(g *grammar.Grammar, catalogs ...Catalog) *Service {
	var catalog Catalog
	if len(catalogs) > 0 {
		catalog = catalogs[0]
	}
	return &Service{grammar: g, catalog: catalog}
}

// Diagnostics returns lexical, parse, or structural findings for doc.
// Syntax failure stops structural analysis because there is no valid tree to
// inspect. Diagnostics use Nezha-native source positions.
func (s *Service) Diagnostics(doc Document, scope value.Scope) []Diagnostic {
	file := doc.Path
	if file == "" {
		file = doc.ID
	}
	lx := syntax.NewLexer(s.grammar, file, doc.Source, nil)
	tokens, err := lx.Tokenize()
	if err != nil {
		d := diagnosticFromSyntaxError(err, "lex")
		s.diagnostic(doc, d)
		return []Diagnostic{d}
	}
	parser := syntax.NewParser(s.grammar, tokens, doc.Source, nil)
	node, err := parser.Parse()
	if err != nil {
		d := diagnosticFromSyntaxError(err, "parse")
		s.diagnostic(doc, d)
		return []Diagnostic{d}
	}
	diagnostics := analyzeNode(s.grammar, file, node, scope, s.catalog)
	for _, d := range diagnostics {
		s.diagnostic(doc, d)
	}
	return diagnostics
}

func (s *Service) diagnostic(doc Document, d Diagnostic) {
}

func diagnosticFromSyntaxError(err error, fallback string) Diagnostic {
	var sourceErr *syntax.SourceError
	if errors.As(err, &sourceErr) {
		return Diagnostic{
			Pos:      sourceErr.Pos,
			Severity: "error",
			Source:   sourceErr.Kind,
			Message:  sourceErr.Message,
		}
	}
	return Diagnostic{
		Pos:      engine.Position{},
		Severity: "error",
		Source:   fallback,
		Message:  err.Error(),
	}
}

// Format returns the canonical formatting of doc. The formatter's existing
// parse-preservation gate remains authoritative; invalid or meaning-changing
// transformations return an error rather than an edit.
func (s *Service) Format(doc Document) (string, error) {
	file := doc.Path
	if file == "" {
		file = doc.ID
	}
	out, err := formatter.FormatSource(s.grammar, file, doc.Source)
	if err != nil {
		return "", err
	}
	return out, nil
}
