package substrate

import (
	"testing"
	"time"

	"aiki/engine/observe"
	"aiki/engine/runtime/hal"
	"aiki/engine/semantics/value"
)

type profileContextTarget struct{ seen observe.NodeID }

func (*profileContextTarget) Type() value.Type               { return value.FunctionType }
func (*profileContextTarget) Inspect() string                { return "<context-target>" }
func (*profileContextTarget) Call([]value.Value) value.Value { return value.EMPTY }
func (t *profileContextTarget) CallWithContext(_ []value.Value, ctx *hal.EvalContext) value.Value {
	t.seen = ctx.ProfileCursor.Node()
	return value.EMPTY
}

func TestApplyCarriesObservedTargetCursorWithoutChangingDispatch(t *testing.T) {
	registry := observe.NewNodeRegistry()
	schema := observe.NewObservationSchema()
	network := observe.NewNetwork(registry, schema, observe.NetworkConfig{Depth: observe.DepthEdges, EdgeCapacity: 8})
	applyID := registry.Register(observe.NodeSpec{Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "test", Name: "_apply"}, Boundary: "intrinsic"})
	targetID := registry.Register(observe.NodeSpec{Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "test", Name: "target"}, Boundary: "runtime"})
	caller := network.Cursor(applyID)
	target := &profileContextTarget{}
	ctx := &hal.EvalContext{
		ProfileCursor: caller,
		ProfileCall: func(fn value.Value, c observe.Cursor) observe.Cursor {
			if fn != target {
				t.Fatalf("unexpected target %v", fn)
			}
			return c.Call(targetID)
		},
	}
	result := halApply([]value.Value{target, &value.List{}}, ctx)
	if value.IsFault(result) {
		t.Fatalf("halApply: %s", result.Inspect())
	}
	if target.seen != targetID {
		t.Fatalf("target cursor = %d, want %d", target.seen, targetID)
	}
	snap := network.Snapshot()
	if len(snap.Edges) != 1 || snap.Edges[0].Caller != applyID || snap.Edges[0].Callee != targetID || snap.Edges[0].Count != 1 {
		t.Fatalf("apply edge = %#v", snap.Edges)
	}
}

func TestApplyUsesTargetedAllocationCallbackOnObservedTargetEdge(t *testing.T) {
	registry := observe.NewNodeRegistry()
	schema := observe.NewObservationSchema()
	network := observe.NewNetwork(registry, schema, observe.NetworkConfig{Depth: observe.DepthEdges, EdgeCapacity: 8})
	applyID := registry.Register(observe.NodeSpec{Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "test", Name: "_apply"}, Boundary: "intrinsic"})
	targetID := registry.Register(observe.NodeSpec{Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "test", Name: "target"}, Boundary: "runtime"})
	caller := network.Cursor(applyID)
	target := &profileContextTarget{}
	calls := 0
	ctx := &hal.EvalContext{
		ProfileCursor: caller,
		ProfileCall: func(fn value.Value, c observe.Cursor) observe.Cursor {
			if fn != target {
				t.Fatalf("unexpected target %v", fn)
			}
			return c.Call(targetID)
		},
		ProfileAllocationCall: func(gotCaller, gotCallee observe.Cursor, call func() value.Value) value.Value {
			calls++
			if gotCaller.Node() != applyID || gotCallee.Node() != targetID {
				t.Fatalf("allocation edge = %d -> %d, want %d -> %d", gotCaller.Node(), gotCallee.Node(), applyID, targetID)
			}
			return call()
		},
	}
	result := halApply([]value.Value{target, &value.List{}}, ctx)
	if value.IsFault(result) {
		t.Fatalf("halApply: %s", result.Inspect())
	}
	if calls != 1 {
		t.Fatalf("allocation callback calls = %d, want 1", calls)
	}
	if target.seen != targetID {
		t.Fatalf("target cursor = %d, want %d", target.seen, targetID)
	}
}

func TestApplyTimingUsesObservedTargetEdgeAndNestedSelfAccounting(t *testing.T) {
	registry := observe.NewNodeRegistry()
	schema := observe.NewObservationSchema()
	network := observe.NewNetwork(registry, schema, observe.NetworkConfig{Depth: observe.DepthEdges, EdgeCapacity: 8, Timing: true})
	applyID := registry.Register(observe.NodeSpec{Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "test", Name: "_apply"}, Boundary: "intrinsic"})
	targetID := registry.Register(observe.NodeSpec{Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "test", Name: "target"}, Boundary: "runtime"})
	caller := network.Cursor(applyID)
	target := &profileContextTarget{}
	stack := network.NewTimingStack()
	stack.Push(0, applyID)
	ctx := &hal.EvalContext{
		ProfileCursor: caller,
		ProfileTiming: stack,
		ProfileCall: func(fn value.Value, c observe.Cursor) observe.Cursor {
			if fn != target {
				t.Fatalf("unexpected target %v", fn)
			}
			return c.Call(targetID)
		},
	}
	start := time.Now()
	result := halApply([]value.Value{target, &value.List{}}, ctx)
	if value.IsFault(result) {
		t.Fatalf("halApply: %s", result.Inspect())
	}
	if time.Since(start) < 0 {
		t.Fatal("clock moved backwards")
	}
	stack.Pop()
	snap := network.Snapshot()
	rows := observe.TimingRows(observe.GraphMeasurement{Nodes: registry.Nodes(), Edges: snap.Edges, Timings: snap.Timings})
	if len(rows) != 2 {
		t.Fatalf("timing rows=%d want 2", len(rows))
	}
	var applyRow, targetRow observe.TimingRow
	for _, row := range rows {
		switch row.CalleeID {
		case applyID:
			applyRow = row
		case targetID:
			targetRow = row
		}
	}
	if targetRow.InclusiveNS <= 0 {
		t.Fatalf("target inclusive=%d", targetRow.InclusiveNS)
	}
	if applyRow.InclusiveNS < targetRow.InclusiveNS {
		t.Fatalf("apply inclusive=%d target=%d", applyRow.InclusiveNS, targetRow.InclusiveNS)
	}
	if applyRow.SelfNS > applyRow.InclusiveNS {
		t.Fatalf("apply self=%d inclusive=%d", applyRow.SelfNS, applyRow.InclusiveNS)
	}
}
