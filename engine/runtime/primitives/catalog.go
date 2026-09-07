// Package primitives owns the architectural classification of Nezha runtime
// primitives. Concrete substrates bind implementations to these stable names;
// the role classification is not substrate metadata.
package primitives

import (
	"sort"

	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
)

// Role identifies why a runtime primitive exists.
type Role string

const (
	RoleIntrinsic Role = "intrinsic"
	RoleRuntime   Role = "runtime"
	RoleProvider  Role = "provider"
	RoleHost      Role = "host"
	RoleService   Role = "service"
)

var nonHALRoles = map[string]Role{
	// Evaluator/language intrinsics.
	"_apply": RoleIntrinsic,

	// Constitutive language/value/runtime primitives.
	"_first":            RoleRuntime,
	"_rest":             RoleRuntime,
	"_length":           RoleRuntime,
	"_prepend":          RoleRuntime,
	"_append":           RoleRuntime,
	"_empty":            RoleRuntime,
	"_type":             RoleRuntime,
	"_equal":            RoleRuntime,
	"_shape":            RoleRuntime,
	"_make_shaped_list": RoleRuntime,
	"_to_str":           RoleRuntime,
}

// RoleOf returns the architectural role for a primitive. Canonical HAL host
// operations derive their host role from the HAL operation authority rather
// than being repeated in this catalog.
func RoleOf(name string) (Role, bool) {
	if _, ok := hal.OperationDefinitions()[name]; ok {
		return RoleHost, true
	}
	role, ok := nonHALRoles[name]
	return role, ok
}

// Definitions returns a copy of every runtime primitive role, including
// canonical HAL host operations.
func Definitions() map[string]Role {
	out := make(map[string]Role, len(nonHALRoles)+len(hal.OperationDefinitions()))
	for name, role := range nonHALRoles {
		out[name] = role
	}
	for name := range hal.OperationDefinitions() {
		out[name] = RoleHost
	}
	return out
}

// NamesForScope returns the architectural runtime names visible to tooling for
// a lexical scope. User code sees only the language intrinsics directly;
// trusted/prelude scopes also see the primitive namespace.
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
