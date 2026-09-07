package substrate

import (
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

func halType(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 1 {
		return value.NewFault("type: want 1 argument, got %d", len(args))
	}
	return &value.Symbol{Val: string(args[0].Type())}
}

func halEqual(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 2 {
		return value.NewFault("equal: want 2 arguments, got %d", len(args))
	}
	if valuesEqual(args[0], args[1]) {
		return value.TRUE
	}
	return value.FALSE
}

func valuesEqual(a, b value.Value) bool {
	if a.Type() != b.Type() {
		return false
	}
	switch av := a.(type) {
	case *value.Number:
		bv := b.(*value.Number)
		return av.Compare(bv) == 0
	case *value.String:
		bv := b.(*value.String)
		return av.Val == bv.Val
	case *value.Symbol:
		bv := b.(*value.Symbol)
		return av.Val == bv.Val
	case *value.Boolean:
		bv := b.(*value.Boolean)
		return av.Val == bv.Val
	case *value.Rune:
		bv := b.(*value.Rune)
		return av.Val == bv.Val
	default:
		return false
	}
}
