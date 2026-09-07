package grammar_test

import (
	"nezha/engine/language/ebnfxparser"
	"nezha/engine/language/grammar"
	"reflect"
	"testing"
)

func TestAnalysisCurrentGrammar(t *testing.T) {
	g, err := ebnfxparser.NewParser("grammar.ebnfx", grammar.EbnfxSource).Parse()
	if err != nil {
		t.Fatal(err)
	}
	a := g.Analysis()
	if a == nil {
		t.Fatal("analysis is nil")
	}
	if got := len(a.ProductionNames()); got != 28 {
		t.Fatalf("productions = %d, want 28", got)
	}

	wantRefs := map[string]struct{}{
		"NAME": {}, "NUMBER": {}, "STRING": {}, "RUNE": {}, "SYMBOL": {}, "SHAPE": {},
	}
	if got := a.TokenRefs(); !reflect.DeepEqual(got, wantRefs) {
		t.Fatalf("TokenRefs = %v, want %v", got, wantRefs)
	}
	for name := range wantRefs {
		if _, ok := a.ASTNodeTypes()[name]; !ok {
			t.Errorf("ASTNodeTypes missing grammar.TokenRef %s", name)
		}
	}
	for _, name := range a.ProductionNames() {
		if _, ok := a.ASTNodeTypes()[name]; !ok {
			t.Errorf("ASTNodeTypes missing production %s", name)
		}
	}

	ops := a.TerminalAlternatives("BINOP")
	if len(ops) != 10 {
		t.Fatalf("BINOP terminals = %d, want 10", len(ops))
	}
	for _, op := range []string{"+", "-", "*", "/", "<", ">", "<=", ">=", "and", "or"} {
		if _, ok := ops[op]; !ok {
			t.Errorf("BINOP terminals missing %q", op)
		}
	}

	if a.NewlineError != nil {
		t.Fatalf("newline analysis: %v", a.NewlineError)
	}
	if a.Newline == nil {
		t.Fatal("newline analysis is nil")
	}
}

func TestAnalysisStructuralFactsSurviveMissingNewlinePrerequisites(t *testing.T) {
	g := &grammar.Grammar{Productions: map[string]*grammar.Production{
		"program": {Name: "program", Expr: &grammar.TokenRef{Name: "NAME"}},
	}}
	a := g.Reanalyze()
	if _, ok := a.TokenRefs()["NAME"]; !ok {
		t.Fatal("structural grammar.TokenRef missing")
	}
	if a.Newline != nil {
		t.Fatal("unexpected newline analysis")
	}
	if a.NewlineError == nil {
		t.Fatal("expected newline analysis error")
	}
}

func TestReanalyzeReflectsDeliberateMutation(t *testing.T) {
	g := &grammar.Grammar{Productions: map[string]*grammar.Production{
		"program": {Name: "program", Expr: &grammar.TokenRef{Name: "NAME"}},
	}}
	g.Reanalyze()
	g.Productions["program"].Expr = &grammar.Sequence{Exprs: []grammar.Expression{
		&grammar.TokenRef{Name: "NAME"},
		&grammar.TokenRef{Name: "FAKE_TOKEN"},
	}}
	if _, ok := g.Analysis().TokenRefs()["FAKE_TOKEN"]; ok {
		t.Fatal("analysis changed without explicit reanalysis")
	}
	g.Reanalyze()
	if _, ok := g.Analysis().TokenRefs()["FAKE_TOKEN"]; !ok {
		t.Fatal("reanalysis did not reflect mutation")
	}
}

func TestAnalyzeNewlineRuleReturnsCachedAnalysis(t *testing.T) {
	g, err := ebnfxparser.NewParser("grammar.ebnfx", grammar.EbnfxSource).Parse()
	if err != nil {
		t.Fatal(err)
	}
	cached := g.Analysis().Newline
	first, err := g.AnalyzeNewlineRule()
	if err != nil {
		t.Fatal(err)
	}
	second, err := g.AnalyzeNewlineRule()
	if err != nil {
		t.Fatal(err)
	}
	if first != cached || second != cached {
		t.Fatal("AnalyzeNewlineRule did not return cached analysis")
	}
}
