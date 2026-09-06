package evaluator

import (
	"runtime"

	"aiki/engine"
	"aiki/engine/observe"
	"aiki/engine/runtime/hal"
	"aiki/engine/runtime/primitives"
	"aiki/engine/semantics/value"
)

// ProfileProbe is the evaluator-facing adapter around the network-owned
// profiling context. All Aiki profiling facts now terminate in the network;
// this adapter owns no accounting state of its own. Coverage is a separate
// test-tool concern and does not route through ProfileProbe.
type ProfileProbe struct {
	semantic         *observe.SemanticCapture
	numberEvents     observe.NumberRealizationEventIDs
	callEvents       observe.CallRealizationEventIDs
	callMeasures     observe.CallRealizationMeasureIDs
	listEvents       observe.ListRealizationEventIDs
	listMeasures     observe.ListRealizationMeasureIDs
	envEvents        observe.EnvRealizationEventIDs
	network          *observe.Network
	registry         *observe.NodeRegistry
	primitiveNodes   []observe.NodeID // indexed by authoritative primitive ordinal
	allocationTarget observe.NodeID
}

func NewProfileProbe(attributed bool) *ProfileProbe {
	return newProfileProbe(attributed, observe.DepthFlat, false, nil, nil)
}

// NewProfileProbeWithDepth enables the network fidelity needed by internal
// profiling tests and the later reporting cut. Existing profile behavior stays
// level zero until the command surface explicitly opts into edge capture.
func NewProfileProbeWithDepth(attributed bool, depth observe.Depth) *ProfileProbe {
	return newProfileProbe(attributed, depth, false, nil, nil)
}

func newProfileProbe(attributed bool, depth observe.Depth, timing bool, registry *observe.NodeRegistry, primitiveNodes []observe.NodeID) *ProfileProbe {
	schema := observe.NewObservationSchema()
	events := observe.RegisterSemanticEvents(schema)
	numberEvents := observe.RegisterNumberRealizationEvents(schema)
	callEvents, callMeasures := observe.RegisterCallRealizationEvents(schema)
	listEvents, listMeasures := observe.RegisterListRealizationEvents(schema)
	envEvents := observe.RegisterEnvRealizationEvents(schema)
	if registry == nil {
		registry = observe.NewNodeRegistry()
	}
	if primitiveNodes == nil {
		defs := primitives.ProfileDefinitions()
		primitiveNodes = make([]observe.NodeID, len(defs)+1)
		for _, def := range defs {
			id := registry.Register(observe.NodeSpec{
				Identity: observe.NodeIdentity{Kind: observe.NodePrimitive, Authority: "runtime/primitives", Name: def.Name},
				Layer:    "runtime", Boundary: string(def.Role), Function: def.Name,
			})
			primitiveNodes[def.Ordinal] = id
		}
	}
	cfg := observe.NetworkConfig{Depth: depth, Timing: timing}
	if attributed {
		cfg.ScopedEventCapacity = 1 << 15
	}
	network := observe.NewNetwork(registry, schema, cfg)
	return &ProfileProbe{
		semantic:       observe.NewSemanticCapture(network, events, attributed),
		numberEvents:   numberEvents,
		callEvents:     callEvents,
		callMeasures:   callMeasures,
		listEvents:     listEvents,
		listMeasures:   listMeasures,
		envEvents:      envEvents,
		network:        network,
		registry:       registry,
		primitiveNodes: primitiveNodes,
	}
}

// childMeasurement creates an independent capture network in the same node
// identity universe. This keeps function/primitive NodeIDs valid across nested
// profile.measure operations without making IDs global process state.
func (p *ProfileProbe) childMeasurement(attributed bool) *ProfileProbe {
	if p == nil {
		return NewProfileProbe(attributed)
	}
	child := newProfileProbe(attributed, p.network.Depth, false, p.registry, p.primitiveNodes)
	if p.allocationTarget != 0 {
		child.allocationTarget = p.allocationTarget
		child.network.EnableAllocationAttribution(0)
	}
	return child
}

func (p *ProfileProbe) childTimedMeasurement(attributed bool) *ProfileProbe {
	if p == nil {
		return newProfileProbe(attributed, observe.DepthEdges, true, nil, nil)
	}
	child := newProfileProbe(attributed, observe.DepthEdges, true, p.registry, p.primitiveNodes)
	return child
}

func NewTimedProfileProbe(attributed bool) *ProfileProbe {
	return newProfileProbe(attributed, observe.DepthEdges, true, nil, nil)
}

func NewTimedProfileProbeWithDepth(attributed bool, depth observe.Depth) *ProfileProbe {
	if depth < observe.DepthEdges {
		depth = observe.DepthEdges
	}
	return newProfileProbe(attributed, depth, true, nil, nil)
}

func (p *ProfileProbe) NewTimingStack() *observe.TimingStack {
	if p == nil || p.network == nil {
		return nil
	}
	return p.network.NewTimingStack()
}

func (p *ProfileProbe) BeginRootTiming(env *value.Env, file string) func() {
	if p == nil || env == nil || p.network == nil || !p.network.TimingEnabled() {
		return func() {}
	}
	cursor := p.rootCursor(file)
	env.SetProfileCursor(cursor)
	stack := p.network.NewTimingStack()
	env.SetProfileTiming(stack)
	stack.Push(0, cursor.Node())
	return stack.Pop
}

func (p *ProfileProbe) rootCursor(file string) observe.Cursor {
	if p == nil || p.network == nil {
		return observe.Cursor{}
	}
	if file == "" {
		file = "<main>"
	}
	id := p.registry.Register(observe.NodeSpec{
		Identity: observe.NodeIdentity{Kind: observe.NodeRoot, Authority: "aiki/source", Name: "<main>", File: file},
		Layer:    "semantic", Boundary: "function", Function: "<main>", Source: file,
	})
	return p.network.Cursor(id)
}

func (p *ProfileProbe) userFunctionNode(fn *value.Function) observe.NodeID {
	if p == nil || fn == nil {
		return 0
	}
	token := p.registry.Token()
	if id, ok := fn.ProfileNodeBinding(token); ok {
		return observe.NodeID(id)
	}
	name := fn.Name
	if name == "" {
		name = "<anonymous>"
	}
	identity := observe.NodeIdentity{Kind: observe.NodeFunction, Authority: "aiki/source", Name: name, File: fn.ProfileFile, Line: fn.ProfileLine, Col: fn.ProfileCol}
	if fn.ProfileFile == "" && fn.ProfileLine == 0 && fn.ProfileCol == 0 {
		identity.Instance = fn.ProfileInstanceID()
	}
	id := p.registry.Register(observe.NodeSpec{
		Identity: identity, Layer: "semantic", Boundary: "function", Function: name, Source: fn.ProfileFile,
	})
	fn.SetProfileNodeBinding(token, uint32(id))
	return id
}

func (p *ProfileProbe) primitiveNode(fn value.Callable) observe.NodeID {
	if primitive, ok := fn.(hal.ProfilePrimitive); ok {
		ordinal := primitive.ProfilePrimitiveOrdinal()
		if ordinal == 0 || int(ordinal) >= len(p.primitiveNodes) {
			return 0
		}
		return p.primitiveNodes[ordinal]
	}
	dynamic, ok := fn.(hal.ProfileDynamicCallable)
	if !ok || p == nil || p.registry == nil {
		return 0
	}
	token := p.registry.Token()
	if id, ok := dynamic.ProfileNodeBinding(token); ok {
		return observe.NodeID(id)
	}
	id := p.registry.Register(dynamic.ProfileNodeSpec())
	dynamic.SetProfileNodeBinding(token, uint32(id))
	return id
}

func (p *ProfileProbe) calleeNode(fn value.Value) observe.NodeID {
	switch f := fn.(type) {
	case *value.Function:
		return p.userFunctionNode(f)
	case value.Callable:
		return p.primitiveNode(f)
	default:
		return 0
	}
}

func (p *ProfileProbe) callCursor(fn value.Value, caller observe.Cursor) observe.Cursor {
	if p == nil || p.network == nil {
		return observe.Cursor{}
	}
	callee := p.calleeNode(fn)
	if callee == 0 {
		return observe.Cursor{}
	}
	if p.network.Depth >= observe.DepthEdges {
		if caller.Node() == 0 {
			return observe.Cursor{}
		}
		return caller.Call(callee)
	}
	node, ok := p.registry.Node(callee)
	if !ok {
		return observe.Cursor{}
	}
	switch node.Identity.Kind {
	case observe.NodeFunction:
		p.network.Hit(p.callEvents.FunctionTransition)
	case observe.NodePrimitive:
		p.network.Hit(p.callEvents.PrimitiveTransition)
	}
	return observe.Cursor{}
}

func (p *ProfileProbe) NetworkSnapshot() observe.Snapshot {
	if p == nil || p.network == nil {
		return observe.Snapshot{}
	}
	return p.network.Snapshot()
}

// EnableAllocationAttribution selects one registered primitive for expensive
// runtime allocator bracketing. Selection is resolved once to NodeID before
// execution; routine profiling does no name lookup or allocation measurement.
func (p *ProfileProbe) EnableAllocationAttribution(name string) bool {
	if p == nil || p.network == nil || p.network.Depth < observe.DepthEdges || name == "" {
		return false
	}
	for _, node := range p.registry.Nodes() {
		if node.Identity.Kind == observe.NodePrimitive && node.Identity.Name == name {
			p.allocationTarget = node.ID
			p.network.EnableAllocationAttribution(0)
			return true
		}
	}
	return false
}

func (p *ProfileProbe) allocationEnabled() bool {
	return p != nil && p.network != nil && p.allocationTarget != 0
}

// profileAllocationCall brackets only the selected primitive and records the
// process allocator delta on the existing caller->callee identity. ReadMemStats
// is intentionally absent from routine profiling. Because MemStats is process
// global, concurrent unrelated allocation can contaminate this diagnostic
// measurement; callers should use controlled witnesses when exact attribution
// matters.
func (p *ProfileProbe) profileAllocationCall(caller, callee observe.Cursor, call func() value.Value) value.Value {
	if !p.allocationEnabled() || callee.Node() != p.allocationTarget || caller.Node() == 0 {
		return call()
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	result := call()
	runtime.ReadMemStats(&after)
	allocBytes := int64(after.TotalAlloc - before.TotalAlloc)
	mallocs := int64(after.Mallocs - before.Mallocs)
	p.network.AddAllocation(caller.Node(), callee.Node(), allocBytes, mallocs)
	return result
}

func (p *ProfileProbe) NodeRegistry() *observe.NodeRegistry {
	if p == nil {
		return nil
	}
	return p.registry
}

func (p *ProfileProbe) Hit(kind observe.SemanticKind, site observe.SemanticSite) {
	if p != nil && p.semantic != nil {
		p.semantic.Hit(kind, site)
	}
}

func (p *ProfileProbe) WantsSites() bool {
	return p != nil && p.semantic != nil && p.semantic.WantsSites()
}

func (p *ProfileProbe) NumberArithmeticResult(left, right, result *value.Number) {
	if p == nil || p.network == nil || result == nil {
		return
	}
	resultRep := result.ProfileRepresentation()
	switch resultRep {
	case value.NumberSmallInteger:
		p.network.Hit(p.numberEvents.ArithmeticSmallInteger)
	case value.NumberCompactRational:
		p.network.Hit(p.numberEvents.ArithmeticCompactRational)
	case value.NumberBinaryCarrier:
		p.network.Hit(p.numberEvents.ArithmeticBinaryCarrier)
	case value.NumberBigRational:
		p.network.Hit(p.numberEvents.ArithmeticBigRational)
	}
	if left == nil || right == nil {
		return
	}
	lrep := left.ProfileRepresentation()
	rrep := right.ProfileRepresentation()
	if lrep == value.NumberBinaryCarrier && rrep == value.NumberBinaryCarrier {
		if resultRep == value.NumberBinaryCarrier || resultRep == value.NumberSmallInteger {
			p.network.Hit(p.numberEvents.BinaryCertified)
		} else {
			p.network.Hit(p.numberEvents.BinaryFallback)
		}
	}
	if resultRep == value.NumberBigRational && lrep != value.NumberBigRational && rrep != value.NumberBigRational {
		p.network.Hit(p.numberEvents.PromotedBigRational)
	}
}

func (p *ProfileProbe) NumberCallResult(result value.Value) {
	if p == nil || p.network == nil {
		return
	}
	n, ok := result.(*value.Number)
	if !ok || n == nil {
		return
	}
	switch n.ProfileRepresentation() {
	case value.NumberSmallInteger:
		p.network.Hit(p.numberEvents.CallSmallInteger)
	case value.NumberCompactRational:
		p.network.Hit(p.numberEvents.CallCompactRational)
	case value.NumberBinaryCarrier:
		p.network.Hit(p.numberEvents.CallBinaryCarrier)
	case value.NumberBigRational:
		p.network.Hit(p.numberEvents.CallBigRational)
	}
}

func eventCount(snapshot observe.Snapshot, id observe.EventID) int64 {
	idx := int(id) - 1
	if idx < 0 || idx >= len(snapshot.Events) {
		return 0
	}
	return snapshot.Events[idx].Count
}

func measureCount(snapshot observe.Snapshot, id observe.MeasureID) int64 {
	idx := int(id) - 1
	if idx < 0 || idx >= len(snapshot.Measures) {
		return 0
	}
	return snapshot.Measures[idx].Count
}

func (p *ProfileProbe) callSnapshot(snapshot observe.Snapshot) observe.CallRealizationCounts {
	ids := p.callEvents
	var functionCalls, primitiveCalls int64
	if p.network.Depth >= observe.DepthEdges {
		for _, edge := range snapshot.Edges {
			node, ok := p.registry.Node(edge.Callee)
			if !ok {
				continue
			}
			switch node.Identity.Kind {
			case observe.NodeFunction:
				functionCalls += edge.Count
			case observe.NodePrimitive:
				primitiveCalls += edge.Count
			}
		}
	} else {
		functionCalls = eventCount(snapshot, ids.FunctionTransition)
		primitiveCalls = eventCount(snapshot, ids.PrimitiveTransition)
	}

	newInline := eventCount(snapshot, ids.ArgFrameNewInline)
	newPromoted := eventCount(snapshot, ids.ArgFrameNewPromoted)
	reusedInline := eventCount(snapshot, ids.ArgFrameReusedInline)
	reusedPromoted := eventCount(snapshot, ids.ArgFrameReusedPromoted)
	return observe.CallRealizationCounts{
		Function:         functionCalls,
		Primitive:        primitiveCalls,
		TailReuse:        eventCount(snapshot, ids.TailReuse),
		TailEnvReuse:     eventCount(snapshot, ids.TailEnvReuse),
		ArgArity0:        eventCount(snapshot, ids.ArgArity0),
		ArgArity1:        eventCount(snapshot, ids.ArgArity1),
		ArgArity2:        eventCount(snapshot, ids.ArgArity2),
		ArgArity3:        eventCount(snapshot, ids.ArgArity3),
		ArgArity4:        eventCount(snapshot, ids.ArgArity4),
		ArgArity5Plus:    eventCount(snapshot, ids.ArgArity5Plus),
		ArgsEvaluated:    measureCount(snapshot, p.callMeasures.ArgsEvaluated),
		ArgFrameNew:      newInline + newPromoted,
		ArgFrameReused:   reusedInline + reusedPromoted,
		ArgFramePromoted: newPromoted + reusedPromoted,
		ArgDurable:       eventCount(snapshot, ids.ArgDurable),
		ArgTailTransfer:  eventCount(snapshot, ids.ArgTailTransfer),
	}
}

func (p *ProfileProbe) listSnapshot(snapshot observe.Snapshot) observe.ListRealizationCounts {
	ids := p.listEvents
	return observe.ListRealizationCounts{
		FrontierPromoted:      eventCount(snapshot, ids.FrontierPromoted),
		FrontierExtended:      eventCount(snapshot, ids.FrontierExtended),
		FrontierGrown:         eventCount(snapshot, ids.FrontierGrown),
		FrontierForked:        eventCount(snapshot, ids.FrontierForked),
		ElementsCopied:        measureCount(snapshot, p.listMeasures.ElementsCopied),
		BackingSlotsAllocated: measureCount(snapshot, p.listMeasures.BackingSlotsAllocated),
	}
}

func (p *ProfileProbe) envSnapshot(snapshot observe.Snapshot) observe.EnvRealizationCounts {
	ids := p.envEvents
	return observe.EnvRealizationCounts{
		PhysicalRoot:               eventCount(snapshot, ids.PhysicalRoot),
		PhysicalEnclosed:           eventCount(snapshot, ids.PhysicalEnclosed),
		PhysicalCall:               eventCount(snapshot, ids.PhysicalCall),
		PhysicalIsolated:           eventCount(snapshot, ids.PhysicalIsolated),
		LogicalCall:                eventCount(snapshot, ids.LogicalCall),
		CallReached1:               eventCount(snapshot, ids.CallReached1),
		CallReached2:               eventCount(snapshot, ids.CallReached2),
		CallReached3:               eventCount(snapshot, ids.CallReached3),
		CallReached5:               eventCount(snapshot, ids.CallReached5),
		EnclosedReached1:           eventCount(snapshot, ids.EnclosedReached1),
		EnclosedReached2:           eventCount(snapshot, ids.EnclosedReached2),
		EnclosedReached3:           eventCount(snapshot, ids.EnclosedReached3),
		EnclosedReached5:           eventCount(snapshot, ids.EnclosedReached5),
		CallCompactAllocations:     eventCount(snapshot, ids.CallCompactAllocation),
		EnclosedCompactAllocations: eventCount(snapshot, ids.EnclosedCompactAllocation),
		CallMapPromotions:          eventCount(snapshot, ids.CallMapPromotion),
		EnclosedMapPromotions:      eventCount(snapshot, ids.EnclosedMapPromotion),
		CallLocalNew:               eventCount(snapshot, ids.CallLocalNew),
		CallLocalUpdate:            eventCount(snapshot, ids.CallLocalUpdate),
		EnclosedLocalNew:           eventCount(snapshot, ids.EnclosedLocalNew),
		EnclosedLocalUpdate:        eventCount(snapshot, ids.EnclosedLocalUpdate),
	}
}

func (p *ProfileProbe) numberArithmeticSnapshot(snapshot observe.Snapshot) observe.NumberRealizationCounts {
	ids := p.numberEvents
	return observe.NumberRealizationCounts{
		ResultSmallInteger:    eventCount(snapshot, ids.ArithmeticSmallInteger),
		ResultCompactRational: eventCount(snapshot, ids.ArithmeticCompactRational),
		ResultBinaryCarrier:   eventCount(snapshot, ids.ArithmeticBinaryCarrier),
		ResultBigRational:     eventCount(snapshot, ids.ArithmeticBigRational),
		BinaryCertified:       eventCount(snapshot, ids.BinaryCertified),
		BinaryFallback:        eventCount(snapshot, ids.BinaryFallback),
		PromotedBigRational:   eventCount(snapshot, ids.PromotedBigRational),
	}
}

func (p *ProfileProbe) numberCallSnapshot(snapshot observe.Snapshot) observe.NumberRealizationCounts {
	ids := p.numberEvents
	return observe.NumberRealizationCounts{
		ResultSmallInteger:    eventCount(snapshot, ids.CallSmallInteger),
		ResultCompactRational: eventCount(snapshot, ids.CallCompactRational),
		ResultBinaryCarrier:   eventCount(snapshot, ids.CallBinaryCarrier),
		ResultBigRational:     eventCount(snapshot, ids.CallBigRational),
	}
}

func (p *ProfileProbe) TailCallReuse() {
	if p != nil && p.network != nil {
		p.network.Hit(p.callEvents.TailReuse)
	}
}
func (p *ProfileProbe) TailEnvReuse() {
	if p != nil && p.network != nil {
		p.network.Hit(p.callEvents.TailEnvReuse)
	}
}

func (p *ProfileProbe) RecordCallArity(arity int) {
	if p == nil || p.network == nil {
		return
	}
	switch arity {
	case 0:
		p.network.Hit(p.callEvents.ArgArity0)
	case 1:
		p.network.Hit(p.callEvents.ArgArity1)
	case 2:
		p.network.Hit(p.callEvents.ArgArity2)
	case 3:
		p.network.Hit(p.callEvents.ArgArity3)
	case 4:
		p.network.Hit(p.callEvents.ArgArity4)
	default:
		p.network.Hit(p.callEvents.ArgArity5Plus)
	}
	p.network.Add(p.callMeasures.ArgsEvaluated, int64(arity))
}

func (p *ProfileProbe) RecordArgFrame(reused, promoted bool) {
	if p == nil || p.network == nil {
		return
	}
	var event observe.EventID
	switch {
	case !reused && !promoted:
		event = p.callEvents.ArgFrameNewInline
	case !reused && promoted:
		event = p.callEvents.ArgFrameNewPromoted
	case reused && !promoted:
		event = p.callEvents.ArgFrameReusedInline
	case reused && promoted:
		event = p.callEvents.ArgFrameReusedPromoted
	}
	p.network.Hit(event)
}

func (p *ProfileProbe) RecordArgDurable() {
	if p != nil && p.network != nil {
		p.network.Hit(p.callEvents.ArgDurable)
	}
}

func (p *ProfileProbe) RecordArgTailTransfer() {
	if p != nil && p.network != nil {
		p.network.Hit(p.callEvents.ArgTailTransfer)
	}
}
func (p *ProfileProbe) RecordListAppend(promoted, extended, grown, forked bool, copied, allocated int) {
	if p == nil || p.network == nil {
		return
	}
	if promoted {
		p.network.Hit(p.listEvents.FrontierPromoted)
	}
	if extended {
		p.network.Hit(p.listEvents.FrontierExtended)
	}
	if grown {
		p.network.Hit(p.listEvents.FrontierGrown)
	}
	if forked {
		p.network.Hit(p.listEvents.FrontierForked)
	}
	if copied > 0 {
		p.network.Add(p.listMeasures.ElementsCopied, int64(copied))
	}
	if allocated > 0 {
		p.network.Add(p.listMeasures.BackingSlotsAllocated, int64(allocated))
	}
}

func (p *ProfileProbe) RecordEnvPhysical(kind observe.EnvKind) {
	if p == nil || p.network == nil {
		return
	}
	var event observe.EventID
	switch kind {
	case observe.EnvKindRoot:
		event = p.envEvents.PhysicalRoot
	case observe.EnvKindEnclosed:
		event = p.envEvents.PhysicalEnclosed
	case observe.EnvKindCall:
		event = p.envEvents.PhysicalCall
	case observe.EnvKindIsolated:
		event = p.envEvents.PhysicalIsolated
	}
	if event != 0 {
		p.network.Hit(event)
	}
}

func (p *ProfileProbe) RecordEnvLogicalCall() {
	if p != nil && p.network != nil {
		p.network.Hit(p.envEvents.LogicalCall)
	}
}

func (p *ProfileProbe) RecordEnvBindingThreshold(kind observe.EnvKind, threshold int) {
	if p == nil || p.network == nil {
		return
	}
	var event observe.EventID
	switch kind {
	case observe.EnvKindCall:
		switch threshold {
		case 1:
			event = p.envEvents.CallReached1
		case 2:
			event = p.envEvents.CallReached2
		case 3:
			event = p.envEvents.CallReached3
		case 5:
			event = p.envEvents.CallReached5
		}
	case observe.EnvKindEnclosed:
		switch threshold {
		case 1:
			event = p.envEvents.EnclosedReached1
		case 2:
			event = p.envEvents.EnclosedReached2
		case 3:
			event = p.envEvents.EnclosedReached3
		case 5:
			event = p.envEvents.EnclosedReached5
		}
	}
	if event != 0 {
		p.network.Hit(event)
	}
}

func (p *ProfileProbe) RecordEnvCompactAllocation(kind observe.EnvKind) {
	if p == nil || p.network == nil {
		return
	}
	switch kind {
	case observe.EnvKindCall:
		p.network.Hit(p.envEvents.CallCompactAllocation)
	case observe.EnvKindEnclosed:
		p.network.Hit(p.envEvents.EnclosedCompactAllocation)
	}
}

func (p *ProfileProbe) RecordEnvMapPromotion(kind observe.EnvKind) {
	if p == nil || p.network == nil {
		return
	}
	switch kind {
	case observe.EnvKindCall:
		p.network.Hit(p.envEvents.CallMapPromotion)
	case observe.EnvKindEnclosed:
		p.network.Hit(p.envEvents.EnclosedMapPromotion)
	}
}

func (p *ProfileProbe) RecordEnvLocalSet(kind observe.EnvKind, update bool) {
	if p == nil || p.network == nil {
		return
	}
	switch kind {
	case observe.EnvKindCall:
		if update {
			p.network.Hit(p.envEvents.CallLocalUpdate)
		} else {
			p.network.Hit(p.envEvents.CallLocalNew)
		}
	case observe.EnvKindEnclosed:
		if update {
			p.network.Hit(p.envEvents.EnclosedLocalUpdate)
		} else {
			p.network.Hit(p.envEvents.EnclosedLocalNew)
		}
	}
}

func (p *ProfileProbe) Measurement() engine.SemanticMeasurement {
	if p == nil {
		return engine.SemanticMeasurement{}
	}
	m := p.semantic.Measurement()
	snapshot := p.network.Snapshot()
	m.Numbers = p.numberArithmeticSnapshot(snapshot)
	m.CallNumbers = p.numberCallSnapshot(snapshot)
	m.Calls = p.callSnapshot(snapshot)
	m.Lists = p.listSnapshot(snapshot)
	m.Envs = p.envSnapshot(snapshot)
	if p.network.Depth >= observe.DepthEdges {
		m.Graph = observe.GraphMeasurement{
			Nodes:              p.registry.Nodes(),
			Edges:              append([]observe.EdgeCount(nil), snapshot.Edges...),
			Allocations:        append([]observe.EdgeAllocationCount(nil), snapshot.Allocations...),
			Timings:            append([]observe.EdgeTiming(nil), snapshot.Timings...),
			EdgeOverflow:       snapshot.EdgeOverflow,
			AllocationOverflow: snapshot.AllocationOverflow,
			TimingOverflow:     snapshot.TimingOverflow,
		}
	}
	return m
}
