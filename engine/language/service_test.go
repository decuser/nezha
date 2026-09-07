package language

import (
	"nezha/engine/formatter"
	"strings"
	"testing"

	"nezha/engine"

	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

func testService(t *testing.T) *Service {
	t.Helper()
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	return NewService(g)
}

func TestDiagnosticsLexErrorStructured(t *testing.T) {
	s := testService(t)
	d := s.Diagnostics(Document{ID: "mem", Path: "test.nez", Source: "let x = `\n"}, value.ScopeUser)
	if len(d) != 1 || d[0].Source != "lex" || d[0].Pos.Line != 1 || d[0].Pos.Col != 9 || !strings.Contains(d[0].Message, "unexpected character") {
		t.Fatalf("diagnostics = %#v", d)
	}
}

func TestDiagnosticsParseErrorStructured(t *testing.T) {
	s := testService(t)
	d := s.Diagnostics(Document{Path: "test.nez", Source: "let x =\n"}, value.ScopeUser)
	if len(d) != 1 || d[0].Source != "parse" || d[0].Pos.Line < 1 || d[0].Severity != "error" {
		t.Fatalf("diagnostics = %#v", d)
	}
}

func TestDiagnosticsStructural(t *testing.T) {
	s := testService(t)
	d := s.Diagnostics(Document{Path: "test.nez", Source: "let x = missing\n"}, value.ScopeUser)
	if len(d) != 1 || d[0].Source != "lint" || d[0].Severity != "error" || !strings.Contains(d[0].Message, "undefined") {
		t.Fatalf("diagnostics = %#v", d)
	}
}

func TestSymbolsAndDefinitionUseLexicalScopes(t *testing.T) {
	s := testService(t)
	src := "let outer = 1\nlet f = (x) {\n    let y = x\n    return y + outer\n}\n"
	doc := Document{Path: "test.nez", Source: src}
	syms, err := s.Symbols(doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) != 4 {
		t.Fatalf("symbols=%#v", syms)
	}
	for _, tc := range []struct {
		pos  engine.Position
		want string
		line int
	}{
		{engine.Position{Line: 3, Col: 13}, "x", 2},
		{engine.Position{Line: 4, Col: 12}, "y", 3},
		{engine.Position{Line: 4, Col: 18}, "outer", 1}, // caret may be inside the identifier
	} {
		got, err := s.Definition(doc, tc.pos)
		if err != nil {
			t.Fatal(err)
		}
		if !got.Found || got.Symbol.Name != tc.want || got.Symbol.Pos.Line != tc.line {
			t.Fatalf("definition at %#v = %#v", tc.pos, got)
		}
	}
}

func TestFormatUsesCanonicalFormatter(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(g)
	doc := Document{ID: "test.nez", Path: "test.nez", Source: "let   x=1\n"}
	got, err := svc.Format(doc)
	if err != nil {
		t.Fatal(err)
	}
	want, err := formatter.FormatSource(g, doc.Path, doc.Source)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("service format differs from canonical formatter:\nservice=%q\ncanonical=%q", got, want)
	}
	if got != "let x = 1\n" {
		t.Fatalf("unexpected formatted result %q", got)
	}
}

func TestFormatRejectsInvalidSource(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(g)
	if _, err := svc.Format(Document{ID: "bad.nez", Path: "bad.nez", Source: "let x =\n"}); err == nil {
		t.Fatal("expected invalid source to be rejected")
	}
}

type completionCatalog struct{}

func (completionCatalog) VisibleNames(scope value.Scope) []string { return []string{"print", "length"} }
func (completionCatalog) Help(name string) (HelpEntry, bool) {
	if name == "print" {
		return HelpEntry{Name: "print", Template: "print(val, ...)", Summary: "Print values.", Doc: "Prints values without a newline."}, true
	}
	return HelpEntry{}, false
}
func TestCompletionUsesLexicalScopeAndCatalog(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(g, completionCatalog{})
	src := "let outer = 1\nlet f = (x) {\n    let inner = x\n    return inner + outer\n}\n"
	items, err := s.Completion(Document{Path: "test.nez", Source: src}, engine.Position{Line: 4, Col: 12}, value.ScopeUser)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, item := range items {
		got[item.Name] = true
	}
	for _, want := range []string{"outer", "f", "x", "inner", "print", "length"} {
		if !got[want] {
			t.Fatalf("completion missing %q: %#v", want, items)
		}
	}
}

func TestHoverUsesLexicalDefinitionAndAuthoredHelp(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatal(err)
	}
	s := NewService(g, completionCatalog{})
	doc := Document{Path: "test.nez", Source: "let outer = 1\nlet f = () { return outer }\nlet z = print(outer)\n"}
	h, err := s.Hover(doc, engine.Position{Line: 2, Col: 21})
	if err != nil {
		t.Fatal(err)
	}
	if !h.Found || h.Name != "outer" || !strings.Contains(h.Summary, "defined at 1") {
		t.Fatalf("lexical hover=%#v", h)
	}
	h, err = s.Hover(doc, engine.Position{Line: 3, Col: 9})
	if err != nil {
		t.Fatal(err)
	}
	if !h.Found || h.Signature != "print(val, ...)" || h.Summary != "Print values." {
		t.Fatalf("help hover=%#v", h)
	}
}
