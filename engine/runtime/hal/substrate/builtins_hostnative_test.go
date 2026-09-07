package substrate

import (
	"testing"

	"nezha/engine/runtime/hostnative"
	"nezha/engine/semantics/value"
)

func TestHostNativeBindResolvesOnceToDirectCallable(t *testing.T) {
	const implementationName = "test/substrate/hostnative-bind"
	calls := 0
	hostnative.Register(implementationName, hostnative.Implementation{Operations: map[string]hostnative.Operation{
		"step": func(args []value.Value) value.Value {
			calls++
			return value.TRUE
		},
	}})

	bound := halHostNativeBind([]value.Value{
		&value.String{Val: implementationName},
		&value.String{Val: "step"},
	}, nil)
	callable, ok := bound.(value.Callable)
	if !ok {
		t.Fatalf("bind result = %T, want callable", bound)
	}

	// Resolution is complete before steady-state execution. Removing or changing
	// registry lookup is not involved in Call; the callable owns the resolved fn.
	if got := callable.Call(nil); got != value.TRUE {
		t.Fatalf("call result = %v", got)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if named, ok := callable.(*hostNativeOperation); !ok || named.ProfileName() != implementationName+"/step" {
		t.Fatalf("bound callable profile identity = %#v", callable)
	}
}
