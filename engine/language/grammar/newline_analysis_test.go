package grammar_test

import (
	"nezha/engine/language/ebnfxparser"
	"nezha/engine/language/grammar"
	"strings"
	"testing"
)

func TestAnalyzeNewlineRuleCurrentGrammar(t *testing.T) {
	g, err := ebnfxparser.NewParser("grammar.ebnfx", grammar.EbnfxSource).Parse()
	if err != nil {
		t.Fatal(err)
	}
	got, err := g.AnalyzeNewlineRule()
	if err != nil {
		t.Fatal(err)
	}

	assertSymbols(t, "ExpressionEnd", got.ExpressionEnd,
		"NAME", "NUMBER", "RUNE", "STRING", "SYMBOL", ")", "]", "false", "true", "}")
	assertSymbols(t, "StatementFirst", got.StatementFirst,
		"NAME", "NUMBER", "RUNE", "STRING", "SYMBOL", "(", "-", "[", "false", "if", "let", "match", "not", "return", "true", "while")
	assertSymbols(t, "Continuation", got.Continuation,
		"(", "*", "+", "-", ".", "/", "<", "<=", ">", ">=", "[", "and", "or", "|>")
	assertSymbols(t, "Ambiguous", got.Ambiguous, "(", "-", "[")
	assertSymbols(t, "Overblocked", got.Overblocked,
		"*", "+", ".", "/", "<", "<=", ">", ">=", "and", "or", "|>")
	assertSymbols(t, "UncoveredEnd", got.UncoveredEnd, "}")
	assertSymbols(t, "DeclaredImpossible", got.DeclaredImpossible)
}

func assertSymbols(t *testing.T, name string, got []grammar.SurfaceSymbol, want ...string) {
	t.Helper()
	values := make([]string, len(got))
	for i, symbol := range got {
		values[i] = symbol.String()
	}
	if strings.Join(values, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("%s = %#v, want %#v", name, values, want)
	}
}
