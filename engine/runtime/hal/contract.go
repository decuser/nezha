// Package hal defines the runtime contract that the evaluator uses
// to interact with the host platform.
package hal

import (
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// ContextCallable is a host callable that needs the active Nezha evaluation
// context.
type ContextCallable interface {
	value.Callable
	CallWithContext(args []value.Value, ctx *EvalContext) value.Value
}

// EvalContextRequired is an optional callable capability. ContextCallable
// preserves the existing host-call ABI, while this capability tells the
// evaluator whether constructing an EvalContext can affect the call. A
// ContextCallable that does not require context may be invoked through Call.
type EvalContextRequired interface {
	NeedsEvalContext() bool
}

// EvalContext provides evaluation context to builtins that need it.
// Most builtins ignore this; intrinsics such as apply use it to
// access the evaluator, grammar, and environment.
type EvalContext struct {
	Env     *value.Env
	Node    *syntax.Node
	Grammar *grammar.Grammar
	// Eval is a callback to evaluate AST nodes. Used by apply.
	Eval func(*syntax.Node, *value.Env) value.Value
}

// RuntimeContract defines the behavioral requirements of the evaluator.
// The evaluator delegates all host-level effects through this interface.
type RuntimeContract interface {
	// Execute calls a registered primitive function by name.
	// Context provides evaluation state for intrinsics that need it.
	Execute(name string, args []value.Value, ctx *EvalContext) (value.Value, error)

	// HasBuiltin checks whether a name is executable under the supplied authority.
	HasBuiltin(name string, authority value.Authority) bool

	// GetBuiltin returns a callable for the named builtin under the supplied authority.
	GetBuiltin(name string, authority value.Authority) (value.Callable, bool)
}
