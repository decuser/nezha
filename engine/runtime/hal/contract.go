// Package hal defines the runtime contract that the evaluator uses
// to interact with the host platform.
package hal

import (
	"nezha/engine"
	"nezha/engine/observe"
	"nezha/engine/semantics/value"
	"nezha/engine/syntax"
	"nezha/engine/syntax/grammar"
)

// ContextCallable is a host callable that needs the active Nezha evaluation
// context. It avoids storing mutable call context in the runtime, which is
// important when spawned computations execute concurrently.
type ContextCallable interface {
	value.Callable
	CallWithContext(args []value.Value, ctx *EvalContext) value.Value
}

// EvalContextRequired is an optional callable capability. ContextCallable
// preserves the existing host-call ABI, while this capability tells the
// evaluator whether constructing an EvalContext can affect the call. A
// ContextCallable that does not require context may be invoked through Call.
type EvalContextRequired interface {
	NeedsEvalContext() bool
}

// ProbeCallable is a context-free callable that can consume an already-active
// semantic probe without requiring full EvalContext construction. It exists for
// hidden realization counters on otherwise context-free primitives.
type ProbeCallable interface {
	value.Callable
	CallWithProbe(args []value.Value, probe engine.SemanticProbe) value.Value
}

// RealizationProbeRequired advertises whether a ProbeCallable has any behavior
// to record when a semantic probe is active.
type RealizationProbeRequired interface {
	NeedsRealizationProbe() bool
}

// AsyncFaultSource exposes faults raised by spawned computations. Blocking
// concurrency operations may observe this channel so a worker fault cannot
// leave another computation waiting forever for a message that will never
// arrive.
type AsyncFaultSource interface {
	AsyncFaults() <-chan *value.Fault
	ReportAsyncFault(*value.Fault)
}

// EvalContext provides evaluation context to builtins that need it.
// Most builtins ignore this; intrinsics (import, export, apply, load, spawn)
// use it to access the evaluator, grammar, and environment.
type EvalContext struct {
	Env                   *value.Env
	Node                  *syntax.Node
	Grammar               *grammar.Grammar
	Probe                 engine.SemanticProbe
	ProfileCursor         observe.Cursor
	ProfileCall           func(fn value.Value, caller observe.Cursor) observe.Cursor
	ProfileAllocationCall func(caller, callee observe.Cursor, call func() value.Value) value.Value
	ProfileTiming         *observe.TimingStack
	Measure               func(fn value.Value, args []value.Value, attributed bool) (value.Value, engine.SemanticMeasurement)
	MeasureTimed          func(fn value.Value, args []value.Value, attributed bool) (value.Value, engine.SemanticMeasurement)
	Labels                engine.ProfileLabels
	WithProfileLabels     func(labels engine.ProfileLabels, restore engine.ProfileLabels, fn func())
	AsyncFault            <-chan *value.Fault
	ReportAsyncFault      func(*value.Fault)
	// Eval is a callback to evaluate AST nodes. Used by apply, spawn.
	Eval func(*syntax.Node, *value.Env) value.Value
}

// RuntimeContract defines the behavioral requirements of the evaluator.
// The evaluator delegates all host-level effects through this interface.
type RuntimeContract interface {
	// Execute calls a registered primitive function by name.
	// Context provides evaluation state for intrinsics that need it.
	Execute(name string, args []value.Value, ctx *EvalContext) (value.Value, error)

	// HasBuiltin checks whether a name is executable under the supplied authority.
	HasBuiltin(name string, authority value.Authority) bool

	// GetBuiltin returns a callable for the named builtin under the supplied authority.
	GetBuiltin(name string, authority value.Authority) (value.Callable, bool)
}

// ProfileLabeler is an optional runtime capability used to correlate host CPU
// samples with the Nezha computation that caused them. Runtimes that do not
// support labeled profiling need not implement it.
type ProfileLabeler interface {
	SetProfileLabels(enabled bool)
	WithProfileLabels(labels engine.ProfileLabels, restore engine.ProfileLabels, fn func())
}

// ProfileLabelState lets hot evaluator paths avoid constructing label-region
// closures when labels are disabled. A ProfileLabeler without this optional
// query is treated conservatively as enabled.
type ProfileLabelState interface {
	ProfileLabelsEnabled() bool
}

// ProfileNamed optionally supplies a stable substrate name for a callable.
type ProfileNamed interface {
	ProfileName() string
}

// ProfileDynamicCallable lets a callable created by a generic provider binding
// participate in the profiling network without requiring a statically named
// language primitive. Registration is off the hot path; the callable caches one
// NodeID per profiling identity space.
type ProfileDynamicCallable interface {
	ProfileNamed
	ProfileNodeSpec() observe.NodeSpec
	ProfileNodeBinding(registryToken uint32) (uint32, bool)
	SetProfileNodeBinding(registryToken, nodeID uint32)
}

// ProfilePrimitive is implemented by registered runtime primitives. The
// ordinal is assigned once from the authoritative primitive catalog during
// runtime registration so profiling dispatch can identify primitives without
// a steady-state name lookup.
type ProfilePrimitive interface {
	ProfileNamed
	ProfilePrimitiveOrdinal() uint32
}
