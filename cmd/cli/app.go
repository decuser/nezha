package cli

import (
	"flag"
	"fmt"
	"os"

	"nezha/runner"
)

func Main() {
	// Check for subcommands before flag parsing. The command registry is also
	// the authority used by top-level help, so dispatch and discovery cannot drift.
	if len(os.Args) > 1 {
		if code, ok := dispatchCommand(os.Args[1], os.Args[2:]); ok {
			if code != 0 {
				os.Exit(code)
			}
			return
		}
	}

	opts := parseOptions()

	if opts.Expr != "" {
		runExpr(opts.Expr, opts)
		return
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	runFile(flag.Arg(0), opts)
}

func runFile(filename string, opts Options) {
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: program arguments are not supported")
		flag.Usage()
		os.Exit(2)
	}
	if err := runner.Run(filename); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runExpr(expr string, opts Options) {
	result, err := runner.RunExpr(expr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if opts.Debug {
		fmt.Println(result)
	}
}
