// Package coverage defines the static anatomy used by runtime coverage.
// Dynamic execution truth belongs to engine/observe's profiling network; this
// package describes what parsed Nezha source could be covered.
package coverage

import (
	"fmt"
	"sort"

	"nezha/engine/observe"
	"nezha/engine/syntax"
)

// Site is one static, source-backed Nezha syntax site. Sites deliberately use
// source/AST identity rather than runtime NodeIDs so never-executed code exists
// without being manufactured in the dynamic profiling network.
type Site struct {
	File     string
	Line     int
	Col      int
	Node     string
	Function string
	Module   string
	Layer    string
}

// Model is the static coverable-site universe for one or more parsed units.
type Model struct {
	Sites []Site
}

// FromAST derives static coverable sites from parsed Nezha structure.
//
// Coverage is source-structure coverage, not punctuation coverage: TERMINAL
// nodes are excluded. Every remaining source-backed AST node is a static site;
// line coverage is the set of distinct lines containing at least one such site.
// This removes comments, blank lines, and punctuation-only lines from the
// denominator without asking the runtime network to invent unexecuted nodes.
func FromAST(root *syntax.Node) Model {
	if root == nil {
		return Model{}
	}
	bindings := make(map[*syntax.Node]string)
	collectFunctionBindings(root, bindings)

	seen := make(map[siteKey]bool)
	sites := make([]Site, 0)
	walkStatic(root, "<main>", bindings, seen, &sites)
	sort.Slice(sites, func(i, j int) bool {
		a, b := sites[i], sites[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Col != b.Col {
			return a.Col < b.Col
		}
		if a.Function != b.Function {
			return a.Function < b.Function
		}
		return a.Node < b.Node
	})
	return Model{Sites: sites}
}

type siteKey struct {
	file     string
	line     int
	col      int
	node     string
	function string
}

func walkStatic(node *syntax.Node, function string, bindings map[*syntax.Node]string, seen map[siteKey]bool, out *[]Site) {
	if node == nil {
		return
	}
	if node.Pos.Line > 0 && node.Type != "TERMINAL" {
		key := siteKey{node.Pos.File, node.Pos.Line, node.Pos.Col, node.Type, function}
		if !seen[key] {
			seen[key] = true
			*out = append(*out, Site{
				File: node.Pos.File, Line: node.Pos.Line, Col: node.Pos.Col,
				Node: node.Type, Function: function,
				Module: observe.ModuleFromFile(node.Pos.File), Layer: "semantic",
			})
		}
	}

	childFunction := function
	if node.Type == "func_literal" {
		if name := bindings[node]; name != "" {
			childFunction = name
		} else {
			childFunction = fmt.Sprintf("<anonymous@%d:%d>", node.Pos.Line, node.Pos.Col)
		}
	}
	for _, child := range node.Children {
		walkStatic(child, childFunction, bindings, seen, out)
	}
}

func collectFunctionBindings(node *syntax.Node, bindings map[*syntax.Node]string) {
	if node == nil {
		return
	}
	if node.Type == "let_stmt" {
		name := directChildValue(node, "NAME")
		if name != "" {
			if fn := firstDescendant(node, "func_literal"); fn != nil {
				bindings[fn] = name
			}
		}
	}
	for _, child := range node.Children {
		collectFunctionBindings(child, bindings)
	}
}

func directChildValue(node *syntax.Node, typ string) string {
	for _, child := range node.Children {
		if child != nil && child.Type == typ {
			return child.Value
		}
	}
	return ""
}

func firstDescendant(node *syntax.Node, typ string) *syntax.Node {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		if child.Type == typ {
			return child
		}
		if found := firstDescendant(child, typ); found != nil {
			return found
		}
	}
	return nil
}

// LinesByFile returns the static line denominator derived from Sites.
func (m Model) LinesByFile() map[string]map[int]bool {
	out := make(map[string]map[int]bool)
	for _, site := range m.Sites {
		if site.File == "" || site.Line <= 0 {
			continue
		}
		if out[site.File] == nil {
			out[site.File] = make(map[int]bool)
		}
		out[site.File][site.Line] = true
	}
	return out
}
