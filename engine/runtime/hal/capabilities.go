package hal

// Capability names a host-dependent Nezha facility as a set of HAL operation
// identities. Capabilities are metadata; they never participate in dispatch.
type Capability struct {
	Name       string
	Operations []string
}

var capabilityRegistry = map[string]Capability{
	"basic-io": {
		Name:       "basic-io",
		Operations: []string{"HAL.io.print", "HAL.io.read"},
	},
}

// CapabilityDefinition returns an independent copy of one capability.
func CapabilityDefinition(name string) (Capability, bool) {
	capability, ok := capabilityRegistry[name]
	if !ok {
		return Capability{}, false
	}
	capability.Operations = append([]string(nil), capability.Operations...)
	return capability, true
}

// Capabilities returns an independent copy of the capability registry.
func Capabilities() map[string]Capability {
	out := make(map[string]Capability, len(capabilityRegistry))
	for name := range capabilityRegistry {
		capability, _ := CapabilityDefinition(name)
		out[name] = capability
	}
	return out
}

// CapabilityAvailable reports whether every HAL operation required by a
// capability is present in the supplied bound-identity set.
func CapabilityAvailable(name string, boundIdentities map[string]bool) bool {
	capability, ok := CapabilityDefinition(name)
	if !ok {
		return false
	}
	for _, identity := range capability.Operations {
		if !boundIdentities[identity] {
			return false
		}
	}
	return true
}
