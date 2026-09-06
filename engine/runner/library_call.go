package runner

import (
	"fmt"

	"aiki/engine/runtime/hal"
	"aiki/engine/semantics/value"
)

// CallLibrary loads a named Aiki package in a normal initialized session and
// invokes one exported function with already-typed Aiki values. It is the
// narrow Go-to-Aiki adapter for Go CLI plumbing that delegates semantic work to
// an Aiki library instead of reconstructing that work in Go.
func CallLibrary(moduleName, functionName string, args ...value.Value) (value.Value, error) {
	s, err := NewSession()
	if err != nil {
		return nil, err
	}
	defer s.Runtime.CloseAllResources()

	ctx := &hal.EvalContext{
		Env:     s.Env,
		Grammar: s.Grammar,
		Eval:    s.Evaluator.Eval,
	}
	mod, errVal := s.Runtime.LoadModuleForHost(moduleName, ctx)
	if errVal != nil {
		return nil, fmt.Errorf("loading %s: %s", moduleName, errVal.Inspect())
	}
	fn, ok := mod.Get(functionName)
	if !ok {
		return nil, fmt.Errorf("%s: function %q is not exported", moduleName, functionName)
	}
	result := s.Evaluator.CallValue(fn, args, s.Env)
	if fault, ok := result.(*value.Fault); ok {
		return nil, fmt.Errorf("%s", fault.Inspect())
	}
	return result, nil
}
