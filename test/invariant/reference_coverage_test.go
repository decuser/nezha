package invariant

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"nezha/engine/language/grammar"
	"nezha/engine/language/loader"
	"nezha/engine/syntax"
)

// TestReferenceCorpusCoversGrammar keeps reference/ aligned with the authored
// grammar. The reference programs are a positive, readable witness of the
// language surface; behavior tests remain free to be narrower regression cases.
func TestReferenceCorpusCoversGrammar(t *testing.T) {
	g, err := loader.Load("grammar.ebnfx", grammar.EbnfxSource, "grammar.help", grammar.HelpSource)
	if err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join("..", "..", "reference", "*.nez"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("reference corpus is empty")
	}
	sort.Strings(files)

	seenTypes := map[string]bool{}
	seenValues := map[string]bool{}
	for _, file := range files {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		lx := syntax.NewLexer(g, file, string(source))
		tokens, err := lx.Tokenize()
		if err != nil {
			t.Fatalf("%s: lex: %v", file, err)
		}
		p := syntax.NewParser(g, tokens, string(source))
		root, err := p.Parse()
		if err != nil {
			t.Fatalf("%s: parse: %v", file, err)
		}
		walkReferenceNode(root, seenTypes, seenValues)
	}

	var missing []string
	for _, production := range g.Analysis().ProductionNames() {
		if !seenTypes[production] {
			missing = append(missing, "production "+production)
		}
	}
	for op := range g.Analysis().TerminalAlternatives("BINOP") {
		if !seenValues[op] {
			missing = append(missing, "BINOP "+op)
		}
	}
	for _, token := range []string{"NUMBER", "STRING", "RUNE", "SYMBOL", "SHAPE", "NAME"} {
		if !seenTypes[token] {
			missing = append(missing, "token "+token)
		}
	}
	for _, terminal := range []string{"let", "if", "else", "while", "match", "return", "true", "false", "not", "|>", ".", "=", "...", ";"} {
		if !seenValues[terminal] {
			missing = append(missing, "terminal "+terminal)
		}
	}
	if len(missing) != 0 {
		sort.Strings(missing)
		for _, item := range missing {
			t.Errorf("reference corpus missing %s", item)
		}
	}
}

func walkReferenceNode(node *syntax.Node, types, values map[string]bool) {
	if node == nil {
		return
	}
	types[node.Type] = true
	if node.Value != "" {
		values[node.Value] = true
	}
	for _, child := range node.Children {
		walkReferenceNode(child, types, values)
	}
}
