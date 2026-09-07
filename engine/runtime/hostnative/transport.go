// Package hostnative contains generic mechanics for host-native implementations
// of Nezha semantic contracts. It deliberately operates below the ordinary Nezha
// callable/value ABI: hot interactions use fixed-width host values and pre-resolved
// bindings, crossing back into interpreted Nezha only when explicitly required.
package hostnative

import "fmt"

// Word is the fixed-width carrier used between admitted host-native implementations.
// Its representation is implementation-private; ordinary Nezha numeric semantics do
// not constrain or reinterpret these bits.
type Word uint64

// EndpointID is assigned once when an endpoint is registered. Runtime routing
// uses the numeric id directly; names and other descriptive metadata are not
// part of the transfer path.
type EndpointID uint32

// DirectEndpoint is the admitted fixed-signature endpoint ABI. Offset is
// precomputed by registration so the endpoint does not need to decode the
// machine-global address again. Packet and response remain packed machine bits.
type DirectEndpoint func(offset uint32, packet Word) Word

type endpointBinding struct {
	direct DirectEndpoint
}

// Outcome is the result of one transport operation. Direct endpoints return a
// response packet immediately. Interpreted endpoints set Yield and preserve the
// same packed packet plus the already-resolved endpoint id and local offset for
// the wrapper that must cross into Nezha.
type Outcome struct {
	Packet   Word
	Endpoint EndpointID
	Offset   uint32
	Yield    bool
}

// Transport resolves a fixed-width selector space to admitted endpoint
// bindings. Each selector has one precomputed route word:
//
//	high 32 bits: endpoint id
//	low  32 bits: endpoint-local offset
//
// Registration therefore pays range/overlap work once. A steady-state
// transfer performs one table lookup, one binding lookup, and either one direct
// fixed-signature call or one explicit interpreted yield.
type Transport struct {
	routes   []uint64
	bindings []endpointBinding
}

// NewTransport constructs a selector-space transport. miss is endpoint zero and
// handles every selector that has not been explicitly registered. selectorBits is
// deliberately configurable so the mechanism is not tied to one machine bus.
func NewTransport(selectorBits uint8, miss DirectEndpoint) (*Transport, error) {
	if selectorBits == 0 || selectorBits > 24 {
		return nil, fmt.Errorf("host-native transport: selector width %d out of range", selectorBits)
	}
	if miss == nil {
		return nil, fmt.Errorf("host-native transport: miss endpoint is required")
	}
	size := 1 << selectorBits
	t := &Transport{
		routes:   make([]uint64, size),
		bindings: []endpointBinding{{direct: miss}},
	}
	// Endpoint zero is the miss binding. Preserve the original selector as its
	// local offset so the miss implementation can report/encode the exact failing
	// selector without unpacking it from the packet.
	for selector := range t.routes {
		t.routes[selector] = uint64(uint32(selector))
	}
	return t, nil
}

// RegisterDirect admits a direct host-native implementation for the inclusive selector range.
// The endpoint receives selector-start as its local offset. Registration rejects
// overlap so runtime dispatch never needs an ordered decision tree. A combined
// host-native implementation may choose a selector wider than the architectural address—for
// example address plus cycle bits—so operation selection can also be resolved
// at registration time when useful.
func (t *Transport) RegisterDirect(start, end uint32, endpoint DirectEndpoint) (EndpointID, error) {
	if endpoint == nil {
		return 0, fmt.Errorf("host-native transport: direct endpoint is required")
	}
	return t.register(start, end, endpoint)
}

// ReplaceDirect replaces an already-registered inclusive selector range with a
// direct endpoint. This is composition-time refinement: it is intended for a
// host-native implementation that can realize a strict subset of a broader
// interpreted binding. Every selector in the range must already be registered
// and still interpreted, so replacement cannot silently shadow another direct
// implementation. Runtime dispatch is unchanged: routes are rewritten once and
// Transfer still performs the same single route lookup and binding call.
func (t *Transport) ReplaceDirect(start, end uint32, endpoint DirectEndpoint) (EndpointID, error) {
	if endpoint == nil {
		return 0, fmt.Errorf("host-native transport: direct endpoint is required")
	}
	if start > end || uint64(end) >= uint64(len(t.routes)) {
		return 0, fmt.Errorf("host-native transport: invalid range %d..%d", start, end)
	}
	for selector := start; selector <= end; selector++ {
		id := EndpointID(t.routes[selector] >> 32)
		if id == 0 {
			return 0, fmt.Errorf("host-native transport: selector %d is not registered", selector)
		}
		if t.bindings[id].direct != nil {
			return 0, fmt.Errorf("host-native transport: selector %d already has direct endpoint", selector)
		}
		if selector == ^uint32(0) {
			break
		}
	}
	if len(t.bindings) == int(^EndpointID(0)) {
		return 0, fmt.Errorf("host-native transport: endpoint id space exhausted")
	}
	id := EndpointID(len(t.bindings))
	t.bindings = append(t.bindings, endpointBinding{direct: endpoint})
	for selector := start; selector <= end; selector++ {
		offset := selector - start
		t.routes[selector] = uint64(id)<<32 | uint64(offset)
		if selector == ^uint32(0) {
			break
		}
	}
	return id, nil
}

// RegisterInterpreted admits an endpoint whose semantic transition is still
// implemented in Nezha. Runtime transport resolves it identically to a direct
// endpoint, but returns an explicit yield rather than entering the evaluator
// through the hot transport call.
func (t *Transport) RegisterInterpreted(start, end uint32) (EndpointID, error) {
	return t.register(start, end, nil)
}

func (t *Transport) register(start, end uint32, direct DirectEndpoint) (EndpointID, error) {
	if start > end || uint64(end) >= uint64(len(t.routes)) {
		return 0, fmt.Errorf("host-native transport: invalid range %d..%d", start, end)
	}
	for address := start; address <= end; address++ {
		if EndpointID(t.routes[address]>>32) != 0 {
			return 0, fmt.Errorf("host-native transport: selector %d already registered", address)
		}
		if address == ^uint32(0) {
			break
		}
	}
	if len(t.bindings) == int(^EndpointID(0)) {
		return 0, fmt.Errorf("host-native transport: endpoint id space exhausted")
	}
	id := EndpointID(len(t.bindings))
	t.bindings = append(t.bindings, endpointBinding{direct: direct})
	for address := start; address <= end; address++ {
		offset := address - start
		t.routes[address] = uint64(id)<<32 | uint64(offset)
		if address == ^uint32(0) {
			break
		}
	}
	return id, nil
}

// Unregister removes every route owned by id and restores the miss binding.
// Topology changes are composition-time/control-plane work, so this deliberately
// scans the selector table rather than adding steady-state indirection. Endpoint
// ids are not reused; stale admitted ids therefore cannot silently name a new
// endpoint.
func (t *Transport) Unregister(id EndpointID) error {
	if id == 0 || int(id) >= len(t.bindings) {
		return fmt.Errorf("host-native transport: invalid endpoint id %d", id)
	}
	found := false
	for selector, route := range t.routes {
		if EndpointID(route>>32) == id {
			t.routes[selector] = uint64(uint32(selector))
			found = true
		}
	}
	if !found {
		return fmt.Errorf("host-native transport: endpoint id %d is not registered", id)
	}
	t.bindings[id] = endpointBinding{}
	return nil
}

// Transfer executes the canonical admitted transport mechanic. The caller owns
// selector-width admission; combined host-native implementations already operate on
// fixed-width selectors and therefore do not pay a redundant range/type check
// on every transfer.
func (t *Transport) Transfer(selector uint32, packet Word) Outcome {
	route := t.routes[selector]
	id := EndpointID(route >> 32)
	offset := uint32(route)
	binding := &t.bindings[id]
	if binding.direct != nil {
		return Outcome{
			Packet:   binding.direct(offset, packet),
			Endpoint: id,
			Offset:   offset,
		}
	}
	return Outcome{Packet: packet, Endpoint: id, Offset: offset, Yield: true}
}
