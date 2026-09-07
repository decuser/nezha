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

// BuiltinFunc is a HAL-level function that may use evaluation context.
// Simple builtins ignore ctx; intrinsics such as apply use it.
type BuiltinFunc func(args []value.Value, ctx *hal.EvalContext) value.Value

// Builtin is a HAL-level function implementing value.Callable.
type Builtin struct {
	name         string
	fn           BuiltinFunc
	runtime      *GoRuntime // back-reference for context access
	needsContext bool
}

func (b *Builtin) Type() value.Type { return value.FunctionType }
func (b *Builtin) Inspect() string  { return fmt.Sprintf("<builtin: %s>", b.name) }

// Call invokes the builtin. Context is retrieved from the runtime's current context.
func (b *Builtin) Call(args []value.Value) value.Value {
	return b.fn(args, nil)
}

func (b *Builtin) CallWithContext(args []value.Value, ctx *hal.EvalContext) value.Value {
	return b.fn(args, ctx)
}

func (b *Builtin) NeedsEvalContext() bool { return b.needsContext }

// Verify Builtin implements Callable
var _ value.Callable = (*Builtin)(nil)
var _ hal.ContextCallable = (*Builtin)(nil)
var _ hal.EvalContextRequired = (*Builtin)(nil)

// GoRuntime implements hal.RuntimeContract using Go substrate bindings. Native
// machinery is separated by architectural role, and only canonical host
// operations populate hostBindings. User-visible names are defined in Nezha.
type GoRuntime struct {
	intrinsics        map[string]*Builtin
	runtimePrimitives map[string]*Builtin
	providers         map[string]*Builtin
	hostRegistry      map[string]*Builtin
	services          map[string]*Builtin
	hostBindings      map[string]hal.HostOperation
	stdout            io.Writer
	mu                sync.RWMutex
}

// Verify GoRuntime implements RuntimeContract
var _ hal.RuntimeContract = (*GoRuntime)(nil)
var _ hal.HostOperationProvider = (*GoRuntime)(nil)

// NewGoRuntime creates a new Go runtime substrate.
func NewGoRuntime() *GoRuntime {
	rt := &GoRuntime{
		intrinsics:        make(map[string]*Builtin),
		runtimePrimitives: make(map[string]*Builtin),
		providers:         make(map[string]*Builtin),
		hostRegistry:      make(map[string]*Builtin),
		services:          make(map[string]*Builtin),
		hostBindings:      make(map[string]hal.HostOperation),
		stdout:            os.Stdout,
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

func (g *GoRuntime) Execute(name string, args []value.Value, ctx *hal.EvalContext) (value.Value, error) {
	g.mu.RLock()
	b, ok := g.lookupBuiltin(name)
	g.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("builtin %s not found", name)
	}

	result := b.fn(args, ctx)
	if fault, ok := result.(*value.Fault); ok {
		return nil, fmt.Errorf("%s", fault.Message)
	}
	return result, nil
}

// authorityKey returns the architectural grant checked for a runtime binding.
// Canonical host operations are authorized by HAL identity; non-HAL language,
// provider, and service primitives retain their implementation primitive name.
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

// registerHost registers an existing host-role compatibility primitive and
// attaches the canonical host-operation contract that it realizes. Not every
// host-role primitive has canonical metadata yet; canonical descriptors are a
// stricter subset of this compatibility registry.
func (g *GoRuntime) registerHost(op hal.HostOperation, fn BuiltinFunc) {
	if op.Identity == "" || op.Primitive == "" || op.SubstrateProvenance == "" {
		panic("host operation registration requires identity, primitive, and substrate provenance")
	}
	if _, exists := g.hostBindings[op.Primitive]; exists {
		panic(fmt.Sprintf("duplicate host operation registration: %s", op.Primitive))
	}
	g.registerPrimitive(op.Primitive, fn)
	g.hostBindings[op.Primitive] = op
}
