package substrate

import (
	"testing"

	"nezha/engine/runtime/hal"
	"nezha/engine/runtime/primitives"
	"nezha/engine/semantics/value"
)

func TestBuiltinRegistryMatchesPrimitiveCatalog(t *testing.T) {
	rt := NewGoRuntime()

	definitions := primitives.Definitions()
	registrations := rt.PrimitiveRegistrations()

	if got, want := len(registrations), len(definitions); got != want {
		t.Fatalf("registered %d primitives, architecture defines %d", got, want)
	}
	for name := range definitions {
		if _, ok := registrations[name]; !ok {
			t.Errorf("architectural primitive %s has no substrate binding", name)
		}
	}
	for name := range registrations {
		if !definitions[name] {
			t.Errorf("substrate primitive %s has no architectural definition", name)
		}
	}

	for _, name := range []string{"_first", "_print"} {
		if _, ok := rt.lookupBuiltin(name); !ok {
			t.Errorf("runtime lookup lost %s", name)
		}
	}

	for _, name := range []string{"_dot", "_line", "_rect", "_fill_rect", "_circle", "_fill_circle", "_arc", "_clear", "_set_bg", "_set_fg", "_pen_size", "_set_turtle"} {
		if _, ok := rt.lookupBuiltin(name); ok {
			t.Errorf("obsolete Canvas compatibility primitive still registered: %s", name)
		}
	}

	if got, want := len(rt.hostBindings), len(hal.OperationDefinitions()); got != want {
		t.Errorf("canonical host bindings = %d, HAL defines %d", got, want)
	}
}

func TestBuiltinArityIsEnforcedAtDispatch(t *testing.T) {
	rt := NewGoRuntime()
	b, ok := rt.lookupBuiltin("_first")
	if !ok {
		t.Fatal("_first not registered")
	}
	got := b.Call(nil)
	fault, ok := got.(*value.Fault)
	if !ok {
		t.Fatalf("wrong arity returned %T, want fault", got)
	}
	if want := "first: want 1 argument, got 0"; fault.Message != want {
		t.Fatalf("wrong arity fault = %q, want %q", fault.Message, want)
	}
}
