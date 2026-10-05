// Package program composes the Nezha engine into canonical program execution.
package program

import (
	"errors"
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

// ErrInterrupted reports that a host interrupted the current evaluation.
// It is execution control, not a Nezha fault and not a HAL capability.
var ErrInterrupted = errors.New("evaluation interrupted")

// RunSource executes supplied Nezha source under the given filename in the
// canonical prelude/user environment. One-shot execution is implemented as a
// short-lived Session so it remains semantically identical to session clients.
func RunSource(rt Runtime, filename, source string) error {
	s, err := NewSession(rt)
	if err != nil {
		return err
	}
	_, err = s.Eval(filename, source)
	return err
}

// EvalExpr evaluates a single expression in the canonical Nezha program
// environment and returns the inspected result.
func EvalExpr(rt Runtime, expr string) (string, error) {
	s, err := NewSession(rt)
	if err != nil {
		return "", err
	}
	result, err := s.Eval("<expr>", expr)
	if err != nil {
		return "", err
	}
	return result.Inspect(), nil
}

func loadGrammar() (*grammar.Grammar, error) {
	return loader.Load("grammar.ebnfx", grammar.EbnfxSource, "grammar.help", grammar.HelpSource)
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
