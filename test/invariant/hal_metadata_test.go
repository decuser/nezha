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

	err := hal.ValidateMetadata(ops)
	if err == nil || !strings.Contains(err.Error(), "HAL identity "+first.Identity+" is shared") {
		t.Fatalf("expected duplicate HAL identity failure, got %v", err)
	}
}

func TestHALMetadataInvariantRejectsEmptyEffect(t *testing.T) {
	ops := hal.OperationDefinitions()
	op := ops["_print"]
	op.Effect = ""
	ops["_print"] = op

	err := hal.ValidateMetadata(ops)
	if err == nil || !strings.Contains(err.Error(), "operation _print has empty effect") {
		t.Fatalf("expected empty effect failure, got %v", err)
	}
}

func TestHALMetadataInvariantRejectsMissingBinding(t *testing.T) {
	ops := hal.OperationDefinitions()
	op := ops["_print"]
	op.NezhaBindings = nil
	ops["_print"] = op

	err := hal.ValidateMetadata(ops)
	if err == nil || !strings.Contains(err.Error(), "operation _print has no Nezha binding") {
		t.Fatalf("expected missing binding failure, got %v", err)
	}
}
