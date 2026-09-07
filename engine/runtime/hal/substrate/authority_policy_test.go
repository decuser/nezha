package substrate

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"

	"aiki/engine/runtime/hal"
	"aiki/engine/semantics/value"
)

func TestAuthorityPolicyIsExplicitAndNarrow(t *testing.T) {
	rt := NewGoRuntime()

	unlisted := rt.AuthorityForSource("/tmp/project/example.ai")
	if unlisted.Allows("HAL.io.print") || unlisted.Allows("_test_run") {
		t.Fatal("ordinary source path alone must not confer authority")
	}
}

func TestPreludeAuthorityIsDeclaredNotUniversal(t *testing.T) {
	rt := NewGoRuntime()
	auth := rt.AuthorityForSource("engine/runtime/prelude/prelude.ai")
	if !auth.Allows("HAL.io.print") || !auth.Allows("_length") {
		t.Fatal("prelude missing declared primitive authority")
	}
	if auth.Allows("_test_run") {
		t.Fatal("prelude must not receive test harness authority")
	}
}

func TestAuthorityPolicyMatchesTrustedSourceDependencies(t *testing.T) {
	rt := NewGoRuntime()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate authority policy test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(here), "../../../.."))
	ident := regexp.MustCompile(`\b_[A-Za-z][A-Za-z0-9_]*\b`)

	for source, declared := range hal.AuthorityPolicy() {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source)))
		if err != nil {
			t.Fatalf("%s: reading trusted source: %v", source, err)
		}

		actual := map[string]bool{}
		for _, name := range ident.FindAllString(string(data), -1) {
			if _, registered := rt.lookupBuiltin(name); registered {
				actual[name] = true
			}
		}
		want := map[string]bool{}
		for _, name := range declared {
			want[name] = true
		}

		for name := range actual {
			if !want[name] {
				t.Errorf("%s: raw dependency %s is not declared in authority policy", source, name)
			}
		}
		for name := range want {
			if !actual[name] {
				t.Errorf("%s: authority policy grants unused raw primitive %s", source, name)
			}
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
