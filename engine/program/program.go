// Package program composes the Nezha engine into canonical program execution.
package program

import (
	"fmt"

	"nezha/engine/language/grammar"
	"nezha/engine/language/loader"
	"nezha/engine/runtime/hal"
	"nezha/engine/runtime/prelude"
	"nezha/engine/semantics/evaluator"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
)

// Runtime is the runtime capability required to construct and execute a
// canonical Nezha program. Evaluator behavior uses the HAL contract; program
// construction additionally needs the authority granted while evaluating the
// embedded prelude.
type Runtime interface {
	hal.RuntimeContract
	PreludeAuthority() value.Authority
}

// RunSource executes supplied Nezha source under the given filename in the
// canonical prelude/user environment.
func RunSource(rt Runtime, filename, source string) error {
	g, err := loadGrammar()
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

// EvalExpr evaluates a single expression in the canonical Nezha program
// environment and returns the inspected result.
func EvalExpr(rt Runtime, expr string) (string, error) {
	g, err := loadGrammar()
	if err != nil {
		return "", err
	}

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

func loadGrammar() (*grammar.Grammar, error) {
	return loader.Load("grammar.ebnfx", grammar.EbnfxSource, "grammar.help", grammar.HelpSource)
}

func evalSource(g *grammar.Grammar, rt Runtime, filename, source string, env *value.Env) error {
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

func loadPrelude(g *grammar.Grammar, rt Runtime, env *value.Env) error {
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
	return executionResultError(ev.Eval(ast, env))
}

func executionResultError(result value.Value) error {
	if fault, ok := result.(*value.Fault); ok {
		return fmt.Errorf("%s", fault.Inspect())
	}
	return nil
}
