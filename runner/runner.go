// Package runner provides host-facing convenience for executing Nezha programs.
package runner

import (
	"fmt"
	"os"

	"nezha/engine/program"
	"nezha/engine/runtime/hal/substrate"
)

// Run executes a Nezha source file with the standard Go runtime substrate.
func Run(filename string) error {
	rt := substrate.NewGoRuntime()
	return RunWithRuntime(filename, rt)
}

// RunWithRuntime executes a Nezha source file in the supplied runtime host
// world. The caller owns the runtime's lifetime.
func RunWithRuntime(filename string, rt *substrate.GoRuntime) error {
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return program.RunSource(rt, filename, string(source))
}

// RunSource executes Nezha source code with the standard Go runtime substrate.
func RunSource(filename, source string) error {
	rt := substrate.NewGoRuntime()
	return RunSourceWithRuntime(filename, source, rt)
}

// RunSourceWithRuntime executes supplied Nezha source under the given filename in
// the supplied runtime host world. The caller owns the runtime's lifetime.
func RunSourceWithRuntime(filename, source string, rt *substrate.GoRuntime) error {
	return program.RunSource(rt, filename, source)
}

// RunExpr evaluates a single expression and returns the inspected result.
func RunExpr(expr string) (string, error) {
	rt := substrate.NewGoRuntime()
	return program.EvalExpr(rt, expr)
}
