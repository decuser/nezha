package hal

var operationDefinitions = map[string]HostOperation{
	"_print": hostOperation("HAL.io.print", "_print", "runtime.io", "may block", "fault", bindings(
		binding("print", "engine/runtime/prelude/prelude.nez"),
		binding("println", "engine/runtime/prelude/prelude.nez"),
	)),
}

func binding(name, source string) NezhaBinding {
	return NezhaBinding{Name: name, Source: source}
}

func bindings(bs ...NezhaBinding) []NezhaBinding { return bs }

func hostOperation(identity, primitive, context, blocking, errors string, nezha []NezhaBinding) HostOperation {
	observation := ""
	if len(nezha) > 0 {
		observation = nezha[0].Name
	}
	return HostOperation{
		Identity:            identity,
		Primitive:           primitive,
		Authority:           identity,
		Context:             []string{context},
		Effect:              "bounded call",
		Blocking:            blocking,
		Lifetime:            "stateless",
		Optionality:         "constitutive",
		ErrorContract:       errors,
		SemanticObservation: observation,
		NezhaBindings:       nezha,
	}
}

func fileOperation(identity, primitive, observation string, b NezhaBinding) HostOperation {
	op := hostOperation(identity, primitive, "runtime.filesystem", "may block", "fault or shaped :io result", bindings(b))
	op.SemanticObservation = observation
	return op
}

func pathOperation(identity, primitive, observation string, b NezhaBinding) HostOperation {
	return fileOperation(identity, primitive, observation, b)
}

func optionalPathOperation(identity, primitive, observation string, b NezhaBinding) HostOperation {
	op := fileOperation(identity, primitive, observation, b)
	op.Optionality = "optional"
	return op
}

func resourceAcquisition(identity, primitive, observation string, b NezhaBinding) HostOperation {
	op := fileOperation(identity, primitive, observation, b)
	op.Effect = "resource acquisition"
	op.Lifetime = "returns resource"
	return op
}

func resourceOperation(identity, primitive, observation string, b NezhaBinding) HostOperation {
	op := fileOperation(identity, primitive, observation, b)
	op.Lifetime = "operates on resource"
	return op
}

// OperationDefinition returns the architectural contract for a substrate primitive.
// Substrate provenance is intentionally supplied by the substrate binding layer.
func OperationDefinition(primitive string) (HostOperation, bool) {
	op, ok := operationDefinitions[primitive]
	if !ok {
		return HostOperation{}, false
	}
	op.Context = append([]string(nil), op.Context...)
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
