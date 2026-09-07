package substrate

import (
	"os"

	"nezha/engine"
	"nezha/engine/observe"
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

	// Dispatch based on function type. The execution path remains the intrinsic's
	// existing direct dispatch; profiling only attaches the already-resolved
	// caller/callee cursor to that transition.
	targetCtx := ctx
	if ctx != nil && ctx.ProfileCall != nil {
		child := *ctx
		child.ProfileCursor = ctx.ProfileCall(fn, ctx.ProfileCursor)
		targetCtx = &child
	}
	var timing *observe.TimingStack
	if ctx != nil && targetCtx != nil && ctx.ProfileTiming != nil && targetCtx.ProfileCursor.Node() != 0 {
		timing = ctx.ProfileTiming
		timing.Push(ctx.ProfileCursor.Node(), targetCtx.ProfileCursor.Node())
	}
	call := func() value.Value {
		switch f := fn.(type) {
		case *value.Function:
			semanticHitDetail(ctx, engine.SemanticCall, funcName(f))
			return applyUserFunction(f, list.Elements, targetCtx)
		case hal.ContextCallable:
			return f.CallWithContext(list.Elements, targetCtx)
		case value.Callable:
			return f.Call(list.Elements)
		default:
			return value.NewFault("apply: first argument must be function, got %s", fn.Type())
		}
	}
	var result value.Value
	if ctx != nil && targetCtx != nil && ctx.ProfileAllocationCall != nil {
		result = ctx.ProfileAllocationCall(ctx.ProfileCursor, targetCtx.ProfileCursor, call)
	} else {
		result = call()
	}
	if timing != nil {
		timing.Pop()
	}
	return result
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
	callEnv.SetSemanticProbe(ctx.Probe)
	if ctx.ProfileCursor.Node() != 0 {
		callEnv.SetProfileCursor(ctx.ProfileCursor)
	}
	if ctx.ProfileTiming != nil {
		callEnv.SetProfileTiming(ctx.ProfileTiming)
	}

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

	var result value.Value
	run := func() { result = ctx.Eval(body, callEnv) }
	if ctx.WithProfileLabels != nil {
		ctx.WithProfileLabels(engine.ProfileLabels{
			Layer:    "semantic",
			Function: funcName(fn),
			File:     callEnv.GetFile(),
		}, ctx.Labels, run)
	} else {
		run()
	}

	if ret, ok := result.(*value.Return); ok {
		return ret.Val
	}

	return result
}

// halExport implements export(:name1, :name2, ...).
// Records exported names on the environment.
func halExport(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) == 0 {
		return value.NewFault("export: want at least 1 argument")
	}

	if ctx == nil || ctx.Env == nil {
		return value.NewFault("export: environment not available")
	}

	var names []string
	for _, arg := range args {
		sym, ok := arg.(*value.Symbol)
		if !ok {
			return value.NewFault("export: expected symbol, got %s", arg.Type())
		}
		names = append(names, sym.Val)
	}

	ctx.Env.SetExports(names)
	return value.EMPTY
}

// halImport validates the call shape for the historical import primitive, but
// Nezha no longer has a module loader behind it.
func (g *GoRuntime) halImport(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) < 1 {
		return value.NewFault("import: want at least module name")
	}
	if _, ok := args[0].(*value.String); !ok {
		return value.NewFault("import: expected string module name, got %s", args[0].Type())
	}
	for _, arg := range args[1:] {
		if _, ok := arg.(*value.Symbol); !ok {
			return value.NewFault("import: expected symbol, got %s", arg.Type())
		}
	}
	return value.NewShapedError("import", "import: modules are not available")
}

// halUse validates the call shape for the historical use primitive, but Nezha
// no longer has a module loader behind it.
func (g *GoRuntime) halUse(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("use: want 1 argument, got %d", len(args))
	}
	if _, ok := args[0].(*value.String); !ok {
		return value.NewFault("use: expected string module name, got %s", args[0].Type())
	}
	return value.NewShapedError("use", "use: modules are not available")
}

func halLoad(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("load: want 1 argument, got %d", len(args))
	}

	if ctx == nil || ctx.Grammar == nil || ctx.Eval == nil || ctx.Env == nil {
		return value.NewFault("load: evaluation context not available")
	}

	pathStr, ok := args[0].(*value.String)
	if !ok {
		return value.NewFault("load: expected string path, got %s", args[0].Type())
	}

	modulePath := resolveModulePath(pathStr.Val, ctx.Env)
	if modulePath == "" {
		return value.NewShapedError("load", "load: cannot find '%s'", pathStr.Val)
	}

	data, err := os.ReadFile(modulePath)
	if err != nil {
		return value.NewShapedError("load", "load: cannot read '%s': %s", modulePath, err)
	}

	lexer := syntax.NewLexer(ctx.Grammar, modulePath, string(data), nil)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return value.NewShapedError("load", "load: lex error in '%s': %s", modulePath, err)
	}

	parser := syntax.NewParser(ctx.Grammar, tokens, string(data), nil)
	ast, err := parser.Parse()
	if err != nil {
		return value.NewShapedError("load", "load: parse error in '%s': %s", modulePath, err)
	}

	// Evaluate in current environment
	return ctx.Eval(ast, ctx.Env)
}
