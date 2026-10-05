package hal

// HostOperation describes one canonical crossing from Nezha into the host.
// The current specimen has a deliberately small set of host crossings for
// text I/O. This metadata records those boundaries without preserving the
// larger Aiki-era capability/profile taxonomy.
type HostOperation struct {
	Identity            string
	Primitive           string
	Effect              string
	Blocking            bool
	SubstrateProvenance string
	NezhaBindings       []NezhaBinding
}

// NezhaBinding records one programmer-facing Nezha name implemented through a
// host operation and the source file that owns that meaning.
type NezhaBinding struct {
	Name   string
	Source string
}

// HostOperationProvider is an optional runtime capability for inspecting the
// canonical host contracts currently bound by a substrate. It is separate from
// RuntimeContract so evaluator behavior does not depend on metadata inspection.
type HostOperationProvider interface {
	HostOperations() []HostOperation
}
