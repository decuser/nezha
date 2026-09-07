package prelude

import (
	"fmt"
	"sort"
	"strings"

	"nezha/engine/runtime/help"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// Catalog is the engine-owned derived view of the authored prelude source,
// help, and documentation artifacts. The artifacts remain authoritative; the
// catalog joins them by procedure identity for runtime and tooling consumers.
type Catalog struct {
	Registry *help.Registry
	Names    []string
}

// LoadCatalog derives and validates the prelude surface from the real Nezha
// source plus its authored help and documentation.
func LoadCatalog(g *grammar.Grammar) (*Catalog, error) {
	return ValidateSources(g, Source, HelpSource, DocSource)
}

// ValidateSources validates a prelude source/help/doc triple through the same
// catalog construction path used by the runtime. It exists so architectural
// invariant tests can deliberately mutate one authority without duplicating the
// join logic.
func ValidateSources(g *grammar.Grammar, source, helpSource, docSource string) (*Catalog, error) {
	declared, err := declaredPreludeFunctions(g, "engine/runtime/prelude/prelude.nez", source)
	if err != nil {
		return nil, fmt.Errorf("analyzing prelude source: %w", err)
	}

	funcs, err := help.ParseHelpFile("prelude.help", helpSource)
	if err != nil {
		return nil, err
	}
	docs, err := help.ParseDocFile("prelude.doc", docSource)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(declared))
	for name := range declared {
		if strings.HasPrefix(name, "_") {
			continue
		}
		names = append(names, name)
	}
	if err := validateHelpCoverage(declared, funcs); err != nil {
		return nil, err
	}
	if err := validateDocCoverage(declared, docs); err != nil {
		return nil, err
	}
	sort.Strings(names)

	registry := help.NewRegistry()
	registry.Merge(funcs, docs)
	return &Catalog{Registry: registry, Names: names}, nil
}

func declaredPreludeFunctions(g *grammar.Grammar, file, source string) (map[string]bool, error) {
	lx := syntax.NewLexer(g, file, source)
	tokens, err := lx.Tokenize()
	if err != nil {
		return nil, err
	}
	p := syntax.NewParser(g, tokens, source)
	root, err := p.Parse()
	if err != nil {
		return nil, err
	}
	declared := map[string]bool{}
	var walk func(*syntax.Node)
	walk = func(node *syntax.Node) {
		if node == nil || node.Type == "block" {
			return
		}
		if node.Type == "let_stmt" {
			if name, ok := functionBindingName(node); ok {
				declared[name] = true
			}
			return
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)
	return declared, nil
}

func functionBindingName(node *syntax.Node) (string, bool) {
	var name string
	var valueNode *syntax.Node
	for i, child := range node.Children {
		if child.Type == "NAME" && name == "" {
			name = child.Value
		}
		if child.Type == "TERMINAL" && child.Value == "=" && i+1 < len(node.Children) {
			valueNode = node.Children[i+1]
		}
	}
	if name == "" || valueNode == nil || findFirst(valueNode, "func_literal") == nil {
		return "", false
	}
	return name, true
}

func findFirst(node *syntax.Node, typ string) *syntax.Node {
	if node == nil {
		return nil
	}
	if node.Type == typ {
		return node
	}
	for _, child := range node.Children {
		if found := findFirst(child, typ); found != nil {
			return found
		}
	}
	return nil
}

func validateHelpCoverage(declared map[string]bool, entries map[string]help.FuncEntry) error {
	var problems []string
	for name := range declared {
		if _, ok := entries[name]; !ok {
			problems = append(problems, fmt.Sprintf("missing help for '%s'", name))
		}
	}
	for name := range entries {
		if !declared[name] {
			problems = append(problems, fmt.Sprintf("orphan help entry '%s'", name))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("prelude help mismatch:\n  %s", strings.Join(problems, "\n  "))
}

func validateDocCoverage(declared map[string]bool, entries map[string]help.DocEntry) error {
	var problems []string
	for name := range declared {
		if _, ok := entries[name]; !ok {
			problems = append(problems, fmt.Sprintf("missing doc for '%s'", name))
		}
	}
	for name := range entries {
		if !declared[name] {
			problems = append(problems, fmt.Sprintf("orphan doc entry '%s'", name))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("prelude doc mismatch:\n  %s", strings.Join(problems, "\n  "))
}
