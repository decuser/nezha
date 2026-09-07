package aikiapp

import (
	"flag"
	"fmt"
	"os"
)

const appName = "aiki"

type Options struct {
	Version    bool
	Debug      bool
	Expr       string
	CPUProfile string
	TraceFile  string
}

func parseOptions() Options {
	showVersion := flag.Bool("v", false, "show version and exit")
	debug := flag.Bool("d", false, "debug output")
	expr := flag.String("e", "", "evaluate expression and exit")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")
	tracefile := flag.String("trace", "", "write runtime trace to file")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: aiki [options] <file.ai [args...]>")
		fmt.Fprintln(os.Stderr, "       aiki <command> [command options] [args...]")
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
		Version:    *showVersion,
		Debug:      *debug,
		Expr:       *expr,
		CPUProfile: *cpuprofile,
		TraceFile:  *tracefile,
	}
}
