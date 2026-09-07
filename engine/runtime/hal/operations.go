package hal

var operationDefinitions = map[string]HostOperation{
	"_print": hostOperation("HAL.io.print", "_print", "runtime.io", "may block", "fault", bindings(
		binding("print", "engine/runtime/prelude/prelude.ai"),
		binding("println", "engine/runtime/prelude/prelude.ai"),
	)),
	"_read": hostOperation("HAL.io.read", "_read", "runtime.io", "may block", "fault or shaped :io/:end result", bindings(
		binding("read", "engine/runtime/prelude/prelude.ai"),
		binding("input", "engine/runtime/prelude/prelude.ai"),
	)),
	"_file_open":        resourceAcquisition("HAL.file.open", "_file_open", "file.open", binding("file.open", "lib/file/file.ai")),
	"_file_read_text":   resourceOperation("HAL.file.read_text", "_file_read_text", "file.read_text", binding("file.read_text", "lib/file/file.ai")),
	"_file_read_bytes":  resourceOperation("HAL.file.read_bytes", "_file_read_bytes", "file.read_bytes", binding("file.read_bytes", "lib/file/file.ai")),
	"_file_read_line":   resourceOperation("HAL.file.read_line", "_file_read_line", "file.read_line", binding("file.read_line", "lib/file/file.ai")),
	"_file_write_text":  resourceOperation("HAL.file.write_text", "_file_write_text", "file.write_text", binding("file.write_text", "lib/file/file.ai")),
	"_file_write_bytes": resourceOperation("HAL.file.write_bytes", "_file_write_bytes", "file.write_bytes", binding("file.write_bytes", "lib/file/file.ai")),
	"_file_close":       resourceOperation("HAL.file.close", "_file_close", "file.close", binding("file.close", "lib/file/file.ai")),
	"_file_exists":      pathOperation("HAL.file.exists", "_file_exists", "file.exists", binding("file.exists", "lib/file/file.ai")),
	"_file_delete":      pathOperation("HAL.file.delete", "_file_delete", "file.delete", binding("file.delete", "lib/file/file.ai")),
	"_file_list":        pathOperation("HAL.file.list", "_file_list", "file.list", binding("file.list", "lib/file/file.ai")),
	"_file_entries":     pathOperation("HAL.file.entries", "_file_entries", "file.entries", binding("file.entries", "lib/file/file.ai")),
	"_file_read_at":     resourceOperation("HAL.file.read_at", "_file_read_at", "file.read_at", binding("file.read_at", "lib/file/file.ai")),
	"_file_write_at":    resourceOperation("HAL.file.write_at", "_file_write_at", "file.write_at", binding("file.write_at", "lib/file/file.ai")),
	"_file_stat":        pathOperation("HAL.file.stat", "_file_stat", "file.stat", binding("file.stat", "lib/file/file.ai")),
	"_file_rename":      pathOperation("HAL.file.rename", "_file_rename", "file.rename", binding("file.rename", "lib/file/file.ai")),
	"_file_mkdir":       pathOperation("HAL.file.mkdir", "_file_mkdir", "file.mkdir", binding("file.mkdir", "lib/file/file.ai")),
	"_file_mkdir_all":   pathOperation("HAL.file.mkdir_all", "_file_mkdir_all", "file.mkdir_all", binding("file.mkdir_all", "lib/file/file.ai")),
	"_file_remove_all":  pathOperation("HAL.file.remove_all", "_file_remove_all", "file.remove_all", binding("file.remove_all", "lib/file/file.ai")),
	"_file_temp":        pathOperation("HAL.file.temp", "_file_temp", "file.temp", binding("file.temp", "lib/file/file.ai")),
	"_file_temp_dir":    pathOperation("HAL.file.temp_dir", "_file_temp_dir", "file.temp_dir", binding("file.temp_dir", "lib/file/file.ai")),
	"_file_copy":        pathOperation("HAL.file.copy", "_file_copy", "file.copy", binding("file.copy", "lib/file/file.ai")),
	"_file_size":        pathOperation("HAL.file.size", "_file_size", "file.size", binding("file.size", "lib/file/file.ai")),
	"_file_walk":        pathOperation("HAL.file.walk", "_file_walk", "file.walk", binding("file.walk", "lib/file/file.ai")),
	"_file_symlink":     optionalPathOperation("HAL.file.symlink", "_file_symlink", "file.symlink", binding("file.symlink", "lib/file/file.ai")),
	"_file_read_link":   optionalPathOperation("HAL.file.read_link", "_file_read_link", "file.read_link", binding("file.read_link", "lib/file/file.ai")),
	"_file_lock": {
		Identity:            "HAL.file.lock",
		Primitive:           "_file_lock",
		Authority:           "HAL.file.lock",
		Context:             []string{"runtime.filesystem", "runtime.resources"},
		Effect:              "resource acquisition",
		Blocking:            "may block",
		Lifetime:            "returns resource",
		Optionality:         "constitutive",
		ErrorContract:       "fault or shaped :lock result",
		SemanticObservation: "file.lock",
		AikiBindings:        bindings(binding("file.lock", "lib/file/file.ai")),
	},
	"_file_try_lock": {
		Identity:            "HAL.file.try_lock",
		Primitive:           "_file_try_lock",
		Authority:           "HAL.file.try_lock",
		Context:             []string{"runtime.filesystem", "runtime.resources"},
		Effect:              "resource acquisition",
		Blocking:            "nonblocking",
		Lifetime:            "returns resource",
		Optionality:         "constitutive",
		ErrorContract:       "fault, false, or shaped :lock result",
		SemanticObservation: "file.try_lock",
		AikiBindings:        bindings(binding("file.try_lock", "lib/file/file.ai")),
	},
	"_file_unlock": {
		Identity:            "HAL.file.unlock",
		Primitive:           "_file_unlock",
		Authority:           "HAL.file.unlock",
		Context:             []string{"runtime.resources"},
		Effect:              "state mutation",
		Blocking:            "nonblocking",
		Lifetime:            "operates on resource",
		Optionality:         "constitutive",
		ErrorContract:       "fault or shaped :lock result",
		SemanticObservation: "file.unlock",
		AikiBindings:        bindings(binding("file.unlock", "lib/file/file.ai")),
	},
	"_file_permissions": optionalPathOperation("HAL.file.permissions", "_file_permissions", "file.permissions", binding("file.permissions", "lib/file/file.ai")),
	"_file_chmod":       optionalPathOperation("HAL.file.chmod", "_file_chmod", "file.chmod", binding("file.chmod", "lib/file/file.ai")),
}

func binding(name, source string) AikiBinding {
	return AikiBinding{Name: name, Source: source}
}

func bindings(bs ...AikiBinding) []AikiBinding { return bs }

func hostOperation(identity, primitive, context, blocking, errors string, aiki []AikiBinding) HostOperation {
	observation := ""
	if len(aiki) > 0 {
		observation = aiki[0].Name
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
		AikiBindings:        aiki,
	}
}

func fileOperation(identity, primitive, observation string, b AikiBinding) HostOperation {
	op := hostOperation(identity, primitive, "runtime.filesystem", "may block", "fault or shaped :io result", bindings(b))
	op.SemanticObservation = observation
	return op
}

func pathOperation(identity, primitive, observation string, b AikiBinding) HostOperation {
	return fileOperation(identity, primitive, observation, b)
}

func optionalPathOperation(identity, primitive, observation string, b AikiBinding) HostOperation {
	op := fileOperation(identity, primitive, observation, b)
	op.Optionality = "optional"
	return op
}

func resourceAcquisition(identity, primitive, observation string, b AikiBinding) HostOperation {
	op := fileOperation(identity, primitive, observation, b)
	op.Effect = "resource acquisition"
	op.Lifetime = "returns resource"
	return op
}

func resourceOperation(identity, primitive, observation string, b AikiBinding) HostOperation {
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
	op.AikiBindings = append([]AikiBinding(nil), op.AikiBindings...)
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
