package cli

import (
	"flag"
	"fmt"
	"os"

	"nezha/engine/semantics/value"
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
	if err := runner.Run(filename, flag.Args()[1:]...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runExpr(expr string, opts Options) {
	sess, err := runner.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result := sess.Eval(expr)

	if f, ok := result.(*value.Fault); ok {
		fmt.Fprintln(os.Stderr, f.Inspect())
		os.Exit(1)
	}

	if opts.Debug {
		fmt.Println(result.Inspect())
	}
}
