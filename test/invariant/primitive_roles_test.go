package invariant

import (
	"strings"
	"testing"

	"aiki/engine/runtime/hal"
	"aiki/engine/runtime/primitives"
)

func TestPrimitiveRolesAreCoherent(t *testing.T) {
	if err := primitives.ValidateCanonicalDefinitions(); err != nil {
		t.Fatal(err)
	}
}

func TestPrimitiveRoleInvariantRejectsMissingHALPrimitive(t *testing.T) {
	defs := primitives.Definitions()
	delete(defs, "_print")
	err := primitives.ValidateDefinitions(defs, hal.OperationDefinitions())
	if err == nil || !strings.Contains(err.Error(), "HAL primitive _print has no architectural role") {
		t.Fatalf("expected missing HAL role failure, got %v", err)
	}
}

func TestPrimitiveRoleInvariantRejectsWrongHALRole(t *testing.T) {
	defs := primitives.Definitions()
	defs["_print"] = primitives.RoleRuntime
	err := primitives.ValidateDefinitions(defs, hal.OperationDefinitions())
	if err == nil || !strings.Contains(err.Error(), "HAL primitive _print has role runtime, want host") {
		t.Fatalf("expected wrong HAL role failure, got %v", err)
	}
}

func TestPrimitiveRoleInvariantRejectsHostWithoutHALOperation(t *testing.T) {
	definitions := primitives.Definitions()
	definitions["_host_without_hal"] = primitives.RoleHost
	err := primitives.ValidateDefinitions(definitions, hal.OperationDefinitions())
	if err == nil || !strings.Contains(err.Error(), "host primitive _host_without_hal has no canonical HAL operation") {
		t.Fatalf("expected host-without-HAL failure, got %v", err)
	}
}
