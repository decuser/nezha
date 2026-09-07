// Package hal defines the runtime contract that the evaluator uses
// to interact with the host platform.
package hal

import "nezha/engine/semantics/value"

// RuntimeContract defines the behavioral requirements of the evaluator.
// The evaluator delegates all host-level effects through this interface.
type RuntimeContract interface {
	// Execute calls a registered primitive function by name.
	Execute(name string, args []value.Value) (value.Value, error)

	// HasBuiltin checks whether a name is executable under the supplied authority.
	HasBuiltin(name string, authority value.Authority) bool

	// GetBuiltin returns a callable for the named builtin under the supplied authority.
	GetBuiltin(name string, authority value.Authority) (value.Callable, bool)
}
