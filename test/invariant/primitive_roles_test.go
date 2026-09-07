package invariant

import (
	"strings"
	"testing"

	"nezha/engine/runtime/hal"
	"nezha/engine/runtime/primitives"
)

func TestPrimitiveCatalogIsCoherent(t *testing.T) {
	if err := primitives.ValidateCanonicalDefinitions(); err != nil {
		t.Fatal(err)
	}
}

func TestPrimitiveCatalogInvariantRejectsMissingHALPrimitive(t *testing.T) {
	defs := primitives.Definitions()
	delete(defs, "_print")
	err := primitives.ValidateDefinitions(defs, hal.OperationDefinitions())
	if err == nil || !strings.Contains(err.Error(), "HAL primitive _print has no architectural definition") {
		t.Fatalf("expected missing HAL primitive failure, got %v", err)
	}
}

func TestPrimitiveCatalogInvariantRejectsUnknownPrimitive(t *testing.T) {
	definitions := primitives.Definitions()
	definitions["_host_without_hal"] = true
	err := primitives.ValidateDefinitions(definitions, hal.OperationDefinitions())
	if err == nil || !strings.Contains(err.Error(), "runtime primitive _host_without_hal is not runtime or HAL-defined") {
		t.Fatalf("expected unknown primitive failure, got %v", err)
	}
}
