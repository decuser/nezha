package substrate

import (
	"aiki/engine/semantics/value"
	"testing"
)

func smListV(xs ...value.Value) value.Value { return &value.List{Elements: xs} }

func smProgramValue(t *testing.T, emitCap int64) value.Value {
	t.Helper()
	transitions := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), smListV(smListV(num(1), num(0), num(1))), num(10), num(0)),
		smListV(num(1), num(1), num(-1), num(0), num(0), smListV(smListV(num(1), num(0), num(1))), num(11), num(0)),
	)
	p := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(emitCap), transitions}, nil)
	if value.IsFault(p) {
		t.Fatal(p.Inspect())
	}
	return p
}

func TestStateMachineIntrinsicStepAndBoundedEmit(t *testing.T) {
	p := smProgramValue(t, 2)
	m := halStateMachineNew([]value.Value{p, smListV(num(0))}, nil)
	if value.IsFault(m) {
		t.Fatal(m.Inspect())
	}
	if got := halStateMachineRun([]value.Value{m, num(1), num(5)}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if got := halStateMachineState([]value.Value{m}, nil).(*value.Number).Int64Value(); got != 1 {
		t.Fatalf("state=%d", got)
	}
	if got := halStateMachineSlot([]value.Value{m, num(0)}, nil).(*value.Number).Int64Value(); got != 5 {
		t.Fatalf("slot=%d", got)
	}
	if got := halStateMachineDropped([]value.Value{m}, nil).(*value.Number).Int64Value(); got != 3 {
		t.Fatalf("dropped=%d", got)
	}
	out := halStateMachineOutputs([]value.Value{m}, nil).(*value.List)
	if len(out.Elements) != 2 {
		t.Fatalf("outputs=%d", len(out.Elements))
	}
}

func TestStateMachineIntrinsicGuardAndWrap(t *testing.T) {
	transitions := smListV(smListV(num(0), num(7), num(0), num(65535), num(1), smListV(smListV(num(1), num(0), num(1))), num(-1), num(-1)))
	p := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), transitions}, nil)
	if value.IsFault(p) {
		t.Fatal(p.Inspect())
	}
	m := halStateMachineNew([]value.Value{p, smListV(num(65535))}, nil)
	if value.IsFault(m) {
		t.Fatal(m.Inspect())
	}
	if got := halStateMachineStep([]value.Value{m, num(7)}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if got := halStateMachineSlot([]value.Value{m, num(0)}, nil).(*value.Number).Int64Value(); got != 0 {
		t.Fatalf("wrap=%d", got)
	}
}

func TestStateMachineIntrinsicRejectsInvalidProgram(t *testing.T) {
	bad := smListV(smListV(num(0), num(1), num(-1), num(0), num(2), smListV(), num(-1), num(-1)))
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), bad}, nil).(*value.Fault); !ok {
		t.Fatal("invalid next state admitted")
	}
}

func TestStateMachineIntrinsicMaskedGuardBoundWordLoadAndRunUntil(t *testing.T) {
	backing := &value.FixedStore{Kind: "word", Words: make([]uint16, 4)}
	backing.FixedSet(2, 0x1234)
	ops := smListV(
		smListV(num(7), num(1), num(1), num(0)), // slot1 >>= 1
		smListV(num(8), num(2), num(0), num(1)), // slot2 <- binding0[slot1]
		smListV(num(3), num(2), num(255), num(0)),
		smListV(num(4), num(2), num(256), num(0)),
	)
	transitions := smListV(
		smListV(num(0), num(1), num(0), num(240), num(160), num(1), ops, num(-1), num(-1)),
		smListV(num(1), num(1), num(-1), num(0), num(0), num(2), smListV(), num(-1), num(-1)),
	)
	p := halStateMachineCompile([]value.Value{num(3), num(3), num(0), num(0), transitions}, nil)
	if value.IsFault(p) {
		t.Fatal(p.Inspect())
	}
	m := halStateMachineNew([]value.Value{p, smListV(num(175), num(4), num(0)), smListV(backing)}, nil)
	if value.IsFault(m) {
		t.Fatal(m.Inspect())
	}
	got := halStateMachineRunUntil([]value.Value{m, num(1), num(10), num(2)}, nil)
	if value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if steps := got.(*value.Number).Int64Value(); steps != 2 {
		t.Fatalf("steps=%d", steps)
	}
	if st := halStateMachineState([]value.Value{m}, nil).(*value.Number).Int64Value(); st != 2 {
		t.Fatalf("state=%d", st)
	}
	if v := halStateMachineSlot([]value.Value{m, num(2)}, nil).(*value.Number).Int64Value(); v != 0x134 {
		t.Fatalf("loaded/transformed slot=%#x", v)
	}
}

func TestStateMachineIntrinsicAddr18WordLoadAbove16BitAddress(t *testing.T) {
	backing := &value.FixedStore{Kind: "word", Words: make([]uint16, 32769)}
	backing.FixedSet(32768, 0x5200)
	ops := smListV(
		smListV(num(smOpAddr18FromWord), num(0), num(0), num(0)),
		smListV(num(smOpAddr18Shl), num(0), num(6), num(0)),
		smListV(num(smOpAddr18AddWord), num(0), num(0), num(1)),
		smListV(num(smOpLoadWordAddr18), num(2), num(0), num(0)),
		smListV(num(smOpAdd), num(3), num(1)),
	)
	transitions := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), ops, num(-1), num(-1)),
	)
	p := halStateMachineCompile([]value.Value{num(2), num(4), num(0), num(0), transitions, num(1)}, nil)
	if value.IsFault(p) {
		t.Fatal(p.Inspect())
	}
	m := halStateMachineNew([]value.Value{p, smListV(num(1024), num(0), num(0), num(65535)), smListV(backing)}, nil)
	if value.IsFault(m) {
		t.Fatal(m.Inspect())
	}
	if got := halStateMachineStep([]value.Value{m, num(1)}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if got := halStateMachineSlot([]value.Value{m, num(2)}, nil).(*value.Number).Int64Value(); got != 0x5200 {
		t.Fatalf("addr18 load=%#x", got)
	}
	if got := halStateMachineSlot([]value.Value{m, num(3)}, nil).(*value.Number).Int64Value(); got != 0 {
		t.Fatalf("word slot widened: %#x", got)
	}
}

func TestStateMachineIntrinsicRejectsInvalidAddr18Operands(t *testing.T) {
	badWide := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), smListV(
			smListV(num(smOpAddr18FromWord), num(1), num(0), num(0)),
		), num(-1), num(-1)),
	)
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), badWide, num(1)}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range addr18 slot admitted")
	}

	badWordSource := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), smListV(
			smListV(num(smOpAddr18AddWord), num(0), num(0), num(1)),
		), num(-1), num(-1)),
	)
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), badWordSource, num(1)}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range word source admitted for addr18 operation")
	}

	badShift := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), smListV(
			smListV(num(smOpAddr18Shl), num(0), num(32), num(0)),
		), num(-1), num(-1)),
	)
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), badShift, num(1)}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range addr18 shift admitted")
	}

	badLoad := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), smListV(
			smListV(num(smOpLoadWordAddr18), num(0), num(0), num(1)),
		), num(-1), num(-1)),
	)
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), badLoad, num(1)}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range addr18 load slot admitted")
	}
}

func TestStateMachineIntrinsicWordALUAndPredicates(t *testing.T) {
	cases := []struct {
		name        string
		left, right int64
		want        [9]int64
	}{
		{"small", 1, 2, [9]int64{3, 0xffff, 0, 3, 3, 0, 1, 0, 0}},
		{"unsigned-wrap", 0xffff, 1, [9]int64{0, 0xfffe, 1, 0xffff, 0xfffe, 1, 0, 0, 0}},
		{"add-overflow", 0x7fff, 1, [9]int64{0x8000, 0x7ffe, 1, 0x7fff, 0x7ffe, 0, 0, 1, 0}},
		{"sub-overflow", 0x8000, 1, [9]int64{0x8001, 0x7fff, 0, 0x8001, 0x8001, 0, 0, 0, 1}},
		{"signed-boundary", 0x8000, 0xffff, [9]int64{0x7fff, 0x8001, 0x8000, 0xffff, 0x7fff, 1, 1, 1, 0}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ops := smListV(
				smListV(num(smOpAddWord), num(2), num(0), num(1)),
				smListV(num(smOpSubWord), num(3), num(0), num(1)),
				smListV(num(smOpAndWord), num(4), num(0), num(1)),
				smListV(num(smOpOrWord), num(5), num(0), num(1)),
				smListV(num(smOpXorWord), num(6), num(0), num(1)),
				smListV(num(smOpAddCarryWord), num(7), num(0), num(1)),
				smListV(num(smOpSubBorrowWord), num(8), num(0), num(1)),
				smListV(num(smOpAddOverflowWord), num(9), num(0), num(1)),
				smListV(num(smOpSubOverflowWord), num(10), num(0), num(1)),
			)
			transitions := smListV(
				smListV(num(0), num(1), num(-1), num(0), num(0), num(1), ops, num(-1), num(-1)),
			)
			p := halStateMachineCompile([]value.Value{num(2), num(11), num(0), num(0), transitions}, nil)
			if value.IsFault(p) {
				t.Fatal(p.Inspect())
			}
			m := halStateMachineNew([]value.Value{p, smListV(
				num(tc.left), num(tc.right), num(0), num(0), num(0), num(0),
				num(0), num(0), num(0), num(0), num(0),
			)}, nil)
			if value.IsFault(m) {
				t.Fatal(m.Inspect())
			}
			if got := halStateMachineStep([]value.Value{m, num(1)}, nil); value.IsFault(got) {
				t.Fatal(got.Inspect())
			}
			for i, expected := range tc.want {
				got := halStateMachineSlot([]value.Value{m, num(int64(i + 2))}, nil).(*value.Number).Int64Value()
				if got != expected {
					t.Fatalf("ALU slot %d=%#x want %#x", i+2, got, expected)
				}
			}
		})
	}
}

func TestStateMachineIntrinsicRejectsInvalidWordALUSlots(t *testing.T) {
	bad := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(1), smListV(
			smListV(num(smOpSubWord), num(0), num(0), num(2)),
		), num(-1), num(-1)),
	)
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(2), num(0), num(0), bad}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range word ALU source admitted")
	}
}

func TestStateMachineOperationNumberContract(t *testing.T) {
	got := []int{
		smOpSet, smOpAdd, smOpCopy, smOpAnd, smOpOr, smOpXor,
		smOpShl, smOpShr, smOpLoadWord, smOpStoreWord,
		smOpAddr18FromWord, smOpAddr18Shl, smOpAddr18AddWord, smOpLoadWordAddr18,
		smOpAddWord, smOpSubWord, smOpAndWord, smOpOrWord, smOpXorWord,
		smOpAddCarryWord, smOpSubBorrowWord, smOpAddOverflowWord, smOpSubOverflowWord,
		smOpStageWordAddr18, smOpStageByteAddr18, smOpLoadByteAddr18,
	}
	for i, actual := range got {
		if actual != i {
			t.Fatalf("state-machine op %d has id %d", i, actual)
		}
	}
}

func TestStateMachineTransactionalBoundStore(t *testing.T) {
	backing := &value.FixedStore{Kind: "word", Words: make([]uint16, 4)}
	backing.FixedSet(1, 0x1234)
	ops1 := smListV(
		smListV(num(smOpAddr18FromWord), num(0), num(0), num(0)),
		smListV(num(smOpStageWordAddr18), num(1), num(0), num(0)),
		smListV(num(smOpLoadWordAddr18), num(2), num(0), num(0)),
	)
	ops2 := smListV(
		smListV(num(smOpSet), num(0), num(3), num(0)),
		smListV(num(smOpAddr18FromWord), num(0), num(0), num(0)),
		smListV(num(smOpStageByteAddr18), num(3), num(0), num(0)),
		smListV(num(smOpLoadByteAddr18), num(4), num(0), num(0)),
		smListV(num(smOpSet), num(0), num(2), num(0)),
		smListV(num(smOpAddr18FromWord), num(0), num(0), num(0)),
		smListV(num(smOpLoadWordAddr18), num(2), num(0), num(0)),
	)
	transitions := smListV(
		smListV(num(0), num(1), num(-1), num(0), num(0), num(1), ops1, num(-1), num(-1)),
		smListV(num(1), num(1), num(-1), num(0), num(0), num(2), ops2, num(-1), num(-1)),
	)
	p := halStateMachineCompile([]value.Value{num(3), num(5), num(0), num(0), transitions, num(1)}, nil)
	if value.IsFault(p) {
		t.Fatal(p.Inspect())
	}
	m := halStateMachineNew([]value.Value{p, smListV(num(2), num(0xabcd), num(0), num(0xee), num(0)), smListV(backing)}, nil)
	if value.IsFault(m) {
		t.Fatal(m.Inspect())
	}
	if got := halStateMachineStep([]value.Value{m, num(1)}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if backing.FixedGet(1) != 0x1234 {
		t.Fatalf("staged word escaped before commit: %#x", backing.FixedGet(1))
	}
	if got := halStateMachineSlot([]value.Value{m, num(2)}, nil).(*value.Number).Int64Value(); got != 0xabcd {
		t.Fatalf("overlay word=%#x", got)
	}
	if got := halStateMachineStep([]value.Value{m, num(1)}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if got := halStateMachineSlot([]value.Value{m, num(4)}, nil).(*value.Number).Int64Value(); got != 0xee {
		t.Fatalf("overlay byte=%#x", got)
	}
	if got := halStateMachineSlot([]value.Value{m, num(2)}, nil).(*value.Number).Int64Value(); got != 0xeecd {
		t.Fatalf("overlay byte lane word=%#x", got)
	}
	if got := halStateMachineStagedWrites([]value.Value{m}, nil).(*value.Number).Int64Value(); got != 2 {
		t.Fatalf("staged writes=%d", got)
	}
	if got := halStateMachineCommitWrites([]value.Value{m}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if backing.FixedGet(1) != 0xeecd {
		t.Fatalf("committed word=%#x", backing.FixedGet(1))
	}
	if got := halStateMachineStagedWrites([]value.Value{m}, nil).(*value.Number).Int64Value(); got != 0 {
		t.Fatalf("staged writes after commit=%d", got)
	}
}

func TestStateMachineTransactionalDiscard(t *testing.T) {
	backing := &value.FixedStore{Kind: "word", Words: make([]uint16, 2)}
	backing.FixedSet(0, 0x1111)
	ops := smListV(
		smListV(num(smOpAddr18FromWord), num(0), num(0), num(0)),
		smListV(num(smOpStageWordAddr18), num(1), num(0), num(0)),
	)
	transitions := smListV(smListV(num(0), num(1), num(-1), num(0), num(0), num(1), ops, num(-1), num(-1)))
	p := halStateMachineCompile([]value.Value{num(2), num(2), num(0), num(0), transitions, num(1)}, nil)
	if value.IsFault(p) {
		t.Fatal(p.Inspect())
	}
	m := halStateMachineNew([]value.Value{p, smListV(num(0), num(0x2222)), smListV(backing)}, nil)
	if got := halStateMachineStep([]value.Value{m, num(1)}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if got := halStateMachineDiscardWrites([]value.Value{m}, nil); value.IsFault(got) {
		t.Fatal(got.Inspect())
	}
	if backing.FixedGet(0) != 0x1111 {
		t.Fatalf("discard mutated backing: %#x", backing.FixedGet(0))
	}
}

func TestStateMachineTransactionalAdmissionRejectsBadSlots(t *testing.T) {
	badStage := smListV(smListV(num(0), num(1), num(-1), num(0), num(0), num(1), smListV(
		smListV(num(smOpStageWordAddr18), num(0), num(0), num(1)),
	), num(-1), num(-1)))
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), badStage, num(1)}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range staged addr18 slot admitted")
	}
	badLoad := smListV(smListV(num(0), num(1), num(-1), num(0), num(0), num(1), smListV(
		smListV(num(smOpLoadByteAddr18), num(1), num(0), num(0)),
	), num(-1), num(-1)))
	if _, ok := halStateMachineCompile([]value.Value{num(2), num(1), num(0), num(0), badLoad, num(1)}, nil).(*value.Fault); !ok {
		t.Fatal("out-of-range byte-load destination admitted")
	}
}
