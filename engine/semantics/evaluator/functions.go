package evaluator

import (
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
)

func (e *Evaluator) applyFunction(fn value.Value, args []value.Value, node *syntax.Node, env *value.Env) value.Value {
	return e.applyFunctionOwned(fn, args, nil, node, env)
}

// CallValue invokes an already-resolved Nezha value through the ordinary call
// semantics. It exists for typed host adapters; it does not create a second
// calling convention.
func (e *Evaluator) CallValue(fn value.Value, args []value.Value, env *value.Env) value.Value {
	return e.applyFunction(fn, args, &syntax.Node{}, env)
}

// applyFunctionOwned is the internal call boundary for argument-frame ownership.
// argFrame is non-nil only for a proven non-escaping user function.
func (e *Evaluator) applyFunctionOwned(fn value.Value, args []value.Value, argFrame *argFrame, node *syntax.Node, env *value.Env) value.Value {
	switch f := fn.(type) {
	case *value.Function:
		return e.applyUserFunctionOwned(f, args, argFrame, node, env)
	case *ApplyIntrinsic:
		if argFrame != nil {
			defer e.releaseArgFrame(argFrame)
		}
		result := e.applyIntrinsic(args, node, env)
		if fault, ok := result.(*value.Fault); ok && fault.File == "" {
			fault.File = env.GetFile()
			fault.Line = node.Pos.Line
			fault.Source = env.GetSourceLine(node.Pos.Line)
			fault.Stack = env.CopyStack()
		}
		return result
	case value.Callable:
		if argFrame != nil {
			defer e.releaseArgFrame(argFrame)
		}

		result := f.Call(args)
		// Annotate HAL faults with call site location.
		if fault, ok := result.(*value.Fault); ok && fault.File == "" {
			fault.File = env.GetFile()
			fault.Line = node.Pos.Line
			fault.Source = env.GetSourceLine(node.Pos.Line)
			fault.Stack = env.CopyStack()
		}
		return result
	default:
		if argFrame != nil {
			e.releaseArgFrame(argFrame)
		}
		return e.makeFault(node, env, "not a function: %s", fn.Type())
	}
}

func (e *Evaluator) applyUserFunctionOwned(fn *value.Function, args []value.Value, initialArgFrame *argFrame, node *syntax.Node, env *value.Env) value.Value {
	currentFn := fn
	currentArgs := args
	currentArgFrame := initialArgFrame
	callSite := node
	pushed := false
	var reusableCallEnv *value.Env

	for {
		fnEnv, ok := currentFn.Env.(*value.Env)
		if !ok {
			if currentArgFrame != nil {
				e.releaseArgFrame(currentArgFrame)
			}
			return e.makeFault(callSite, env, "invalid function environment")
		}

		// Check argument count (excluding rest params)
		if len(currentArgs) < len(currentFn.Params) {
			if currentArgFrame != nil {
				e.releaseArgFrame(currentArgFrame)
			}
			return e.makeFault(callSite, env, "%s: want %d arguments, got %d", currentFn.Name, len(currentFn.Params), len(currentArgs))
		}

		var callEnv *value.Env
		if reusableCallEnv != nil {
			callEnv = reusableCallEnv
			callEnv.ResetCallEnv(fnEnv, env)
			reusableCallEnv = nil
		} else {
			callEnv = value.NewCallEnv(fnEnv, env)
		}
		callEnv.BindCallParams(currentFn.Params, currentArgs)

		if currentFn.Rest != "" {
			restStart := len(currentFn.Params)
			var restArgs []value.Value
			if restStart < len(currentArgs) {
				restArgs = currentArgs[restStart:]
			}
			callEnv.Set(currentFn.Rest, &value.List{Elements: restArgs})
		}

		body, ok := currentFn.Body.(*syntax.Node)
		if !ok {
			if currentArgFrame != nil {
				callEnv.ClearCallParams()
				e.releaseArgFrame(currentArgFrame)
			}
			return e.makeFault(callSite, env, "invalid function body")
		}

		funcName := currentFn.Name
		if funcName == "" {
			funcName = "<anonymous>"
		}

		// Enforce non tail call stack limit on first entry only.
		line := callSite.Pos.Line
		scope := callEnv.GetScope()
		if !pushed {
			limit := callEnv.GetStackLimit()
			if limit > 0 && callEnv.StackDepth() >= limit {
				if currentArgFrame != nil {
					callEnv.ClearCallParams()
					e.releaseArgFrame(currentArgFrame)
				}
				return e.makeFault(callSite, callEnv, "stack overflow")
			}
			callEnv.PushFrame(funcName, line, scope)
			pushed = true
		} else {
			// Tail call - reuse the top frame.
			callEnv.ReplaceTopFrame(funcName, line, scope)
		}

		result := e.evalTail(body, callEnv)

		// Tail call jump
		if tc, ok := result.(*tailCallValue); ok {
			if currentFn.TailEnvReusable {
				reusableCallEnv = callEnv
			}
			if currentArgFrame != nil {
				callEnv.ClearCallParams()
				e.releaseArgFrame(currentArgFrame)
			}
			currentFn = tc.Fn
			currentArgs = tc.Args
			currentArgFrame = tc.ArgFrame
			callSite = tc.Node
			continue
		}

		// Return control flow
		if ret, ok := result.(*value.Return); ok {
			if tc, ok := ret.Val.(*tailCallValue); ok {
				if currentFn.TailEnvReusable {
					reusableCallEnv = callEnv
				}
				if currentArgFrame != nil {
					callEnv.ClearCallParams()
					e.releaseArgFrame(currentArgFrame)
				}
				currentFn = tc.Fn
				currentArgs = tc.Args
				currentArgFrame = tc.ArgFrame
				callSite = tc.Node
				continue
			}
			if currentArgFrame != nil {
				callEnv.ClearCallParams()
				e.releaseArgFrame(currentArgFrame)
			}
			callEnv.PopFrame()
			return ret.Val
		}

		if shouldHalt(result) {
			if currentArgFrame != nil {
				callEnv.ClearCallParams()
				e.releaseArgFrame(currentArgFrame)
			}
			callEnv.PopFrame()
			return result
		}

		if currentArgFrame != nil {
			callEnv.ClearCallParams()
			e.releaseArgFrame(currentArgFrame)
		}
		callEnv.PopFrame()
		return result
	}
}

func funcNameValue(fn *value.Function) string {
	if fn == nil {
		return "<fn>"
	}
	if fn.Name != "" {
		return fn.Name
	}
	return "<anonymous>"
}

func (e *Evaluator) extractParams(node *syntax.Node) ([]string, string) {
	count := 0
	for _, child := range node.Children {
		if child.Type == "param_list" {
			for _, p := range child.Children {
				if p.Type == "NAME" {
					count++
				}
			}
		} else if child.Type == "NAME" {
			count++
		}
	}

	var params []string
	if count > 0 {
		params = make([]string, 0, count)
	}
	var rest string

	for _, child := range node.Children {
		if child.Type == "param_list" {
			for _, p := range child.Children {
				if p.Type == "NAME" {
					params = append(params, p.Value)
				}
			}
		}
		if child.Type == "rest_param" {
			for _, p := range child.Children {
				if p.Type == "NAME" {
					rest = p.Value
				}
			}
		}
		if child.Type == "NAME" {
			params = append(params, child.Value)
		}
	}

	return params, rest
}
