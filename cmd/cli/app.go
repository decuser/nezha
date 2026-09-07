package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"nezha/engine/runner"
	"nezha/engine/semantics/value"
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
	err := runner.Run(filename, flag.Args()[1:]...)
	if err == nil {
		return
	}
	var exitErr *runner.ExitStatusError
	if errors.As(err, &exitErr) {
		os.Exit(exitErr.Code)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func runExpr(expr string, opts Options) {
	sess, err := runner.NewSession()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result := sess.Eval(expr)

	if _, ok := result.(*value.ExitSignal); ok {
		return
	}
	if exit, ok := result.(*value.ProgramExitSignal); ok {
		sess.Runtime.CloseAllResources()
		os.Exit(exit.Code)
	}

	if f, ok := result.(*value.Fault); ok {
		fmt.Fprintln(os.Stderr, f.Inspect())
		os.Exit(1)
	}

	if opts.Debug {
		fmt.Println(result.Inspect())
	}
}
