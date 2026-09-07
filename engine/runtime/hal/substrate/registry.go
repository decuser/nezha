package substrate

import (
	"fmt"

	"nezha/engine/runtime/primitives"
)

func (g *GoRuntime) lookupBuiltin(name string) (*Builtin, bool) {
	b, ok := g.builtins[name]
	return b, ok
}

func (g *GoRuntime) registerPrimitive(name string, arity Arity, fn BuiltinFunc) {
	if !primitives.IsRuntimePrimitive(name) {
		panic("runtime primitive has no architectural definition: " + name)
	}
	g.registerBuiltin(name, arity, fn)
}

func (g *GoRuntime) registerBuiltin(name string, arity Arity, fn BuiltinFunc) {
	if fn == nil {
		panic("runtime primitive registration has nil function: " + name)
	}
	if !arity.valid() {
		panic("runtime primitive registration has invalid arity: " + name)
	}
	if _, exists := g.lookupBuiltin(name); exists {
		panic("duplicate runtime primitive registration: " + name)
	}
	g.builtins[name] = &Builtin{name: name, arity: arity, fn: fn}
}

// PrimitiveRegistrations returns the runtime's actual primitive registrations.
// The returned map is a validation snapshot; callers cannot mutate runtime
// dispatch through it.
func (g *GoRuntime) PrimitiveRegistrations() map[string]Arity {
	out := map[string]Arity{}
	for name, builtin := range g.builtins {
		out[name] = builtin.arity
	}
	return out
}

// String is intentionally compact for test diagnostics.
func (a Arity) String() string {
	if a.Max == -1 {
		return fmt.Sprintf("%d..", a.Min)
	}
	if a.Min == a.Max {
		return fmt.Sprintf("%d", a.Min)
	}
	return fmt.Sprintf("%d..%d", a.Min, a.Max)
}
