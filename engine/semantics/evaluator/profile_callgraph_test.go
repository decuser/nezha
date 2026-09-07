package evaluator

import (
	"testing"

	"nezha/engine"
	"nezha/engine/observe"
	"nezha/engine/runtime/hal"
	"nezha/engine/runtime/primitives"
	"nezha/engine/semantics/value"
)

type profileTestPrimitive struct{ ordinal uint32 }

func (*profileTestPrimitive) Type() value.Type                  { return value.FunctionType }
func (*profileTestPrimitive) Inspect() string                   { return "<profile-test-primitive>" }
func (*profileTestPrimitive) Call([]value.Value) value.Value    { return value.EMPTY }
func (*profileTestPrimitive) ProfileName() string               { return "same" }
func (p *profileTestPrimitive) ProfilePrimitiveOrdinal() uint32 { return p.ordinal }

var _ hal.ProfilePrimitive = (*profileTestPrimitive)(nil)

func TestProfileCallIdentityDoesNotCollapseSamePrintableName(t *testing.T) {
	p := NewProfileProbeWithDepth(false, observe.DepthEdges)
	f1 := &value.Function{Name: "same", ProfileFile: "a.nez", ProfileLine: 10, ProfileCol: 2}
	f2 := &value.Function{Name: "same", ProfileFile: "b.nez", ProfileLine: 10, ProfileCol: 2}
	id1 := p.userFunctionNode(f1)
	id2 := p.userFunctionNode(f2)
	if id1 == id2 {
		t.Fatalf("same-name source functions collapsed to node %d", id1)
	}

	ordinal, ok := primitives.ProfileOrdinal("_equal")
	if !ok {
		t.Fatal("_equal missing from primitive profile catalog")
	}
	pid := p.primitiveNode(&profileTestPrimitive{ordinal: ordinal})
	if pid == 0 || pid == id1 || pid == id2 {
		t.Fatalf("primitive identity collided: primitive=%d f1=%d f2=%d", pid, id1, id2)
	}
}

func TestProfileCallSteadyStateAllocatesZero(t *testing.T) {
	p := NewProfileProbeWithDepth(false, observe.DepthEdges)
	root := p.rootCursor("alloc.nez")
	fn := &value.Function{Name: "f", ProfileFile: "alloc.nez", ProfileLine: 1, ProfileCol: 1}
	// First encounter performs registration and installs the function's compact
	// registry-token/NodeID cache. The steady state below must remain integer-only.
	child := p.callCursor(fn, root)
	if child.Node() == 0 {
		t.Fatal("missing child cursor")
	}
	allocs := testing.AllocsPerRun(1000, func() { _ = p.callCursor(fn, root) })
	if allocs != 0 {
		t.Fatalf("steady-state profile call capture allocations = %v, want 0", allocs)
	}
}

func TestCallRealizationTransitionPopulationFlatAndEdgesAgree(t *testing.T) {
	ordinal, ok := primitives.ProfileOrdinal("_equal")
	if !ok {
		t.Fatal("_equal missing from primitive profile catalog")
	}
	primitive := &profileTestPrimitive{ordinal: ordinal}
	fn := &value.Function{Name: "f", ProfileFile: "calls.nez", ProfileLine: 1, ProfileCol: 1}

	flat := NewProfileProbe(false)
	flat.callCursor(fn, observe.Cursor{})
	flat.callCursor(primitive, observe.Cursor{})
	flatCalls := flat.Measurement().Calls
	if flatCalls.Function != 1 || flatCalls.Primitive != 1 {
		t.Fatalf("flat call populations = %+v, want function=1 primitive=1", flatCalls)
	}

	edges := NewProfileProbeWithDepth(false, observe.DepthEdges)
	root := edges.rootCursor("calls.nez")
	child := edges.callCursor(fn, root)
	if child.Node() == 0 {
		t.Fatal("missing function cursor")
	}
	edges.callCursor(primitive, child)
	edgeCalls := edges.Measurement().Calls
	if edgeCalls.Function != 1 || edgeCalls.Primitive != 1 {
		t.Fatalf("edge-derived call populations = %+v, want function=1 primitive=1", edgeCalls)
	}
	snapshot := edges.NetworkSnapshot()
	if got := eventCount(snapshot, edges.callEvents.FunctionTransition); got != 0 {
		t.Fatalf("edge capture also counted flat function transition: %d", got)
	}
	if got := eventCount(snapshot, edges.callEvents.PrimitiveTransition); got != 0 {
		t.Fatalf("edge capture also counted flat primitive transition: %d", got)
	}
}

func TestReturnTailCallAdvancesProfileCursor(t *testing.T) {
	ev, g := makeEvalWithProfile(t)
	ev.Profile = NewProfileProbeWithDepth(false, observe.DepthEdges)
	result := evalSource(t, ev, g, `
let sum_tail = (n, acc) {
    if n <= 0 { return acc }
    return sum_tail(n - 1, acc + n)
}
sum_tail(5, 0)
`)
	if value.IsFault(result) {
		t.Fatalf("tail recursion faulted: %s", result.Inspect())
	}
	if result.Inspect() != "15" {
		t.Fatalf("tail recursion result = %s, want 15", result.Inspect())
	}

	probe := ev.Profile
	var functionNode observe.NodeID
	for _, node := range probe.NodeRegistry().Nodes() {
		if node.Identity.Kind == observe.NodeFunction && node.Function == "sum_tail" {
			functionNode = node.ID
			break
		}
	}
	if functionNode == 0 {
		t.Fatal("sum_tail function node not registered")
	}
	var selfCalls int64
	for _, edge := range probe.NetworkSnapshot().Edges {
		if edge.Caller == functionNode && edge.Callee == functionNode {
			selfCalls += edge.Count
		}
	}
	if selfCalls != 5 {
		t.Fatalf("sum_tail self tail edges = %d, want 5", selfCalls)
	}
	if got := probe.Measurement().Calls.Function; got != 6 {
		t.Fatalf("function transition population = %d, want 6", got)
	}
}

type profileAllocPrimitive struct {
	ordinal uint32
	keep    [][]byte
}

func (*profileAllocPrimitive) Type() value.Type                  { return value.FunctionType }
func (*profileAllocPrimitive) Inspect() string                   { return "<_equal>" }
func (p *profileAllocPrimitive) ProfileName() string             { return "_equal" }
func (p *profileAllocPrimitive) ProfilePrimitiveOrdinal() uint32 { return p.ordinal }
func (p *profileAllocPrimitive) Call([]value.Value) value.Value {
	p.keep = append(p.keep, make([]byte, 64<<10))
	return value.EMPTY
}

func TestTargetedPrimitiveAllocationUsesNetworkEdgeIdentity(t *testing.T) {
	ordinal, ok := primitives.ProfileOrdinal("_equal")
	if !ok {
		t.Fatal("_equal missing from primitive profile catalog")
	}
	probe := NewProfileProbeWithDepth(false, observe.DepthEdges)
	if !probe.EnableAllocationAttribution("_equal") {
		t.Fatal("failed to enable _equal allocation attribution")
	}
	env := value.NewEnvWithScope(value.ScopeUser)
	env.SetFile("alloc-target.nez")
	env.SetSemanticProbe(probe)
	ev := New(nil, engine.SilentObserver{})
	primitive := &profileAllocPrimitive{ordinal: ordinal}
	if got := ev.applyFunction(primitive, nil, nil, env); value.IsFault(got) {
		t.Fatalf("primitive call faulted: %s", got.Inspect())
	}
	m := probe.Measurement()
	if len(m.Graph.Allocations) != 1 {
		t.Fatalf("allocation edges = %d, want 1", len(m.Graph.Allocations))
	}
	row := m.Graph.Allocations[0]
	if row.AllocBytes < 64<<10 || row.Mallocs < 1 {
		t.Fatalf("allocation attribution = %+v, want at least 64KiB and one malloc", row)
	}
	if row.Caller == 0 || row.Callee == 0 {
		t.Fatalf("allocation attribution lost edge identity: %+v", row)
	}
}
