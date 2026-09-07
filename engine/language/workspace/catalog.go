// Package workspace adapts the engine module workspace to the neutral
// language.Catalog contract.
package workspace

import (
	"sort"

	"nezha/engine/language"
	"nezha/engine/runtime/prelude"
	"nezha/engine/runtime/primitives"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax/grammar"
)

var _ language.Catalog = (*Catalog)(nil)

type Catalog struct {
	grammar        *grammar.Grammar
	preludeCatalog *prelude.Catalog
}

func NewCatalog(g *grammar.Grammar) *Catalog {
	return &Catalog{grammar: g}
}

func (c *Catalog) authoredPrelude() *prelude.Catalog {
	if c.preludeCatalog != nil {
		return c.preludeCatalog
	}
	catalog, err := prelude.LoadCatalog(c.grammar)
	if err != nil {
		return nil
	}
	c.preludeCatalog = catalog
	return catalog
}

func (c *Catalog) VisibleNames(scope value.Scope) []string {
	set := map[string]bool{}
	for _, name := range primitives.NamesForScope(scope) {
		set[name] = true
	}
	if scope == value.ScopeUser {
		if catalog := c.authoredPrelude(); catalog != nil {
			for _, name := range catalog.Names {
				set[name] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (c *Catalog) Help(name string) (language.HelpEntry, bool) {
	catalog := c.authoredPrelude()
	if catalog == nil {
		return language.HelpEntry{}, false
	}
	entry := catalog.Registry.GetHelp(name)
	if entry == nil {
		return language.HelpEntry{}, false
	}
	doc := ""
	if d := catalog.Registry.GetDoc(name); d != nil {
		doc = d.Doc
	}
	return language.HelpEntry{Name: name, Template: entry.Template, Summary: entry.Help, Doc: doc}, true
}
