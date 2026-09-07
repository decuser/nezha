package substrate

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"nezha/engine/semantics/value"
)

func TestNoAuthorityIsEmpty(t *testing.T) {
	if value.NoAuthority().Allows("HAL.io.print") || value.NoAuthority().Allows("_length") {
		t.Fatal("ordinary source authority must be empty")
	}
}

func TestPreludeAuthorityIsDeclaredNotUniversal(t *testing.T) {
	rt := NewGoRuntime()
	auth := rt.PreludeAuthority()
	if !auth.Allows("HAL.io.print") || !auth.Allows("_length") {
		t.Fatal("prelude missing declared primitive authority")
	}
	if auth.Allows("_print") {
		t.Fatal("prelude must receive canonical host identity, not raw _print authority")
	}
	if auth.Allows("_test_run") {
		t.Fatal("prelude must not receive test harness authority")
	}
}

func TestPreludeAuthorityMatchesPreludeDependencies(t *testing.T) {
	rt := NewGoRuntime()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate authority policy test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "../../../.."))
	preludePath := filepath.Join(root, "engine/runtime/prelude/prelude.nez")
	data, err := os.ReadFile(preludePath)
	if err != nil {
		t.Fatalf("reading prelude: %v", err)
	}

	ident := regexp.MustCompile(`\b_[A-Za-z][A-Za-z0-9_]*\b`)
	actual := map[string]bool{}
	for _, name := range ident.FindAllString(string(data), -1) {
		if _, registered := rt.lookupBuiltin(name); registered || name == "_apply" {
			actual[name] = true
		}
	}
	want := map[string]bool{}
	for _, name := range PreludeAuthorityPrimitives() {
		want[name] = true
	}

	for name := range actual {
		if !want[name] {
			t.Errorf("prelude raw dependency %s is not declared in prelude authority", name)
		}
	}
	for name := range want {
		if !actual[name] {
			t.Errorf("prelude authority grants unused raw primitive %s", name)
		}
	}
}

func TestHostBindingAuthorizationUsesCanonicalHALIdentity(t *testing.T) {
	rt := NewGoRuntime()
	canonical := value.NewAuthority("HAL.io.print")
	if !rt.HasBuiltin("_print", canonical) {
		t.Fatal("HAL.io.print authority must authorize the _print substrate binding")
	}
	raw := value.NewAuthority("_print")
	if rt.HasBuiltin("_print", raw) {
		t.Fatal("raw _print name must not itself grant canonical host authority")
	}
}
