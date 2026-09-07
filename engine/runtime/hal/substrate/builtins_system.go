package substrate

import (
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

func (g *GoRuntime) halModuleRoots(args []value.Value, ctx *hal.EvalContext) value.Value {
	if len(args) != 0 {
		return value.NewFault("module_roots: want 0 arguments, got %d", len(args))
	}
	if g.moduleRegistry == nil {
		return value.NewFault("module_roots: module registry is not initialized")
	}
	roots := g.moduleRegistry.Roots()
	elements := make([]value.Value, len(roots))
	for i, root := range roots {
		elements[i] = &value.String{Val: root}
	}
	return &value.List{Elements: elements}
}
