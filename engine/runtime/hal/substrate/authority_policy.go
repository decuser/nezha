package substrate

import "nezha/engine/semantics/value"

// preludePrimitiveAuthority records the raw privileged names that the embedded
// prelude is allowed to resolve. Ordinary source files receive no raw primitive
// authority; they use the public bindings created by prelude.nez.
var preludePrimitiveAuthority = []string{
	"_append",
	"_apply",
	"_empty",
	"_equal",
	"_first",
	"_length",
	"_make_shaped_list",
	"_prepend",
	"_print",
	"_read",
	"_rest",
	"_shape",
	"_to_str",
	"_type",
}

// PreludeAuthority returns the exact grants used while evaluating the embedded
// prelude. Host bindings are translated to their canonical HAL identities;
// evaluator-owned and leaf runtime primitives retain their implementation names.
func (g *GoRuntime) PreludeAuthority() value.Authority {
	grants := make([]string, 0, len(preludePrimitiveAuthority))
	for _, name := range preludePrimitiveAuthority {
		grants = append(grants, g.authorityKey(name))
	}
	return value.NewAuthority(grants...)
}

// PreludeAuthorityPrimitives returns the raw prelude dependency list for
// invariant tests. It is intentionally not path-based; the program layer decides when
// it is evaluating the embedded prelude.
func PreludeAuthorityPrimitives() []string {
	return append([]string(nil), preludePrimitiveAuthority...)
}
