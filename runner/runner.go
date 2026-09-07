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
	return RunSourceWithRuntime(filename, string(source), rt)
}

// RunSource executes Nezha source code with the standard Go runtime substrate.
func RunSource(filename, source string) error {
	rt := substrate.NewGoRuntime()
	return RunSourceWithRuntime(filename, source, rt)
}

// RunSourceWithRuntime executes supplied Nezha source under the given filename in
// the supplied runtime host world. The caller owns the runtime's lifetime.
func RunSourceWithRuntime(filename, source string, rt *substrate.GoRuntime) error {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return fmt.Errorf("loading grammar: %w", err)
	}

	preludeEnv := value.NewEnvWithScope(value.ScopePrelude)
	preludeEnv.SetAuthority(rt.PreludeAuthority())
	if err := loadPrelude(g, rt, preludeEnv); err != nil {
		return fmt.Errorf("loading prelude: %w", err)
	}

	userEnv := value.NewEnclosedEnvWithScope(preludeEnv, value.ScopeUser)
	userEnv.SetAuthority(value.NoAuthority())
	return evalSource(g, rt, filename, source, userEnv)
}

func evalSource(g *grammar.Grammar, rt *substrate.GoRuntime, filename, source string, env *value.Env) error {
	lexer := syntax.NewLexer(g, filename, source)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return fmt.Errorf("lexer: %w", err)
	}

	parser := syntax.NewParser(g, tokens, source)
	ast, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("parser: %w", err)
	}

	env.SetFile(filename)
	env.SetSource(source)
	ev := evaluator.New(rt)
	ev.SetGrammar(g)
	return executionResultError(ev.Eval(ast, env))
}

// loadPrelude parses and evaluates the prelude.
func loadPrelude(g *grammar.Grammar, rt *substrate.GoRuntime, env *value.Env) error {
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

// initHelpRegistry validates the prelude help and doc files.
func initHelpRegistry(g *grammar.Grammar) error {
	_, err := prelude.LoadCatalog(g)
	return err
}

// RunExpr evaluates a single expression and returns the inspected result.
func RunExpr(expr string) (string, error) {
	g, err := grammar.Load("grammar.ebnfx", syntax.EbnfxSource, "grammar.help", syntax.HelpSource)
	if err != nil {
		return "", err
	}

	rt := substrate.NewGoRuntime()
	preludeEnv := value.NewEnvWithScope(value.ScopePrelude)
	preludeEnv.SetAuthority(rt.PreludeAuthority())
	if err := loadPrelude(g, rt, preludeEnv); err != nil {
		return "", fmt.Errorf("loading prelude: %w", err)
	}

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
