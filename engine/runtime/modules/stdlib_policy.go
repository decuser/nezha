package modules

// StdlibSemanticRole identifies who owns the meaning of a shipped Aiki module.
// It is deliberately separate from implementation realization and from HAL
// authority. A provider-backed implementation is not automatically interop,
// and a runtime/host capability is not automatically an acceleration.
type StdlibSemanticRole string

const (
	RolePortable          StdlibSemanticRole = "portable"
	RoleRuntimeCapability StdlibSemanticRole = "runtime-capability"
	RoleHostCapability    StdlibSemanticRole = "host-capability"
	RoleInterop           StdlibSemanticRole = "interop"
	RoleInternal          StdlibSemanticRole = "internal"
)

// StdlibRealization describes how a shipped module realizes its public surface.
// Native means the library semantics are implemented in Aiki. Intrinsic means
// the module exposes irreducible runtime/host machinery. Mixed is intentional
// composition of more than one realization kind and must never be used to make
// a /native package appear pure.
type StdlibRealization string

const (
	RealizationNative    StdlibRealization = "native"
	RealizationFFI       StdlibRealization = "ffi"
	RealizationIntrinsic StdlibRealization = "intrinsic"
	RealizationMixed     StdlibRealization = "mixed"
)

// StdlibModulePolicy is the module-layer authority for current shipped-module
// classification. SemanticAuthority names the module whose Aiki contract an FFI
// acceleration is intended to preserve. AccelerationSurface optionally names an
// exact exported subset of that authority; empty means the complete authority
// surface. SemanticAuthority is empty for capabilities, interop, internal modules,
// and portable modules without an accelerated sibling.
type StdlibModulePolicy struct {
	Module              string
	Role                StdlibSemanticRole
	Realization         StdlibRealization
	SemanticAuthority   string
	AccelerationSurface []string
}

// stdlibModulePolicies is the module-layer semantic authority for the shipped
// tree. It records intended role and realization; executable invariants verify
// that names, native paths, provider use, and acceleration surfaces tell the
// same truth as this declaration.
var stdlibModulePolicies = []StdlibModulePolicy{
	// Portable semantics.
	{Module: "hash/native", Role: RolePortable, Realization: RealizationNative},
	{Module: "hash/ffi", Role: RolePortable, Realization: RealizationFFI, SemanticAuthority: "hash/native"},
	{Module: "math/native", Role: RolePortable, Realization: RealizationNative},
	{Module: "math/ffi", Role: RolePortable, Realization: RealizationFFI, SemanticAuthority: "math/native", AccelerationSurface: []string{"floor", "ceil", "modulo"}},
	{Module: "math/approx", Role: RoleInterop, Realization: RealizationFFI},
	{Module: "number", Role: RolePortable, Realization: RealizationNative},
	{Module: "string/native", Role: RolePortable, Realization: RealizationNative},
	{Module: "string/ffi", Role: RolePortable, Realization: RealizationFFI, SemanticAuthority: "string/native"},
	{Module: "string/case_table", Role: RoleInternal, Realization: RealizationNative},

	// Runtime-owned Aiki facilities.
	{Module: "io", Role: RoleRuntimeCapability, Realization: RealizationIntrinsic},
	{Module: "store", Role: RoleRuntimeCapability, Realization: RealizationIntrinsic},
	{Module: "store/ffi", Role: RoleRuntimeCapability, Realization: RealizationFFI},

	// Host-mediated facilities. path intentionally composes portable Aiki code
	// with host/runtime operations and is therefore mixed.
	{Module: "file", Role: RoleHostCapability, Realization: RealizationIntrinsic},
	{Module: "path", Role: RoleHostCapability, Realization: RealizationMixed},
	{Module: "random", Role: RoleHostCapability, Realization: RealizationIntrinsic},
	{Module: "signal", Role: RoleHostCapability, Realization: RealizationIntrinsic},
	{Module: "system", Role: RoleHostCapability, Realization: RealizationIntrinsic},
	{Module: "time", Role: RoleHostCapability, Realization: RealizationIntrinsic},

	// Tooling/runtime implementation support rather than general stdlib semantics.
	{Module: "profile", Role: RoleInternal, Realization: RealizationMixed},
	{Module: "test", Role: RoleInternal, Realization: RealizationIntrinsic},
}

// StdlibModulePolicies returns a copy of all shipped module policy declarations.
func StdlibModulePolicies() []StdlibModulePolicy {
	out := make([]StdlibModulePolicy, len(stdlibModulePolicies))
	copy(out, stdlibModulePolicies)
	return out
}

// StdlibModulePolicyFor reports the policy for a source-declared module.
func StdlibModulePolicyFor(module string) (StdlibModulePolicy, bool) {
	for _, policy := range stdlibModulePolicies {
		if policy.Module == module {
			return policy, true
		}
	}
	return StdlibModulePolicy{}, false
}
