package machinevalue

import "testing"

func TestFiniteMachineValuesAreInternedAndMasked(t *testing.T) {
	if Word(012345) != Word(012345) {
		t.Fatal("word values are not interned")
	}
	if Word(0x10000) != Word(0) {
		t.Fatal("word value did not mask to 16 bits")
	}
	if Byte(0x100) != Byte(0) {
		t.Fatal("byte value did not mask to 8 bits")
	}
	if Addr18(0x40000) != Addr18(0) {
		t.Fatal("address value did not mask to 18 bits")
	}
	if bits, ok := Bits(Word(012345), "word"); !ok || bits != 012345 {
		t.Fatalf("Bits(word) = %#o, %v", bits, ok)
	}
	if _, ok := Bits(Word(1), "byte"); ok {
		t.Fatal("word accepted as byte")
	}
}
