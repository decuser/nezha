package substrate

import (
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

func halFirst(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("first: want 1 argument, got %d", len(args))
	}
	switch a := args[0].(type) {
	case *value.List:
		if a.Len() == 0 {
			return value.NewFault("first: empty list")
		}
		return a.At(0)
	case *value.String:
		r, ok := a.FirstRune()
		if !ok {
			return value.NewFault("first: empty string")
		}
		return &value.Rune{Val: r}
	default:
		return value.NewFault("first: expected list or string")
	}
}

func halRest(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("rest: want 1 argument, got %d", len(args))
	}
	switch a := args[0].(type) {
	case *value.List:
		if a.Len() == 0 {
			return &value.List{Elements: []value.Value{}}
		}
		return &value.List{Elements: a.LogicalElements()[1:]}
	case *value.String:
		runes := []rune(a.Val)
		if len(runes) == 0 {
			return &value.String{Val: ""}
		}
		return &value.String{Val: string(runes[1:])}
	default:
		return value.NewFault("rest: expected list or string")
	}
}

func halLength(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("length: want 1 argument, got %d", len(args))
	}
	switch a := args[0].(type) {
	case *value.List:
		return value.NewNumber(int64(a.Len()), 1)
	case *value.String:
		return value.NewNumber(int64(a.RuneLen()), 1)
	default:
		return value.NewFault("length: expected list or string")
	}
}

func halPrepend(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 2 {
		return value.NewFault("prepend: want 2 arguments, got %d", len(args))
	}
	list, ok := args[0].(*value.List)
	if !ok {
		return value.NewFault("prepend: first argument must be list")
	}
	newElems := make([]value.Value, list.Len()+1)
	newElems[0] = args[1]
	copy(newElems[1:], list.LogicalElements())
	return &value.List{Elements: newElems, Shape: list.Shape}
}

func halAppend(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 2 {
		return value.NewFault("append: want 2 arguments, got %d", len(args))
	}
	list, ok := args[0].(*value.List)
	if !ok {
		return value.NewFault("append: first argument must be list")
	}
	result, _ := list.Append(args[1])
	return result
}

func halEmpty(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("empty: want 1 argument, got %d", len(args))
	}
	switch a := args[0].(type) {
	case *value.List:
		if a.Len() == 0 {
			return value.TRUE
		}
		return value.FALSE
	case *value.String:
		if len(a.Val) == 0 {
			return value.TRUE
		}
		return value.FALSE
	default:
		return value.NewFault("empty: expected list or string")
	}
}

// halMakeShapedList creates a shaped list from a shape name and elements list.
func halMakeShapedList(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 2 {
		return value.NewFault("make_shaped_list: want 2 arguments, got %d", len(args))
	}
	sym, ok := args[0].(*value.Symbol)
	if !ok {
		return value.NewFault("make_shaped_list: expected symbol for shape, got %s", args[0].Type())
	}
	list, ok := args[1].(*value.List)
	if !ok {
		return value.NewFault("make_shaped_list: expected list for elements, got %s", args[1].Type())
	}
	return &value.List{Elements: list.LogicalElements(), Shape: sym.Val}
}
