package langcheck

import (
	"sort"

	"nezha/engine/runtime/prelude"
	"nezha/engine/runtime/primitives"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax/grammar"
)

var _ Catalog = (*WorkspaceCatalog)(nil)

type WorkspaceCatalog struct {
	grammar        *grammar.Grammar
	preludeCatalog *prelude.Catalog
}

func NewWorkspaceCatalog(g *grammar.Grammar) *WorkspaceCatalog {
	return &WorkspaceCatalog{grammar: g}
}

func (c *WorkspaceCatalog) authoredPrelude() *prelude.Catalog {
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

func (c *WorkspaceCatalog) VisibleNames(scope value.Scope) []string {
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

func (c *WorkspaceCatalog) Help(name string) (HelpEntry, bool) {
	catalog := c.authoredPrelude()
	if catalog == nil {
		return HelpEntry{}, false
	}
	entry := catalog.Registry.GetHelp(name)
	if entry == nil {
		return HelpEntry{}, false
	}
	doc := ""
	if d := catalog.Registry.GetDoc(name); d != nil {
		doc = d.Doc
	}
	return HelpEntry{Name: name, Template: entry.Template, Summary: entry.Help, Doc: doc}, true
}
