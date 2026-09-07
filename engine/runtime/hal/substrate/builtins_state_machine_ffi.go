package substrate

import (
	"nezha/engine/runtime/hal"
	"nezha/engine/semantics/value"
	"sync"
)

const stateMachineProvider = "state_machine/ffi"

const (
	smOpSet = iota
	smOpAdd
	smOpCopy
	smOpAnd
	smOpOr
	smOpXor
	smOpShl
	smOpShr
	smOpLoadWord
	smOpStoreWord
	smOpAddr18FromWord
	smOpAddr18Shl
	smOpAddr18AddWord
	smOpLoadWordAddr18
	smOpAddWord
	smOpSubWord
	smOpAndWord
	smOpOrWord
	smOpXorWord
	smOpAddCarryWord
	smOpSubBorrowWord
	smOpAddOverflowWord
	smOpSubOverflowWord
	smOpStageWordAddr18
	smOpStageByteAddr18
	smOpLoadByteAddr18
)

type smOp struct {
	kind uint16
	a    uint16
	b    uint16
	c    uint16
}

type smTransition struct {
	guard      bool
	guardSlot  uint16
	guardMask  uint16
	guardValue uint16
	nextState  uint16
	ops        []smOp
	emit       bool
	emitCode   uint16
	emitSlot   uint16
}

type smProgram struct {
	stateCount      uint16
	slotCount       uint16
	addr18SlotCount uint16
	initial         uint16
	emitCap         int
	byKey           map[uint32][]smTransition
}

type smOutput struct {
	code  uint16
	value uint16
}

type smWrite struct {
	binding uint16
	address uint32
	value   uint16
	byte    bool
}

type smMachine struct {
	mu       sync.Mutex
	program  *smProgram
	state    uint16
	slots    []uint16
	addr18   []uint32
	bindings []*value.FixedStore
	outputs  []smOutput
	outHead  int
	outLen   int
	dropped  uint64
	writes   []smWrite
}

var smRegistry = struct {
	sync.Mutex
	next     uint32
	programs map[uint32]*smProgram
	machines map[uint32]*smMachine
}{next: 1, programs: map[uint32]*smProgram{}, machines: map[uint32]*smMachine{}}

func smNumber(v value.Value, name string, max uint64) (uint64, value.Value) {
	n, ok := v.(*value.Number)
	if !ok || !n.IsInt() || n.Sign() < 0 || !n.IsUint64() {
		return 0, value.NewFault("%s: expected non-negative integer", name)
	}
	x := n.Uint64Value()
	if x > max {
		return 0, value.NewFault("%s: value out of range", name)
	}
	return x, nil
}

func smSigned(v value.Value, name string, min, max int64) (int64, value.Value) {
	n, ok := v.(*value.Number)
	if !ok || !n.IsInt() || !n.IsInt64() {
		return 0, value.NewFault("%s: expected integer", name)
	}
	x := n.Int64Value()
	if x < min || x > max {
		return 0, value.NewFault("%s: value out of range", name)
	}
	return x, nil
}

func smList(v value.Value, name string) (*value.List, value.Value) {
	l, ok := v.(*value.List)
	if !ok {
		return nil, value.NewFault("%s: expected list", name)
	}
	return l, nil
}

func halStateMachineCompile(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.compile"
	if len(args) != 5 && len(args) != 6 {
		return value.NewFault("%s: want 5 arguments plus optional addr18 slot count, got %d", name, len(args))
	}
	states, f := smNumber(args[0], name, 0xffff)
	if f != nil || states == 0 {
		if f != nil {
			return f
		}
		return value.NewFault("%s: state count must be positive", name)
	}
	slots, f := smNumber(args[1], name, 0xffff)
	if f != nil {
		return f
	}
	initial, f := smNumber(args[2], name, 0xffff)
	if f != nil {
		return f
	}
	if initial >= states {
		return value.NewFault("%s: initial state out of range", name)
	}
	emitCap, f := smNumber(args[3], name, 1<<20)
	if f != nil {
		return f
	}
	transitions, f := smList(args[4], name)
	if f != nil {
		return f
	}
	addr18Slots := uint64(0)
	if len(args) == 6 {
		addr18Slots, f = smNumber(args[5], name, 0xffff)
		if f != nil {
			return f
		}
	}

	p := &smProgram{stateCount: uint16(states), slotCount: uint16(slots), addr18SlotCount: uint16(addr18Slots), initial: uint16(initial), emitCap: int(emitCap), byKey: make(map[uint32][]smTransition)}
	for ti, tv := range transitions.Elements {
		row, ff := smList(tv, name)
		if ff != nil {
			return value.NewFault("%s: transition %d must be a list", name, ti)
		}
		if len(row.Elements) != 8 && len(row.Elements) != 9 {
			return value.NewFault("%s: transition %d wants 8 legacy or 9 masked fields", name, ti)
		}
		st, ff := smNumber(row.Elements[0], name, 0xffff)
		if ff != nil {
			return ff
		}
		ev, ff := smNumber(row.Elements[1], name, 0xffff)
		if ff != nil {
			return ff
		}
		if st >= states {
			return value.NewFault("%s: transition %d state out of range", name, ti)
		}
		gs, ff := smSigned(row.Elements[2], name, -1, 0xffff)
		if ff != nil {
			return ff
		}
		guardMask := uint64(0xffff)
		guardValueIndex := 3
		nextIndex := 4
		opsIndex := 5
		emitCodeIndex := 6
		emitSlotIndex := 7
		if len(row.Elements) == 9 {
			guardMask, ff = smNumber(row.Elements[3], name, 0xffff)
			if ff != nil {
				return ff
			}
			guardValueIndex = 4
			nextIndex = 5
			opsIndex = 6
			emitCodeIndex = 7
			emitSlotIndex = 8
		}
		gv, ff := smNumber(row.Elements[guardValueIndex], name, 0xffff)
		if ff != nil {
			return ff
		}
		ns, ff := smNumber(row.Elements[nextIndex], name, 0xffff)
		if ff != nil {
			return ff
		}
		if ns >= states {
			return value.NewFault("%s: transition %d next state out of range", name, ti)
		}
		opsList, ff := smList(row.Elements[opsIndex], name)
		if ff != nil {
			return ff
		}
		ec, ff := smSigned(row.Elements[emitCodeIndex], name, -1, 0xffff)
		if ff != nil {
			return ff
		}
		es, ff := smSigned(row.Elements[emitSlotIndex], name, -1, 0xffff)
		if ff != nil {
			return ff
		}

		t := smTransition{nextState: uint16(ns)}
		if gs >= 0 {
			if gs >= int64(slots) {
				return value.NewFault("%s: transition %d guard slot out of range", name, ti)
			}
			t.guard = true
			t.guardSlot = uint16(gs)
			t.guardMask = uint16(guardMask)
			t.guardValue = uint16(gv)
		}
		if ec >= 0 {
			if es < 0 || es >= int64(slots) {
				return value.NewFault("%s: transition %d emit slot out of range", name, ti)
			}
			t.emit = true
			t.emitCode = uint16(ec)
			t.emitSlot = uint16(es)
		}
		t.ops = make([]smOp, 0, len(opsList.Elements))
		for oi, ov := range opsList.Elements {
			opRow, e := smList(ov, name)
			if e != nil {
				return e
			}
			if len(opRow.Elements) != 3 && len(opRow.Elements) != 4 {
				return value.NewFault("%s: transition %d op %d wants 3 legacy or 4 fields", name, ti, oi)
			}
			kind, e := smNumber(opRow.Elements[0], name, smOpLoadByteAddr18)
			if e != nil {
				return e
			}
			a, e := smNumber(opRow.Elements[1], name, 0xffff)
			if e != nil {
				return e
			}
			b, e := smNumber(opRow.Elements[2], name, 0xffff)
			if e != nil {
				return e
			}
			c := uint64(0)
			if len(opRow.Elements) == 4 {
				c, e = smNumber(opRow.Elements[3], name, 0xffff)
				if e != nil {
					return e
				}
			}
			switch kind {
			case smOpAddr18FromWord, smOpAddr18Shl, smOpAddr18AddWord:
				if a >= addr18Slots {
					return value.NewFault("%s: transition %d op %d addr18 slot out of range", name, ti, oi)
				}
				if (kind == smOpAddr18FromWord || kind == smOpAddr18AddWord) && c >= slots {
					return value.NewFault("%s: transition %d op %d word source slot out of range", name, ti, oi)
				}
				if kind == smOpAddr18Shl && b > 31 {
					return value.NewFault("%s: transition %d op %d addr18 shift out of range", name, ti, oi)
				}
			case smOpLoadWordAddr18, smOpLoadByteAddr18:
				if a >= slots {
					return value.NewFault("%s: transition %d op %d destination slot out of range", name, ti, oi)
				}
				if c >= addr18Slots {
					return value.NewFault("%s: transition %d op %d addr18 index slot out of range", name, ti, oi)
				}
			case smOpStageWordAddr18, smOpStageByteAddr18:
				if a >= slots {
					return value.NewFault("%s: transition %d op %d staged value slot out of range", name, ti, oi)
				}
				if c >= addr18Slots {
					return value.NewFault("%s: transition %d op %d staged addr18 slot out of range", name, ti, oi)
				}
			case smOpAddWord, smOpSubWord, smOpAndWord, smOpOrWord, smOpXorWord, smOpAddCarryWord, smOpSubBorrowWord, smOpAddOverflowWord, smOpSubOverflowWord:
				if a >= slots || b >= slots || c >= slots {
					return value.NewFault("%s: transition %d op %d word ALU slot out of range", name, ti, oi)
				}
			default:
				if a >= slots {
					return value.NewFault("%s: transition %d op %d slot out of range", name, ti, oi)
				}
				if (kind == smOpCopy || kind == smOpLoadWord || kind == smOpStoreWord) && c >= slots {
					return value.NewFault("%s: transition %d op %d source/index slot out of range", name, ti, oi)
				}
			}
			t.ops = append(t.ops, smOp{kind: uint16(kind), a: uint16(a), b: uint16(b), c: uint16(c)})
		}
		key := uint32(st)<<16 | uint32(ev)
		p.byKey[key] = append(p.byKey[key], t)
	}
	smRegistry.Lock()
	id := smRegistry.next
	smRegistry.next++
	smRegistry.programs[id] = p
	smRegistry.Unlock()
	return &value.Opaque{Provider: stateMachineProvider, Kind: "program", Bits: id}
}

func smProgramArg(v value.Value, name string) (*smProgram, value.Value) {
	o, ok := v.(*value.Opaque)
	if !ok || o.Provider != stateMachineProvider || o.Kind != "program" {
		return nil, value.NewFault("%s: expected state-machine program", name)
	}
	smRegistry.Lock()
	p := smRegistry.programs[o.Bits]
	smRegistry.Unlock()
	if p == nil {
		return nil, value.NewFault("%s: unknown state-machine program", name)
	}
	return p, nil
}
func smMachineArg(v value.Value, name string) (*smMachine, value.Value) {
	o, ok := v.(*value.Opaque)
	if !ok || o.Provider != stateMachineProvider || o.Kind != "machine" {
		return nil, value.NewFault("%s: expected state-machine instance", name)
	}
	smRegistry.Lock()
	m := smRegistry.machines[o.Bits]
	smRegistry.Unlock()
	if m == nil {
		return nil, value.NewFault("%s: unknown state-machine instance", name)
	}
	return m, nil
}

func halStateMachineNew(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.new"
	if len(args) != 2 && len(args) != 3 {
		return value.NewFault("%s: want 2 arguments plus optional bindings", name)
	}
	p, f := smProgramArg(args[0], name)
	if f != nil {
		return f
	}
	init, f := smList(args[1], name)
	if f != nil {
		return f
	}
	if len(init.Elements) != int(p.slotCount) {
		return value.NewFault("%s: initial slot count mismatch", name)
	}
	slots := make([]uint16, len(init.Elements))
	for i, v := range init.Elements {
		x, ff := smNumber(v, name, 0xffff)
		if ff != nil {
			return ff
		}
		slots[i] = uint16(x)
	}
	bindings := []*value.FixedStore{}
	if len(args) == 3 {
		bl, ff := smList(args[2], name)
		if ff != nil {
			return ff
		}
		bindings = make([]*value.FixedStore, len(bl.Elements))
		for i, bv := range bl.Elements {
			bs, ok := bv.(*value.FixedStore)
			if !ok || bs.Kind != "word" {
				return value.NewFault("%s: binding %d must be a word store", name, i)
			}
			bindings[i] = bs
		}
	}
	m := &smMachine{program: p, state: p.initial, slots: slots, addr18: make([]uint32, int(p.addr18SlotCount)), bindings: bindings}
	if p.emitCap > 0 {
		m.outputs = make([]smOutput, p.emitCap)
	}
	smRegistry.Lock()
	id := smRegistry.next
	smRegistry.next++
	smRegistry.machines[id] = m
	smRegistry.Unlock()
	return &value.Opaque{Provider: stateMachineProvider, Kind: "machine", Bits: id}
}

func (m *smMachine) publish(o smOutput) {
	if len(m.outputs) == 0 {
		m.dropped++
		return
	}
	if m.outLen == len(m.outputs) {
		m.dropped++
		return
	}
	idx := (m.outHead + m.outLen) % len(m.outputs)
	m.outputs[idx] = o
	m.outLen++
}

func (m *smMachine) effectiveWord(binding uint16, address uint32) (uint16, string) {
	if int(binding) >= len(m.bindings) {
		return 0, "word binding out of range"
	}
	if address&1 != 0 {
		return 0, "word address is odd"
	}
	idx := int(address >> 1)
	if idx < 0 || idx >= m.bindings[binding].FixedLen() {
		return 0, "word address out of range"
	}
	word := uint16(m.bindings[binding].FixedGet(idx))
	for _, w := range m.writes {
		if w.binding != binding || (w.address>>1) != (address>>1) {
			continue
		}
		if !w.byte {
			word = w.value
			continue
		}
		if w.address&1 == 0 {
			word = (word & 0xff00) | (w.value & 0x00ff)
		} else {
			word = (word & 0x00ff) | ((w.value & 0x00ff) << 8)
		}
	}
	return word, ""
}

func (m *smMachine) effectiveByte(binding uint16, address uint32) (uint16, string) {
	word, fault := m.effectiveWord(binding, address&^1)
	if fault != "" {
		return 0, fault
	}
	if address&1 == 0 {
		return word & 0x00ff, ""
	}
	return (word >> 8) & 0x00ff, ""
}

func (m *smMachine) stageWrite(binding uint16, address uint32, v uint16, byteWrite bool) string {
	if int(binding) >= len(m.bindings) {
		return "staged write binding out of range"
	}
	if !byteWrite && address&1 != 0 {
		return "staged word write odd address"
	}
	idx := int(address >> 1)
	if idx < 0 || idx >= m.bindings[binding].FixedLen() {
		return "staged write address out of range"
	}
	m.writes = append(m.writes, smWrite{binding: binding, address: address, value: v, byte: byteWrite})
	return ""
}

func (m *smMachine) commitWrites() string {
	// Every write is range-checked when staged, and FixedStore writes cannot fail.
	// Applying the journal in order therefore makes commit a single explicit
	// architectural boundary rather than a sequence of fallible guest cycles.
	for _, w := range m.writes {
		idx := int(w.address >> 1)
		if !w.byte {
			m.bindings[w.binding].FixedSet(idx, uint32(w.value))
			continue
		}
		word := uint16(m.bindings[w.binding].FixedGet(idx))
		if w.address&1 == 0 {
			word = (word & 0xff00) | (w.value & 0x00ff)
		} else {
			word = (word & 0x00ff) | ((w.value & 0x00ff) << 8)
		}
		m.bindings[w.binding].FixedSet(idx, uint32(word))
	}
	m.writes = m.writes[:0]
	return ""
}

func (m *smMachine) step(event uint16) (bool, string) {
	ts := m.program.byKey[uint32(m.state)<<16|uint32(event)]
	for _, t := range ts {
		if t.guard && (m.slots[t.guardSlot]&t.guardMask) != t.guardValue {
			continue
		}
		for _, op := range t.ops {
			switch op.kind {
			case smOpSet:
				m.slots[op.a] = op.b
			case smOpAdd:
				m.slots[op.a] += op.b
			case smOpCopy:
				m.slots[op.a] = m.slots[op.c]
			case smOpAnd:
				m.slots[op.a] &= op.b
			case smOpOr:
				m.slots[op.a] |= op.b
			case smOpXor:
				m.slots[op.a] ^= op.b
			case smOpShl:
				m.slots[op.a] <<= uint(op.b & 15)
			case smOpShr:
				m.slots[op.a] >>= uint(op.b & 15)
			case smOpLoadWord:
				address := uint32(m.slots[op.c]) << 1
				word, fault := m.effectiveWord(op.b, address)
				if fault != "" {
					return false, "word-load " + fault
				}
				m.slots[op.a] = word
			case smOpStoreWord:
				if int(op.b) >= len(m.bindings) {
					return false, "word-store binding out of range"
				}
				idx := int(m.slots[op.c])
				if idx < 0 || idx >= m.bindings[op.b].FixedLen() {
					return false, "word-store index out of range"
				}
				m.bindings[op.b].FixedSet(idx, uint32(m.slots[op.a]))
			case smOpAddr18FromWord:
				m.addr18[op.a] = uint32(m.slots[op.c])
			case smOpAddr18Shl:
				m.addr18[op.a] = (m.addr18[op.a] << uint(op.b)) & 0x3ffff
			case smOpAddr18AddWord:
				m.addr18[op.a] = (m.addr18[op.a] + uint32(m.slots[op.c])) & 0x3ffff
			case smOpLoadWordAddr18:
				address := m.addr18[op.c]
				word, fault := m.effectiveWord(op.b, address)
				if fault != "" {
					return false, "addr18 word-load " + fault
				}
				m.slots[op.a] = word
			case smOpAddWord:
				m.slots[op.a] = m.slots[op.b] + m.slots[op.c]
			case smOpSubWord:
				m.slots[op.a] = m.slots[op.b] - m.slots[op.c]
			case smOpAndWord:
				m.slots[op.a] = m.slots[op.b] & m.slots[op.c]
			case smOpOrWord:
				m.slots[op.a] = m.slots[op.b] | m.slots[op.c]
			case smOpXorWord:
				m.slots[op.a] = m.slots[op.b] ^ m.slots[op.c]
			case smOpAddCarryWord:
				if uint32(m.slots[op.b])+uint32(m.slots[op.c]) > 0xffff {
					m.slots[op.a] = 1
				} else {
					m.slots[op.a] = 0
				}
			case smOpSubBorrowWord:
				if m.slots[op.b] < m.slots[op.c] {
					m.slots[op.a] = 1
				} else {
					m.slots[op.a] = 0
				}
			case smOpAddOverflowWord:
				left := int32(int16(m.slots[op.b]))
				right := int32(int16(m.slots[op.c]))
				sum := left + right
				if sum < -32768 || sum > 32767 {
					m.slots[op.a] = 1
				} else {
					m.slots[op.a] = 0
				}
			case smOpSubOverflowWord:
				left := int32(int16(m.slots[op.b]))
				right := int32(int16(m.slots[op.c]))
				difference := left - right
				if difference < -32768 || difference > 32767 {
					m.slots[op.a] = 1
				} else {
					m.slots[op.a] = 0
				}
			case smOpStageWordAddr18:
				if fault := m.stageWrite(op.b, m.addr18[op.c], m.slots[op.a], false); fault != "" {
					return false, fault
				}
			case smOpStageByteAddr18:
				if fault := m.stageWrite(op.b, m.addr18[op.c], m.slots[op.a], true); fault != "" {
					return false, fault
				}
			case smOpLoadByteAddr18:
				byteValue, fault := m.effectiveByte(op.b, m.addr18[op.c])
				if fault != "" {
					return false, "addr18 byte-load " + fault
				}
				m.slots[op.a] = byteValue
			}
		}
		m.state = t.nextState
		if t.emit {
			m.publish(smOutput{code: t.emitCode, value: m.slots[t.emitSlot]})
		}
		return true, ""
	}
	return false, ""
}

func halStateMachineStep(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.step"
	if len(args) != 2 {
		return value.NewFault("%s: want 2 arguments", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	ev, f := smNumber(args[1], name, 0xffff)
	if f != nil {
		return f
	}
	m.mu.Lock()
	ok, runtimeFault := m.step(uint16(ev))
	m.mu.Unlock()
	if runtimeFault != "" {
		return value.NewFault("%s: %s", name, runtimeFault)
	}
	if !ok {
		return value.NewFault("%s: no admitted transition", name)
	}
	return value.TRUE
}

func halStateMachineRun(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.run"
	if len(args) != 3 {
		return value.NewFault("%s: want 3 arguments", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	ev, f := smNumber(args[1], name, 0xffff)
	if f != nil {
		return f
	}
	count, f := smNumber(args[2], name, 1<<62)
	if f != nil {
		return f
	}
	m.mu.Lock()
	var done uint64
	for done < count {
		ok, runtimeFault := m.step(uint16(ev))
		if runtimeFault != "" {
			m.mu.Unlock()
			return value.NewFault("%s: %s after %d steps", name, runtimeFault, done)
		}
		if !ok {
			m.mu.Unlock()
			return value.NewFault("%s: no admitted transition after %d steps", name, done)
		}
		done++
	}
	m.mu.Unlock()
	if done > uint64(^uint64(0)>>1) {
		return value.NewFault("%s: count exceeds Nezha integer boundary", name)
	}
	return value.NewNumber(int64(done), 1)
}

func halStateMachineRunUntil(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.run_until"
	if len(args) != 4 {
		return value.NewFault("%s: want 4 arguments", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	ev, f := smNumber(args[1], name, 0xffff)
	if f != nil {
		return f
	}
	count, f := smNumber(args[2], name, 1<<62)
	if f != nil {
		return f
	}
	stop, f := smNumber(args[3], name, 0xffff)
	if f != nil {
		return f
	}
	if stop >= uint64(m.program.stateCount) {
		return value.NewFault("%s: stop state out of range", name)
	}
	m.mu.Lock()
	var done uint64
	for done < count && m.state != uint16(stop) {
		ok, runtimeFault := m.step(uint16(ev))
		if runtimeFault != "" {
			m.mu.Unlock()
			return value.NewFault("%s: %s after %d steps", name, runtimeFault, done)
		}
		if !ok {
			m.mu.Unlock()
			return value.NewFault("%s: no admitted transition after %d steps", name, done)
		}
		done++
	}
	m.mu.Unlock()
	if done > uint64(^uint64(0)>>1) {
		return value.NewFault("%s: count exceeds Nezha integer boundary", name)
	}
	return value.NewNumber(int64(done), 1)
}

func halStateMachineState(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.state"
	if len(args) != 1 {
		return value.NewFault("%s: want 1 argument", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	m.mu.Lock()
	st := m.state
	m.mu.Unlock()
	return value.NewNumber(int64(st), 1)
}
func halStateMachineSlot(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.slot"
	if len(args) != 2 {
		return value.NewFault("%s: want 2 arguments", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	idx, f := smNumber(args[1], name, 0xffff)
	if f != nil {
		return f
	}
	if idx >= uint64(m.program.slotCount) {
		return value.NewFault("%s: slot out of range", name)
	}
	m.mu.Lock()
	v := m.slots[idx]
	m.mu.Unlock()
	return value.NewNumber(int64(v), 1)
}
func halStateMachineOutputs(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.outputs"
	if len(args) != 1 {
		return value.NewFault("%s: want 1 argument", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	m.mu.Lock()
	out := make([]value.Value, m.outLen)
	for i := 0; i < m.outLen; i++ {
		o := m.outputs[(m.outHead+i)%len(m.outputs)]
		out[i] = &value.List{Elements: []value.Value{value.NewNumber(int64(o.code), 1), value.NewNumber(int64(o.value), 1)}}
	}
	m.mu.Unlock()
	return &value.List{Elements: out}
}
func halStateMachineDropped(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.dropped"
	if len(args) != 1 {
		return value.NewFault("%s: want 1 argument", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	m.mu.Lock()
	d := m.dropped
	m.mu.Unlock()
	if d > uint64(^uint64(0)>>1) {
		return value.NewFault("%s: value exceeds Nezha integer boundary", name)
	}
	return value.NewNumber(int64(d), 1)
}

func halStateMachineStagedWrites(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.staged_writes"
	if len(args) != 1 {
		return value.NewFault("%s: want 1 argument", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	m.mu.Lock()
	count := len(m.writes)
	m.mu.Unlock()
	return value.NewNumber(int64(count), 1)
}

func halStateMachineCommitWrites(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.commit_writes"
	if len(args) != 1 {
		return value.NewFault("%s: want 1 argument", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	m.mu.Lock()
	fault := m.commitWrites()
	m.mu.Unlock()
	if fault != "" {
		return value.NewFault("%s: %s", name, fault)
	}
	return value.TRUE
}

func halStateMachineDiscardWrites(args []value.Value, ctx *hal.EvalContext) value.Value {
	const name = "state_machine/ffi.discard_writes"
	if len(args) != 1 {
		return value.NewFault("%s: want 1 argument", name)
	}
	m, f := smMachineArg(args[0], name)
	if f != nil {
		return f
	}
	m.mu.Lock()
	m.writes = m.writes[:0]
	m.mu.Unlock()
	return value.TRUE
}
