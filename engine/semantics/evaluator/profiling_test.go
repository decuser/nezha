package evaluator

import (
	"testing"

	"aiki/engine"
	"aiki/engine/observe"
	"aiki/engine/semantics/value"
	"aiki/engine/syntax"
	"aiki/engine/syntax/grammar"
)

func makeEvalWithProfile(t *testing.T) (*Evaluator, *grammar.Grammar) {
	t.Helper()
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	ev := New(nil, engine.SilentObserver{})
	ev.SetGrammar(g)
	ev.Profile = NewProfileProbe(false)
	return ev, g
}

func evalSource(t *testing.T, ev *Evaluator, g *grammar.Grammar, source string) value.Value {
	t.Helper()
	lexer := syntax.NewLexer(g, "test.ai", source, engine.SilentObserver{})
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	parser := syntax.NewParser(g, tokens, "test.ai", engine.SilentObserver{})
	tree, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	env := value.NewEnvWithScope(value.ScopeUser)
	env.SetFile("test.ai")
	env.SetSource(source)
	return ev.Eval(tree, env)
}

func TestProfileArithmetic(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, "1 + 2 + 3")
	// Two additions
	if ev.Profile.Measurement().Counts.Arithmetic != 2 {
		t.Errorf("arithmetic: expected 2, got %d", ev.Profile.Measurement().Counts.Arithmetic)
	}
}

func TestProfileComparison(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, "1 < 2")
	if ev.Profile.Measurement().Counts.Comparison != 1 {
		t.Errorf("comparison: expected 1, got %d", ev.Profile.Measurement().Counts.Comparison)
	}
}

func TestProfileCall(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, `
let f = () { 1 }
f()
f()
f()
`)
	if ev.Profile.Measurement().Counts.Call != 3 {
		t.Errorf("call: expected 3, got %d", ev.Profile.Measurement().Counts.Call)
	}
}

func TestProfileIteration(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, `
let x = 0
while x < 10 {
    x = x + 1
}
`)
	if ev.Profile.Measurement().Counts.Iteration != 10 {
		t.Errorf("iteration: expected 10, got %d", ev.Profile.Measurement().Counts.Iteration)
	}
	// Also: 10 comparisons in the condition, 10 additions in the body
	if ev.Profile.Measurement().Counts.Comparison != 11 {
		// 10 true + 1 false exit check
		t.Errorf("comparison: expected 11, got %d", ev.Profile.Measurement().Counts.Comparison)
	}
	if ev.Profile.Measurement().Counts.Arithmetic != 10 {
		t.Errorf("arithmetic: expected 10, got %d", ev.Profile.Measurement().Counts.Arithmetic)
	}
}

func TestProfileIndex(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, `
let xs = [10, 20, 30]
xs[0]
xs[1]
xs[2]
`)
	if ev.Profile.Measurement().Counts.Index != 3 {
		t.Errorf("index: expected 3, got %d", ev.Profile.Measurement().Counts.Index)
	}
}

func TestProfileNegation(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, "-5")
	if ev.Profile.Measurement().Counts.Arithmetic != 1 {
		t.Errorf("arithmetic (negation): expected 1, got %d", ev.Profile.Measurement().Counts.Arithmetic)
	}
}

func TestProfilingDisabledByDefault(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	ev := New(nil, engine.SilentObserver{})
	ev.SetGrammar(g)
	// Profile is nil — no semantic probes fire, no panic
	evalSource(t, ev, g, "1 + 2 * 3")
	if ev.Profile != nil {
		t.Error("profile should be nil by default")
	}
}

func TestProfileTailCalls(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	evalSource(t, ev, g, `
let countdown = (n) {
	if n <= 0 {
		return 0
	}
	countdown(n - 1)
}
countdown(5)
`)
	if ev.Profile.Measurement().Counts.Call != 6 {
		t.Errorf("call: expected 6 including 5 proper tail calls, got %d", ev.Profile.Measurement().Counts.Call)
	}
}

func TestNumberRealizationUsesEnvironmentProbe(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	ev := New(nil, engine.SilentObserver{})
	ev.SetGrammar(g)
	probe := NewProfileProbe(false)
	source := "1 + 2 * 3"
	lexer := syntax.NewLexer(g, "test.ai", source, engine.SilentObserver{})
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	parser := syntax.NewParser(g, tokens, source, engine.SilentObserver{})
	tree, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	env := value.NewEnvWithScope(value.ScopeUser)
	env.SetSemanticProbe(probe)
	env.SetFile("test.ai")
	env.SetSource(source)
	result := ev.Eval(tree, env)
	if got := result.Inspect(); got != "9" {
		t.Fatalf("result: got %s want 9", got)
	}
	n := probe.Measurement().Numbers
	if n.ResultSmallInteger != 2 {
		t.Fatalf("small integer results through environment probe: got %d want 2", n.ResultSmallInteger)
	}
}

func TestNumberRealizationNetworkEvents(t *testing.T) {
	probe := NewProfileProbe(false)
	a := value.NewNumber(2, 1)
	b := value.NewNumber(3, 1)
	probe.NumberArithmeticResult(a, b, a.Add(b))

	f1, ok := value.NewNumberFromFloat64(0.5)
	if !ok {
		t.Fatal("0.5 carrier rejected")
	}
	f2, ok := value.NewNumberFromFloat64(0.25)
	if !ok {
		t.Fatal("0.25 carrier rejected")
	}
	probe.NumberArithmeticResult(f1, f2, f1.Add(f2))

	n := probe.Measurement().Numbers
	if n.ResultSmallInteger != 1 {
		t.Fatalf("small integer results: got %d want 1", n.ResultSmallInteger)
	}
	if n.ResultBinaryCarrier != 1 || n.BinaryCertified != 1 || n.BinaryFallback != 0 {
		t.Fatalf("binary realization: %+v", n)
	}
}

func TestNumberCallRealizationUsesEnvironmentProbe(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	ev := New(nil, engine.SilentObserver{})
	ev.SetGrammar(g)
	probe := NewProfileProbe(false)
	source := "let f = () { 42 }\nf()\n"
	lexer := syntax.NewLexer(g, "test.ai", source, engine.SilentObserver{})
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	parser := syntax.NewParser(g, tokens, source, engine.SilentObserver{})
	tree, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	env := value.NewEnvWithScope(value.ScopeUser)
	env.SetSemanticProbe(probe)
	env.SetFile("test.ai")
	env.SetSource(source)
	result := ev.Eval(tree, env)
	if got := result.Inspect(); got != "42" {
		t.Fatalf("result: got %s want 42", got)
	}
	n := probe.Measurement().CallNumbers
	if n.ResultSmallInteger != 1 {
		t.Fatalf("small integer call returns through environment probe: got %d want 1", n.ResultSmallInteger)
	}
}

func TestTailRecursiveArgumentFramesTransfer(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	result := evalSource(t, ev, g, `
let sum_tail = (n, acc) {
    if n <= 0 { return acc }
    return sum_tail(n - 1, acc + n)
}
sum_tail(5000, 0)
`)
	if value.IsFault(result) {
		t.Fatalf("tail recursion faulted: %s", result.Inspect())
	}
	if result.Inspect() != "12502500" {
		t.Fatalf("tail recursion result = %s, want 12502500", result.Inspect())
	}
	calls := ev.Profile.Measurement().Calls
	if calls.ArgTailTransfer < 4999 {
		t.Fatalf("tail transfers = %d, want at least 4999", calls.ArgTailTransfer)
	}
	if calls.ArgFrameReused == 0 {
		t.Fatal("deep tail recursion never reused an argument frame")
	}
	if calls.ArgFrameNew >= calls.ArgTailTransfer {
		t.Fatalf("tail recursion did not bound frame growth: new=%d transfers=%d", calls.ArgFrameNew, calls.ArgTailTransfer)
	}
}

func TestEnvironmentRealizationUsesNetworkAccounting(t *testing.T) {
	probe := NewProfileProbe(false)
	root := value.NewEnv()
	root.SetSemanticProbe(probe)

	enclosed := value.NewEnclosedEnv(root)
	enclosed.Set("a", value.NewNumber(1, 1))
	enclosed.Set("b", value.NewNumber(2, 1))
	enclosed.Set("a", value.NewNumber(3, 1))

	call := value.NewCallEnv(root, root)
	for i, name := range []string{"a", "b", "c", "d", "e"} {
		call.Set(name, value.NewNumber(int64(i+1), 1))
	}
	call.Set("a", value.NewNumber(9, 1))

	// A tail-reused call environment is a second logical call but not a second
	// physical allocation. Its local high-water lifetime starts over.
	call.ResetCallEnv(root, root)
	call.Set("z", value.NewNumber(10, 1))

	envs := probe.Measurement().Envs
	if envs.PhysicalEnclosed != 1 || envs.PhysicalCall != 1 || envs.LogicalCall != 2 {
		t.Fatalf("environment populations: got enclosed=%d physical_call=%d logical_call=%d", envs.PhysicalEnclosed, envs.PhysicalCall, envs.LogicalCall)
	}
	if envs.CallReached1 != 2 || envs.CallReached2 != 1 || envs.CallReached3 != 1 || envs.CallReached5 != 1 {
		t.Fatalf("call high-water crossings: %+v", envs)
	}
	if envs.EnclosedReached1 != 1 || envs.EnclosedReached2 != 1 || envs.EnclosedReached3 != 0 || envs.EnclosedReached5 != 0 {
		t.Fatalf("enclosed high-water crossings: %+v", envs)
	}
	if envs.CallCompactAllocations != 1 || envs.CallMapPromotions != 1 || envs.CallLocalNew != 6 || envs.CallLocalUpdate != 1 {
		t.Fatalf("call realization: %+v", envs)
	}
	if envs.EnclosedCompactAllocations != 1 || envs.EnclosedMapPromotions != 0 || envs.EnclosedLocalNew != 2 || envs.EnclosedLocalUpdate != 1 {
		t.Fatalf("enclosed realization: %+v", envs)
	}
}

type recordingEvalProbe struct {
	hits []struct {
		node string
		site engine.SemanticSite
	}
}

func (p *recordingEvalProbe) Hit(binding *observe.NodeBinding) bool { return false }

func (p *recordingEvalProbe) BindAndHit(node string, site engine.SemanticSite, binding *observe.NodeBinding) {
	p.hits = append(p.hits, struct {
		node string
		site engine.SemanticSite
	}{node: node, site: site})
}

func TestEvalProbeObservesEveryEvaluatorEntryWithoutSeparateCoverageAccounting(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("loading grammar: %v", err)
	}
	ev := New(nil, engine.SilentObserver{})
	ev.SetGrammar(g)
	probe := &recordingEvalProbe{}
	ev.EvalProbe = probe

	evalSource(t, ev, g, "1 + 2\n")
	if len(probe.hits) == 0 {
		t.Fatal("eval probe observed no evaluator entries")
	}
	for _, hit := range probe.hits {
		if hit.site.File != "test.ai" || hit.site.Line != 1 {
			t.Fatalf("unexpected eval site: %#v", hit)
		}
		if hit.site.Detail != hit.node {
			t.Fatalf("detail=%q node=%q", hit.site.Detail, hit.node)
		}
	}
}
