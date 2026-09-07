package evaluator

import (
	"fmt"

	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
)

const applyIntrinsicName = "_apply"

// ApplyIntrinsic is evaluator-owned because applying a Nezha function evaluates
// Nezha bodies and must use the evaluator's ordinary call path.
type ApplyIntrinsic struct{}

func (a *ApplyIntrinsic) Type() value.Type { return value.FunctionType }
func (a *ApplyIntrinsic) Inspect() string  { return fmt.Sprintf("<intrinsic: %s>", applyIntrinsicName) }

// Call exists only to satisfy value.Callable. The evaluator recognizes
// ApplyIntrinsic before the generic callable path so normal execution never
// reaches this fallback.
func (a *ApplyIntrinsic) Call(args []value.Value) value.Value {
	return value.NewFault("%s: evaluator context not available", applyIntrinsicName)
}

var _ value.Callable = (*ApplyIntrinsic)(nil)

func (e *Evaluator) evaluatorIntrinsic(name string, authority value.Authority) (value.Callable, bool) {
	if name != applyIntrinsicName || !authority.Allows(applyIntrinsicName) {
		return nil, false
	}
	return &ApplyIntrinsic{}, true
}

func (e *Evaluator) applyIntrinsic(args []value.Value, node *syntax.Node, env *value.Env) value.Value {
	if len(args) != 2 {
		return value.NewFault("apply: want 2 arguments, got %d", len(args))
	}

	fn := args[0]
	listVal := args[1]

	list, ok := listVal.(*value.List)
	if !ok {
		return value.NewFault("apply: second argument must be list, got %s", listVal.Type())
	}

	return e.applyFunction(fn, list.Elements, node, env)
}
