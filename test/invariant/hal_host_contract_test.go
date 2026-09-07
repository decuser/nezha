package invariant

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"nezha/engine/runtime/hal"
	"nezha/engine/runtime/hal/substrate"
	"nezha/engine/semantics/value"
)

// TestHALHostOperationThreeNameCoverage makes the M1 three-name relationship
// executable. Every canonical host contract must have one compatibility
// primitive, one substrate provenance name, and at least one Nezha-facing owner.
// The Nezha source named by the descriptor must actually depend on that primitive.
func TestHALHostOperationThreeNameCoverage(t *testing.T) {
	root := distributionRoot(t)
	rt := substrate.NewGoRuntime()
	ops := rt.HostOperations()
	if err := validateHostOperationCoverage(ops, hal.OperationDefinitions()); err != nil {
		t.Fatal(err)
	}

	visible := map[string]bool{}
	for _, name := range rt.BuiltinNames(value.ScopePrelude) {
		visible[name] = true
	}

	identities := map[string]string{}
	primitives := map[string]string{}
	for _, op := range ops {
		if !strings.HasPrefix(op.Identity, "HAL.") {
			t.Errorf("%s: canonical identity must begin HAL.", op.Primitive)
		}
		if op.SubstrateProvenance == "" || !strings.HasPrefix(op.SubstrateProvenance, "go:") {
			t.Errorf("%s: missing Go substrate provenance", op.Identity)
		}
		if op.Effect == "" {
			t.Errorf("%s: missing host effect description", op.Identity)
		}
		if prior, ok := identities[op.Identity]; ok {
			t.Errorf("duplicate HAL identity %s on %s and %s", op.Identity, prior, op.Primitive)
		}
		identities[op.Identity] = op.Primitive
		if prior, ok := primitives[op.Primitive]; ok {
			t.Errorf("primitive %s bound by both %s and %s", op.Primitive, prior, op.Identity)
		}
		primitives[op.Primitive] = op.Identity
		if !visible[op.Primitive] {
			t.Errorf("%s: compatibility primitive %s is not registered", op.Identity, op.Primitive)
		}
		if len(op.NezhaBindings) == 0 {
			t.Errorf("%s: no Nezha programmer-facing binding recorded", op.Identity)
		}

		for _, b := range op.NezhaBindings {
			assertNezhaBindingUsesPrimitive(t, root, b.Name, b.Source, op.Primitive)
			authority := rt.PreludeAuthority()
			if !authority.Allows(op.Identity) {
				t.Errorf("%s: prelude lacks canonical authority %s", op.Identity, op.Identity)
			}
			if authority.Allows(op.Primitive) {
				t.Errorf("%s: prelude grants raw host binding %s instead of canonical identity", op.Identity, op.Primitive)
			}
		}
	}
}

func assertNezhaBindingUsesPrimitive(t *testing.T, root, qualifiedName, source, primitive string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(source))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("%s: read Nezha binding source %s: %v", qualifiedName, source, err)
		return
	}

	name := qualifiedName
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	pattern := `(?ms)^let\s+` + regexp.QuoteMeta(name) + `\s*=\s*\([^)]*\)\s*\{([^}]*)\}`
	re := regexp.MustCompile(pattern)
	match := re.FindSubmatch(data)
	if match == nil {
		t.Errorf("%s: cannot find wrapper %s in %s", qualifiedName, name, source)
		return
	}
	if !strings.Contains(string(match[1]), primitive) {
		t.Errorf("%s: wrapper in %s does not depend on %s", qualifiedName, source, primitive)
	}
}

func TestHALHostCoverageInvariantRejectsMissingBinding(t *testing.T) {
	rt := substrate.NewGoRuntime()
	ops := rt.HostOperations()
	if len(ops) == 0 {
		t.Fatal("runtime has no host operations")
	}
	missing := ops[0].Identity
	ops = ops[1:]
	err := validateHostOperationCoverage(ops, hal.OperationDefinitions())
	if err == nil || !strings.Contains(err.Error(), "missing runtime host binding "+missing) {
		t.Fatalf("expected missing host binding failure for %s, got %v", missing, err)
	}
}

func validateHostOperationCoverage(bound []hal.HostOperation, definitions map[string]hal.HostOperation) error {
	want := map[string]bool{}
	for _, op := range definitions {
		want[op.Identity] = true
	}
	seen := map[string]bool{}
	var problems []string
	for _, op := range bound {
		if !want[op.Identity] {
			problems = append(problems, "runtime binds unknown HAL identity "+op.Identity)
		}
		if seen[op.Identity] {
			problems = append(problems, "runtime binds duplicate HAL identity "+op.Identity)
		}
		seen[op.Identity] = true
	}
	for identity := range want {
		if !seen[identity] {
			problems = append(problems, "missing runtime host binding "+identity)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("HAL host-binding invariant failure:\n  %s", strings.Join(problems, "\n  "))
}
