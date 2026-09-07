package substrate

import (
	"nezha/engine/semantics/value"
)

func halType(args []value.Value) value.Value {
	return &value.Symbol{Val: string(args[0].Type())}
}

func halEqual(args []value.Value) value.Value {
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
