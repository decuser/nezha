package hostnative

import (
	"testing"

	"aiki/engine/semantics/value"
)

func TestRegisterResolveReturnsBoundOperation(t *testing.T) {
	const name = "test/implementation/register-resolve"
	calls := 0
	want := value.NewNumber(7, 1)
	Register(name, Implementation{Operations: map[string]Operation{
		"run": func(args []value.Value) value.Value {
			calls++
			if len(args) != 1 || args[0] != value.TRUE {
				t.Fatalf("args = %#v", args)
			}
			return want
		},
	}})

	op, err := Resolve(name, "run")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := op([]value.Value{value.TRUE}); got != want {
		t.Fatalf("operation result = %v, want %v", got, want)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestResolveRejectsUnknownImplementationAndOperation(t *testing.T) {
	const name = "test/implementation/unknown-operation"
	Register(name, Implementation{Operations: map[string]Operation{
		"known": func(args []value.Value) value.Value { return value.TRUE },
	}})
	if _, err := Resolve("test/implementation/does-not-exist", "known"); err == nil {
		t.Fatal("unknown implementation resolved")
	}
	if _, err := Resolve(name, "missing"); err == nil {
		t.Fatal("unknown operation resolved")
	}
}
