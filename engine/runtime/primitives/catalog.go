// Package primitives owns the architectural names of Nezha runtime primitives.
// Concrete substrates bind leaf implementations to these stable names. Host
// boundary metadata lives in the HAL package rather than in a role registry.
package primitives

import (
	"sort"
	"strings"

	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

var runtimePrimitives = map[string]bool{
	"_first":            true,
	"_rest":             true,
	"_length":           true,
	"_prepend":          true,
	"_append":           true,
	"_empty":            true,
	"_type":             true,
	"_equal":            true,
	"_shape":            true,
	"_make_shaped_list": true,
	"_to_str":           true,
}

// DisplayName returns the user-facing stem used in primitive diagnostics.
func DisplayName(name string) string {
	return strings.TrimPrefix(name, "_")
}

// IsRuntimePrimitive reports whether name is a non-HAL leaf runtime primitive.
func IsRuntimePrimitive(name string) bool {
	return runtimePrimitives[name]
}

// Definitions returns a copy of every runtime primitive name, including
// canonical HAL host operations.
func Definitions() map[string]bool {
	out := make(map[string]bool, len(runtimePrimitives)+len(hal.OperationDefinitions()))
	for name := range runtimePrimitives {
		out[name] = true
	}
	for name := range hal.OperationDefinitions() {
		out[name] = true
	}
	return out
}

// NamesForScope returns the architectural runtime names visible to tooling for
// a lexical scope. User code sees only ordinary prelude bindings; trusted
// prelude scopes also see the primitive namespace.
func NamesForScope(scope value.Scope) []string {
	set := map[string]bool{}
	if scope != value.ScopeUser {
		for name := range Definitions() {
			set[name] = true
		}
	}
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
