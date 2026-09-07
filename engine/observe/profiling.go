// Package observe defines neutral observation contracts shared by the Nezha
// evaluator, runtime substrate, and semantic values. It intentionally imports
// no higher-level engine packages so low-level semantic state can participate
// in observation without depending on the engine facade.
package observe

import (
	"sort"
	"sync"
)

// SemanticKind names a Nezha-level operation. These names describe language
// work, not incidental work performed by a particular substrate.
type SemanticKind string

const (
	SemanticArithmetic SemanticKind = "arithmetic"
	SemanticComparison SemanticKind = "comparison"
	SemanticCall       SemanticKind = "call"
	SemanticIteration  SemanticKind = "iteration"
	SemanticIndex      SemanticKind = "index"
	SemanticSend       SemanticKind = "send"
	SemanticReceive    SemanticKind = "receive"
	SemanticStoreRead  SemanticKind = "store_read"
	SemanticStoreWrite SemanticKind = "store_write"
)

// SemanticSite identifies the Nezha source site responsible for semantic work.
type SemanticSite struct {
	File     string
	Line     int
	Col      int
	Function string
	Detail   string
	Source   string
}

// SemanticProbe receives Nezha-level work events at semantic choke points.
// Implementations must be safe for concurrent use because spawn can execute
// Nezha code concurrently.
type SemanticProbe interface {
	Hit(kind SemanticKind, site SemanticSite)
}

// AttributionProbe is implemented by probes that want source-level identity.
// Summary-only capture does not request sites, keeping its hot path small.
type AttributionProbe interface {
	SemanticProbe
	WantsSites() bool
}

// SemanticCounts is a stable snapshot of Nezha-level work.
type SemanticCounts struct {
	Arithmetic int64
	Comparison int64
	Call       int64
	Iteration  int64
	Index      int64
	Send       int64
	Receive    int64
	StoreRead  int64
	StoreWrite int64
}

// NumberRealizationCounts describes hidden runtime representations observed
// while evaluating numeric operations. These are realization facts, not Nezha
// semantic units; representation remains unobservable to Nezha programs.
type NumberRealizationCounts struct {
	ResultSmallInteger    int64
	ResultCompactRational int64
	ResultBinaryCarrier   int64
	ResultBigRational     int64
	BinaryCertified       int64
	BinaryFallback        int64
	PromotedBigRational   int64
}

// NumberRealizationEventIDs binds the intentional numeric realization facts to
// the network schema. Arithmetic-result and call-return populations remain
// distinct because they answer different questions. Classification overlap
// (for example binary-certified plus result-binary-carrier) is represented by
// separate irreducible events, not by a second accounting subsystem.
type NumberRealizationEventIDs struct {
	ArithmeticSmallInteger    EventID
	ArithmeticCompactRational EventID
	ArithmeticBinaryCarrier   EventID
	ArithmeticBigRational     EventID
	BinaryCertified           EventID
	BinaryFallback            EventID
	PromotedBigRational       EventID

	CallSmallInteger    EventID
	CallCompactRational EventID
	CallBinaryCarrier   EventID
	CallBigRational     EventID
}

func RegisterNumberRealizationEvents(schema *ObservationSchema) NumberRealizationEventIDs {
	return NumberRealizationEventIDs{
		ArithmeticSmallInteger:    schema.RegisterEvent("number.arithmetic.result.small_integer"),
		ArithmeticCompactRational: schema.RegisterEvent("number.arithmetic.result.compact_rational"),
		ArithmeticBinaryCarrier:   schema.RegisterEvent("number.arithmetic.result.binary_carrier"),
		ArithmeticBigRational:     schema.RegisterEvent("number.arithmetic.result.big_rational"),
		BinaryCertified:           schema.RegisterEvent("number.arithmetic.binary_certified"),
		BinaryFallback:            schema.RegisterEvent("number.arithmetic.binary_fallback"),
		PromotedBigRational:       schema.RegisterEvent("number.arithmetic.promoted_big_rational"),
		CallSmallInteger:          schema.RegisterEvent("number.call_return.small_integer"),
		CallCompactRational:       schema.RegisterEvent("number.call_return.compact_rational"),
		CallBinaryCarrier:         schema.RegisterEvent("number.call_return.binary_carrier"),
		CallBigRational:           schema.RegisterEvent("number.call_return.big_rational"),
	}
}

// CallRealizationEventIDs binds irreducible call-realization observations to
// the profiling network. Function/primitive transition populations are captured
// separately: flat profiles use transition events while edge profiles derive
// them from caller->callee edges.
type CallRealizationEventIDs struct {
	FunctionTransition     EventID
	PrimitiveTransition    EventID
	TailReuse              EventID
	TailEnvReuse           EventID
	ArgArity0              EventID
	ArgArity1              EventID
	ArgArity2              EventID
	ArgArity3              EventID
	ArgArity4              EventID
	ArgArity5Plus          EventID
	ArgFrameNewInline      EventID
	ArgFrameNewPromoted    EventID
	ArgFrameReusedInline   EventID
	ArgFrameReusedPromoted EventID
	ArgDurable             EventID
	ArgTailTransfer        EventID
}

type CallRealizationMeasureIDs struct {
	ArgsEvaluated MeasureID
}

func RegisterCallRealizationEvents(schema *ObservationSchema) (CallRealizationEventIDs, CallRealizationMeasureIDs) {
	return CallRealizationEventIDs{
			FunctionTransition:     schema.RegisterEvent("call.transition.function"),
			PrimitiveTransition:    schema.RegisterEvent("call.transition.primitive"),
			TailReuse:              schema.RegisterEvent("call.tail_reuse"),
			TailEnvReuse:           schema.RegisterEvent("call.tail_env_reuse"),
			ArgArity0:              schema.RegisterEvent("call.arg.arity.0"),
			ArgArity1:              schema.RegisterEvent("call.arg.arity.1"),
			ArgArity2:              schema.RegisterEvent("call.arg.arity.2"),
			ArgArity3:              schema.RegisterEvent("call.arg.arity.3"),
			ArgArity4:              schema.RegisterEvent("call.arg.arity.4"),
			ArgArity5Plus:          schema.RegisterEvent("call.arg.arity.5_plus"),
			ArgFrameNewInline:      schema.RegisterEvent("call.arg_frame.new_inline"),
			ArgFrameNewPromoted:    schema.RegisterEvent("call.arg_frame.new_promoted"),
			ArgFrameReusedInline:   schema.RegisterEvent("call.arg_frame.reused_inline"),
			ArgFrameReusedPromoted: schema.RegisterEvent("call.arg_frame.reused_promoted"),
			ArgDurable:             schema.RegisterEvent("call.arg.durable"),
			ArgTailTransfer:        schema.RegisterEvent("call.arg.tail_transfer"),
		}, CallRealizationMeasureIDs{
			ArgsEvaluated: schema.RegisterMeasure("call.args_evaluated"),
		}
}

// CallRealizationCounts is a report DTO derived from the profiling network.
// Function and Primitive are truthful call-target populations, not the legacy
// applyFunctionOwned branch-entry counters they replace.
type CallRealizationCounts struct {
	Function     int64
	Primitive    int64
	TailReuse    int64
	TailEnvReuse int64

	ArgArity0     int64
	ArgArity1     int64
	ArgArity2     int64
	ArgArity3     int64
	ArgArity4     int64
	ArgArity5Plus int64
	ArgsEvaluated int64

	ArgFrameNew      int64
	ArgFrameReused   int64
	ArgFramePromoted int64
	ArgDurable       int64
	ArgTailTransfer  int64
}

// ListRealizationEventIDs binds persistent-list append realization events to
// the profiling network. These describe physical representation outcomes, not
// Nezha semantic operations or user-visible list kinds.
type ListRealizationEventIDs struct {
	FrontierPromoted EventID
	FrontierExtended EventID
	FrontierGrown    EventID
	FrontierForked   EventID
}

type ListRealizationMeasureIDs struct {
	ElementsCopied        MeasureID
	BackingSlotsAllocated MeasureID
}

func RegisterListRealizationEvents(schema *ObservationSchema) (ListRealizationEventIDs, ListRealizationMeasureIDs) {
	return ListRealizationEventIDs{
			FrontierPromoted: schema.RegisterEvent("list.frontier.promoted"),
			FrontierExtended: schema.RegisterEvent("list.frontier.extended"),
			FrontierGrown:    schema.RegisterEvent("list.frontier.grown"),
			FrontierForked:   schema.RegisterEvent("list.frontier.forked"),
		}, ListRealizationMeasureIDs{
			ElementsCopied:        schema.RegisterMeasure("list.elements_copied"),
			BackingSlotsAllocated: schema.RegisterMeasure("list.backing_slots_allocated"),
		}
}

// ListRealizationCounts is a report DTO derived from the profiling network.
// These are profiling facts, not Nezha semantic units or user-visible kinds.
type ListRealizationCounts struct {
	FrontierPromoted      int64
	FrontierExtended      int64
	FrontierGrown         int64
	FrontierForked        int64
	ElementsCopied        int64
	BackingSlotsAllocated int64
}

// ListRealizationProbe is the narrow observation hook used at the append
// realization choke point. Implementations forward the facts into their
// observation substrate; this interface owns no accounting state.
type ListRealizationProbe interface {
	SemanticProbe
	RecordListAppend(promoted, extended, grown, forked bool, copied, allocated int)
}

// EnvKind classifies hidden environment realizations for profiling.
// These are runtime/storage categories, not Nezha scope kinds.
type EnvKind uint8

const (
	EnvKindRoot EnvKind = iota
	EnvKindEnclosed
	EnvKindCall
	EnvKindIsolated
)

// EnvRealizationEventIDs binds hidden environment realization facts to the
// profiling network. Threshold events are monotone high-water crossings within
// one logical environment lifetime; report buckets are derived from them.
type EnvRealizationEventIDs struct {
	PhysicalRoot     EventID
	PhysicalEnclosed EventID
	PhysicalCall     EventID
	PhysicalIsolated EventID
	LogicalCall      EventID

	CallReached1     EventID
	CallReached2     EventID
	CallReached3     EventID
	CallReached5     EventID
	EnclosedReached1 EventID
	EnclosedReached2 EventID
	EnclosedReached3 EventID
	EnclosedReached5 EventID

	CallCompactAllocation     EventID
	EnclosedCompactAllocation EventID
	CallMapPromotion          EventID
	EnclosedMapPromotion      EventID
	CallLocalNew              EventID
	CallLocalUpdate           EventID
	EnclosedLocalNew          EventID
	EnclosedLocalUpdate       EventID
}

func RegisterEnvRealizationEvents(schema *ObservationSchema) EnvRealizationEventIDs {
	return EnvRealizationEventIDs{
		PhysicalRoot:              schema.RegisterEvent("env.physical.root"),
		PhysicalEnclosed:          schema.RegisterEvent("env.physical.enclosed"),
		PhysicalCall:              schema.RegisterEvent("env.physical.call"),
		PhysicalIsolated:          schema.RegisterEvent("env.physical.isolated"),
		LogicalCall:               schema.RegisterEvent("env.logical.call"),
		CallReached1:              schema.RegisterEvent("env.call.local_highwater.reached_1"),
		CallReached2:              schema.RegisterEvent("env.call.local_highwater.reached_2"),
		CallReached3:              schema.RegisterEvent("env.call.local_highwater.reached_3"),
		CallReached5:              schema.RegisterEvent("env.call.local_highwater.reached_5"),
		EnclosedReached1:          schema.RegisterEvent("env.enclosed.local_highwater.reached_1"),
		EnclosedReached2:          schema.RegisterEvent("env.enclosed.local_highwater.reached_2"),
		EnclosedReached3:          schema.RegisterEvent("env.enclosed.local_highwater.reached_3"),
		EnclosedReached5:          schema.RegisterEvent("env.enclosed.local_highwater.reached_5"),
		CallCompactAllocation:     schema.RegisterEvent("env.call.compact_allocation"),
		EnclosedCompactAllocation: schema.RegisterEvent("env.enclosed.compact_allocation"),
		CallMapPromotion:          schema.RegisterEvent("env.call.map_promotion"),
		EnclosedMapPromotion:      schema.RegisterEvent("env.enclosed.map_promotion"),
		CallLocalNew:              schema.RegisterEvent("env.call.local.new"),
		CallLocalUpdate:           schema.RegisterEvent("env.call.local.update"),
		EnclosedLocalNew:          schema.RegisterEvent("env.enclosed.local.new"),
		EnclosedLocalUpdate:       schema.RegisterEvent("env.enclosed.local.update"),
	}
}

// EnvRealizationCounts describes physical environment allocation and logical
// local-binding pressure. Binding thresholds record the maximum number of
// ordinary local store bindings reached during one logical environment
// lifetime; borrowed call parameters are not included.
type EnvRealizationCounts struct {
	PhysicalRoot     int64
	PhysicalEnclosed int64
	PhysicalCall     int64
	PhysicalIsolated int64
	LogicalCall      int64

	CallReached1 int64
	CallReached2 int64
	CallReached3 int64
	CallReached5 int64

	EnclosedReached1 int64
	EnclosedReached2 int64
	EnclosedReached3 int64
	EnclosedReached5 int64

	CallCompactAllocations     int64
	EnclosedCompactAllocations int64
	CallMapPromotions          int64
	EnclosedMapPromotions      int64
	CallLocalNew               int64
	CallLocalUpdate            int64
	EnclosedLocalNew           int64
	EnclosedLocalUpdate        int64
}

// EnvRealizationProbe is an optional profiling extension used by the semantic
// value layer. Implementations must be safe for concurrent use.
type EnvRealizationProbe interface {
	SemanticProbe
	RecordEnvPhysical(kind EnvKind)
	RecordEnvLogicalCall()
	RecordEnvBindingThreshold(kind EnvKind, threshold int)
	RecordEnvCompactAllocation(kind EnvKind)
	RecordEnvMapPromotion(kind EnvKind)
	RecordEnvLocalSet(kind EnvKind, update bool)
}

// SemanticSiteCount records the number of observations at one semantic site.
type SemanticSiteCount struct {
	Kind  SemanticKind
	Site  SemanticSite
	Count int64
}

// SemanticMeasurement is the result of one measured Nezha computation.
type SemanticMeasurement struct {
	Counts        SemanticCounts
	Numbers       NumberRealizationCounts
	CallNumbers   NumberRealizationCounts
	Calls         CallRealizationCounts
	Lists         ListRealizationCounts
	Envs          EnvRealizationCounts
	Sites         []SemanticSiteCount
	Graph         GraphMeasurement
	HostElapsedNS int64
}

// GraphMeasurement is the immutable post-execution call-network view exposed
// to analysis/reporting. It contains no independent accounting: Nodes are
// registry metadata and Edges are the network's captured direct transitions.
type GraphMeasurement struct {
	Nodes              []Node
	Edges              []EdgeCount
	Allocations        []EdgeAllocationCount
	Timings            []EdgeTiming
	EdgeOverflow       int64
	AllocationOverflow int64
	TimingOverflow     int64
}

// TimingRow resolves one timed caller->callee identity with its existing call
// population. Timing owns durations only; Count is derived from Graph.Edges.
type TimingRow struct {
	Caller      Node
	Callee      Node
	CallerID    NodeID
	CalleeID    NodeID
	Count       int64
	InclusiveNS int64
	SelfNS      int64
}

// TimingRows returns timed call identities ordered by descending self time,
// then inclusive time. CallerID may be zero for a measured root span.
func TimingRows(g GraphMeasurement) []TimingRow {
	if len(g.Timings) == 0 {
		return nil
	}
	nodes := make(map[NodeID]Node, len(g.Nodes))
	for _, node := range g.Nodes {
		nodes[node.ID] = node
	}
	counts := make(map[[2]NodeID]int64, len(g.Edges))
	for _, edge := range g.Edges {
		counts[[2]NodeID{edge.Caller, edge.Callee}] += edge.Count
	}
	out := make([]TimingRow, 0, len(g.Timings))
	for _, timing := range g.Timings {
		row := TimingRow{
			CallerID: timing.Caller, CalleeID: timing.Callee,
			Count:       counts[[2]NodeID{timing.Caller, timing.Callee}],
			InclusiveNS: timing.InclusiveNS, SelfNS: timing.SelfNS,
		}
		if timing.Caller != 0 {
			row.Caller = nodes[timing.Caller]
		} else {
			row.Count = 1
		}
		row.Callee = nodes[timing.Callee]
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SelfNS != out[j].SelfNS {
			return out[i].SelfNS > out[j].SelfNS
		}
		if out[i].InclusiveNS != out[j].InclusiveNS {
			return out[i].InclusiveNS > out[j].InclusiveNS
		}
		if out[i].CallerID != out[j].CallerID {
			return out[i].CallerID < out[j].CallerID
		}
		return out[i].CalleeID < out[j].CalleeID
	})
	return out
}

type TimingRollup struct {
	Name        string
	Calls       int64
	InclusiveNS int64
	SelfNS      int64
	Nodes       int
}

func timingRollup(g GraphMeasurement, key func(Node) string) []TimingRollup {
	type accum struct {
		calls, inclusive, self int64
		nodes                  map[NodeID]struct{}
	}
	m := make(map[string]*accum)
	for _, row := range TimingRows(g) {
		name := key(row.Callee)
		if name == "" {
			name = "<unknown>"
		}
		a := m[name]
		if a == nil {
			a = &accum{nodes: make(map[NodeID]struct{})}
			m[name] = a
		}
		a.calls += row.Count
		a.inclusive += row.InclusiveNS
		a.self += row.SelfNS
		a.nodes[row.CalleeID] = struct{}{}
	}
	out := make([]TimingRollup, 0, len(m))
	for name, a := range m {
		out = append(out, TimingRollup{Name: name, Calls: a.calls, InclusiveNS: a.inclusive, SelfNS: a.self, Nodes: len(a.nodes)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SelfNS != out[j].SelfNS {
			return out[i].SelfNS > out[j].SelfNS
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func FileTimingRollup(g GraphMeasurement) []TimingRollup {
	return timingRollup(g, func(n Node) string { return n.Identity.File })
}
func ModuleTimingRollup(g GraphMeasurement) []TimingRollup {
	return timingRollup(g, func(n Node) string { return n.Module })
}
func LayerTimingRollup(g GraphMeasurement) []TimingRollup {
	return timingRollup(g, func(n Node) string { return n.Layer })
}
func BoundaryTimingRollup(g GraphMeasurement) []TimingRollup {
	return timingRollup(g, func(n Node) string { return n.Boundary })
}

type NodeTimingRow struct {
	Node        Node
	Count       int64
	InclusiveNS int64
	SelfNS      int64
}

func NodeTimingRows(g GraphMeasurement) []NodeTimingRow {
	type accum struct{ count, inclusive, self int64 }
	m := make(map[NodeID]*accum)
	nodes := make(map[NodeID]Node, len(g.Nodes))
	for _, n := range g.Nodes {
		nodes[n.ID] = n
	}
	for _, row := range TimingRows(g) {
		a := m[row.CalleeID]
		if a == nil {
			a = &accum{}
			m[row.CalleeID] = a
		}
		a.count += row.Count
		a.inclusive += row.InclusiveNS
		a.self += row.SelfNS
	}
	out := make([]NodeTimingRow, 0, len(m))
	for id, a := range m {
		out = append(out, NodeTimingRow{Node: nodes[id], Count: a.count, InclusiveNS: a.inclusive, SelfNS: a.self})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SelfNS != out[j].SelfNS {
			return out[i].SelfNS > out[j].SelfNS
		}
		if out[i].InclusiveNS != out[j].InclusiveNS {
			return out[i].InclusiveNS > out[j].InclusiveNS
		}
		return out[i].Node.ID < out[j].Node.ID
	})
	return out
}

type BoundaryTiming struct {
	From        string
	To          string
	Calls       int64
	InclusiveNS int64
	SelfNS      int64
}

func BoundaryCrossingTimings(g GraphMeasurement) []BoundaryTiming {
	type key struct{ from, to string }
	m := make(map[key]*BoundaryTiming)
	for _, row := range TimingRows(g) {
		if row.CallerID == 0 || row.Caller.Boundary == "" || row.Callee.Boundary == "" || row.Caller.Boundary == row.Callee.Boundary {
			continue
		}
		k := key{row.Caller.Boundary, row.Callee.Boundary}
		a := m[k]
		if a == nil {
			a = &BoundaryTiming{From: k.from, To: k.to}
			m[k] = a
		}
		a.Calls += row.Count
		a.InclusiveNS += row.InclusiveNS
		a.SelfNS += row.SelfNS
	}
	out := make([]BoundaryTiming, 0, len(m))
	for _, a := range m {
		out = append(out, *a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SelfNS != out[j].SelfNS {
			return out[i].SelfNS > out[j].SelfNS
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

// ProfileLabels are the stable correlation dimensions written into Go CPU
// profiles. The fixed struct is intentionally comparable so the Go substrate
// can cache pprof label contexts without allocating on every Nezha call.
type ProfileLabels struct {
	Layer     string
	Function  string
	File      string
	Line      string
	Primitive string
}

// SemanticEventIDs binds the stable semantic vocabulary to one observation
// schema. The IDs are assigned before capture begins and used directly on the
// execution path.
type SemanticEventIDs struct {
	Arithmetic EventID
	Comparison EventID
	Call       EventID
	Iteration  EventID
	Index      EventID
	Send       EventID
	Receive    EventID
	StoreRead  EventID
	StoreWrite EventID
}

func RegisterSemanticEvents(schema *ObservationSchema) SemanticEventIDs {
	return SemanticEventIDs{
		Arithmetic: schema.RegisterEvent(string(SemanticArithmetic)),
		Comparison: schema.RegisterEvent(string(SemanticComparison)),
		Call:       schema.RegisterEvent(string(SemanticCall)),
		Iteration:  schema.RegisterEvent(string(SemanticIteration)),
		Index:      schema.RegisterEvent(string(SemanticIndex)),
		Send:       schema.RegisterEvent(string(SemanticSend)),
		Receive:    schema.RegisterEvent(string(SemanticReceive)),
		StoreRead:  schema.RegisterEvent(string(SemanticStoreRead)),
		StoreWrite: schema.RegisterEvent(string(SemanticStoreWrite)),
	}
}

func (ids SemanticEventIDs) event(kind SemanticKind) EventID {
	switch kind {
	case SemanticArithmetic:
		return ids.Arithmetic
	case SemanticComparison:
		return ids.Comparison
	case SemanticCall:
		return ids.Call
	case SemanticIteration:
		return ids.Iteration
	case SemanticIndex:
		return ids.Index
	case SemanticSend:
		return ids.Send
	case SemanticReceive:
		return ids.Receive
	case SemanticStoreRead:
		return ids.StoreRead
	case SemanticStoreWrite:
		return ids.StoreWrite
	default:
		panic("unknown semantic profiling kind")
	}
}

func (ids SemanticEventIDs) kind(id EventID) SemanticKind {
	switch id {
	case ids.Arithmetic:
		return SemanticArithmetic
	case ids.Comparison:
		return SemanticComparison
	case ids.Call:
		return SemanticCall
	case ids.Iteration:
		return SemanticIteration
	case ids.Index:
		return SemanticIndex
	case ids.Send:
		return SemanticSend
	case ids.Receive:
		return SemanticReceive
	case ids.StoreRead:
		return SemanticStoreRead
	case ids.StoreWrite:
		return SemanticStoreWrite
	default:
		panic("unknown semantic profiling event ID")
	}
}

// SemanticCapture is the network-native semantic observation endpoint. It is
// intentionally only an adapter from existing evaluator choke points into the
// profiling network; accounting state lives exclusively in Network.
//
// With attribution enabled, each event is captured once as (site node,event)
// and flat totals are derived by Network.Snapshot. Without attribution, the
// event is captured once in the flat event store.
type SemanticCapture struct {
	network      *Network
	events       SemanticEventIDs
	attributed   bool
	unattributed NodeID

	siteMu  sync.Mutex
	siteIDs map[semanticSiteIdentity]NodeID
}

type semanticSiteIdentity struct {
	file     string
	line     int
	col      int
	function string
	detail   string
	source   string
}

func NewSemanticCapture(network *Network, events SemanticEventIDs, attributed bool) *SemanticCapture {
	if network == nil {
		panic("semantic capture requires profiling network")
	}
	c := &SemanticCapture{network: network, events: events, attributed: attributed}
	if attributed {
		if len(network.scopedEvents.keys) == 0 {
			panic("attributed semantic capture requires scoped-event storage")
		}
		c.siteIDs = make(map[semanticSiteIdentity]NodeID)
		c.unattributed = network.Registry.Register(NodeSpec{
			Identity: NodeIdentity{Kind: NodeRoot, Name: "<unattributed-semantic>"},
			Layer:    "semantic",
		})
	}
	return c
}

func (c *SemanticCapture) WantsSites() bool { return c != nil && c.attributed }

func (c *SemanticCapture) Hit(kind SemanticKind, site SemanticSite) {
	if c == nil {
		return
	}
	event := c.events.event(kind)
	if !c.attributed {
		c.network.Hit(event)
		return
	}
	c.network.HitScoped(c.siteNode(site), event)
}

func (c *SemanticCapture) siteNode(site SemanticSite) NodeID {
	if site.Line <= 0 {
		return c.unattributed
	}
	// Semantic source attribution uses structural source identity. The same
	// source site may execute under different dynamic frames; dynamic frame
	// identity therefore does not participate in this attribution key.
	key := semanticSiteIdentity{
		file: site.File, line: site.Line, col: site.Col,
		detail: site.Detail, source: site.Source,
	}
	c.siteMu.Lock()
	defer c.siteMu.Unlock()
	if id := c.siteIDs[key]; id != 0 {
		return id
	}
	id := c.network.Registry.Register(NodeSpec{
		Identity: NodeIdentity{
			Kind: NodeSite, Authority: site.Function, Name: site.Detail,
			File: site.File, Line: site.Line, Col: site.Col,
		},
		Layer: "semantic", Function: site.Function, Detail: site.Detail, Source: site.Source,
	})
	c.siteIDs[key] = id
	return id
}

func (c *SemanticCapture) Measurement() SemanticMeasurement {
	if c == nil {
		return SemanticMeasurement{}
	}
	snapshot := c.network.Snapshot()
	m := SemanticMeasurement{}
	for _, event := range snapshot.Events {
		switch SemanticKind(event.Name) {
		case SemanticArithmetic:
			m.Counts.Arithmetic = event.Count
		case SemanticComparison:
			m.Counts.Comparison = event.Count
		case SemanticCall:
			m.Counts.Call = event.Count
		case SemanticIteration:
			m.Counts.Iteration = event.Count
		case SemanticIndex:
			m.Counts.Index = event.Count
		case SemanticSend:
			m.Counts.Send = event.Count
		case SemanticReceive:
			m.Counts.Receive = event.Count
		case SemanticStoreRead:
			m.Counts.StoreRead = event.Count
		case SemanticStoreWrite:
			m.Counts.StoreWrite = event.Count
		}
	}
	if !c.attributed {
		return m
	}
	for _, scoped := range snapshot.ScopedEvents {
		node, ok := c.network.Registry.Node(scoped.Subject)
		if !ok || node.Identity.Kind != NodeSite {
			continue
		}
		m.Sites = append(m.Sites, SemanticSiteCount{
			Kind: c.events.kind(scoped.Event),
			Site: SemanticSite{
				File: node.Identity.File, Line: node.Identity.Line, Col: node.Identity.Col,
				Function: node.Function, Detail: node.Detail, Source: node.Source,
			},
			Count: scoped.Count,
		})
	}
	sort.Slice(m.Sites, func(i, j int) bool {
		a, b := m.Sites[i], m.Sites[j]
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Site.File != b.Site.File {
			return a.Site.File < b.Site.File
		}
		if a.Site.Line != b.Site.Line {
			return a.Site.Line < b.Site.Line
		}
		if a.Site.Col != b.Site.Col {
			return a.Site.Col < b.Site.Col
		}
		if a.Site.Function != b.Site.Function {
			return a.Site.Function < b.Site.Function
		}
		return a.Site.Detail < b.Site.Detail
	})
	return m
}

// EvalSiteCount records AST evaluation entries observed at one source site.
// It is a result view over network-owned scoped events, not accounting state.
type EvalSiteCount struct {
	Node  string
	Site  SemanticSite
	Count int64
}

// EvalCapture is the network-native endpoint for AST execution entry. Counts
// live exclusively in Network node-hit storage. AST objects cache only the
// registry-token/NodeID binding needed to reach that storage cheaply.
type EvalCapture struct {
	network *Network
}

func NewEvalCapture(network *Network) *EvalCapture {
	if network == nil {
		panic("eval capture requires profiling network")
	}
	if len(network.nodeHits) == 0 {
		panic("eval capture requires node-hit storage")
	}
	return &EvalCapture{network: network}
}

// Hit records an Eval occurrence when the AST node already carries a binding
// for this registry. It returns false only on first encounter in this identity
// space so the evaluator can construct source metadata on the slow path.
func (c *EvalCapture) Hit(binding *NodeBinding) bool {
	if c == nil || binding == nil {
		return false
	}
	id := binding.Load(c.network.Registry.Token())
	if id == 0 {
		return false
	}
	c.network.HitNode(id)
	return true
}

// BindAndHit resolves one AST node's source identity once for this registry,
// stores the compact binding on the AST object, and records the occurrence.
func (c *EvalCapture) BindAndHit(node string, site SemanticSite, binding *NodeBinding) {
	if c == nil || binding == nil || site.Line <= 0 {
		return
	}
	site.Detail = node
	id := c.network.Registry.Register(NodeSpec{
		Identity: NodeIdentity{
			Kind: NodeSite, Authority: "nezha/eval", Name: node,
			File: site.File, Line: site.Line, Col: site.Col,
		},
		Layer: "semantic", Detail: node, Source: site.Source,
	})
	binding.Store(c.network.Registry.Token(), id)
	c.network.HitNode(id)
}

// Hits resolves captured evaluation entries for coverage analysis.
func (c *EvalCapture) Hits() []EvalSiteCount {
	if c == nil {
		return nil
	}
	out := make([]EvalSiteCount, 0)
	for _, node := range c.network.Registry.Nodes() {
		if node.Identity.Kind != NodeSite || node.Identity.Authority != "nezha/eval" {
			continue
		}
		count := c.network.NodeHitCount(node.ID)
		if count == 0 {
			continue
		}
		out = append(out, EvalSiteCount{
			Node: node.Detail,
			Site: SemanticSite{
				File: node.Identity.File, Line: node.Identity.Line, Col: node.Identity.Col,
				Function: node.Function, Detail: node.Detail, Source: node.Source,
			},
			Count: count,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Site.File != b.Site.File {
			return a.Site.File < b.Site.File
		}
		if a.Site.Line != b.Site.Line {
			return a.Site.Line < b.Site.Line
		}
		if a.Site.Col != b.Site.Col {
			return a.Site.Col < b.Site.Col
		}
		return a.Node < b.Node
	})
	return out
}
