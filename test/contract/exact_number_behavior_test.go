package contract

import (
	"testing"

	"nezha/engine/runtime/hal/substrate"
	"nezha/engine/semantics/evaluator"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// eval helper for invariant tests
func eval(t *testing.T, source string) value.Value {
	g, _ := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	lexer := syntax.NewLexer(g, "test", source, nil)
	tokens, _ := lexer.Tokenize()
	parser := syntax.NewParser(g, tokens, source, nil)
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	rt := substrate.NewGoRuntime()
	ev := evaluator.New(rt, nil)
	env := value.NewEnvWithScope(value.ScopePrelude)
	return ev.Eval(ast, env)
}

// TestNumbersKeepExactSemanticBoundary verifies representation changes do not alter Number semantics.
func TestNumbersKeepExactSemanticBoundary(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		verify func(t *testing.T, v value.Value)
	}{
		{
			name: "integer literal",
			expr: "42",
			verify: func(t *testing.T, v value.Value) {
				assertNumber(t, v)
			},
		},
		{
			name: "rational literal",
			expr: "1/3",
			verify: func(t *testing.T, v value.Value) {
				n := assertNumber(t, v)
				if n != nil && !n.Equal(value.NewNumber(1, 3)) {
					t.Errorf("expected 1/3, got %s", n.RatString())
				}
			},
		},
		{
			name: "addition preserves rational",
			expr: "1/3 + 1/3",
			verify: func(t *testing.T, v value.Value) {
				n := assertNumber(t, v)
				if n != nil && !n.Equal(value.NewNumber(2, 3)) {
					t.Errorf("expected 2/3, got %s", n.RatString())
				}
			},
		},
		{
			name: "division creates rational",
			expr: "1 / 3",
			verify: func(t *testing.T, v value.Value) {
				n := assertNumber(t, v)
				if n != nil && !n.Equal(value.NewNumber(1, 3)) {
					t.Errorf("expected 1/3, got %s", n.RatString())
				}
			},
		},
		{
			name: "multiplication preserves rational",
			expr: "(1/3) * 3",
			verify: func(t *testing.T, v value.Value) {
				n := assertNumber(t, v)
				if n != nil && !n.Equal(value.NewNumber(1, 1)) {
					t.Errorf("expected 1, got %s", n.RatString())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := eval(t, tt.expr)
			tt.verify(t, v)
		})
	}
}

// assertNumber verifies only the semantic boundary, never a hidden representation.
func assertNumber(t *testing.T, v value.Value) *value.Number {
	t.Helper()
	if v == nil {
		t.Error("value is nil")
		return nil
	}
	n, ok := v.(*value.Number)
	if !ok {
		t.Errorf("expected *value.Number, got %T", v)
		return nil
	}
	if n.Type() != value.NumberType {
		t.Errorf("expected number type, got %s", n.Type())
	}
	return n
}
