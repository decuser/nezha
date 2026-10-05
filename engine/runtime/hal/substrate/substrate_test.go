package substrate

import (
	"bytes"
	"strings"
	"testing"

	"nezha/engine/semantics/value"
)

func callBuiltin(t *testing.T, rt *GoRuntime, name string, args []value.Value) value.Value {
	t.Helper()
	b, ok := rt.lookupBuiltin(name)
	if !ok {
		t.Fatalf("missing builtin %s", name)
	}
	return b.Call(args)
}

func TestFirst(t *testing.T) {
	rt := NewGoRuntime()
	list := &value.List{Elements: []value.Value{value.NewNumber(1, 1), value.NewNumber(2, 1)}}
	v := callBuiltin(t, rt, "_first", []value.Value{list})
	if v.Inspect() != "1" {
		t.Errorf("got %s", v.Inspect())
	}
}

func TestRest(t *testing.T) {
	rt := NewGoRuntime()
	list := &value.List{Elements: []value.Value{value.NewNumber(1, 1), value.NewNumber(2, 1)}}
	v := callBuiltin(t, rt, "_rest", []value.Value{list})
	if v.Inspect() != "[2]" {
		t.Errorf("got %s", v.Inspect())
	}
}

func TestLength(t *testing.T) {
	rt := NewGoRuntime()
	list := &value.List{Elements: []value.Value{value.NewNumber(1, 1), value.NewNumber(2, 1)}}
	v := callBuiltin(t, rt, "_length", []value.Value{list})
	if v.Inspect() != "2" {
		t.Errorf("got %s", v.Inspect())
	}
}

func TestEmpty(t *testing.T) {
	rt := NewGoRuntime()
	v := callBuiltin(t, rt, "_empty", []value.Value{&value.List{}})
	if v != value.TRUE {
		t.Error("expected true")
	}
}

func TestPrint(t *testing.T) {
	var buf bytes.Buffer
	rt := NewGoRuntime()
	rt.SetOutput(&buf)
	callBuiltin(t, rt, "_print", []value.Value{&value.String{Val: "hello"}})
	if buf.String() != "hello" {
		t.Errorf("got %q", buf.String())
	}
}

func TestRead(t *testing.T) {
	rt := NewGoRuntime()
	rt.SetInput(strings.NewReader("alpha\nbeta\r\ngamma"))

	for i, want := range []string{"alpha", "beta", "gamma", ""} {
		got := callBuiltin(t, rt, "_read", nil)
		str, ok := got.(*value.String)
		if !ok {
			t.Fatalf("read %d returned %T, want string", i, got)
		}
		if str.Val != want {
			t.Fatalf("read %d = %q, want %q", i, str.Val, want)
		}
	}
}

func TestEqual(t *testing.T) {
	rt := NewGoRuntime()
	v := callBuiltin(t, rt, "_equal", []value.Value{value.NewNumber(1, 1), value.NewNumber(1, 1)})
	if v != value.TRUE {
		t.Error("expected true")
	}
}

// TestNoAuthorityGetsNothing verifies no authority cannot access any builtins from runtime.
func TestNoAuthorityGetsNothing(t *testing.T) {
	rt := NewGoRuntime()

	// No-authority code cannot see anything - all access comes from prelude.nez bindings in Env
	if rt.HasBuiltin("first", value.NoAuthority()) {
		t.Error("user should NOT see first from runtime")
	}
	if rt.HasBuiltin("_first", value.NoAuthority()) {
		t.Error("user should NOT see _first")
	}

	_, ok := rt.GetBuiltin("first", value.NoAuthority())
	if ok {
		t.Error("user should NOT get first from runtime")
	}
	_, ok = rt.GetBuiltin("_first", value.NoAuthority())
	if ok {
		t.Error("user should NOT get _first")
	}
}

// TestExplicitAuthoritySeesOnlyGrantedPrefixed verifies prelude scope sees only _prefixed HAL primitives.
func TestExplicitAuthoritySeesOnlyGrantedPrefixed(t *testing.T) {
	rt := NewGoRuntime()

	// Authority does not expose non-prefixed (those come from prelude.nez, not registry)
	if rt.HasBuiltin("first", value.NewAuthority("_first")) {
		t.Error("authority should NOT see non-prefixed 'first' from registry")
	}

	// Explicit authority can see _prefixed
	if !rt.HasBuiltin("_first", value.NewAuthority("_first")) {
		t.Error("authority should see _first")
	}

	// GetBuiltin matches HasBuiltin
	_, ok := rt.GetBuiltin("first", value.NewAuthority("_first"))
	if ok {
		t.Error("authority should NOT get non-prefixed 'first' from registry")
	}

	b, ok := rt.GetBuiltin("_first", value.NewAuthority("_first"))
	if !ok || b == nil {
		t.Error("authority GetBuiltin(_first) failed")
	}
}
