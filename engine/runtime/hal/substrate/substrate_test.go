package substrate

import (
	"bytes"
	"testing"

	"nezha/engine/semantics/value"
)

func TestFirst(t *testing.T) {
	rt := NewGoRuntime()
	list := &value.List{Elements: []value.Value{value.NewNumber(1, 1), value.NewNumber(2, 1)}}
	v, _ := rt.Execute("_first", []value.Value{list})
	if v.Inspect() != "1" {
		t.Errorf("got %s", v.Inspect())
	}
}

func TestRest(t *testing.T) {
	rt := NewGoRuntime()
	list := &value.List{Elements: []value.Value{value.NewNumber(1, 1), value.NewNumber(2, 1)}}
	v, _ := rt.Execute("_rest", []value.Value{list})
	if v.Inspect() != "[2]" {
		t.Errorf("got %s", v.Inspect())
	}
}

func TestLength(t *testing.T) {
	rt := NewGoRuntime()
	list := &value.List{Elements: []value.Value{value.NewNumber(1, 1), value.NewNumber(2, 1)}}
	v, _ := rt.Execute("_length", []value.Value{list})
	if v.Inspect() != "2" {
		t.Errorf("got %s", v.Inspect())
	}
}

func TestEmpty(t *testing.T) {
	rt := NewGoRuntime()
	v, _ := rt.Execute("_empty", []value.Value{&value.List{}})
	if v != value.TRUE {
		t.Error("expected true")
	}
}

func TestPrint(t *testing.T) {
	var buf bytes.Buffer
	rt := NewGoRuntime()
	rt.SetIO(nil, &buf)
	rt.Execute("_print", []value.Value{&value.String{Val: "hello"}})
	if buf.String() != "hello" {
		t.Errorf("got %q", buf.String())
	}
}

func TestEqual(t *testing.T) {
	rt := NewGoRuntime()
	v, _ := rt.Execute("_equal", []value.Value{value.NewNumber(1, 1), value.NewNumber(1, 1)})
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
