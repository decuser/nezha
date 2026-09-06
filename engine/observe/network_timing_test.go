package observe

import (
	"testing"
	"time"
)

func TestTimingStackAttributesInclusiveAndSelfWithoutOwningCallCounts(t *testing.T) {
	registry := NewNodeRegistry()
	root := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeRoot, Name: "root"}, Layer: "semantic"})
	parent := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "parent"}, Layer: "semantic"})
	child := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "child"}, Layer: "semantic"})
	schema := NewObservationSchema()
	n := NewNetwork(registry, schema, NetworkConfig{Depth: DepthEdges, Timing: true})
	n.Edge(root, parent)
	n.Edge(parent, child)

	stack := n.NewTimingStack()
	stack.Push(root, parent)
	time.Sleep(200 * time.Microsecond)
	stack.Push(parent, child)
	time.Sleep(200 * time.Microsecond)
	stack.Pop()
	time.Sleep(200 * time.Microsecond)
	stack.Pop()

	s := n.Snapshot()
	if len(s.Edges) != 2 {
		t.Fatalf("edges=%d want 2", len(s.Edges))
	}
	rows := TimingRows(GraphMeasurement{Nodes: registry.Nodes(), Edges: s.Edges, Timings: s.Timings})
	if len(rows) != 2 {
		t.Fatalf("timings=%d want 2", len(rows))
	}
	byCallee := map[NodeID]TimingRow{}
	for _, row := range rows {
		byCallee[row.CalleeID] = row
	}
	p := byCallee[parent]
	c := byCallee[child]
	if p.Count != 1 || c.Count != 1 {
		t.Fatalf("timing duplicated call authority: parent=%d child=%d", p.Count, c.Count)
	}
	if p.InclusiveNS <= c.InclusiveNS {
		t.Fatalf("parent inclusive=%d child inclusive=%d", p.InclusiveNS, c.InclusiveNS)
	}
	if p.SelfNS <= 0 || p.SelfNS >= p.InclusiveNS {
		t.Fatalf("parent self=%d inclusive=%d", p.SelfNS, p.InclusiveNS)
	}
	if c.SelfNS <= 0 || c.SelfNS > c.InclusiveNS {
		t.Fatalf("child self=%d inclusive=%d", c.SelfNS, c.InclusiveNS)
	}
}

func TestTimingStackForkIsIndependentDynamicBranch(t *testing.T) {
	registry := NewNodeRegistry()
	parent := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "parent"}})
	spawned := registry.Register(NodeSpec{Identity: NodeIdentity{Kind: NodeFunction, Name: "spawned"}})
	schema := NewObservationSchema()
	n := NewNetwork(registry, schema, NetworkConfig{Depth: DepthEdges, Timing: true})

	main := n.NewTimingStack()
	main.Push(0, parent)
	fork := main.Fork()
	fork.Push(parent, spawned)
	time.Sleep(200 * time.Microsecond)
	fork.Pop()
	main.Pop()

	rows := TimingRows(GraphMeasurement{Nodes: registry.Nodes(), Timings: n.Snapshot().Timings})
	if len(rows) != 2 {
		t.Fatalf("timings=%d want 2", len(rows))
	}
	var parentRow TimingRow
	for _, row := range rows {
		if row.CalleeID == parent {
			parentRow = row
		}
	}
	// The forked span must not be subtracted as a synchronous child of parent.
	if parentRow.SelfNS != parentRow.InclusiveNS {
		t.Fatalf("fork contaminated parent self: self=%d inclusive=%d", parentRow.SelfNS, parentRow.InclusiveNS)
	}
}
