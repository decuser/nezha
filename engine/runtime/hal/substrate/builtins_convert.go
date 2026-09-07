package substrate

import (
	"nezha/engine/semantics/value"
)

// halShape returns the shape of a list, or :list if unshaped.
func halShape(args []value.Value) value.Value {
	list, ok := args[0].(*value.List)
	if !ok {
		return &value.Symbol{Val: "list"}
	}
	if list.Shape == "" {
		return &value.Symbol{Val: "list"}
	}
	return &value.Symbol{Val: list.Shape}
}

// halToStr converts a value to its string representation.
// Unlike Inspect(), this returns the value content, not display format.
func halToStr(args []value.Value) value.Value {
	switch v := args[0].(type) {
	case *value.Rune:
		return &value.String{Val: string(v.Val)}
	case *value.String:
		return v
	case *value.Number:
		return &value.String{Val: v.RatString()}
	case *value.Boolean:
		if v.Val {
			return &value.String{Val: "true"}
		}
		return &value.String{Val: "false"}
	case *value.Symbol:
		return &value.String{Val: ":" + v.Val}
	default:
		return &value.String{Val: args[0].Inspect()}
	}
}
