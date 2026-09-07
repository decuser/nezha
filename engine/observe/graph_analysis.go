package observe

import (
	"path/filepath"
	"sort"
	"strings"
)

// NodeCallCount is a post-run incoming-call population for one node.
type NodeCallCount struct {
	Node  Node
	Count int64
}

// RollupCount is a post-run call population grouped by one node dimension.
type RollupCount struct {
	Name  string
	Count int64
	Nodes int
}

// ResolvedEdge is one captured direct edge with immutable node metadata.
type ResolvedEdge struct {
	Caller Node
	Callee Node
	Count  int64
}

// ResolvedAllocation is one targeted allocator delta resolved onto an existing
// caller->callee identity. Call population remains owned by the edge graph.
type ResolvedAllocation struct {
	Caller     Node
	Callee     Node
	AllocBytes int64
	Mallocs    int64
}

// CallersOf returns direct callers of all nodes whose printable function/name
// matches query. Same-named nodes remain distinct in the result.
func CallersOf(g GraphMeasurement, query string) []ResolvedEdge {
	matches := matchingNodeIDs(g.Nodes, query)
	if len(matches) == 0 {
		return nil
	}
	byID := nodeMap(g.Nodes)
	out := make([]ResolvedEdge, 0)
	for _, edge := range g.Edges {
		if !matches[edge.Callee] {
			continue
		}
		caller, ok1 := byID[edge.Caller]
		callee, ok2 := byID[edge.Callee]
		if ok1 && ok2 {
			out = append(out, ResolvedEdge{Caller: caller, Callee: callee, Count: edge.Count})
		}
	}
	sortResolvedEdges(out)
	return out
}

// TraverseFrom returns edges reachable from matching start nodes within hops.
// hops <= 0 means unrestricted traversal. Each captured direct edge appears at
// most once even when it is reachable along multiple paths.
func TraverseFrom(g GraphMeasurement, query string, hops int) []ResolvedEdge {
	starts := matchingNodeIDs(g.Nodes, query)
	if len(starts) == 0 {
		return nil
	}
	byID := nodeMap(g.Nodes)
	outgoing := make(map[NodeID][]EdgeCount)
	for _, edge := range g.Edges {
		outgoing[edge.Caller] = append(outgoing[edge.Caller], edge)
	}
	type state struct {
		id    NodeID
		depth int
	}
	queue := make([]state, 0, len(starts))
	seenNode := make(map[NodeID]int)
	for id := range starts {
		queue = append(queue, state{id: id})
		seenNode[id] = 0
	}
	seenEdge := make(map[[2]NodeID]bool)
	out := make([]ResolvedEdge, 0)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if hops > 0 && cur.depth >= hops {
			continue
		}
		for _, edge := range outgoing[cur.id] {
			key := [2]NodeID{edge.Caller, edge.Callee}
			if !seenEdge[key] {
				caller, ok1 := byID[edge.Caller]
				callee, ok2 := byID[edge.Callee]
				if ok1 && ok2 {
					out = append(out, ResolvedEdge{Caller: caller, Callee: callee, Count: edge.Count})
				}
				seenEdge[key] = true
			}
			nextDepth := cur.depth + 1
			if old, ok := seenNode[edge.Callee]; !ok || nextDepth < old {
				seenNode[edge.Callee] = nextDepth
				queue = append(queue, state{id: edge.Callee, depth: nextDepth})
			}
		}
	}
	sortResolvedEdges(out)
	return out
}

func TopNodes(g GraphMeasurement) []NodeCallCount {
	byID := nodeMap(g.Nodes)
	counts := make(map[NodeID]int64)
	for _, edge := range g.Edges {
		counts[edge.Callee] += edge.Count
	}
	out := make([]NodeCallCount, 0, len(counts))
	for id, count := range counts {
		if node, ok := byID[id]; ok {
			out = append(out, NodeCallCount{Node: node, Count: count})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return NodeDisplayName(out[i].Node) < NodeDisplayName(out[j].Node)
	})
	return out
}

func TopPrimitives(g GraphMeasurement) []NodeCallCount {
	all := TopNodes(g)
	out := all[:0]
	for _, n := range all {
		if n.Node.Identity.Kind == NodePrimitive {
			out = append(out, n)
		}
	}
	return out
}

func TopEdges(g GraphMeasurement) []ResolvedEdge {
	byID := nodeMap(g.Nodes)
	out := make([]ResolvedEdge, 0, len(g.Edges))
	for _, edge := range g.Edges {
		caller, ok1 := byID[edge.Caller]
		callee, ok2 := byID[edge.Callee]
		if ok1 && ok2 {
			out = append(out, ResolvedEdge{Caller: caller, Callee: callee, Count: edge.Count})
		}
	}
	sortResolvedEdges(out)
	return out
}

func AllocationEdges(g GraphMeasurement) []ResolvedAllocation {
	byID := nodeMap(g.Nodes)
	out := make([]ResolvedAllocation, 0, len(g.Allocations))
	for _, allocation := range g.Allocations {
		caller, ok1 := byID[allocation.Caller]
		callee, ok2 := byID[allocation.Callee]
		if ok1 && ok2 {
			out = append(out, ResolvedAllocation{
				Caller: caller, Callee: callee, AllocBytes: allocation.AllocBytes, Mallocs: allocation.Mallocs,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].AllocBytes != out[j].AllocBytes {
			return out[i].AllocBytes > out[j].AllocBytes
		}
		if out[i].Mallocs != out[j].Mallocs {
			return out[i].Mallocs > out[j].Mallocs
		}
		a := NodeDisplayName(out[i].Caller) + "->" + NodeDisplayName(out[i].Callee)
		b := NodeDisplayName(out[j].Caller) + "->" + NodeDisplayName(out[j].Callee)
		return a < b
	})
	return out
}

func FileRollup(g GraphMeasurement) []RollupCount {
	return rollup(g, func(n Node) string { return n.Identity.File })
}

func ModuleRollup(g GraphMeasurement) []RollupCount {
	return rollup(g, func(n Node) string {
		if n.Module != "" {
			return n.Module
		}
		return ModuleFromFile(n.Identity.File)
	})
}

func LayerRollup(g GraphMeasurement) []RollupCount {
	return rollup(g, func(n Node) string { return n.Layer })
}

func BoundaryRollup(g GraphMeasurement) []BoundaryCrossing {
	byID := nodeMap(g.Nodes)
	type key struct{ from, to string }
	counts := make(map[key]int64)
	for _, edge := range g.Edges {
		caller, ok1 := byID[edge.Caller]
		callee, ok2 := byID[edge.Callee]
		if !ok1 || !ok2 || caller.Boundary == "" || callee.Boundary == "" || caller.Boundary == callee.Boundary {
			continue
		}
		counts[key{caller.Boundary, callee.Boundary}] += edge.Count
	}
	out := make([]BoundaryCrossing, 0, len(counts))
	for k, count := range counts {
		out = append(out, BoundaryCrossing{From: k.from, To: k.to, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func NodeDisplayName(n Node) string {
	if n.Function != "" {
		return n.Function
	}
	if n.Identity.Name != "" {
		return n.Identity.Name
	}
	return n.Identity.Kind.String()
}

func NodeLocation(n Node) string {
	if n.Identity.File == "" {
		return ""
	}
	if n.Identity.Line > 0 {
		return n.Identity.File + ":" + itoa(n.Identity.Line)
	}
	return n.Identity.File
}

func rollup(g GraphMeasurement, dimension func(Node) string) []RollupCount {
	byID := nodeMap(g.Nodes)
	counts := make(map[string]int64)
	nodes := make(map[string]map[NodeID]bool)
	for _, edge := range g.Edges {
		n, ok := byID[edge.Callee]
		if !ok {
			continue
		}
		name := dimension(n)
		if name == "" {
			continue
		}
		counts[name] += edge.Count
		if nodes[name] == nil {
			nodes[name] = make(map[NodeID]bool)
		}
		nodes[name][edge.Callee] = true
	}
	out := make([]RollupCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, RollupCount{Name: name, Count: count, Nodes: len(nodes[name])})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func nodeMap(nodes []Node) map[NodeID]Node {
	out := make(map[NodeID]Node, len(nodes))
	for _, n := range nodes {
		out[n.ID] = n
	}
	return out
}
func matchingNodeIDs(nodes []Node, query string) map[NodeID]bool {
	out := make(map[NodeID]bool)
	for _, n := range nodes {
		if NodeDisplayName(n) == query || n.Identity.Name == query {
			out[n.ID] = true
		}
	}
	return out
}
func sortResolvedEdges(out []ResolvedEdge) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		a := NodeDisplayName(out[i].Caller) + "->" + NodeDisplayName(out[i].Callee)
		b := NodeDisplayName(out[j].Caller) + "->" + NodeDisplayName(out[j].Callee)
		if a != b {
			return a < b
		}
		return out[i].Caller.ID < out[j].Caller.ID
	})
}

// ModuleFromFile derives the canonical library/module coordinate from a Nezha source path.
// It is shared by dynamic graph rollups and static coverage anatomy.
func ModuleFromFile(file string) string {
	if file == "" {
		return ""
	}
	p := filepath.ToSlash(file)
	idx := strings.LastIndex(p, "/lib/")
	if idx >= 0 {
		p = p[idx+5:]
	} else if strings.HasPrefix(p, "lib/") {
		p = p[4:]
	} else {
		return ""
	}
	p = strings.TrimSuffix(p, ".ai")
	parts := strings.Split(p, "/")
	if len(parts) > 1 && parts[len(parts)-1] == parts[len(parts)-2] {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, "/")
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
