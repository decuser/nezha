package coverage

import (
	"testing"

	"nezha/engine"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

func TestFromASTDefinesStaticSitesWithoutPunctuationLines(t *testing.T) {
	root := &syntax.Node{Type: "program", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 1}, Children: []*syntax.Node{
		{Type: "statement", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 1}, Children: []*syntax.Node{
			{Type: "let_stmt", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 1}, Children: []*syntax.Node{
				{Type: "TERMINAL", Value: "let", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 1}},
				{Type: "NAME", Value: "f", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 5}},
				{Type: "expr", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 9}, Children: []*syntax.Node{
					{Type: "func_literal", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 9}, Children: []*syntax.Node{
						{Type: "block", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 1, Col: 13}, Children: []*syntax.Node{
							{Type: "statement", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 2, Col: 2}},
							{Type: "TERMINAL", Value: "}", Pos: engine.Position{File: "lib/demo/demo.nez", Line: 3, Col: 1}},
						}},
					}},
				}},
			}},
		}},
	}}

	model := FromAST(root)
	lines := model.LinesByFile()["lib/demo/demo.nez"]
	if !lines[1] || !lines[2] {
		t.Fatalf("coverable lines = %#v, want lines 1 and 2", lines)
	}
	if lines[3] {
		t.Fatalf("punctuation-only line 3 unexpectedly coverable: %#v", lines)
	}
	foundFunctionBody := false
	for _, site := range model.Sites {
		if site.Line == 2 && site.Function == "f" && site.Module == "demo" && site.Layer == "semantic" {
			foundFunctionBody = true
		}
	}
	if !foundFunctionBody {
		t.Fatalf("function/module/layer classification missing from sites: %#v", model.Sites)
	}
}

func TestFromASTRealParserExcludesBlankCommentAndClosingBraceLines(t *testing.T) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		t.Fatalf("load grammar: %v", err)
	}
	source := "# comment\n\nlet f = (x) {\n\treturn x + 1\n}\n"
	lexer := syntax.NewLexer(g, "lib/demo/demo.nez", source, engine.SilentObserver{})
	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("lex: %v", err)
	}
	parser := syntax.NewParser(g, tokens, source, engine.SilentObserver{})
	ast, err := parser.Parse()
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	lines := FromAST(ast).LinesByFile()["lib/demo/demo.nez"]
	if lines[1] || lines[2] || lines[5] {
		t.Fatalf("comment/blank/closing-brace lines unexpectedly coverable: %#v", lines)
	}
	if !lines[3] || !lines[4] {
		t.Fatalf("definition/body lines missing from static model: %#v", lines)
	}
}
