package aikiapp

import (
	"fmt"

	aikidebug "aiki/cmd/subcommands/dev/debug"
	aikienginesmoke "aiki/cmd/subcommands/dev/enginesmoke"
	aikismoke "aiki/cmd/subcommands/dev/smoke"
	aikitest "aiki/cmd/subcommands/dev/test"
	aikicheck "aiki/cmd/subcommands/tools/check"
	aikidistfmt "aiki/cmd/subcommands/tools/distfmt"
	aikifmt "aiki/cmd/subcommands/tools/fmt"
	aikilint "aiki/cmd/subcommands/tools/lint"
	aikilsp "aiki/cmd/subcommands/tools/lsp"
	aikiprofile "aiki/cmd/subcommands/tools/profile"
	aikitags "aiki/cmd/subcommands/tools/tags"
	aikitreecheck "aiki/cmd/subcommands/tools/treecheck"
)

type command struct {
	Name        string
	Description string
	Run         func([]string) int
}

var commands = []command{
	{Name: "check", Description: "static source diagnostics", Run: aikicheck.Run},
	{Name: "debug", Description: "inspect lexer/parser/evaluator stages", Run: aikidebug.Run},
	{Name: "distfmt", Description: "apply distribution/source presentation style", Run: aikidistfmt.Run},
	{Name: "enginesmoke", Description: "run structural engine specimens", Run: aikienginesmoke.Run},
	{Name: "fmt", Description: "format Aiki source", Run: aikifmt.Run},
	{Name: "lint", Description: "lint Aiki source", Run: aikilint.Run},
	{Name: "lsp", Description: "run the language server", Run: aikilsp.Run},
	{Name: "profile", Description: "profile an Aiki program", Run: aikiprofile.Run},
	{Name: "smoke", Description: "run smoke/golden tests", Run: aikismoke.Run},
	{Name: "tags", Description: "generate editor tags", Run: aikitags.Run},
	{Name: "test", Description: "run Aiki-native tests", Run: aikitest.Run},
	{Name: "treecheck", Description: "verify repository distribution relationships", Run: aikitreecheck.Run},
	{Name: "version", Description: "print the Aiki version", Run: func([]string) int {
		fmt.Println(Version)
		return 0
	}},
}

func dispatchCommand(name string, args []string) (int, bool) {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd.Run(args), true
		}
	}
	return 0, false
}
