package substrate

import (
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
)

// halApply implements apply(fn, list) - spreads list as args.
func halApply(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 2 {
		return value.NewFault("apply: want 2 arguments, got %d", len(args))
	}

	fn := args[0]
	listVal := args[1]

	list, ok := listVal.(*value.List)
	if !ok {
		return value.NewFault("apply: second argument must be list, got %s", listVal.Type())
	}

	switch f := fn.(type) {
	case *value.Function:
		return applyUserFunction(f, list.Elements, ctx)
	case hal.ContextCallable:
		return f.CallWithContext(list.Elements, ctx)
	case value.Callable:
		return f.Call(list.Elements)
	default:
		return value.NewFault("apply: first argument must be function, got %s", fn.Type())
	}
}

// funcName returns a function's name for diagnostics, or a placeholder when
// the function is anonymous.
func funcName(fn *value.Function) string {
	if fn.Name != "" {
		return fn.Name
	}
	return "function"
}

// applyUserFunction calls a user-defined function with args.
func applyUserFunction(fn *value.Function, args []value.Value, ctx *hal.EvalContext) value.Value {
	if ctx == nil || ctx.Eval == nil {
		return value.NewFault("apply: evaluation context not available")
	}

	fnEnv, ok := fn.Env.(*value.Env)
	if !ok {
		return value.NewFault("apply: invalid function environment")
	}

	// Same minimum-arity rule as ordinary function application.
	if len(args) < len(fn.Params) {
		return value.NewFault("%s: want %d arguments, got %d", funcName(fn), len(fn.Params), len(args))
	}

	callEnv := value.NewCallEnv(fnEnv, ctx.Env)
	for i, param := range fn.Params {
		callEnv.Set(param, args[i])
	}

	if fn.Rest != "" {
		restStart := len(fn.Params)
		var restArgs []value.Value
		if restStart < len(args) {
			restArgs = args[restStart:]
		}
		callEnv.Set(fn.Rest, &value.List{Elements: restArgs})
	}

	body, ok := fn.Body.(*syntax.Node)
	if !ok {
		return value.NewFault("apply: invalid function body")
	}

	result := ctx.Eval(body, callEnv)
	if ret, ok := result.(*value.Return); ok {
		return ret.Val
	}
	return result
}
