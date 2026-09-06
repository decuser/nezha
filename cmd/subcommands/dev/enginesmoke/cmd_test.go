package enginesmoke

import (
	"os"
	"path/filepath"
	"testing"

	"aiki/engine/syntax"
	"aiki/engine/syntax/grammar"
)

func withDistributionRoot(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("cannot find distribution root at %s: %v", root, err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestCollectProductionsCountsOnlyGrammarNodes(t *testing.T) {
	g := &grammar.Grammar{Productions: map[string]*grammar.Production{
		"program":   {Name: "program"},
		"statement": {Name: "statement"},
	}}
	ast := &syntax.Node{Type: "program", Children: []*syntax.Node{
		{Type: "statement", Children: []*syntax.Node{{Type: "NUMBER"}}},
	}}
	covered := map[string]bool{}
	collectProductions(ast, g, covered)
	if !covered["program"] || !covered["statement"] {
		t.Fatalf("expected grammar productions covered, got %#v", covered)
	}
	if covered["NUMBER"] {
		t.Fatalf("token/value node must not count as a grammar production")
	}
}

func TestCollectInputsSupportsArbitraryRecursiveSubtree(t *testing.T) {
	withDistributionRoot(t)
	d := t.TempDir()
	sub := filepath.Join(d, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(sub, "x_engine.ai")
	if err := os.WriteFile(p, []byte("let x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := collectInputs([]string{sub + "/..."})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != p {
		t.Fatalf("collect=%v want [%s]", got, p)
	}
}
