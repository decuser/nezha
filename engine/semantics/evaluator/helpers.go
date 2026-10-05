package evaluator

import (
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
)

// interruptValue is an evaluator-private control sentinel. Like tailCallValue,
// it exists only to move engine control through the value-returning evaluator
// without adding a script-visible value kind.
type interruptValue struct{}

func (*interruptValue) Type() value.Type { return value.Type("interrupt") }
func (*interruptValue) Inspect() string  { return "<interrupt>" }

var interrupted value.Value = &interruptValue{}

// isInterrupted reports whether a result is the evaluator-private interruption sentinel.
func isInterrupted(v value.Value) bool {
	_, ok := v.(*interruptValue)
	return ok
}

// shouldHalt returns true if evaluation should halt on this value.
// Faults and host interruption halt evaluation. Shaped errors flow as data.
func shouldHalt(v value.Value) bool {
	return value.IsFault(v) || isInterrupted(v)
}

func (e *Evaluator) nonTerminalChildren(node *syntax.Node) []*syntax.Node {
	var result []*syntax.Node
	for _, child := range node.Children {
		if child.Type != "TERMINAL" {
			result = append(result, child)
		}
	}
	return result
}

func (e *Evaluator) makeFault(node *syntax.Node, env *value.Env, format string, args ...interface{}) *value.Fault {
	return value.NewFaultAt(
		env.GetFile(),
		node.Pos.Line,
		env.GetSourceLine(node.Pos.Line),
		env.CopyStack(),
		format,
		args...,
	)
}

// resolveTailCalls drains internal tail call sentinels so they never escape evaluator boundaries.
func (e *Evaluator) resolveTailCalls(v value.Value, env *value.Env) value.Value {
	for {
		tc, ok := v.(*tailCallValue)
		if !ok {
			return v
		}
		v = e.applyUserFunctionOwned(tc.Fn, tc.Args, tc.ArgFrame, tc.Node, env)
	}
}
