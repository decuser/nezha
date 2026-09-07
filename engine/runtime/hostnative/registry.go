// Package hostnative is the generic registration and execution boundary for
// host-native implementations of Nezha semantic contracts. Implementation and
// operation names are resolved only during composition; steady-state calls use
// the returned direct function value.
package hostnative

import (
	"fmt"
	"sync"

	"nezha/engine/semantics/value"
)

type Operation func(args []value.Value) value.Value

type Implementation struct {
	Operations map[string]Operation
}

var registry = struct {
	sync.RWMutex
	implementations map[string]Implementation
}{implementations: map[string]Implementation{}}

func Register(name string, implementation Implementation) {
	if name == "" {
		panic("host-native implementation has empty name")
	}
	if len(implementation.Operations) == 0 {
		panic("host-native implementation has no operations: " + name)
	}
	copied := make(map[string]Operation, len(implementation.Operations))
	for operation, fn := range implementation.Operations {
		if operation == "" || fn == nil {
			panic("invalid host-native operation for implementation: " + name)
		}
		copied[operation] = fn
	}
	registry.Lock()
	defer registry.Unlock()
	if _, exists := registry.implementations[name]; exists {
		panic("duplicate host-native implementation: " + name)
	}
	registry.implementations[name] = Implementation{Operations: copied}
}

func Resolve(name, operation string) (Operation, error) {
	registry.RLock()
	implementation, ok := registry.implementations[name]
	registry.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown host-native implementation %q", name)
	}
	fn, ok := implementation.Operations[operation]
	if !ok {
		return nil, fmt.Errorf("host-native implementation %q has no operation %q", name, operation)
	}
	return fn, nil
}
