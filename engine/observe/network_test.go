package observe

import (
	"sync"
	"testing"
)

func TestNodeIdentityDoesNotCollapseNames(t *testing.T) {
	r := NewNodeRegistry()
	primitive := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodePrimitive, Authority: "runtime", Name: "equal"}, Layer: "runtime"})
	fnA := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "equal", File: "a.ai", Line: 10, Col: 1}, Layer: "semantic"})
	fnB := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "equal", File: "b.ai", Line: 10, Col: 1}, Layer: "semantic"})
	anonA := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, File: "a.ai", Line: 20, Col: 2}, Layer: "semantic"})
	anonB := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, File: "a.ai", Line: 21, Col: 2}, Layer: "semantic"})

	ids := []NodeID{primitive, fnA, fnB, anonA, anonB}
	seen := map[NodeID]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("distinct node identities collapsed to %d", id)
		}
		seen[id] = true
	}
	if again := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "equal", File: "a.ai", Line: 10, Col: 1}, Layer: "semantic"}); again != fnA {
		t.Fatalf("same identity changed ID: got %d want %d", again, fnA)
	}
}

func TestNetworkFlatEventsAndMeasures(t *testing.T) {
	s := NewObservationSchema()
	call := s.RegisterEvent("semantic.call")
	copied := s.RegisterMeasure("list.elements_copied")
	r := NewNodeRegistry()
	n := NewNetwork(r, s, NetworkConfig{Depth: DepthFlat})

	for i := 0; i < 7; i++ {
		n.Hit(call)
	}
	n.Add(copied, 11)
	n.Add(copied, 5)

	snap := n.Snapshot()
	if got := snap.Events[0].Count; got != 7 {
		t.Fatalf("event count = %d, want 7", got)
	}
	if got := snap.Measures[0].Count; got != 16 {
		t.Fatalf("measure count = %d, want 16", got)
	}
	if len(snap.Edges) != 0 {
		t.Fatalf("flat network captured %d edges", len(snap.Edges))
	}
}

func TestNetworkCursorCarriesCallerIdentity(t *testing.T) {
	s := NewObservationSchema()
	r := NewNodeRegistry()
	root := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeRoot, Name: "<main>"}, Layer: "semantic"})
	caller := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "caller", File: "x.ai", Line: 1}, Layer: "semantic"})
	callee := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodePrimitive, Authority: "runtime", Name: "_equal"}, Layer: "runtime", Boundary: "runtime"})
	n := NewNetwork(r, s, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 8})

	callerCursor := n.Cursor(root).Call(caller)
	for i := 0; i < 3; i++ {
		child := callerCursor.Call(callee)
		if child.Node() != callee {
			t.Fatalf("child cursor node = %d, want %d", child.Node(), callee)
		}
	}

	snap := n.Snapshot()
	want := map[[2]NodeID]int64{
		{root, caller}:   1,
		{caller, callee}: 3,
	}
	for _, e := range snap.Edges {
		key := [2]NodeID{e.Caller, e.Callee}
		if want[key] != e.Count {
			t.Fatalf("edge %v count = %d, want %d", key, e.Count, want[key])
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing edges: %v", want)
	}
}

func TestEdgeCaptureConcurrentAndAllocationFreeAfterSetup(t *testing.T) {
	s := NewObservationSchema()
	r := NewNodeRegistry()
	a := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "a", File: "x.ai", Line: 1}})
	b := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "b", File: "x.ai", Line: 2}})
	n := NewNetwork(r, s, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 16})

	// Use a separate network for the allocation witness because AllocsPerRun
	// performs an implementation-defined warm-up invocation.
	allocNet := NewNetwork(r, s, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 16})
	allocNet.Edge(a, b)
	allocs := testing.AllocsPerRun(1000, func() { allocNet.Edge(a, b) })
	if allocs != 0 {
		t.Fatalf("steady-state edge capture allocs/run = %v, want 0", allocs)
	}

	const workers = 8
	const each = 1000
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < each; j++ {
				n.Edge(a, b)
			}
		}()
	}
	wg.Wait()

	snap := n.Snapshot()
	if len(snap.Edges) != 1 {
		t.Fatalf("edge count = %d, want 1", len(snap.Edges))
	}
	want := int64(workers * each)
	if snap.Edges[0].Count != want {
		t.Fatalf("captured count = %d, want %d", snap.Edges[0].Count, want)
	}
}

func TestEdgeOverflowIsExplicit(t *testing.T) {
	s := NewObservationSchema()
	r := NewNodeRegistry()
	a := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "a", File: "x.ai", Line: 1}})
	b := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "b", File: "x.ai", Line: 2}})
	c := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "c", File: "x.ai", Line: 3}})
	d := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "d", File: "x.ai", Line: 4}})
	n := NewNetwork(r, s, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 2})

	n.Edge(a, b)
	n.Edge(a, c)
	n.Edge(a, d)
	if got := n.Snapshot().EdgeOverflow; got != 1 {
		t.Fatalf("overflow = %d, want 1", got)
	}
}

func TestScopedEventsCaptureOnceAndDeriveFlatTotal(t *testing.T) {
	schema := NewObservationSchema()
	events := RegisterSemanticEvents(schema)
	registry := NewNodeRegistry()
	network := NewNetwork(registry, schema, NetworkConfig{ScopedEventCapacity: 16})
	capture := NewSemanticCapture(network, events, true)

	site := SemanticSite{File: "x.ai", Line: 7, Col: 3, Function: "f", Detail: "+", Source: "a + b"}
	capture.Hit(SemanticArithmetic, site)
	capture.Hit(SemanticArithmetic, site)

	m := capture.Measurement()
	if m.Counts.Arithmetic != 2 {
		t.Fatalf("arithmetic total = %d, want 2", m.Counts.Arithmetic)
	}
	if len(m.Sites) != 1 || m.Sites[0].Count != 2 {
		t.Fatalf("site counts = %+v, want one site with count 2", m.Sites)
	}
	snap := network.Snapshot()
	if len(snap.ScopedEvents) != 1 {
		t.Fatalf("scoped events = %d, want 1", len(snap.ScopedEvents))
	}
	// The flat store itself remains zero in attributed mode; Snapshot derives
	// the flat total from the single scoped fact rather than dual-writing it.
	if got := network.events[int(events.Arithmetic)].Load(); got != 0 {
		t.Fatalf("flat arithmetic storage = %d, want 0 (capture once)", got)
	}
}

func TestSemanticSitesWithSamePrintableDetailRemainDistinct(t *testing.T) {
	schema := NewObservationSchema()
	events := RegisterSemanticEvents(schema)
	registry := NewNodeRegistry()
	network := NewNetwork(registry, schema, NetworkConfig{ScopedEventCapacity: 16})
	capture := NewSemanticCapture(network, events, true)

	capture.Hit(SemanticCall, SemanticSite{File: "a.ai", Line: 10, Col: 2, Function: "f", Detail: "work"})
	capture.Hit(SemanticCall, SemanticSite{File: "b.ai", Line: 10, Col: 2, Function: "f", Detail: "work"})

	m := capture.Measurement()
	if m.Counts.Call != 2 || len(m.Sites) != 2 {
		t.Fatalf("measurement = %+v, want two distinct attributed calls", m)
	}
}

func TestCallViewsDeriveOnlyFromEdges(t *testing.T) {
	schema := NewObservationSchema()
	registry := NewNodeRegistry()
	caller := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "f", File: "a.ai", Line: 1}, Boundary: "function"})
	primitive := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodePrimitive, Authority: "runtime/primitives", Name: "_equal"}, Boundary: "runtime"})
	n := NewNetwork(registry, schema, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 8})
	cursor := n.Cursor(caller)
	cursor.Call(primitive)
	cursor.Call(primitive)

	snapshot := n.Snapshot()
	incoming := IncomingCallCounts(snapshot)
	if got := incoming[primitive]; got != 2 {
		t.Fatalf("incoming primitive calls = %d, want 2", got)
	}
	crossings := BoundaryCrossings(snapshot, registry)
	if len(crossings) != 1 || crossings[0].From != "function" || crossings[0].To != "runtime" || crossings[0].Count != 2 {
		t.Fatalf("boundary crossings = %#v", crossings)
	}
}

func TestTargetedAllocationAttributionUsesEdgeIdentity(t *testing.T) {
	r := NewNodeRegistry()
	s := NewObservationSchema()
	a := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Authority: "test", Name: "caller", File: "a.ai", Line: 1}})
	b := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodePrimitive, Authority: "test", Name: "_alloc"}, Boundary: "runtime"})
	n := NewNetwork(r, s, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 8})
	n.EnableAllocationAttribution(8)
	n.Edge(a, b)
	n.AddAllocation(a, b, 100, 2)
	n.AddAllocation(a, b, 50, 1)

	snap := n.Snapshot()
	if len(snap.Allocations) != 1 {
		t.Fatalf("allocation edges = %d, want 1", len(snap.Allocations))
	}
	got := snap.Allocations[0]
	if got.Caller != a || got.Callee != b || got.AllocBytes != 150 || got.Mallocs != 3 {
		t.Fatalf("allocation attribution = %+v, want caller=%d callee=%d bytes=150 mallocs=3", got, a, b)
	}
	if snap.AllocationOverflow != 0 {
		t.Fatalf("allocation overflow = %d, want 0", snap.AllocationOverflow)
	}
}

func TestRoutineEdgeNetworkHasNoAllocationStorage(t *testing.T) {
	r := NewNodeRegistry()
	s := NewObservationSchema()
	a := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Authority: "test", Name: "a", File: "a.ai", Line: 1}})
	b := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Authority: "test", Name: "b", File: "a.ai", Line: 2}})
	n := NewNetwork(r, s, NetworkConfig{Depth: DepthEdges, EdgeCapacity: 8})
	n.Edge(a, b)
	if got := n.Snapshot(); len(got.Allocations) != 0 || got.AllocationOverflow != 0 {
		t.Fatalf("routine edge network unexpectedly has allocation state: %+v", got)
	}
}
