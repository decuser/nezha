package substrate

import (
	"fmt"
	"sort"

	"nezha/engine/runtime/hal"
)

// goHostProvenance records how this substrate realizes each architectural HAL
// operation. Architectural metadata lives in engine/runtime/hal; this file is
// intentionally Go-substrate-specific.
var goHostProvenance = map[string]string{
	"_print": "go:fmt.Fprint/io.Writer",
}

func goHostOperation(primitive string) hal.HostOperation {
	op, ok := hal.OperationDefinition(primitive)
	if !ok {
		panic(fmt.Sprintf("missing HAL operation definition: %s", primitive))
	}
	provenance, ok := goHostProvenance[primitive]
	if !ok || provenance == "" {
		panic(fmt.Sprintf("missing Go substrate provenance: %s", primitive))
	}
	op.SubstrateProvenance = provenance
	return op
}

// HostOperations returns the canonical host-operation descriptors bound by this
// runtime, sorted by HAL identity. The returned slice is independent of runtime
// execution state and is intended for tooling, validation, and observation.
func (g *GoRuntime) HostOperations() []hal.HostOperation {
	out := make([]hal.HostOperation, 0, len(g.hostBindings))
	for _, op := range g.hostBindings {
		copyOp := op
		copyOp.NezhaBindings = append([]hal.NezhaBinding(nil), op.NezhaBindings...)
		out = append(out, copyOp)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	return out
}
