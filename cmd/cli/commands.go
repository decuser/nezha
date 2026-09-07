package cli

import (
	"fmt"

	nezhadebug "nezha/cmd/subcommands/dev/debug"
	nezhaenginesmoke "nezha/cmd/subcommands/dev/enginesmoke"
	nezhasmoke "nezha/cmd/subcommands/dev/smoke"
	nezhatest "nezha/cmd/subcommands/dev/test"
	nezhacheck "nezha/cmd/subcommands/tools/check"
	nezhadistfmt "nezha/cmd/subcommands/tools/distfmt"
	nezhafmt "nezha/cmd/subcommands/tools/fmt"
	nezhalint "nezha/cmd/subcommands/tools/lint"
	nezhalsp "nezha/cmd/subcommands/tools/lsp"
	nezhaprofile "nezha/cmd/subcommands/tools/profile"
	nezhatags "nezha/cmd/subcommands/tools/tags"
	nezhatreecheck "nezha/cmd/subcommands/tools/treecheck"
)

type command struct {
	Name        string
	Description string
	Run         func([]string) int
}

var commands = []command{
	{Name: "check", Description: "static source diagnostics", Run: nezhacheck.Run},
	{Name: "debug", Description: "inspect lexer/parser/evaluator stages", Run: nezhadebug.Run},
	{Name: "distfmt", Description: "apply distribution/source presentation style", Run: nezhadistfmt.Run},
	{Name: "enginesmoke", Description: "run structural engine specimens", Run: nezhaenginesmoke.Run},
	{Name: "fmt", Description: "format Nezha source", Run: nezhafmt.Run},
	{Name: "lint", Description: "lint Nezha source", Run: nezhalint.Run},
	{Name: "lsp", Description: "run the language server", Run: nezhalsp.Run},
	{Name: "profile", Description: "profile a Nezha program", Run: nezhaprofile.Run},
	{Name: "smoke", Description: "run smoke/golden tests", Run: nezhasmoke.Run},
	{Name: "tags", Description: "generate editor tags", Run: nezhatags.Run},
	{Name: "test", Description: "run Nezha-native tests", Run: nezhatest.Run},
	{Name: "treecheck", Description: "verify repository distribution relationships", Run: nezhatreecheck.Run},
	{Name: "version", Description: "print the Nezha version", Run: func([]string) int {
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
