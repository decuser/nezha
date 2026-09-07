// Package runner provides the runner for executing Nezha programs.
package runner

import (
	"fmt"
	"os"

	"nezha/engine"
	"nezha/engine/runtime/hal/substrate"
	"nezha/engine/runtime/prelude"
	"nezha/engine/semantics/evaluator"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// ExitStatusError carries a requested nonzero Nezha program exit status back to the CLI.
type ExitStatusError struct{ Code int }

func (e *ExitStatusError) Error() string { return fmt.Sprintf("exit status %d", e.Code) }

func executionResultError(result value.Value) error {
	if fault, ok := result.(*value.Fault); ok {
		return fmt.Errorf("%s", fault.Inspect())
	}
	if exit, ok := result.(*value.ProgramExitSignal); ok {
		if exit.Code == 0 {
			return nil
		}
		return &ExitStatusError{Code: exit.Code}
	}
	return nil
}

// Run executes a Nezha source file. programArgs are exposed to the program
// through system.args(), excluding the interpreter and source filename.
func Run(filename string, programArgs ...string) error {
	rt := substrate.NewGoRuntime()
	defer rt.CloseAllResources()
	return RunWithRuntime(filename, rt, programArgs...)
}

// RunWithRuntime executes a Nezha source file in the supplied runtime host
// world. The caller owns the runtime's lifetime and cleanup.
func RunWithRuntime(filename string, rt *substrate.GoRuntime, programArgs ...string) error {
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return runSourceWithRuntime(filename, string(source), programArgs, rt)
}

// RunSource executes Nezha source code.
func RunSource(filename, source string) error {
	rt := substrate.NewGoRuntime()
	defer rt.CloseAllResources()
	return runSourceWithRuntime(filename, source, nil, rt)
}

func runSourceWithRuntime(filename, source string, programArgs []string, rt *substrate.GoRuntime) error {
	return runSourceWithRuntimeProbe(filename, source, programArgs, rt, nil)
}

// RunSourceWithRuntime executes supplied Nezha source under the given filename in
// the supplied runtime host world. The caller owns the runtime's lifetime.
func RunSourceWithRuntime(filename, source string, rt *substrate.GoRuntime, programArgs ...string) error {
	return runSourceWithRuntimeProbe(filename, source, programArgs, rt, nil)
}

// RunSourceWithEvalProbeRuntime executes supplied Nezha source while routing AST
// execution-entry observations to the supplied probe.
func RunSourceWithEvalProbeRuntime(filename, source string, probe engine.EvalProbe, rt *substrate.GoRuntime, programArgs ...string) error {
	return runSourceWithRuntimeProbe(filename, source, programArgs, rt, probe)
}

// RunWithEvalProbeRuntime executes a Nezha source file while routing AST
// execution-entry observations to the supplied nervous-system endpoint. The
// probe owns no accounting state; callers typically supply observe.EvalCapture.
func RunWithEvalProbeRuntime(filename string, probe engine.EvalProbe, rt *substrate.GoRuntime, programArgs ...string) error {
	source, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return runSourceWithRuntimeProbe(filename, string(source), programArgs, rt, probe)
}

func runSourceWithRuntimeProbe(filename, source string, programArgs []string, rt *substrate.GoRuntime, evalProbe engine.EvalProbe) error {
	// Load grammar
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return fmt.Errorf("loading grammar: %w", err)
	}

	rt.SetProgramArgs(programArgs)

	// Create prelude environment with ScopePrelude
	preludeEnv := value.NewEnvWithScope(value.ScopePrelude)
	preludeEnv.SetAuthority(rt.AuthorityForSource("engine/runtime/prelude/prelude.nez"))

	// Load prelude
	if err := loadPrelude(g, rt, preludeEnv); err != nil {
		return fmt.Errorf("loading prelude: %w", err)
	}

	// Create user environment enclosed by prelude.
	userEnv := value.NewEnclosedEnvWithScope(preludeEnv, value.ScopeUser)
	userEnv.SetAuthority(rt.AuthorityForSource(filename))

	// Lex user code
	lexer := syntax.NewLexer(g, filename, source, nil)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("lexer: %w", err)
	}

	// Parse user code
	parser := syntax.NewParser(g, tokens, source, nil)
	ast, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parser: %w", err)
	}

	// Eval user code
	userEnv.SetFile(filename)
	userEnv.SetSource(source)
	ev := evaluator.New(rt, nil)
	ev.SetGrammar(g)
	ev.EvalProbe = evalProbe
	result := ev.Eval(ast, userEnv)
	return executionResultError(result)
}

// loadPrelude parses and evaluates the prelude.
func loadPrelude(g *grammar.Grammar, rt *substrate.GoRuntime, env *value.Env) error {
	// Initialize help registry
	if err := initHelpRegistry(g, rt); err != nil {
		return fmt.Errorf("loading help: %w", err)
	}

	lexer := syntax.NewLexer(g, "<prelude>", prelude.Source, nil)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return err
	}

	parser := syntax.NewParser(g, tokens, prelude.Source, nil)
	ast, err := parser.Parse()
	if err != nil {
		return err
	}

	env.SetFile("<prelude>")
	env.SetSource(prelude.Source)
	ev := evaluator.New(rt, nil)
	ev.SetGrammar(g)
	result := ev.Eval(ast, env)
	if fault, ok := result.(*value.Fault); ok {
		return fmt.Errorf("%s", fault.Inspect())
	}

	return nil
}

// initHelpRegistry loads prelude help and doc files with validation.

func initHelpRegistry(g *grammar.Grammar, rt *substrate.GoRuntime) error {
	catalog, err := prelude.LoadCatalog(g)
	if err != nil {
		return err
	}
	rt.SetHelpRegistry(catalog.Registry)
	return nil
}

// RunExpr evaluates a single expression and returns the result.
func RunExpr(expr string) (string, error) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return "", err
	}

	rt := substrate.NewGoRuntime()
	defer rt.CloseAllResources()

	// Create prelude environment with ScopePrelude
	preludeEnv := value.NewEnvWithScope(value.ScopePrelude)
	preludeEnv.SetAuthority(rt.AuthorityForSource("engine/runtime/prelude/prelude.nez"))

	// Load prelude
	if err := loadPrelude(g, rt, preludeEnv); err != nil {
		return "", fmt.Errorf("loading prelude: %w", err)
	}

	// Create user environment enclosed by prelude
	userEnv := value.NewEnclosedEnvWithScope(preludeEnv, value.ScopeUser)
	userEnv.SetAuthority(value.NoAuthority())

	lexer := syntax.NewLexer(g, "<expr>", expr, nil)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return "", err
	}

	parser := syntax.NewParser(g, tokens, expr, nil)
	ast, err := parser.Parse()
	if err != nil {
		return "", err
	}

	ev := evaluator.New(rt, nil)
	ev.SetGrammar(g)
	result := ev.Eval(ast, userEnv)
	if fault, ok := result.(*value.Fault); ok {
		return "", fmt.Errorf("%s", fault.Inspect())
	}

	return result.Inspect(), nil
}
