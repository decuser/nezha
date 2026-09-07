package invariant

import (
	"strings"
	"testing"

	"nezha/engine/runtime/hal"
)

func TestHALCanonicalMetadataIsCoherent(t *testing.T) {
	if err := hal.ValidateCanonicalMetadata(); err != nil {
		t.Fatal(err)
	}
}

func TestHALMetadataInvariantRejectsDuplicateIdentity(t *testing.T) {
	ops := hal.OperationDefinitions()
	first := ops["_print"]
	duplicate := first
	duplicate.Primitive = "_print_alias"
	ops["_print_alias"] = duplicate

	err := hal.ValidateMetadata(ops, hal.Capabilities(), hal.RuntimeProfiles())
	if err == nil || !strings.Contains(err.Error(), "HAL identity "+first.Identity+" is shared") {
		t.Fatalf("expected duplicate HAL identity failure, got %v", err)
	}
}

func TestHALMetadataInvariantRejectsUnknownCapabilityOperation(t *testing.T) {
	caps := hal.Capabilities()
	capability := caps["basic-io"]
	capability.Operations = append(capability.Operations, "HAL.io.does_not_exist")
	caps["basic-io"] = capability

	err := hal.ValidateMetadata(hal.OperationDefinitions(), caps, hal.RuntimeProfiles())
	if err == nil || !strings.Contains(err.Error(), "capability basic-io names unknown HAL operation HAL.io.does_not_exist") {
		t.Fatalf("expected unknown HAL operation failure, got %v", err)
	}
}

func TestHALMetadataInvariantRejectsUnknownProfileCapability(t *testing.T) {
	profiles := hal.RuntimeProfiles()
	profile := profiles[hal.DefaultRuntimeProfile]
	profile.Capabilities = append(profile.Capabilities, "does-not-exist")
	profiles[hal.DefaultRuntimeProfile] = profile

	err := hal.ValidateMetadata(hal.OperationDefinitions(), hal.Capabilities(), profiles)
	if err == nil || !strings.Contains(err.Error(), "profile desktop names unknown capability does-not-exist") {
		t.Fatalf("expected unknown capability failure, got %v", err)
	}
}

func TestHALCapabilityAvailabilityRejectsMissingOperation(t *testing.T) {
	bound := map[string]bool{}
	for _, op := range hal.OperationDefinitions() {
		bound[op.Identity] = true
	}
	if !hal.CapabilityAvailable("basic-io", bound) {
		t.Fatal("basic-io should be available with all canonical HAL operations bound")
	}
	delete(bound, "HAL.io.print")
	if hal.CapabilityAvailable("basic-io", bound) {
		t.Fatal("basic-io must be unavailable when HAL.io.print is absent")
	}
}

func TestHALProfileGateRejectsMissingRequiredCapability(t *testing.T) {
	has := func(name string) bool { return name != "basic-io" }
	err := hal.ValidateProfileAvailability(hal.DefaultRuntimeProfile, has)
	if err == nil || !strings.Contains(err.Error(), "requires unsupported capability: :basic-io") {
		t.Fatalf("expected required-capability profile failure, got %v", err)
	}
}

func TestHALMetadataInvariantRejectsUnknownContext(t *testing.T) {
	ops := hal.OperationDefinitions()
	op := ops["_print"]
	op.Context = []string{"runtime.typo"}
	ops["_print"] = op

	err := hal.ValidateMetadata(ops, hal.Capabilities(), hal.RuntimeProfiles())
	if err == nil || !strings.Contains(err.Error(), "operation _print has unknown context runtime.typo") {
		t.Fatalf("expected unknown context failure, got %v", err)
	}
}

func TestHALMetadataInvariantRejectsUnknownDescriptorVocabulary(t *testing.T) {
	mutations := []struct {
		name string
		edit func(*hal.HostOperation)
		want string
	}{
		{"effect", func(op *hal.HostOperation) { op.Effect = "mystery" }, `unknown effect "mystery"`},
		{"blocking", func(op *hal.HostOperation) { op.Blocking = "sometimes-ish" }, `unknown blocking class "sometimes-ish"`},
		{"lifetime", func(op *hal.HostOperation) { op.Lifetime = "forever maybe" }, `unknown lifetime "forever maybe"`},
		{"optionality", func(op *hal.HostOperation) { op.Optionality = "recommended" }, `unknown optionality "recommended"`},
		{"error", func(op *hal.HostOperation) { op.ErrorContract = "whatever Go says" }, `unknown error contract "whatever Go says"`},
	}

	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			ops := hal.OperationDefinitions()
			op := ops["_print"]
			mutation.edit(&op)
			ops["_print"] = op
			err := hal.ValidateMetadata(ops, hal.Capabilities(), hal.RuntimeProfiles())
			if err == nil || !strings.Contains(err.Error(), mutation.want) {
				t.Fatalf("expected descriptor vocabulary failure %q, got %v", mutation.want, err)
			}
		})
	}
}
