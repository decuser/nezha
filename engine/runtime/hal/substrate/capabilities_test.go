package substrate

import (
	"testing"

	"nezha/engine/runtime/hal"
)

func TestGoProvenanceCoversHALOperationDefinitions(t *testing.T) {
	definitions := hal.OperationDefinitions()
	for primitive := range definitions {
		if goHostProvenance[primitive] == "" {
			t.Errorf("HAL operation %s has no Go substrate provenance", primitive)
		}
	}
	for primitive := range goHostProvenance {
		if _, ok := definitions[primitive]; !ok {
			t.Errorf("Go substrate provenance names unknown HAL operation %s", primitive)
		}
	}
}

func TestCapabilityAvailabilityReflectsActualBindings(t *testing.T) {
	rt := NewGoRuntime()
	if !rt.HasCapability("basic-io") {
		t.Fatal("default Go runtime should provide :basic-io")
	}
	if rt.HasCapability("does-not-exist") {
		t.Fatal("unknown capability must not report available")
	}

	rt.mu.Lock()
	delete(rt.hostBindings, "_print")
	rt.mu.Unlock()
	if rt.HasCapability("basic-io") {
		t.Fatal(":basic-io must become unavailable when a required HAL binding is absent")
	}
}

func TestProfileGateRejectsMissingRequiredCapability(t *testing.T) {
	rt := NewGoRuntime()
	rt.mu.Lock()
	delete(rt.hostBindings, "_print")
	rt.mu.Unlock()
	if err := rt.ValidateProfile(hal.DefaultRuntimeProfile); err == nil {
		t.Fatal("desktop profile should reject missing :basic-io")
	}
	if err := rt.ValidateProfile("minimal"); err != nil {
		t.Fatalf("minimal profile should not require host capabilities: %v", err)
	}
}
