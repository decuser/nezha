// Package substrate implements the Go runtime substrate for HAL.
package substrate

import (
	"fmt"
	"io"
	"os"
	"sync"

	"nezha/engine/runtime/hal"
	"nezha/engine/runtime/primitives"
	"nezha/engine/semantics/value"
)

// BuiltinFunc is a HAL-level leaf function.
type BuiltinFunc func(args []value.Value) value.Value

// Arity declares the argument shape checked before a builtin implementation runs.
type Arity struct {
	Min int
	Max int // -1 means variadic.
}

func fixedArity(n int) Arity      { return Arity{Min: n, Max: n} }
func variadicArity(min int) Arity { return Arity{Min: min, Max: -1} }

func (a Arity) valid() bool {
	return a.Min >= 0 && (a.Max == -1 || a.Max >= a.Min)
}

func (a Arity) accepts(n int) bool {
	if n < a.Min {
		return false
	}
	return a.Max == -1 || n <= a.Max
}

func (a Arity) fault(name string, got int) *value.Fault {
	display := primitives.DisplayName(name)
	if a.Max == -1 {
		return value.NewFault("%s: want at least %d arguments, got %d", display, a.Min, got)
	}
	if a.Min == a.Max {
		plural := "arguments"
		if a.Min == 1 {
			plural = "argument"
		}
		return value.NewFault("%s: want %d %s, got %d", display, a.Min, plural, got)
	}
	return value.NewFault("%s: want %d..%d arguments, got %d", display, a.Min, a.Max, got)
}

// Builtin is a HAL-level function implementing value.Callable.
type Builtin struct {
	name  string
	arity Arity
	fn    BuiltinFunc
}

func (b *Builtin) Type() value.Type { return value.FunctionType }
func (b *Builtin) Inspect() string  { return fmt.Sprintf("<builtin: %s>", b.name) }

func (b *Builtin) Call(args []value.Value) value.Value {
	if !b.arity.accepts(len(args)) {
		return b.arity.fault(b.name, len(args))
	}
	return b.fn(args)
}

// Verify Builtin implements Callable.
var _ value.Callable = (*Builtin)(nil)

// GoRuntime implements hal.RuntimeContract using Go substrate bindings. The
// runtime dispatch table is flat: all surviving substrate builtins are leaf
// functions. Host-crossing operations are documented separately in hostBindings.
type GoRuntime struct {
	builtins     map[string]*Builtin
	hostBindings map[string]hal.HostOperation
	stdout       io.Writer
	mu           sync.RWMutex
}

// Verify GoRuntime implements RuntimeContract.
var _ hal.RuntimeContract = (*GoRuntime)(nil)
var _ hal.HostOperationProvider = (*GoRuntime)(nil)

// NewGoRuntime creates a new Go runtime substrate.
func NewGoRuntime() *GoRuntime {
	rt := &GoRuntime{
		builtins:     make(map[string]*Builtin),
		hostBindings: make(map[string]hal.HostOperation),
		stdout:       os.Stdout,
	}
	rt.registerHAL()
	return rt
}

// SetIO replaces the runtime-owned output endpoint. The input argument is
// ignored and retained only so existing test harnesses can keep a stable call
// shape while Nezha has no host input operation. A nil output restores stdout.
func (g *GoRuntime) SetIO(_ io.Reader, out io.Writer) {
	g.mu.Lock()
	if out == nil {
		out = os.Stdout
	}
	g.stdout = out
	g.mu.Unlock()
}

func (g *GoRuntime) Execute(name string, args []value.Value) (value.Value, error) {
	g.mu.RLock()
	b, ok := g.lookupBuiltin(name)
	g.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("builtin %s not found", name)
	}

	result := b.Call(args)
	if fault, ok := result.(*value.Fault); ok {
		return nil, fmt.Errorf("%s", fault.Message)
	}
	return result, nil
}

// authorityKey returns the architectural grant checked for a runtime binding.
// Canonical host operations are authorized by HAL identity; leaf runtime
// primitives use their implementation primitive name.
func (g *GoRuntime) authorityKey(name string) string {
	g.mu.RLock()
	op, ok := g.hostBindings[name]
	g.mu.RUnlock()
	if ok {
		return op.Authority
	}
	return name
}

// HasBuiltin checks whether a runtime binding is executable under the supplied
// lexical authority. Scope is intentionally not consulted.
func (g *GoRuntime) HasBuiltin(name string, authority value.Authority) bool {
	if !authority.Allows(g.authorityKey(name)) {
		return false
	}
	g.mu.RLock()
	_, ok := g.lookupBuiltin(name)
	g.mu.RUnlock()
	return ok
}

// GetBuiltin resolves a runtime binding only when lexical authority grants its
// architectural key. Host operations use canonical HAL identities; other
// primitives use their implementation primitive names.
func (g *GoRuntime) GetBuiltin(name string, authority value.Authority) (value.Callable, bool) {
	if !authority.Allows(g.authorityKey(name)) {
		return nil, false
	}
	g.mu.RLock()
	b, ok := g.lookupBuiltin(name)
	g.mu.RUnlock()
	return b, ok
}

// BuiltinNames returns the architectural runtime names visible to tooling.
// Primitive classification and scope visibility are engine-owned; the Go
// substrate binds implementations to that vocabulary.
func (g *GoRuntime) BuiltinNames(scope value.Scope) []string {
	return primitives.NamesForScope(scope)
}

// registerHost registers a leaf builtin that crosses the HAL boundary and
// attaches the canonical host-operation contract that it realizes.
func (g *GoRuntime) registerHost(op hal.HostOperation, arity Arity, fn BuiltinFunc) {
	if op.Identity == "" || op.Primitive == "" || op.SubstrateProvenance == "" {
		panic("host operation registration requires identity, primitive, and substrate provenance")
	}
	if _, exists := g.hostBindings[op.Primitive]; exists {
		panic(fmt.Sprintf("duplicate host operation registration: %s", op.Primitive))
	}
	g.registerBuiltin(op.Primitive, arity, fn)
	g.hostBindings[op.Primitive] = op
}
