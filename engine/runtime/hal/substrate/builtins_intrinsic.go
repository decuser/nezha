package substrate

import (
	"os"

	"aiki/engine"
	"aiki/engine/observe"
	"aiki/engine/runtime/hal"
	"aiki/engine/semantics/value"
	"aiki/engine/syntax"
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

// halSpawn implements spawn(fn, args...) - runs function concurrently.
// Spawned functions run with isolated env - only args are visible, no closure capture.
func halSpawn(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) < 1 {
		return value.NewFault("spawn: want at least 1 argument (function)")
	}

	if ctx == nil || ctx.Eval == nil {
		return value.NewFault("spawn: evaluation context not available")
	}

	fn, ok := args[0].(*value.Function)
	if !ok {
		return value.NewFault("spawn: expected function as first argument, got %s", args[0].Type())
	}
	if ctx.ReportAsyncFault == nil {
		return value.NewFault("spawn: asynchronous fault reporting not available")
	}

	// Capture arguments passed to spawn: spawn(fn, arg1, arg2)
	fnArgs := args[1:]
	semanticHitDetail(ctx, engine.SemanticCall, funcName(fn))

	// Launch goroutine with isolated env
	go func() {
		result := applyUserFunctionIsolated(fn, fnArgs, ctx)
		if fault, ok := result.(*value.Fault); ok {
			ctx.ReportAsyncFault(fault)
		}
	}()

	return value.TRUE
}

// applyUserFunctionIsolated runs a function with fresh env - no closure capture.
// Only the passed arguments are visible to the function, plus prelude bindings.
func applyUserFunctionIsolated(fn *value.Function, args []value.Value, ctx *hal.EvalContext) value.Value {
	if ctx == nil || ctx.Eval == nil {
		return value.NewFault("spawn: evaluation context not available")
	}

	// Get prelude env from context - spawned fn can see prelude but not user bindings
	preludeEnv := ctx.Env.GetPreludeEnv()
	if preludeEnv == nil {
		return value.NewFault("spawn: could not find prelude environment")
	}

	// Same minimum-arity rule as ordinary function application.
	if len(args) < len(fn.Params) {
		return value.NewFault("%s: want %d arguments, got %d", funcName(fn), len(fn.Params), len(args))
	}

	// Fresh env enclosed by prelude - sees prelude bindings but not outer user scope
	authority := value.NoAuthority()
	if fnEnv, ok := fn.Env.(*value.Env); ok && fnEnv != nil {
		authority = fnEnv.GetAuthority()
	}
	callEnv := value.NewIsolatedEnclosedEnvWithAuthority(preludeEnv, authority)
	callEnv.SetSemanticProbe(ctx.Probe)

	// Copy the shape vocabulary visible where the function was created.
	// A shape definition is an ordered list of field names, not a value: it
	// carries no data out of the spawning environment, and the shape name
	// already crosses the boundary on the value itself. Without this, a shaped
	// argument arrives intact but field access on it faults "unknown shape".
	if fnEnv, ok := fn.Env.(*value.Env); ok && fnEnv != nil {
		// Source-file provenance is lexical metadata, not captured program state.
		// Preserve it across spawn so relative imports performed by the spawned
		// function resolve relative to the file that defined that function rather
		// than falling back to the process working directory.
		callEnv.SetFile(fnEnv.GetFile())

		for _, def := range fnEnv.CollectShapes() {
			callEnv.DefineShape(def)
		}
	}

	// Bind parameters, under the same minimum-arity rule as ordinary calls.
	for i, param := range fn.Params {
		callEnv.Set(param, args[i])
	}

	// Bind rest parameter
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
		return value.NewFault("spawn: invalid function body")
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

// containsStr checks if a string slice contains a value.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
