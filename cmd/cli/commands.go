package cli

import (
	"fmt"
	"os"

	nezhadebug "nezha/cmd/subcommands/dev/debug"
	nezhaenginesmoke "nezha/cmd/subcommands/dev/enginesmoke"
	nezhasmoke "nezha/cmd/subcommands/dev/smoke"
	nezhadistfmt "nezha/cmd/subcommands/tools/distfmt"
	nezhafmt "nezha/cmd/subcommands/tools/fmt"
	nezhalint "nezha/cmd/subcommands/tools/lint"
	nezhatreecheck "nezha/cmd/subcommands/tools/treecheck"
)

type command struct {
	Name        string
	Description string
	Run         func([]string) int
}

var commands = []command{
	{Name: "debug", Description: "inspect lexer/parser/evaluator stages", Run: nezhadebug.Run},
	{Name: "fmt", Description: "format Nezha source", Run: nezhafmt.Run},
	{Name: "gate", Description: "run repository validation gates", Run: runGate},
	{Name: "lint", Description: "lint Nezha source", Run: nezhalint.Run},
	{Name: "version", Description: "print the Nezha version", Run: func([]string) int {
		fmt.Println(Version)
		return 0
	}},
}

var gateCommands = []command{
	{Name: "distfmt", Description: "apply distribution/source presentation style", Run: nezhadistfmt.Run},
	{Name: "enginesmoke", Description: "run structural engine specimens", Run: nezhaenginesmoke.Run},
	{Name: "smoke", Description: "run smoke/golden tests", Run: nezhasmoke.Run},
	{Name: "treecheck", Description: "verify repository distribution relationships", Run: nezhatreecheck.Run},
}

func dispatchCommand(name string, args []string) (int, bool) {
	for _, cmd := range commands {
		if cmd.Name == name {
			return cmd.Run(args), true
		}
	}
	return 0, false
}

func runGate(args []string) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		printGateUsage()
		if len(args) == 0 {
			return 2
		}
		return 0
	}
	name := args[0]
	for _, cmd := range gateCommands {
		if cmd.Name == name {
			return cmd.Run(args[1:])
		}
	}
	fmt.Fprintf(os.Stderr, "unknown gate %q\n", name)
	printGateUsage()
	return 2
}

func printGateUsage() {
	fmt.Fprintln(os.Stderr, "usage: nezha gate <command> [command options] [args...]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "gate commands:")
	for _, cmd := range gateCommands {
		fmt.Fprintf(os.Stderr, "  %-12s %s\n", cmd.Name, cmd.Description)
	}
}
