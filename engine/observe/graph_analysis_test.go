package observe

import "testing"

func testGraphMeasurement(t *testing.T) GraphMeasurement {
	t.Helper()
	r := NewNodeRegistry()
	root := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeRoot, Name: "<main>", File: "main.ai"}, Layer: "semantic", Boundary: "function", Function: "<main>"})
	f := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "f", File: "/tmp/nezha/lib/foo/foo.ai", Line: 1}, Layer: "semantic", Boundary: "function", Function: "f"})
	g := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "g", File: "/tmp/nezha/lib/foo/native.ai", Line: 2}, Layer: "semantic", Boundary: "function", Function: "g"})
	p := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodePrimitive, Authority: "runtime/primitives", Name: "_equal"}, Layer: "runtime", Boundary: "runtime", Function: "_equal"})
	return GraphMeasurement{Nodes: r.Nodes(), Edges: []EdgeCount{{Caller: root, Callee: f, Count: 2}, {Caller: f, Callee: g, Count: 3}, {Caller: g, Callee: p, Count: 5}}}
}

func TestGraphAnalysisDerivesCallsAndBoundaries(t *testing.T) {
	g := testGraphMeasurement(t)
	prims := TopPrimitives(g)
	if len(prims) != 1 || NodeDisplayName(prims[0].Node) != "_equal" || prims[0].Count != 5 {
		t.Fatalf("top primitives = %#v", prims)
	}
	b := BoundaryRollup(g)
	if len(b) != 1 || b[0].From != "function" || b[0].To != "runtime" || b[0].Count != 5 {
		t.Fatalf("boundaries = %#v", b)
	}
}

func TestGraphAnalysisCallersAndTraversal(t *testing.T) {
	g := testGraphMeasurement(t)
	callers := CallersOf(g, "_equal")
	if len(callers) != 1 || NodeDisplayName(callers[0].Caller) != "g" || callers[0].Count != 5 {
		t.Fatalf("callers = %#v", callers)
	}
	two := TraverseFrom(g, "f", 2)
	if len(two) != 2 {
		t.Fatalf("depth-2 traversal has %d edges, want 2: %#v", len(two), two)
	}
	full := TraverseFrom(g, "<main>", 0)
	if len(full) != 3 {
		t.Fatalf("full traversal has %d edges, want 3", len(full))
	}
}

func TestGraphAnalysisRollups(t *testing.T) {
	g := testGraphMeasurement(t)
	mods := ModuleRollup(g)
	if len(mods) < 1 || mods[0].Name != "foo/native" && mods[0].Name != "foo" {
		t.Fatalf("module rollup = %#v", mods)
	}
	layers := LayerRollup(g)
	if len(layers) != 2 {
		t.Fatalf("layer rollup = %#v", layers)
	}
}

func TestAllocationEdgesResolveExistingNodeIdentity(t *testing.T) {
	r := NewNodeRegistry()
	caller := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Authority: "test", Name: "work", File: "x.ai", Line: 1}, Function: "work"})
	callee := r.Register(NodeSpec{Identity: NodeIdentity{Kind: NodePrimitive, Authority: "test", Name: "_alloc"}, Function: "_alloc", Boundary: "runtime"})
	g := GraphMeasurement{
		Nodes:       r.Nodes(),
		Allocations: []EdgeAllocationCount{{Caller: caller, Callee: callee, AllocBytes: 4096, Mallocs: 4}},
	}
	rows := AllocationEdges(g)
	if len(rows) != 1 || NodeDisplayName(rows[0].Caller) != "work" || NodeDisplayName(rows[0].Callee) != "_alloc" || rows[0].AllocBytes != 4096 || rows[0].Mallocs != 4 {
		t.Fatalf("resolved allocations = %#v", rows)
	}
}
