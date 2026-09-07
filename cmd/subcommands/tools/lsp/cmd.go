package lsp

import (
	"fmt"
	"os"

	"nezha/engine/language"
	"nezha/engine/language/workspace"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

func Run(args []string) int {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, "usage: nezha lsp")
		return 2
	}
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		fmt.Fprintln(os.Stderr, "lsp:", err)
		return 1
	}
	service := language.NewService(g, workspace.NewCatalog(g))
	if err := Serve(os.Stdin, os.Stdout, service); err != nil {
		fmt.Fprintln(os.Stderr, "lsp:", err)
		return 1
	}
	return 0
}
