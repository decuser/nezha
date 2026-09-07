package observe

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// NodeID is the compact execution-time identity of one profiling node.
// Zero is reserved for "no node"; registered nodes begin at one.
type NodeID uint32

// EventID and MeasureID are compact identities for irreducible observations.
// They are assigned while the observation schema is being built, before a
// Network is created. The hot path therefore records only integer IDs.
type EventID uint32
type MeasureID uint32

// NodeBinding caches one NodeID in a particular registry identity space.
// It is safe to embed in long-lived runtime entities that may be observed by
// different profiling networks over time. The steady-state path is one atomic
// load plus integer comparison.
type NodeBinding struct{ value atomic.Uint64 }

func (b *NodeBinding) Load(registryToken uint32) NodeID {
	if b == nil || registryToken == 0 {
		return 0
	}
	v := b.value.Load()
	if uint32(v>>32) != registryToken {
		return 0
	}
	return NodeID(uint32(v))
}

func (b *NodeBinding) Store(registryToken uint32, id NodeID) {
	if b == nil || registryToken == 0 || id == 0 {
		return
	}
	b.value.Store(uint64(registryToken)<<32 | uint64(uint32(id)))
}

// NodeKind identifies the kind of runtime entity represented by a node.
type NodeKind uint8

const (
	NodeInvalid NodeKind = iota
	NodeRoot
	NodeFunction
	NodePrimitive
	NodeSite
)

func (k NodeKind) String() string {
	switch k {
	case NodeRoot:
		return "root"
	case NodeFunction:
		return "function"
	case NodePrimitive:
		return "primitive"
	case NodeSite:
		return "site"
	default:
		return "invalid"
	}
}

// NodeIdentity is the stable semantic identity used only while registering a
// node. It deliberately contains more than a printable name: two functions
// with the same name in different source locations, and functions/primitives
// sharing a name, remain distinct nodes.
//
// Authority is the owning classification authority when one exists (for
// example a primitive role/registration authority). Instance is available for
// entities that have no durable source location; callers should prefer source
// identity and use Instance only when there is no better identity.
type NodeIdentity struct {
	Kind      NodeKind
	Authority string
	Name      string
	File      string
	Line      int
	Col       int
	Instance  uint64
}

// Node describes one registered runtime entity. Classification metadata is
// resolved off the execution path and is immutable after registration.
type Node struct {
	ID       NodeID
	Identity NodeIdentity
	Module   string
	Layer    string
	Boundary string
	Function string
	Detail   string
	Source   string
}

// NodeSpec supplies identity plus analysis metadata when registering a node.
type NodeSpec struct {
	Identity NodeIdentity
	Module   string
	Layer    string
	Boundary string
	Function string
	Detail   string
	Source   string
}

// ObservationSchema owns the stable IDs for irreducible event and numeric
// measure facts. It is an accounting vocabulary, not accounting storage.
// Once a Network is created from a schema the network snapshots the schema's
// size; callers must register all event/measure kinds before capture begins.
type ObservationSchema struct {
	mu           sync.Mutex
	events       map[string]EventID
	measures     map[string]MeasureID
	eventNames   []string
	measureNames []string
}

func NewObservationSchema() *ObservationSchema {
	return &ObservationSchema{
		events:   make(map[string]EventID),
		measures: make(map[string]MeasureID),
	}
}

func (s *ObservationSchema) RegisterEvent(name string) EventID {
	if name == "" {
		panic("profiling event name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.events[name]; ok {
		return id
	}
	id := EventID(len(s.eventNames) + 1)
	s.events[name] = id
	s.eventNames = append(s.eventNames, name)
	return id
}

func (s *ObservationSchema) RegisterMeasure(name string) MeasureID {
	if name == "" {
		panic("profiling measure name is empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.measures[name]; ok {
		return id
	}
	id := MeasureID(len(s.measureNames) + 1)
	s.measures[name] = id
	s.measureNames = append(s.measureNames, name)
	return id
}

func (s *ObservationSchema) snapshot() schemaSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return schemaSnapshot{
		eventNames:   append([]string(nil), s.eventNames...),
		measureNames: append([]string(nil), s.measureNames...),
	}
}

type schemaSnapshot struct {
	eventNames   []string
	measureNames []string
}

// NodeRegistry assigns compact identities and owns immutable node metadata.
// Registration is intentionally off the event hot path. Dynamic entities may
// be registered at first encounter, after which callers retain the NodeID.
type NodeRegistry struct {
	mu    sync.RWMutex
	token uint32
	byKey map[NodeIdentity]NodeID
	nodes []Node
}

var nextRegistryToken atomic.Uint32

func NewNodeRegistry() *NodeRegistry {
	token := nextRegistryToken.Add(1)
	if token == 0 {
		panic("profiling node-registry token space exhausted")
	}
	return &NodeRegistry{token: token, byKey: make(map[NodeIdentity]NodeID)}
}

// Token identifies one node-identity universe. Dynamic values may cache a
// compact (registry token, node ID) binding and validate it with integer loads
// on the steady-state call path.
func (r *NodeRegistry) Token() uint32 {
	if r == nil {
		return 0
	}
	return r.token
}

func (r *NodeRegistry) Register(spec NodeSpec) NodeID {
	if err := validateNodeIdentity(spec.Identity); err != nil {
		panic(err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byKey[spec.Identity]; ok {
		existing := r.nodes[int(id)-1]
		if existing.Module != spec.Module || existing.Layer != spec.Layer || existing.Boundary != spec.Boundary ||
			existing.Function != spec.Function || existing.Detail != spec.Detail || existing.Source != spec.Source {
			panic(fmt.Sprintf("profiling node %v re-registered with different classification", spec.Identity))
		}
		return id
	}
	id := NodeID(len(r.nodes) + 1)
	r.byKey[spec.Identity] = id
	r.nodes = append(r.nodes, Node{
		ID: id, Identity: spec.Identity, Module: spec.Module, Layer: spec.Layer, Boundary: spec.Boundary,
		Function: spec.Function, Detail: spec.Detail, Source: spec.Source,
	})
	return id
}

func validateNodeIdentity(id NodeIdentity) error {
	if id.Kind == NodeInvalid {
		return fmt.Errorf("profiling node kind is invalid")
	}
	if id.Kind == NodeRoot {
		if id.Name == "" {
			return fmt.Errorf("profiling root node name is empty")
		}
		return nil
	}
	if id.Name == "" && id.File == "" && id.Instance == 0 {
		return fmt.Errorf("profiling node has no stable identity")
	}
	return nil
}

func (r *NodeRegistry) Node(id NodeID) (Node, bool) {
	if id == 0 {
		return Node{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	idx := int(id) - 1
	if idx < 0 || idx >= len(r.nodes) {
		return Node{}, false
	}
	return r.nodes[idx], true
}

func (r *NodeRegistry) Nodes() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Node(nil), r.nodes...)
}

// Depth controls capture fidelity. Level zero records only events/measures;
// level one and above additionally record direct execution edges. Higher-hop
// traversal is analysis over those captured direct edges; path-sensitive call
// chain accounting is intentionally not implied by this type.
type Depth uint8

const (
	DepthFlat Depth = iota
	DepthEdges
)

// NetworkConfig fixes all bounded storage before execution starts.
type NetworkConfig struct {
	Depth               Depth
	EdgeCapacity        int
	ScopedEventCapacity int
	NodeHitCapacity     int
	Timing              bool
	TimingCapacity      int
}

const defaultEdgeCapacity = 1 << 14

// Network is the sole storage substrate intended for Nezha profiling facts.
// It has five primitive observation forms: node identity (in Registry), dense
// node hits, direct edge transitions, irreducible event occurrences, and numeric
// measures.
// Reports and rollups are derived after execution.
type Network struct {
	Registry *NodeRegistry
	Depth    Depth

	eventNames   []string
	measureNames []string
	events       []atomic.Int64
	measures     []atomic.Int64
	edges        edgeTable
	scopedEvents edgeTable
	allocations  allocationEdgeTable
	timings      timingEdgeTable
	nodeHits     []atomic.Int64
}

func NewNetwork(registry *NodeRegistry, schema *ObservationSchema, cfg NetworkConfig) *Network {
	if registry == nil {
		panic("profiling network requires node registry")
	}
	if schema == nil {
		panic("profiling network requires observation schema")
	}
	ss := schema.snapshot()
	n := &Network{
		Registry:     registry,
		Depth:        cfg.Depth,
		eventNames:   ss.eventNames,
		measureNames: ss.measureNames,
		events:       make([]atomic.Int64, len(ss.eventNames)+1),
		measures:     make([]atomic.Int64, len(ss.measureNames)+1),
	}
	if cfg.Depth >= DepthEdges {
		capacity := cfg.EdgeCapacity
		if capacity == 0 {
			capacity = defaultEdgeCapacity
		}
		n.edges = newEdgeTable(capacity)
	}
	if cfg.ScopedEventCapacity > 0 {
		n.scopedEvents = newEdgeTable(cfg.ScopedEventCapacity)
	}
	if cfg.Timing {
		capacity := cfg.TimingCapacity
		if capacity == 0 {
			capacity = defaultEdgeCapacity
		}
		n.timings = newTimingEdgeTable(capacity)
	}
	if cfg.NodeHitCapacity > 0 {
		n.nodeHits = make([]atomic.Int64, cfg.NodeHitCapacity+1)
	}
	return n
}

// Hit records one irreducible event occurrence. IDs are range-checked because
// a schema mismatch is an instrumentation defect, not a fact to silently lose.
func (n *Network) Hit(id EventID) {
	if n == nil {
		return
	}
	idx := int(id)
	if idx <= 0 || idx >= len(n.events) {
		panic("profiling event ID outside network schema")
	}
	n.events[idx].Add(1)
}

// HitNode records one occurrence at an already-resolved node identity. It is
// intended for very hot observations whose subject identity is cached by the
// execution entity. Storage is dense and preallocated; the hot path is a range
// check plus one atomic increment.
func (n *Network) HitNode(subject NodeID) {
	if n == nil {
		return
	}
	idx := int(subject)
	if idx <= 0 || idx >= len(n.nodeHits) {
		panic("profiling node hit outside configured node-hit capacity")
	}
	n.nodeHits[idx].Add(1)
}

// NodeHitCount returns the captured occurrence count for one node.
func (n *Network) NodeHitCount(subject NodeID) int64 {
	if n == nil {
		return 0
	}
	idx := int(subject)
	if idx <= 0 || idx >= len(n.nodeHits) {
		return 0
	}
	return n.nodeHits[idx].Load()
}

// HitScoped records one event occurrence attributed to a profiling subject.
// It deliberately does not also increment the flat event counter: event totals
// are derived from scoped observations at snapshot time, preserving the
// capture-once invariant.
func (n *Network) HitScoped(subject NodeID, id EventID) {
	if n == nil {
		return
	}
	idx := int(id)
	if idx <= 0 || idx >= len(n.events) {
		panic("profiling event ID outside network schema")
	}
	if subject == 0 {
		panic("profiling scoped event contains zero subject node ID")
	}
	if len(n.scopedEvents.keys) == 0 {
		panic("profiling network has no scoped-event storage")
	}
	n.scopedEvents.add(subject, NodeID(id))
}

// Add records an irreducible numeric measure (bytes, slots, copies, etc.).
func (n *Network) Add(id MeasureID, delta int64) {
	if n == nil {
		return
	}
	idx := int(id)
	if idx <= 0 || idx >= len(n.measures) {
		panic("profiling measure ID outside network schema")
	}
	n.measures[idx].Add(delta)
}

// Edge records one direct caller -> callee transition. Level zero deliberately
// ignores edges. The table is preallocated and records using integer IDs only:
// no map lookup, allocation, symbol resolution, or formatting occurs here.
func (n *Network) Edge(caller, callee NodeID) {
	if n == nil || n.Depth < DepthEdges {
		return
	}
	if caller == 0 || callee == 0 {
		panic("profiling edge contains zero node ID")
	}
	n.edges.add(caller, callee)
}

// EnableAllocationAttribution installs bounded network-owned storage for the
// optional expensive allocation dimension. It is called before execution; the
// routine flat/edge paths do not allocate this table.
func (n *Network) EnableAllocationAttribution(capacity int) {
	if n == nil {
		return
	}
	if n.Depth < DepthEdges {
		panic("allocation attribution requires profiling edges")
	}
	if capacity == 0 {
		capacity = defaultEdgeCapacity
	}
	n.allocations = newAllocationEdgeTable(capacity)
}

// AddAllocation attributes one externally measured allocation delta to an
// existing caller->callee identity. The measurement is optional and expensive;
// routine profiling never calls this method.
func (n *Network) AddAllocation(caller, callee NodeID, allocBytes, mallocs int64) {
	if n == nil {
		return
	}
	if caller == 0 || callee == 0 {
		panic("profiling allocation attribution contains zero node ID")
	}
	if len(n.allocations.keys) == 0 {
		panic("profiling network has no allocation-attribution storage")
	}
	n.allocations.add(caller, callee, allocBytes, mallocs)
}

// TimingEnabled reports whether host-time attribution is active for this network.
func (n *Network) TimingEnabled() bool {
	return n != nil && len(n.timings.keys) != 0
}

// AddTiming records host-time attribution for one dynamic caller->callee span.
// Call counts remain owned by the ordinary edge table; timing is an optional
// measure on the same identity and never increments the call population.
func (n *Network) AddTiming(caller, callee NodeID, inclusiveNS, selfNS int64) {
	if n == nil || len(n.timings.keys) == 0 || callee == 0 {
		return
	}
	if inclusiveNS < 0 {
		inclusiveNS = 0
	}
	if selfNS < 0 {
		selfNS = 0
	}
	n.timings.add(caller, callee, inclusiveNS, selfNS)
}

// TimingStack is one dynamic execution branch's nested host-time attribution
// state. It is intentionally separate per isolated/spawned branch so concurrent
// wall time is never treated as synchronous child time of another goroutine.
type TimingStack struct {
	network *Network
	frames  []timingFrame
}

type timingFrame struct {
	caller  NodeID
	callee  NodeID
	started time.Time
	childNS int64
}

func (n *Network) NewTimingStack() *TimingStack {
	if !n.TimingEnabled() {
		return nil
	}
	return &TimingStack{network: n, frames: make([]timingFrame, 0, 64)}
}

// Fork creates an independent dynamic stack in the same timing network.
func (s *TimingStack) Fork() *TimingStack {
	if s == nil || s.network == nil {
		return nil
	}
	return &TimingStack{network: s.network, frames: make([]timingFrame, 0, 64)}
}

func (s *TimingStack) Push(caller, callee NodeID) {
	if s == nil || s.network == nil || callee == 0 {
		return
	}
	s.frames = append(s.frames, timingFrame{caller: caller, callee: callee, started: time.Now()})
}

func (s *TimingStack) Pop() {
	if s == nil || len(s.frames) == 0 {
		return
	}
	i := len(s.frames) - 1
	f := s.frames[i]
	elapsed := time.Since(f.started).Nanoseconds()
	self := elapsed - f.childNS
	if self < 0 {
		self = 0
	}
	s.frames = s.frames[:i]
	s.network.AddTiming(f.caller, f.callee, elapsed, self)
	if len(s.frames) > 0 {
		s.frames[len(s.frames)-1].childNS += elapsed
	}
}

// Cursor is the execution context carried by the evaluator/runtime. It avoids
// reconstructing caller identity from names or environment inspection: the
// current node is explicit state. A call records the edge and returns the child
// context to attach to the callee's execution environment.
type Cursor struct {
	network *Network
	node    NodeID
}

func (n *Network) Cursor(node NodeID) Cursor {
	if n == nil || node == 0 {
		return Cursor{}
	}
	return Cursor{network: n, node: node}
}

func (c Cursor) Node() NodeID { return c.node }

func (c Cursor) Call(callee NodeID) Cursor {
	if c.network == nil {
		return Cursor{}
	}
	c.network.Edge(c.node, callee)
	return Cursor{network: c.network, node: callee}
}

func (c Cursor) Hit(id EventID) {
	if c.network != nil {
		c.network.Hit(id)
	}
}

func (c Cursor) Add(id MeasureID, delta int64) {
	if c.network != nil {
		c.network.Add(id, delta)
	}
}

// Snapshot is immutable post-execution accounting state. It contains compact
// IDs plus resolved names for registered event/measure kinds; node metadata is
// obtained from the registry and reporting code can derive arbitrary rollups.
type Snapshot struct {
	Events              []NamedCount
	Measures            []NamedCount
	Edges               []EdgeCount
	ScopedEvents        []ScopedEventCount
	Allocations         []EdgeAllocationCount
	Timings             []EdgeTiming
	EdgeOverflow        int64
	ScopedEventOverflow int64
	AllocationOverflow  int64
	TimingOverflow      int64
}

type NamedCount struct {
	Name  string
	Count int64
}

type EdgeCount struct {
	Caller NodeID
	Callee NodeID
	Count  int64
}

// EdgeTiming is optional host-time attribution on one dynamic call identity.
// Caller may be zero for a measured root span. Durations are nanoseconds.
type EdgeTiming struct {
	Caller      NodeID
	Callee      NodeID
	InclusiveNS int64
	SelfNS      int64
}

// EdgeAllocationCount is an optional externally measured allocator delta
// attributed to one captured caller->callee identity. It owns no call count;
// call populations remain derived from Edges.
type EdgeAllocationCount struct {
	Caller     NodeID
	Callee     NodeID
	AllocBytes int64
	Mallocs    int64
}

type ScopedEventCount struct {
	Subject NodeID
	Event   EventID
	Count   int64
}

func (n *Network) Snapshot() Snapshot {
	if n == nil {
		return Snapshot{}
	}
	out := Snapshot{
		Events:   make([]NamedCount, 0, len(n.eventNames)),
		Measures: make([]NamedCount, 0, len(n.measureNames)),
	}
	eventTotals := make([]int64, len(n.eventNames)+1)
	for i := range n.eventNames {
		eventTotals[i+1] = n.events[i+1].Load()
	}
	if len(n.scopedEvents.keys) > 0 {
		for _, pair := range n.scopedEvents.snapshot() {
			event := EventID(pair.Callee)
			idx := int(event)
			if idx <= 0 || idx >= len(eventTotals) {
				panic("profiling scoped event contains event outside network schema")
			}
			out.ScopedEvents = append(out.ScopedEvents, ScopedEventCount{Subject: pair.Caller, Event: event, Count: pair.Count})
			eventTotals[idx] += pair.Count
		}
		out.ScopedEventOverflow = n.scopedEvents.overflow.Load()
	}
	for i, name := range n.eventNames {
		out.Events = append(out.Events, NamedCount{Name: name, Count: eventTotals[i+1]})
	}
	for i, name := range n.measureNames {
		out.Measures = append(out.Measures, NamedCount{Name: name, Count: n.measures[i+1].Load()})
	}
	if n.Depth >= DepthEdges {
		out.Edges = n.edges.snapshot()
		out.EdgeOverflow = n.edges.overflow.Load()
	}
	if len(n.allocations.keys) > 0 {
		out.Allocations = n.allocations.snapshot()
		out.AllocationOverflow = n.allocations.overflow.Load()
	}
	if len(n.timings.keys) > 0 {
		out.Timings = n.timings.snapshot()
		out.TimingOverflow = n.timings.overflow.Load()
	}
	return out
}

// edgeTable is a fixed-capacity, open-addressed table. key==0 means empty.
// Valid edges always have a non-zero caller and callee, so their packed key can
// never equal the empty sentinel. Counts are independent atomics and snapshots
// are only meaningful after execution has quiesced.
type edgeTable struct {
	mask     uint64
	keys     []atomic.Uint64
	counts   []atomic.Int64
	overflow atomic.Int64
}

func newEdgeTable(capacity int) edgeTable {
	if capacity < 2 {
		capacity = 2
	}
	capacity = nextPowerOfTwo(capacity)
	return edgeTable{
		mask:   uint64(capacity - 1),
		keys:   make([]atomic.Uint64, capacity),
		counts: make([]atomic.Int64, capacity),
	}
}

func nextPowerOfTwo(n int) int {
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

func packEdge(caller, callee NodeID) uint64 {
	return (uint64(caller) << 32) | uint64(callee)
}

func (t *edgeTable) add(caller, callee NodeID) {
	encoded := packEdge(caller, callee)
	idx := mix64(encoded) & t.mask
	for probes := uint64(0); probes <= t.mask; {
		key := t.keys[idx].Load()
		if key == encoded {
			t.counts[idx].Add(1)
			return
		}
		if key == 0 {
			if t.keys[idx].CompareAndSwap(0, encoded) {
				t.counts[idx].Add(1)
				return
			}
			// A concurrent writer claimed this slot. Re-read the same slot: it
			// may have installed our edge, in which case advancing would permit
			// duplicate entries for one caller/callee pair.
			continue
		}
		probes++
		idx = (idx + 1) & t.mask
	}
	t.overflow.Add(1)
}

func (t *edgeTable) snapshot() []EdgeCount {
	out := make([]EdgeCount, 0)
	for i := range t.keys {
		encoded := t.keys[i].Load()
		if encoded == 0 {
			continue
		}
		packed := encoded
		out = append(out, EdgeCount{
			Caller: NodeID(packed >> 32),
			Callee: NodeID(uint32(packed)),
			Count:  t.counts[i].Load(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Caller != out[j].Caller {
			return out[i].Caller < out[j].Caller
		}
		return out[i].Callee < out[j].Callee
	})
	return out
}

// timingEdgeTable is the optional fixed-capacity store for attributed host
// durations. It shares caller/callee identity with direct call edges but owns no
// call count, preserving edges as the call-population authority.
type timingEdgeTable struct {
	mask        uint64
	keys        []atomic.Uint64
	inclusiveNS []atomic.Int64
	selfNS      []atomic.Int64
	overflow    atomic.Int64
}

func newTimingEdgeTable(capacity int) timingEdgeTable {
	if capacity < 2 {
		capacity = 2
	}
	capacity = nextPowerOfTwo(capacity)
	return timingEdgeTable{
		mask:        uint64(capacity - 1),
		keys:        make([]atomic.Uint64, capacity),
		inclusiveNS: make([]atomic.Int64, capacity),
		selfNS:      make([]atomic.Int64, capacity),
	}
}

func (t *timingEdgeTable) add(caller, callee NodeID, inclusiveNS, selfNS int64) {
	encoded := packEdge(caller, callee)
	if encoded == 0 {
		return
	}
	idx := mix64(encoded) & t.mask
	for probes := uint64(0); probes <= t.mask; {
		key := t.keys[idx].Load()
		if key == encoded {
			t.inclusiveNS[idx].Add(inclusiveNS)
			t.selfNS[idx].Add(selfNS)
			return
		}
		if key == 0 {
			if t.keys[idx].CompareAndSwap(0, encoded) {
				t.inclusiveNS[idx].Add(inclusiveNS)
				t.selfNS[idx].Add(selfNS)
				return
			}
			continue
		}
		probes++
		idx = (idx + 1) & t.mask
	}
	t.overflow.Add(1)
}

func (t *timingEdgeTable) snapshot() []EdgeTiming {
	out := make([]EdgeTiming, 0)
	for i := range t.keys {
		encoded := t.keys[i].Load()
		if encoded == 0 {
			continue
		}
		out = append(out, EdgeTiming{
			Caller:      NodeID(encoded >> 32),
			Callee:      NodeID(uint32(encoded)),
			InclusiveNS: t.inclusiveNS[i].Load(),
			SelfNS:      t.selfNS[i].Load(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Caller != out[j].Caller {
			return out[i].Caller < out[j].Caller
		}
		return out[i].Callee < out[j].Callee
	})
	return out
}

// allocationEdgeTable is the optional fixed-capacity store for externally
// measured allocator deltas. It shares the direct edge identity but does not
// duplicate call populations. The table exists only when targeted attribution
// is explicitly enabled.
type allocationEdgeTable struct {
	mask       uint64
	keys       []atomic.Uint64
	allocBytes []atomic.Int64
	mallocs    []atomic.Int64
	overflow   atomic.Int64
}

func newAllocationEdgeTable(capacity int) allocationEdgeTable {
	if capacity < 2 {
		capacity = 2
	}
	capacity = nextPowerOfTwo(capacity)
	return allocationEdgeTable{
		mask:       uint64(capacity - 1),
		keys:       make([]atomic.Uint64, capacity),
		allocBytes: make([]atomic.Int64, capacity),
		mallocs:    make([]atomic.Int64, capacity),
	}
}

func (t *allocationEdgeTable) add(caller, callee NodeID, allocBytes, mallocs int64) {
	encoded := packEdge(caller, callee)
	idx := mix64(encoded) & t.mask
	for probes := uint64(0); probes <= t.mask; {
		key := t.keys[idx].Load()
		if key == encoded {
			t.allocBytes[idx].Add(allocBytes)
			t.mallocs[idx].Add(mallocs)
			return
		}
		if key == 0 {
			if t.keys[idx].CompareAndSwap(0, encoded) {
				t.allocBytes[idx].Add(allocBytes)
				t.mallocs[idx].Add(mallocs)
				return
			}
			continue
		}
		probes++
		idx = (idx + 1) & t.mask
	}
	t.overflow.Add(1)
}

func (t *allocationEdgeTable) snapshot() []EdgeAllocationCount {
	out := make([]EdgeAllocationCount, 0)
	for i := range t.keys {
		encoded := t.keys[i].Load()
		if encoded == 0 {
			continue
		}
		out = append(out, EdgeAllocationCount{
			Caller:     NodeID(encoded >> 32),
			Callee:     NodeID(uint32(encoded)),
			AllocBytes: t.allocBytes[i].Load(),
			Mallocs:    t.mallocs[i].Load(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Caller != out[j].Caller {
			return out[i].Caller < out[j].Caller
		}
		return out[i].Callee < out[j].Callee
	})
	return out
}

// mix64 is the SplitMix64 finalizer, used only to distribute packed integer
// edge identities over the fixed table.
func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

// IncomingCallCounts derives node call totals from direct edges. There is no
// independently maintained per-node call counter: incoming edges are the
// accounting authority for this view.
func IncomingCallCounts(snapshot Snapshot) map[NodeID]int64 {
	out := make(map[NodeID]int64)
	for _, edge := range snapshot.Edges {
		out[edge.Callee] += edge.Count
	}
	return out
}

// BoundaryCrossing identifies one derived architectural boundary transition.
type BoundaryCrossing struct {
	From  string
	To    string
	Count int64
}

// BoundaryCrossings derives boundary transitions from classified direct edges.
// Same-boundary calls are deliberately omitted: a boundary crossing is an edge
// property, never a separately incremented callee-class histogram.
func BoundaryCrossings(snapshot Snapshot, registry *NodeRegistry) []BoundaryCrossing {
	if registry == nil {
		return nil
	}
	type key struct{ from, to string }
	counts := make(map[key]int64)
	for _, edge := range snapshot.Edges {
		caller, okCaller := registry.Node(edge.Caller)
		callee, okCallee := registry.Node(edge.Callee)
		if !okCaller || !okCallee || caller.Boundary == "" || callee.Boundary == "" || caller.Boundary == callee.Boundary {
			continue
		}
		counts[key{caller.Boundary, callee.Boundary}] += edge.Count
	}
	out := make([]BoundaryCrossing, 0, len(counts))
	for k, count := range counts {
		out = append(out, BoundaryCrossing{From: k.from, To: k.to, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}
