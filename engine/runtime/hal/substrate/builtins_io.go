package substrate

import (
	"fmt"

	"nezha/engine/semantics/value"
)

func (g *GoRuntime) halPrint(args []value.Value) value.Value {
	for _, arg := range args {
		if s, ok := arg.(*value.String); ok {
			fmt.Fprint(g.stdout, s.Val)
		} else {
			fmt.Fprint(g.stdout, arg.Inspect())
		}
	}
	return value.EMPTY
}
