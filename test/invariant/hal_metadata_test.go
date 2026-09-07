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

func TestHALMetadataInvariantRejectsUnknownContext(t *testing.T) {
	ops := hal.OperationDefinitions()
	op := ops["_print"]
	op.Context = []string{"runtime.typo"}
	ops["_print"] = op

	err := hal.ValidateMetadata(ops)
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
			err := hal.ValidateMetadata(ops)
			if err == nil || !strings.Contains(err.Error(), mutation.want) {
				t.Fatalf("expected descriptor vocabulary failure %q, got %v", mutation.want, err)
			}
		})
	}
}
