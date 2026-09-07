package invariant

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"nezha/engine/runtime/hal/substrate"
	"nezha/engine/runtime/primitives"
)

func TestRuntimePrimitiveRegistrationsMatchArchitecture(t *testing.T) {
	rt := substrate.NewGoRuntime()
	if err := validatePrimitiveRegistrations(primitives.Definitions(), rt.PrimitiveRegistrations()); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeRegistrationInvariantRejectsMissingPrimitive(t *testing.T) {
	rt := substrate.NewGoRuntime()
	actual := rt.PrimitiveRegistrations()
	delete(actual, "_print")
	err := validatePrimitiveRegistrations(primitives.Definitions(), actual)
	if err == nil || !strings.Contains(err.Error(), "missing runtime primitive _print") {
		t.Fatalf("expected missing primitive failure, got %v", err)
	}
}

func TestRuntimeRegistrationInvariantRejectsUnknownPrimitive(t *testing.T) {
	rt := substrate.NewGoRuntime()
	actual := rt.PrimitiveRegistrations()
	actual["_ghost"] = substrate.Arity{Min: 0, Max: 0}
	err := validatePrimitiveRegistrations(primitives.Definitions(), actual)
	if err == nil || !strings.Contains(err.Error(), "runtime registers primitive with no architectural definition _ghost") {
		t.Fatalf("expected unknown primitive failure, got %v", err)
	}
}

func validatePrimitiveRegistrations(want map[string]bool, actual map[string]substrate.Arity) error {
	var problems []string
	for name := range want {
		if _, ok := actual[name]; !ok {
			problems = append(problems, "missing runtime primitive "+name)
		}
	}
	for name := range actual {
		if !want[name] {
			problems = append(problems, "runtime registers primitive with no architectural definition "+name)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("runtime primitive-registration invariant failure:\n  %s", strings.Join(problems, "\n  "))
}
