package hal

var operationDefinitions = map[string]HostOperation{
	"_print": {
		Identity:  "HAL.io.print",
		Primitive: "_print",
		Effect:    "write output",
		Blocking:  true,
		NezhaBindings: []NezhaBinding{
			{Name: "print", Source: "engine/runtime/prelude/prelude.nez"},
			{Name: "println", Source: "engine/runtime/prelude/prelude.nez"},
		},
	},
}

// OperationDefinition returns the architectural contract for a substrate primitive.
// Substrate provenance is intentionally supplied by the substrate binding layer.
func OperationDefinition(primitive string) (HostOperation, bool) {
	op, ok := operationDefinitions[primitive]
	if !ok {
		return HostOperation{}, false
	}
	op.NezhaBindings = append([]NezhaBinding(nil), op.NezhaBindings...)
	return op, true
}

// OperationDefinitions returns an independent copy of the canonical HAL operation set.
func OperationDefinitions() map[string]HostOperation {
	out := make(map[string]HostOperation, len(operationDefinitions))
	for primitive := range operationDefinitions {
		op, _ := OperationDefinition(primitive)
		out[primitive] = op
	}
	return out
}
