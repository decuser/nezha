// Package primitives owns the architectural classification of Nezha runtime
// primitives. Concrete substrates bind implementations to these stable names;
// the role classification is not substrate metadata.
package primitives

import (
	"sort"
	"sync"

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
	"_apply":  RoleIntrinsic,
	"_import": RoleIntrinsic,
	"_use":    RoleIntrinsic,
	"_export": RoleIntrinsic,
	"_load":   RoleIntrinsic,

	// Constitutive language/value/runtime primitives.
	"_first":            RoleRuntime,
	"_rest":             RoleRuntime,
	"_length":           RoleRuntime,
	"_prepend":          RoleRuntime,
	"_append":           RoleRuntime,
	"_empty":            RoleRuntime,
	"_range":            RoleRuntime,
	"_type":             RoleRuntime,
	"_stack_limit":      RoleRuntime,
	"_inspect":          RoleRuntime,
	"_equal":            RoleRuntime,
	"_ord":              RoleRuntime,
	"_chr":              RoleRuntime,
	"_floor":            RoleProvider,
	"_ceil":             RoleProvider,
	"_truncate":         RoleRuntime,
	"_modulo":           RoleProvider,
	"_shape":            RoleRuntime,
	"_make_shaped_list": RoleRuntime,
	"_to_str":           RoleRuntime,
	"_to_decimal":       RoleRuntime,
	"_to_number":        RoleRuntime,
	"_to_symbol":        RoleRuntime,
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

// ProfileDefinition is the stable, ordered primitive identity used by the
// profiling network. Ordinal is one-based so zero remains "not a primitive".
type ProfileDefinition struct {
	Ordinal uint32
	Name    string
	Role    Role
}

var (
	profileOnce   sync.Once
	profileDefs   []ProfileDefinition
	profileByName map[string]uint32
)

func initProfileDefinitions() {
	defs := Definitions()
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)
	profileDefs = make([]ProfileDefinition, 0, len(names))
	profileByName = make(map[string]uint32, len(names))
	for i, name := range names {
		ordinal := uint32(i + 1)
		profileDefs = append(profileDefs, ProfileDefinition{Ordinal: ordinal, Name: name, Role: defs[name]})
		profileByName[name] = ordinal
	}
}

// ProfileDefinitions returns the authoritative primitive catalog in stable
// profiling order. Callers may retain the returned value; a defensive copy is
// returned so the catalog itself remains immutable.
func ProfileDefinitions() []ProfileDefinition {
	profileOnce.Do(initProfileDefinitions)
	out := make([]ProfileDefinition, len(profileDefs))
	copy(out, profileDefs)
	return out
}

// ProfileOrdinal resolves one primitive to its stable catalog ordinal. This is
// intended for registration time; steady-state dispatch carries the ordinal on
// the callable itself and does not perform a name lookup.
func ProfileOrdinal(name string) (uint32, bool) {
	profileOnce.Do(initProfileDefinitions)
	id, ok := profileByName[name]
	return id, ok
}

// NamesForScope returns the architectural runtime names visible to tooling for
// a lexical scope. User code sees only the language intrinsics directly;
// trusted/prelude scopes also see the primitive namespace.
func NamesForScope(scope value.Scope) []string {
	set := map[string]bool{
		"import": true,
		"use":    true,
		"export": true,
	}
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
