package cli

import (
	"flag"
	"fmt"
	"os"
)

const appName = "nezha"

type Options struct {
	Version bool
	Debug   bool
	Expr    string
}

func parseOptions() Options {
	showVersion := flag.Bool("v", false, "show version and exit")
	debug := flag.Bool("d", false, "debug output")
	expr := flag.String("e", "", "evaluate expression and exit")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: nezha [options] <file.nez [args...]>")
		fmt.Fprintln(os.Stderr, "       nezha <command> [command options] [args...]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "commands:")
		for _, cmd := range commands {
			fmt.Fprintf(os.Stderr, "  %-12s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "options:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s %s\n", appName, Version)
		os.Exit(0)
	}

	return Options{
		Version: *showVersion,
		Debug:   *debug,
		Expr:    *expr,
	}
}
