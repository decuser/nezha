// Package runner provides the runner for executing Nezha programs.
package runner

import (
	"fmt"
	"os"

	"nezha/engine/runtime/hal/substrate"
	"nezha/engine/runtime/prelude"
	"nezha/engine/semantics/evaluator"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

func executionResultError(result value.Value) error {
	if fault, ok := result.(*value.Fault); ok {
		return fmt.Errorf("%s", fault.Inspect())
	}
	return nil
}

// Run executes a Nezha source file. programArgs are exposed to the program
// through system.args(), excluding the interpreter and source filename.
func Run(filename string, programArgs ...string) error {
	rt := substrate.NewGoRuntime()
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
	return runSourceWithRuntime(filename, source, nil, rt)
}

func runSourceWithRuntime(filename, source string, programArgs []string, rt *substrate.GoRuntime) error {
	return runSourceWithRuntimeDirect(filename, source, programArgs, rt)
}

// RunSourceWithRuntime executes supplied Nezha source under the given filename in
// the supplied runtime host world. The caller owns the runtime's lifetime.
func RunSourceWithRuntime(filename, source string, rt *substrate.GoRuntime, programArgs ...string) error {
	return runSourceWithRuntimeDirect(filename, source, programArgs, rt)
}

func runSourceWithRuntimeDirect(filename, source string, programArgs []string, rt *substrate.GoRuntime) error {
	// Load grammar
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return fmt.Errorf("loading grammar: %w", err)
	}

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
	lexer := syntax.NewLexer(g, filename, source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("lexer: %w", err)
	}

	// Parse user code
	parser := syntax.NewParser(g, tokens, source)
	ast, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parser: %w", err)
	}

	// Eval user code
	userEnv.SetFile(filename)
	userEnv.SetSource(source)
	ev := evaluator.New(rt)
	ev.SetGrammar(g)
	result := ev.Eval(ast, userEnv)
	return executionResultError(result)
}

// loadPrelude parses and evaluates the prelude.
func loadPrelude(g *grammar.Grammar, rt *substrate.GoRuntime, env *value.Env) error {
	// Initialize help registry
	if err := initHelpRegistry(g); err != nil {
		return fmt.Errorf("loading help: %w", err)
	}

	lexer := syntax.NewLexer(g, "<prelude>", prelude.Source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return err
	}

	parser := syntax.NewParser(g, tokens, prelude.Source)
	ast, err := parser.Parse()
	if err != nil {
		return err
	}

	env.SetFile("<prelude>")
	env.SetSource(prelude.Source)
	ev := evaluator.New(rt)
	ev.SetGrammar(g)
	result := ev.Eval(ast, env)
	if fault, ok := result.(*value.Fault); ok {
		return fmt.Errorf("%s", fault.Inspect())
	}

	return nil
}

// initHelpRegistry loads prelude help and doc files with validation.

func initHelpRegistry(g *grammar.Grammar) error {
	_, err := prelude.LoadCatalog(g)
	return err
}

// RunExpr evaluates a single expression and returns the result.
func RunExpr(expr string) (string, error) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return "", err
	}

	rt := substrate.NewGoRuntime()

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

	lexer := syntax.NewLexer(g, "<expr>", expr)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return "", err
	}

	parser := syntax.NewParser(g, tokens, expr)
	ast, err := parser.Parse()
	if err != nil {
		return "", err
	}

	ev := evaluator.New(rt)
	ev.SetGrammar(g)
	result := ev.Eval(ast, userEnv)
	if fault, ok := result.(*value.Fault); ok {
		return "", fmt.Errorf("%s", fault.Inspect())
	}

	return result.Inspect(), nil
}
